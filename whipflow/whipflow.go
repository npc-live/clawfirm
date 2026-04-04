// Package whipflow provides the top-level API for parsing, validating,
// and executing WhipFlow programs (.whip files).
package whipflow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/whipflow/ast"
	"github.com/ai-gateway/clawfirm/whipflow/parser"
	"github.com/ai-gateway/clawfirm/whipflow/runtime"
	"github.com/ai-gateway/clawfirm/whipflow/validator"
)

// Re-export key types for convenient access.
type ExecutionResult = runtime.ExecutionResult
type ValidationResult = validator.ValidationResult
type SessionProgress = runtime.SessionProgress

// Parse tokenizes and parses WhipFlow source code into an AST.
func Parse(source string) (*ast.Program, []error) {
	result := parser.Parse(source)
	if len(result.Errors) > 0 {
		errs := make([]error, len(result.Errors))
		for i, e := range result.Errors {
			errs[i] = fmt.Errorf("line %d:%d: %s", e.Span.Start.Line, e.Span.Start.Column, e.Message)
		}
		return result.Program, errs
	}
	return result.Program, nil
}

// Validate performs semantic validation on a parsed program.
func Validate(program *ast.Program) *validator.ValidationResult {
	return validator.Validate(program)
}

// Option configures execution.
type Option func(*executeConfig)

type executeConfig struct {
	runtimeConfig           *runtime.RuntimeConfig
	stateStorePath          string
	fileName                string
	toolRegistry            *runtime.ToolRegistry
	piConfig                *config.Config
	nativeProviders         map[string]runtime.Provider
	sessionProgressCallback func(runtime.SessionProgress)
	initialInputs           map[string]string
	retryFromSession        int // -1 means not set
}

// WithRuntimeConfig sets the runtime configuration.
func WithRuntimeConfig(cfg *runtime.RuntimeConfig) Option {
	return func(c *executeConfig) { c.runtimeConfig = cfg }
}

// WithStateStore enables SQLite state persistence at the given path.
func WithStateStore(dbPath string) Option {
	return func(c *executeConfig) { c.stateStorePath = dbPath }
}

// WithFileName sets the source file name (used for state store keying).
func WithFileName(name string) Option {
	return func(c *executeConfig) { c.fileName = name }
}

// WithToolRegistry provides a custom tool registry.
func WithToolRegistry(tr *runtime.ToolRegistry) Option {
	return func(c *executeConfig) { c.toolRegistry = tr }
}

// WithPiConfig sets the clawfirm config, enabling WhipFlow to resolve
// native providers from the top-level agents and providers sections
// in ~/.clawfirm/config.yml.
func WithPiConfig(cfg *config.Config) Option {
	return func(c *executeConfig) { c.piConfig = cfg }
}

// WithSessionProgressCallback registers a callback invoked before (Done=false)
// and after (Done=true) each session execution.
func WithSessionProgressCallback(cb func(SessionProgress)) Option {
	return func(c *executeConfig) { c.sessionProgressCallback = cb }
}

// WithNativeProvider registers a named native provider for session execution.
// This allows WhipFlow sessions to use clawfirm's in-process agent system
// instead of spawning external CLI processes.
func WithNativeProvider(name string, p runtime.Provider) Option {
	return func(c *executeConfig) {
		if c.nativeProviders == nil {
			c.nativeProviders = make(map[string]runtime.Provider)
		}
		c.nativeProviders[name] = p
	}
}

// WithInitialInputs pre-fills answers for ask statements, bypassing stdin.
func WithInitialInputs(inputs map[string]string) Option {
	return func(c *executeConfig) { c.initialInputs = inputs }
}

// WithRetryFromSession configures the execution to retry from a specific
// session index. Sessions before this index are replayed from the state store;
// sessions at and after this index are re-executed.
func WithRetryFromSession(idx int) Option {
	return func(c *executeConfig) { c.retryFromSession = idx }
}

