package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// AgentLoopConfig carries all configuration for a single agent loop invocation.
type AgentLoopConfig struct {
	// Model to use for LLM calls.
	Model types.Model
	// ToolExecution controls whether tools run sequentially or in parallel.
	ToolExecution types.ToolExecutionMode
	// MaxTurns limits the number of LLM round-trips per prompt (0 = unlimited).
	MaxTurns int

	// ConvertToLLM optionally transforms messages before sending to the LLM.
	ConvertToLLM func([]types.Message) ([]types.Message, error)
	// TransformContext optionally prunes/injects messages before each LLM call.
	TransformContext func(context.Context, []types.Message) ([]types.Message, error)
	// GetAPIKey dynamically provides an API key for the given provider.
	GetAPIKey func(provider string) (string, error)
	// BeforeToolCall is called before each tool execution; returning Block=true skips the tool.
	BeforeToolCall func(ctx BeforeToolCallCtx) (BeforeToolCallResult, error)
	// AfterToolCall is called after each tool execution; can replace the result.
	AfterToolCall func(ctx AfterToolCallCtx) (AfterToolCallResult, error)
	// GetSteeringMessages returns messages to inject after the current turn ends.
	GetSteeringMessages func() ([]types.Message, error)
	// GetFollowUpMessages returns messages to inject after the agent stops (to trigger a new turn).
	GetFollowUpMessages func() ([]types.Message, error)
	// Options are extra streaming/model options.
	Options types.StreamOptions
}

// AgentLoopContinue is identical to AgentLoop but does not prepend any new
// prompts — it resumes the loop from the messages already in agentCtx.
// Use this for retry scenarios or when the caller has already injected
// messages directly into AgentContext.Messages before calling.
func AgentLoopContinue(
	ctx context.Context,
	prov provider.LLMProvider,
	agentCtx AgentContext,
	config AgentLoopConfig,
	emit func(types.AgentEvent),
) ([]types.Message, error) {
	return AgentLoop(ctx, prov, nil, agentCtx, config, emit)
}

