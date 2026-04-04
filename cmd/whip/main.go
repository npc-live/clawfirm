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
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/whipflow"
)

//go:embed SKILL.md
var skillMD []byte

func main() {
	cfgPath := flag.String("config", "", "path to config.yml (default: ~/.clawfirm/config.yml)")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: whip [flags] <file.whip>")
		fmt.Fprintln(os.Stderr, "       whip validate <file.whip> [...]")
		fmt.Fprintln(os.Stderr, "       whip install-skills [--force]")
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

	result, err := whipflow.RunFile(filePath, whipflow.WithPiConfig(cfg))
	if err != nil {
		log.Fatalf("whipflow: %v", err)
	}

	for _, output := range result.Outputs {
		fmt.Println(output)
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
