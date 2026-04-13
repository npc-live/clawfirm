// Package parser implements a recursive descent parser for the WhipFlow language.
// It takes a stream of tokens produced by the lexer and produces an abstract
// syntax tree (AST).
package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ai-gateway/clawfirm/whipflow/ast"
	"github.com/ai-gateway/clawfirm/whipflow/lexer"
	"github.com/ai-gateway/clawfirm/whipflow/token"
)

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

// ParseError represents a single error encountered during parsing.
type ParseError struct {
	Message string
	Span    token.SourceSpan
}

// ParseResult holds the output of a Parse invocation.
type ParseResult struct {
	Program *ast.Program
	Errors  []ParseError
}

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

// Parser is a recursive descent parser for WhipFlow source code.
type Parser struct {
	tokens []token.Token
	pos    int
	errors []ParseError
}

// Parse tokenizes the given source string and parses it into an AST.
func Parse(source string) ParseResult {
	result := lexer.Tokenize(source)

	// Convert lexer errors into ParseErrors so callers see them.
	var errs []ParseError
	for _, le := range result.Errors {
		errs = append(errs, ParseError{
			Message: le.Message,
			Span:    le.Span,
		})
	}

	// Filter out NEWLINE and COMMENT tokens — the parser works with
	// significant tokens only (INDENT/DEDENT delimit blocks).
	filtered := make([]token.Token, 0, len(result.Tokens))
	for _, t := range result.Tokens {
		if t.Type != token.NEWLINE && t.Type != token.COMMENT {
			filtered = append(filtered, t)
		}
	}

	p := &Parser{tokens: filtered}
	program := p.parseProgram()

	return ParseResult{
		Program: program,
		Errors:  append(errs, p.errors...),
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// peek returns the current token without advancing.
func (p *Parser) peek() token.Token {
	if p.pos >= len(p.tokens) {
		return token.Token{Type: token.EOF}
	}
	return p.tokens[p.pos]
}

// peekAt returns the token at the given offset from the current position.
func (p *Parser) peekAt(offset int) token.Token {
	idx := p.pos + offset
	if idx >= len(p.tokens) {
		return token.Token{Type: token.EOF}
	}
	return p.tokens[idx]
}

// advance consumes and returns the current token.
func (p *Parser) advance() token.Token {
	t := p.peek()
	if t.Type != token.EOF {
		p.pos++
	}
	return t
}

// check reports whether the current token has the given type.
func (p *Parser) check(tt token.TokenType) bool {
	return p.peek().Type == tt
}

// match advances if the current token matches any of the given types.
// It returns true if a token was consumed.
func (p *Parser) match(types ...token.TokenType) bool {
	for _, tt := range types {
		if p.check(tt) {
			p.advance()
			return true
		}
	}
	return false
}

// expect consumes the current token if it matches tt. Otherwise it records an
// error and returns a zero-value token.
func (p *Parser) expect(tt token.TokenType) token.Token {
	if p.check(tt) {
		return p.advance()
	}
	p.addError("expected "+tt.String()+", got "+p.peek().Type.String(), p.peek().Span)
	return token.Token{Type: tt, Span: p.peek().Span}
}

// isAtEnd reports whether the parser has reached the end of the token stream.
func (p *Parser) isAtEnd() bool {
	return p.peek().Type == token.EOF
}

// addError records a parse error.
func (p *Parser) addError(msg string, span token.SourceSpan) {
	p.errors = append(p.errors, ParseError{Message: msg, Span: span})
}

// skipNewlines is a no-op since NEWLINE tokens are filtered before parsing.
// It is retained for structural parity with the TypeScript implementation.
func (p *Parser) skipNewlines() {}

// spanFrom builds a SourceSpan from start to the span of the most recently
// consumed token.
func (p *Parser) spanFrom(start token.SourceSpan) token.SourceSpan {
	if p.pos > 0 {
		return token.SourceSpan{Start: start.Start, End: p.tokens[p.pos-1].Span.End}
	}
	return start
}

// ---------------------------------------------------------------------------
// Program
// ---------------------------------------------------------------------------

func (p *Parser) parseProgram() *ast.Program {
	startSpan := p.peek().Span
	var stmts []ast.Node

	for !p.isAtEnd() {
		p.skipNewlines()
		if p.isAtEnd() {
			break
		}
		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	return &ast.Program{
		Span:       p.spanFrom(startSpan),
		Statements: stmts,
	}
}

// ---------------------------------------------------------------------------
// Body helper
// ---------------------------------------------------------------------------

// parseBody expects an INDENT token, parses statements until a matching DEDENT
// (or EOF), and returns the collected nodes.
func (p *Parser) parseBody() []ast.Node {
	p.expect(token.INDENT)
	var body []ast.Node
	for !p.isAtEnd() && !p.check(token.DEDENT) {
		p.skipNewlines()
		if p.check(token.DEDENT) || p.isAtEnd() {
			break
		}
		stmt := p.parseStatement()
		if stmt != nil {
			body = append(body, stmt)
		}
	}
	if p.check(token.DEDENT) {
		p.advance()
	}
	return body
}

// ---------------------------------------------------------------------------
// Statement dispatch
// ---------------------------------------------------------------------------

func (p *Parser) parseStatement() ast.Node {
	p.skipNewlines()

	switch p.peek().Type {
	case token.AGENT:
		return p.parseAgentDefinition()
	case token.SESSION:
		return p.parseSessionStatement()
	case token.IMPORT:
		return p.parseImportStatement()
	case token.LET:
		return p.parseLetBinding()
	case token.CONST:
		return p.parseConstBinding()
	case token.DO:
		return p.parseDoBlock()
	case token.PARALLEL:
		return p.parseParallelBlock()
	case token.LOOP:
		return p.parseLoopBlock()
	case token.REPEAT:
		return p.parseRepeatBlock()
	case token.FOR:
		return p.parseForEachBlock(false)
	case token.TRY:
		return p.parseTryBlock()
	case token.THROW:
		return p.parseThrowStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.CHOICE:
		return p.parseChoiceBlock()
	case token.IF:
		return p.parseIfStatement()
	case token.BLOCK:
		return p.parseBlockDefinition()
	case token.ASK:
		return p.parseAskStatement()
	case token.RUN:
		return p.parseRunStatement()
	case token.SKILL:
		return p.parseSkillInvocation()
	case token.IDENTIFIER:
		// Check for assignment: IDENTIFIER EQUALS ...
		if p.peekAt(1).Type == token.EQUALS {
			return p.parseAssignment()
		}
		// Otherwise it is an error — identifiers alone are not statements.
		t := p.advance()
		p.addError("unexpected identifier '"+t.Value+"' at statement level", t.Span)
		return nil
	default:
		// Skip unknown tokens to avoid infinite loops.
		t := p.advance()
		p.addError("unexpected token "+t.Type.String()+" at statement level", t.Span)
		return nil
	}
}

// ---------------------------------------------------------------------------
// Assignment
// ---------------------------------------------------------------------------

func (p *Parser) parseAssignment() ast.Node {
	startSpan := p.peek().Span
	nameTok := p.advance() // IDENTIFIER
	p.expect(token.EQUALS)
	val := p.parseExpression()
	return &ast.Assignment{
		Span: p.spanFrom(startSpan),
		Name: &ast.Identifier{Span: nameTok.Span, Name: nameTok.Value},
		Value: val,
	}
}

// ---------------------------------------------------------------------------
// Agent definition
// ---------------------------------------------------------------------------

func (p *Parser) parseAgentDefinition() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.AGENT)

	nameTok := p.expect(token.IDENTIFIER)
	name := &ast.Identifier{Span: nameTok.Span, Name: nameTok.Value}

	p.expect(token.COLON)
	p.expect(token.INDENT)

	var properties []*ast.Property
	var body []ast.Node

	for !p.isAtEnd() && !p.check(token.DEDENT) {
		p.skipNewlines()
		if p.check(token.DEDENT) || p.isAtEnd() {
			break
		}

		if p.isPropertyKeyword() {
			prop := p.parseProperty()
			if prop != nil {
				properties = append(properties, prop)
			}
		} else {
			stmt := p.parseStatement()
			if stmt != nil {
				body = append(body, stmt)
			}
		}
	}

	if p.check(token.DEDENT) {
		p.advance()
	}

	return &ast.AgentDefinition{
		Span:       p.spanFrom(startSpan),
		Name:       name,
		Properties: properties,
		Body:       body,
	}
}

// ---------------------------------------------------------------------------
// Session statement
// ---------------------------------------------------------------------------

func (p *Parser) parseSessionStatement() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.SESSION)

	var agentIdent *ast.Identifier
	var nameIdent *ast.Identifier
	var prompt ast.Node
	var properties []*ast.Property

	// session : agentName
	// session name : agentName
	// session "inline prompt"
	// session:
	//   ...properties...

	// Check if next is COLON (no agent name, property block or bare session)
	if p.check(token.COLON) {
		p.advance() // consume COLON
		// Could be: agent name immediately, or INDENT for property block
		if p.check(token.IDENTIFIER) {
			agentTok := p.advance()
			agentIdent = &ast.Identifier{Span: agentTok.Span, Name: agentTok.Value}
		}
		if p.check(token.INDENT) {
			properties = p.parsePropertyBlock()
		}
	} else if p.check(token.IDENTIFIER) {
		// Could be: session agentName   OR   session name : agentName
		idTok := p.advance()

		if p.check(token.COLON) {
			// session name : agentName
			p.advance() // consume COLON
			nameIdent = &ast.Identifier{Span: idTok.Span, Name: idTok.Value}

			if p.check(token.IDENTIFIER) {
				agentTok := p.advance()
				agentIdent = &ast.Identifier{Span: agentTok.Span, Name: agentTok.Value}
			}
			if p.check(token.INDENT) {
				properties = p.parsePropertyBlock()
			}
		} else {
			// session agentName (no COLON follows — bare reference)
			agentIdent = &ast.Identifier{Span: idTok.Span, Name: idTok.Value}
		}
	} else if p.check(token.STRING) {
		prompt = p.parseStringExpression()
	}

	return &ast.SessionStatement{
		Span:       p.spanFrom(startSpan),
		Prompt:     prompt,
		Agent:      agentIdent,
		Name:       nameIdent,
		Properties: properties,
	}
}

