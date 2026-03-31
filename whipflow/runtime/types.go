package runtime

import (
	"context"

	"github.com/ai-gateway/clawfirm/whipflow/token"
)

// RuntimeValue is the universal value type in WhipFlow (Go's any/interface{}).
type RuntimeValue = any

// SessionResult is the result from an AI session execution.
type SessionResult struct {
	Output   string
	Metadata SessionMetadata
}

type SessionMetadata struct {
	Model      string
	Duration   int64 // milliseconds
	TokensUsed int
	ToolCalls  []any
}

// IsSessionResult checks if a value is a *SessionResult.
func IsSessionResult(v any) (*SessionResult, bool) {
	sr, ok := v.(*SessionResult)
	return sr, ok
}

// ExecutionResult holds the outcome of a full program execution.
type ExecutionResult struct {
	Success        bool
	Outputs        map[string]RuntimeValue
	SessionOutputs []*SessionResult
	FinalContext    *ContextSnapshot
	Errors         []ExecutionError
	Metadata       ExecutionMetadata
}

type ExecutionMetadata struct {
	Duration           int64
	SessionsCreated    int
	StatementsExecuted int
}

// ExecutionError represents an error during execution.
type ExecutionError struct {
	Type     string // syntax, runtime, timeout, permission, variable
	Message  string
	Location *token.SourceSpan
	Stack    []string
}

func (e *ExecutionError) Error() string { return e.Message }

// ContextSnapshot captures variable state for session context.
type ContextSnapshot struct {
	Variables map[string]RuntimeValue
	Metadata  ContextSnapshotMeta
}

type ContextSnapshotMeta struct {
	Timestamp     int64
	ExecutionPath []string
}

// RuntimeConfig holds execution configuration.
type RuntimeConfig struct {
	DefaultModel          string // opus, sonnet, haiku
	SessionTimeout        int64  // ms
	TotalExecutionTimeout int64  // ms
	MaxLoopIterations     int
	MaxCallDepth          int
	MaxConcurrentSessions int
	MaxVariableSize       int64
	MaxTotalMemory        int64
	Debug                 bool
	TraceExecution        bool
	LogLevel              string // debug, info, warn, error
	DefaultProvider       string
	ConditionProvider     string
	// Ctx is the parent context for cancellation. When cancelled, running
	// CLI subprocesses (e.g. claude-code) are killed immediately.
	Ctx context.Context
	// VaultEnv, if set, is called before each CLI provider execution to
	// obtain extra environment variables (secrets from the vault).
	VaultEnv func() map[string]string
}

// DefaultRuntimeConfig returns the default configuration.
func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		DefaultModel:          "sonnet",
		SessionTimeout:        300000,
		TotalExecutionTimeout: 3600000,
		MaxLoopIterations:     100,
		MaxCallDepth:          50,
		MaxConcurrentSessions: 10,
		MaxVariableSize:       10 * 1024 * 1024,
		MaxTotalMemory:        100 * 1024 * 1024,
		Debug:                 false,
		TraceExecution:        false,
		LogLevel:              "info",
		DefaultProvider:       "claude-code",
	}
}

// ControlFlow signals for break/continue/return/throw using panic/recover.
type ControlFlow struct {
	Type  string // break, continue, return, throw
	Value RuntimeValue
	Err   error
}

// AgentInstance holds an agent's configuration.
type AgentInstance struct {
	Name        string
	Model       string
	Provider    string
	Skills      []string
	Tools       []string
	Permissions PermissionRules
	Prompt      string
}

// PermissionRules defines agent permissions.
type PermissionRules struct {
	Bash      string // allow, deny
	FileRead  string
	FileWrite string
	Network   string
	Tools     []string
	Extra     map[string]string
}

// SessionSpec defines what a session needs to execute.
type SessionSpec struct {
	Agent      *AgentInstance
	Prompt     string
	Context    *ContextSnapshot
	Name       string
	Properties map[string]RuntimeValue
}

// JoinStrategy for parallel blocks.
type JoinStrategy string

const (
	JoinAll   JoinStrategy = "all"
	JoinFirst JoinStrategy = "first"
	JoinAny   JoinStrategy = "any"
)

// FailureStrategy for parallel blocks.
type FailureStrategy string

const (
	FailFast     FailureStrategy = "fail-fast"
	FailContinue FailureStrategy = "continue"
	FailIgnore   FailureStrategy = "ignore"
)

// ExecutionEvent for tracking history.
type ExecutionEvent struct {
	Type        string // statement, session, condition, error
	Description string
	Timestamp   int64
	Result      any
}

// EnrichedExecutionContext provides rich context for discretion evaluation.
type EnrichedExecutionContext struct {
	FileName             string
	CurrentBlock         string
	CurrentIteration     int
	Variables            map[string]RuntimeValue
	RecentChanges        []string
	RecentEvents         []ExecutionEvent
	ExecutionPath        []string
	TotalStatements      int
	ExecutedStatements   int
	RemainingStatements  int
	RecentSessionOutputs []string
	LoopInfo             *LoopInfo
}

// LoopInfo holds metadata about the current loop iteration.
type LoopInfo struct {
	Iteration       int
	MaxIterations   *int
	PreviousResults []any
}

// BuiltinProviders lists known provider names.
var BuiltinProviders = []string{"claude-code", "claude", "opencode", "aider", "pi", "fetch"}
