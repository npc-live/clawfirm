// Command clawfirm is the unified CLI for the Clawfirm AI Gateway.
//
// Usage:
//
//	clawfirm run <file.whip>                  # run a WhipFlow workflow
//	clawfirm run -config ./config.yml <file.whip>
//	clawfirm install-skills [--force]         # install WhipFlow skill for Claude Code
package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-gateway/clawfirm/app"
	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/whipflow"
)

//go:embed SKILL.md
var skillMD []byte

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runWhip(os.Args[2:])
	case "validate":
		runValidate(os.Args[2:])
	case "vault":
		runVault(os.Args[2:])
	case "daemon":
		runDaemon(os.Args[2:])
	case "skill":
		runSkill(os.Args[2:])
	case "install-skills":
		force := len(os.Args) > 2 && os.Args[2] == "--force"
		if err := installSkills(force); err != nil {
			log.Fatalf("install-skills: %v", err)
		}
	case "version":
		fmt.Println("clawfirm", app.Version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "clawfirm: unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: clawfirm <command> [flags] [args]

Commands:
  run <file.whip>          Run a WhipFlow workflow
  validate <file.whip>     Validate a WhipFlow workflow (parse + semantic check)
  vault <subcommand>       Encrypted secret vault (init, set, get, list, ...)
  daemon start|stop|status Manage the watchdog daemon
  skill search <query>     Search the remote skill registry
  skill install <name>     Install a skill from the registry
  skill sync               Sync skills to client directories
  skill list               List installed skills
  install-skills [--force] Install WhipFlow skill for Claude Code
  version                  Print version
  help                     Show this help`)
}

func runWhip(args []string) {
	var cfgPath string
	initialInputs := map[string]string{}

	// Simple flag parsing before positional arg.
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-config", "--config":
			if i+1 >= len(args) {
				log.Fatal("run: -config requires a path")
			}
			cfgPath = args[i+1]
			i += 2
			continue
		case "-var", "--var":
			if i+1 >= len(args) {
				log.Fatal("run: -var requires key=value")
			}
			kv := args[i+1]
			idx := strings.IndexByte(kv, '=')
			if idx < 0 {
				log.Fatalf("run: -var %q must be in key=value format", kv)
			}
			initialInputs[kv[:idx]] = kv[idx+1:]
			i += 2
			continue
		}
		break
	}
	rest := args[i:]

	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "usage: clawfirm run [flags] <file.whip>")
		os.Exit(1)
	}

	filePath := rest[0]

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	opts := []whipflow.Option{whipflow.WithPiConfig(cfg)}
	if len(initialInputs) > 0 {
		opts = append(opts, whipflow.WithInitialInputs(initialInputs))
	}

	result, err := whipflow.RunFile(filePath, opts...)
	if err != nil {
		log.Fatalf("whipflow: %v", err)
	}

	for _, output := range result.Outputs {
		fmt.Println(output)
	}
}

func runValidate(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: clawfirm validate <file.whip> [<file2.whip> ...]")
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

		// Also show complexity analysis.
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
