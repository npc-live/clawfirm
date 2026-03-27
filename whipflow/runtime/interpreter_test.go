package runtime

import (
	"context"
	"testing"

	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/types"
	"github.com/ai-gateway/pi-go/whipflow/ast"
	"github.com/ai-gateway/pi-go/whipflow/token"
)

// mockLLMProvider is a test double that returns a fixed text response.
type mockLLMProvider struct {
	response string
}

func (m *mockLLMProvider) ID() string { return "mock" }
func (m *mockLLMProvider) Models() []types.Model {
	return []types.Model{{ID: "mock-model", Provider: "mock"}}
}
func (m *mockLLMProvider) Stream(_ context.Context, _ provider.LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	ch := make(chan types.AssistantMessageEvent, 3)
	ch <- types.AssistantMessageEvent{Type: types.StreamEventStart}
	ch <- types.AssistantMessageEvent{
		Type: types.StreamEventDone,
		Message: &types.AssistantMessage{
			Role: "assistant",
			Content: []types.ContentBlock{
				&types.TextContent{Type: "text", Text: m.response},
			},
			StopReason: types.StopReasonStop,
		},
		Reason: types.StopReasonStop,
	}
	close(ch)
	return ch, nil
}

func zeroSpan() token.SourceSpan {
	return token.SourceSpan{}
}

func TestInterpreterLetBinding(t *testing.T) {
	env := NewRuntimeEnvironment(nil)
	interp := NewInterpreter(env)

	program := &ast.Program{
		Span: zeroSpan(),
		Statements: []ast.Node{
			&ast.LetBinding{
				Span: zeroSpan(),
				Name: &ast.Identifier{Span: zeroSpan(), Name: "x"},
				Value: &ast.StringLiteral{Span: zeroSpan(), Value: "hello"},
			},
		},
	}

	result, err := interp.Execute(program)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got errors: %v", result.Errors)
	}

	// Check variable was declared.
	v, verr := env.Context.GetVariable("x")
	if verr != nil {
		t.Fatalf("variable x not found: %v", verr)
	}
	if v != "hello" {
		t.Errorf("expected 'hello', got %v", v)
	}
}

func TestInterpreterConstBinding(t *testing.T) {
	env := NewRuntimeEnvironment(nil)
	interp := NewInterpreter(env)

	program := &ast.Program{
		Span: zeroSpan(),
		Statements: []ast.Node{
			&ast.ConstBinding{
				Span: zeroSpan(),
				Name: &ast.Identifier{Span: zeroSpan(), Name: "PI"},
				Value: &ast.NumberLiteral{Span: zeroSpan(), Value: 3.14, Raw: "3.14"},
			},
		},
	}

	result, err := interp.Execute(program)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success")
	}

	v, verr := env.Context.GetVariable("PI")
	if verr != nil {
		t.Fatalf("variable PI not found: %v", verr)
	}
	if v != 3.14 {
		t.Errorf("expected 3.14, got %v", v)
	}
}

func TestInterpreterRepeatBlock(t *testing.T) {
	env := NewRuntimeEnvironment(nil)
	interp := NewInterpreter(env)

	// repeat 3 as i:
	//   let x = i  (each iteration overwrites; last value = 2)
	program := &ast.Program{
		Span: zeroSpan(),
		Statements: []ast.Node{
			&ast.LetBinding{
				Span:  zeroSpan(),
				Name:  &ast.Identifier{Span: zeroSpan(), Name: "result"},
				Value: &ast.NumberLiteral{Span: zeroSpan(), Value: 0, Raw: "0"},
			},
			&ast.RepeatBlock{
				Span:     zeroSpan(),
				Count:    &ast.NumberLiteral{Span: zeroSpan(), Value: 3, Raw: "3"},
				IndexVar: &ast.Identifier{Span: zeroSpan(), Name: "i"},
				Body:     []ast.Node{},
			},
		},
	}

	result, err := interp.Execute(program)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got errors: %v", result.Errors)
	}
}

func TestInterpreterAgentDefinition(t *testing.T) {
	env := NewRuntimeEnvironment(nil)
	interp := NewInterpreter(env)

	program := &ast.Program{
		Span: zeroSpan(),
		Statements: []ast.Node{
			&ast.AgentDefinition{
				Span: zeroSpan(),
				Name: &ast.Identifier{Span: zeroSpan(), Name: "coder"},
				Properties: []*ast.Property{
					{
						Span: zeroSpan(),
						Name: &ast.Identifier{Span: zeroSpan(), Name: "model"},
						Value: &ast.Identifier{Span: zeroSpan(), Name: "sonnet"},
					},
				},
				Body: nil,
			},
		},
	}

	result, err := interp.Execute(program)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success")
	}

	agent, ok := env.GetAgent("coder")
	if !ok {
		t.Fatal("agent 'coder' not found")
	}
	if agent.Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got %q", agent.Model)
	}
}

