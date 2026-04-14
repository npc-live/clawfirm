package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/types"
	"gopkg.in/yaml.v3"
)

// HealerConfig holds the configuration for auto-healing broken YAML steps.
type HealerConfig struct {
	Provider   provider.LLMProvider
	Model      string
	IssueRepo  string // e.g. "org/repo" for gh issue create
	MaxRetries int    // default 3
}

// StepFailure captures all context about a failed YAML step.
type StepFailure struct {
	AdapterPath string
	Platform    string
	CommandName string
	StepIndex   int
	Step        Step
	Preceding   []Step // steps before the failed one (context for eval→upload patterns)
	Remaining   []Step // steps after the failed one
	Error       string
	Vars        map[string]string
}

// HealAction is the LLM's JSON response describing how to fix the step.
type HealAction struct {
	Action      string `json:"action"`                 // click, fill, upload, eval, type_rich
	Selector    string `json:"selector,omitempty"`      // CSS selector to use
	Value       string `json:"value,omitempty"`         // value for fill/type_rich
	JS          string `json:"js,omitempty"`            // JavaScript for eval action
	Explanation string `json:"explanation"`             // why the fix works
	NewSelector string `json:"new_selector,omitempty"`  // replacement selector for YAML patch
}

// healStep attempts to recover from a failed step by asking an LLM to inspect
// the DOM snapshot and suggest a fix. Returns whether healing succeeded.
func healStep(ctx context.Context, cfg *HealerConfig, failure StepFailure, stepExec *StepExecutor) (bool, *HealAction, error) {
	snapshot := stepExec.Snapshot()
	if snapshot == "" {
		return false, nil, fmt.Errorf("healer: empty DOM snapshot")
	}
	currentURL := stepExec.GetURL()

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("healer: attempt %d/%d for step %d (%s)", attempt, maxRetries, failure.StepIndex, failure.Error)

		systemPrompt, userPrompt := buildHealPrompt(failure, snapshot, currentURL)

		msg := types.NewUserMessage(&types.TextContent{
			Type: types.ContentTypeText,
			Text: userPrompt,
		})

		req := provider.LLMRequest{
			Model:        types.Model{ID: cfg.Model},
			SystemPrompt: systemPrompt,
			Messages:     []types.Message{msg},
		}

		eventCh, err := cfg.Provider.Stream(ctx, req)
		if err != nil {
			log.Printf("healer: LLM call failed: %v", err)
			continue
		}

		var responseText string
		for event := range eventCh {
			switch event.Type {
			case types.StreamEventTextDelta:
				responseText += event.Delta
			case types.StreamEventError:
				if event.Error != nil {
					log.Printf("healer: LLM stream error: %s", event.Error.ErrorMessage)
				}
			}
		}

		action, err := parseHealAction(responseText)
		if err != nil {
			log.Printf("healer: failed to parse LLM response: %v (response: %s)", err, truncStr(responseText, 200))
			continue
		}

		log.Printf("healer: LLM suggests action=%s selector=%q explanation=%s", action.Action, action.Selector, action.Explanation)

		if err := executeHealAction(action, stepExec, failure.Vars); err != nil {
			log.Printf("healer: action execution failed: %v", err)
			continue
		}

		log.Printf("healer: step %d healed successfully", failure.StepIndex)

		// Patch YAML and file issue in background.
		if action.NewSelector != "" {
			oldSelector := extractFailedSelector(failure.Step)
			if oldSelector != "" && oldSelector != action.NewSelector {
				if err := patchYAMLSelector(failure.AdapterPath, failure.CommandName, oldSelector, action.NewSelector); err != nil {
					log.Printf("healer: YAML patch failed: %v", err)
				} else {
					log.Printf("healer: patched YAML selector %q → %q", oldSelector, action.NewSelector)
				}
			}
		}

		if cfg.IssueRepo != "" {
			go fileGitHubIssue(cfg.IssueRepo, failure, action)
		}

		return true, action, nil
	}

	return false, nil, fmt.Errorf("healer: exhausted %d retries", maxRetries)
}

// buildHealPrompt constructs the system and user prompts for the healing LLM call.
func buildHealPrompt(failure StepFailure, snapshot, currentURL string) (system, user string) {
	system = `You are a browser automation repair agent. A YAML-driven browser step failed (selector not found, assertion failed, or timeout). You are given:
- The preceding steps (already executed) for context
- The failed step and error
- The DOM snapshot of the current page
- The remaining steps to understand the goal

Respond with ONLY a valid JSON object (no markdown, no explanation outside JSON):
- "action": one of "click", "fill", "upload", "eval", "type_rich"
- "selector": CSS selector (for click/fill/upload/type_rich)
- "value": value string (for fill/type_rich)
- "js": JavaScript code (for eval — use this for shadow DOM, complex element access, or when CSS selectors won't work)
- "explanation": brief explanation of what changed and why this fix works
- "new_selector": replacement CSS selector for the YAML file (empty if eval-based fix)

IMPORTANT:
- The DOM snapshot is an accessibility tree. Elements inside shadow roots (e.g. wujie-app) may not be directly accessible via CSS selectors. Use "eval" action with JavaScript to access shadow root elements.
- If preceding steps used eval to prepare elements (e.g. moving file inputs from shadow root to document), your fix may need to do the same.
- For upload steps, the selector must point to an actual input[type=file] element.
- Choose stable selectors (data attributes, IDs, ARIA labels over class names).`

	stepYAML, _ := yaml.Marshal(failure.Step)

	// Show last few preceding steps (most relevant context).
	preceding := failure.Preceding
	if len(preceding) > 5 {
		preceding = preceding[len(preceding)-5:]
	}
	precedingYAML, _ := yaml.Marshal(preceding)
	remainingYAML, _ := yaml.Marshal(failure.Remaining)

	// Truncate snapshot to avoid exceeding context limits.
	snap := snapshot
	if len(snap) > 60000 {
		snap = snap[:60000] + "\n... (truncated)"
	}

	user = fmt.Sprintf(`## Preceding Steps (already executed)
%s

## Failed Step (index %d)
%s

## Error
%s

## Current URL
%s

## Variables
%s

## Remaining Steps
%s

## DOM Snapshot
%s`, string(precedingYAML), failure.StepIndex, string(stepYAML), failure.Error, currentURL, formatVars(failure.Vars), string(remainingYAML), snap)

	return system, user
}