// ---------------------------------------------------------------------------
// Import statement
// ---------------------------------------------------------------------------

func (p *Parser) parseImportStatement() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.IMPORT)

	skillName := p.parseStringExpression()

	p.expect(token.FROM)

	source := p.parseStringExpression()

	return &ast.ImportStatement{
		Span:      p.spanFrom(startSpan),
		SkillName: skillName,
		Source:    source,
	}
}

// ---------------------------------------------------------------------------
// Let / Const bindings
// ---------------------------------------------------------------------------

func (p *Parser) parseLetBinding() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.LET)

	nameTok := p.expect(token.IDENTIFIER)
	name := &ast.Identifier{Span: nameTok.Span, Name: nameTok.Value}

	p.expect(token.EQUALS)
	val := p.parseExpression()

	return &ast.LetBinding{
		Span:  p.spanFrom(startSpan),
		Name:  name,
		Value: val,
	}
}

func (p *Parser) parseConstBinding() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.CONST)

	nameTok := p.expect(token.IDENTIFIER)
	name := &ast.Identifier{Span: nameTok.Span, Name: nameTok.Value}

	p.expect(token.EQUALS)
	val := p.parseExpression()

	return &ast.ConstBinding{
		Span:  p.spanFrom(startSpan),
		Name:  name,
		Value: val,
	}
}

