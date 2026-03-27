// Package token defines the token types and related structures for the
// WhipFlow language lexer and parser.
package token

// TokenType represents the type of a lexical token.
type TokenType int

const (
	// Literals
	STRING     TokenType = iota // string literal
	NUMBER                      // numeric literal
	IDENTIFIER                  // identifier

	// Comments
	COMMENT // comment

	// Keywords
	IMPORT
	FROM
	AGENT
	SESSION
	MODEL
	PROMPT
	BLOCK
	DO
	PARALLEL
	CHOICE
	LET
	CONST
	LOOP
	UNTIL
	WHILE
	REPEAT
	FOR
	IN
	AS
	IF
	ELIF
	ELSE
	OPTION
	TRY
	CATCH
	FINALLY
	THROW
	RETURN
	RETRY
	BACKOFF
	MAP
	FILTER
	REDUCE
	PMAP
	SKILLS
	TOOLS
	PERMISSIONS
	CONTEXT
	ASK
	RUN
	SKILL

	// Operators and punctuation
	ARROW    // ->
	PIPE     // |
	EQUALS   // =
	COLON    // :
	COMMA    // ,
	LPAREN   // (
	RPAREN   // )
	LBRACKET // [
	RBRACKET // ]
	LBRACE   // {
	RBRACE   // }

	// Discretion markers
	DISCRETION           // **...**
	MULTILINE_DISCRETION // ***...***

	// Whitespace-significant tokens
	NEWLINE // newline
	INDENT  // indent increase
	DEDENT  // indent decrease

	// Special
	EOF   // end of file
	ERROR // lexer error
)

// tokenNames maps each TokenType to its string representation.
var tokenNames = [...]string{
	STRING:     "STRING",
	NUMBER:     "NUMBER",
	IDENTIFIER: "IDENTIFIER",

	COMMENT: "COMMENT",

	IMPORT:      "IMPORT",
	FROM:        "FROM",
	AGENT:       "AGENT",
	SESSION:     "SESSION",
	MODEL:       "MODEL",
	PROMPT:      "PROMPT",
	BLOCK:       "BLOCK",
	DO:          "DO",
	PARALLEL:    "PARALLEL",
	CHOICE:      "CHOICE",
	LET:         "LET",
	CONST:       "CONST",
	LOOP:        "LOOP",
	UNTIL:       "UNTIL",
	WHILE:       "WHILE",
	REPEAT:      "REPEAT",
	FOR:         "FOR",
	IN:          "IN",
	AS:          "AS",
	IF:          "IF",
	ELIF:        "ELIF",
	ELSE:        "ELSE",
	OPTION:      "OPTION",
	TRY:         "TRY",
	CATCH:       "CATCH",
	FINALLY:     "FINALLY",
	THROW:       "THROW",
	RETURN:      "RETURN",
	RETRY:       "RETRY",
	BACKOFF:     "BACKOFF",
	MAP:         "MAP",
	FILTER:      "FILTER",
	REDUCE:      "REDUCE",
	PMAP:        "PMAP",
	SKILLS:      "SKILLS",
	TOOLS:       "TOOLS",
	PERMISSIONS: "PERMISSIONS",
	CONTEXT:     "CONTEXT",
	ASK:         "ASK",
	RUN:         "RUN",
	SKILL:       "SKILL",

	ARROW:    "ARROW",
	PIPE:     "PIPE",
	EQUALS:   "EQUALS",
	COLON:    "COLON",
	COMMA:    "COMMA",
	LPAREN:   "LPAREN",
	RPAREN:   "RPAREN",
	LBRACKET: "LBRACKET",
	RBRACKET: "RBRACKET",
	LBRACE:   "LBRACE",
	RBRACE:   "RBRACE",

	DISCRETION:           "DISCRETION",
	MULTILINE_DISCRETION: "MULTILINE_DISCRETION",

	NEWLINE: "NEWLINE",
	INDENT:  "INDENT",
	DEDENT:  "DEDENT",

	EOF:   "EOF",
	ERROR: "ERROR",
}

// String returns the human-readable name of the token type.
func (tt TokenType) String() string {
	if int(tt) >= 0 && int(tt) < len(tokenNames) {
		return tokenNames[tt]
	}
	return "UNKNOWN"
}

// SourceLocation represents a position in source code.
type SourceLocation struct {
	Line   int // 1-based line number
	Column int // 1-based column number
	Offset int // 0-based byte offset from the start of the source
}

// SourceSpan represents a range in source code from Start to End.
type SourceSpan struct {
	Start SourceLocation
	End   SourceLocation
}

// Token represents a single lexical token produced by the lexer.
type Token struct {
	Type       TokenType            // the type of the token
	Value      string               // the textual value of the token
	Span       SourceSpan           // the source location span of the token
	IsTrivia   bool                 // whether this token is trivia (whitespace, comments)
	StringMeta *StringTokenMetadata // metadata for string tokens; nil for non-string tokens
}

// StringTokenMetadata holds additional information about string literal tokens,
// including their raw representation, quoting style, escape sequences, and
// interpolations.
type StringTokenMetadata struct {
	Raw             string               // the original raw string including delimiters
	IsTripleQuoted  bool                 // whether the string uses triple-quote (""") delimiters
	EscapeSequences []EscapeSequenceInfo // escape sequences found in the string
	Interpolations  []InterpolationInfo  // interpolation expressions found in the string
}

// EscapeSequenceInfo describes a single escape sequence within a string literal.
type EscapeSequenceInfo struct {
	Type     string // the kind of escape: "standard", "unicode", or "invalid"
	Sequence string // the raw escape sequence (e.g. `\n`, `\u0041`)
	Resolved string // the resolved character(s) the escape represents
	Offset   int    // byte offset of the escape sequence within the string value
}

// InterpolationInfo describes a single interpolation expression within a string
// literal.
type InterpolationInfo struct {
	VarName string // the variable name referenced in the interpolation
	Offset  int    // byte offset of the interpolation within the string value
	Raw     string // the raw interpolation text including delimiters (e.g. `${name}`)
}

// Keywords maps lowercase keyword strings to their corresponding TokenType.
var Keywords = map[string]TokenType{
	"import":      IMPORT,
	"from":        FROM,
	"agent":       AGENT,
	"session":     SESSION,
	"model":       MODEL,
	"prompt":      PROMPT,
	"block":       BLOCK,
	"do":          DO,
	"parallel":    PARALLEL,
	"choice":      CHOICE,
	"let":         LET,
	"const":       CONST,
	"loop":        LOOP,
	"until":       UNTIL,
	"while":       WHILE,
	"repeat":      REPEAT,
	"for":         FOR,
	"in":          IN,
	"as":          AS,
	"if":          IF,
	"elif":        ELIF,
	"else":        ELSE,
	"option":      OPTION,
	"try":         TRY,
	"catch":       CATCH,
	"finally":     FINALLY,
	"throw":       THROW,
	"return":      RETURN,
	"retry":       RETRY,
	"backoff":     BACKOFF,
	"map":         MAP,
	"filter":      FILTER,
	"reduce":      REDUCE,
	"pmap":        PMAP,
	"skills":      SKILLS,
	"tools":       TOOLS,
	"permissions": PERMISSIONS,
	"context":     CONTEXT,
	"ask":         ASK,
	"run":         RUN,
	"skill":       SKILL,
}

// IsKeyword reports whether the given TokenType is a keyword token.
func IsKeyword(tt TokenType) bool {
	return tt >= IMPORT && tt <= SKILL
}
