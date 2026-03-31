// Package validator performs semantic validation on WhipFlow programs.
package validator

import (
	"fmt"
	"strings"

	"github.com/ai-gateway/clawfirm/whipflow/ast"
	"github.com/ai-gateway/clawfirm/whipflow/token"
)

// ValidationError represents a validation issue.
type ValidationError struct {
	Message  string
	Span     token.SourceSpan
	Severity string // "error", "warning", "info"
}

func (e ValidationError) Error() string { return e.Message }

// ValidationResult holds the outcome of validation.
type ValidationResult struct {
	Valid    bool
	Errors   []ValidationError
	Warnings []ValidationError
}

// Valid model names.
var validModels = map[string]bool{
	"opus": true, "sonnet": true, "haiku": true,
}

// Valid join strategies.
var validJoinStrategies = map[string]bool{
	"all": true, "first": true, "any": true,
}

// Valid on-fail strategies.
var validOnFailStrategies = map[string]bool{
	"fail-fast": true, "continue": true, "ignore": true,
}

// Built-in provider names.
var builtinProviders = map[string]bool{
	"claude-code": true, "claude": true, "opencode": true,
	"aider": true, "pi": true, "fetch": true,
}

// scopeType tracks the kind of scope for context-sensitive validation.
type scopeType string

const (
	scopeGlobal   scopeType = "global"
	scopeBlock    scopeType = "block"
	scopeLoop     scopeType = "loop"
	scopeFunction scopeType = "function"
	scopeTry      scopeType = "try"
	scopeCatch    scopeType = "catch"
)

// binding tracks a declared variable.
type binding struct {
	name    string
	isConst bool
	span    token.SourceSpan
}

// scope holds variable bindings for a lexical scope.
type scope struct {
	scopeType scopeType
	bindings  map[string]*binding
}

// Validator performs semantic validation.
type Validator struct {
	errors   []ValidationError
	warnings []ValidationError
	agents   map[string]token.SourceSpan // declared agent names
	blocks   map[string]token.SourceSpan // declared block names
	scopes   []scope
}

// Validate validates a parsed program.
func Validate(program *ast.Program) *ValidationResult {
	v := &Validator{
		agents: make(map[string]token.SourceSpan),
		blocks: make(map[string]token.SourceSpan),
	}
	v.pushScope(scopeGlobal)
	v.validateProgram(program)

	return &ValidationResult{
		Valid:    len(v.errors) == 0,
		Errors:   v.errors,
		Warnings: v.warnings,
	}
}

func (v *Validator) pushScope(st scopeType) {
	v.scopes = append(v.scopes, scope{scopeType: st, bindings: make(map[string]*binding)})
}

func (v *Validator) popScope() {
	if len(v.scopes) > 0 {
		v.scopes = v.scopes[:len(v.scopes)-1]
	}
}

func (v *Validator) currentScope() *scope {
	return &v.scopes[len(v.scopes)-1]
}

func (v *Validator) declareVar(name string, isConst bool, span token.SourceSpan) {
	s := v.currentScope()
	if _, exists := s.bindings[name]; exists {
		v.addError(fmt.Sprintf("variable '%s' is already declared in this scope", name), span)
		return
	}
	s.bindings[name] = &binding{name: name, isConst: isConst, span: span}
}

func (v *Validator) lookupVar(name string) *binding {
	for i := len(v.scopes) - 1; i >= 0; i-- {
		if b, ok := v.scopes[i].bindings[name]; ok {
			return b
		}
	}
	return nil
}

func (v *Validator) isInLoop() bool {
	for i := len(v.scopes) - 1; i >= 0; i-- {
		if v.scopes[i].scopeType == scopeLoop {
			return true
		}
	}
	return false
}

func (v *Validator) addError(msg string, span token.SourceSpan) {
	v.errors = append(v.errors, ValidationError{Message: msg, Span: span, Severity: "error"})
}

func (v *Validator) addWarning(msg string, span token.SourceSpan) {
	v.warnings = append(v.warnings, ValidationError{Message: msg, Span: span, Severity: "warning"})
}

