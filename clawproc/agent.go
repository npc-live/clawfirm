package clawproc

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ai-gateway/clawfirm/types"
)

// ClawAgent wraps a claw subprocess and implements gateway.AgentRunner.
type ClawAgent struct {
	proc     *Process
	mu       sync.RWMutex
	startErr error // non-nil if Start() failed

	listeners   []listenerEntry
	listenerSeq uint64
	isRunning   bool
	model       string
	idleCh      chan struct{}
	abortFlag   atomic.Bool
}

type listenerEntry struct {
	id uint64
	fn func(types.AgentEvent)
}

// NewClawAgent creates a ClawAgent backed by the given Process.
func NewClawAgent(proc *Process) *ClawAgent {
	ch := make(chan struct{})
	close(ch) // starts idle
	return &ClawAgent{
		proc:   proc,
		idleCh: ch,
	}
}

// Start spawns the subprocess. Must be called before PromptMessages.
// If Start fails, PromptMessages will emit an error event with the failure reason.
func (a *ClawAgent) Start(ctx context.Context) error {
	ready, err := a.proc.Start(ctx)
	if err != nil {
		a.startErr = err
		return err
	}
	a.model = ready.Model
	return nil
}

// PromptMessages extracts text from msgs and sends to the claw subprocess.
func (a *ClawAgent) PromptMessages(ctx context.Context, msgs []types.Message) error {
	a.mu.Lock()
	if a.isRunning {
		a.mu.Unlock()
		return nil
	}
	a.isRunning = true
	a.idleCh = make(chan struct{})
	a.abortFlag.Store(false)
	idleCh := a.idleCh
	a.mu.Unlock()

	// If the subprocess never started (or was killed by Abort), restart it.
	if a.startErr != nil || !a.proc.Alive() {
		if _, err := a.proc.Start(ctx); err != nil {
			a.startErr = err
			go func() {
				defer func() {
					a.mu.Lock()
					a.isRunning = false
					a.mu.Unlock()
					close(idleCh)
				}()
				a.emitError("claw engine failed to start: " + err.Error())
			}()
			return nil
		}
		a.startErr = nil
	}

	// Extract text from the user messages.
	text := extractText(msgs)
	if text == "" {
		a.mu.Lock()
		a.isRunning = false
		a.mu.Unlock()
		close(idleCh)
		return nil
	}

	go func() {
		defer func() {
			a.mu.Lock()
			a.isRunning = false
			a.mu.Unlock()
			close(idleCh)
		}()

		if err := a.proc.SendPrompt(text); err != nil {
			a.emitError("claw engine error: " + err.Error())
			return
		}

		a.emit(types.AgentEvent{Type: types.EventAgentStart})

		var cumUsage types.CumulativeUsage
		a.readLoop(&cumUsage)
	}()

	return nil
}

