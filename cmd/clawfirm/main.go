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
	case "vault":
		runVault(os.Args[2:])
	case "skill":
		runSkill(os.Args[2:])
	case "install-skills":
		force := len(os.Args) > 2 && os.Args[2] == "--force"
		if err := installSkills(force); err != nil {
			log.Fatalf("install-skills: %v", err)
		}
	case "version":
		fmt.Println("clawfirm dev")
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
  vault <subcommand>       Encrypted secret vault (init, set, get, list, ...)
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

	// Simple flag parsing before positional arg.
	i := 0
	for i < len(args) {
		if args[i] == "-config" || args[i] == "--config" {
			if i+1 >= len(args) {
				log.Fatal("run: -config requires a path")
			}
			cfgPath = args[i+1]
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

	result, err := whipflow.RunFile(filePath, whipflow.WithPiConfig(cfg))
	if err != nil {
		log.Fatalf("whipflow: %v", err)
	}

	for _, output := range result.Outputs {
		fmt.Println(output)
	}
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
