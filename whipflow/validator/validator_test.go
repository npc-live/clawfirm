package validator

import (
	"testing"

	"github.com/ai-gateway/pi-go/whipflow/parser"
)

func TestValidateSimpleProgram(t *testing.T) {
	source := `session "hello world"`
	result := parser.Parse(source)
	if len(result.Errors) > 0 {
		t.Fatalf("parse errors: %v", result.Errors)
	}

	vResult := Validate(result.Program)
	if !vResult.Valid {
		t.Errorf("expected valid, got errors: %v", vResult.Errors)
	}
}

func TestValidateUndefinedAgent(t *testing.T) {
	source := `session: unknown-agent
  prompt: "hello"`
	result := parser.Parse(source)
	if len(result.Errors) > 0 {
		t.Fatalf("parse errors: %v", result.Errors)
	}

	vResult := Validate(result.Program)
	if vResult.Valid {
		t.Error("expected validation errors for undefined agent")
	}
}

func TestValidateAgentDefinitionAndUse(t *testing.T) {
	source := `agent coder:
  model: sonnet
  prompt: "You are a coder"

session: coder
  prompt: "Write hello world"`
	result := parser.Parse(source)
	if len(result.Errors) > 0 {
		t.Fatalf("parse errors: %v", result.Errors)
	}

	vResult := Validate(result.Program)
	if !vResult.Valid {
		t.Errorf("expected valid, got errors: %v", vResult.Errors)
	}
}

func TestValidateInvalidModel(t *testing.T) {
	source := `agent coder:
  model: gpt4`
	result := parser.Parse(source)
	if len(result.Errors) > 0 {
		t.Fatalf("parse errors: %v", result.Errors)
	}

	vResult := Validate(result.Program)
	hasModelError := false
	for _, e := range vResult.Errors {
		if e.Severity == "error" {
			hasModelError = true
		}
	}
	if !hasModelError {
		t.Error("expected error for invalid model 'gpt4'")
	}
}

func TestValidateLetConst(t *testing.T) {
	source := `let x = "hello"
const y = "world"`
	result := parser.Parse(source)
	if len(result.Errors) > 0 {
		t.Fatalf("parse errors: %v", result.Errors)
	}

	vResult := Validate(result.Program)
	if !vResult.Valid {
		t.Errorf("expected valid, got errors: %v", vResult.Errors)
	}
}
