package ast

import "github.com/ai-gateway/clawfirm/whipflow/token"

// Node is the interface that all AST nodes implement.
type Node interface {
	nodeType() string
	GetSpan() token.SourceSpan
}

// ---------------------------------------------------------------------------
// Literals
// ---------------------------------------------------------------------------

// StringLiteral represents a plain string literal.
type StringLiteral struct {
	Span            token.SourceSpan
	Value           string
	Raw             string
	IsTripleQuoted  bool
	EscapeSequences []token.EscapeSequenceInfo
}

func (n *StringLiteral) nodeType() string          { return "StringLiteral" }
func (n *StringLiteral) GetSpan() token.SourceSpan { return n.Span }

// InterpolatedString represents a string containing interpolated expressions.
type InterpolatedString struct {
	Span            token.SourceSpan
	Parts           []Node // StringLiteral or Identifier
	Raw             string
	IsTripleQuoted  bool
	Value           string
	EscapeSequences []token.EscapeSequenceInfo
}

func (n *InterpolatedString) nodeType() string          { return "InterpolatedString" }
func (n *InterpolatedString) GetSpan() token.SourceSpan { return n.Span }

// NumberLiteral represents a numeric literal.
type NumberLiteral struct {
	Span  token.SourceSpan
	Value float64
	Raw   string
}

func (n *NumberLiteral) nodeType() string          { return "NumberLiteral" }
func (n *NumberLiteral) GetSpan() token.SourceSpan { return n.Span }

// Identifier represents a named reference.
type Identifier struct {
	Span token.SourceSpan
	Name string
}

func (n *Identifier) nodeType() string          { return "Identifier" }
func (n *Identifier) GetSpan() token.SourceSpan { return n.Span }

// Discretion represents a natural-language expression used as a condition or criteria.
type Discretion struct {
	Span        token.SourceSpan
	Expression  string
	IsMultiline bool
}

func (n *Discretion) nodeType() string          { return "Discretion" }
func (n *Discretion) GetSpan() token.SourceSpan { return n.Span }

// ---------------------------------------------------------------------------
// Root
// ---------------------------------------------------------------------------

// Program is the top-level AST node representing an entire source file.
type Program struct {
	Span       token.SourceSpan
	Statements []Node
	Comments   []*Comment
}

func (n *Program) nodeType() string          { return "Program" }
func (n *Program) GetSpan() token.SourceSpan { return n.Span }

// Comment represents a comment in the source.
type Comment struct {
	Span     token.SourceSpan
	Value    string
	IsInline bool
}

func (n *Comment) nodeType() string          { return "Comment" }
func (n *Comment) GetSpan() token.SourceSpan { return n.Span }

// CommentStatement wraps a Comment so it can appear as a statement.
type CommentStatement struct {
	Span    token.SourceSpan
	Comment *Comment
}

func (n *CommentStatement) nodeType() string          { return "CommentStatement" }
func (n *CommentStatement) GetSpan() token.SourceSpan { return n.Span }

// ---------------------------------------------------------------------------
// Statements
// ---------------------------------------------------------------------------

// SessionStatement represents a session declaration.
type SessionStatement struct {
	Span          token.SourceSpan
	Prompt        Node // StringLiteral, InterpolatedString, or nil
	Agent         *Identifier
	Name          *Identifier
	Properties    []*Property
	InlineComment *Comment
}

func (n *SessionStatement) nodeType() string          { return "SessionStatement" }
func (n *SessionStatement) GetSpan() token.SourceSpan { return n.Span }

// LetBinding represents a let variable binding.
type LetBinding struct {
	Span  token.SourceSpan
	Name  *Identifier
	Value Node
}

func (n *LetBinding) nodeType() string          { return "LetBinding" }
func (n *LetBinding) GetSpan() token.SourceSpan { return n.Span }

// ConstBinding represents a const variable binding.
type ConstBinding struct {
	Span  token.SourceSpan
	Name  *Identifier
	Value Node
}

func (n *ConstBinding) nodeType() string          { return "ConstBinding" }
func (n *ConstBinding) GetSpan() token.SourceSpan { return n.Span }

// Assignment represents a variable reassignment.
type Assignment struct {
	Span  token.SourceSpan
	Name  *Identifier
	Value Node
}

func (n *Assignment) nodeType() string          { return "Assignment" }
func (n *Assignment) GetSpan() token.SourceSpan { return n.Span }

// AskStatement represents a prompt to the user and stores the response.
type AskStatement struct {
	Span     token.SourceSpan
	Variable *Identifier
	Prompt   Node
}

