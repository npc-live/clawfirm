// cmd/run-whip — thin runner that calls whipflow.RunFile with user_inputs,
// mirroring what WhipflowRun.Execute does when mode="execute".
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/whipflow"
	"github.com/ai-gateway/clawfirm/whipflow/runtime"
)

func main() {
	// Inputs — identical to the whipflow_run tool call requested by the user.
	filePath := "/Users/qing/.clawfirm/workflows/media.whip"
	userInputs := map[string]string{
		"track":            "「AI 工具」",
		"ref_audio_url":    "/Users/qing/projects/pi-go/cover-asset/Olivia-CN2.m4a",
		"avatar_image_url": "/Users/qing/projects/pi-go/cover-asset/oliva.jpg",
	}

	// Load clawfirm config so providers/agents defined in ~/.clawfirm/config.yml are available.
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintln(os.Stderr, "config load error:", err)
		// Non-fatal — continue with empty config.
		cfg = &config.Config{}
	}

	var execOpts []whipflow.Option

	// Attach config so WhipFlow can resolve native agents.
	execOpts = append(execOpts, whipflow.WithPiConfig(cfg))

	// Build runtime config mirroring WhipflowRun.Execute logic.
	rCfg := runtime.DefaultRuntimeConfig()
	if cfg.Whipflow.DefaultProvider != "" {
		rCfg.DefaultProvider = cfg.Whipflow.DefaultProvider
	} else {
		defaultProvider := cfg.DefaultAgent
		if defaultProvider == "" && len(cfg.Agents) > 0 {
			defaultProvider = cfg.Agents[0].Name
		}
		if defaultProvider != "" {
			rCfg.DefaultProvider = defaultProvider
		}
	}
	execOpts = append(execOpts, whipflow.WithRuntimeConfig(&rCfg))

	// Pre-fill ask inputs.
	execOpts = append(execOpts, whipflow.WithInitialInputs(userInputs))

	// Print session progress to stdout as sessions complete.
	execOpts = append(execOpts, whipflow.WithSessionProgressCallback(func(p whipflow.SessionProgress) {
		if !p.Done {
			fmt.Printf("\n=== Session %d starting: %s (provider: %s) ===\n", p.Index, p.Name, p.Provider)
			fmt.Printf("Prompt: %s\n", truncate(p.Prompt, 300))
		} else {
			fmt.Printf("\n=== Session %d complete (%dms) ===\n", p.Index, p.DurationMs)
			if p.Error != "" {
				fmt.Printf("Error: %s\n", p.Error)
			} else {
				fmt.Printf("Output: %s\n", truncate(p.Output, 1000))
			}
		}
	}))

	log.Printf("Executing workflow: %s", filePath)
	log.Printf("user_inputs: %v", userInputs)

	result, err := whipflow.RunFile(filePath, execOpts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nExecution error: %v\n", err)
		os.Exit(1)
	}

	// Print full result summary.
	fmt.Printf("\n=== Workflow Result ===\n")
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Sessions created: %d\n", result.Metadata.SessionsCreated)
	fmt.Printf("Statements executed: %d\n", result.Metadata.StatementsExecuted)
	fmt.Printf("Total duration: %dms\n", result.Metadata.Duration)

	if len(result.SessionOutputs) > 0 {
		fmt.Printf("\n--- Session Outputs ---\n")
		for i, s := range result.SessionOutputs {
			fmt.Printf("\n[Session %d] model=%s duration=%dms\n%s\n",
				i+1, s.Metadata.Model, s.Metadata.Duration, s.Output)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\n--- Errors ---\n")
		for _, e := range result.Errors {
			fmt.Printf("  [%s] %s\n", e.Type, e.Message)
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "... [truncated]"
}
