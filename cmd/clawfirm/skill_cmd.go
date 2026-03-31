package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/ai-gateway/clawfirm/skillctl"
)

func runSkill(args []string) {
	if len(args) == 0 {
		printSkillUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "search":
		runSkillSearch(args[1:])
	case "install":
		runSkillInstall(args[1:])
	case "sync":
		runSkillSync(args[1:])
	case "list":
		runSkillList(args[1:])
	case "help", "-h", "--help":
		printSkillUsage()
	default:
		fmt.Fprintf(os.Stderr, "clawfirm skill: unknown subcommand %q\n\n", args[0])
		printSkillUsage()
		os.Exit(1)
	}
}

func printSkillUsage() {
	fmt.Fprintln(os.Stderr, `Usage: clawfirm skill <command> [flags]

Commands:
  search <query>               Search the remote skill registry
  install <name[@version]>     Install a skill from the registry
  sync                         Sync skills to client directories
  list                         List installed skills`)
}

func runSkillSearch(args []string) {
	if len(args) == 0 {
		log.Fatal("usage: clawfirm skill search <query>")
	}
	query := args[0]

	client := skillctl.NewClient()
	result, err := client.Search(context.Background(), query)
	if err != nil {
		log.Fatalf("search: %v", err)
	}

	if len(result.Skills) == 0 {
		fmt.Println("No skills found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION")
	for _, s := range result.Skills {
		fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, s.Version, s.Description)
	}
	w.Flush()
}

func runSkillInstall(args []string) {
	fs := flag.NewFlagSet("skill install", flag.ExitOnError)
	force := fs.Bool("force", false, "overwrite existing skill")
	sync := fs.Bool("sync", false, "sync after install")
	fs.Parse(args)

	if fs.NArg() == 0 {
		log.Fatal("usage: clawfirm skill install <name[@version]> [--force] [--sync]")
	}
	name := fs.Arg(0)

	client := skillctl.NewClient()
	result, err := client.Install(context.Background(), skillctl.InstallOptions{
		Name:  name,
		Force: *force,
		Sync:  *sync,
	})
	if err != nil {
		log.Fatalf("install: %v", err)
	}

	fmt.Printf("Installed %s@%s → %s\n", result.Name, result.Version, result.InstallDir)
	if result.Synced {
		fmt.Println("Skills synced to client directories.")
	}
}

func runSkillSync(args []string) {
	fs := flag.NewFlagSet("skill sync", flag.ExitOnError)
	client := fs.String("client", "", "sync only this client (e.g. clawfirm, claude-code)")
	dryRun := fs.Bool("dry-run", false, "show what would be done without making changes")
	fs.Parse(args)

	result, err := skillctl.Sync(skillctl.SyncOptions{
		Client: *client,
		DryRun: *dryRun,
	})
	if err != nil {
		log.Fatalf("sync: %v", err)
	}

	if *dryRun {
		fmt.Println("[dry-run]")
	}
	for _, a := range result.Created {
		fmt.Printf("  + %s/%s → %s\n", a.Client, a.Skill, a.Target)
	}
	for _, a := range result.Removed {
		fmt.Printf("  - %s/%s\n", a.Client, a.Skill)
	}
	if len(result.Created) == 0 && len(result.Removed) == 0 {
		fmt.Println("Nothing to do — all skills are in sync.")
	} else {
		fmt.Printf("Created: %d, Removed: %d, Skipped: %d\n",
			len(result.Created), len(result.Removed), len(result.Skipped))
	}
}

func runSkillList(args []string) {
	reg, err := skillctl.LoadRegistry(skillctl.DefaultRegistryPath())
	if err != nil {
		log.Fatalf("list: %v", err)
	}

	if len(reg.Skills) == 0 {
		fmt.Println("No skills installed.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tENABLED\tCLIENTS")
	for name, se := range reg.Skills {
		clients := "all"
		if len(se.Clients) > 0 {
			clients = ""
			for i, c := range se.Clients {
				if i > 0 {
					clients += ", "
				}
				clients += c
			}
		}
		enabled := "no"
		if se.Enabled {
			enabled = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, enabled, clients)
	}
	w.Flush()
}
