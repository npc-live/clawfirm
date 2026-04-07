package clawproc

import "encoding/json"

// Event represents a single NDJSON event from the claw subprocess stdout.
type Event struct {
	Type         string `json:"type"`
	Text         string `json:"text,omitempty"`           // text_delta
	ID           string `json:"id,omitempty"`             // tool_use
	Name         string `json:"name,omitempty"`           // tool_use
	Input        string `json:"input,omitempty"`          // tool_use (JSON string)
	ToolUseID    string `json:"tool_use_id,omitempty"`    // tool_result
	ToolName     string `json:"tool_name,omitempty"`      // tool_result
	Content      string `json:"content,omitempty"`        // tool_result
	IsError      bool   `json:"is_error,omitempty"`       // tool_result
	InputTokens  int    `json:"input_tokens,omitempty"`   // usage / turn_end
	OutputTokens int    `json:"output_tokens,omitempty"`  // usage / turn_end
	StopReason   string `json:"stop_reason,omitempty"`    // turn_end
	Iterations   int    `json:"iterations,omitempty"`     // turn_end
	SessionID    string `json:"session_id,omitempty"`     // ready / session_init
	Model        string `json:"model,omitempty"`          // ready
	Message      string `json:"message,omitempty"`        // error
}

// stdinCommand is a JSON command written to the claw subprocess stdin.
type stdinCommand struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ParseEvent parses a single NDJSON line into an Event.
func ParseEvent(line []byte) (*Event, error) {
	var ev Event
	if err := json.Unmarshal(line, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}
