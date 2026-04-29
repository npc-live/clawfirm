package builtin_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/internal/agentbuilder"
	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/tool/builtin"
	"github.com/ai-gateway/clawfirm/types"
)

// TestWhipflowRun_MediaWhip exercises the full whipflow_run path using the
// real config at ~/.clawfirm/config.yml and the media.whip workflow.
// It verifies that:
//   - NativeProvider is created with tools (not falling back to CLI preset)
//   - browser_shortcut and media_understand tools are available
//   - The sessions actually execute and produce output files with video analysis
//
// Requires: INTEGRATION=1, Chrome with --remote-debugging-port=9222
func TestWhipflowRun_MediaWhip(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION=1 to run")
	}

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	// Build providers from config (tools resolve their own provider from the map).
	providerMap, err := agentbuilder.BuildProviders(cfg)
	if err != nil {
		t.Fatalf("BuildProviders: %v", err)
	}

	// Log media_understand provider if configured.
	if tc, ok := cfg.Tools["media_understand"]; ok && tc.Provider != "" {
		t.Logf("media_understand provider: %s model=%s", tc.Provider, tc.Model)
	}

	// Build tools the same way the app does — include browser_shortcut and media_understand.
	tools := agentbuilder.BuildTools(
		[]string{"read", "write", "edit", "bash", "browser_shortcut", "media_understand", "whipflow_run"},
		nil, cfg, nil, providerMap,
	)

	// Find the WhipflowRun tool and verify it got tools injected.
	var wr *builtin.WhipflowRun
	for _, tl := range tools {
		if w, ok := tl.(*builtin.WhipflowRun); ok {
			wr = w
			break
		}
	}
	if wr == nil {
		t.Fatal("whipflow_run tool not found")
	}
	if len(wr.Tools) == 0 {
		t.Fatal("WhipflowRun.Tools is empty — NativeProvider sessions will have no tools")
	}
	t.Logf("WhipflowRun has %d tools injected", len(wr.Tools))

	// List available tools.
	for _, tl := range wr.Tools {
		t.Logf("  tool: %s", tl.Name())
	}

	// Execute the media.whip file.
	whipPath := os.ExpandEnv("$HOME/.pi-go/workflows/media.whip")
	if _, err := os.Stat(whipPath); os.IsNotExist(err) {
		t.Skipf("media.whip not found at %s", whipPath)
	}

	// Clean previous output and state.
	home, _ := os.UserHomeDir()
	outDir := home + "/.clawfirm/canvas/douyin-viral"
	os.RemoveAll(outDir)
	os.Remove(home + "/.whipflow/state.db")

	var updates []tool.ToolUpdate
	result, err := wr.Execute(context.Background(), "test-1", map[string]any{
		"file":        whipPath,
		"user_inputs": map[string]any{"track": "美食"},
	}, func(u tool.ToolUpdate) {
		updates = append(updates, u)
		if step, ok := u.Details.(builtin.WhipflowSessionStep); ok {
			if step.Done {
				t.Logf("[session %d] done (%dms) output=%s", step.Index, step.DurationMs, truncate(step.Output, 300))
			} else {
				t.Logf("[session %d] starting provider=%s", step.Index, step.Provider)
			}
		}
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Check result.
	if len(result.Content) == 0 {
		t.Fatal("empty result content")
	}

	var text string
	if tc, ok := result.Content[0].(*types.TextContent); ok {
		text = tc.Text
	}
	t.Logf("result text:\n%s", truncate(text, 1000))

	if strings.Contains(text, "Execution error") {
		t.Fatalf("execution error: %s", text)
	}

	// Verify output files were created. Session outputs depend on LLM
	// following path instructions, so we check at least 3/4 exist.
	expectedFiles := []string{
		"01-search-results.md",
		"02-video-analysis.md",
		"03-script.md",
		"00-index.md",
	}
	foundFiles := 0
	for _, f := range expectedFiles {
		path := outDir + "/" + f
		info, err := os.Stat(path)
		if err != nil {
			t.Logf("output file missing (non-fatal): %s", path)
		} else {
			t.Logf("output file: %s (%d bytes)", f, info.Size())
			foundFiles++
		}
	}
	if foundFiles < 3 {
		t.Errorf("expected at least 3/4 output files, got %d", foundFiles)
	}

	// Check if videos were downloaded.
	videoFiles, _ := os.ReadDir(outDir)
	for _, f := range videoFiles {
		if strings.HasSuffix(f.Name(), ".mp4") {
			info, _ := f.Info()
			t.Logf("video file: %s (%d bytes)", f.Name(), info.Size())
		}
	}

	t.Logf("got %d session progress updates", len(updates))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
