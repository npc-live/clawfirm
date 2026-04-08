package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"time"

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
	// MessageSaver is called when a NativeProvider session completes with message history.
	// toolExecID identifies the tool invocation; sessionIndex is 0-based.
	MessageSaver func(toolExecID string, sessionIndex int, agentName string, messages []any)
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
			"stop_after_session": map[string]any{
				"type":        "integer",
				"description": "Stop execution after completing this session index (0-based, inclusive). Used for step-by-step debug mode.",
			},
			"replay_outputs": map[string]any{
				"type":        "array",
				"description": "Previous session outputs for replay — used instead of the SQLite state store when retrying. Each item must have 'index' (int) and 'output' (string). Extracted from the tool result of a prior partial whipflow_run call stored in message history.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"index":  map[string]any{"type": "integer"},
						"output": map[string]any{"type": "string"},
					},
					"required": []string{"index", "output"},
				},
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
	HasHistory bool   `json:"has_history,omitempty"` // true when message history was saved to DB
	StreamText string `json:"stream_text,omitempty"` // incremental text delta for live streaming
	// Messages holds the conversation turns for this session (set when Done=true and NativeProvider).
	Messages []any `json:"messages,omitempty"`
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

	// Replay previous session outputs supplied directly (from message history),
	// bypassing the SQLite state store entirely.
	if raw, ok := params["replay_outputs"]; ok {
		if arr, ok := raw.([]any); ok && len(arr) > 0 {
			var records []runtime.SessionRecord
			for _, item := range arr {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				var idx int
				switch v := m["index"].(type) {
				case float64:
					idx = int(v)
				case int:
					idx = v
				}
				output, _ := m["output"].(string)
				records = append(records, runtime.SessionRecord{
					SessionIndex: idx,
					Output:       output,
				})
			}
			if len(records) > 0 {
				execOpts = append(execOpts, whipflow.WithReplaySessions(records))
			}
		}
	}

	// Retry from a specific session index (state-store path, used with file= mode).
	if raw, ok := params["retry_from_session"]; ok {
		switch v := raw.(type) {
		case float64:
			execOpts = append(execOpts, whipflow.WithRetryFromSession(int(v)))
		case int:
			execOpts = append(execOpts, whipflow.WithRetryFromSession(v))
		}
	}

	// Stop after a specific session index (step-by-step debug mode).
	if raw, ok := params["stop_after_session"]; ok {
		switch v := raw.(type) {
		case float64:
			execOpts = append(execOpts, whipflow.WithStopAfterSession(int(v)))
		case int:
			execOpts = append(execOpts, whipflow.WithStopAfterSession(v))
		}
	}

	// Persist session steps to sidecar file so they survive app restart.
	// The sidecar is read by GetToolExecutionState in app.go on reload.
	var sidecarMu sync.Mutex
	var sidecarSteps []WhipflowSessionStep
	var lastSidecarWrite time.Time
	var debouncePending bool
	writeSidecar := func(step WhipflowSessionStep) {
		sidecarMu.Lock()
		// Merge: update existing index or append
		found := false
		for i, s := range sidecarSteps {
			if s.Index == step.Index {
				if step.StreamText != "" {
					step.StreamText = s.StreamText + step.StreamText
				}
				sidecarSteps[i] = step
				found = true
				break
			}
		}
		if !found {
			sidecarSteps = append(sidecarSteps, step)
		}
		snapshot := make([]WhipflowSessionStep, len(sidecarSteps))
		copy(snapshot, sidecarSteps)

		// Decide whether to write: always on session start/done, debounce stream deltas (2s).
		isStreamDelta := step.StreamText != "" && !step.Done
		shouldWrite := !isStreamDelta
		if isStreamDelta {
			now := time.Now()
			if now.Sub(lastSidecarWrite) >= 2*time.Second {
				shouldWrite = true
			} else if !debouncePending {
				// Schedule a deferred write so crash during long streaming doesn't lose all progress.
				debouncePending = true
				go func() {
					time.Sleep(2 * time.Second)
					sidecarMu.Lock()
					debouncePending = false
					snap := make([]WhipflowSessionStep, len(sidecarSteps))
					copy(snap, sidecarSteps)
					sidecarMu.Unlock()
					flushSidecar(id, snap)
				}()
			}
		}
		if shouldWrite {
			lastSidecarWrite = time.Now()
		}
		sidecarMu.Unlock()

		if shouldWrite {
			flushSidecar(id, snapshot)
		}
	}

	// Emit a ToolUpdate for each session start/end so the frontend can render
	// per-session progress in real time. Also persist message history via MessageSaver.
	execOpts = append(execOpts, whipflow.WithSessionProgressCallback(func(p whipflow.SessionProgress) {
		// Persist message history when session completes with NativeProvider messages.
		if p.Done && len(p.Messages) > 0 && w.MessageSaver != nil {
			w.MessageSaver(id, p.Index, p.AgentName, p.Messages)
		}
		step := WhipflowSessionStep{
			Index:      p.Index,
			Name:       p.Name,
			Provider:   p.Provider,
			Prompt:     p.Prompt,
			Done:       p.Done,
			Output:     p.Output,
			DurationMs: p.DurationMs,
			Error:      p.Error,
			HasHistory: p.Done && len(p.Messages) > 0,
			StreamText: p.StreamText,
		}
		writeSidecar(step)
		if onUpdate != nil {
			// Include messages when session is done so frontend can render conversation history.
			if p.Done && len(p.Messages) > 0 {
				step.Messages = p.Messages
			}
			onUpdate(tool.ToolUpdate{Details: step})
		}
	}))

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

	// Build structured session output list for replay in future retry calls.
	// Each entry is keyed by session index so the next partial run can pass them
	// back as replay_outputs without needing the SQLite state store.
	type sessionOutput struct {
		Index      int    `json:"index"`
		Output     string `json:"output"`
		Model      string `json:"model,omitempty"`
		DurationMs int64  `json:"duration_ms,omitempty"`
	}
	sessionOutputs := make([]sessionOutput, 0, len(result.SessionOutputs))
	for i, s := range result.SessionOutputs {
		sessionOutputs = append(sessionOutputs, sessionOutput{
			Index:      i,
			Output:     s.Output,
			Model:      s.Metadata.Model,
			DurationMs: s.Metadata.Duration,
		})
	}

	// Structured result payload stored as tool result in message history.
	// The agent can read session_outputs to pass as replay_outputs on retry.
	type whipflowResult struct {
		Success        bool            `json:"success"`
		Sessions       int             `json:"sessions"`
		DurationMs     int64           `json:"duration_ms"`
		SessionOutputs []sessionOutput `json:"session_outputs"`
		Errors         []string        `json:"errors,omitempty"`
	}
	wResult := whipflowResult{
		Success:        result.Success,
		Sessions:       result.Metadata.SessionsCreated,
		DurationMs:     result.Metadata.Duration,
		SessionOutputs: sessionOutputs,
	}
	for _, e := range result.Errors {
		wResult.Errors = append(wResult.Errors, fmt.Sprintf("[%s] %s", e.Type, e.Message))
	}
	resultJSON, _ := json.Marshal(wResult)

	// Human-readable text summary for agent consumption.
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Workflow completed: success=%v, sessions=%d, duration=%dms\n",
		result.Success, result.Metadata.SessionsCreated, result.Metadata.Duration))
	for i, s := range result.SessionOutputs {
		sb.WriteString(fmt.Sprintf("\n[Session %d] (%s, %dms)\n%s\n",
			i, s.Metadata.Model, s.Metadata.Duration, s.Output))
	}
	if len(result.Errors) > 0 {
		sb.WriteString("\nErrors:\n")
		for _, e := range result.Errors {
			sb.WriteString(fmt.Sprintf("  [%s] %s\n", e.Type, e.Message))
		}
	}
	sb.WriteString("\nJSON result (for replay_outputs on retry):\n")
	sb.Write(resultJSON)

	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Text: sb.String()}},
		Details: wResult,
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

// flushSidecar atomically writes session steps to the sidecar JSON file (write-to-tmp + rename).
func flushSidecar(id string, steps []WhipflowSessionStep) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".clawfirm", "whipflow-steps")
	_ = os.MkdirAll(dir, 0o755)
	data, _ := json.Marshal(steps)
	tmpPath := filepath.Join(dir, id+".json.tmp")
	_ = os.WriteFile(tmpPath, data, 0o644)
	_ = os.Rename(tmpPath, filepath.Join(dir, id+".json"))
}

// readWhipFile reads a .whip file and returns its content as a string.
func readWhipFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
