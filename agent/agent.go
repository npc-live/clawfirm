package agent

import (
	"context"
	"sync"

	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// AgentOption configures an Agent at construction time.
type AgentOption func(*Agent)

// WithSystemPrompt sets the initial system prompt.
func WithSystemPrompt(s string) AgentOption {
	return func(a *Agent) { a.state.SystemPrompt = s }
}

// WithModel sets the initial model.
func WithModel(m types.Model) AgentOption {
	return func(a *Agent) { a.state.Model = m }
}

// WithThinkingLevel sets the initial thinking level.
func WithThinkingLevel(l types.ThinkingLevel) AgentOption {
	return func(a *Agent) { a.state.ThinkingLevel = l }
}

// WithTools sets the initial tool list.
func WithTools(tools []tool.AgentTool) AgentOption {
	return func(a *Agent) { a.state.Tools = tools }
}

// WithLoopConfig applies a pre-built AgentLoopConfig (merged on top of defaults).
func WithLoopConfig(cfg AgentLoopConfig) AgentOption {
	return func(a *Agent) { a.loopConfig = cfg }
}

// WithSteeringMode sets the initial steering delivery mode.
func WithSteeringMode(m types.SteeringMode) AgentOption {
	return func(a *Agent) {
		a.steeringMode = m
		a.steeringQ.SetMode(string(m))
	}
}

// WithFollowUpMode sets the initial follow-up delivery mode.
func WithFollowUpMode(m types.FollowUpMode) AgentOption {
	return func(a *Agent) {
		a.followUpMode = m
		a.followUpQ.SetMode(string(m))
	}
}

// Agent manages the lifecycle of an LLM agent, including conversation history,
// event subscription, steering, and follow-up injection.
type Agent struct {
	mu           sync.RWMutex
	state        AgentState
	provider     provider.LLMProvider
	loopConfig   AgentLoopConfig
	listeners    []listenerEntry
	steeringQ    *messageQueue
	followUpQ    *messageQueue
	steeringMode types.SteeringMode
	followUpMode types.FollowUpMode
	abortFn      context.CancelFunc
	idleCh       chan struct{}
}

type listenerEntry struct {
	id uint64
	fn func(types.AgentEvent)
}

var listenerSeq uint64

// NewAgent constructs an Agent backed by the given provider.
func NewAgent(p provider.LLMProvider, opts ...AgentOption) *Agent {
	a := &Agent{
		provider:     p,
		steeringMode: types.SteeringModeAll,
		followUpMode: types.FollowUpModeAll,
		idleCh:       make(chan struct{}),
	}
	a.steeringQ = newMessageQueue(string(a.steeringMode))
	a.followUpQ = newMessageQueue(string(a.followUpMode))
	// Start with idleCh already closed (idle)
	close(a.idleCh)

	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Prompt sends a text input to the agent and starts the loop.
func (a *Agent) Prompt(ctx context.Context, input string) error {
	return a.PromptMessages(ctx, []types.Message{
		&types.UserMessage{
			Role:    "user",
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: input}},
		},
	})
}

// PromptMessages starts the agent loop with the given messages as prompts.
func (a *Agent) PromptMessages(ctx context.Context, msgs []types.Message) error {
	a.mu.Lock()
	if a.state.IsRunning {
		a.mu.Unlock()
		return nil // or return an error; current choice is no-op
	}
	a.state.IsRunning = true
	a.state.Error = ""
	// Reset idleCh
	a.idleCh = make(chan struct{})
	loopCtx, cancel := context.WithCancel(ctx)
	a.abortFn = cancel
	// Snapshot state while holding the lock (buildLoopConfig would deadlock here)
	baseCfg := a.loopConfig
	model := a.state.Model
	thinkingLevel := a.state.ThinkingLevel
	agentCtx := AgentContext{
		SystemPrompt: a.state.SystemPrompt,
		Messages:     append([]types.Message{}, a.state.Messages...),
		Tools:        append([]tool.AgentTool{}, a.state.Tools...),
	}
	idleCh := a.idleCh
	steeringQ := a.steeringQ
	followUpQ := a.followUpQ
	a.mu.Unlock()

	// Build config outside the lock
	cfg := baseCfg
	cfg.Model = model
	cfg.Options.ThinkingLevel = thinkingLevel
	cfg.GetSteeringMessages = func() ([]types.Message, error) {
		return steeringQ.Drain(), nil
	}
	cfg.GetFollowUpMessages = func() ([]types.Message, error) {
		return followUpQ.Drain(), nil
	}

	go func() {
		defer func() {
			cancel()
			a.mu.Lock()
			a.state.IsRunning = false
			a.abortFn = nil
			closedIdleCh := idleCh
			a.mu.Unlock()
			close(closedIdleCh)
		}()

		history, err := AgentLoop(loopCtx, a.provider, msgs, agentCtx, cfg, a.emit)
		a.mu.Lock()
		a.state.Messages = history
		if err != nil && err != context.Canceled {
			a.state.Error = err.Error()
		}
		a.mu.Unlock()
	}()

	return nil
}

