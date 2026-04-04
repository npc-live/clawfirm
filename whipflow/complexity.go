package whipflow

import "github.com/ai-gateway/clawfirm/whipflow/ast"

// ComplexityTier classifies a WhipFlow program's complexity.
type ComplexityTier string

const (
	TierSimple  ComplexityTier = "simple"
	TierMedium  ComplexityTier = "medium"
	TierComplex ComplexityTier = "complex"
)

// ComplexityAnalysis holds the result of analyzing a program's complexity.
type ComplexityAnalysis struct {
	Tier         ComplexityTier `json:"tier"`
	SessionCount int            `json:"session_count"`
	HasParallel  bool           `json:"has_parallel"`
	HasLoops     bool           `json:"has_loops"`
	HasChoice    bool           `json:"has_choice"`
	HasAsk       bool           `json:"has_ask"`
	Steps        []StepPreview  `json:"steps"`
}

// StepPreview is a summary of a single session in the program.
type StepPreview struct {
	Index  int    `json:"index"`
	Name   string `json:"name,omitempty"`
	Agent  string `json:"agent,omitempty"`
	Prompt string `json:"prompt"`
}

// AnalyzeComplexity walks the AST and classifies the program's complexity.
func AnalyzeComplexity(program *ast.Program) *ComplexityAnalysis {
	a := &ComplexityAnalysis{}
	walkNodes(program.Statements, a)

	// Classify tier.
	switch {
	case a.SessionCount <= 1 && !a.HasAsk && !a.HasParallel && !a.HasLoops && !a.HasChoice:
		a.Tier = TierSimple
	case a.SessionCount >= 2 && a.SessionCount <= 4 && !a.HasParallel && !a.HasLoops:
		a.Tier = TierMedium
	default:
		a.Tier = TierComplex
	}

	return a
}

func walkNodes(nodes []ast.Node, a *ComplexityAnalysis) {
	// Track pending variable name from let/const bindings.
	// In WhipFlow, `let x = session "..."` parses as a LetBinding (nil value)
	// followed by a SessionStatement — the binding captures the session result.
	var pendingName string
	for _, n := range nodes {
		switch v := n.(type) {
		case *ast.LetBinding:
			if v.Value == nil {
				pendingName = v.Name.Name
				continue
			}
			pendingName = ""
		case *ast.ConstBinding:
			if v.Value == nil {
				pendingName = v.Name.Name
				continue
			}
			pendingName = ""
		case *ast.SessionStatement:
			walkSession(v, pendingName, a)
			pendingName = ""
			continue
		default:
			pendingName = ""
		}
		walkNode(n, a)
	}
}

func walkSession(v *ast.SessionStatement, bindingName string, a *ComplexityAnalysis) {
	step := StepPreview{
		Index: a.SessionCount,
	}
	// Use the binding name (from `let x = session`) or the session's own Name field.
	if bindingName != "" {
		step.Name = bindingName
	} else if v.Name != nil {
		step.Name = v.Name.Name
	}
	if v.Agent != nil {
		step.Agent = v.Agent.Name
	}
	step.Prompt = extractPromptText(v.Prompt)
	a.Steps = append(a.Steps, step)
	a.SessionCount++
}

func walkNode(n ast.Node, a *ComplexityAnalysis) {
	switch v := n.(type) {
	case *ast.AskStatement:
		a.HasAsk = true

	case *ast.ParallelBlock:
		a.HasParallel = true
		walkNodes(v.Body, a)

	case *ast.LoopBlock:
		a.HasLoops = true
		walkNodes(v.Body, a)

	case *ast.RepeatBlock:
		a.HasLoops = true
		walkNodes(v.Body, a)

	case *ast.ForEachBlock:
		a.HasLoops = true
		walkNodes(v.Body, a)

	case *ast.ChoiceBlock:
		a.HasChoice = true
		for _, opt := range v.Options {
			walkNodes(opt.Body, a)
		}

	case *ast.IfStatement:
		walkNodes(v.ThenBody, a)
		for _, elif := range v.ElseIfClauses {
			walkNodes(elif.Body, a)
		}
		walkNodes(v.ElseBody, a)

	case *ast.TryBlock:
		walkNodes(v.TryBody, a)
		walkNodes(v.CatchBody, a)
		walkNodes(v.FinallyBody, a)

	case *ast.DoBlock:
		walkNodes(v.Body, a)

	case *ast.BlockDefinition:
		walkNodes(v.Body, a)

	case *ast.AgentDefinition:
		walkNodes(v.Body, a)
	}
}

// extractPromptText returns the first 120 characters of a prompt node's text.
func extractPromptText(n ast.Node) string {
	var text string
	switch v := n.(type) {
	case *ast.StringLiteral:
		text = v.Value
	case *ast.InterpolatedString:
		text = v.Raw
	default:
		return ""
	}
	if len(text) > 120 {
		return text[:120] + "..."
	}
	return text
}