// parseHealAction extracts a HealAction from the LLM's JSON response.
func parseHealAction(response string) (*HealAction, error) {
	// Find the first { and last } to extract JSON even if surrounded by text.
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start < 0 || end < 0 || end <= start {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	var action HealAction
	if err := json.Unmarshal([]byte(response[start:end+1]), &action); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if action.Action == "" {
		return nil, fmt.Errorf("missing 'action' field")
	}
	return &action, nil
}

// executeHealAction dispatches the healed action to the step executor.
func executeHealAction(action *HealAction, stepExec *StepExecutor, vars map[string]string) error {
	switch action.Action {
	case "click":
		if action.Selector == "" {
			return fmt.Errorf("click action requires selector")
		}
		r := stepExec.Click(action.Selector)
		if !r.OK {
			return fmt.Errorf("click %q: %s", action.Selector, r.Error)
		}
	case "fill":
		if action.Selector == "" {
			return fmt.Errorf("fill action requires selector")
		}
		val := interpolate(action.Value, vars)
		r := stepExec.Fill(action.Selector, val)
		if !r.OK {
			return fmt.Errorf("fill %q: %s", action.Selector, r.Error)
		}
	case "type_rich":
		if action.Selector == "" {
			return fmt.Errorf("type_rich action requires selector")
		}
		val := interpolate(action.Value, vars)
		r := stepExec.TypeContentEditable(action.Selector, val)
		if !r.OK {
			return fmt.Errorf("type_rich %q: %s", action.Selector, r.Error)
		}
	case "upload":
		if action.Selector == "" {
			return fmt.Errorf("upload action requires selector")
		}
		val := interpolate(action.Value, vars)
		r := stepExec.Upload(action.Selector, val)
		if !r.OK {
			return fmt.Errorf("upload %q: %s", action.Selector, r.Error)
		}
	case "eval":
		if action.JS == "" {
			return fmt.Errorf("eval action requires js")
		}
		js := interpolate(action.JS, vars)
		r := stepExec.Eval(js)
		if !r.OK {
			return fmt.Errorf("eval: %s", r.Error)
		}
	default:
		return fmt.Errorf("unknown action %q", action.Action)
	}
	return nil
}

// patchYAMLSelector does a simple string replacement in the raw YAML file,
// preserving comments and formatting.
func patchYAMLSelector(filePath, commandName, oldSelector, newSelector string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	old := string(data)
	updated := strings.Replace(old, oldSelector, newSelector, 1)
	if updated == old {
		return fmt.Errorf("selector %q not found in %s", oldSelector, filePath)
	}

	return os.WriteFile(filePath, []byte(updated), 0644)
}

// fileGitHubIssue creates a GitHub issue documenting the selector change.
// Runs in a goroutine — errors are logged but do not affect the caller.
func fileGitHubIssue(repo string, failure StepFailure, action *HealAction) {
	oldSel := extractFailedSelector(failure.Step)
	title := fmt.Sprintf("[auto-heal] %s: selector changed in %s step %d",
		failure.Platform, failure.CommandName, failure.StepIndex)

	body := fmt.Sprintf(`## Auto-Healed Selector Change

**Platform:** %s
**Command:** %s
**Step:** %d
**Old selector:** `+"`%s`"+`
**New selector:** `+"`%s`"+`
**Error:** %s

### Explanation
%s

### Action Taken
- Action: %s
- YAML file patched: %s

_This issue was automatically created by the browser shortcut auto-healer._`,
		failure.Platform, failure.CommandName, failure.StepIndex,
		oldSel, action.NewSelector, failure.Error,
		action.Explanation, action.Action, failure.AdapterPath)

	cmd := exec.Command("gh", "issue", "create",
		"--repo", repo,
		"--title", title,
		"--body", body,
		"--label", "auto-heal",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("healer: gh issue create failed: %v\n%s", err, string(out))
		return
	}
	log.Printf("healer: filed GitHub issue: %s", strings.TrimSpace(string(out)))
}

// extractFailedSelector returns the CSS selector from a failed step, if any.
func extractFailedSelector(step Step) string {
	switch {
	case step.Click != nil:
		switch v := step.Click.(type) {
		case string:
			return v
		case map[string]any:
			if sel, ok := v["selector"].(string); ok {
				return sel
			}
		}
	case step.Fill != nil:
		return step.Fill.Selector
	case step.TypeRich != nil:
		return step.TypeRich.Selector
	case step.Upload != nil:
		return step.Upload.Selector
	}
	return ""
}

// formatVars formats the variable map for display in the prompt.
func formatVars(vars map[string]string) string {
	if len(vars) == 0 {
		return "(none)"
	}
	var sb strings.Builder
	for k, v := range vars {
		fmt.Fprintf(&sb, "  %s = %s\n", k, truncStr(v, 100))
	}
	return sb.String()
}