// AgentLoop runs the LLM+tool loop until the model stops without more follow-up.
// It emits AgentEvent values via emit and returns the full message history.
func AgentLoop(
	ctx context.Context,
	prov provider.LLMProvider,
	prompts []types.Message,
	agentCtx AgentContext,
	config AgentLoopConfig,
	emit func(types.AgentEvent),
) ([]types.Message, error) {
	// Append initial prompts
	agentCtx.Messages = append(agentCtx.Messages, prompts...)

	emit(types.AgentEvent{Type: types.EventAgentStart})

	// Build tool registry from context tools
	reg := tool.NewRegistry()
	for _, t := range agentCtx.Tools {
		reg.Register(t)
	}

	// Build tool schemas for the LLM
	toolSchemas := make([]provider.ToolSchema, 0, len(agentCtx.Tools))
	for _, t := range agentCtx.Tools {
		toolSchemas = append(toolSchemas, provider.ToolSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Schema(),
		})
	}

	turn := 0
	for {
		turn++
		log.Printf("[agent-loop] === turn %d start, history=%d msgs ===", turn, len(agentCtx.Messages))

		if err := ctx.Err(); err != nil {
			log.Printf("[agent-loop] ctx cancelled: %v", err)
			return agentCtx.Messages, err
		}

		// Enforce max turns if configured
		if config.MaxTurns > 0 && turn > config.MaxTurns {
			log.Printf("[agent-loop] max turns (%d) reached, stopping", config.MaxTurns)
			break
		}

		// Optionally transform context (e.g., prune)
		msgs := agentCtx.Messages
		if config.TransformContext != nil {
			transformed, err := config.TransformContext(ctx, msgs)
			if err != nil {
				return agentCtx.Messages, fmt.Errorf("agent loop: transform context: %w", err)
			}
			msgs = transformed
		}

		// Optionally convert messages for LLM format
		if config.ConvertToLLM != nil {
			converted, err := config.ConvertToLLM(msgs)
			if err != nil {
				return agentCtx.Messages, fmt.Errorf("agent loop: convert to LLM: %w", err)
			}
			msgs = converted
		}

		// Resolve API key
		options := config.Options
		if config.GetAPIKey != nil {
			key, err := config.GetAPIKey(prov.ID())
			if err == nil {
				options.APIKey = key
			}
		}

		// Build LLM request
		req := provider.LLMRequest{
			Model:        config.Model,
			SystemPrompt: agentCtx.SystemPrompt,
			Messages:     msgs,
			Tools:        toolSchemas,
			Options:      options,
		}

		emit(types.AgentEvent{Type: types.EventTurnStart})

		// Call provider
		log.Printf("[agent-loop] turn %d: calling LLM (model=%s, tools=%d, msgs=%d)", turn, config.Model.ID, len(toolSchemas), len(msgs))
		eventCh, err := prov.Stream(ctx, req)
		if err != nil {
			log.Printf("[agent-loop] turn %d: stream error: %v", turn, err)
			return agentCtx.Messages, fmt.Errorf("agent loop: stream: %w", err)
		}

		// Consume stream events
		assistantMsg := &types.AssistantMessage{
			Role:     "assistant",
			Provider: prov.ID(),
			Model:    config.Model.ID,
		}
		emit(types.AgentEvent{
			Type:         types.EventMessageStart,
			AssistantMsg: assistantMsg,
		})

		for streamEv := range eventCh {
			if ctx.Err() != nil {
				break
			}
			// Forward stream event
			emit(types.AgentEvent{
				Type:         types.EventMessageUpdate,
				AssistantMsg: assistantMsg,
				StreamEvent:  &streamEv,
			})

			// Merge delta into partial message
			switch streamEv.Type {
			case types.StreamEventDone:
				if streamEv.Message != nil {
					assistantMsg = streamEv.Message
				}
			case types.StreamEventError:
				if streamEv.Error != nil {
					assistantMsg = streamEv.Error
				}
			}
		}

		// Ensure timestamp is set
		if assistantMsg.Timestamp == 0 {
			assistantMsg.Timestamp = time.Now().UnixMilli()
		}

		// Log what the LLM returned
		contentTypes := make([]string, 0, len(assistantMsg.Content))
		for _, b := range assistantMsg.Content {
			switch b.(type) {
			case *types.TextContent:
				contentTypes = append(contentTypes, "text")
			case *types.ToolCall:
				tc := b.(*types.ToolCall)
				contentTypes = append(contentTypes, "tool_call:"+tc.Name)
			default:
				contentTypes = append(contentTypes, fmt.Sprintf("%T", b))
			}
		}
		log.Printf("[agent-loop] turn %d: LLM done — stopReason=%q, content=%v, usage=in:%d/out:%d",
			turn, assistantMsg.StopReason, contentTypes, assistantMsg.Usage.Input, assistantMsg.Usage.Output)

		emit(types.AgentEvent{
			Type:         types.EventMessageEnd,
			AssistantMsg: assistantMsg,
		})

		// Append assistant message to history
		agentCtx.Messages = append(agentCtx.Messages, assistantMsg)

		// Handle tool calls if stop reason is toolUse
		var toolResults []types.ToolResultMessage
		if assistantMsg.StopReason == types.StopReasonToolUse {
			log.Printf("[agent-loop] turn %d: executing tool calls...", turn)
			// Collect tool calls from assistant message content
			var toolCalls []types.ToolCall
			for _, block := range assistantMsg.Content {
				if tc, ok := block.(*types.ToolCall); ok {
					toolCalls = append(toolCalls, *tc)
				}
			}

			var execErr error
			if config.ToolExecution == types.ToolExecutionParallel {
				toolResults, execErr = ExecuteParallel(ctx, toolCalls, reg, assistantMsg,
					config.BeforeToolCall, config.AfterToolCall, emit)
			} else {
				toolResults, execErr = ExecuteSequential(ctx, toolCalls, reg, assistantMsg,
					config.BeforeToolCall, config.AfterToolCall, emit)
			}
			if execErr != nil {
				return agentCtx.Messages, fmt.Errorf("agent loop: tool execution: %w", execErr)
			}

			// Append tool results to history
			for i := range toolResults {
				agentCtx.Messages = append(agentCtx.Messages, &toolResults[i])
			}
		}

		emit(types.AgentEvent{
			Type:        types.EventTurnEnd,
			Message:     assistantMsg,
			ToolResults: toolResults,
		})

		// Inject steering messages if available
		if config.GetSteeringMessages != nil {
			steerMsgs, err := config.GetSteeringMessages()
			if err == nil && len(steerMsgs) > 0 {
				agentCtx.Messages = append(agentCtx.Messages, steerMsgs...)
				// Continue the loop with steering messages
				continue
			}
		}

		// Auto-continue when output was truncated by token limit.
		if assistantMsg.StopReason == types.StopReasonLength {
			log.Printf("[agent-loop] turn %d: length-truncated, auto-continuing", turn)
			agentCtx.Messages = append(agentCtx.Messages, &types.UserMessage{
				Role: "user",
				Content: []types.ContentBlock{
					&types.TextContent{
						Type: types.ContentTypeText,
						Text: "[System: your previous response was truncated due to length. Please continue from where you left off.]",
					},
				},
			})
			continue
		}

		// If stop reason is not toolUse, check for follow-up
		if assistantMsg.StopReason != types.StopReasonToolUse {
			log.Printf("[agent-loop] turn %d: not toolUse (reason=%q), checking follow-ups", turn, assistantMsg.StopReason)
			if config.GetFollowUpMessages != nil {
				followUpMsgs, err := config.GetFollowUpMessages()
				if err == nil && len(followUpMsgs) > 0 {
					agentCtx.Messages = append(agentCtx.Messages, followUpMsgs...)
					continue
				}
			}
			// No more follow-ups; agent is done
			log.Printf("[agent-loop] turn %d: no follow-ups, breaking loop", turn)
			break
		}
		log.Printf("[agent-loop] turn %d: toolUse — continuing loop", turn)
	}

	log.Printf("[agent-loop] === agent done after %d turns, total %d msgs ===", turn, len(agentCtx.Messages))

	emit(types.AgentEvent{
		Type:     types.EventAgentEnd,
		Messages: agentCtx.Messages,
	})

	return agentCtx.Messages, nil
}