// Continue resumes the agent loop without new user prompts.
func (a *Agent) Continue(ctx context.Context) error {
	return a.PromptMessages(ctx, nil)
}

// Abort cancels any in-progress agent loop.
func (a *Agent) Abort() {
	a.mu.RLock()
	fn := a.abortFn
	a.mu.RUnlock()
	if fn != nil {
		fn()
	}
}

// WaitForIdle blocks until the agent is not running or ctx is cancelled.
func (a *Agent) WaitForIdle(ctx context.Context) error {
	a.mu.RLock()
	ch := a.idleCh
	a.mu.RUnlock()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Steer injects a message to be delivered after the current turn ends.
func (a *Agent) Steer(msg types.Message) {
	a.steeringQ.Enqueue(msg)
}

// FollowUp injects a message to trigger a new loop after the agent stops.
func (a *Agent) FollowUp(msg types.Message) {
	a.followUpQ.Enqueue(msg)
}

// Subscribe registers an event listener. Returns an unsubscribe function.
func (a *Agent) Subscribe(fn func(types.AgentEvent)) func() {
	a.mu.Lock()
	defer a.mu.Unlock()
	listenerSeq++
	id := listenerSeq
	a.listeners = append(a.listeners, listenerEntry{id: id, fn: fn})
	return func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		for i, l := range a.listeners {
			if l.id == id {
				a.listeners = append(a.listeners[:i], a.listeners[i+1:]...)
				return
			}
		}
	}
}

// State returns a snapshot of the agent's current state.
func (a *Agent) State() AgentState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	snap := a.state
	snap.Messages = append([]types.Message{}, a.state.Messages...)
	snap.Tools = append([]tool.AgentTool{}, a.state.Tools...)
	return snap
}

// SetSystemPrompt updates the system prompt (takes effect on the next loop run).
func (a *Agent) SetSystemPrompt(s string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.SystemPrompt = s
}

// SetModel updates the model (takes effect on the next loop run).
func (a *Agent) SetModel(m types.Model) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Model = m
}

// SetThinkingLevel updates the thinking level (takes effect on the next loop run).
func (a *Agent) SetThinkingLevel(l types.ThinkingLevel) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.ThinkingLevel = l
}

// SetTools replaces the tool list (takes effect on the next loop run).
func (a *Agent) SetTools(tools []tool.AgentTool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Tools = tools
}

// SetSteeringMode changes how queued steering messages are delivered.
func (a *Agent) SetSteeringMode(m types.SteeringMode) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.steeringMode = m
	a.steeringQ.SetMode(string(m))
}

// SetFollowUpMode changes how queued follow-up messages are delivered.
func (a *Agent) SetFollowUpMode(m types.FollowUpMode) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.followUpMode = m
	a.followUpQ.SetMode(string(m))
}

// AppendMessage appends a message to the conversation history.
func (a *Agent) AppendMessage(m types.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Messages = append(a.state.Messages, m)
}

// ReplaceMessages replaces the full message history.
func (a *Agent) ReplaceMessages(ms []types.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Messages = ms
}

// ClearMessages empties the message history.
func (a *Agent) ClearMessages() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Messages = nil
}

// emit broadcasts an event to all registered listeners.
func (a *Agent) emit(ev types.AgentEvent) {
	a.mu.RLock()
	listeners := append([]listenerEntry{}, a.listeners...)
	a.mu.RUnlock()
	for _, l := range listeners {
		l.fn(ev)
	}
}

// buildLoopConfig constructs the AgentLoopConfig from current agent state.
func (a *Agent) buildLoopConfig() AgentLoopConfig {
	a.mu.RLock()
	cfg := a.loopConfig
	model := a.state.Model
	thinkingLevel := a.state.ThinkingLevel
	a.mu.RUnlock()

	cfg.Model = model
	cfg.Options.ThinkingLevel = thinkingLevel

	// Wire up steering/follow-up queues
	steeringQ := a.steeringQ
	followUpQ := a.followUpQ
	cfg.GetSteeringMessages = func() ([]types.Message, error) {
		msgs := steeringQ.Drain()
		return msgs, nil
	}
	cfg.GetFollowUpMessages = func() ([]types.Message, error) {
		msgs := followUpQ.Drain()
		return msgs, nil
	}

	return cfg
}