// readLoop reads NDJSON events from the subprocess until turn_end or error.
func (a *ClawAgent) readLoop(cumUsage *types.CumulativeUsage) {
	// Per-message accumulators — reset on each message_stop.
	var textParts []string
	var toolCalls []types.ContentBlock

	// Per-turn accumulator for tool results.
	var toolResults []types.ToolResultMessage

	// whipflowStopCh is non-nil while a whipflow_run tool is executing.
	// Closing it signals tailWhipflowProgress to drain and exit.
	var whipflowStopCh chan struct{}

	flushMessage := func() {
		if len(textParts) == 0 && len(toolCalls) == 0 {
			return
		}
		var content []types.ContentBlock
		if joined := strings.Join(textParts, ""); joined != "" {
			content = append(content, &types.TextContent{Type: types.ContentTypeText, Text: joined})
		}
		content = append(content, toolCalls...)
		msg := &types.AssistantMessage{
			Role:      "assistant",
			Content:   content,
			Model:     a.model,
			Timestamp: time.Now().UnixMilli(),
		}
		a.emit(types.AgentEvent{
			Type:         types.EventMessageEnd,
			AssistantMsg: msg,
		})
		textParts = nil
		toolCalls = nil
	}

	for {
		ev, err := a.proc.ReadEvent()
		if err != nil {
			if err != io.EOF {
				log.Printf("clawproc: read event: %v", err)
			}
			flushMessage()
			a.emit(types.AgentEvent{Type: types.EventAgentEnd, CumulativeUsage: cumUsage})
			return
		}

		switch ev.Type {
		case "session_init":
			// Internal bookkeeping, no Go event needed.

		case "text_delta":
			textParts = append(textParts, ev.Text)
			a.emit(types.AgentEvent{
				Type: types.EventMessageUpdate,
				StreamEvent: &types.AssistantMessageEvent{
					Type:  types.StreamEventTextDelta,
					Delta: ev.Text,
				},
			})

		case "tool_use":
			log.Printf("clawproc: tool_use name=%q id=%s", ev.Name, ev.ID)
			// Parse arguments from JSON string to map.
			var args map[string]any
			if ev.Input != "" {
				_ = json.Unmarshal([]byte(ev.Input), &args)
			}
			toolCalls = append(toolCalls, &types.ToolCall{
				Type:      types.ContentTypeToolCall,
				ID:        ev.ID,
				Name:      ev.Name,
				Arguments: args,
			})
			a.emit(types.AgentEvent{
				Type:       types.EventToolExecutionStart,
				ToolCallID: ev.ID,
				ToolName:   ev.Name,
				ToolArgs:   ev.Input,
			})
			// For whipflow_run, start tailing the progress file.
			if ev.Name == "whipflow_run" {
				stopCh := make(chan struct{})
				whipflowStopCh = stopCh
				go tailWhipflowProgress(stopCh, ev.ID, a)
			}

		case "tool_result":
			if ev.ToolName == "whipflow_run" || ev.ToolName == "execute_whipflow" {
				log.Printf("clawproc: tool_result name=%q content_len=%d content_prefix=%.200s",
					ev.ToolName, len(ev.Content), ev.Content)
			}
			// Stop the whipflow progress tailer (if any) and let it drain.
			if whipflowStopCh != nil {
				close(whipflowStopCh)
				whipflowStopCh = nil
			}
			toolResults = append(toolResults, types.ToolResultMessage{
				Role:       "tool",
				ToolCallID: ev.ToolUseID,
				ToolName:   ev.ToolName,
				Content:    []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: ev.Content}},
				IsError:    ev.IsError,
				Timestamp:  time.Now().UnixMilli(),
			})
			a.emit(types.AgentEvent{
				Type:        types.EventToolExecutionEnd,
				ToolCallID:  ev.ToolUseID,
				ToolName:    ev.ToolName,
				ToolResult:  ev.Content,
				ToolIsError: ev.IsError,
			})

		case "usage":
			cumUsage.TotalInput += ev.InputTokens
			cumUsage.TotalOutput += ev.OutputTokens

		case "message_stop":
			flushMessage()

		case "turn_end":
			flushMessage()
			cumUsage.TurnCount += ev.Iterations
			a.emit(types.AgentEvent{
				Type:            types.EventTurnEnd,
				CumulativeUsage: cumUsage,
				ToolResults:     toolResults,
			})
			toolResults = nil
			a.emit(types.AgentEvent{
				Type:            types.EventAgentEnd,
				CumulativeUsage: cumUsage,
			})
			return

		case "error":
			log.Printf("clawproc: error from subprocess: %s", ev.Message)
			flushMessage()
			a.emit(types.AgentEvent{
				Type:            types.EventAgentEnd,
				CumulativeUsage: cumUsage,
			})
			return

		default:
			log.Printf("clawproc: unhandled event type=%q", ev.Type)
		}
	}
}

// Abort signals the subprocess to cancel the current turn.
// It sends the abort command immediately (non-blocking) so it reaches
// the subprocess even while readLoop is blocked on ReadEvent().
func (a *ClawAgent) Abort() {
	a.abortFlag.Store(true)
	_ = a.proc.SendAbort()
}

// WaitForIdle blocks until the agent is idle or ctx is cancelled.
func (a *ClawAgent) WaitForIdle(ctx context.Context) error {
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

// Subscribe registers an event listener. Returns an unsubscribe function.
func (a *ClawAgent) Subscribe(fn func(types.AgentEvent)) func() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.listenerSeq++
	id := a.listenerSeq
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

// State returns a minimal agent state snapshot.
func (a *ClawAgent) State() types.AgentState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return types.AgentState{
		Model:     types.Model{ID: a.model},
		IsRunning: a.isRunning,
	}
}

// ClearMessages is a no-op for the claw subprocess (it manages its own history).
func (a *ClawAgent) ClearMessages() {}

