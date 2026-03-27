package parser

import (
	"testing"

	"github.com/ai-gateway/pi-go/whipflow/ast"
)

func TestParseSimpleSession(t *testing.T) {
	source := `session "hello world"`
	result := Parse(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", result.Errors)
	}
	if result.Program == nil {
		t.Fatal("expected program, got nil")
	}
	if len(result.Program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(result.Program.Statements))
	}

	sess, ok := result.Program.Statements[0].(*ast.SessionStatement)
	if !ok {
		t.Fatalf("expected SessionStatement, got %T", result.Program.Statements[0])
	}
	if sess.Prompt == nil {
		t.Fatal("expected prompt, got nil")
	}
}

func TestParseAgentDefinition(t *testing.T) {
	source := `agent coder:
  model: sonnet
  prompt: "You are a coding assistant"`

	result := Parse(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", result.Errors)
	}
	if len(result.Program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(result.Program.Statements))
	}

	agent, ok := result.Program.Statements[0].(*ast.AgentDefinition)
	if !ok {
		t.Fatalf("expected AgentDefinition, got %T", result.Program.Statements[0])
	}
	if agent.Name.Name != "coder" {
		t.Errorf("expected agent name 'coder', got %q", agent.Name.Name)
	}
}

func TestParseLetBinding(t *testing.T) {
	source := `let x = "hello"`
	result := Parse(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", result.Errors)
	}

	binding, ok := result.Program.Statements[0].(*ast.LetBinding)
	if !ok {
		t.Fatalf("expected LetBinding, got %T", result.Program.Statements[0])
	}
	if binding.Name.Name != "x" {
		t.Errorf("expected var name 'x', got %q", binding.Name.Name)
	}
}

func TestParseConstBinding(t *testing.T) {
	source := `const PI = 3.14`
	result := Parse(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", result.Errors)
	}

	binding, ok := result.Program.Statements[0].(*ast.ConstBinding)
	if !ok {
		t.Fatalf("expected ConstBinding, got %T", result.Program.Statements[0])
	}
	if binding.Name.Name != "PI" {
		t.Errorf("expected var name 'PI', got %q", binding.Name.Name)
	}
}

func TestParseRepeatBlock(t *testing.T) {
	source := `repeat 3:
  session "do something"`

	result := Parse(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", result.Errors)
	}

	repeat, ok := result.Program.Statements[0].(*ast.RepeatBlock)
	if !ok {
		t.Fatalf("expected RepeatBlock, got %T", result.Program.Statements[0])
	}
	if len(repeat.Body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestParseIfStatement(t *testing.T) {
	source := `if **is it ready**:
  session "ship it"
else:
  session "keep working"`

	result := Parse(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", result.Errors)
	}

	ifStmt, ok := result.Program.Statements[0].(*ast.IfStatement)
	if !ok {
		t.Fatalf("expected IfStatement, got %T", result.Program.Statements[0])
	}
	if len(ifStmt.ThenBody) == 0 {
		t.Error("expected non-empty then body")
	}
	if ifStmt.ElseBody == nil || len(ifStmt.ElseBody) == 0 {
		t.Error("expected non-empty else body")
	}
}

func TestParseTryBlock(t *testing.T) {
	source := `try:
  session "risky operation"
catch:
  session "handle error"`

	result := Parse(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", result.Errors)
	}

	tryBlock, ok := result.Program.Statements[0].(*ast.TryBlock)
	if !ok {
		t.Fatalf("expected TryBlock, got %T", result.Program.Statements[0])
	}
	if len(tryBlock.TryBody) == 0 {
		t.Error("expected non-empty try body")
	}
	if tryBlock.CatchBody == nil || len(tryBlock.CatchBody) == 0 {
		t.Error("expected non-empty catch body")
	}
}

func TestParseImport(t *testing.T) {
	source := `import "summarize" from "./skills/summarize.md"`
	result := Parse(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", result.Errors)
	}

	imp, ok := result.Program.Statements[0].(*ast.ImportStatement)
	if !ok {
		t.Fatalf("expected ImportStatement, got %T", result.Program.Statements[0])
	}
	_ = imp
}

func TestParseArray(t *testing.T) {
	source := `let items = ["a", "b", "c"]`
	result := Parse(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", result.Errors)
	}

	binding, ok := result.Program.Statements[0].(*ast.LetBinding)
	if !ok {
		t.Fatalf("expected LetBinding, got %T", result.Program.Statements[0])
	}
	arr, ok := binding.Value.(*ast.ArrayExpression)
	if !ok {
		t.Fatalf("expected ArrayExpression, got %T", binding.Value)
	}
	if len(arr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestParseThrow(t *testing.T) {
	source := `throw "something went wrong"`
	result := Parse(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", result.Errors)
	}

	_, ok := result.Program.Statements[0].(*ast.ThrowStatement)
	if !ok {
		t.Fatalf("expected ThrowStatement, got %T", result.Program.Statements[0])
	}
}

func TestParseReturn(t *testing.T) {
	source := `return "done"`
	result := Parse(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", result.Errors)
	}

	_, ok := result.Program.Statements[0].(*ast.ReturnStatement)
	if !ok {
		t.Fatalf("expected ReturnStatement, got %T", result.Program.Statements[0])
	}
}
