package provider

import (
	"fmt"
	"sync"
)

// Registry holds named LLMProvider instances.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]LLMProvider
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]LLMProvider)}
}

// Register adds a provider. Overwrites any existing provider with the same ID.
func (r *Registry) Register(p LLMProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.ID()] = p
}

// Get retrieves a provider by ID, returning an error if not found.
func (r *Registry) Get(id string) (LLMProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	if !ok {
		return nil, fmt.Errorf("provider %q not registered", id)
	}
	return p, nil
}

// All returns all registered providers.
func (r *Registry) All() []LLMProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]LLMProvider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	return out
}
