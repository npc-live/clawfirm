// Package lexer implements the WhipFlow tokenizer.
//
// It handles indentation-based structure, string interpolation,
// escape sequences, discretion markers, and comments.
package lexer

import (
	"fmt"
	"unicode/utf8"

	"github.com/ai-gateway/clawfirm/whipflow/token"
)

// Options configures lexer behavior.
type Options struct {
	IncludeComments bool
}

// Result holds the output of tokenization.
type Result struct {
	Tokens []token.Token
	Errors []Error
}

// Error represents a lexer error.
type Error struct {
	Message  string
	Span     token.SourceSpan
	Severity string // "error" or "warning"
}

func (e Error) Error() string { return e.Message }

// Lexer tokenizes WhipFlow source code.
type Lexer struct {
	source      string
	pos         int
	line        int
	column      int
	tokens      []token.Token
	errors      []Error
	opts        Options
	indentStack []int
}

// New creates a new Lexer for the given source.
func New(source string, opts ...Options) *Lexer {
	o := Options{IncludeComments: true}
	if len(opts) > 0 {
		o = opts[0]
	}
	return &Lexer{
		source:      source,
		line:        1,
		column:      1,
		opts:        o,
		indentStack: []int{0},
	}
}

// Tokenize scans the source and returns all tokens.
func (l *Lexer) Tokenize() Result {
	l.tokens = nil
	l.errors = nil
	l.pos = 0
	l.line = 1
	l.column = 1
	l.indentStack = []int{0}

	for !l.isAtEnd() {
		l.scanToken()
	}

	// Emit remaining DEDENTs.
	for len(l.indentStack) > 1 {
		l.indentStack = l.indentStack[:len(l.indentStack)-1]
		l.addToken(token.DEDENT, "")
	}

	l.addToken(token.EOF, "")

	return Result{Tokens: l.tokens, Errors: l.errors}
}

func (l *Lexer) scanToken() {
	// Handle start of line — check indentation.
	if l.column == 1 {
		l.handleIndentation()
		if l.isAtEnd() {
			return
		}
	}

	c := l.peek()

	// Skip horizontal whitespace (not at start of line).
	if c == ' ' || c == '\t' {
		l.advance()
		return
	}

	// Newline.
	if c == '\n' {
		l.addToken(token.NEWLINE, "\n")
		l.advance()
		l.line++
		l.column = 1
		return
	}

	// Carriage return.
	if c == '\r' {
		l.advance()
		if l.peek() == '\n' {
			l.advance()
		}
		l.addToken(token.NEWLINE, "\n")
		l.line++
		l.column = 1
		return
	}

	// Comment.
	if c == '#' {
		l.scanComment()
		return
	}

	// String literal.
	if c == '"' {
		l.scanString()
		return
	}

	// Number literal.
	if isDigit(c) {
		l.scanNumber()
		return
	}

	// Identifier or keyword.
	if isAlpha(c) {
		l.scanIdentifier()
		return
	}

	// Operators and punctuation.
	switch c {
	case ':':
		l.addTokenAndAdvance(token.COLON, ":")
	case ',':
		l.addTokenAndAdvance(token.COMMA, ",")
	case '(':
		l.addTokenAndAdvance(token.LPAREN, "(")
	case ')':
		l.addTokenAndAdvance(token.RPAREN, ")")
	case '[':
		l.addTokenAndAdvance(token.LBRACKET, "[")
	case ']':
		l.addTokenAndAdvance(token.RBRACKET, "]")
	case '{':
		l.addTokenAndAdvance(token.LBRACE, "{")
	case '}':
		l.addTokenAndAdvance(token.RBRACE, "}")
	case '|':
		l.addTokenAndAdvance(token.PIPE, "|")
	case '=':
		l.addTokenAndAdvance(token.EQUALS, "=")
	case '-':
		if l.peekNext() == '>' {
			start := l.currentLocation()
			l.advance()
			l.advance()
			l.addTokenAt(token.ARROW, "->", start)
		} else {
			l.addError(fmt.Sprintf("Unexpected character: %c", c))
			l.advance()
		}
	case '*':
		l.scanDiscretion()
	default:
		l.addError(fmt.Sprintf("Unexpected character: %c", c))
		l.advance()
	}
}

