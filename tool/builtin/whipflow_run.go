package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
	"github.com/ai-gateway/clawfirm/whipflow"
	"github.com/ai-gateway/clawfirm/whipflow/runtime"
)

// WhipflowRun is a tool that executes WhipFlow (.whip) workflows.
// PiConfig is the loaded ~/.clawfirm/config.yml; when set it allows WhipFlow
// sessions to resolve agents and providers defined there.
// Tools are the AgentTool instances available to NativeProvider sessions.
type WhipflowRun struct {
	PiConfig     *config.Config
	VaultEnv     func() map[string]string
	Tools        []tool.AgentTool
	SkillLoader  *Skill // shared Skill tool instance for resolving skill content
}

func (w *WhipflowRun) Name() string        { return "whipflow_run" }
func (w *WhipflowRun) Description() string {
	return "Execute a WhipFlow (.whip) workflow from source code or a file path. If the workflow contains `ask` statements (user input variables), you MUST provide their values via user_inputs. Ask the user for these values BEFORE calling this tool."
}
func (w *WhipflowRun) Label() string        { return "Run Workflow" }
func (w *WhipflowRun) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"source": map[string]any{
				"type":        "string",
				"description": "WhipFlow source code to execute directly.",
			},
			"file": map[string]any{
				"type":        "string",
				"description": "Path to a .whip file to execute.",
			},
			"mode": map[string]any{
				"type":        "string",
				"enum":        []string{"auto", "execute", "preview"},
				"description": "Execution mode. 'auto' (default): analyze complexity and auto-decide — simple tasks execute immediately, others return a preview. 'execute': run the workflow immediately (backward-compatible). 'preview': parse, validate, and return complexity analysis without executing.",
			},
			"user_inputs": map[string]any{
				"type":        "object",
				"description": "Required when the workflow has `ask` statements. Map of variable name → value. Example: if the whip has `ask asset_class: \"What asset class?\"`, pass {\"asset_class\": \"Real Estate\"}. Ask the user for these values before running.",
				"additionalProperties": map[string]any{"type": "string"},
			},
			"retry_from_session": map[string]any{
				"type":        "integer",
				"description": "Session index (0-based) to retry from. Sessions before this index are replayed from saved state; this session and all after it are re-executed.",
			},
		},
	}
}

