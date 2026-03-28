package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// GetCurrentTime returns the current time in the requested format.
type GetCurrentTime struct{}

func (t *GetCurrentTime) Name() string        { return "get_current_time" }
func (t *GetCurrentTime) Description() string { return "Returns the current time in the requested format." }
func (t *GetCurrentTime) Label() string       { return "Get Current Time" }
func (t *GetCurrentTime) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"format": map[string]any{
				"type":        "string",
				"description": `Time format. Shortcuts: "unix" (Unix timestamp), "rfc3339" (default), "date" (YYYY-MM-DD), "time" (HH:MM:SS). Or any Go layout string, e.g. "2006-01-02 15:04:05".`,
			},
			"timezone": map[string]any{
				"type":        "string",
				"description": `IANA timezone name, e.g. "UTC", "America/New_York", "Asia/Shanghai". Defaults to local timezone.`,
			},
		},
	}
}

func (t *GetCurrentTime) Execute(_ context.Context, _ string, params map[string]any, _ func(tool.ToolUpdate)) (tool.ToolResult, error) {
	format, _ := params["format"].(string)
	tz, _ := params["timezone"].(string)

	loc := time.Local
	if tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}

	now := time.Now().In(loc)

	var result string
	switch format {
	case "", "rfc3339":
		result = now.Format(time.RFC3339)
	case "unix":
		result = fmt.Sprintf("%d", now.Unix())
	case "date":
		result = now.Format("2006-01-02")
	case "time":
		result = now.Format("15:04:05")
	default:
		result = now.Format(format)
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Text: result}},
	}, nil
}