func (n *AskStatement) nodeType() string          { return "AskStatement" }
func (n *AskStatement) GetSpan() token.SourceSpan { return n.Span }

// RunStatement represents execution of an external file.
type RunStatement struct {
	Span     token.SourceSpan
	FilePath Node
}

func (n *RunStatement) nodeType() string          { return "RunStatement" }
func (n *RunStatement) GetSpan() token.SourceSpan { return n.Span }

// SkillInvocation represents a call to a named skill.
type SkillInvocation struct {
	Span      token.SourceSpan
	SkillName *Identifier
	Params    []*SkillParam
	OutputVar *Identifier
}

func (n *SkillInvocation) nodeType() string          { return "SkillInvocation" }
func (n *SkillInvocation) GetSpan() token.SourceSpan { return n.Span }

// ImportStatement represents importing a skill from an external source.
type ImportStatement struct {
	Span      token.SourceSpan
	SkillName Node
	Source    Node
}

func (n *ImportStatement) nodeType() string          { return "ImportStatement" }
func (n *ImportStatement) GetSpan() token.SourceSpan { return n.Span }

// ThrowStatement represents throwing an error.
type ThrowStatement struct {
	Span    token.SourceSpan
	Message Node // can be nil
}

func (n *ThrowStatement) nodeType() string          { return "ThrowStatement" }
func (n *ThrowStatement) GetSpan() token.SourceSpan { return n.Span }

// ReturnStatement represents returning a value from a block or agent.
type ReturnStatement struct {
	Span  token.SourceSpan
	Value Node // can be nil
}

func (n *ReturnStatement) nodeType() string          { return "ReturnStatement" }
func (n *ReturnStatement) GetSpan() token.SourceSpan { return n.Span }

// ---------------------------------------------------------------------------
// Definitions
// ---------------------------------------------------------------------------

// AgentDefinition represents the definition of an agent.
type AgentDefinition struct {
	Span       token.SourceSpan
	Name       *Identifier
	Properties []*Property
	Body       []Node
}

func (n *AgentDefinition) nodeType() string          { return "AgentDefinition" }
func (n *AgentDefinition) GetSpan() token.SourceSpan { return n.Span }

// BlockDefinition represents a reusable named block.
type BlockDefinition struct {
	Span       token.SourceSpan
	Name       *Identifier
	Parameters []*Identifier
	Body       []Node
}

func (n *BlockDefinition) nodeType() string          { return "BlockDefinition" }
func (n *BlockDefinition) GetSpan() token.SourceSpan { return n.Span }

// ---------------------------------------------------------------------------
// Blocks
// ---------------------------------------------------------------------------

// DoBlock represents a do-block invocation of a named block.
type DoBlock struct {
	Span      token.SourceSpan
	Name      *Identifier
	Arguments []Node
	Body      []Node
}

func (n *DoBlock) nodeType() string          { return "DoBlock" }
func (n *DoBlock) GetSpan() token.SourceSpan { return n.Span }

// ParallelBlock represents parallel execution of its body statements.
type ParallelBlock struct {
	Span         token.SourceSpan
	JoinStrategy Node
	AnyCount     *NumberLiteral
	OnFail       Node
	Body         []Node
}

func (n *ParallelBlock) nodeType() string          { return "ParallelBlock" }
func (n *ParallelBlock) GetSpan() token.SourceSpan { return n.Span }

// LoopBlock represents a loop, until, or while construct.
type LoopBlock struct {
	Span          token.SourceSpan
	Variant       string // "loop", "until", or "while"
	Condition     *Discretion
	IterationVar  *Identifier
	MaxIterations *NumberLiteral
	Body          []Node
}

func (n *LoopBlock) nodeType() string          { return "LoopBlock" }
func (n *LoopBlock) GetSpan() token.SourceSpan { return n.Span }

// RepeatBlock represents repeating a body a fixed number of times.
type RepeatBlock struct {
	Span     token.SourceSpan
	Count    Node // NumberLiteral or Identifier
	IndexVar *Identifier
	Body     []Node
}

func (n *RepeatBlock) nodeType() string          { return "RepeatBlock" }
func (n *RepeatBlock) GetSpan() token.SourceSpan { return n.Span }

// ForEachBlock represents iteration over a collection.
type ForEachBlock struct {
	Span       token.SourceSpan
	ItemVar    *Identifier
	IndexVar   *Identifier
	Collection Node
	IsParallel bool
	Modifiers  []*Property
	Body       []Node
}

