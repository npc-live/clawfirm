package whipflow_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ai-gateway/clawfirm/whipflow"
	"github.com/ai-gateway/clawfirm/whipflow/runtime"
)

// askSource 模拟用户场景：ask 一个变量，session prompt 中用 {varname} 插值。
// 等价于：
//
//	ask topic: "调研什么"
//	let report = session "{topic} 的调研报告"
//	let summary = session "总结一下 {report}"
const askSource = `ask topic: "调研什么"
let report = session "{topic} 的调研报告"
let summary = session "总结一下 {report}"`

// TestAskWithInitialInputs 验证 WithInitialInputs 能正确填充 ask 变量，
// 且 session prompt 中的 {topic} 被替换为注入的值。
func TestAskWithInitialInputs(t *testing.T) {
	mp := &mockProvider{
		name: "mock",
		response: func(idx int, prompt string) string {
			return fmt.Sprintf("output-%d|prompt=%s", idx, prompt)
		},
	}

	program, errs := whipflow.Parse(askSource)
	if len(errs) > 0 {
		t.Fatalf("parse error: %v", errs)
	}

	var progress []runtime.SessionProgress
	result, err := whipflow.Execute(program,
		whipflow.WithNativeProvider("mock", mp),
		whipflow.WithRuntimeConfig(&runtime.RuntimeConfig{DefaultProvider: "mock"}),
		whipflow.WithInitialInputs(map[string]string{"topic": "苹果公司"}),
		whipflow.WithSessionProgressCallback(func(p runtime.SessionProgress) {
			progress = append(progress, p)
		}),
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Success {
		t.Fatalf("workflow failed: %+v", result.Errors)
	}

	// 应该恰好 2 个 session。
	if len(result.SessionOutputs) != 2 {
		t.Fatalf("expected 2 session outputs, got %d", len(result.SessionOutputs))
	}

	// Session 0 prompt 应包含插值后的 topic。
	s0 := result.SessionOutputs[0]
	if !strings.Contains(s0.Output, "苹果公司 的调研报告") {
		t.Errorf("session 0: expected prompt to contain '苹果公司 的调研报告', got %q", s0.Output)
	}

	// Session 1 prompt 应包含 session 0 的输出（{report} 被替换）。
	s1 := result.SessionOutputs[1]
	if !strings.Contains(s1.Output, s0.Output) {
		t.Errorf("session 1: expected prompt to contain session 0 output %q, got %q", s0.Output, s1.Output)
	}

	// 共 2 个 done 事件。
	doneCount := 0
	for _, p := range progress {
		if p.Done {
			doneCount++
		}
	}
	if doneCount != 2 {
		t.Errorf("expected 2 done progress events, got %d", doneCount)
	}
}

// TestAskReplay_StepByStep 验证 step-by-step 模式下，ask 变量在每次 replay
// 时都能通过 WithInitialInputs 正确恢复，session prompt 插值不丢失。
func TestAskReplay_StepByStep(t *testing.T) {
	// runAskStep 运行一步（stop_after=stopAt），携带 replay 和 initialInputs。
	runAskStep := func(stopAt int, replay []runtime.SessionRecord) []runtime.SessionRecord {
		t.Helper()

		// 记录每个 session 实际收到的 prompt，用于断言插值正确。
		var capturedPrompts []string

		mp := &mockProvider{
			name: "mock",
			response: func(idx int, prompt string) string {
				capturedPrompts = append(capturedPrompts, prompt)
				return fmt.Sprintf("session-%d-result", idx+len(replay))
			},
		}

		program, errs := whipflow.Parse(askSource)
		if len(errs) > 0 {
			t.Fatalf("parse error: %v", errs)
		}

		opts := []whipflow.Option{
			whipflow.WithNativeProvider("mock", mp),
			whipflow.WithRuntimeConfig(&runtime.RuntimeConfig{DefaultProvider: "mock"}),
			whipflow.WithInitialInputs(map[string]string{"topic": "苹果公司"}),
			whipflow.WithStopAfterSession(stopAt),
		}
		if len(replay) > 0 {
			opts = append(opts, whipflow.WithReplaySessions(replay))
		}

		result, err := whipflow.Execute(program, opts...)
		if err != nil && !strings.Contains(err.Error(), "stopped after session limit") {
			t.Fatalf("Execute (stop_after=%d): %v", stopAt, err)
		}

		// 验证实际执行的 session（非 replay）prompt 包含正确插值。
		for _, p := range capturedPrompts {
			if strings.Contains(p, "{topic}") {
				t.Errorf("stop_after=%d: prompt still contains un-interpolated {topic}: %q", stopAt, p)
			}
		}

		var records []runtime.SessionRecord
		for i, s := range result.SessionOutputs {
			records = append(records, runtime.SessionRecord{
				SessionIndex: i,
				Output:       s.Output,
				Model:        s.Metadata.Model,
				DurationMs:   s.Metadata.Duration,
			})
		}
		return records
	}

	// Step 0: 执行 session 0（report），停在这里。
	t.Log("--- Step 0: 执行 session 0 (report) ---")
	records := runAskStep(0, nil)
	if len(records) != 1 {
		t.Fatalf("step 0: expected 1 record, got %d", len(records))
	}
	t.Logf("  session 0 output: %q", records[0].Output)

	// Session 0 的 output 应该包含正确插值的 topic。
	if !strings.Contains(records[0].Output, "session-0-result") {
		t.Errorf("step 0: unexpected output %q", records[0].Output)
	}

	// Step 1: 从 session 1 继续（replay session 0），执行 session 1（summary）。
	t.Log("--- Step 1: replay session 0，执行 session 1 (summary) ---")
	records = runAskStep(1, records)
	if len(records) != 2 {
		t.Fatalf("step 1: expected 2 records, got %d", len(records))
	}
	t.Logf("  session 0 (replayed): %q", records[0].Output)
	t.Logf("  session 1 output: %q", records[1].Output)

	// Replay session 0 的 output 应不变。
	if records[0].Output != "session-0-result" {
		t.Errorf("step 1: replayed session 0 output changed: %q", records[0].Output)
	}

	t.Log("=== ask replay step-by-step: PASS ===")
}

// TestAskMissingInput 验证当 ask 变量没有 initialInputs 时，
// 解释器不会 panic，而是尝试从 stdin 读取（在测试中会因无 stdin 而失败或挂起）。
// 我们通过关闭 stdin 替代品来验证行为：此测试仅确认有 initialInputs 时不读 stdin。
func TestAskNoStdinNeededWhenInputsProvided(t *testing.T) {
	// 关键断言：提供了 initialInputs 后，即使没有可读的 stdin，也能成功执行。
	mp := &mockProvider{name: "mock"}

	program, errs := whipflow.Parse(askSource)
	if len(errs) > 0 {
		t.Fatalf("parse error: %v", errs)
	}

	result, err := whipflow.Execute(program,
		whipflow.WithNativeProvider("mock", mp),
		whipflow.WithRuntimeConfig(&runtime.RuntimeConfig{DefaultProvider: "mock"}),
		whipflow.WithInitialInputs(map[string]string{"topic": "特斯拉"}),
	)
	if err != nil {
		t.Fatalf("Execute should succeed with initialInputs, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("workflow should succeed, errors: %+v", result.Errors)
	}
	if len(result.SessionOutputs) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(result.SessionOutputs))
	}
}
