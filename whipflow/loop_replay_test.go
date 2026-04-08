package whipflow_test

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ai-gateway/clawfirm/whipflow"
	"github.com/ai-gateway/clawfirm/whipflow/runtime"
)

// mockProvider is a fake Provider that returns canned responses keyed by session index.
// It uses an atomic counter so concurrent use is safe.
type mockProvider struct {
	name     string
	counter  atomic.Int32
	response func(idx int, prompt string) string // if nil, returns "session-{idx}"
}

func (m *mockProvider) ProviderName() string { return m.name }

func (m *mockProvider) ExecuteSession(spec runtime.SessionSpec, cfg runtime.RuntimeConfig, enableTools bool, allowedTools []string, skillPrompts []string) (*runtime.SessionResult, error) {
	return m.ExecuteSessionStream(spec, cfg, enableTools, allowedTools, skillPrompts, nil)
}

func (m *mockProvider) ExecuteSessionStream(spec runtime.SessionSpec, cfg runtime.RuntimeConfig, enableTools bool, allowedTools []string, skillPrompts []string, onStream func(string)) (*runtime.SessionResult, error) {
	idx := int(m.counter.Add(1) - 1)
	var out string
	if m.response != nil {
		out = m.response(idx, spec.Prompt)
	} else {
		out = fmt.Sprintf("session-%d-output", idx)
	}
	if onStream != nil {
		onStream(out)
	}
	return &runtime.SessionResult{Output: out}, nil
}

const loopTestSource = "let items = \"apples,bananas,oranges\"\n" +
	"let intro = session \"intro: {items}\"\n" +
	"let summary = \"\"\n" +
	"repeat 3 as i:\n" +
	"  let fruit = session \"pick fruit {i} from {items}\"\n" +
	"  let analysis = session \"describe {fruit}\"\n" +
	"  let summary = session \"merge summary {summary} with {analysis}\"\n" +
	"let final = session \"write report: {summary}\""

// sessionLabels is for human-friendly test output.
var sessionLabels = []string{
	"[0] intro",
	"[1] fruit-1", "[2] analysis-1", "[3] summary-1",
	"[4] fruit-2", "[5] analysis-2", "[6] summary-2",
	"[7] fruit-3", "[8] analysis-3", "[9] summary-3",
	"[10] final",
}

func label(idx int) string {
	if idx < len(sessionLabels) {
		return sessionLabels[idx]
	}
	return fmt.Sprintf("[%d]", idx)
}

// runStopAfter runs loop_test source with a fresh mock provider, stopping after session `stopAt`.
// Returns the session outputs recorded so far.
func runStopAfter(t *testing.T, stopAt int, replay []runtime.SessionRecord, startCounter int) []runtime.SessionRecord {
	t.Helper()

	mp := &mockProvider{name: "mock"}
	mp.counter.Store(int32(startCounter))

	var progress []runtime.SessionProgress
	program, parseErrs := whipflow.Parse(loopTestSource)
	if len(parseErrs) > 0 {
		t.Fatalf("parse error: %v", parseErrs)
	}

	opts := []whipflow.Option{
		whipflow.WithNativeProvider("mock", mp),
		whipflow.WithRuntimeConfig(&runtime.RuntimeConfig{DefaultProvider: "mock"}),
		whipflow.WithSessionProgressCallback(func(p whipflow.SessionProgress) {
			progress = append(progress, p)
			if p.Done {
				t.Logf("  %s done → %q", label(p.Index), truncateStr(p.Output, 60))
			}
		}),
		whipflow.WithStopAfterSession(stopAt),
	}
	if len(replay) > 0 {
		opts = append(opts, whipflow.WithReplaySessions(replay))
	}

	result, err := whipflow.Execute(program, opts...)
	if err != nil && !strings.Contains(err.Error(), "stopped after session limit") {
		t.Fatalf("Execute (stop_after=%d): unexpected error: %v", stopAt, err)
	}

	// Collect session_outputs from result.
	var records []runtime.SessionRecord
	for _, s := range result.SessionOutputs {
		records = append(records, runtime.SessionRecord{
			SessionIndex: len(records),
			Output:       s.Output,
			Model:        s.Metadata.Model,
			DurationMs:   s.Metadata.Duration,
		})
	}
	t.Logf("  → %d session outputs collected", len(records))
	return records
}