// Close shuts down the subprocess.
func (a *ClawAgent) Close() error {
	return a.proc.Close()
}

func (a *ClawAgent) emit(ev types.AgentEvent) {
	a.mu.RLock()
	listeners := append([]listenerEntry{}, a.listeners...)
	a.mu.RUnlock()
	for _, l := range listeners {
		l.fn(ev)
	}
}

// emitError sends an error as a visible text_delta message + agent_end so the
// user sees it in the chat UI.
func (a *ClawAgent) emitError(msg string) {
	log.Printf("clawproc: %s", msg)
	a.emit(types.AgentEvent{
		Type: types.EventMessageUpdate,
		StreamEvent: &types.AssistantMessageEvent{
			Type:  types.StreamEventTextDelta,
			Delta: "[Error] " + msg,
		},
	})
	a.emit(types.AgentEvent{Type: types.EventAgentEnd})
}

// extractText pulls all user text blocks from the message slice and joins them.
// This ensures file-attachment hints (added by the gateway session) are included
// alongside the user's original text.
func extractText(msgs []types.Message) string {
	var parts []string
	for _, m := range msgs {
		if um, ok := m.(*types.UserMessage); ok {
			for _, b := range um.Content {
				if tb, ok := b.(*types.TextContent); ok && tb.Text != "" {
					parts = append(parts, tb.Text)
				}
			}
		}
	}
	return strings.Join(parts, "\n")
}

// whipflowProgressPath returns the temp file path that execute_whipflow (Rust)
// writes session_progress NDJSON lines to. Must match std::env::temp_dir() on
// the same OS (macOS uses $TMPDIR=/var/folders/..., not /tmp).
func whipflowProgressPath() string {
	return filepath.Join(os.TempDir(), "whipflow-progress.ndjson")
}

// tailWhipflowProgress tails whipflowProgressPath and emits an
// EventToolExecutionUpdate for each session_progress line it reads.
// It stops when stopCh is closed and drains any remaining lines before returning.
//
// Uses manual Read + position tracking instead of bufio.Scanner because
// Scanner.Scan() returns false permanently at EOF and won't re-read new
// data appended to the file later — defeating the "tail -f" behaviour.
func tailWhipflowProgress(stopCh <-chan struct{}, toolCallID string, a *ClawAgent) {
	progressPath := whipflowProgressPath()

	// Wait up to 5s for Rust to create AND truncate the file.
	// We record the time just before waiting so we can verify the file was
	// written AFTER this run started (avoids reading stale data from a
	// previous run that already truncated+wrote the file).
	startTime := time.Now()
	deadline := startTime.Add(5 * time.Second)
	var f *os.File
	for {
		var err error
		f, err = os.Open(progressPath)
		if err == nil {
			// File exists — verify it was truncated by THIS run, not a previous one.
			// Rust truncates (writes empty bytes) right before spawning whip.
			// We wait until the file's mtime is >= startTime.
			if info, statErr := f.Stat(); statErr == nil && !info.ModTime().Before(startTime) {
				break
			}
			f.Close()
		}
		if time.Now().After(deadline) {
			log.Printf("clawproc: whipflow progress file not ready at %s after 5s", progressPath)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	defer f.Close()

	emitLine := func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}
		var step map[string]any
		if err := json.Unmarshal([]byte(line), &step); err != nil {
			return
		}
		log.Printf("clawproc: whipflow progress: %s", line)
		a.emit(types.AgentEvent{
			Type:          types.EventToolExecutionUpdate,
			ToolCallID:    toolCallID,
			ToolName:      "whipflow_run",
			PartialResult: step,
		})
	}

	buf := make([]byte, 32*1024)
	var pending string // incomplete line buffer

	readNew := func() {
		for {
			n, err := f.Read(buf)
			if n > 0 {
				chunk := pending + string(buf[:n])
				lines := strings.Split(chunk, "\n")
				// All lines except the last are complete.
				for _, l := range lines[:len(lines)-1] {
					emitLine(l)
				}
				pending = lines[len(lines)-1] // might be partial
			}
			if err != nil {
				break // io.EOF or real error — stop reading this batch
			}
		}
	}

	for {
		readNew()
		select {
		case <-stopCh:
			// One final read to drain lines written just before tool_result.
			readNew()
			if pending != "" {
				emitLine(pending)
			}
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}
