// Command pi is a stateless CLI for running single-turn AI prompts
// using agents defined in ~/.pi-go/config.yml.
//
// Usage:
//
//	pi -p "your prompt here"
//	echo "your prompt" | pi
//	pi --agent myagent -p "prompt"
//	pi --output-format stream-json -p "prompt"   # for WhipFlow
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/config"
	"github.com/ai-gateway/pi-go/internal/agentbuilder"
	"github.com/ai-gateway/pi-go/skill"
	"github.com/ai-gateway/pi-go/types"
)

func main() {
	cfgPath := flag.String("config", "", "path to config.yml (default: ~/.pi-go/config.yml)")
	agentName := flag.String("agent", "", "agent name to use (default: default_agent from config)")
	prompt := flag.String("p", "", "prompt text (reads from stdin if empty)")
	outputFmt := flag.String("output-format", "text", "output format: text or stream-json")
	timeout := flag.Int("timeout", 300, "timeout in seconds")
	// --no-session is always true for this CLI; flag kept for compatibility
	_ = flag.Bool("no-session", true, "stateless mode (always true)")
	flag.Parse()

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fatal(*outputFmt, fmt.Sprintf("config: %v", err))
	}

	// ── Agent selection ───────────────────────────────────────────────────────
	name := *agentName
	if name == "" {
		name = cfg.DefaultAgent
	}
	if name == "" && len(cfg.Agents) > 0 {
		name = cfg.Agents[0].Name
	}
	if name == "" {
		fatal(*outputFmt, "no agents defined in config.yml")
	}

	ac, ok := cfg.Agent(name)
	if !ok {
		fatal(*outputFmt, fmt.Sprintf("agent %q not found in config", name))
	}

	// ── Prompt ────────────────────────────────────────────────────────────────
	promptText := *prompt
	if promptText == "" {
		var sb strings.Builder
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			sb.WriteString(scanner.Text())
			sb.WriteString("\n")
		}
		promptText = strings.TrimRight(sb.String(), "\n")
	}
	if promptText == "" {
		fatal(*outputFmt, "no prompt provided (use -p or pipe via stdin)")
	}

	// ── Provider ──────────────────────────────────────────────────────────────
	pc, ok := cfg.Providers[ac.Provider]
	if !ok {
		fatal(*outputFmt, fmt.Sprintf("provider %q not found for agent %q", ac.Provider, name))
	}
	prov, err := agentbuilder.BuildProvider(ac.Provider, pc)
	if err != nil {
		fatal(*outputFmt, fmt.Sprintf("provider: %v", err))
	}

	// ── Tools ─────────────────────────────────────────────────────────────────
	// memMgr is nil — memory_search/memory_get require a DB; not opened here.
	tools := agentbuilder.BuildTools(ac.Tools, nil, cfg, nil)

	// ── Skills ────────────────────────────────────────────────────────────────
	skillResult := skill.Load(skill.LoadOptions{SkillPaths: ac.SkillPaths})
	compacted := skill.CompactSkillPaths(skillResult.Skills)
	skillsPrompt, _, _ := skill.ApplySkillsPromptLimits(compacted)

	// ── System prompt ─────────────────────────────────────────────────────────
	systemPrompt := agent.BuildSystemPrompt(agent.SystemPromptParams{
		SkillsPrompt: skillsPrompt,
		ExtraPrompt:  ac.SystemPrompt,
		PromptMode:   agent.PromptModeFull,
	})

	// ── Model ─────────────────────────────────────────────────────────────────
	maxTokens := ac.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	model := types.Model{ID: ac.Model, Provider: ac.Provider, MaxTokens: maxTokens}

	// ── Agent ─────────────────────────────────────────────────────────────────
	agentOpts := []agent.AgentOption{
		agent.WithModel(model),
		agent.WithSystemPrompt(systemPrompt),
	}
	if len(tools) > 0 {
		agentOpts = append(agentOpts, agent.WithTools(tools))
	}
	a := agent.NewAgent(prov, agentOpts...)

	// ── Execute ───────────────────────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	if err := a.Prompt(ctx, promptText); err != nil {
		fatal(*outputFmt, fmt.Sprintf("prompt: %v", err))
	}
	if err := a.WaitForIdle(ctx); err != nil {
		fatal(*outputFmt, fmt.Sprintf("wait: %v", err))
	}

	// ── Output ────────────────────────────────────────────────────────────────
	result := extractOutput(a.State().Messages)
	emit(*outputFmt, result)
}

// extractOutput returns the last assistant text from the message history.
func extractOutput(messages []types.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		am, ok := messages[i].(*types.AssistantMessage)
		if !ok {
			continue
		}
		var parts []string
		for _, block := range am.Content {
			if tc, ok := block.(*types.TextContent); ok && tc.Text != "" {
				parts = append(parts, tc.Text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return ""
}

// emit writes the result in the requested format to stdout.
func emit(format, result string) {
	if format == "stream-json" {
		data, _ := json.Marshal(map[string]any{
			"type":     "result",
			"result":   result,
			"is_error": false,
		})
		fmt.Println(string(data))
		return
	}
	fmt.Print(result)
}

// fatal exits with an error in the requested format.
func fatal(format, msg string) {
	if format == "stream-json" {
		data, _ := json.Marshal(map[string]any{
			"type":     "result",
			"result":   msg,
			"is_error": true,
		})
		fmt.Println(string(data))
		os.Exit(1)
	}
	log.Fatal(msg)
}
