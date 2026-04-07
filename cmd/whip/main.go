// Command whip runs a WhipFlow (.whip) workflow file.
//
// Usage:
//
//	go run ./cmd/whip <file.whip>
//	go run ./cmd/whip -config ~/.clawfirm/config.yml <file.whip>
//	go run ./cmd/whip install-skills [--force]
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/whipflow"
)

// progressEvent is the NDJSON line emitted to stdout for each session event.
type progressEvent struct {
	Type       string `json:"type"`
	Index      int    `json:"index"`
	Name       string `json:"name,omitempty"`
	Provider   string `json:"provider,omitempty"`
	Done       bool   `json:"done"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	StreamText string `json:"stream_text,omitempty"`
}

func emitProgress(p whipflow.SessionProgress) {
	ev := progressEvent{
		Type:       "session_progress",
		Index:      p.Index,
		Name:       p.Name,
		Provider:   p.Provider,
		Done:       p.Done,
		DurationMs: p.DurationMs,
		Output:     p.Output,
		Error:      p.Error,
		StreamText: p.StreamText,
	}
	b, _ := json.Marshal(ev)
	fmt.Println(string(b))
}

// inputFlags collects multiple -input key=value flags.
type inputFlags []string

func (f *inputFlags) String() string { return strings.Join(*f, ", ") }
func (f *inputFlags) Set(v string) error {
	*f = append(*f, v)
	return nil
}

//go:embed SKILL.md
var skillMD []byte

func main() {
	cfgPath := flag.String("config", "", "path to config.yml (default: ~/.clawfirm/config.yml)")
	var inputs inputFlags
	flag.Var(&inputs, "input", "pre-fill ask variable: key=value (repeatable)")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: whip [flags] <file.whip>")
		fmt.Fprintln(os.Stderr, "       whip validate <file.whip> [...]")
		fmt.Fprintln(os.Stderr, "       whip install-skills [--force]")
		fmt.Fprintln(os.Stderr, "flags:")
		fmt.Fprintln(os.Stderr, "  -input key=value   pre-fill ask variable (repeatable)")
		fmt.Fprintln(os.Stderr, "  -config path       path to config.yml")
		os.Exit(1)
	}

	if args[0] == "install-skills" {
		force := len(args) > 1 && args[1] == "--force"
		if err := installSkills(force); err != nil {
			log.Fatalf("install-skills: %v", err)
		}
		return
	}

	if args[0] == "validate" {
		runValidate(args[1:])
		return
	}

	filePath := args[0]

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	opts := []whipflow.Option{
		whipflow.WithPiConfig(cfg),
		// Emit each session as an NDJSON event immediately when it completes;
		// context between sessions flows through the interpreter's variable
		// environment (markdown output stored as $last / named session vars).
		whipflow.WithSessionProgressCallback(emitProgress),
	}

	// Parse -input flags into initial inputs map.
	if len(inputs) > 0 {
		inputMap := make(map[string]string, len(inputs))
		for _, kv := range inputs {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				inputMap[parts[0]] = parts[1]
			}
		}
		opts = append(opts, whipflow.WithInitialInputs(inputMap))
	}

	if _, err := whipflow.RunFile(filePath, opts...); err != nil {
		log.Fatalf("whipflow: %v", err)
	}
}

func runValidate(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: whip validate <file.whip> [<file2.whip> ...]")
		os.Exit(1)
	}

	hasErrors := false
	for _, filePath := range args {
		source, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", filePath, err)
			hasErrors = true
			continue
		}

		program, parseErrors := whipflow.Parse(string(source))
		if len(parseErrors) > 0 {
			for _, e := range parseErrors {
				fmt.Fprintf(os.Stderr, "%s: %v\n", filePath, e)
			}
			hasErrors = true
			continue
		}

		vResult := whipflow.Validate(program)
		if !vResult.Valid {
			for _, e := range vResult.Errors {
				fmt.Fprintf(os.Stderr, "%s: %s\n", filePath, e.Message)
			}
			hasErrors = true
			continue
		}

		analysis := whipflow.AnalyzeComplexity(program)
		fmt.Printf("%s: OK (%s, %d session%s)\n",
			filePath, analysis.Tier, analysis.SessionCount,
			plural(analysis.SessionCount))
	}

	if hasErrors {
		os.Exit(1)
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func installSkills(force bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	destDir := filepath.Join(home, ".claude", "skills", "whipflow")
	destFile := filepath.Join(destDir, "SKILL.md")

	if !force {
		if _, err := os.Stat(destFile); err == nil {
			fmt.Printf("Already installed: %s\nUse --force to overwrite.\n", destFile)
			return nil
		}
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(destFile, skillMD, 0644); err != nil {
		return err
	}
	fmt.Printf("Installed: %s\n", destFile)
	return nil
}