// ---------------------------------------------------------------------------
// Do block
// ---------------------------------------------------------------------------

func (p *Parser) parseDoBlock() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.DO)

	var name *ast.Identifier
	var args []ast.Node

	// Optional block name and arguments: do myBlock arg1 arg2 :
	if p.check(token.IDENTIFIER) {
		nameTok := p.advance()
		name = &ast.Identifier{Span: nameTok.Span, Name: nameTok.Value}

		// Parse arguments until COLON
		for !p.isAtEnd() && !p.check(token.COLON) && !p.check(token.INDENT) {
			arg := p.parseExpression()
			if arg != nil {
				args = append(args, arg)
			} else {
				break
			}
		}
	}

	p.expect(token.COLON)
	body := p.parseBody()

	return &ast.DoBlock{
		Span:      p.spanFrom(startSpan),
		Name:      name,
		Arguments: args,
		Body:      body,
	}
}

// ---------------------------------------------------------------------------
// Parallel block
// ---------------------------------------------------------------------------

func (p *Parser) parseParallelBlock() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.PARALLEL)

	// parallel for → delegate to parseForEachBlock with IsParallel = true
	if p.check(token.FOR) {
		return p.parseForEachBlock(true)
	}

	var joinStrategy ast.Node
	var anyCount *ast.NumberLiteral
	var onFail ast.Node

	// Optional modifiers in parentheses: (join: "all", on-fail: "continue")
	if p.check(token.LPAREN) {
		p.advance()
		joinStrategy, anyCount, onFail = p.parseParallelModifiers()
		p.expect(token.RPAREN)
	}

	p.expect(token.COLON)
	body := p.parseBody()

	return &ast.ParallelBlock{
		Span:         p.spanFrom(startSpan),
		JoinStrategy: joinStrategy,
		AnyCount:     anyCount,
		OnFail:       onFail,
		Body:         body,
	}
}

func (p *Parser) parseParallelModifiers() (joinStrategy ast.Node, anyCount *ast.NumberLiteral, onFail ast.Node) {
	for !p.isAtEnd() && !p.check(token.RPAREN) {
		if p.check(token.IDENTIFIER) || p.check(token.COMMA) {
			if p.check(token.COMMA) {
				p.advance()
				continue
			}

			keyTok := p.advance()
			key := keyTok.Value

			p.expect(token.COLON)

			switch key {
			case "join":
				joinStrategy = p.parseExpression()
			case "any":
				numTok := p.advance()
				val, err := strconv.ParseFloat(numTok.Value, 64)
				if err != nil {
					p.addError(fmt.Sprintf("invalid number for 'any': %s", numTok.Value), numTok.Span)
				}
				anyCount = &ast.NumberLiteral{Span: numTok.Span, Value: val, Raw: numTok.Value}
			case "on-fail":
				onFail = p.parseExpression()
			default:
				p.parseExpression() // consume value
			}
		} else {
			break
		}
	}
	return
}

// ---------------------------------------------------------------------------
// Loop block
// ---------------------------------------------------------------------------

func (p *Parser) parseLoopBlock() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.LOOP)

	variant := "loop"
	var condition *ast.Discretion
	var iterationVar *ast.Identifier
	var maxIterations *ast.NumberLiteral

	// Variant detection
	if p.check(token.UNTIL) {
		p.advance()
		variant = "until"
		condition = p.parseDiscretionExpression()
	} else if p.check(token.WHILE) {
		p.advance()
		variant = "while"
		condition = p.parseDiscretionExpression()
	}

	// Optional (max: N) safety limit
	if p.check(token.LPAREN) {
		p.advance()
		for !p.isAtEnd() && !p.check(token.RPAREN) {
			if p.check(token.IDENTIFIER) || p.check(token.COMMA) {
				if p.check(token.COMMA) {
					p.advance()
					continue
				}
				keyTok := p.advance()
				p.expect(token.COLON)
				if keyTok.Value == "max" {
					numTok := p.advance()
					val, err := strconv.ParseFloat(numTok.Value, 64)
					if err != nil {
						p.addError(fmt.Sprintf("invalid number for 'max': %s", numTok.Value), numTok.Span)
					}
					maxIterations = &ast.NumberLiteral{Span: numTok.Span, Value: val, Raw: numTok.Value}
				} else {
					p.parseExpression()
				}
			} else {
				break
			}
		}
		p.expect(token.RPAREN)
	}

	// Optional "as i" iteration variable
	if p.check(token.AS) {
		p.advance()
		varTok := p.expect(token.IDENTIFIER)
		iterationVar = &ast.Identifier{Span: varTok.Span, Name: varTok.Value}
	}

	p.expect(token.COLON)
	body := p.parseBody()

	return &ast.LoopBlock{
		Span:          p.spanFrom(startSpan),
		Variant:       variant,
		Condition:     condition,
		IterationVar:  iterationVar,
		MaxIterations: maxIterations,
		Body:          body,
	}
}

