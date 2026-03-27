package stream

import (
	"bufio"
	"context"
	"io"
	"strconv"
	"strings"
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
// The channel is closed when body is exhausted or ctx is cancelled.
func ParseSSEStream(ctx context.Context, body io.ReadCloser) <-chan SSEEvent {
	ch := make(chan SSEEvent, 16)
	go func() {
		defer close(ch)
		defer body.Close()
		reader := NewSSEReader(body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			ev, err := reader.ReadEvent()
			if err != nil {
				return
			}
			if ev == nil {
				continue
			}
			select {
			case ch <- *ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}