func (n *ForEachBlock) nodeType() string          { return "ForEachBlock" }
func (n *ForEachBlock) GetSpan() token.SourceSpan { return n.Span }

// TryBlock represents a try/catch/finally construct.
type TryBlock struct {
	Span        token.SourceSpan
	TryBody     []Node
	CatchBody   []Node
	FinallyBody []Node
	ErrorVar    *Identifier
}

func (n *TryBlock) nodeType() string          { return "TryBlock" }
func (n *TryBlock) GetSpan() token.SourceSpan { return n.Span }

// ChoiceBlock represents a choice among labeled options.
type ChoiceBlock struct {
	Span     token.SourceSpan
	Criteria *Discretion
	Options  []*ChoiceOption
}

func (n *ChoiceBlock) nodeType() string          { return "ChoiceBlock" }
func (n *ChoiceBlock) GetSpan() token.SourceSpan { return n.Span }

// IfStatement represents conditional execution.
type IfStatement struct {
	Span          token.SourceSpan
	Condition     *Discretion
	ThenBody      []Node
	ElseIfClauses []*ElseIfClause
	ElseBody      []Node
}

func (n *IfStatement) nodeType() string          { return "IfStatement" }
func (n *IfStatement) GetSpan() token.SourceSpan { return n.Span }

// ---------------------------------------------------------------------------
// Supporting types
// ---------------------------------------------------------------------------

// Property represents a key-value property (used in agents, sessions, etc.).
type Property struct {
	Span  token.SourceSpan
	Name  *Identifier
	Value Node
}

func (n *Property) nodeType() string          { return "Property" }
func (n *Property) GetSpan() token.SourceSpan { return n.Span }

// ChoiceOption represents a single option within a ChoiceBlock.
type ChoiceOption struct {
	Span  token.SourceSpan
	Label Node
	Body  []Node
}

func (n *ChoiceOption) nodeType() string          { return "ChoiceOption" }
func (n *ChoiceOption) GetSpan() token.SourceSpan { return n.Span }

// ElseIfClause represents an else-if branch in an IfStatement.
type ElseIfClause struct {
	Span      token.SourceSpan
	Condition *Discretion
	Body      []Node
}

func (n *ElseIfClause) nodeType() string          { return "ElseIfClause" }
func (n *ElseIfClause) GetSpan() token.SourceSpan { return n.Span }

// SkillParam represents a named parameter passed to a skill invocation.
type SkillParam struct {
	Span  token.SourceSpan
	Name  *Identifier
	Value Node
}

func (n *SkillParam) nodeType() string          { return "SkillParam" }
func (n *SkillParam) GetSpan() token.SourceSpan { return n.Span }

// ---------------------------------------------------------------------------
// Expressions
// ---------------------------------------------------------------------------

// ArrayExpression represents an array literal.
type ArrayExpression struct {
	Span     token.SourceSpan
	Elements []Node
}

func (n *ArrayExpression) nodeType() string          { return "ArrayExpression" }
func (n *ArrayExpression) GetSpan() token.SourceSpan { return n.Span }

// ObjectExpression represents an object literal.
type ObjectExpression struct {
	Span       token.SourceSpan
	Properties []*Property
}

func (n *ObjectExpression) nodeType() string          { return "ObjectExpression" }
func (n *ObjectExpression) GetSpan() token.SourceSpan { return n.Span }

// PipeExpression represents a pipeline of operations on an input.
type PipeExpression struct {
	Span       token.SourceSpan
	Input      Node
	Operations []*PipeOperation
}

func (n *PipeExpression) nodeType() string          { return "PipeExpression" }
func (n *PipeExpression) GetSpan() token.SourceSpan { return n.Span }

// PipeOperation represents a single operation in a pipeline.
type PipeOperation struct {
	Span     token.SourceSpan
	Operator string // "map", "filter", "reduce", or "pmap"
	AccVar   *Identifier
	ItemVar  *Identifier
	Body     []Node
}

func (n *PipeOperation) nodeType() string          { return "PipeOperation" }
func (n *PipeOperation) GetSpan() token.SourceSpan { return n.Span }

// ArrowExpression represents a left -> right arrow expression.
type ArrowExpression struct {
	Span  token.SourceSpan
	Left  Node
	Right Node
}

func (n *ArrowExpression) nodeType() string          { return "ArrowExpression" }
func (n *ArrowExpression) GetSpan() token.SourceSpan { return n.Span }