func (v *Validator) validateProgram(prog *ast.Program) {
	// First pass: collect agent and block declarations.
	for _, stmt := range prog.Statements {
		switch n := stmt.(type) {
		case *ast.AgentDefinition:
			if _, exists := v.agents[n.Name.Name]; exists {
				v.addError(fmt.Sprintf("duplicate agent definition: '%s'", n.Name.Name), n.GetSpan())
			} else {
				v.agents[n.Name.Name] = n.GetSpan()
			}
		case *ast.BlockDefinition:
			if _, exists := v.blocks[n.Name.Name]; exists {
				v.addError(fmt.Sprintf("duplicate block definition: '%s'", n.Name.Name), n.GetSpan())
			} else {
				v.blocks[n.Name.Name] = n.GetSpan()
			}
		}
	}

	// Second pass: validate all statements.
	for _, stmt := range prog.Statements {
		v.validateStatement(stmt)
	}
}

func (v *Validator) validateStatement(stmt ast.Node) {
	if stmt == nil {
		return
	}
	switch n := stmt.(type) {
	case *ast.SessionStatement:
		v.validateSessionStatement(n)
	case *ast.AgentDefinition:
		v.validateAgentDefinition(n)
	case *ast.BlockDefinition:
		v.validateBlockDefinition(n)
	case *ast.LetBinding:
		v.validateExpression(n.Value)
		v.declareVar(n.Name.Name, false, n.GetSpan())
	case *ast.ConstBinding:
		v.validateExpression(n.Value)
		v.declareVar(n.Name.Name, true, n.GetSpan())
	case *ast.Assignment:
		v.validateAssignment(n)
	case *ast.DoBlock:
		v.validateDoBlock(n)
	case *ast.ParallelBlock:
		v.validateParallelBlock(n)
	case *ast.LoopBlock:
		v.validateLoopBlock(n)
	case *ast.RepeatBlock:
		v.validateRepeatBlock(n)
	case *ast.ForEachBlock:
		v.validateForEachBlock(n)
	case *ast.TryBlock:
		v.validateTryBlock(n)
	case *ast.ThrowStatement:
		if n.Message != nil {
			v.validateExpression(n.Message)
		}
	case *ast.ReturnStatement:
		if n.Value != nil {
			v.validateExpression(n.Value)
		}
	case *ast.ChoiceBlock:
		v.validateChoiceBlock(n)
	case *ast.IfStatement:
		v.validateIfStatement(n)
	case *ast.ImportStatement:
		// Import validation: just check expressions.
		v.validateExpression(n.SkillName)
		v.validateExpression(n.Source)
	case *ast.AskStatement:
		v.validateExpression(n.Prompt)
		v.declareVar(n.Variable.Name, false, n.GetSpan())
	case *ast.RunStatement:
		v.validateExpression(n.FilePath)
	case *ast.SkillInvocation:
		v.validateSkillInvocation(n)
	case *ast.CommentStatement:
		// No validation needed.
	case *ast.ArrowExpression:
		v.validateExpression(n)
	case *ast.PipeExpression:
		v.validateExpression(n)
	}
}

func (v *Validator) validateSessionStatement(n *ast.SessionStatement) {
	// Validate agent reference.
	if n.Agent != nil {
		if _, ok := v.agents[n.Agent.Name]; !ok {
			v.addError(fmt.Sprintf("undefined agent: '%s'", n.Agent.Name), n.Agent.GetSpan())
		}
	}

	// Validate prompt.
	if n.Prompt != nil {
		v.validateExpression(n.Prompt)
	}

	// Validate properties.
	for _, prop := range n.Properties {
		v.validateSessionProperty(prop)
	}
}

func (v *Validator) validateSessionProperty(prop *ast.Property) {
	name := prop.Name.Name

	switch name {
	case "model":
		v.validateModelValue(prop.Value, prop.GetSpan())
	case "prompt":
		v.validateExpression(prop.Value)
	case "provider":
		v.validateProviderValue(prop.Value, prop.GetSpan())
	case "skills", "tools", "permissions", "context":
		v.validateExpression(prop.Value)
	default:
		v.addWarning(fmt.Sprintf("unknown session property: '%s'", name), prop.GetSpan())
	}
}

func (v *Validator) validateModelValue(expr ast.Node, span token.SourceSpan) {
	switch e := expr.(type) {
	case *ast.Identifier:
		if !validModels[e.Name] {
			v.addError(fmt.Sprintf("invalid model: '%s' (expected opus, sonnet, or haiku)", e.Name), span)
		}
	case *ast.StringLiteral:
		if !validModels[e.Value] {
			v.addError(fmt.Sprintf("invalid model: '%s' (expected opus, sonnet, or haiku)", e.Value), span)
		}
	}
}

