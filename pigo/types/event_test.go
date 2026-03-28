package types

import (
	"testing"
)

func TestAgentEventTypeConstants(t *testing.T) {
	cases := []struct {
		got  AgentEventType
		want string
	}{
		{EventAgentStart, "agent_start"},
		{EventAgentEnd, "agent_end"},
		{EventTurnStart, "turn_start"},
		{EventTurnEnd, "turn_end"},
		{EventMessageStart, "message_start"},
		{EventMessageUpdate, "message_update"},
		{EventMessageEnd, "message_end"},
		{EventToolExecutionStart, "tool_execution_start"},
		{EventToolExecutionUpdate, "tool_execution_update"},
		{EventToolExecutionEnd, "tool_execution_end"},
	}
	for _, c := range cases {
		if string(c.got) != c.want {
			t.Errorf("AgentEventType %q: got %q want %q", c.want, c.got, c.want)
		}
	}
}

func TestAgentEventFields(t *testing.T) {
	msg := &AssistantMessage{Role: "assistant", Model: "test"}
	ev := AgentEvent{
		Type:         EventMessageEnd,
		AssistantMsg: msg,
		ToolCallID:   "call_1",
		ToolName:     "search",
	}
	if ev.Type != EventMessageEnd {
		t.Errorf("Type: got %q want %q", ev.Type, EventMessageEnd)
	}
	if ev.AssistantMsg == nil || ev.AssistantMsg.Model != "test" {
		t.Errorf("AssistantMsg not set correctly")
	}
	if ev.ToolCallID != "call_1" {
		t.Errorf("ToolCallID: got %q want %q", ev.ToolCallID, "call_1")
	}
}