// handleIndentation processes indentation at the start of a line.
func (l *Lexer) handleIndentation() {
	indent := 0

	for !l.isAtEnd() && (l.peek() == ' ' || l.peek() == '\t') {
		if l.peek() == ' ' {
			indent++
		} else {
			indent = (indent/4)*4 + 4
		}
		l.advance()
	}

	// Skip empty lines.
	if l.isAtEnd() || l.peek() == '\n' || l.peek() == '\r' {
		return
	}

	currentIndent := l.indentStack[len(l.indentStack)-1]

	if indent > currentIndent {
		l.indentStack = append(l.indentStack, indent)
		l.addToken(token.INDENT, "")
	} else if indent < currentIndent {
		for len(l.indentStack) > 1 && l.indentStack[len(l.indentStack)-1] > indent {
			// Peek: would popping leave us below current indent?
			nextLevel := l.indentStack[len(l.indentStack)-2]
			if nextLevel < indent {
				// Snap current top to indent — treat as same level.
				l.indentStack[len(l.indentStack)-1] = indent
				break
			}
			l.indentStack = l.indentStack[:len(l.indentStack)-1]
			l.addToken(token.DEDENT, "")
		}

		// If still no exact match, snap to current indent.
		if l.indentStack[len(l.indentStack)-1] != indent {
			l.indentStack[len(l.indentStack)-1] = indent
		}
	}
}

// scanComment scans a # comment to end of line.
func (l *Lexer) scanComment() {
	start := l.currentLocation()
	value := ""

	for !l.isAtEnd() && l.peek() != '\n' && l.peek() != '\r' {
		value += string(l.peek())
		l.advance()
	}

	if l.opts.IncludeComments {
		l.addTokenAtTrivia(token.COMMENT, value, start, true)
	}
}

