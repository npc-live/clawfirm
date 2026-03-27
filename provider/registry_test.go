package provider

import (
	"context"
	"testing"

	"github.com/ai-gateway/pi-go/types"
)

// fakeProvider is a minimal LLMProvider for registry tests.
type fakeProvider struct{ id string }

func (f *fakeProvider) ID() string { return f.id }
func (f *fakeProvider) Models() []types.Model { return nil }
func (f *fakeProvider) Stream(_ context.Context, _ LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	ch := make(chan types.AssistantMessageEvent)
	close(ch)
	return ch, nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeProvider{id: "test"})

	p, err := r.Get("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID() != "test" {
		t.Errorf("ID: got %q want test", p.ID())
	}
}

func TestRegistryGetMissing(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("missing")
	if err == nil {
		t.Error("expected error for missing provider")
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeProvider{id: "a"})
	r.Register(&fakeProvider{id: "b"})

	all := r.All()
	if len(all) != 2 {
		t.Errorf("All() len: got %d want 2", len(all))
	}
}