// Execute runs a parsed WhipFlow program.
func Execute(program *ast.Program, opts ...Option) (*runtime.ExecutionResult, error) {
	cfg := &executeConfig{retryFromSession: -1}
	for _, opt := range opts {
		opt(cfg)
	}

	env := runtime.NewRuntimeEnvironment(cfg.runtimeConfig)

	var interpOpts []runtime.InterpreterOption

	// Tool registry.
	tr := cfg.toolRegistry
	if tr == nil {
		tr = runtime.NewToolRegistry()
	}
	interpOpts = append(interpOpts, runtime.WithToolRegistry(tr))

	// State store for resume support.
	var store *runtime.StateStore
	if cfg.stateStorePath != "" {
		var err error
		store, err = runtime.NewStateStore(cfg.stateStorePath)
		if err != nil {
			return nil, fmt.Errorf("whipflow: failed to open state store: %w", err)
		}
		defer store.Close()
		interpOpts = append(interpOpts, runtime.WithStateStore(store))

		// Check for incomplete run to resume.
		if cfg.fileName != "" {
			if prev, err := store.FindIncompleteRun(cfg.fileName); err == nil && prev != nil {
				// If retrying from a specific session, delete that session and all after it.
				if cfg.retryFromSession >= 0 {
					if err := store.DeleteSessionsFrom(prev.ID, cfg.retryFromSession); err != nil {
						return nil, fmt.Errorf("whipflow: failed to delete sessions for retry: %w", err)
					}
					env.Log("info", fmt.Sprintf("Retry: deleted sessions from index %d in run #%d", cfg.retryFromSession, prev.ID))
				}
				sessions, err := store.GetCompletedSessions(prev.ID)
				if err == nil && len(sessions) > 0 {
					interpOpts = append(interpOpts, runtime.WithReplaySessions(sessions))
					env.Log("info", fmt.Sprintf("Resuming from run #%d with %d completed sessions", prev.ID, len(sessions)))
				}
			}
		}
	}

	if cfg.fileName != "" {
		interpOpts = append(interpOpts, runtime.WithFileName(cfg.fileName))
	}

	// Clawfirm config for resolving providers from ~/.clawfirm/config.yml.
	if cfg.piConfig != nil {
		interpOpts = append(interpOpts, runtime.WithPiConfig(cfg.piConfig))
	}

	// Inject native providers if configured programmatically.
	if len(cfg.nativeProviders) > 0 {
		interpOpts = append(interpOpts, runtime.WithNativeProviders(cfg.nativeProviders))
	}

	// Session progress callback.
	if cfg.sessionProgressCallback != nil {
		interpOpts = append(interpOpts, runtime.WithSessionProgressCallback(cfg.sessionProgressCallback))
	}

	// Pre-filled ask inputs.
	if len(cfg.initialInputs) > 0 {
		interpOpts = append(interpOpts, runtime.WithInitialInputs(cfg.initialInputs))
	}

	interp := runtime.NewInterpreter(env, interpOpts...)
	return interp.Execute(program)
}

// RunFile reads, parses, validates, and executes a .whip file.
func RunFile(path string, opts ...Option) (*runtime.ExecutionResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("whipflow: invalid path: %w", err)
	}

	source, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("whipflow: cannot read file: %w", err)
	}

	program, parseErrors := Parse(string(source))
	if len(parseErrors) > 0 {
		return nil, fmt.Errorf("whipflow: parse errors: %v", parseErrors[0])
	}

	vResult := Validate(program)
	if !vResult.Valid {
		return nil, fmt.Errorf("whipflow: validation error: %s", vResult.Errors[0].Message)
	}

	// Default state store in ~/.clawfirm/whipflow-state.db
	homeDir, _ := os.UserHomeDir()
	defaultStorePath := filepath.Join(homeDir, ".clawfirm", "whipflow-state.db")

	allOpts := []Option{
		WithFileName(absPath),
		WithStateStore(defaultStorePath),
	}
	allOpts = append(allOpts, opts...)

	return Execute(program, allOpts...)
}
