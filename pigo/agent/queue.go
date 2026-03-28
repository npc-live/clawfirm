package agent

import (
	"sync"

	"github.com/ai-gateway/pi-go/types"
)

// messageQueue is a thread-safe FIFO for pending messages.
type messageQueue struct {
	mu   sync.Mutex
	msgs []types.Message
	mode string // "all" | "one-at-a-time"
}

func newMessageQueue(mode string) *messageQueue {
	if mode == "" {
		mode = "all"
	}
	return &messageQueue{mode: mode}
}

// Enqueue adds a message to the queue.
func (q *messageQueue) Enqueue(m types.Message) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.msgs = append(q.msgs, m)
}

// Drain returns and removes pending messages.
// In "one-at-a-time" mode, only one message is returned per call.
// In "all" mode, all messages are returned and cleared.
func (q *messageQueue) Drain() []types.Message {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.msgs) == 0 {
		return nil
	}
	if q.mode == "one-at-a-time" {
		msg := q.msgs[0]
		q.msgs = q.msgs[1:]
		return []types.Message{msg}
	}
	out := q.msgs
	q.msgs = nil
	return out
}

// Len returns the number of queued messages.
func (q *messageQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.msgs)
}

// Clear removes all queued messages.
func (q *messageQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.msgs = nil
}

// SetMode changes the drain mode ("all" or "one-at-a-time").
func (q *messageQueue) SetMode(mode string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.mode = mode
}
