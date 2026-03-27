package stream

import (
	"context"
	"testing"

	"github.com/ai-gateway/pi-go/types"
)

func TestEventStreamNext(t *testing.T) {
	ch := make(chan types.AssistantMessageEvent, 2)
	ch <- types.AssistantMessageEvent{Type: types.StreamEventTextDelta, Delta: "hello"}
	ch <- types.AssistantMessageEvent{Type: types.StreamEventDone}
	close(ch)

	es := NewEventStream(ch)
	ctx := context.Background()

	ev, ok := es.Next(ctx)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if ev.Delta != "hello" {
		t.Errorf("Delta: got %q want %q", ev.Delta, "hello")
	}

	ev2, ok2 := es.Next(ctx)
	if !ok2 {
		t.Fatal("expected ok=true for done event")
	}
	if ev2.Type != types.StreamEventDone {
		t.Errorf("Type: got %q want done", ev2.Type)
	}

	_, ok3 := es.Next(ctx)
	if ok3 {
		t.Error("expected ok=false after channel closed")
	}
}

func TestCollectMessage(t *testing.T) {
	msg := &types.AssistantMessage{
		Role:       "assistant",
		Model:      "test-model",
		StopReason: types.StopReasonStop,
	}
	ch := make(chan types.AssistantMessageEvent, 2)
	ch <- types.AssistantMessageEvent{Type: types.StreamEventTextDelta, Delta: "hi"}
	ch <- types.AssistantMessageEvent{Type: types.StreamEventDone, Message: msg}
	close(ch)

	result, err := CollectMessage(context.Background(), ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil message")
	}
	if result.Model != "test-model" {
		t.Errorf("Model: got %q want test-model", result.Model)
	}
}

func TestCollectMessageContextCancel(t *testing.T) {
	ch := make(chan types.AssistantMessageEvent) // never sends
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := CollectMessage(ctx, ch)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
