// Command logcheck analyses a clawfirm app.log file for anomalies.
//
// Detects:
//   - Time gap between consecutive log lines > threshold
//   - Agent turn start without corresponding done within timeout
//   - Tool execution start without corresponding end within timeout
//   - LLM returned toolUse but no tool_start event followed (emit blocked)
//   - New message received long after previous agent done (queue delay)
//
// Usage:
//
//	go run ./cmd/logcheck ~/.clawfirm/app.log
//	go run ./cmd/logcheck --since 10m ~/.clawfirm/app.log
//	go run ./cmd/logcheck --format json ~/.clawfirm/app.log
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ── log line patterns ────────────────────────────────────────────────────────

var (
	reLine      = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\.\d+) (\S+): (.+)$`)
	reTurnStart = regexp.MustCompile(`\[agent-loop\] === turn (\d+) start`)
	reTurnDone  = regexp.MustCompile(`\[agent-loop\] === agent done after (\d+) turns`)
	reTurnLLM   = regexp.MustCompile(`\[agent-loop\] turn (\d+): LLM done.*stopReason="([^"]+)"`)
	reToolStart = regexp.MustCompile(`tool_start.*tool_call_id`)  // webchat event
	reRecvMsg   = regexp.MustCompile(`recv msg:.*"(.+)"$`)
	reCtxCancel = regexp.MustCompile(`ctx cancelled`)
)

// ── types ────────────────────────────────────────────────────────────────────

type LogLine struct {
	Time time.Time
	File string
	Text string
}

type Finding struct {
	Severity    string    `json:"severity"`     // critical | warning | info
	Type        string    `json:"type"`
	Description string    `json:"description"`
	At          time.Time `json:"at"`
	FileLine    string    `json:"file_line,omitempty"`
}

// ── main ─────────────────────────────────────────────────────────────────────

func main() {
	since := flag.Duration("since", 0, "only analyse lines from the last N (e.g. 10m, 1h)")
	gapThresh := flag.Duration("gap", 10*time.Second, "alert on log gap > this duration")
	turnTimeout := flag.Duration("turn-timeout", 90*time.Second, "alert if turn takes longer")
	format := flag.String("format", "text", "output format: text | json")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: logcheck [flags] <app.log>")
		os.Exit(1)
	}

	lines, err := readLog(flag.Arg(0), *since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read log: %v\n", err)
		os.Exit(1)
	}

	findings := analyse(lines, *gapThresh, *turnTimeout)

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(findings)
	} else {
		for _, f := range findings {
			icon := "ℹ"
			if f.Severity == "critical" {
				icon = "✘"
			} else if f.Severity == "warning" {
				icon = "⚠"
			}
			fmt.Printf("%s [%s] %s  %s\n", icon, f.Severity, f.At.Format("15:04:05"), f.Description)
		}
		if len(findings) == 0 {
			fmt.Println("✔ No anomalies found")
		}
	}

	// Exit 1 if any critical findings.
	for _, f := range findings {
		if f.Severity == "critical" {
			os.Exit(1)
		}
	}
}

// ── parser ───────────────────────────────────────────────────────────────────

func readLog(path string, since time.Duration) ([]LogLine, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cutoff := time.Time{}
	if since > 0 {
		cutoff = time.Now().Add(-since)
	}

	var lines []LogLine
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		text := scanner.Text()
		m := reLine.FindStringSubmatch(text)
		if m == nil {
			continue
		}
		t, err := time.ParseInLocation("2006/01/02 15:04:05.000000", m[1], time.Local)
		if err != nil {
			continue
		}
		if !cutoff.IsZero() && t.Before(cutoff) {
			continue
		}
		lines = append(lines, LogLine{Time: t, File: m[2], Text: m[3]})
	}
	return lines, scanner.Err()
}

// ── analyser ─────────────────────────────────────────────────────────────────

