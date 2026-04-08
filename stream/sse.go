package stream

import (
	"bufio"
	"context"
	"io"
	"log"
	"strconv"
	"strings"
	"time"
)

// SSEEvent represents a single Server-Sent Events message.
type SSEEvent struct {
	ID    string
	Event string
	Data  string
	Retry int
}

// SSEReader reads SSE events from an io.Reader line by line.
type SSEReader struct {
	scanner *bufio.Scanner
	// accumulated fields for the current event
	id    string
	event string
	data  strings.Builder
	retry int
}

// NewSSEReader creates an SSEReader that parses SSE from r.
func NewSSEReader(r io.Reader) *SSEReader {
	return &SSEReader{scanner: bufio.NewScanner(r)}
}

// ReadEvent reads and returns the next SSE event.
// Returns nil, io.EOF when the stream ends.
func (r *SSEReader) ReadEvent() (*SSEEvent, error) {
	for r.scanner.Scan() {
		line := r.scanner.Text()
		if line == "" {
			// blank line = event boundary; dispatch if we have data
			if r.data.Len() > 0 || r.event != "" || r.id != "" {
				ev := &SSEEvent{
					ID:    r.id,
					Event: r.event,
					Data:  strings.TrimSuffix(r.data.String(), "\n"),
					Retry: r.retry,
				}
				// reset accumulated state
				r.id = ""
				r.event = ""
				r.data.Reset()
				r.retry = 0
				return ev, nil
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			// comment, ignore
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		// trim single leading space from value if present
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}
		switch field {
		case "id":
			r.id = value
		case "event":
			r.event = value
		case "data":
			if r.data.Len() > 0 {
				r.data.WriteByte('\n')
			}
			r.data.WriteString(value)
		case "retry":
			if n, err := strconv.Atoi(value); err == nil {
				r.retry = n
			}
		}
	}
	if err := r.scanner.Err(); err != nil {
		return nil, err
	}
	// flush any trailing event not terminated by blank line
	if r.data.Len() > 0 || r.event != "" || r.id != "" {
		ev := &SSEEvent{
			ID:    r.id,
			Event: r.event,
			Data:  strings.TrimSuffix(r.data.String(), "\n"),
			Retry: r.retry,
		}
		r.id = ""
		r.event = ""
		r.data.Reset()
		r.retry = 0
		return ev, nil
	}
	return nil, io.EOF
}

// ParseSSEStream reads SSE events from body and sends them on the returned channel.
// The channel is closed when body is exhausted, ctx is cancelled, or no event
// arrives within 3 minutes (stall timeout).
func ParseSSEStream(ctx context.Context, body io.ReadCloser) <-chan SSEEvent {
	ch := make(chan SSEEvent, 16)

	// Close body when ctx is cancelled to unblock scanner.Scan().
	go func() {
		<-ctx.Done()
		body.Close()
	}()

	go func() {
		defer close(ch)
		defer body.Close()
		reader := NewSSEReader(body)

		const stallTimeout = 3 * time.Minute

		for {
			if ctx.Err() != nil {
				return
			}

			// Read event in a goroutine with stall timeout.
			type readResult struct {
				ev  *SSEEvent
				err error
			}
			resultCh := make(chan readResult, 1)
			go func() {
				ev, err := reader.ReadEvent()
				resultCh <- readResult{ev, err}
			}()

			select {
			case <-ctx.Done():
				return
			case <-time.After(stallTimeout):
				log.Printf("[sse-stream] stall timeout (%v) — no SSE event received, closing body", stallTimeout)
				body.Close() // force-close to unblock the scanner
				return
			case res := <-resultCh:
				if res.err != nil {
					return
				}
				if res.ev == nil {
					continue
				}
				select {
				case ch <- *res.ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch
}