func TestInterpreterArrayExpression(t *testing.T) {
	env := NewRuntimeEnvironment(nil)
	interp := NewInterpreter(env)

	program := &ast.Program{
		Span: zeroSpan(),
		Statements: []ast.Node{
			&ast.LetBinding{
				Span: zeroSpan(),
				Name: &ast.Identifier{Span: zeroSpan(), Name: "items"},
				Value: &ast.ArrayExpression{
					Span: zeroSpan(),
					Elements: []ast.Node{
						&ast.StringLiteral{Span: zeroSpan(), Value: "a"},
						&ast.StringLiteral{Span: zeroSpan(), Value: "b"},
						&ast.StringLiteral{Span: zeroSpan(), Value: "c"},
					},
				},
			},
		},
	}

	result, err := interp.Execute(program)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success")
	}

	v, verr := env.Context.GetVariable("items")
	if verr != nil {
		t.Fatalf("variable items not found: %v", verr)
	}

	items, ok := v.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", v)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestInterpreterTryCatch(t *testing.T) {
	env := NewRuntimeEnvironment(nil)
	interp := NewInterpreter(env)

	program := &ast.Program{
		Span: zeroSpan(),
		Statements: []ast.Node{
			&ast.TryBlock{
				Span: zeroSpan(),
				TryBody: []ast.Node{
					&ast.ThrowStatement{
						Span:    zeroSpan(),
						Message: &ast.StringLiteral{Span: zeroSpan(), Value: "oops"},
					},
				},
				CatchBody: []ast.Node{
					&ast.LetBinding{
						Span:  zeroSpan(),
						Name:  &ast.Identifier{Span: zeroSpan(), Name: "caught"},
						Value: &ast.StringLiteral{Span: zeroSpan(), Value: "handled"},
					},
				},
				ErrorVar: &ast.Identifier{Span: zeroSpan(), Name: "err"},
			},
		},
	}

	result, err := interp.Execute(program)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success (error should be caught)")
	}
}

func TestNativeProviderSessionExecution(t *testing.T) {
	// Create a mock LLM provider that returns a fixed response.
	mockProv := &mockLLMProvider{response: "Hello from native provider!"}

	// Create a NativeProvider wrapping the mock.
	np, err := NewNativeProvider("test-native", "mock-model", mockProv)
	if err != nil {
		t.Fatalf("failed to create NativeProvider: %v", err)
	}

	if np.ProviderName() != "test-native" {
		t.Errorf("expected provider name 'test-native', got %q", np.ProviderName())
	}

	// Execute a session directly via the provider.
	result, err := np.ExecuteSession(
		SessionSpec{Prompt: "Say hello"},
		DefaultRuntimeConfig(),
		false, nil, nil,
	)
	if err != nil {
		t.Fatalf("ExecuteSession failed: %v", err)
	}
	if result.Output != "Hello from native provider!" {
		t.Errorf("expected 'Hello from native provider!', got %q", result.Output)
	}
	if result.Metadata.Model != "mock-model" {
		t.Errorf("expected model 'mock-model', got %q", result.Metadata.Model)
	}
}

func TestInterpreterWithNativeProvider(t *testing.T) {
	// Create a mock provider and NativeProvider.
	mockProv := &mockLLMProvider{response: "generated code here"}
	np, err := NewNativeProvider("my-agent", "mock-model", mockProv)
	if err != nil {
		t.Fatalf("failed to create NativeProvider: %v", err)
	}

	// Configure the interpreter to use "my-agent" as the default provider.
	cfg := DefaultRuntimeConfig()
	cfg.DefaultProvider = "my-agent"
	env := NewRuntimeEnvironment(&cfg)

	interp := NewInterpreter(env,
		WithNativeProviders(map[string]Provider{"my-agent": np}),
	)

	// Build a program with a named session statement (Name is used to save results).
	program := &ast.Program{
		Span: zeroSpan(),
		Statements: []ast.Node{
			&ast.SessionStatement{
				Span:   zeroSpan(),
				Prompt: &ast.StringLiteral{Span: zeroSpan(), Value: "Write some code"},
				Name:   &ast.Identifier{Span: zeroSpan(), Name: "output"},
			},
		},
	}

	result, err := interp.Execute(program)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got errors: %v", result.Errors)
	}

	// Verify the session result was saved to the variable.
	v, verr := env.Context.GetVariable("output")
	if verr != nil {
		t.Fatalf("variable 'output' not found: %v", verr)
	}
	sr, ok := IsSessionResult(v)
	if !ok {
		t.Fatalf("expected *SessionResult, got %T", v)
	}
	if sr.Output != "generated code here" {
		t.Errorf("expected 'generated code here', got %q", sr.Output)
	}
}

func TestResolveProviderFallback(t *testing.T) {
	// Native registry should take priority.
	mockProv := &mockLLMProvider{response: "native"}
	np, _ := NewNativeProvider("test", "m", mockProv)
	registry := map[string]Provider{"my-native": np}

	p, err := ResolveProvider("my-native", nil, registry)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if p.ProviderName() != "test" {
		t.Errorf("expected 'test', got %q", p.ProviderName())
	}

	// Unknown provider should error.
	_, err = ResolveProvider("nonexistent-xyz", nil, nil)
	if err == nil {
		t.Error("expected error for unknown provider")
	}

	// Built-in CLI preset should resolve.
	p, err = ResolveProvider("claude-code", nil, nil)
	if err != nil {
		t.Fatalf("expected claude-code to resolve, got: %v", err)
	}
	if p.ProviderName() != "claude-code" {
		t.Errorf("expected 'claude-code', got %q", p.ProviderName())
	}
}
