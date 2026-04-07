package whipflow_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/ai-gateway/clawfirm/whipflow"
)

func TestClaudeCodeProvider(t *testing.T) {
	// Skip if claw binary is not available (CI or dev without build).
	if _, err := exec.LookPath("claw"); err != nil {
		if _, err := os.Stat("../claw-code/rust/target/debug/claw"); err != nil {
			t.Skip("claw binary not found, skipping integration test")
		}
	}
	src := `let result = session "请用一句话回答：你是哪个AI模型？提到版本。"`

	prog, errs := whipflow.Parse(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	result, err := whipflow.Execute(prog,
		whipflow.WithSessionProgressCallback(func(p whipflow.SessionProgress) {
			if !p.Done {
				fmt.Printf("  [session %d] provider=%s starting...\n", p.Index, p.Provider)
			} else {
				fmt.Printf("  [session %d] provider=%s done (%dms)\n", p.Index, p.Provider, p.DurationMs)
				fmt.Printf("  Output: %s\n", p.Output)
				if p.Error != "" {
					fmt.Printf("  Error: %s\n", p.Error)
				}
			}
		}),
	)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if !result.Success {
		t.Fatalf("execution failed: %v", result.Errors)
	}
	if len(result.SessionOutputs) == 0 {
		t.Fatal("no session outputs")
	}
	t.Logf("Output: %s", result.SessionOutputs[0].Output)
}