// ---------------------------------------------------------------------------
// Repeat block
// ---------------------------------------------------------------------------

func (p *Parser) parseRepeatBlock() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.REPEAT)

	// Count: NUMBER or IDENTIFIER
	var count ast.Node
	if p.check(token.NUMBER) {
		numTok := p.advance()
		val, err := strconv.ParseFloat(numTok.Value, 64)
		if err != nil {
			p.addError(fmt.Sprintf("invalid number for 'repeat': %s", numTok.Value), numTok.Span)
		}
		count = &ast.NumberLiteral{Span: numTok.Span, Value: val, Raw: numTok.Value}
	} else if p.check(token.IDENTIFIER) {
		idTok := p.advance()
		count = &ast.Identifier{Span: idTok.Span, Name: idTok.Value}
	} else {
		p.addError("expected number or identifier after 'repeat'", p.peek().Span)
	}

	// Optional: as indexVar
	var indexVar *ast.Identifier
	if p.check(token.AS) {
		p.advance()
		varTok := p.expect(token.IDENTIFIER)
		indexVar = &ast.Identifier{Span: varTok.Span, Name: varTok.Value}
	}

	p.expect(token.COLON)
	body := p.parseBody()

	return &ast.RepeatBlock{
		Span:     p.spanFrom(startSpan),
		Count:    count,
		IndexVar: indexVar,
		Body:     body,
	}
}

// ---------------------------------------------------------------------------
// For-each block
// ---------------------------------------------------------------------------

func (p *Parser) parseForEachBlock(isParallel bool) ast.Node {
	startSpan := p.peek().Span
	if !isParallel {
		// Regular for — the PARALLEL token was already consumed by the caller
		// when isParallel is true.
	}
	p.expect(token.FOR)

	itemTok := p.expect(token.IDENTIFIER)
	itemVar := &ast.Identifier{Span: itemTok.Span, Name: itemTok.Value}

	var indexVar *ast.Identifier
	if p.check(token.COMMA) {
		p.advance()
		idxTok := p.expect(token.IDENTIFIER)
		indexVar = &ast.Identifier{Span: idxTok.Span, Name: idxTok.Value}
	}

	p.expect(token.IN)
	collection := p.parseExpression()

	// Optional parenthesized modifiers
	var modifiers []*ast.Property
	if p.check(token.LPAREN) {
		p.advance()
		modifiers = p.parseModifierList()
		p.expect(token.RPAREN)
	}

	p.expect(token.COLON)
	body := p.parseBody()

	return &ast.ForEachBlock{
		Span:       p.spanFrom(startSpan),
		ItemVar:    itemVar,
		IndexVar:   indexVar,
		Collection: collection,
		IsParallel: isParallel,
		Modifiers:  modifiers,
		Body:       body,
	}
}

func (p *Parser) parseModifierList() []*ast.Property {
	var mods []*ast.Property
	for !p.isAtEnd() && !p.check(token.RPAREN) {
		if p.check(token.COMMA) {
			p.advance()
			continue
		}
		if p.check(token.IDENTIFIER) {
			modStart := p.peek().Span
			keyTok := p.advance()
			keyIdent := &ast.Identifier{Span: keyTok.Span, Name: keyTok.Value}
			p.expect(token.COLON)
			val := p.parseExpression()
			mods = append(mods, &ast.Property{
				Span:  p.spanFrom(modStart),
				Name:  keyIdent,
				Value: val,
			})
		} else {
			break
		}
	}
	return mods
}

// ---------------------------------------------------------------------------
// Try block
// ---------------------------------------------------------------------------

func (p *Parser) parseTryBlock() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.TRY)
	p.expect(token.COLON)
	tryBody := p.parseBody()

	var catchBody []ast.Node
	var finallyBody []ast.Node
	var errorVar *ast.Identifier

	// Optional catch clause
	if p.check(token.CATCH) {
		p.advance()

		// Optional: as errorVar
		if p.check(token.AS) {
			p.advance()
			varTok := p.expect(token.IDENTIFIER)
			errorVar = &ast.Identifier{Span: varTok.Span, Name: varTok.Value}
		}

		p.expect(token.COLON)
		catchBody = p.parseBody()
	}

	// Optional finally clause
	if p.check(token.FINALLY) {
		p.advance()
		p.expect(token.COLON)
		finallyBody = p.parseBody()
	}

	return &ast.TryBlock{
		Span:        p.spanFrom(startSpan),
		TryBody:     tryBody,
		CatchBody:   catchBody,
		FinallyBody: finallyBody,
		ErrorVar:    errorVar,
	}
}

// ---------------------------------------------------------------------------
// Throw statement
// ---------------------------------------------------------------------------

func (p *Parser) parseThrowStatement() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.THROW)

	var msg ast.Node
	if !p.isAtEnd() && !p.check(token.DEDENT) && !p.check(token.EOF) {
		if p.check(token.STRING) || p.check(token.IDENTIFIER) || p.check(token.DISCRETION) || p.check(token.MULTILINE_DISCRETION) {
			msg = p.parseExpression()
		}
	}

	return &ast.ThrowStatement{
		Span:    p.spanFrom(startSpan),
		Message: msg,
	}
}

