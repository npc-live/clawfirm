package message

import (
	"encoding/json"
	"fmt"

	"github.com/ai-gateway/pi-go/types"
)

// rawMessage is used for JSON deserialization dispatch.
type rawMessage struct {
	Role string `json:"role"`
}

// MarshalMessages serializes a message slice to JSON.
func MarshalMessages(messages []types.Message) ([]byte, error) {
	return json.Marshal(messages)
}

// UnmarshalMessages deserializes a JSON array of messages, dispatching by role.
func UnmarshalMessages(data []byte) ([]types.Message, error) {
	var raws []json.RawMessage
	if err := json.Unmarshal(data, &raws); err != nil {
		return nil, err
	}

	msgs := make([]types.Message, 0, len(raws))
	for _, raw := range raws {
		var r rawMessage
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		var msg types.Message
		switch r.Role {
		case "user":
			v := &types.UserMessage{}
			if err := json.Unmarshal(raw, v); err != nil {
				return nil, err
			}
			msg = v
		case "assistant":
			v := &types.AssistantMessage{}
			if err := json.Unmarshal(raw, v); err != nil {
				return nil, err
			}
			msg = v
		case "tool":
			v := &types.ToolResultMessage{}
			if err := json.Unmarshal(raw, v); err != nil {
				return nil, err
			}
			msg = v
		default:
			return nil, fmt.Errorf("unknown message role: %q", r.Role)
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}
