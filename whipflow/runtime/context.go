package runtime

import (
	"fmt"
	"sync"
	"time"

	"github.com/ai-gateway/clawfirm/whipflow/token"
)

// variable stored in a scope.
type variable struct {
	Name       string
	Value      RuntimeValue
	IsConst    bool
	DeclaredAt token.SourceSpan
}

// Scope holds variables and links to parent scope.
type Scope struct {
	mu        sync.RWMutex
	variables map[string]*variable
	parent    *Scope
}

func newScope(parent *Scope) *Scope {
	return &Scope{variables: make(map[string]*variable), parent: parent}
}

func (s *Scope) declare(name string, value RuntimeValue, isConst bool, location token.SourceSpan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.variables[name]; exists {
		return fmt.Errorf("variable '%s' is already declared in this scope", name)
	}
	s.variables[name] = &variable{Name: name, Value: value, IsConst: isConst, DeclaredAt: location}
	return nil
}

func (s *Scope) get(name string) (*variable, bool) {
	s.mu.RLock()
	v, ok := s.variables[name]
	s.mu.RUnlock()
	if ok {
		return v, true
	}
	if s.parent != nil {
		return s.parent.get(name)
	}
	return nil, false
}

func (s *Scope) set(name string, value RuntimeValue) error {
	s.mu.Lock()
	v, ok := s.variables[name]
	if ok {
		if v.IsConst {
			s.mu.Unlock()
			return fmt.Errorf("cannot reassign const variable '%s'", name)
		}
		v.Value = value
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()
	if s.parent != nil {
		return s.parent.set(name, value)
	}
	return fmt.Errorf("variable '%s' is not declared", name)
}

func (s *Scope) has(name string) bool {
	s.mu.RLock()
	_, ok := s.variables[name]
	s.mu.RUnlock()
	if ok {
		return true
	}
	if s.parent != nil {
		return s.parent.has(name)
	}
	return false
}

func (s *Scope) getAll() map[string]*variable {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*variable, len(s.variables))
	for k, v := range s.variables {
		result[k] = v
	}
	return result
}

// ContextManager manages variable scopes with a scope stack.
type ContextManager struct {
	scopeStack    []*Scope
	executionPath []string
}

// NewContextManager creates a new ContextManager with a global scope.
func NewContextManager() *ContextManager {
	cm := &ContextManager{}
	cm.PushScope()
	return cm
}

// PushScope creates a new child scope and pushes it onto the stack.
func (cm *ContextManager) PushScope() {
	var parent *Scope
	if len(cm.scopeStack) > 0 {
		parent = cm.scopeStack[len(cm.scopeStack)-1]
	}
	cm.scopeStack = append(cm.scopeStack, newScope(parent))
}

// PopScope removes the top scope from the stack.
func (cm *ContextManager) PopScope() error {
	if len(cm.scopeStack) <= 1 {
		return fmt.Errorf("cannot pop global scope")
	}
	cm.scopeStack = cm.scopeStack[:len(cm.scopeStack)-1]
	return nil
}

func (cm *ContextManager) currentScope() *Scope {
	return cm.scopeStack[len(cm.scopeStack)-1]
}

// DeclareVariable declares a new variable in the current scope.
func (cm *ContextManager) DeclareVariable(name string, value RuntimeValue, isConst bool, location token.SourceSpan) error {
	return cm.currentScope().declare(name, value, isConst, location)
}

// GetVariable retrieves a variable's value by name, searching up the scope chain.
func (cm *ContextManager) GetVariable(name string) (RuntimeValue, error) {
	v, ok := cm.currentScope().get(name)
	if !ok {
		return nil, fmt.Errorf("variable '%s' is not defined", name)
	}
	return v.Value, nil
}

// SetVariable assigns a new value to an existing variable.
func (cm *ContextManager) SetVariable(name string, value RuntimeValue) error {
	return cm.currentScope().set(name, value)
}

// HasVariable checks whether a variable exists in the scope chain.
func (cm *ContextManager) HasVariable(name string) bool {
	return cm.currentScope().has(name)
}

// AddToExecutionPath appends a description to the execution path trace.
func (cm *ContextManager) AddToExecutionPath(description string) {
	cm.executionPath = append(cm.executionPath, description)
}

// CaptureContext creates a snapshot of the current variable state.
// If variableNames are provided, only those variables are captured;
// otherwise all variables across all scopes are captured.
func (cm *ContextManager) CaptureContext(variableNames ...string) *ContextSnapshot {
	vars := make(map[string]RuntimeValue)
	if len(variableNames) > 0 {
		for _, name := range variableNames {
			if v, err := cm.GetVariable(name); err == nil {
				vars[name] = v
			}
		}
	} else {
		for _, scope := range cm.scopeStack {
			for name, v := range scope.getAll() {
				vars[name] = v.Value
			}
		}
	}
	path := make([]string, len(cm.executionPath))
	copy(path, cm.executionPath)
	return &ContextSnapshot{
		Variables: vars,
		Metadata: ContextSnapshotMeta{
			Timestamp:     time.Now().UnixMilli(),
			ExecutionPath: path,
		},
	}
}

// GetAllVariables returns all variables across all scopes.
func (cm *ContextManager) GetAllVariables() map[string]RuntimeValue {
	all := make(map[string]RuntimeValue)
	for _, scope := range cm.scopeStack {
		for name, v := range scope.getAll() {
			all[name] = v.Value
		}
	}
	return all
}

// GetExecutionPath returns a copy of the execution path.
func (cm *ContextManager) GetExecutionPath() []string {
	path := make([]string, len(cm.executionPath))
	copy(path, cm.executionPath)
	return path
}

// Reset clears all scopes and execution path, then creates a fresh global scope.
func (cm *ContextManager) Reset() {
	cm.scopeStack = nil
	cm.executionPath = nil
	cm.PushScope()
}
