package lexer

import (
	"testing"

	"github.com/ai-gateway/pi-go/whipflow/token"
)

func TestTokenizeBasicKeywords(t *testing.T) {
	source := "agent session model prompt"
	result := Tokenize(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	expected := []token.TokenType{token.AGENT, token.SESSION, token.MODEL, token.PROMPT, token.EOF}
	got := tokenTypes(result.Tokens)

	if len(got) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(got), got)
	}
	for i, e := range expected {
		if got[i] != e {
			t.Errorf("token[%d]: expected %s, got %s", i, e, got[i])
		}
	}
}

func TestTokenizeString(t *testing.T) {
	source := `session "hello world"`
	result := Tokenize(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	types := tokenTypes(result.Tokens)
	if types[0] != token.SESSION || types[1] != token.STRING {
		t.Errorf("expected SESSION STRING, got %v", types)
	}
	if result.Tokens[1].Value != "hello world" {
		t.Errorf("expected string value 'hello world', got %q", result.Tokens[1].Value)
	}
}

func TestTokenizeStringEscapes(t *testing.T) {
	source := `"hello\nworld\t!"`
	result := Tokenize(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	if result.Tokens[0].Value != "hello\nworld\t!" {
		t.Errorf("expected escaped string, got %q", result.Tokens[0].Value)
	}

	meta := result.Tokens[0].StringMeta
	if meta == nil {
		t.Fatal("expected string metadata")
	}
	if len(meta.EscapeSequences) != 2 {
		t.Errorf("expected 2 escape sequences, got %d", len(meta.EscapeSequences))
	}
}

func TestTokenizeStringInterpolation(t *testing.T) {
	source := `"hello {name}, welcome"`
	result := Tokenize(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	meta := result.Tokens[0].StringMeta
	if meta == nil {
		t.Fatal("expected string metadata")
	}
	if len(meta.Interpolations) != 1 {
		t.Fatalf("expected 1 interpolation, got %d", len(meta.Interpolations))
	}
	if meta.Interpolations[0].VarName != "name" {
		t.Errorf("expected interpolation varname 'name', got %q", meta.Interpolations[0].VarName)
	}
}

func TestTokenizeTripleQuotedString(t *testing.T) {
	source := `"""hello
world"""`
	result := Tokenize(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	if result.Tokens[0].Value != "hello\nworld" {
		t.Errorf("expected 'hello\\nworld', got %q", result.Tokens[0].Value)
	}
	if !result.Tokens[0].StringMeta.IsTripleQuoted {
		t.Error("expected IsTripleQuoted=true")
	}
}

func TestTokenizeIndentation(t *testing.T) {
	source := `agent coder:
  model: sonnet
  prompt: "hello"
session "do it"`

	result := Tokenize(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	types := tokenTypes(result.Tokens)

	// Should contain INDENT and DEDENT.
	hasIndent := false
	hasDedent := false
	for _, tt := range types {
		if tt == token.INDENT {
			hasIndent = true
		}
		if tt == token.DEDENT {
			hasDedent = true
		}
	}
	if !hasIndent {
		t.Error("expected INDENT token")
	}
	if !hasDedent {
		t.Error("expected DEDENT token")
	}
}

func TestTokenizeNumber(t *testing.T) {
	source := "repeat 42"
	result := Tokenize(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	if result.Tokens[1].Type != token.NUMBER || result.Tokens[1].Value != "42" {
		t.Errorf("expected NUMBER '42', got %s %q", result.Tokens[1].Type, result.Tokens[1].Value)
	}
}

func TestTokenizeDecimalNumber(t *testing.T) {
	source := "3.14"
	result := Tokenize(source)

	if result.Tokens[0].Type != token.NUMBER || result.Tokens[0].Value != "3.14" {
		t.Errorf("expected NUMBER '3.14', got %s %q", result.Tokens[0].Type, result.Tokens[0].Value)
	}
}

func TestTokenizeArrow(t *testing.T) {
	source := "a -> b"
	result := Tokenize(source)

	types := tokenTypes(result.Tokens)
	if types[1] != token.ARROW {
		t.Errorf("expected ARROW, got %s", types[1])
	}
}

func TestTokenizeDiscretion(t *testing.T) {
	source := "**is this good enough?**"
	result := Tokenize(source)

	if result.Tokens[0].Type != token.DISCRETION {
		t.Errorf("expected DISCRETION, got %s", result.Tokens[0].Type)
	}
}

func TestTokenizeMultilineDiscretion(t *testing.T) {
	source := "***is this\ngood enough?***"
	result := Tokenize(source)

	if result.Tokens[0].Type != token.MULTILINE_DISCRETION {
		t.Errorf("expected MULTILINE_DISCRETION, got %s", result.Tokens[0].Type)
	}
}

func TestTokenizeComment(t *testing.T) {
	source := "# this is a comment\nagent coder:"
	result := Tokenize(source, Options{IncludeComments: true})

	hasComment := false
	for _, tok := range result.Tokens {
		if tok.Type == token.COMMENT {
			hasComment = true
		}
	}
	if !hasComment {
		t.Error("expected COMMENT token")
	}
}

func TestTokenizeOperators(t *testing.T) {
	source := ": , ( ) [ ] { } | ="
	result := Tokenize(source)

	expected := []token.TokenType{
		token.COLON, token.COMMA, token.LPAREN, token.RPAREN,
		token.LBRACKET, token.RBRACKET, token.LBRACE, token.RBRACE,
		token.PIPE, token.EQUALS, token.EOF,
	}

	types := tokenTypes(result.Tokens)
	for i, e := range expected {
		if i >= len(types) {
			t.Fatalf("missing token at index %d", i)
		}
		if types[i] != e {
			t.Errorf("token[%d]: expected %s, got %s", i, e, types[i])
		}
	}
}

func TestTokenizeWithoutComments(t *testing.T) {
	source := "# comment\nagent coder:"
	result := TokenizeWithoutComments(source)

	for _, tok := range result.Tokens {
		if tok.Type == token.COMMENT {
			t.Error("expected no COMMENT tokens")
		}
	}
}

func TestUnterminatedString(t *testing.T) {
	source := `"unterminated`
	result := Tokenize(source)

	if len(result.Errors) == 0 {
		t.Error("expected error for unterminated string")
	}
}

func tokenTypes(tokens []token.Token) []token.TokenType {
	types := make([]token.TokenType, len(tokens))
	for i, t := range tokens {
		types[i] = t.Type
	}
	return types
}
