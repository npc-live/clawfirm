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
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &SSEReader{scanner: s}
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

		streamStart := time.Now()
		eventCount := 0

		log.Printf("[sse-stream] starting reader goroutine at %v", streamStart.Format(time.RFC3339Nano))

		for {
			if ctx.Err() != nil {
				log.Printf("[sse-stream] ctx cancelled after %v (events=%d)", time.Since(streamStart).Round(time.Millisecond), eventCount)
				return
			}

			// Read event in a goroutine with stall timeout.
			type readResult struct {
				ev  *SSEEvent
				err error
			}
			resultCh := make(chan readResult, 1)
			readStart := time.Now()
			go func() {
				ev, err := reader.ReadEvent()
				resultCh <- readResult{ev, err}
			}()

			select {
			case <-ctx.Done():
				log.Printf("[sse-stream] ctx cancelled while waiting for event (elapsed=%v, events=%d)",
					time.Since(streamStart).Round(time.Millisecond), eventCount)
				return
			case <-time.After(stallTimeout):
				log.Printf("[sse-stream] stall timeout (%v) after %v elapsed — no SSE event received (events=%d), closing body",
					stallTimeout, time.Since(streamStart).Round(time.Millisecond), eventCount)
				body.Close() // force-close to unblock the scanner
				return
			case res := <-resultCh:
				readDuration := time.Since(readStart)
				elapsed := time.Since(streamStart)
				if res.err != nil {
					// Log the exact error type to distinguish RST/EOF/timeout
					log.Printf("[sse-stream] ReadEvent error after %v (events=%d, read_took=%v): %T: %v",
						elapsed.Round(time.Millisecond), eventCount, readDuration.Round(time.Millisecond), res.err, res.err)
					return
				}
				if res.ev == nil {
					continue
				}
				eventCount++
				evType := res.ev.Event
				if evType == "" {
					evType = "(no-event-field)"
				}
				dataLen := len(res.ev.Data)
				log.Printf("[sse-stream] event #%d type=%q data_len=%d elapsed=%v read_took=%v",
					eventCount, evType, dataLen, elapsed.Round(time.Millisecond), readDuration.Round(time.Millisecond))
				select {
				case ch <- *res.ev:
				case <-ctx.Done():
					log.Printf("[sse-stream] ctx cancelled while sending event #%d to channel (elapsed=%v)",
						eventCount, elapsed.Round(time.Millisecond))
					return
				}
			}
		}
	}()
	return ch
}
