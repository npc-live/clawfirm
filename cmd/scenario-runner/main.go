// Command scenario-runner executes YAML test scenarios against the clawfirm
// gateway WebSocket API and outputs a JSON result.
//
// Usage:
//
//	go run ./cmd/scenario-runner scenarios/send_then_stop.yaml
//	go run ./cmd/scenario-runner -server ws://localhost:9988 scenarios/*.yaml
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ai-gateway/clawfirm/cmd/scenario-runner/runner"
)

func main() {
	server := flag.String("server", "ws://localhost:9988", "gateway WebSocket base URL")
	verbose := flag.Bool("v", false, "verbose: print events to stderr")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: scenario-runner [flags] <scenario.yaml> ...")
		os.Exit(1)
	}

	var results []*runner.Result
	allPassed := true

	for _, pattern := range flag.Args() {
		files, err := filepath.Glob(pattern)
		if err != nil || len(files) == 0 {
			files = []string{pattern}
		}
		for _, f := range files {
			sc, err := runner.LoadScenario(f)
			if err != nil {
				log.Printf("load %s: %v", f, err)
				results = append(results, &runner.Result{
					Scenario: f,
					Passed:   false,
					Error:    err.Error(),
				})
				allPassed = false
				continue
			}
			sc.Server = defaultServer(sc.Server, *server)

			r := runner.Run(sc, *verbose)
			results = append(results, r)
			if !r.Passed {
				allPassed = false
			}
			printSummary(r)
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(results)

	if !allPassed {
		os.Exit(1)
	}
}

func defaultServer(scenarioServer, flagServer string) string {
	if scenarioServer != "" {
		return scenarioServer
	}
	return flagServer
}

func printSummary(r *runner.Result) {
	status := "PASS"
	if !r.Passed {
		status = "FAIL"
	}
	var failed []string
	for _, a := range r.Assertions {
		if !a.Passed {
			failed = append(failed, a.Name+": "+a.Error)
		}
	}
	msg := fmt.Sprintf("[%s] %s (%dms)", status, r.Scenario, r.DurationMs)
	if r.Error != "" {
		msg += " — " + r.Error
	} else if len(failed) > 0 {
		msg += " — " + strings.Join(failed, "; ")
	}
	fmt.Fprintln(os.Stderr, msg)
	_ = time.Now() // keep import
}