// ---------------------------------------------------------------------------
// Return statement
// ---------------------------------------------------------------------------

func (p *Parser) parseReturnStatement() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.RETURN)

	var val ast.Node
	if !p.isAtEnd() && !p.check(token.DEDENT) && !p.check(token.EOF) {
		if p.check(token.STRING) || p.check(token.NUMBER) || p.check(token.IDENTIFIER) ||
			p.check(token.LBRACKET) || p.check(token.LBRACE) ||
			p.check(token.DISCRETION) || p.check(token.MULTILINE_DISCRETION) {
			val = p.parseExpression()
		}
	}

	return &ast.ReturnStatement{
		Span:  p.spanFrom(startSpan),
		Value: val,
	}
}

// ---------------------------------------------------------------------------
// Choice block
// ---------------------------------------------------------------------------

func (p *Parser) parseChoiceBlock() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.CHOICE)

	criteria := p.parseDiscretionExpression()

	p.expect(token.COLON)
	p.expect(token.INDENT)

	var options []*ast.ChoiceOption
	for !p.isAtEnd() && !p.check(token.DEDENT) {
		p.skipNewlines()
		if p.check(token.DEDENT) || p.isAtEnd() {
			break
		}
		if p.check(token.OPTION) {
			opt := p.parseChoiceOption()
			if opt != nil {
				options = append(options, opt)
			}
		} else {
			// Skip unexpected tokens within choice block
			t := p.advance()
			p.addError("expected 'option' in choice block, got "+t.Type.String(), t.Span)
		}
	}

	if p.check(token.DEDENT) {
		p.advance()
	}

	return &ast.ChoiceBlock{
		Span:     p.spanFrom(startSpan),
		Criteria: criteria,
		Options:  options,
	}
}

func (p *Parser) parseChoiceOption() *ast.ChoiceOption {
	startSpan := p.peek().Span
	p.expect(token.OPTION)

	var label ast.Node
	if p.check(token.STRING) {
		label = p.parseStringExpression()
	} else if p.check(token.DISCRETION) || p.check(token.MULTILINE_DISCRETION) {
		label = p.parseDiscretionNode()
	} else {
		p.addError("expected string or discretion after 'option'", p.peek().Span)
	}

	p.expect(token.COLON)
	body := p.parseBody()

	return &ast.ChoiceOption{
		Span:  p.spanFrom(startSpan),
		Label: label,
		Body:  body,
	}
}

// ---------------------------------------------------------------------------
// If statement
// ---------------------------------------------------------------------------

func (p *Parser) parseIfStatement() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.IF)

	condition := p.parseDiscretionExpression()

	p.expect(token.COLON)
	thenBody := p.parseBody()

	var elseIfClauses []*ast.ElseIfClause
	var elseBody []ast.Node

	// Optional elif clauses
	for p.check(token.ELIF) {
		elifStart := p.peek().Span
		p.advance()
		elifCond := p.parseDiscretionExpression()
		p.expect(token.COLON)
		elifBody := p.parseBody()
		elseIfClauses = append(elseIfClauses, &ast.ElseIfClause{
			Span:      p.spanFrom(elifStart),
			Condition: elifCond,
			Body:      elifBody,
		})
	}

	// Optional else clause
	if p.check(token.ELSE) {
		p.advance()
		p.expect(token.COLON)
		elseBody = p.parseBody()
	}

	return &ast.IfStatement{
		Span:          p.spanFrom(startSpan),
		Condition:     condition,
		ThenBody:      thenBody,
		ElseIfClauses: elseIfClauses,
		ElseBody:      elseBody,
	}
}

// ---------------------------------------------------------------------------
// Block definition
// ---------------------------------------------------------------------------

func (p *Parser) parseBlockDefinition() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.BLOCK)

	nameTok := p.expect(token.IDENTIFIER)
	name := &ast.Identifier{Span: nameTok.Span, Name: nameTok.Value}

	// Optional parameter list
	var params []*ast.Identifier
	if p.check(token.LPAREN) {
		p.advance()
		for !p.isAtEnd() && !p.check(token.RPAREN) {
			if p.check(token.COMMA) {
				p.advance()
				continue
			}
			paramTok := p.expect(token.IDENTIFIER)
			params = append(params, &ast.Identifier{Span: paramTok.Span, Name: paramTok.Value})
		}
		p.expect(token.RPAREN)
	}

	p.expect(token.COLON)
	body := p.parseBody()

	return &ast.BlockDefinition{
		Span:       p.spanFrom(startSpan),
		Name:       name,
		Parameters: params,
		Body:       body,
	}
}

// ---------------------------------------------------------------------------
// Ask statement
// ---------------------------------------------------------------------------

func (p *Parser) parseAskStatement() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.ASK)

	varTok := p.expect(token.IDENTIFIER)
	varIdent := &ast.Identifier{Span: varTok.Span, Name: varTok.Value}

	p.expect(token.COLON)

	prompt := p.parseExpression()

	return &ast.AskStatement{
		Span:     p.spanFrom(startSpan),
		Variable: varIdent,
		Prompt:   prompt,
	}
}

// ---------------------------------------------------------------------------
// Run statement
// ---------------------------------------------------------------------------