// scanString scans a string literal.
func (l *Lexer) scanString() {
	start := l.currentLocation()
	rawStart := l.pos
	l.advance() // consume opening quote

	// Check for triple-quoted string.
	if l.peek() == '"' && l.peekNext() == '"' {
		l.advance()
		l.advance()
		l.scanTripleQuotedString(start, rawStart)
		return
	}

	value := ""
	var escapeSequences []token.EscapeSequenceInfo
	var interpolations []token.InterpolationInfo
	valueOffset := 0

	for !l.isAtEnd() && l.peek() != '"' {
		if l.peek() == '\n' || l.peek() == '\r' {
			l.addError("Unterminated string literal")
			return
		}

		// Check for interpolation {varname}.
		if l.peek() == '{' {
			interpStart := l.pos
			interpValueOffset := valueOffset
			l.advance()

			varName := ""
			for !l.isAtEnd() && l.peek() != '}' && l.peek() != '"' && l.peek() != '\n' {
				c := l.peek()
				if isAlphaNumericOrHyphen(c) || (len(varName) == 0 && isAlpha(c)) {
					varName += string(c)
					l.advance()
				} else {
					break
				}
			}

			if l.peek() == '}' && len(varName) > 0 {
				l.advance()
				rawInterp := l.source[interpStart:l.pos]
				interpolations = append(interpolations, token.InterpolationInfo{
					VarName: varName,
					Offset:  interpValueOffset,
					Raw:     rawInterp,
				})
				value += rawInterp
				valueOffset += len(rawInterp)
			} else if l.peek() == '}' {
				l.advance()
				value += "{}"
				valueOffset += 2
			} else {
				value += "{" + varName
				valueOffset += 1 + len(varName)
			}
			continue
		}

		if l.peek() == '\\' {
			escStart := l.currentLocation()
			_ = escStart
			escRawOffset := l.pos - rawStart
			l.advance()

			if l.isAtEnd() {
				l.addError("Unterminated string literal")
				return
			}

			escaped := l.peek()
			var escInfo *token.EscapeSequenceInfo

			switch escaped {
			case 'n':
				value += "\n"
				valueOffset++
				escInfo = &token.EscapeSequenceInfo{Type: "standard", Sequence: `\n`, Resolved: "\n", Offset: escRawOffset}
				l.advance()
			case 't':
				value += "\t"
				valueOffset++
				escInfo = &token.EscapeSequenceInfo{Type: "standard", Sequence: `\t`, Resolved: "\t", Offset: escRawOffset}
				l.advance()
			case 'r':
				value += "\r"
				valueOffset++
				escInfo = &token.EscapeSequenceInfo{Type: "standard", Sequence: `\r`, Resolved: "\r", Offset: escRawOffset}
				l.advance()
			case '\\':
				value += "\\"
				valueOffset++
				escInfo = &token.EscapeSequenceInfo{Type: "standard", Sequence: `\\`, Resolved: "\\", Offset: escRawOffset}
				l.advance()
			case '"':
				value += "\""
				valueOffset++
				escInfo = &token.EscapeSequenceInfo{Type: "standard", Sequence: `\"`, Resolved: "\"", Offset: escRawOffset}
				l.advance()
			case '#':
				value += "#"
				valueOffset++
				escInfo = &token.EscapeSequenceInfo{Type: "standard", Sequence: `\#`, Resolved: "#", Offset: escRawOffset}
				l.advance()
			case '0':
				value += "\x00"
				valueOffset++
				escInfo = &token.EscapeSequenceInfo{Type: "standard", Sequence: `\0`, Resolved: "\x00", Offset: escRawOffset}
				l.advance()
			case '{':
				value += "{"
				valueOffset++
				escInfo = &token.EscapeSequenceInfo{Type: "standard", Sequence: `\{`, Resolved: "{", Offset: escRawOffset}
				l.advance()
			case '}':
				value += "}"
				valueOffset++
				escInfo = &token.EscapeSequenceInfo{Type: "standard", Sequence: `\}`, Resolved: "}", Offset: escRawOffset}
				l.advance()
			case 'u':
				l.advance() // consume 'u'
				hex, char, ok := l.scanUnicodeEscape()
				if ok {
					value += char
					valueOffset++
					escInfo = &token.EscapeSequenceInfo{Type: "unicode", Sequence: `\u` + hex, Resolved: char, Offset: escRawOffset}
				} else {
					value += "u"
					valueOffset++
					escInfo = &token.EscapeSequenceInfo{Type: "invalid", Sequence: `\u`, Resolved: "u", Offset: escRawOffset}
				}
			default:
				l.addWarning(fmt.Sprintf(`Unrecognized escape sequence: \%c`, escaped))
				value += string(escaped)
				valueOffset++
				escInfo = &token.EscapeSequenceInfo{Type: "invalid", Sequence: fmt.Sprintf(`\%c`, escaped), Resolved: string(escaped), Offset: escRawOffset}
				l.advance()
			}

			if escInfo != nil {
				escapeSequences = append(escapeSequences, *escInfo)
			}
		} else {
			// Decode the full UTF-8 rune so multi-byte characters (e.g. CJK)
			// are not corrupted by byte-level processing.
			r, size := utf8.DecodeRuneInString(l.source[l.pos:])
			value += string(r)
			valueOffset += size
			for i := 0; i < size; i++ {
				l.advance()
			}
		}
	}

	if l.isAtEnd() {
		l.addError("Unterminated string literal")
		return
	}

	l.advance() // consume closing quote

	raw := l.source[rawStart:l.pos]
	meta := &token.StringTokenMetadata{
		Raw:              raw,
		IsTripleQuoted:   false,
		EscapeSequences:  escapeSequences,
		Interpolations:   interpolations,
	}

	l.addStringToken(value, start, meta)
}

// scanTripleQuotedString scans a """...""" string.
func (l *Lexer) scanTripleQuotedString(start token.SourceLocation, rawStart int) {
	value := ""
	var interpolations []token.InterpolationInfo
	valueOffset := 0

	for !l.isAtEnd() {
		if l.peek() == '"' && l.peekAt(1) == '"' && l.peekAt(2) == '"' {
			l.advance()
			l.advance()
			l.advance()

			raw := l.source[rawStart:l.pos]
			meta := &token.StringTokenMetadata{
				Raw:            raw,
				IsTripleQuoted: true,
				Interpolations: interpolations,
			}
			l.addStringToken(value, start, meta)
			return
		}

		// Interpolation.
		if l.peek() == '{' {
			interpStart := l.pos
			interpValueOffset := valueOffset
			l.advance()

			varName := ""
			for !l.isAtEnd() && l.peek() != '}' && l.peek() != '"' && l.peek() != '\n' && l.peek() != '\r' {
				c := l.peek()
				if isAlphaNumericOrHyphen(c) || (len(varName) == 0 && isAlpha(c)) {
					varName += string(c)
					l.advance()
				} else {
					break
				}
			}

			if l.peek() == '}' && len(varName) > 0 {
				l.advance()
				rawInterp := l.source[interpStart:l.pos]
				interpolations = append(interpolations, token.InterpolationInfo{
					VarName: varName,
					Offset:  interpValueOffset,
					Raw:     rawInterp,
				})
				value += rawInterp
				valueOffset += len(rawInterp)
			} else if l.peek() == '}' {
				l.advance()
				value += "{}"
				valueOffset += 2
			} else {
				value += "{" + varName
				valueOffset += 1 + len(varName)
			}
			continue
		}

		if l.peek() == '\n' {
			value += "\n"
			valueOffset++
			l.advance()
			l.line++
			l.column = 1
		} else if l.peek() == '\r' {
			l.advance()
			if l.peek() == '\n' {
				l.advance()
			}
			value += "\n"
			valueOffset++
			l.line++
			l.column = 1
		} else {
			r, size := utf8.DecodeRuneInString(l.source[l.pos:])
			value += string(r)
			valueOffset += size
			for i := 0; i < size; i++ {
				l.advance()
			}
		}
	}

	l.addError("Unterminated triple-quoted string")
}

