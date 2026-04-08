package stream

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestSSEReaderBasicEvent(t *testing.T) {
	input := "data: hello\n\n"
	r := NewSSEReader(strings.NewReader(input))
	ev, err := r.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev == nil {
		t.Fatal("expected event, got nil")
	}
	if ev.Data != "hello" {
		t.Errorf("Data: got %q want %q", ev.Data, "hello")
	}

	// next read should return EOF
	_, err = r.ReadEvent()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestSSEReaderAllFields(t *testing.T) {
	input := "id: 42\nevent: ping\ndata: payload\nretry: 3000\n\n"
	r := NewSSEReader(strings.NewReader(input))
	ev, err := r.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.ID != "42" {
		t.Errorf("ID: got %q want %q", ev.ID, "42")
	}
	if ev.Event != "ping" {
		t.Errorf("Event: got %q want %q", ev.Event, "ping")
	}
	if ev.Data != "payload" {
		t.Errorf("Data: got %q want %q", ev.Data, "payload")
	}
	if ev.Retry != 3000 {
		t.Errorf("Retry: got %d want 3000", ev.Retry)
	}
}

func TestSSEReaderMultiLineData(t *testing.T) {
	input := "data: line1\ndata: line2\ndata: line3\n\n"
	r := NewSSEReader(strings.NewReader(input))
	ev, err := r.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "line1\nline2\nline3"
	if ev.Data != want {
		t.Errorf("Data: got %q want %q", ev.Data, want)
	}
}

func TestSSEReaderMultipleEvents(t *testing.T) {
	input := "data: first\n\ndata: second\n\n"
	r := NewSSEReader(strings.NewReader(input))

	ev1, err := r.ReadEvent()
	if err != nil {
		t.Fatalf("event1 error: %v", err)
	}
	if ev1.Data != "first" {
		t.Errorf("event1 Data: got %q want %q", ev1.Data, "first")
	}

	ev2, err := r.ReadEvent()
	if err != nil {
		t.Fatalf("event2 error: %v", err)
	}
	if ev2.Data != "second" {
		t.Errorf("event2 Data: got %q want %q", ev2.Data, "second")
	}

	_, err = r.ReadEvent()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestSSEReaderCommentIgnored(t *testing.T) {
	input := ": this is a comment\ndata: real\n\n"
	r := NewSSEReader(strings.NewReader(input))
	ev, err := r.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Data != "real" {
		t.Errorf("Data: got %q want %q", ev.Data, "real")
	}
}

func TestSSEReaderEmptyBody(t *testing.T) {
	r := NewSSEReader(strings.NewReader(""))
	ev, err := r.ReadEvent()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
	if ev != nil {
		t.Errorf("expected nil event, got %+v", ev)
	}
}

func TestSSEReaderNoTrailingBlankLine(t *testing.T) {
	// Event without trailing blank line (stream closed)
	input := "data: trailing"
	r := NewSSEReader(strings.NewReader(input))
	ev, err := r.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev == nil {
		t.Fatal("expected event from trailing data")
	}
	if ev.Data != "trailing" {
		t.Errorf("Data: got %q want %q", ev.Data, "trailing")
	}
}

func TestParseSSEStreamContextCancel(t *testing.T) {
	pr, pw := io.Pipe()

	// Write one event then block
	go func() {
		pw.Write([]byte("data: first\n\n")) //nolint
		// block until pipe is closed
		buf := make([]byte, 1)
		pr.Read(buf) //nolint
	}()

	ctx, cancel := context.WithCancel(context.Background())
	ch := ParseSSEStream(ctx, pr)

	// Read first event
	ev := <-ch
	if ev.Data != "first" {
		t.Errorf("Data: got %q want %q", ev.Data, "first")
	}

	// Cancel - channel should close
	cancel()
	pw.Close()

	// Drain channel; it must eventually close
	for range ch {
	}
}

func TestParseSSEStreamMultipleEvents(t *testing.T) {
	input := "data: a\n\ndata: b\n\ndata: c\n\n"
	body := io.NopCloser(strings.NewReader(input))
	ctx := context.Background()
	ch := ParseSSEStream(ctx, body)

	var events []SSEEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	expected := []string{"a", "b", "c"}
	for i, ev := range events {
		if ev.Data != expected[i] {
			t.Errorf("event[%d] Data: got %q want %q", i, ev.Data, expected[i])
		}
	}
}