func (p *Parser) parseRunStatement() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.RUN)

	filePath := p.parseStringExpression()

	return &ast.RunStatement{
		Span:     p.spanFrom(startSpan),
		FilePath: filePath,
	}
}

// ---------------------------------------------------------------------------
// Skill invocation
// ---------------------------------------------------------------------------

func (p *Parser) parseSkillInvocation() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.SKILL)

	skillNameTok := p.expect(token.IDENTIFIER)
	skillName := &ast.Identifier{Span: skillNameTok.Span, Name: skillNameTok.Value}

	// Parse params: name = expression, ... until ARROW or end-of-statement
	var params []*ast.SkillParam
	for !p.isAtEnd() && !p.check(token.ARROW) && !p.check(token.DEDENT) &&
		!p.check(token.EOF) && !p.check(token.INDENT) {
		if p.check(token.COMMA) {
			p.advance()
			continue
		}
		if p.check(token.IDENTIFIER) && p.peekAt(1).Type == token.EQUALS {
			paramStart := p.peek().Span
			paramNameTok := p.advance()
			paramName := &ast.Identifier{Span: paramNameTok.Span, Name: paramNameTok.Value}
			p.expect(token.EQUALS)
			val := p.parseExpression()
			params = append(params, &ast.SkillParam{
				Span:  p.spanFrom(paramStart),
				Name:  paramName,
				Value: val,
			})
		} else {
			break
		}
	}

	// Optional: -> outputVar
	var outputVar *ast.Identifier
	if p.check(token.ARROW) {
		p.advance()
		outTok := p.expect(token.IDENTIFIER)
		outputVar = &ast.Identifier{Span: outTok.Span, Name: outTok.Value}
	}

	return &ast.SkillInvocation{
		Span:      p.spanFrom(startSpan),
		SkillName: skillName,
		Params:    params,
		OutputVar: outputVar,
	}
}

// ---------------------------------------------------------------------------
// Properties
// ---------------------------------------------------------------------------

// isPropertyKeyword reports whether the current token is a property keyword
// that can appear inside agent/session blocks.
func (p *Parser) isPropertyKeyword() bool {
	switch p.peek().Type {
	case token.MODEL, token.PROMPT, token.SKILLS, token.TOOLS,
		token.PERMISSIONS, token.CONTEXT:
		return true
	}
	// Also accept identifiers that match known property names (e.g., "provider")
	if p.peek().Type == token.IDENTIFIER {
		switch p.peek().Value {
		case "provider":
			return true
		}
	}
	return false
}

// parseProperty parses a single property like "model: gpt-4" or "skills:" followed
// by an indented block.
func (p *Parser) parseProperty() *ast.Property {
	startSpan := p.peek().Span
	keyTok := p.advance()
	keyIdent := &ast.Identifier{Span: keyTok.Span, Name: keyTok.Value}

	p.expect(token.COLON)

	var val ast.Node

	// If next token is INDENT, the property value is a block (e.g., skills list)
	if p.check(token.INDENT) {
		p.advance() // consume INDENT
		var items []ast.Node
		for !p.isAtEnd() && !p.check(token.DEDENT) {
			p.skipNewlines()
			if p.check(token.DEDENT) || p.isAtEnd() {
				break
			}
			item := p.parseExpression()
			if item != nil {
				items = append(items, item)
			}
		}
		if p.check(token.DEDENT) {
			p.advance()
		}
		// Wrap items in an ArrayExpression
		val = &ast.ArrayExpression{
			Span:     p.spanFrom(startSpan),
			Elements: items,
		}
	} else {
		val = p.parseExpression()
	}

	return &ast.Property{
		Span:  p.spanFrom(startSpan),
		Name:  keyIdent,
		Value: val,
	}
}

// parsePropertyBlock parses properties within an indented block (used by session).
func (p *Parser) parsePropertyBlock() []*ast.Property {
	p.expect(token.INDENT)
	var props []*ast.Property
	for !p.isAtEnd() && !p.check(token.DEDENT) {
		p.skipNewlines()
		if p.check(token.DEDENT) || p.isAtEnd() {
			break
		}
		if p.isPropertyKeyword() {
			prop := p.parseProperty()
			if prop != nil {
				props = append(props, prop)
			}
		} else {
			// Might also be an identifier key
			if p.check(token.IDENTIFIER) && p.peekAt(1).Type == token.COLON {
				prop := p.parseProperty()
				if prop != nil {
					props = append(props, prop)
				}
			} else {
				t := p.advance()
				p.addError("unexpected token in property block: "+t.Type.String(), t.Span)
			}
		}
	}
	if p.check(token.DEDENT) {
		p.advance()
	}
	return props
}

// ---------------------------------------------------------------------------
// Expressions
// ---------------------------------------------------------------------------

func (p *Parser) parseExpression() ast.Node {
	node := p.parsePrimary()
	if node == nil {
		return nil
	}

	// Post-primary: check for pipe or arrow
	for {
		if p.check(token.PIPE) {
			node = p.parsePipeExpression(node)
		} else if p.check(token.ARROW) {
			node = p.parseArrowExpression(node)
		} else {
			break
		}
	}

	return node
}

