package runtime

import (
	"fmt"
	"log"
	"time"
)

// RuntimeEnvironment provides infrastructure for execution.
type RuntimeEnvironment struct {
	Config         RuntimeConfig
	Context        *ContextManager
	agents         map[string]*AgentInstance
	errors         []ExecutionError
	startTime      time.Time
	sessionCount   int
	statementCount int
	callDepth      int
}

// NewRuntimeEnvironment creates a new environment with the given config
// (or defaults if config is nil).
func NewRuntimeEnvironment(config *RuntimeConfig) *RuntimeEnvironment {
	cfg := DefaultRuntimeConfig()
	if config != nil {
		cfg = *config
	}
	return &RuntimeEnvironment{
		Config:  cfg,
		Context: NewContextManager(),
		agents:  make(map[string]*AgentInstance),
	}
}

// StartExecution initializes execution state and records the start time.
func (env *RuntimeEnvironment) StartExecution() {
	env.startTime = time.Now()
	env.errors = nil
	env.sessionCount = 0
	env.statementCount = 0
	env.callDepth = 0
}

// GetExecutionDuration returns the elapsed time since StartExecution in milliseconds.
func (env *RuntimeEnvironment) GetExecutionDuration() int64 {
	return time.Since(env.startTime).Milliseconds()
}

// HasTimedOut returns true if the total execution timeout has been exceeded.
func (env *RuntimeEnvironment) HasTimedOut() bool {
	return env.GetExecutionDuration() > env.Config.TotalExecutionTimeout
}

// RegisterAgent registers an agent instance by name.
func (env *RuntimeEnvironment) RegisterAgent(agent *AgentInstance) {
	env.agents[agent.Name] = agent
}

// GetAgent retrieves a registered agent by name.
func (env *RuntimeEnvironment) GetAgent(name string) (*AgentInstance, bool) {
	a, ok := env.agents[name]
	return a, ok
}

// IncrementSessionCount increments the session counter.
func (env *RuntimeEnvironment) IncrementSessionCount() { env.sessionCount++ }

// GetSessionCount returns the current session count.
func (env *RuntimeEnvironment) GetSessionCount() int { return env.sessionCount }

// IncrementStatementCount increments the statement counter.
func (env *RuntimeEnvironment) IncrementStatementCount() { env.statementCount++ }

// GetStatementCount returns the current statement count.
func (env *RuntimeEnvironment) GetStatementCount() int { return env.statementCount }

// IncrementCallDepth increments the call depth and returns an error if the
// maximum call depth has been exceeded.
func (env *RuntimeEnvironment) IncrementCallDepth() error {
	env.callDepth++
	if env.callDepth > env.Config.MaxCallDepth {
		return fmt.Errorf("maximum call depth (%d) exceeded", env.Config.MaxCallDepth)
	}
	return nil
}

// DecrementCallDepth decrements the call depth.
func (env *RuntimeEnvironment) DecrementCallDepth() { env.callDepth-- }

// AddError appends an execution error to the error list.
func (env *RuntimeEnvironment) AddError(err ExecutionError) {
	env.errors = append(env.errors, err)
}

// GetErrors returns a copy of the error list.
func (env *RuntimeEnvironment) GetErrors() []ExecutionError {
	result := make([]ExecutionError, len(env.errors))
	copy(result, env.errors)
	return result
}

// HasErrors returns true if any errors have been recorded.
func (env *RuntimeEnvironment) HasErrors() bool { return len(env.errors) > 0 }

var logLevelRank = map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}

// Log outputs a log message if the given level meets or exceeds the configured log level.
func (env *RuntimeEnvironment) Log(level, message string) {
	if logLevelRank[level] < logLevelRank[env.Config.LogLevel] {
		return
	}
	log.Printf("[%s] %s", level, message)
}

// Trace outputs a trace-level debug message if trace execution is enabled.
func (env *RuntimeEnvironment) Trace(message string) {
	if env.Config.TraceExecution {
		env.Log("debug", "TRACE: "+message)
	}
}

// Reset clears all environment state and reinitializes with a fresh context.
func (env *RuntimeEnvironment) Reset() {
	env.Context.Reset()
	env.agents = make(map[string]*AgentInstance)
	env.errors = nil
	env.startTime = time.Time{}
	env.sessionCount = 0
	env.statementCount = 0
	env.callDepth = 0
}
