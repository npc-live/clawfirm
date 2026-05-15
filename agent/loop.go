package agent

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
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

	// RetryConfig controls retry behavior for transient LLM errors (429, 529, etc.).
	// Zero value uses defaults (10 retries, 500ms base delay).
	RetryConfig RetryConfig

	// Compressor is used for reactive compaction when the LLM returns a
	// "prompt too long" error (NeedCompact). If nil, such errors are fatal.
	Compressor *ContextCompressor

	// TokenBudget caps total tokens (input + output) across all turns.
	// When exceeded the loop exits gracefully. 0 = unlimited.
	TokenBudget int
}

// RetryConfig controls exponential backoff for LLM stream retries.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts. Default: 10.
	MaxRetries int
	// BaseDelay is the initial delay before the first retry. Default: 500ms.
	BaseDelay time.Duration
	// MaxDelay caps the backoff delay. Default: 60s.
	MaxDelay time.Duration
}

func (r RetryConfig) maxRetries() int {
	if r.MaxRetries <= 0 {
		return 10
	}
	return r.MaxRetries
}

func (r RetryConfig) baseDelay() time.Duration {
	if r.BaseDelay <= 0 {
		return 500 * time.Millisecond
	}
	return r.BaseDelay
}

func (r RetryConfig) maxDelay() time.Duration {
	if r.MaxDelay <= 0 {
		return 60 * time.Second
	}
	return r.MaxDelay
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

	// Track which deferred tools have been activated via tool_search.
	activatedTools := make(map[string]bool)

	// Cumulative token usage across all turns.
	cumUsage := &types.CumulativeUsage{}

	turn := 0
	streamRetries := 0
	lastRetriedTurn := 0
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

		// Build tool schemas — only include non-deferred + activated tools.
		toolSchemas := make([]provider.ToolSchema, 0, len(agentCtx.Tools))
		for _, t := range agentCtx.Tools {
			if t.ShouldDefer() && !activatedTools[t.Name()] {
				continue
			}
			toolSchemas = append(toolSchemas, provider.ToolSchema{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Schema(),
			})
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

		// Microcompact: clear old tool result content to reduce prompt size.
		msgs = MicrocompactMessages(msgs, 3)

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

		// Call provider with retry for transient errors
		log.Printf("[agent-loop] turn %d: calling LLM (model=%s, tools=%d, msgs=%d)", turn, config.Model.ID, len(toolSchemas), len(msgs))
		var eventCh <-chan types.AssistantMessageEvent
		streamErr := retryStream(ctx, config.RetryConfig, emit, turn, func() error {
			var err error
			eventCh, err = prov.Stream(ctx, req)
			return err
		})
		if streamErr != nil {
			decision := provider.ClassifyError(streamErr)
			if decision == provider.RetryDecisionNeedCompact && config.Compressor != nil {
				log.Printf("[agent-loop] turn %d: prompt too long, attempting force compaction", turn)
				prevLen := len(agentCtx.Messages)
				compressed, compErr := config.Compressor.ForceCompress(ctx, agentCtx.Messages)
				if compErr != nil {
					log.Printf("[agent-loop] turn %d: force compaction failed: %v", turn, compErr)
					return agentCtx.Messages, fmt.Errorf("agent loop: stream: %w (compaction also failed: %v)", streamErr, compErr)
				}
				agentCtx.Messages = compressed
				log.Printf("[agent-loop] turn %d: compacted %d → %d msgs, retrying turn", turn, prevLen, len(compressed))
				turn-- // retry the same turn number
				continue
			}
			log.Printf("[agent-loop] turn %d: stream error (after retries): %v", turn, streamErr)
			return agentCtx.Messages, fmt.Errorf("agent loop: stream: %w", streamErr)
		}

		// Consume stream events with eager tool execution.
		// Safe tools start as soon as their ToolCallEnd event arrives (while the
		// stream is still in progress). Unsafe tools are queued and executed
		// sequentially after the stream finishes.
		assistantMsg := &types.AssistantMessage{
			Role:     "assistant",
			Provider: prov.ID(),
			Model:    config.Model.ID,
		}
		emit(types.AgentEvent{
			Type:         types.EventMessageStart,
			AssistantMsg: assistantMsg,
		})

		// Streaming tool execution state.
		type indexedResult struct {
			index  int
			result types.ToolResultMessage
			err    error
		}
		var (
			toolCallIndex int
			safeWg        sync.WaitGroup
			safeSem       = make(chan struct{}, smartConcurrencyLimit)
			resultCh      = make(chan indexedResult, 16)
			unsafeQueue   []struct {
				index int
				call  types.ToolCall
			}
		)

		// Collector goroutine — gathers results from eager + deferred executions.
		var allResults []indexedResult
		collectorDone := make(chan struct{})
		go func() {
			for ir := range resultCh {
				allResults = append(allResults, ir)
			}
			close(collectorDone)
		}()

		for streamEv := range eventCh {
			if ctx.Err() != nil {
				// Drain the channel so the provider goroutine can exit.
				go func() {
					for range eventCh {
					}
				}()
				break
			}
			// Forward stream event
			emit(types.AgentEvent{
				Type:         types.EventMessageUpdate,
				AssistantMsg: assistantMsg,
				StreamEvent:  &streamEv,
			})

			switch streamEv.Type {
			case types.StreamEventDone:
				if streamEv.Message != nil {
					assistantMsg = streamEv.Message
				}
			case types.StreamEventError:
				if streamEv.Error != nil {
					assistantMsg = streamEv.Error
				}
			case types.StreamEventToolCallEnd:
				// A tool call completed in the stream — start executing eagerly.
				if streamEv.ToolCall != nil {
					tc := *streamEv.ToolCall
					idx := toolCallIndex
					toolCallIndex++

					if isToolSafe(tc.Name, reg) {
						// Safe tool: execute immediately in background.
						safeWg.Add(1)
						go func() {
							defer safeWg.Done()
							safeSem <- struct{}{}
							defer func() { <-safeSem }()
							r, err := executeOne(ctx, tc, reg, assistantMsg,
								config.BeforeToolCall, config.AfterToolCall, emit)
							resultCh <- indexedResult{idx, r, err}
						}()
					} else {
						// Unsafe tool: queue for sequential execution after stream ends.
						unsafeQueue = append(unsafeQueue, struct {
							index int
							call  types.ToolCall
						}{idx, tc})
					}
				}
			}
		}

		// Stream finished. Wait for all in-flight safe tools.
		safeWg.Wait()

		// Check for cancellation before executing unsafe tools.
		if ctx.Err() != nil {
			close(resultCh)
			<-collectorDone
			log.Printf("[agent-loop] ctx cancelled after stream, stopping")
			assistantMsg.StopReason = types.StopReasonAborted
			if assistantMsg.Timestamp == 0 {
				assistantMsg.Timestamp = time.Now().UnixMilli()
			}
			emit(types.AgentEvent{Type: types.EventMessageEnd, AssistantMsg: assistantMsg})
			stripped := stripToolCalls(assistantMsg)
			if len(stripped.Content) > 0 {
				agentCtx.Messages = append(agentCtx.Messages, stripped)
			}
			emit(types.AgentEvent{Type: types.EventAgentEnd, Messages: agentCtx.Messages})
			return agentCtx.Messages, ctx.Err()
		}

		// Execute queued unsafe tools sequentially.
		for _, uq := range unsafeQueue {
			if ctx.Err() != nil {
				break
			}
			r, err := executeOne(ctx, uq.call, reg, assistantMsg,
				config.BeforeToolCall, config.AfterToolCall, emit)
			resultCh <- indexedResult{uq.index, r, err}
		}
		close(resultCh)
		<-collectorDone

		// Check again after tool execution.
		if ctx.Err() != nil {
			log.Printf("[agent-loop] ctx cancelled after tool execution, stopping")
			assistantMsg.StopReason = types.StopReasonAborted
			if assistantMsg.Timestamp == 0 {
				assistantMsg.Timestamp = time.Now().UnixMilli()
			}
			emit(types.AgentEvent{Type: types.EventMessageEnd, AssistantMsg: assistantMsg})
			stripped := stripToolCalls(assistantMsg)
			if len(stripped.Content) > 0 {
				agentCtx.Messages = append(agentCtx.Messages, stripped)
			}
			emit(types.AgentEvent{Type: types.EventAgentEnd, Messages: agentCtx.Messages})
			return agentCtx.Messages, ctx.Err()
		}

		// Ensure timestamp is set
		if assistantMsg.Timestamp == 0 {
			assistantMsg.Timestamp = time.Now().UnixMilli()
		}

		// Log what the LLM returned
		contentTypes := make([]string, 0, len(assistantMsg.Content))
		for _, b := range assistantMsg.Content {
			switch cb := b.(type) {
			case *types.TextContent:
				contentTypes = append(contentTypes, "text")
			case *types.ToolCall:
				contentTypes = append(contentTypes, "tool_call:"+cb.Name)
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

		// Assemble tool results in original order.
		var toolResults []types.ToolResultMessage
		if toolCallIndex > 0 {
			toolResults = make([]types.ToolResultMessage, toolCallIndex)
			for _, ir := range allResults {
				if ir.err != nil {
					return agentCtx.Messages, fmt.Errorf("agent loop: tool execution: %w", ir.err)
				}
				toolResults[ir.index] = ir.result
			}

			// Collect tool_search activations before appending to history.
			for _, tr := range toolResults {
				if tr.ToolName == "tool_search" {
					if names, ok := tr.Details.([]string); ok {
						for _, n := range names {
							activatedTools[n] = true
						}
					}
				}
			}

			// Append tool results to history
			for i := range toolResults {
				agentCtx.Messages = append(agentCtx.Messages, &toolResults[i])
			}
		}

		// Update cumulative usage.
		cumUsage.TotalInput += assistantMsg.Usage.Input
		cumUsage.TotalOutput += assistantMsg.Usage.Output
		cumUsage.TurnCount++

		cumSnapshot := *cumUsage // snapshot so each event is independent
		emit(types.AgentEvent{
			Type:            types.EventTurnEnd,
			Message:         assistantMsg,
			ToolResults:     toolResults,
			CumulativeUsage: &cumSnapshot,
		})

		// Check token budget.
		if config.TokenBudget > 0 {
			total := cumUsage.TotalInput + cumUsage.TotalOutput
			if total >= config.TokenBudget {
				log.Printf("[agent-loop] turn %d: token budget exceeded (%d >= %d), stopping", turn, total, config.TokenBudget)
				break
			}
		}

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

		// Auto-retry when the stream was interrupted (SSE closed without message_stop).
		// Covers both explicit "stream interrupted" errors and empty responses (0 content, 0 usage)
		// which indicate the SSE connection died before any data arrived.
		isStreamInterrupted := (assistantMsg.StopReason == types.StopReasonError &&
			strings.Contains(assistantMsg.ErrorMessage, "stream interrupted")) ||
			(assistantMsg.StopReason == types.StopReasonError &&
				len(assistantMsg.Content) == 0 && assistantMsg.Usage.Input == 0 && assistantMsg.Usage.Output == 0)
		if isStreamInterrupted {
			if lastRetriedTurn != turn {
				streamRetries = 0
				lastRetriedTurn = turn
			}
			streamRetries++
			if streamRetries <= 2 {
				// Remove the empty failed assistant message we just appended so history stays clean.
				agentCtx.Messages = agentCtx.Messages[:len(agentCtx.Messages)-1]
				log.Printf("[agent-loop] turn %d: stream interrupted, retrying (%d/2)", turn, streamRetries)
				turn--
				continue
			}
			log.Printf("[agent-loop] turn %d: stream interrupted, max retries reached", turn)
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

	log.Printf("[agent-loop] === agent done after %d turns, total %d msgs, tokens=in:%d/out:%d ===",
		turn, len(agentCtx.Messages), cumUsage.TotalInput, cumUsage.TotalOutput)

	emit(types.AgentEvent{
		Type:            types.EventAgentEnd,
		Messages:        agentCtx.Messages,
		CumulativeUsage: cumUsage,
	})

	return agentCtx.Messages, nil
}

// retryStream wraps a prov.Stream() call with exponential backoff for transient errors.
// It only retries errors classified as RetryDecisionRetry; all other errors are returned immediately.
func retryStream(
	ctx context.Context,
	cfg RetryConfig,
	emit func(types.AgentEvent),
	turn int,
	fn func() error,
) error {
	maxRetries := cfg.maxRetries()
	baseDelay := cfg.baseDelay()
	maxDelay := cfg.maxDelay()

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		decision := provider.ClassifyError(lastErr)
		if decision != provider.RetryDecisionRetry {
			// Non-retryable (permanent error or needs compaction) — return immediately.
			return lastErr
		}

		if attempt >= maxRetries {
			break
		}

		// Compute delay: exponential backoff with jitter.
		delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
		if delay > maxDelay {
			delay = maxDelay
		}
		// Add jitter: multiply by random factor in [0.5, 1.5).
		jitter := 0.5 + rand.Float64()
		delay = time.Duration(float64(delay) * jitter)

		// Respect Retry-After header if available.
		if apiErr, ok := provider.IsAPIError(lastErr); ok && apiErr.RetryAfter > 0 {
			if apiErr.RetryAfter > delay {
				delay = apiErr.RetryAfter
			}
		}

		log.Printf("[agent-loop] turn %d: retryable error (attempt %d/%d), retrying in %v: %v",
			turn, attempt+1, maxRetries, delay, lastErr)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastErr
}

// stripToolCalls returns a copy of msg with tool_use blocks removed.
// Used when aborting to keep history in a valid state for future LLM calls.
func stripToolCalls(msg *types.AssistantMessage) *types.AssistantMessage {
	filtered := make([]types.ContentBlock, 0, len(msg.Content))
	for _, b := range msg.Content {
		if _, ok := b.(*types.ToolCall); !ok {
			filtered = append(filtered, b)
		}
	}
	if len(filtered) == len(msg.Content) {
		return msg
	}
	cp := *msg
	cp.Content = filtered
	return &cp
}
