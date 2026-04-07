package runtime

import (
	"testing"

	"github.com/ai-gateway/clawfirm/whipflow/ast"
	"github.com/ai-gateway/clawfirm/whipflow/token"
)

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

func TestResolveProviderFallback(t *testing.T) {
	// Unknown provider should error.
	_, err := ResolveProvider("nonexistent-xyz", nil, nil)
	if err == nil {
		t.Error("expected error for unknown provider")
	}

	// Built-in CLI preset should resolve.
	p, err := ResolveProvider("claude-code", nil, nil)
	if err != nil {
		t.Fatalf("expected claude-code to resolve, got: %v", err)
	}
	if p.ProviderName() != "claude-code" {
		t.Errorf("expected 'claude-code', got %q", p.ProviderName())
	}
}