// scanUnicodeEscape scans 4 hex digits after \u.
func (l *Lexer) scanUnicodeEscape() (hex string, char string, ok bool) {
	hexDigits := ""
	for i := 0; i < 4; i++ {
		if l.isAtEnd() || l.peek() == '"' || l.peek() == '\n' || l.peek() == '\r' {
			l.addError(fmt.Sprintf("Invalid unicode escape: expected 4 hex digits, got %d", i))
			return hexDigits, "", false
		}
		c := l.peek()
		if !isHexDigit(c) {
			l.addError(fmt.Sprintf("Invalid unicode escape: '%c' is not a valid hex digit", c))
			return hexDigits, "", false
		}
		hexDigits += string(c)
		l.advance()
	}

	var codePoint int
	fmt.Sscanf(hexDigits, "%x", &codePoint)
	return hexDigits, string(rune(codePoint)), true
}

// scanNumber scans a number literal.
func (l *Lexer) scanNumber() {
	start := l.currentLocation()
	value := ""

	for !l.isAtEnd() && isDigit(l.peek()) {
		value += string(l.peek())
		l.advance()
	}

	// Handle decimal.
	if l.peek() == '.' && isDigit(l.peekNext()) {
		value += "."
		l.advance()
		for !l.isAtEnd() && isDigit(l.peek()) {
			value += string(l.peek())
			l.advance()
		}
	}

	l.addTokenAt(token.NUMBER, value, start)
}

// scanIdentifier scans an identifier or keyword.
func (l *Lexer) scanIdentifier() {
	start := l.currentLocation()
	value := ""

	for !l.isAtEnd() && isAlphaNumericOrHyphen(l.peek()) {
		// Check for arrow operator.
		if l.peek() == '-' && l.peekNext() == '>' {
			break
		}
		value += string(l.peek())
		l.advance()
	}

	tokenType := token.IDENTIFIER
	if kw, ok := token.Keywords[value]; ok {
		tokenType = kw
	}
	l.addTokenAt(tokenType, value, start)
}

// scanDiscretion scans **...** or ***...***.
func (l *Lexer) scanDiscretion() {
	start := l.currentLocation()

	if l.peek() != '*' || l.peekNext() != '*' {
		l.addError("Unexpected character: *")
		l.advance()
		return
	}

	if l.peekAt(2) == '*' {
		l.scanMultilineDiscretion(start)
	} else {
		l.scanInlineDiscretion(start)
	}
}

// scanInlineDiscretion scans **...**
func (l *Lexer) scanInlineDiscretion(start token.SourceLocation) {
	l.advance() // first *
	l.advance() // second *

	value := "**"

	for !l.isAtEnd() {
		if l.peek() == '*' && l.peekNext() == '*' {
			value += "**"
			l.advance()
			l.advance()
			l.addTokenAt(token.DISCRETION, value, start)
			return
		}

		if l.peek() == '\n' || l.peek() == '\r' {
			l.addError("Unterminated discretion marker (use *** for multi-line)")
			return
		}

		value += string(l.peek())
		l.advance()
	}

	l.addError("Unterminated discretion marker")
}