// WhipflowSessionStep is the structured payload emitted via onUpdate for each
// WhipFlow session. It is serialised as the Details field of a ToolUpdate so
// the frontend can render a rich per-session progress view.
type WhipflowSessionStep struct {
	Index      int    `json:"index"`
	Name       string `json:"name,omitempty"`
	Provider   string `json:"provider,omitempty"`
	Prompt     string `json:"prompt"`
	Done       bool   `json:"done"`
	Output     string `json:"output,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// WhipflowPreview is the structured result returned when mode is "preview" or
// "auto" for non-simple workflows. The frontend renders a StepPreviewCard.
type WhipflowPreview struct {
	Type     string                      `json:"type"` // always "whipflow_preview"
	Analysis *whipflow.ComplexityAnalysis `json:"analysis"`
	Source   string                      `json:"source"`
}

func (w *WhipflowRun) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	source, _ := params["source"].(string)
	filePath, _ := params["file"].(string)
	mode, _ := params["mode"].(string)
	if mode == "" {
		mode = "auto"
	}

	// Extract user_inputs (map[string]string) from params.
	var userInputs map[string]string
	if raw, ok := params["user_inputs"]; ok {
		switch v := raw.(type) {
		case map[string]string:
			userInputs = v
		case map[string]any:
			userInputs = make(map[string]string, len(v))
			for k, val := range v {
				if s, ok := val.(string); ok {
					userInputs[k] = s
				}
			}
		}
	}

	log.Printf("whipflow_run: source_len=%d file=%q user_inputs=%v\nsource:\n%s", len(source), filePath, userInputs, source)

	if source == "" && filePath == "" {
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Text: "Error: either 'source' or 'file' must be provided."}},
		}, nil
	}

	var result *whipflow.ExecutionResult
	var err error

	var execOpts []whipflow.Option
	if w.PiConfig != nil {
		execOpts = append(execOpts, whipflow.WithPiConfig(w.PiConfig))

		// Resolve default provider with priority:
		// 1. whipflow.default_provider in config.yml (explicitly set by user)
		// 2. default_agent in config.yml
		// 3. First agent in agents list
		// 4. Built-in "claude-code" preset (from DefaultRuntimeConfig)
		rCfg := runtime.DefaultRuntimeConfig()
		if w.PiConfig.Whipflow.DefaultProvider != "" {
			rCfg.DefaultProvider = w.PiConfig.Whipflow.DefaultProvider
		} else {
			defaultProvider := w.PiConfig.DefaultAgent
			if defaultProvider == "" && len(w.PiConfig.Agents) > 0 {
				defaultProvider = w.PiConfig.Agents[0].Name
			}
			if defaultProvider != "" {
				rCfg.DefaultProvider = defaultProvider
			}
		}
		rCfg.VaultEnv = w.VaultEnv
		rCfg.Ctx = ctx
		if w.SkillLoader != nil {
			rCfg.SkillResolver = w.SkillLoader.LoadSkill
		}
		execOpts = append(execOpts, whipflow.WithRuntimeConfig(&rCfg))
	}

	// Inject tools into NativeProviders so whipflow sessions can use real tools.
	// When config.yml agents are resolved as NativeProviders, they are created
	// without tools. We inject the tools here so the agent loop can call them.
	if w.PiConfig != nil && len(w.Tools) > 0 {
		for _, ac := range w.PiConfig.Agents {
			pc, ok := w.PiConfig.Providers[ac.Provider]
			if !ok {
				continue
			}
			llmProv, err := runtime.BuildLLMProvider(pc)
			if err != nil {
				log.Printf("whipflow_run: skip agent %q: %v", ac.Name, err)
				continue
			}
			np, err := runtime.NewNativeProvider(ac.Name, ac.Model, llmProv,
				runtime.WithNativeTools(w.Tools),
				runtime.WithMaxTokens(ac.MaxTokens),
				runtime.WithSystemPromptHint(ac.SystemPrompt),
			)
			if err != nil {
				log.Printf("whipflow_run: skip agent %q: %v", ac.Name, err)
				continue
			}
			execOpts = append(execOpts, whipflow.WithNativeProvider(ac.Name, np))
		}
	}

	// Pre-filled ask inputs (from desktop UI form).
	if len(userInputs) > 0 {
		execOpts = append(execOpts, whipflow.WithInitialInputs(userInputs))
	}

	// Retry from a specific session index.
	if raw, ok := params["retry_from_session"]; ok {
		switch v := raw.(type) {
		case float64:
			execOpts = append(execOpts, whipflow.WithRetryFromSession(int(v)))
		case int:
			execOpts = append(execOpts, whipflow.WithRetryFromSession(v))
		}
	}

	// Emit a ToolUpdate for each session start/end so the frontend can render
	// per-session progress in real time.
	if onUpdate != nil {
		execOpts = append(execOpts, whipflow.WithSessionProgressCallback(func(p whipflow.SessionProgress) {
			onUpdate(tool.ToolUpdate{
				Details: WhipflowSessionStep{
					Index:      p.Index,
					Name:       p.Name,
					Provider:   p.Provider,
					Prompt:     p.Prompt,
					Done:       p.Done,
					Output:     p.Output,
					DurationMs: p.DurationMs,
					Error:      p.Error,
				},
			})
		}))
	}

	if filePath != "" {
		// For file mode, read & parse so we can analyze complexity when needed.
		if mode == "preview" || mode == "auto" {
			fileSource, readErr := readWhipFile(filePath)
			if readErr != nil {
				return tool.ToolResult{
					Content: []types.ContentBlock{&types.TextContent{Text: fmt.Sprintf("Error reading file: %v", readErr)}},
				}, nil
			}
			program, parseErrors := whipflow.Parse(fileSource)
			if len(parseErrors) > 0 {
				msgs := make([]string, len(parseErrors))
				for i, e := range parseErrors {
					msgs[i] = e.Error()
				}
				return tool.ToolResult{
					Content: []types.ContentBlock{&types.TextContent{Text: "Parse errors:\n" + strings.Join(msgs, "\n")}},
				}, nil
			}
			vResult := whipflow.Validate(program)
			if !vResult.Valid {
				msgs := make([]string, len(vResult.Errors))
				for i, e := range vResult.Errors {
					msgs[i] = e.Message
				}
				return tool.ToolResult{
					Content: []types.ContentBlock{&types.TextContent{Text: "Validation errors:\n" + strings.Join(msgs, "\n")}},
				}, nil
			}
			analysis := whipflow.AnalyzeComplexity(program)
			if mode == "preview" || (mode == "auto" && shouldPreview(analysis)) {
				return makePreviewResult(analysis, fileSource)
			}
		}
		result, err = whipflow.RunFile(filePath, execOpts...)
	} else {
		program, parseErrors := whipflow.Parse(source)
		if len(parseErrors) > 0 {
			msgs := make([]string, len(parseErrors))
			for i, e := range parseErrors {
				msgs[i] = e.Error()
			}
			return tool.ToolResult{
				Content: []types.ContentBlock{&types.TextContent{Text: "Parse errors:\n" + strings.Join(msgs, "\n")}},
			}, nil
		}

		vResult := whipflow.Validate(program)
		if !vResult.Valid {
			msgs := make([]string, len(vResult.Errors))
			for i, e := range vResult.Errors {
				msgs[i] = e.Message
			}
			return tool.ToolResult{
				Content: []types.ContentBlock{&types.TextContent{Text: "Validation errors:\n" + strings.Join(msgs, "\n")}},
			}, nil
		}

		// For auto/preview modes, analyze complexity before executing.
		if mode == "preview" || mode == "auto" {
			analysis := whipflow.AnalyzeComplexity(program)
			if mode == "preview" || (mode == "auto" && shouldPreview(analysis)) {
				return makePreviewResult(analysis, source)
			}
		}

		result, err = whipflow.Execute(program, execOpts...)
	}

	if err != nil {
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Text: fmt.Sprintf("Execution error: %v", err)}},
		}, nil
	}

	// Format output.
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Workflow completed: success=%v\n", result.Success))
	sb.WriteString(fmt.Sprintf("Sessions: %d, Statements: %d, Duration: %dms\n",
		result.Metadata.SessionsCreated, result.Metadata.StatementsExecuted, result.Metadata.Duration))

	if len(result.SessionOutputs) > 0 {
		sb.WriteString("\n--- Session Outputs ---\n")
		for i, s := range result.SessionOutputs {
			sb.WriteString(fmt.Sprintf("\n[Session %d] (%s, %dms)\n%s\n",
				i+1, s.Metadata.Model, s.Metadata.Duration, s.Output))
		}
	}

	if len(result.Errors) > 0 {
		sb.WriteString("\n--- Errors ---\n")
		for _, e := range result.Errors {
			sb.WriteString(fmt.Sprintf("  [%s] %s\n", e.Type, e.Message))
		}
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Text: sb.String()}},
	}, nil
}

// shouldPreview returns true when auto mode should return a preview instead of executing.
func shouldPreview(a *whipflow.ComplexityAnalysis) bool {
	if a.HasAsk {
		return true
	}
	return a.Tier != whipflow.TierSimple
}

// makePreviewResult serialises a WhipflowPreview as the tool result.
// Details is set to the preview struct (sent to frontend via tool_end event);
// Content holds the JSON text (used by the agent loop as the tool response).
func makePreviewResult(a *whipflow.ComplexityAnalysis, source string) (tool.ToolResult, error) {
	preview := WhipflowPreview{
		Type:     "whipflow_preview",
		Analysis: a,
		Source:   source,
	}
	data, err := json.Marshal(preview)
	if err != nil {
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{Text: fmt.Sprintf("Failed to serialize preview: %v", err)}},
		}, nil
	}
	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Text: string(data)}},
		Details: preview,
	}, nil
}

// readWhipFile reads a .whip file and returns its content as a string.
func readWhipFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
