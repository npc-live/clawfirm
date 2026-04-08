package builtin

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestAskUser_MissingQuestion(t *testing.T) {
	a := &AskUser{}
	_, err := a.Execute(context.Background(), "1", map[string]any{}, nopUpdate)
	if err == nil || !strings.Contains(err.Error(), "question is required") {
		t.Errorf("expected question-required error, got %v", err)
	}
}

func TestAskUser_NilCallback(t *testing.T) {
	a := &AskUser{}
	res, err := a.Execute(context.Background(), "1", map[string]any{"question": "Pick one?"}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "not available") {
		t.Errorf("expected not-available message, got %q", text)
	}
}

func TestAskUser_WithCallback(t *testing.T) {
	a := &AskUser{
		OnQuestion: func(ctx context.Context, q AskUserQuestion) (AskUserAnswer, error) {
			if q.Question != "Which color?" {
				t.Errorf("question: got %q", q.Question)
			}
			if len(q.Options) != 2 {
				t.Errorf("options: got %d want 2", len(q.Options))
			}
			return AskUserAnswer{Selected: []string{"Blue"}}, nil
		},
	}

	params := map[string]any{
		"question": "Which color?",
		"options": []any{
			map[string]any{"label": "Red", "description": "A warm color"},
			map[string]any{"label": "Blue", "description": "A cool color"},
		},
	}

	res, err := a.Execute(context.Background(), "1", params, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "Blue") {
		t.Errorf("expected selected option in output, got %q", text)
	}
}

func TestAskUser_CallbackError(t *testing.T) {
	a := &AskUser{
		OnQuestion: func(ctx context.Context, q AskUserQuestion) (AskUserAnswer, error) {
			return AskUserAnswer{}, fmt.Errorf("user cancelled")
		},
	}
	res, err := a.Execute(context.Background(), "1", map[string]any{"question": "Yes?"}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "user cancelled") {
		t.Errorf("expected error in output, got %q", text)
	}
}

func TestAskUser_FreeText(t *testing.T) {
	a := &AskUser{
		OnQuestion: func(ctx context.Context, q AskUserQuestion) (AskUserAnswer, error) {
			return AskUserAnswer{Text: "I want purple"}, nil
		},
	}
	res, err := a.Execute(context.Background(), "1", map[string]any{"question": "Pick?"}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "I want purple") {
		t.Errorf("expected free text in output, got %q", text)
	}
}

func TestAskUser_MultiSelect(t *testing.T) {
	a := &AskUser{
		OnQuestion: func(ctx context.Context, q AskUserQuestion) (AskUserAnswer, error) {
			if !q.MultiSelect {
				t.Error("expected MultiSelect=true")
			}
			return AskUserAnswer{Selected: []string{"A", "C"}}, nil
		},
	}
	params := map[string]any{
		"question":     "Pick multiple?",
		"multi_select": true,
		"options": []any{
			map[string]any{"label": "A"},
			map[string]any{"label": "B"},
			map[string]any{"label": "C"},
		},
	}
	res, err := a.Execute(context.Background(), "1", params, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "A") || !strings.Contains(text, "C") {
		t.Errorf("expected selected options in output, got %q", text)
	}
}

func TestAskUser_Schema(t *testing.T) {
	a := &AskUser{}
	s := a.Schema()
	props := s["properties"].(map[string]any)
	if _, ok := props["question"]; !ok {
		t.Error("schema missing question property")
	}
	if _, ok := props["options"]; !ok {
		t.Error("schema missing options property")
	}
}

func TestAskUser_EmptyAnswer(t *testing.T) {
	a := &AskUser{
		OnQuestion: func(ctx context.Context, q AskUserQuestion) (AskUserAnswer, error) {
			return AskUserAnswer{}, nil
		},
	}
	res, err := a.Execute(context.Background(), "1", map[string]any{"question": "Hi?"}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "no answer") {
		t.Errorf("expected no-answer message, got %q", text)
	}
}

