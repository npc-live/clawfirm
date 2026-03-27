package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-gateway/pi-go/config"
	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
	"github.com/ai-gateway/pi-go/whipflow"
	"github.com/ai-gateway/pi-go/whipflow/runtime"
)

// WhipflowRun is a tool that executes WhipFlow (.whip) workflows.
// PiConfig is the loaded ~/.pi-go/config.yml; when set it allows WhipFlow
// sessions to resolve agents and providers defined there.
type WhipflowRun struct {
	PiConfig *config.Config
	VaultEnv func() map[string]string
}

func (w *WhipflowRun) Name() string        { return "whipflow_run" }
func (w *WhipflowRun) Description() string  { return "Execute a WhipFlow (.whip) workflow from source code or a file path." }
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

func (w *WhipflowRun) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	source, _ := params["source"].(string)
	filePath, _ := params["file"].(string)

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
		execOpts = append(execOpts, whipflow.WithRuntimeConfig(&rCfg))
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
