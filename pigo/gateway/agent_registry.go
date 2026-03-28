package gateway

import "sync"

// AgentRegistry holds a named set of SessionManagers — one per configured agent.
// The registry is the central lookup used by the HTTP server and channel handlers.
type AgentRegistry struct {
	mu       sync.RWMutex
	managers map[string]*SessionManager
	order    []string // insertion order, used by Names() and Counts()
}

// NewAgentRegistry creates an empty AgentRegistry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{managers: make(map[string]*SessionManager)}
}

// Register adds a named agent's SessionManager.
// Calling Register twice with the same name replaces the previous entry.
func (r *AgentRegistry) Register(name string, mgr *SessionManager) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.managers[name]; !exists {
		r.order = append(r.order, name)
	}
	r.managers[name] = mgr
}

// Get returns the SessionManager for the given agent name.
func (r *AgentRegistry) Get(name string) (*SessionManager, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	mgr, ok := r.managers[name]
	return mgr, ok
}

// Names returns agent names in registration order.
func (r *AgentRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// Counts returns active session counts keyed by agent name.
func (r *AgentRegistry) Counts() map[string]int {
	r.mu.RLock()
	names := make([]string, len(r.order))
	copy(names, r.order)
	r.mu.RUnlock()

	out := make(map[string]int, len(names))
	for _, name := range names {
		r.mu.RLock()
		mgr := r.managers[name]
		r.mu.RUnlock()
		if mgr != nil {
			out[name] = mgr.Count()
		}
	}
	return out
}

// Stop shuts down all registered SessionManagers.
func (r *AgentRegistry) Stop() {
	r.mu.Lock()
	mgrs := make([]*SessionManager, 0, len(r.managers))
	for _, mgr := range r.managers {
		mgrs = append(mgrs, mgr)
	}
	r.mu.Unlock()
	for _, mgr := range mgrs {
		mgr.Stop()
	}
}