func (v *Validator) validateProviderValue(expr ast.Node, span token.SourceSpan) {
	switch e := expr.(type) {
	case *ast.Identifier:
		if !builtinProviders[e.Name] && !strings.HasPrefix(e.Name, "custom:") {
			v.addWarning(fmt.Sprintf("unknown provider: '%s'", e.Name), span)
		}
	case *ast.StringLiteral:
		if !builtinProviders[e.Value] && !strings.HasPrefix(e.Value, "custom:") {
			v.addWarning(fmt.Sprintf("unknown provider: '%s'", e.Value), span)
		}
	}
}

func (v *Validator) validateAgentDefinition(n *ast.AgentDefinition) {
	v.pushScope(scopeBlock)
	defer v.popScope()

	for _, prop := range n.Properties {
		v.validateSessionProperty(prop)
	}
	for _, stmt := range n.Body {
		v.validateStatement(stmt)
	}
}

func (v *Validator) validateBlockDefinition(n *ast.BlockDefinition) {
	v.pushScope(scopeFunction)
	defer v.popScope()

	for _, param := range n.Parameters {
		v.declareVar(param.Name, false, param.GetSpan())
	}
	for _, stmt := range n.Body {
		v.validateStatement(stmt)
	}
}

func (v *Validator) validateAssignment(n *ast.Assignment) {
	v.validateExpression(n.Value)
	b := v.lookupVar(n.Name.Name)
	if b == nil {
		// Implicit declaration: treat undeclared assignment as a let binding.
		v.declareVar(n.Name.Name, false, n.GetSpan())
		return
	}
	if b.isConst {
		v.addError(fmt.Sprintf("cannot reassign const variable '%s'", n.Name.Name), n.GetSpan())
	}
}

func (v *Validator) validateDoBlock(n *ast.DoBlock) {
	v.pushScope(scopeBlock)
	defer v.popScope()

	for _, arg := range n.Arguments {
		v.validateExpression(arg)
	}
	for _, stmt := range n.Body {
		v.validateStatement(stmt)
	}
}

func (v *Validator) validateParallelBlock(n *ast.ParallelBlock) {
	// Validate join strategy.
	if n.JoinStrategy != nil {
		val := nodeStringValue(n.JoinStrategy)
		if val != "" && !validJoinStrategies[val] {
			v.addError(fmt.Sprintf("invalid join strategy: '%s' (expected all, first, or any)", val), n.GetSpan())
		}
	}

	// Validate on-fail strategy.
	if n.OnFail != nil {
		val := nodeStringValue(n.OnFail)
		if val != "" && !validOnFailStrategies[val] {
			v.addError(fmt.Sprintf("invalid on-fail strategy: '%s' (expected fail-fast, continue, or ignore)", val), n.GetSpan())
		}
	}

	v.pushScope(scopeBlock)
	defer v.popScope()

	for _, stmt := range n.Body {
		v.validateStatement(stmt)
	}
}

func (v *Validator) validateLoopBlock(n *ast.LoopBlock) {
	v.pushScope(scopeLoop)
	defer v.popScope()

	if n.IterationVar != nil {
		v.declareVar(n.IterationVar.Name, false, n.IterationVar.GetSpan())
	}

	for _, stmt := range n.Body {
		v.validateStatement(stmt)
	}
}

func (v *Validator) validateRepeatBlock(n *ast.RepeatBlock) {
	v.pushScope(scopeLoop)
	defer v.popScope()

	v.validateExpression(n.Count)
	if n.IndexVar != nil {
		v.declareVar(n.IndexVar.Name, false, n.IndexVar.GetSpan())
	}

	for _, stmt := range n.Body {
		v.validateStatement(stmt)
	}
}

func (v *Validator) validateForEachBlock(n *ast.ForEachBlock) {
	v.pushScope(scopeLoop)
	defer v.popScope()

	v.validateExpression(n.Collection)
	v.declareVar(n.ItemVar.Name, false, n.ItemVar.GetSpan())
	if n.IndexVar != nil {
		v.declareVar(n.IndexVar.Name, false, n.IndexVar.GetSpan())
	}

	for _, mod := range n.Modifiers {
		v.validateExpression(mod.Value)
	}

	for _, stmt := range n.Body {
		v.validateStatement(stmt)
	}
}

