package stream

import (
	"context"
	"io"

	"github.com/ai-gateway/pi-go/types"
)

// EventStream wraps a channel of AssistantMessageEvent with helpers for consumption.
type EventStream struct {
	ch <-chan types.AssistantMessageEvent
}

// NewEventStream wraps an existing channel.
func NewEventStream(ch <-chan types.AssistantMessageEvent) *EventStream {
	return &EventStream{ch: ch}
}

// Next returns the next event from the stream.
// Returns false when the stream is exhausted or ctx is cancelled.
func (s *EventStream) Next(ctx context.Context) (types.AssistantMessageEvent, bool) {
	select {
	case ev, ok := <-s.ch:
		return ev, ok
	case <-ctx.Done():
		return types.AssistantMessageEvent{}, false
	}
}

// Chan returns the underlying channel.
func (s *EventStream) Chan() <-chan types.AssistantMessageEvent {
	return s.ch
}

// CollectMessage consumes the stream until a StreamEventDone or StreamEventError event,
// assembling the final AssistantMessage from deltas.
func CollectMessage(ctx context.Context, ch <-chan types.AssistantMessageEvent) (*types.AssistantMessage, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case ev, ok := <-ch:
			if !ok {
				return nil, io.EOF
			}
			switch ev.Type {
			case types.StreamEventDone:
				if ev.Message != nil {
					return ev.Message, nil
				}
				return nil, io.EOF
			case types.StreamEventError:
				if ev.Error != nil {
					return ev.Error, nil
				}
				return nil, io.EOF
			}
		}
	}
}
