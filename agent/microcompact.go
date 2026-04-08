package agent

import "github.com/ai-gateway/clawfirm/types"

// clearableTools lists tool names whose old results can safely be replaced
// with a short placeholder. These are tools whose output is observational
// (file contents, command output) rather than state-changing actions.
var clearableTools = map[string]bool{
	"read": true, "bash": true, "exec": true, "grep": true,
	"find": true, "ls": true, "fetch": true, "process": true,
}

// MicrocompactMessages replaces the content of old tool results with a short
// placeholder to reduce prompt size. Only tool results from "clearable" tools
// (read, bash, exec, grep, find, ls, fetch, process) older than the most
// recent keepTurns assistant messages are replaced.
//
// This is safe because the LLM has already seen and reasoned about those
// results in earlier turns — the full content is no longer needed.
func MicrocompactMessages(msgs []types.Message, keepTurns int) []types.Message {
	if keepTurns <= 0 {
		keepTurns = 3
	}

	// Find the cutoff: index of the keepTurns-th assistant message from the end.
	assistantCount := 0
	cutoff := 0
	for i := len(msgs) - 1; i >= 0; i-- {
		if _, ok := msgs[i].(*types.AssistantMessage); ok {
			assistantCount++
			if assistantCount >= keepTurns {
				cutoff = i
				break
			}
		}
	}
	if cutoff == 0 {
		return msgs // not enough history to compact
	}

	result := make([]types.Message, len(msgs))
	for i, m := range msgs {
		if i < cutoff {
			if tr, ok := m.(*types.ToolResultMessage); ok && clearableTools[tr.ToolName] {
				cleared := *tr
				cleared.Content = []types.ContentBlock{
					&types.TextContent{
						Type: types.ContentTypeText,
						Text: "[Old tool result content cleared]",
					},
				}
				cleared.Details = nil
				result[i] = &cleared
				continue
			}
		}
		result[i] = m
	}
	return result
}