// scanMultilineDiscretion scans ***...***
func (l *Lexer) scanMultilineDiscretion(start token.SourceLocation) {
	l.advance() // first *
	l.advance() // second *
	l.advance() // third *

	value := "***"

	for !l.isAtEnd() {
		if l.peek() == '*' && l.peekAt(1) == '*' && l.peekAt(2) == '*' {
			value += "***"
			l.advance()
			l.advance()
			l.advance()
			l.addTokenAt(token.MULTILINE_DISCRETION, value, start)
			return
		}

		if l.peek() == '\n' {
			value += "\n"
			l.advance()
			l.line++
			l.column = 1
		} else if l.peek() == '\r' {
			l.advance()
			if l.peek() == '\n' {
				l.advance()
			}
			value += "\n"
			l.line++
			l.column = 1
		} else {
			value += string(l.peek())
			l.advance()
		}
	}

	l.addError("Unterminated multiline discretion marker")
}

// Helper methods.

func (l *Lexer) isAtEnd() bool {
	return l.pos >= len(l.source)
}

func (l *Lexer) peek() byte {
	if l.pos >= len(l.source) {
		return 0
	}
	return l.source[l.pos]
}

func (l *Lexer) peekNext() byte {
	if l.pos+1 >= len(l.source) {
		return 0
	}
	return l.source[l.pos+1]
}

func (l *Lexer) peekAt(offset int) byte {
	if l.pos+offset >= len(l.source) {
		return 0
	}
	return l.source[l.pos+offset]
}

func (l *Lexer) advance() byte {
	if l.pos >= len(l.source) {
		return 0
	}
	c := l.source[l.pos]
	l.pos++
	l.column++
	return c
}

func (l *Lexer) currentLocation() token.SourceLocation {
	return token.SourceLocation{Line: l.line, Column: l.column, Offset: l.pos}
}

func (l *Lexer) addToken(typ token.TokenType, value string) {
	start := l.currentLocation()
	l.addTokenAt(typ, value, start)
}

func (l *Lexer) addTokenAndAdvance(typ token.TokenType, value string) {
	start := l.currentLocation()
	l.advance()
	l.addTokenAt(typ, value, start)
}

func (l *Lexer) addTokenAt(typ token.TokenType, value string, start token.SourceLocation) {
	end := l.currentLocation()
	l.tokens = append(l.tokens, token.Token{
		Type:  typ,
		Value: value,
		Span:  token.SourceSpan{Start: start, End: end},
	})
}

func (l *Lexer) addTokenAtTrivia(typ token.TokenType, value string, start token.SourceLocation, isTrivia bool) {
	end := l.currentLocation()
	l.tokens = append(l.tokens, token.Token{
		Type:    typ,
		Value:   value,
		Span:    token.SourceSpan{Start: start, End: end},
		IsTrivia: isTrivia,
	})
}

func (l *Lexer) addStringToken(value string, start token.SourceLocation, meta *token.StringTokenMetadata) {
	end := l.currentLocation()
	l.tokens = append(l.tokens, token.Token{
		Type:       token.STRING,
		Value:      value,
		Span:       token.SourceSpan{Start: start, End: end},
		StringMeta: meta,
	})
}

func (l *Lexer) addError(message string) {
	loc := l.currentLocation()
	l.errors = append(l.errors, Error{
		Message:  message,
		Span:     token.SourceSpan{Start: loc, End: loc},
		Severity: "error",
	})
}

func (l *Lexer) addWarning(message string) {
	loc := l.currentLocation()
	l.errors = append(l.errors, Error{
		Message:  message,
		Span:     token.SourceSpan{Start: loc, End: loc},
		Severity: "warning",
	})
}

// Character classification helpers.

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isAlphaNumericOrHyphen(c byte) bool {
	return isAlpha(c) || isDigit(c) || c == '-'
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// Tokenize is a convenience function to tokenize source with default options.
func Tokenize(source string, opts ...Options) Result {
	return New(source, opts...).Tokenize()
}

// TokenizeWithoutComments tokenizes and filters out comment tokens.
func TokenizeWithoutComments(source string) Result {
	result := Tokenize(source, Options{IncludeComments: true})
	filtered := make([]token.Token, 0, len(result.Tokens))
	for _, t := range result.Tokens {
		if t.Type != token.COMMENT {
			filtered = append(filtered, t)
		}
	}
	return Result{Tokens: filtered, Errors: result.Errors}
}