func (p *Parser) parsePrimary() ast.Node {
	switch p.peek().Type {
	case token.STRING:
		return p.parseStringExpression()

	case token.NUMBER:
		return p.parseNumberLiteral()

	case token.IDENTIFIER:
		return p.parseIdentifierExpression()

	case token.LBRACKET:
		return p.parseArrayExpression()

	case token.LBRACE:
		return p.parseObjectExpression()

	case token.DISCRETION:
		return p.parseDiscretionNode()

	case token.MULTILINE_DISCRETION:
		return p.parseDiscretionNode()

	case token.SESSION:
		// Allow `session "..."` as an expression so it can be used in
		// let/const bindings: `let x = session "prompt"`.
		return p.parseSessionStatement()

	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// String expression (with interpolation handling)
// ---------------------------------------------------------------------------

func (p *Parser) parseStringExpression() ast.Node {
	tok := p.advance()

	// Check for interpolations in StringMeta
	if tok.StringMeta != nil && len(tok.StringMeta.Interpolations) > 0 {
		return p.buildInterpolatedString(tok)
	}

	return &ast.StringLiteral{
		Span:            tok.Span,
		Value:           tok.Value,
		Raw:             stringMetaRaw(tok),
		IsTripleQuoted:  stringMetaTriple(tok),
		EscapeSequences: stringMetaEscapes(tok),
	}
}

// buildInterpolatedString converts a STRING token with interpolation metadata
// into an InterpolatedString AST node.
func (p *Parser) buildInterpolatedString(tok token.Token) ast.Node {
	meta := tok.StringMeta
	var parts []ast.Node

	value := tok.Value
	lastEnd := 0

	for _, interp := range meta.Interpolations {
		// Find the interpolation raw text (e.g., "{name}") in the value
		idx := strings.Index(value[lastEnd:], interp.Raw)
		if idx < 0 {
			continue
		}
		absIdx := lastEnd + idx

		// Text before the interpolation
		if absIdx > lastEnd {
			textPart := value[lastEnd:absIdx]
			parts = append(parts, &ast.StringLiteral{
				Span:  tok.Span,
				Value: textPart,
			})
		}

		// The interpolated identifier
		parts = append(parts, &ast.Identifier{
			Span: tok.Span,
			Name: interp.VarName,
		})

		lastEnd = absIdx + len(interp.Raw)
	}

	// Trailing text after last interpolation
	if lastEnd < len(value) {
		parts = append(parts, &ast.StringLiteral{
			Span:  tok.Span,
			Value: value[lastEnd:],
		})
	}

	return &ast.InterpolatedString{
		Span:            tok.Span,
		Parts:           parts,
		Raw:             stringMetaRaw(tok),
		IsTripleQuoted:  stringMetaTriple(tok),
		Value:           value,
		EscapeSequences: stringMetaEscapes(tok),
	}
}

// ---------------------------------------------------------------------------
// Number literal
// ---------------------------------------------------------------------------

func (p *Parser) parseNumberLiteral() ast.Node {
	tok := p.advance()
	val, err := strconv.ParseFloat(tok.Value, 64)
	if err != nil {
		p.addError(fmt.Sprintf("invalid number literal: %s", tok.Value), tok.Span)
	}
	return &ast.NumberLiteral{
		Span:  tok.Span,
		Value: val,
		Raw:   tok.Value,
	}
}

// ---------------------------------------------------------------------------
// Identifier expression
// ---------------------------------------------------------------------------

func (p *Parser) parseIdentifierExpression() ast.Node {
	tok := p.advance()
	return &ast.Identifier{
		Span: tok.Span,
		Name: tok.Value,
	}
}

// ---------------------------------------------------------------------------
// Array expression
// ---------------------------------------------------------------------------

func (p *Parser) parseArrayExpression() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.LBRACKET)

	var elements []ast.Node
	for !p.isAtEnd() && !p.check(token.RBRACKET) {
		if p.check(token.COMMA) {
			p.advance()
			continue
		}
		elem := p.parseExpression()
		if elem != nil {
			elements = append(elements, elem)
		} else {
			break
		}
	}

	p.expect(token.RBRACKET)

	return &ast.ArrayExpression{
		Span:     p.spanFrom(startSpan),
		Elements: elements,
	}
}

// ---------------------------------------------------------------------------
// Object expression
// ---------------------------------------------------------------------------

func (p *Parser) parseObjectExpression() ast.Node {
	startSpan := p.peek().Span
	p.expect(token.LBRACE)

	var props []*ast.Property
	for !p.isAtEnd() && !p.check(token.RBRACE) {
		if p.check(token.COMMA) {
			p.advance()
			continue
		}

		propStart := p.peek().Span

		// Key can be an identifier or a string
		var keyIdent *ast.Identifier
		if p.check(token.IDENTIFIER) {
			keyTok := p.advance()
			keyIdent = &ast.Identifier{Span: keyTok.Span, Name: keyTok.Value}
		} else if p.check(token.STRING) {
			strTok := p.advance()
			keyIdent = &ast.Identifier{Span: strTok.Span, Name: strTok.Value}
		} else {
			// Also accept keyword tokens as keys
			if token.IsKeyword(p.peek().Type) {
				keyTok := p.advance()
				keyIdent = &ast.Identifier{Span: keyTok.Span, Name: keyTok.Value}
			} else {
				t := p.advance()
				p.addError("expected property key in object, got "+t.Type.String(), t.Span)
				continue
			}
		}

		p.expect(token.COLON)
		val := p.parseExpression()

		props = append(props, &ast.Property{
			Span:  p.spanFrom(propStart),
			Name:  keyIdent,
			Value: val,
		})
	}

	p.expect(token.RBRACE)

	return &ast.ObjectExpression{
		Span:       p.spanFrom(startSpan),
		Properties: props,
	}
}

