package tool

import (
	"sync"
)

// Registry stores registered AgentTool instances by name.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]AgentTool
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]AgentTool)}
}

// Register adds a tool to the registry, overwriting any existing tool with the same name.
func (r *Registry) Register(t AgentTool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get retrieves a tool by name, returning false if not found.
func (r *Registry) Get(name string) (AgentTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// All returns all registered tools in arbitrary order.
func (r *Registry) All() []AgentTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]AgentTool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}