func (v *Validator) validateTryBlock(n *ast.TryBlock) {
	// try body
	v.pushScope(scopeTry)
	for _, stmt := range n.TryBody {
		v.validateStatement(stmt)
	}
	v.popScope()

	// catch body
	if n.CatchBody != nil {
		v.pushScope(scopeCatch)
		if n.ErrorVar != nil {
			v.declareVar(n.ErrorVar.Name, false, n.ErrorVar.GetSpan())
		}
		for _, stmt := range n.CatchBody {
			v.validateStatement(stmt)
		}
		v.popScope()
	}

	// finally body
	if n.FinallyBody != nil {
		v.pushScope(scopeBlock)
		for _, stmt := range n.FinallyBody {
			v.validateStatement(stmt)
		}
		v.popScope()
	}
}

func (v *Validator) validateChoiceBlock(n *ast.ChoiceBlock) {
	if len(n.Options) == 0 {
		v.addError("choice block must have at least one option", n.GetSpan())
	}
	for _, opt := range n.Options {
		v.validateExpression(opt.Label)
		v.pushScope(scopeBlock)
		for _, stmt := range opt.Body {
			v.validateStatement(stmt)
		}
		v.popScope()
	}
}

func (v *Validator) validateIfStatement(n *ast.IfStatement) {
	// then body
	v.pushScope(scopeBlock)
	for _, stmt := range n.ThenBody {
		v.validateStatement(stmt)
	}
	v.popScope()

	// elif clauses
	for _, elif := range n.ElseIfClauses {
		v.pushScope(scopeBlock)
		for _, stmt := range elif.Body {
			v.validateStatement(stmt)
		}
		v.popScope()
	}

	// else body
	if n.ElseBody != nil {
		v.pushScope(scopeBlock)
		for _, stmt := range n.ElseBody {
			v.validateStatement(stmt)
		}
		v.popScope()
	}
}

func (v *Validator) validateSkillInvocation(n *ast.SkillInvocation) {
	for _, param := range n.Params {
		v.validateExpression(param.Value)
	}
	if n.OutputVar != nil {
		v.declareVar(n.OutputVar.Name, false, n.OutputVar.GetSpan())
	}
}

func (v *Validator) validateExpression(expr ast.Node) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *ast.Identifier:
		// Check variable reference.
		if v.lookupVar(e.Name) == nil {
			// Allow agent names, block names, and some well-known names.
			if _, ok := v.agents[e.Name]; !ok {
				if _, ok := v.blocks[e.Name]; !ok {
					if !isBuiltinName(e.Name) {
						// Don't error — could be a forward reference or property value.
						// The validator is conservative here.
					}
				}
			}
		}
	case *ast.InterpolatedString:
		for _, part := range e.Parts {
			v.validateExpression(part)
		}
	case *ast.ArrayExpression:
		for _, elem := range e.Elements {
			v.validateExpression(elem)
		}
	case *ast.ObjectExpression:
		for _, prop := range e.Properties {
			v.validateExpression(prop.Value)
		}
	case *ast.PipeExpression:
		v.validateExpression(e.Input)
		for _, op := range e.Operations {
			v.pushScope(scopeBlock)
			if op.ItemVar != nil {
				v.declareVar(op.ItemVar.Name, false, op.ItemVar.GetSpan())
			}
			if op.AccVar != nil {
				v.declareVar(op.AccVar.Name, false, op.AccVar.GetSpan())
			}
			for _, stmt := range op.Body {
				v.validateStatement(stmt)
			}
			v.popScope()
		}
	case *ast.ArrowExpression:
		v.validateExpression(e.Left)
		v.validateExpression(e.Right)
	case *ast.SessionStatement:
		v.validateSessionStatement(e)
	case *ast.StringLiteral, *ast.NumberLiteral, *ast.Discretion:
		// Leaf expressions — no further validation.
	}
}

// nodeStringValue extracts a string value from a string literal or identifier node.
func nodeStringValue(n ast.Node) string {
	switch e := n.(type) {
	case *ast.StringLiteral:
		return e.Value
	case *ast.Identifier:
		return e.Name
	case *ast.InterpolatedString:
		return e.Value
	}
	return ""
}

// isBuiltinName returns true for names that are always available.
func isBuiltinName(name string) bool {
	switch name {
	case "true", "false", "null", "nil", "item", "index", "acc",
		"opus", "sonnet", "haiku",
		"all", "first", "any",
		"fail-fast", "continue", "ignore":
		return true
	}
	return false
}
