package builtin

import "github.com/ai-gateway/pi-go/tool"

// CodingTools returns the standard set of file-system + shell tools:
// Read, Write, Edit, Bash.
func CodingTools() []tool.AgentTool {
	return []tool.AgentTool{
		&Read{},
		&Write{},
		&Edit{},
		&Bash{},
	}
}