// ---------------------------------------------------------------------------
// Discretion expressions
// ---------------------------------------------------------------------------

// parseDiscretionExpression parses a DISCRETION or MULTILINE_DISCRETION token
// and returns a *ast.Discretion. If the current token is neither, it adds an
// error and returns a zero-value discretion node.
func (p *Parser) parseDiscretionExpression() *ast.Discretion {
	if p.check(token.DISCRETION) || p.check(token.MULTILINE_DISCRETION) {
		tok := p.advance()
		return &ast.Discretion{
			Span:        tok.Span,
			Expression:  extractDiscretionText(tok.Value),
			IsMultiline: tok.Type == token.MULTILINE_DISCRETION,
		}
	}
	p.addError("expected discretion marker (**...**)", p.peek().Span)
	return &ast.Discretion{
		Span: p.peek().Span,
	}
}

// parseDiscretionNode is like parseDiscretionExpression but returns an ast.Node.
func (p *Parser) parseDiscretionNode() ast.Node {
	tok := p.advance()
	return &ast.Discretion{
		Span:        tok.Span,
		Expression:  extractDiscretionText(tok.Value),
		IsMultiline: tok.Type == token.MULTILINE_DISCRETION,
	}
}

// extractDiscretionText strips the ** or *** delimiters from a discretion
// token value.
func extractDiscretionText(raw string) string {
	if strings.HasPrefix(raw, "***") && strings.HasSuffix(raw, "***") {
		return strings.TrimSpace(raw[3 : len(raw)-3])
	}
	if strings.HasPrefix(raw, "**") && strings.HasSuffix(raw, "**") {
		return strings.TrimSpace(raw[2 : len(raw)-2])
	}
	return raw
}

// ---------------------------------------------------------------------------
// Pipe expression
// ---------------------------------------------------------------------------

func (p *Parser) parsePipeExpression(input ast.Node) ast.Node {
	startSpan := input.GetSpan()
	var operations []*ast.PipeOperation

	for p.check(token.PIPE) {
		p.advance() // consume PIPE

		op := p.parsePipeOperation()
		if op != nil {
			operations = append(operations, op)
		}
	}

	if len(operations) == 0 {
		return input
	}

	return &ast.PipeExpression{
		Span:       p.spanFrom(startSpan),
		Input:      input,
		Operations: operations,
	}
}

func (p *Parser) parsePipeOperation() *ast.PipeOperation {
	startSpan := p.peek().Span

	// Operator: map, filter, reduce, pmap
	var operator string
	switch p.peek().Type {
	case token.MAP:
		operator = "map"
		p.advance()
	case token.FILTER:
		operator = "filter"
		p.advance()
	case token.REDUCE:
		operator = "reduce"
		p.advance()
	case token.PMAP:
		operator = "pmap"
		p.advance()
	default:
		// Accept identifiers too
		if p.check(token.IDENTIFIER) {
			tok := p.advance()
			operator = tok.Value
		} else {
			p.addError("expected pipe operator (map, filter, reduce, pmap)", p.peek().Span)
			return nil
		}
	}

	var accVar *ast.Identifier
	var itemVar *ast.Identifier

	// For reduce: (acc, item) or just (item)
	// For map/filter/pmap: (item)
	if p.check(token.LPAREN) {
		p.advance()
		if p.check(token.IDENTIFIER) {
			firstTok := p.advance()
			if p.check(token.COMMA) {
				// reduce: acc, item
				p.advance()
				accVar = &ast.Identifier{Span: firstTok.Span, Name: firstTok.Value}
				if p.check(token.IDENTIFIER) {
					secondTok := p.advance()
					itemVar = &ast.Identifier{Span: secondTok.Span, Name: secondTok.Value}
				}
			} else {
				itemVar = &ast.Identifier{Span: firstTok.Span, Name: firstTok.Value}
			}
		}
		p.expect(token.RPAREN)
	}

	// Body: either COLON INDENT body DEDENT, or inline expression
	var body []ast.Node
	if p.check(token.COLON) {
		p.advance()
		body = p.parseBody()
	} else {
		// Inline expression
		expr := p.parseExpression()
		if expr != nil {
			body = []ast.Node{expr}
		}
	}

	return &ast.PipeOperation{
		Span:     p.spanFrom(startSpan),
		Operator: operator,
		AccVar:   accVar,
		ItemVar:  itemVar,
		Body:     body,
	}
}

// ---------------------------------------------------------------------------
// Arrow expression
// ---------------------------------------------------------------------------

func (p *Parser) parseArrowExpression(left ast.Node) ast.Node {
	startSpan := left.GetSpan()
	p.expect(token.ARROW)
	right := p.parsePrimary()

	return &ast.ArrowExpression{
		Span:  p.spanFrom(startSpan),
		Left:  left,
		Right: right,
	}
}

// ---------------------------------------------------------------------------
// String metadata helpers
// ---------------------------------------------------------------------------

func stringMetaRaw(tok token.Token) string {
	if tok.StringMeta != nil {
		return tok.StringMeta.Raw
	}
	return tok.Value
}

func stringMetaTriple(tok token.Token) bool {
	if tok.StringMeta != nil {
		return tok.StringMeta.IsTripleQuoted
	}
	return false
}

func stringMetaEscapes(tok token.Token) []token.EscapeSequenceInfo {
	if tok.StringMeta != nil {
		return tok.StringMeta.EscapeSequences
	}
	return nil
}