// TestLoopReplay_StepByStep walks through loop_test one session at a time,
// verifying that replay correctly restores state at each step.
func TestLoopReplay_StepByStep(t *testing.T) {
	t.Log("=== Step-by-step: running 11 sessions one at a time ===")

	var accumulated []runtime.SessionRecord

	for step := 0; step <= 10; step++ {
		t.Logf("--- Step %s ---", label(step))
		records := runStopAfter(t, step, accumulated, len(accumulated))

		if len(records) != step+1 {
			t.Fatalf("step %d: expected %d records, got %d", step, step+1, len(records))
		}

		// Verify last record is the one just executed.
		last := records[step]
		expectedOutput := fmt.Sprintf("session-%d-output", step)
		if last.Output != expectedOutput {
			t.Errorf("step %d: expected output %q, got %q", step, expectedOutput, last.Output)
		}

		accumulated = records
	}

	t.Log("=== All 11 sessions completed step-by-step ===")
	t.Logf("Final accumulated outputs: %d sessions", len(accumulated))
	for i, r := range accumulated {
		t.Logf("  %s: %q", label(i), r.Output)
	}
}

// TestLoopReplay_MidRunRecovery simulates interruption after session 5 (middle of loop)
// and verifies that resuming from session 6 produces the correct continuation.
func TestLoopReplay_MidRunRecovery(t *testing.T) {
	t.Log("=== Mid-run recovery: run to session 5, then resume from 6 ===")

	// Phase 1: Run sessions 0-5 (stop after session 5).
	t.Log("--- Phase 1: running sessions 0-5 ---")
	records05 := runStopAfter(t, 5, nil, 0)

	if len(records05) != 6 {
		t.Fatalf("phase 1: expected 6 records (0-5), got %d", len(records05))
	}
	for i, r := range records05 {
		t.Logf("  %s: %q", label(i), r.Output)
	}

	// Verify sessions 0-5 have expected outputs.
	for i, r := range records05 {
		expected := fmt.Sprintf("session-%d-output", i)
		if r.Output != expected {
			t.Errorf("phase 1 session %d: expected %q, got %q", i, expected, r.Output)
		}
	}

	// Phase 2: Resume from session 6, replaying sessions 0-5 from "message history".
	t.Log("--- Phase 2: resuming from session 6 (replaying 0-5) ---")
	// The mock provider counter starts at 6 (sessions 0-5 are replayed, not re-executed).
	recordsFull := runStopAfter(t, 10, records05, 6)

	if len(recordsFull) != 11 {
		t.Fatalf("phase 2: expected 11 total records, got %d", len(recordsFull))
	}

	// Verify replayed sessions (0-5) are unchanged.
	for i := 0; i <= 5; i++ {
		expected := fmt.Sprintf("session-%d-output", i)
		if recordsFull[i].Output != expected {
			t.Errorf("phase 2 replayed session %d: expected %q, got %q", i, expected, recordsFull[i].Output)
		}
	}

	// Verify resumed sessions (6-10) have correct outputs.
	for i := 6; i <= 10; i++ {
		expected := fmt.Sprintf("session-%d-output", i)
		if recordsFull[i].Output != expected {
			t.Errorf("phase 2 resumed session %d: expected %q, got %q", i, expected, recordsFull[i].Output)
		}
	}

	t.Log("=== Mid-run recovery: PASS ===")
	t.Logf("All 11 sessions verified:")
	for i, r := range recordsFull {
		t.Logf("  %s: %q", label(i), r.Output)
	}
}

// TestLoopReplay_FullRun runs the entire loop in one shot and verifies all 11 outputs.
func TestLoopReplay_FullRun(t *testing.T) {
	t.Log("=== Full run: all 11 sessions ===")

	mp := &mockProvider{name: "mock"}
	var progress []runtime.SessionProgress

	program, parseErrs := whipflow.Parse(loopTestSource)
	if len(parseErrs) > 0 {
		t.Fatalf("parse error: %v", parseErrs)
	}

	result, err := whipflow.Execute(program,
		whipflow.WithNativeProvider("mock", mp),
		whipflow.WithRuntimeConfig(&runtime.RuntimeConfig{DefaultProvider: "mock"}),
		whipflow.WithSessionProgressCallback(func(p whipflow.SessionProgress) {
			progress = append(progress, p)
		}),
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Success {
		t.Fatalf("workflow failed: %+v", result.Errors)
	}

	doneCount := 0
	for _, p := range progress {
		if p.Done {
			doneCount++
		}
	}
	if doneCount != 11 {
		t.Errorf("expected 11 done events, got %d", doneCount)
	}
	if len(result.SessionOutputs) != 11 {
		t.Errorf("expected 11 session outputs, got %d", len(result.SessionOutputs))
	}

	for i, s := range result.SessionOutputs {
		expected := fmt.Sprintf("session-%d-output", i)
		if s.Output != expected {
			t.Errorf("session %d: expected %q, got %q", i, expected, s.Output)
		}
		t.Logf("  %s: %q", label(i), s.Output)
	}

	t.Log("=== Full run: PASS ===")
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
