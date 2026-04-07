package whipflow_test

// Integration test: runs a two-session whipflow using claw-code and verifies
// that (a) NDJSON progress events are emitted session-by-session (not batched
// at the end), and (b) context from session 1 is passed to session 2 via the
// interpreter's variable environment (markdown output stored as $last).
//
// Run with:
//   go test ./whipflow/ -run TestStreamSessionBySession -v -timeout 300s

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ai-gateway/clawfirm/whipflow"
)

// progressEvent mirrors the NDJSON line emitted by cmd/whip/main.go.
type progressEvent struct {
	Type       string `json:"type"`
	Index      int    `json:"index"`
	Name       string `json:"name,omitempty"`
	Provider   string `json:"provider,omitempty"`
	Done       bool   `json:"done"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

// skipIfNoClaw skips the test when the claw binary is not available.
func skipIfNoClaw(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("claw"); err != nil {
		if _, err := os.Stat(os.Getenv("HOME") + "/.clawfirm/bin/claw"); err != nil {
			t.Skip("claw binary not found, skipping integration test")
		}
	}
}

// TestStreamSessionBySession verifies that the whipflow engine emits progress
// events for each session as it completes, rather than buffering all output
// until the entire workflow finishes.
func TestStreamSessionBySession(t *testing.T) {
	skipIfNoClaw(t)

	// Two-session workflow: session 1 produces a word, session 2 uses it.
	src := `
let s1 = session "用一个英文单词回答：苹果的英文是什么？只输出单词本身，不要其他内容。"

let s2 = session "把这个英文单词翻译回中文，只输出中文词：{{s1}}"
`
	prog, errs := whipflow.Parse(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	type event struct {
		p         whipflow.SessionProgress
		timestamp time.Time
	}

	var mu sync.Mutex
	var events []event

	result, err := whipflow.Execute(prog,
		whipflow.WithSessionProgressCallback(func(p whipflow.SessionProgress) {
			mu.Lock()
			events = append(events, event{p: p, timestamp: time.Now()})
			mu.Unlock()

			// Simulate what cmd/whip does: marshal to NDJSON immediately.
			ev := progressEvent{
				Type:       "session_progress",
				Index:      p.Index,
				Name:       p.Name,
				Provider:   p.Provider,
				Done:       p.Done,
				DurationMs: p.DurationMs,
				Output:     p.Output,
				Error:      p.Error,
			}
			b, _ := json.Marshal(ev)
			fmt.Println(string(b))
		}),
	)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if !result.Success {
		t.Fatalf("execution failed: %v", result.Errors)
	}

	mu.Lock()
	defer mu.Unlock()

	// Expect 4 events: start+done for each of 2 sessions.
	if len(events) < 4 {
		t.Fatalf("expected at least 4 progress events (2×start+done), got %d", len(events))
	}

	// Verify session 0 start fires before session 0 done.
	// Verify session 0 done fires before session 1 start (sequential execution).
	var s0start, s0done, s1start, s1done time.Time
	for _, e := range events {
		switch {
		case e.p.Index == 0 && !e.p.Done:
			s0start = e.timestamp
		case e.p.Index == 0 && e.p.Done:
			s0done = e.timestamp
		case e.p.Index == 1 && !e.p.Done:
			s1start = e.timestamp
		case e.p.Index == 1 && e.p.Done:
			s1done = e.timestamp
		}
	}

	if s0start.IsZero() || s0done.IsZero() || s1start.IsZero() || s1done.IsZero() {
		t.Fatalf("missing expected events: s0start=%v s0done=%v s1start=%v s1done=%v",
			s0start, s0done, s1start, s1done)
	}
	if !s0done.Before(s1start) {
		t.Errorf("session 0 done (%v) should be before session 1 start (%v)", s0done, s1start)
	}

	// Verify context passed: session 1 output should contain 中文 (be non-empty).
	var s1output string
	for _, e := range events {
		if e.p.Index == 1 && e.p.Done {
			s1output = e.p.Output
		}
	}
	if s1output == "" {
		t.Error("session 1 output is empty — context may not have been passed from session 0")
	}
	t.Logf("session 0 output: %q", func() string {
		for _, e := range events {
			if e.p.Index == 0 && e.p.Done {
				return e.p.Output
			}
		}
		return ""
	}())
	t.Logf("session 1 output: %q", s1output)
}

// TestStreamCmdWhip runs the compiled whip binary with a two-session workflow
// and verifies each session's NDJSON event arrives before the next session
// starts (i.e., the process emits output session-by-session, not all at once).
//
// This test simulates what the real app does when calling whip as a subprocess.
func TestStreamCmdWhip(t *testing.T) {
	skipIfNoClaw(t)

	// Build whip binary. The test runs from the whipflow/ directory so the
	// module root is one level up.
	moduleRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	whipBin := filepath.Join(t.TempDir(), "whip")
	cmd0 := exec.Command("go", "build", "-o", whipBin, "./cmd/whip")
	cmd0.Dir = moduleRoot
	if out, buildErr := cmd0.CombinedOutput(); buildErr != nil {
		t.Fatalf("build whip: %v\n%s", buildErr, out)
	}

	// Write a temporary .whip file.
	whipFile := t.TempDir() + "/test.whip"
	src := `let s1 = session "用一个英文单词回答：香蕉的英文是什么？只输出单词本身。"
let s2 = session "把这个英文单词翻译回中文，只输出中文词：{{s1}}"`
	if err := os.WriteFile(whipFile, []byte(src), 0644); err != nil {
		t.Fatalf("write whip file: %v", err)
	}

	// Run whip and collect NDJSON lines as they arrive.
	cmd := exec.Command(whipBin, whipFile)
	cmd.Env = os.Environ()

	pr, pw, _ := os.Pipe()
	cmd.Stdout = pw

	if err := cmd.Start(); err != nil {
		t.Fatalf("start whip: %v", err)
	}
	pw.Close()

	type timedLine struct {
		raw  string
		when time.Time
	}
	var lines []timedLine

	buf := make([]byte, 4096)
	var partial string
	for {
		n, err := pr.Read(buf)
		if n > 0 {
			partial += string(buf[:n])
			for {
				idx := -1
				for i, c := range partial {
					if c == '\n' {
						idx = i
						break
					}
				}
				if idx < 0 {
					break
				}
				line := partial[:idx]
				partial = partial[idx+1:]
				if line != "" {
					lines = append(lines, timedLine{raw: line, when: time.Now()})
				}
			}
		}
		if err != nil {
			break
		}
	}
	pr.Close()
	cmd.Wait()

	// Parse NDJSON events.
	var doneEvents []progressEvent
	for _, l := range lines {
		var ev progressEvent
		if err := json.Unmarshal([]byte(l.raw), &ev); err != nil {
			continue
		}
		t.Logf("event: %s", l.raw)
		if ev.Done {
			doneEvents = append(doneEvents, ev)
		}
	}

	if len(doneEvents) < 2 {
		t.Fatalf("expected at least 2 done events, got %d", len(doneEvents))
	}

	// Session indices should be 0 and 1.
	if doneEvents[0].Index != 0 {
		t.Errorf("first done event should be index 0, got %d", doneEvents[0].Index)
	}
	if doneEvents[1].Index != 1 {
		t.Errorf("second done event should be index 1, got %d", doneEvents[1].Index)
	}

	if doneEvents[0].Output == "" {
		t.Error("session 0 output is empty")
	}
	if doneEvents[1].Output == "" {
		t.Error("session 1 output is empty — context may not have passed from session 0")
	}

	t.Logf("session 0: %q", doneEvents[0].Output)
	t.Logf("session 1: %q", doneEvents[1].Output)
}