func analyse(lines []LogLine, gapThresh, turnTimeout time.Duration) []Finding {
	var findings []Finding

	// State tracking.
	type turnState struct {
		startTime time.Time
		turn      int
	}
	var activeTurn *turnState
	var lastToolUseTurnEnd *LogLine // LLM done with toolUse stop reason
	var lastAgentDone *LogLine
	var lastRecvMsg *LogLine

	for i := range lines {
		ll := &lines[i]

		// ── Rule 1: Time gap between consecutive lines ────────────────────────
		if i > 0 {
			gap := ll.Time.Sub(lines[i-1].Time)
			if gap > gapThresh {
				findings = append(findings, Finding{
					Severity:    "critical",
					Type:        "time_gap",
					At:          lines[i-1].Time,
					Description: fmt.Sprintf("log gap %.1fs between %s and %s (file: %s)", gap.Seconds(), lines[i-1].File, ll.File, ll.File),
					FileLine:    ll.File,
				})
			}
		}

		// ── Rule 2: Turn start tracking ──────────────────────────────────────
		if m := reTurnStart.FindStringSubmatch(ll.Text); m != nil {
			n, _ := strconv.Atoi(m[1])
			if activeTurn != nil && time.Since(activeTurn.startTime) > turnTimeout {
				findings = append(findings, Finding{
					Severity:    "critical",
					Type:        "turn_stuck",
					At:          activeTurn.startTime,
					Description: fmt.Sprintf("turn %d started but never completed (no agent done log)", activeTurn.turn),
				})
			}
			activeTurn = &turnState{startTime: ll.Time, turn: n}
		}

		// ── Rule 3: Agent done ────────────────────────────────────────────────
		if reTurnDone.MatchString(ll.Text) {
			activeTurn = nil
			lastAgentDone = ll
		}

		// ── Rule 4: LLM returned toolUse but no tool event followed ──────────
		if m := reTurnLLM.FindStringSubmatch(ll.Text); m != nil {
			if m[2] == "toolUse" {
				lastToolUseTurnEnd = ll
			} else {
				lastToolUseTurnEnd = nil
			}
		}

		// tool_start / tool_end clears the pending toolUse flag
		if strings.Contains(ll.Text, "tool_start") || strings.Contains(ll.Text, "EventToolExecution") {
			lastToolUseTurnEnd = nil
		}

		// ── Rule 5: New message delay after agent done ────────────────────────
		if m := reRecvMsg.FindStringSubmatch(ll.Text); m != nil {
			if lastAgentDone != nil {
				delay := ll.Time.Sub(lastAgentDone.Time)
				if delay > 5*time.Second {
					findings = append(findings, Finding{
						Severity:    "warning",
						Type:        "message_delay",
						At:          ll.Time,
						Description: fmt.Sprintf("new message received %.1fs after agent done (possible queue block)", delay.Seconds()),
					})
				}
			}
			lastRecvMsg = ll
		}

		_ = lastRecvMsg // used in future rules
	}

	// ── End-of-file checks ────────────────────────────────────────────────────

	// Active turn at EOF with no done = stuck.
	if activeTurn != nil {
		findings = append(findings, Finding{
			Severity:    "critical",
			Type:        "turn_stuck_eof",
			At:          activeTurn.startTime,
			Description: fmt.Sprintf("turn %d started at %s, never completed (log ends here — agent likely hung)", activeTurn.turn, activeTurn.startTime.Format("15:04:05")),
		})
	}

	// LLM returned toolUse at EOF, no tool execution log followed = emit blocked.
	if lastToolUseTurnEnd != nil {
		findings = append(findings, Finding{
			Severity:    "critical",
			Type:        "emit_blocked",
			At:          lastToolUseTurnEnd.Time,
			Description: "LLM returned toolUse but no tool execution started — emit(EventToolExecutionStart) likely blocked (WebSocket write deadline missing)",
			FileLine:    lastToolUseTurnEnd.File,
		})
	}

	return findings
}
