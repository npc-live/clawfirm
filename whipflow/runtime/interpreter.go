package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/whipflow/ast"
	"github.com/ai-gateway/clawfirm/whipflow/parser"
)

// ---------------------------------------------------------------------------
// Interpreter
// ---------------------------------------------------------------------------

// SessionProgress is emitted by the interpreter before and after each session.
type SessionProgress struct {
	// Index is the 0-based session counter.
	Index int
	// Name is the session name (empty for unnamed sessions).
	Name string
	// Provider is the resolved provider name.
	Provider string
	// Prompt is the rendered prompt sent to the provider.
	Prompt string
	// Done is false when the session starts, true when it finishes.
	Done bool
	// Output is non-empty only when Done == true.
	Output string
	// DurationMs is set when Done == true.
	DurationMs int64
	// Error is set when Done == true and the session failed.
	Error string
}

// Interpreter executes WhipFlow programs.
type Interpreter struct {
	env          *RuntimeEnvironment
	toolRegistry *ToolRegistry
	blocks       map[string]*ast.BlockDefinition
	providerCache map[string]Provider

	// clawfirm config for resolving providers from ~/.clawfirm/config.yml.
	piConfig *config.Config

	// Native provider registry (injected from outside, e.g. app integration).
	nativeProviders map[string]Provider

	// Pre-filled answers for ask statements (bypasses stdin).
	initialInputs map[string]string

	// State persistence
	stateStore     *StateStore
	currentRunID   int64
	sessionIndex   int
	replaySessions []SessionRecord

	// Context tracking
	executionEvents      []ExecutionEvent
	currentFileName      string
	currentBlockStack    []string
	totalStatements      int
	executedStatements   int
	recentSessionOutputs []string
	allSessionOutputs    []*SessionResult
	loopContext          *LoopInfo

	// Progress callback — called before (Done=false) and after (Done=true) each session.
	onSessionProgress func(SessionProgress)
}

// InterpreterOption configures an Interpreter during construction.
type InterpreterOption func(*Interpreter)

// WithToolRegistry sets a custom tool registry.
func WithToolRegistry(tr *ToolRegistry) InterpreterOption {
	return func(i *Interpreter) { i.toolRegistry = tr }
}

// WithStateStore sets a state store for persistence.
func WithStateStore(ss *StateStore) InterpreterOption {
	return func(i *Interpreter) { i.stateStore = ss }
}

// WithReplaySessions sets previously recorded sessions for replay.
func WithReplaySessions(sessions []SessionRecord) InterpreterOption {
	return func(i *Interpreter) { i.replaySessions = sessions }
}

// WithFileName sets the current file name for context tracking.
func WithFileName(name string) InterpreterOption {
	return func(i *Interpreter) { i.currentFileName = name }
}

// WithSessionProgressCallback registers a callback invoked before (Done=false)
// and after (Done=true) each session execution.
func WithSessionProgressCallback(cb func(SessionProgress)) InterpreterOption {
	return func(i *Interpreter) { i.onSessionProgress = cb }
}

// WithPiConfig sets the clawfirm config for resolving providers from ~/.clawfirm/config.yml.
func WithPiConfig(cfg *config.Config) InterpreterOption {
	return func(i *Interpreter) { i.piConfig = cfg }
}

// WithNativeProviders registers pre-built native providers by name.
func WithNativeProviders(providers map[string]Provider) InterpreterOption {
	return func(i *Interpreter) { i.nativeProviders = providers }
}

// WithInitialInputs pre-fills answers for ask statements, bypassing stdin.
// Keys are the variable names declared in each ask statement.
func WithInitialInputs(inputs map[string]string) InterpreterOption {
	return func(i *Interpreter) { i.initialInputs = inputs }
}

// NewInterpreter creates a new interpreter with the given environment and options.
func NewInterpreter(env *RuntimeEnvironment, opts ...InterpreterOption) *Interpreter {
	interp := &Interpreter{
		env:           env,
		toolRegistry:  NewToolRegistry(),
		blocks:        make(map[string]*ast.BlockDefinition),
		providerCache: make(map[string]Provider),
	}
	for _, opt := range opts {
		opt(interp)
	}
	return interp
}

// ---------------------------------------------------------------------------
// Execute — top-level entry point
// ---------------------------------------------------------------------------

// Execute runs a WhipFlow program and returns the execution result.
func (interp *Interpreter) Execute(program *ast.Program) (*ExecutionResult, error) {
	interp.env.StartExecution()
	interp.executionEvents = nil
	interp.sessionIndex = 0
	interp.executedStatements = 0
	interp.totalStatements = len(program.Statements)
	interp.allSessionOutputs = nil
	interp.recentSessionOutputs = nil

	// Start a run record if state persistence is enabled.
	if interp.stateStore != nil {
		runID, err := interp.stateStore.StartRun(interp.currentFileName)
		if err != nil {
			return nil, fmt.Errorf("interpreter: start run: %w", err)
		}
		interp.currentRunID = runID

		// Re-persist any replay sessions so they appear in the new run.
		for idx, rec := range interp.replaySessions {
			result := &SessionResult{
				Output: rec.Output,
				Metadata: SessionMetadata{
					Model:    rec.Model,
					Duration: rec.DurationMs,
				},
			}
			vars := make(map[string]any)
			if rec.VariablesJSON != "" {
				_ = json.Unmarshal([]byte(rec.VariablesJSON), &vars)
			}
			_ = interp.stateStore.RecordSession(runID, idx, rec.Prompt, result, vars)
		}
	}

	// Execute each top-level statement.
	// Wrap in a closure to recover ControlFlow panics (throw/return) that
	// escape without a surrounding try-catch or block call.
	var execErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				if cf, ok := r.(ControlFlow); ok {
					switch cf.Type {
					case "throw":
						if cf.Err != nil {
							execErr = cf.Err
						} else {
							execErr = fmt.Errorf("%v", cf.Value)
						}
					case "return":
						// Top-level return — not an error, just stop execution.
					default:
						// break/continue at top level is a programming error.
						execErr = fmt.Errorf("unexpected %s outside loop", cf.Type)
					}
					return
				}
				panic(r) // re-panic for non-ControlFlow (real bugs)
			}
		}()

		for _, stmt := range program.Statements {
			if interp.env.HasTimedOut() {
				execErr = fmt.Errorf("execution timed out after %dms", interp.env.Config.TotalExecutionTimeout)
				return
			}
			if err := interp.executeStatement(stmt); err != nil {
				execErr = err
				return
			}
		}
	}()

	// Build the execution result.
	ctx := interp.env.Context.CaptureContext()
	result := &ExecutionResult{
		Success:        execErr == nil && !interp.env.HasErrors(),
		Outputs:        ctx.Variables,
		SessionOutputs: interp.allSessionOutputs,
		FinalContext:    ctx,
		Errors:         interp.env.GetErrors(),
		Metadata: ExecutionMetadata{
			Duration:           interp.env.GetExecutionDuration(),
			SessionsCreated:    interp.env.GetSessionCount(),
			StatementsExecuted: interp.executedStatements,
		},
	}

	// Finalize the run record.
	if interp.stateStore != nil {
		if execErr != nil {
			_ = interp.stateStore.FailRun(interp.currentRunID, execErr.Error())
		} else {
			_ = interp.stateStore.CompleteRun(interp.currentRunID)
		}
	}

	if execErr != nil {
		interp.env.AddError(ExecutionError{
			Type:    "runtime",
			Message: execErr.Error(),
		})
		result.Errors = interp.env.GetErrors()
	}

	return result, execErr
}

// ---------------------------------------------------------------------------
// executeStatement — the main dispatch
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeStatement(stmt ast.Node) error {
	interp.env.IncrementStatementCount()
	interp.executedStatements++
	interp.env.Trace(fmt.Sprintf("executing %T", stmt))

	switch n := stmt.(type) {

	case *ast.SessionStatement:
		return interp.executeSessionStatement(n)

	case *ast.LetBinding:
		val := interp.evaluateExpression(n.Value)
		return interp.env.Context.DeclareVariable(n.Name.Name, val, false, n.Span)

	case *ast.ConstBinding:
		val := interp.evaluateExpression(n.Value)
		return interp.env.Context.DeclareVariable(n.Name.Name, val, true, n.Span)

	case *ast.Assignment:
		val := interp.evaluateExpression(n.Value)
		if !interp.env.Context.HasVariable(n.Name.Name) {
			return interp.env.Context.DeclareVariable(n.Name.Name, val, false, n.Span)
		}
		return interp.env.Context.SetVariable(n.Name.Name, val)

	case *ast.AgentDefinition:
		return interp.executeAgentDefinition(n)

	case *ast.BlockDefinition:
		interp.blocks[n.Name.Name] = n
		return nil

	case *ast.DoBlock:
		return interp.executeDoBlock(n)

	case *ast.ParallelBlock:
		return interp.executeParallelBlock(n)

	case *ast.RepeatBlock:
		return interp.executeRepeatBlock(n)

	case *ast.ForEachBlock:
		return interp.executeForEachBlock(n)

	case *ast.LoopBlock:
		return interp.executeLoopBlock(n)

	case *ast.IfStatement:
		return interp.executeIfStatement(n)

	case *ast.ChoiceBlock:
		return interp.executeChoiceBlock(n)

	case *ast.TryBlock:
		return interp.executeTryBlock(n)

	case *ast.ThrowStatement:
		msg := "error"
		if n.Message != nil {
			msg = runtimeValueToString(interp.evaluateExpression(n.Message))
		}
		panic(ControlFlow{Type: "throw", Value: msg, Err: fmt.Errorf("%s", msg)})

	case *ast.ReturnStatement:
		var val RuntimeValue
		if n.Value != nil {
			val = interp.evaluateExpression(n.Value)
		}
		panic(ControlFlow{Type: "return", Value: val})

	case *ast.ImportStatement:
		// No-op for now; skill imports are not yet implemented.
		return nil

	case *ast.AskStatement:
		return interp.executeAskStatement(n)

	case *ast.RunStatement:
		return interp.executeRunStatement(n)

	case *ast.SkillInvocation:
		return interp.executeSkillInvocation(n)

	case *ast.CommentStatement:
		// Skip comments.
		return nil

	case *ast.ArrowExpression:
		interp.evaluateExpression(n)
		return nil

	case *ast.PipeExpression:
		interp.evaluateExpression(n)
		return nil

	default:
		return fmt.Errorf("interpreter: unknown statement type %T", stmt)
	}
}

// ---------------------------------------------------------------------------
// executeSessionStatement
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeSessionStatement(n *ast.SessionStatement) error {
	interp.env.Context.AddToExecutionPath("session")

	// Build the prompt string.
	prompt := ""
	if n.Prompt != nil {
		prompt = runtimeValueToString(interp.evaluateExpression(n.Prompt))
	}

	// Resolve the agent.
	var agent *AgentInstance
	if n.Agent != nil {
		a, ok := interp.env.GetAgent(n.Agent.Name)
		if !ok {
			return fmt.Errorf("interpreter: unknown agent '%s'", n.Agent.Name)
		}
		agent = a
	} else {
		// Use a default agent if none is specified.
		agent = &AgentInstance{
			Name:     "default",
			Provider: interp.env.Config.DefaultProvider,
		}
	}

	// Build the session spec.
	ctx := interp.env.Context.CaptureContext()
	props := make(map[string]RuntimeValue)
	for _, p := range n.Properties {
		val := interp.evaluateExpression(p.Value)
		props[p.Name.Name] = val
		switch p.Name.Name {
		case "prompt":
			// Allow `prompt: """..."""` property to set the session prompt.
			pv := runtimeValueToString(val)
			interp.env.Log("debug", fmt.Sprintf("session prompt property: len=%d value=%q", len(pv), pv[:min(100, len(pv))]))
			if pv != "" {
				prompt = pv
			}
		case "provider":
			// Allow inline `with provider "..."` to override the agent's provider.
			if pv := runtimeValueToString(val); pv != "" {
				if agent == nil {
					agent = &AgentInstance{Name: "default"}
				}
				agent.Provider = pv
			}
		}
	}

	spec := SessionSpec{
		Agent:      agent,
		Prompt:     prompt,
		Context:    ctx,
		Properties: props,
	}
	if n.Name != nil {
		spec.Name = n.Name.Name
	}

	// Resolve provider name for progress reporting.
	providerName := interp.env.Config.DefaultProvider
	if spec.Agent != nil && spec.Agent.Provider != "" {
		providerName = spec.Agent.Provider
	}

	// Emit session-start progress.
	if interp.onSessionProgress != nil {
		interp.onSessionProgress(SessionProgress{
			Index:    interp.sessionIndex,
			Name:     spec.Name,
			Provider: providerName,
			Prompt:   prompt,
			Done:     false,
		})
	}

	// Check for replay.
	var result *SessionResult
	if interp.sessionIndex < len(interp.replaySessions) {
		rec := interp.replaySessions[interp.sessionIndex]
		interp.env.Log("info", fmt.Sprintf("replaying session %d", interp.sessionIndex))
		result = &SessionResult{
			Output: rec.Output,
			Metadata: SessionMetadata{
				Model:    rec.Model,
				Duration: rec.DurationMs,
			},
		}
	} else {
		var err error
		result, err = interp.executeSession(spec)
		if err != nil {
			// Emit session-error progress.
			if interp.onSessionProgress != nil {
				interp.onSessionProgress(SessionProgress{
					Index:    interp.sessionIndex,
					Name:     spec.Name,
					Provider: providerName,
					Prompt:   prompt,
					Done:     true,
					Error:    err.Error(),
				})
			}
			return fmt.Errorf("interpreter: session execution failed: %w", err)
		}
	}

	// Emit session-done progress.
	if interp.onSessionProgress != nil {
		interp.onSessionProgress(SessionProgress{
			Index:      interp.sessionIndex,
			Name:       spec.Name,
			Provider:   providerName,
			Prompt:     prompt,
			Done:       true,
			Output:     result.Output,
			DurationMs: result.Metadata.Duration,
		})
	}

	interp.env.IncrementSessionCount()
	interp.allSessionOutputs = append(interp.allSessionOutputs, result)

	// Keep recent session outputs for context enrichment.
	interp.recentSessionOutputs = append(interp.recentSessionOutputs, result.Output)
	if len(interp.recentSessionOutputs) > 5 {
		interp.recentSessionOutputs = interp.recentSessionOutputs[len(interp.recentSessionOutputs)-5:]
	}

	// Record execution event.
	interp.executionEvents = append(interp.executionEvents, ExecutionEvent{
		Type:        "session",
		Description: fmt.Sprintf("session '%s'", spec.Name),
		Timestamp:   time.Now().UnixMilli(),
		Result:      result.Output,
	})

	// Store the result as a variable if the session is named.
	if n.Name != nil {
		if err := interp.env.Context.DeclareVariable(n.Name.Name, result, false, n.Span); err != nil {
			// If the variable already exists, update it.
			_ = interp.env.Context.SetVariable(n.Name.Name, result)
		}
	}

	// Also store as "$last" for implicit access.
	if interp.env.Context.HasVariable("$last") {
		_ = interp.env.Context.SetVariable("$last", result)
	} else {
		_ = interp.env.Context.DeclareVariable("$last", result, false, n.Span)
	}

	// Persist to state store.
	if interp.stateStore != nil {
		vars := interp.env.Context.GetAllVariables()
		_ = interp.stateStore.RecordSession(interp.currentRunID, interp.sessionIndex, prompt, result, vars)
	}

	interp.sessionIndex++
	return nil
}

// ---------------------------------------------------------------------------
// executeSession — call the actual provider
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeSession(spec SessionSpec) (*SessionResult, error) {
	// Determine the provider name.
	providerName := interp.env.Config.DefaultProvider
	if spec.Agent != nil && spec.Agent.Provider != "" {
		providerName = spec.Agent.Provider
	}

	// Get or create a cached provider (CLI or Native).
	prov, err := interp.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	// Collect allowed tools.
	var allowedTools []string
	if spec.Agent != nil && len(spec.Agent.Tools) > 0 {
		allowedTools = spec.Agent.Tools
	}

	// Resolve skill prompts from the agent's skills list.
	var skillPrompts []string
	if spec.Agent != nil && len(spec.Agent.Skills) > 0 && interp.env.Config.SkillResolver != nil {
		for _, name := range spec.Agent.Skills {
			content, err := interp.env.Config.SkillResolver(name)
			if err != nil {
				interp.env.Log("warn", fmt.Sprintf("skill %q: %v", name, err))
				continue
			}
			skillPrompts = append(skillPrompts, content)
		}
	}

	enableTools := len(allowedTools) > 0

	result, err := prov.ExecuteSession(spec, interp.env.Config, enableTools, allowedTools, skillPrompts)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// getProvider returns a cached Provider or creates a new one.
// Resolution order: cache → nativeProviders registry → ResolveProvider (config files → presets → custom).
func (interp *Interpreter) getProvider(name string) (Provider, error) {
	if p, ok := interp.providerCache[name]; ok {
		return p, nil
	}
	p, err := ResolveProvider(name, interp.piConfig, interp.nativeProviders)
	if err != nil {
		return nil, fmt.Errorf("interpreter: create provider '%s': %w", name, err)
	}
	interp.providerCache[name] = p
	return p, nil
}

// ---------------------------------------------------------------------------
// evaluateExpression
// ---------------------------------------------------------------------------

func (interp *Interpreter) evaluateExpression(expr ast.Node) RuntimeValue {
	if expr == nil {
		return nil
	}

	switch n := expr.(type) {

	case *ast.StringLiteral:
		return n.Value

	case *ast.InterpolatedString:
		return interp.interpolateString(n.Parts)

	case *ast.NumberLiteral:
		return n.Value

	case *ast.Identifier:
		val, err := interp.env.Context.GetVariable(n.Name)
		if err != nil {
			// Return the identifier name as a string if not found.
			return n.Name
		}
		return val

	case *ast.ArrayExpression:
		result := make([]any, len(n.Elements))
		for i, el := range n.Elements {
			result[i] = interp.evaluateExpression(el)
		}
		return result

	case *ast.ObjectExpression:
		result := make(map[string]any, len(n.Properties))
		for _, p := range n.Properties {
			result[p.Name.Name] = interp.evaluateExpression(p.Value)
		}
		return result

	case *ast.Discretion:
		return n.Expression

	case *ast.PipeExpression:
		return interp.evaluatePipeExpression(n)

	case *ast.ArrowExpression:
		// Execute left, then right. The left result becomes "$last".
		leftVal := interp.evaluateExpression(n.Left)
		if interp.env.Context.HasVariable("$last") {
			_ = interp.env.Context.SetVariable("$last", leftVal)
		} else {
			_ = interp.env.Context.DeclareVariable("$last", leftVal, false, n.Span)
		}
		return interp.evaluateExpression(n.Right)

	case *ast.SessionStatement:
		// Inline session expression.
		if err := interp.executeSessionStatement(n); err != nil {
			interp.env.AddError(ExecutionError{
				Type:    "runtime",
				Message: fmt.Sprintf("inline session error: %s", err),
			})
			return nil
		}
		// Return the most recent session result.
		if len(interp.allSessionOutputs) > 0 {
			return interp.allSessionOutputs[len(interp.allSessionOutputs)-1]
		}
		return nil

	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// evaluateCondition — uses an AI provider to evaluate a natural-language
// discretion expression as a boolean.
// ---------------------------------------------------------------------------

func (interp *Interpreter) evaluateCondition(d *ast.Discretion) bool {
	if d == nil {
		return true
	}

	// Build enriched context for condition evaluation.
	enriched := interp.buildEnrichedContext()

	// Build a prompt that asks the provider to evaluate the condition.
	var sb strings.Builder
	sb.WriteString("You are a condition evaluator. Based on the context below, determine whether the following condition is TRUE or FALSE.\n\n")
	sb.WriteString("## Condition\n")
	sb.WriteString(d.Expression)
	sb.WriteString("\n\n")

	sb.WriteString("## Execution Context\n")
	if enriched.FileName != "" {
		sb.WriteString(fmt.Sprintf("- File: %s\n", enriched.FileName))
	}
	if enriched.CurrentBlock != "" {
		sb.WriteString(fmt.Sprintf("- Current block: %s\n", enriched.CurrentBlock))
	}
	sb.WriteString(fmt.Sprintf("- Statements executed: %d/%d\n", enriched.ExecutedStatements, enriched.TotalStatements))

	if len(enriched.Variables) > 0 {
		sb.WriteString("\n## Variables\n")
		for name, value := range enriched.Variables {
			sb.WriteString(fmt.Sprintf("- %s = %s\n", name, runtimeValueToString(value)))
		}
	}

	if len(enriched.RecentSessionOutputs) > 0 {
		sb.WriteString("\n## Recent Session Outputs\n")
		for i, output := range enriched.RecentSessionOutputs {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, truncateString(output, 500)))
		}
	}

	if enriched.LoopInfo != nil {
		sb.WriteString(fmt.Sprintf("\n## Loop Info\n- Iteration: %d\n", enriched.LoopInfo.Iteration))
		if enriched.LoopInfo.MaxIterations != nil {
			sb.WriteString(fmt.Sprintf("- Max iterations: %d\n", *enriched.LoopInfo.MaxIterations))
		}
	}

	sb.WriteString("\nRespond with ONLY the word 'true' or 'false'. Nothing else.")

	prompt := sb.String()

	// Determine the provider to use for conditions.
	providerName := interp.env.Config.DefaultProvider
	if interp.env.Config.ConditionProvider != "" {
		providerName = interp.env.Config.ConditionProvider
	}

	provider, err := interp.getProvider(providerName)
	if err != nil {
		interp.env.Log("error", fmt.Sprintf("condition evaluation provider error: %s", err))
		return false
	}

	spec := SessionSpec{
		Agent: &AgentInstance{
			Name:     "condition-evaluator",
			Provider: providerName,
		},
		Prompt: prompt,
	}

	result, err := provider.ExecuteSession(spec, interp.env.Config, false, nil, nil)
	if err != nil {
		interp.env.Log("error", fmt.Sprintf("condition evaluation failed: %s", err))
		return false
	}

	// Record the condition evaluation event.
	interp.executionEvents = append(interp.executionEvents, ExecutionEvent{
		Type:        "condition",
		Description: d.Expression,
		Timestamp:   time.Now().UnixMilli(),
		Result:      result.Output,
	})

	// Parse the response.
	answer := strings.TrimSpace(strings.ToLower(result.Output))
	return answer == "true" || strings.HasPrefix(answer, "true")
}

// buildEnrichedContext constructs an EnrichedExecutionContext with the current state.
func (interp *Interpreter) buildEnrichedContext() EnrichedExecutionContext {
	currentBlock := ""
	if len(interp.currentBlockStack) > 0 {
		currentBlock = interp.currentBlockStack[len(interp.currentBlockStack)-1]
	}

	// Collect recent events (up to 10).
	recentEvents := interp.executionEvents
	if len(recentEvents) > 10 {
		recentEvents = recentEvents[len(recentEvents)-10:]
	}

	return EnrichedExecutionContext{
		FileName:             interp.currentFileName,
		CurrentBlock:         currentBlock,
		Variables:            interp.env.Context.GetAllVariables(),
		RecentEvents:         recentEvents,
		ExecutionPath:        interp.env.Context.GetExecutionPath(),
		TotalStatements:      interp.totalStatements,
		ExecutedStatements:   interp.executedStatements,
		RemainingStatements:  interp.totalStatements - interp.executedStatements,
		RecentSessionOutputs: interp.recentSessionOutputs,
		LoopInfo:             interp.loopContext,
	}
}

// ---------------------------------------------------------------------------
// executeAgentDefinition
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeAgentDefinition(n *ast.AgentDefinition) error {
	agent := &AgentInstance{
		Name: n.Name.Name,
	}

	for _, prop := range n.Properties {
		val := interp.evaluateExpression(prop.Value)
		switch prop.Name.Name {
		case "model":
			agent.Model = runtimeValueToString(val)
		case "provider":
			agent.Provider = runtimeValueToString(val)
		case "prompt":
			agent.Prompt = runtimeValueToString(val)
		case "skills":
			if arr, ok := val.([]any); ok {
				for _, s := range arr {
					agent.Skills = append(agent.Skills, runtimeValueToString(s))
				}
			}
		case "tools":
			if arr, ok := val.([]any); ok {
				for _, t := range arr {
					agent.Tools = append(agent.Tools, runtimeValueToString(t))
				}
			}
		case "permissions":
			if m, ok := val.(map[string]any); ok {
				if v, ok := m["bash"]; ok {
					agent.Permissions.Bash = runtimeValueToString(v)
				}
				if v, ok := m["file_read"]; ok {
					agent.Permissions.FileRead = runtimeValueToString(v)
				}
				if v, ok := m["file_write"]; ok {
					agent.Permissions.FileWrite = runtimeValueToString(v)
				}
				if v, ok := m["network"]; ok {
					agent.Permissions.Network = runtimeValueToString(v)
				}
			}
		}
	}

	// Process body statements if any (e.g. nested definitions).
	for _, bodyStmt := range n.Body {
		if err := interp.executeStatement(bodyStmt); err != nil {
			return err
		}
	}

	interp.env.RegisterAgent(agent)
	return nil
}

// ---------------------------------------------------------------------------
// executeDoBlock
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeDoBlock(n *ast.DoBlock) error {
	interp.env.Context.AddToExecutionPath("do")

	// If the do block references a named block definition, execute that.
	if n.Name != nil {
		blockDef, ok := interp.blocks[n.Name.Name]
		if ok {
			return interp.executeBlockCall(blockDef, n.Arguments)
		}
	}

	// Otherwise execute the inline body.
	interp.env.Context.PushScope()
	defer interp.env.Context.PopScope()

	interp.pushBlock("do")
	defer interp.popBlock()

	for _, stmt := range n.Body {
		if err := interp.executeStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

// executeBlockCall invokes a named block definition with arguments.
func (interp *Interpreter) executeBlockCall(block *ast.BlockDefinition, args []ast.Node) error {
	if err := interp.env.IncrementCallDepth(); err != nil {
		return err
	}
	defer interp.env.DecrementCallDepth()

	interp.env.Context.PushScope()
	defer interp.env.Context.PopScope()

	interp.pushBlock(block.Name.Name)
	defer interp.popBlock()

	// Bind parameters to arguments.
	for i, param := range block.Parameters {
		var val RuntimeValue
		if i < len(args) {
			val = interp.evaluateExpression(args[i])
		}
		if err := interp.env.Context.DeclareVariable(param.Name, val, false, param.Span); err != nil {
			return err
		}
	}

	// Execute the block body, catching return control flow.
	var returnValue RuntimeValue
	func() {
		defer func() {
			if r := recover(); r != nil {
				if cf, ok := r.(ControlFlow); ok && cf.Type == "return" {
					returnValue = cf.Value
					return
				}
				panic(r) // re-panic for non-return control flow
			}
		}()
		for _, stmt := range block.Body {
			if err := interp.executeStatement(stmt); err != nil {
				panic(ControlFlow{Type: "throw", Err: err})
			}
		}
	}()

	// Store return value as $last.
	if returnValue != nil {
		if interp.env.Context.HasVariable("$last") {
			_ = interp.env.Context.SetVariable("$last", returnValue)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// executeParallelBlock
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeParallelBlock(n *ast.ParallelBlock) error {
	interp.env.Context.AddToExecutionPath("parallel")
	interp.pushBlock("parallel")
	defer interp.popBlock()

	// Determine join strategy.
	joinStrategy := JoinAll
	if n.JoinStrategy != nil {
		stratStr := runtimeValueToString(interp.evaluateExpression(n.JoinStrategy))
		switch strings.ToLower(stratStr) {
		case "first":
			joinStrategy = JoinFirst
		case "any":
			joinStrategy = JoinAny
		default:
			joinStrategy = JoinAll
		}
	}

	anyCount := 0
	if n.AnyCount != nil {
		anyCount = int(n.AnyCount.Value)
	}

	// Determine failure strategy.
	failStrategy := FailFast
	if n.OnFail != nil {
		failStr := runtimeValueToString(interp.evaluateExpression(n.OnFail))
		switch strings.ToLower(failStr) {
		case "continue":
			failStrategy = FailContinue
		case "ignore":
			failStrategy = FailIgnore
		default:
			failStrategy = FailFast
		}
	}

	type parallelResult struct {
		index int
		err   error
	}

	results := make([]parallelResult, len(n.Body))
	var mu sync.Mutex
	var wg sync.WaitGroup

	doneCh := make(chan struct{})
	var completedCount int

	for i, stmt := range n.Body {
		wg.Add(1)
		go func(idx int, s ast.Node) {
			defer wg.Done()

			var execErr error
			func() {
				defer func() {
					if r := recover(); r != nil {
						if cf, ok := r.(ControlFlow); ok {
							execErr = fmt.Errorf("control flow in parallel: %s", cf.Type)
						} else {
							execErr = fmt.Errorf("panic in parallel branch: %v", r)
						}
					}
				}()
				execErr = interp.executeStatement(s)
			}()

			mu.Lock()
			results[idx] = parallelResult{index: idx, err: execErr}
			completedCount++
			currentCompleted := completedCount
			mu.Unlock()

			// Check if we have enough results for early exit.
			switch joinStrategy {
			case JoinFirst:
				if currentCompleted >= 1 {
					select {
					case doneCh <- struct{}{}:
					default:
					}
				}
			case JoinAny:
				if anyCount > 0 && currentCompleted >= anyCount {
					select {
					case doneCh <- struct{}{}:
					default:
					}
				}
			}
		}(i, stmt)
	}

	// Wait for all goroutines or early exit.
	if joinStrategy == JoinAll {
		wg.Wait()
	} else {
		go func() {
			wg.Wait()
			select {
			case doneCh <- struct{}{}:
			default:
			}
		}()
		<-doneCh
	}

	// Check for errors.
	if failStrategy == FailFast {
		for _, r := range results {
			if r.err != nil {
				return r.err
			}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// executeRepeatBlock
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeRepeatBlock(n *ast.RepeatBlock) error {
	interp.env.Context.AddToExecutionPath("repeat")
	interp.pushBlock("repeat")
	defer interp.popBlock()

	countVal := interp.evaluateExpression(n.Count)
	count := toInt(countVal)
	if count <= 0 {
		return nil
	}

	for i := 0; i < count; i++ {
		interp.env.Context.PushScope()

		// Declare the index variable if specified.
		if n.IndexVar != nil {
			_ = interp.env.Context.DeclareVariable(n.IndexVar.Name, float64(i), false, n.Span)
		}

		err := interp.executeBodyWithBreakContinue(n.Body)
		interp.env.Context.PopScope()

		if err != nil {
			if err.Error() == "break" {
				break
			}
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// executeForEachBlock
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeForEachBlock(n *ast.ForEachBlock) error {
	interp.env.Context.AddToExecutionPath("foreach")
	interp.pushBlock("foreach")
	defer interp.popBlock()

	collectionVal := interp.evaluateExpression(n.Collection)
	items := toSlice(collectionVal)

	if n.IsParallel {
		return interp.executeForEachParallel(n, items)
	}

	for i, item := range items {
		interp.env.Context.PushScope()

		if n.ItemVar != nil {
			_ = interp.env.Context.DeclareVariable(n.ItemVar.Name, item, false, n.Span)
		}
		if n.IndexVar != nil {
			_ = interp.env.Context.DeclareVariable(n.IndexVar.Name, float64(i), false, n.Span)
		}

		err := interp.executeBodyWithBreakContinue(n.Body)
		interp.env.Context.PopScope()

		if err != nil {
			if err.Error() == "break" {
				break
			}
			return err
		}
	}
	return nil
}

func (interp *Interpreter) executeForEachParallel(n *ast.ForEachBlock, items []any) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(items))

	for i, item := range items {
		wg.Add(1)
		go func(idx int, val any) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					if cf, ok := r.(ControlFlow); ok {
						errCh <- fmt.Errorf("control flow in parallel foreach: %s", cf.Type)
					} else {
						errCh <- fmt.Errorf("panic in parallel foreach: %v", r)
					}
				}
			}()

			// Note: parallel foreach shares the environment, which is inherently
			// racy. This is a known limitation; production use should use
			// isolated sub-interpreters.
			for _, stmt := range n.Body {
				if err := interp.executeStatement(stmt); err != nil {
					errCh <- err
					return
				}
			}
		}(i, item)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// executeLoopBlock
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeLoopBlock(n *ast.LoopBlock) error {
	interp.env.Context.AddToExecutionPath("loop")
	interp.pushBlock("loop")
	defer interp.popBlock()

	maxIter := interp.env.Config.MaxLoopIterations
	if n.MaxIterations != nil {
		maxIter = int(n.MaxIterations.Value)
	}

	previousLoopContext := interp.loopContext
	defer func() { interp.loopContext = previousLoopContext }()

	var previousResults []any

	for i := 0; i < maxIter; i++ {
		maxIterCopy := maxIter
		interp.loopContext = &LoopInfo{
			Iteration:       i,
			MaxIterations:   &maxIterCopy,
			PreviousResults: previousResults,
		}

		// Check condition based on variant.
		switch n.Variant {
		case "until":
			if n.Condition != nil && interp.evaluateCondition(n.Condition) {
				return nil // Condition met; exit loop.
			}
		case "while":
			if n.Condition != nil && !interp.evaluateCondition(n.Condition) {
				return nil // Condition no longer met; exit loop.
			}
		case "loop":
			// Infinite loop; relies on break or max iterations.
		}

		interp.env.Context.PushScope()

		if n.IterationVar != nil {
			_ = interp.env.Context.DeclareVariable(n.IterationVar.Name, float64(i), false, n.Span)
		}

		err := interp.executeBodyWithBreakContinue(n.Body)
		interp.env.Context.PopScope()

		if err != nil {
			if err.Error() == "break" {
				break
			}
			return err
		}

		// Collect the latest session output as a loop result if any.
		if len(interp.allSessionOutputs) > 0 {
			last := interp.allSessionOutputs[len(interp.allSessionOutputs)-1]
			previousResults = append(previousResults, last.Output)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// executeIfStatement
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeIfStatement(n *ast.IfStatement) error {
	interp.env.Context.AddToExecutionPath("if")

	// Evaluate the main condition.
	if interp.evaluateCondition(n.Condition) {
		return interp.executeBody(n.ThenBody)
	}

	// Check else-if clauses.
	for _, clause := range n.ElseIfClauses {
		if interp.evaluateCondition(clause.Condition) {
			return interp.executeBody(clause.Body)
		}
	}

	// Execute else body if present.
	if len(n.ElseBody) > 0 {
		return interp.executeBody(n.ElseBody)
	}

	return nil
}

// ---------------------------------------------------------------------------
// executeChoiceBlock
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeChoiceBlock(n *ast.ChoiceBlock) error {
	interp.env.Context.AddToExecutionPath("choice")
	interp.pushBlock("choice")
	defer interp.popBlock()

	if len(n.Options) == 0 {
		return nil
	}

	// Build a prompt asking the provider to choose among options.
	var sb strings.Builder
	sb.WriteString("You are a decision maker. Based on the criteria and context, choose the best option.\n\n")

	if n.Criteria != nil {
		sb.WriteString("## Criteria\n")
		sb.WriteString(n.Criteria.Expression)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Options\n")
	for i, opt := range n.Options {
		label := runtimeValueToString(interp.evaluateExpression(opt.Label))
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, label))
	}

	// Add context variables.
	vars := interp.env.Context.GetAllVariables()
	if len(vars) > 0 {
		sb.WriteString("\n## Context Variables\n")
		for name, value := range vars {
			sb.WriteString(fmt.Sprintf("- %s = %s\n", name, runtimeValueToString(value)))
		}
	}

	sb.WriteString(fmt.Sprintf("\nRespond with ONLY the option number (1-%d). Nothing else.", len(n.Options)))

	prompt := sb.String()

	// Use the condition provider or default.
	providerName := interp.env.Config.DefaultProvider
	if interp.env.Config.ConditionProvider != "" {
		providerName = interp.env.Config.ConditionProvider
	}

	provider, err := interp.getProvider(providerName)
	if err != nil {
		return fmt.Errorf("interpreter: choice provider error: %w", err)
	}

	spec := SessionSpec{
		Agent: &AgentInstance{
			Name:     "choice-evaluator",
			Provider: providerName,
		},
		Prompt: prompt,
	}

	result, err := provider.ExecuteSession(spec, interp.env.Config, false, nil, nil)
	if err != nil {
		return fmt.Errorf("interpreter: choice evaluation failed: %w", err)
	}

	// Parse the chosen option number.
	answer := strings.TrimSpace(result.Output)
	chosenIndex := 0 // default to first option
	for i := len(n.Options); i >= 1; i-- {
		if strings.Contains(answer, fmt.Sprintf("%d", i)) {
			chosenIndex = i - 1
			break
		}
	}

	if chosenIndex >= 0 && chosenIndex < len(n.Options) {
		interp.executionEvents = append(interp.executionEvents, ExecutionEvent{
			Type:        "choice",
			Description: fmt.Sprintf("chose option %d", chosenIndex+1),
			Timestamp:   time.Now().UnixMilli(),
			Result:      answer,
		})
		return interp.executeBody(n.Options[chosenIndex].Body)
	}

	return nil
}

// ---------------------------------------------------------------------------
// executeTryBlock
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeTryBlock(n *ast.TryBlock) error {
	interp.env.Context.AddToExecutionPath("try")

	var tryErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				if cf, ok := r.(ControlFlow); ok {
					switch cf.Type {
					case "throw":
						if cf.Err != nil {
							tryErr = cf.Err
						} else {
							tryErr = fmt.Errorf("%v", cf.Value)
						}
					default:
						// Re-panic for break, continue, return.
						panic(r)
					}
				} else {
					tryErr = fmt.Errorf("%v", r)
				}
			}
		}()

		for _, stmt := range n.TryBody {
			if err := interp.executeStatement(stmt); err != nil {
				tryErr = err
				return
			}
		}
	}()

	// Execute catch block if there was an error.
	if tryErr != nil && len(n.CatchBody) > 0 {
		interp.env.Context.PushScope()
		if n.ErrorVar != nil {
			_ = interp.env.Context.DeclareVariable(n.ErrorVar.Name, tryErr.Error(), false, n.Span)
		}
		for _, stmt := range n.CatchBody {
			if err := interp.executeStatement(stmt); err != nil {
				interp.env.Context.PopScope()
				// Execute finally before returning.
				interp.executeFinallyBody(n.FinallyBody)
				return err
			}
		}
		interp.env.Context.PopScope()
		tryErr = nil // Error handled by catch.
	}

	// Always execute finally block.
	interp.executeFinallyBody(n.FinallyBody)

	return tryErr
}

func (interp *Interpreter) executeFinallyBody(body []ast.Node) {
	for _, stmt := range body {
		_ = interp.executeStatement(stmt)
	}
}

// ---------------------------------------------------------------------------
// executeAskStatement
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeAskStatement(n *ast.AskStatement) error {
	interp.env.Context.AddToExecutionPath("ask")

	prompt := ""
	if n.Prompt != nil {
		prompt = runtimeValueToString(interp.evaluateExpression(n.Prompt))
	}

	// Check pre-filled inputs first (e.g. from desktop app UI).
	if n.Variable != nil {
		if answer, ok := interp.initialInputs[n.Variable.Name]; ok {
			return interp.env.Context.DeclareVariable(n.Variable.Name, answer, false, n.Span)
		}
	}

	// Check if we have a saved user input from a previous run.
	if interp.stateStore != nil && interp.currentRunID > 0 && n.Variable != nil {
		inputs, err := interp.stateStore.GetUserInputs(interp.currentRunID)
		if err == nil {
			if answer, ok := inputs[n.Variable.Name]; ok {
				return interp.env.Context.DeclareVariable(n.Variable.Name, answer, false, n.Span)
			}
		}
	}

	fmt.Print(prompt + " ")
	scanner := bufio.NewScanner(os.Stdin)
	var answer string
	if scanner.Scan() {
		answer = scanner.Text()
	}

	// Save the user input if state store is available.
	if interp.stateStore != nil && interp.currentRunID > 0 && n.Variable != nil {
		_ = interp.stateStore.SaveUserInput(interp.currentRunID, n.Variable.Name, answer)
	}

	if n.Variable != nil {
		return interp.env.Context.DeclareVariable(n.Variable.Name, answer, false, n.Span)
	}
	return nil
}

// ---------------------------------------------------------------------------
// executeRunStatement
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeRunStatement(n *ast.RunStatement) error {
	interp.env.Context.AddToExecutionPath("run")

	filePath := runtimeValueToString(interp.evaluateExpression(n.FilePath))

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("interpreter: run: cannot read file '%s': %w", filePath, err)
	}

	parseResult := parser.Parse(string(data))
	if len(parseResult.Errors) > 0 {
		var msgs []string
		for _, pe := range parseResult.Errors {
			msgs = append(msgs, pe.Message)
		}
		return fmt.Errorf("interpreter: run: parse errors in '%s': %s", filePath, strings.Join(msgs, "; "))
	}

	// Create a sub-interpreter that shares the environment but has its own block map.
	subInterp := NewInterpreter(
		interp.env,
		WithToolRegistry(interp.toolRegistry),
		WithFileName(filePath),
	)
	if interp.stateStore != nil {
		subInterp.stateStore = interp.stateStore
		subInterp.currentRunID = interp.currentRunID
	}
	subInterp.sessionIndex = interp.sessionIndex
	subInterp.providerCache = interp.providerCache
	subInterp.piConfig = interp.piConfig
	subInterp.nativeProviders = interp.nativeProviders

	_, err = subInterp.Execute(parseResult.Program)

	// Propagate session index back.
	interp.sessionIndex = subInterp.sessionIndex
	interp.allSessionOutputs = append(interp.allSessionOutputs, subInterp.allSessionOutputs...)

	return err
}

// ---------------------------------------------------------------------------
// executeSkillInvocation
// ---------------------------------------------------------------------------

func (interp *Interpreter) executeSkillInvocation(n *ast.SkillInvocation) error {
	interp.env.Context.AddToExecutionPath("skill")

	// Evaluate parameters.
	params := make(map[string]any, len(n.Params))
	for _, p := range n.Params {
		params[p.Name.Name] = interp.evaluateExpression(p.Value)
	}

	// Check if the skill is registered as a tool.
	if interp.toolRegistry != nil {
		if _, ok := interp.toolRegistry.Get(n.SkillName.Name); ok {
			result, err := interp.toolRegistry.Execute(n.SkillName.Name, params)
			if err != nil {
				return fmt.Errorf("interpreter: skill '%s' execution error: %w", n.SkillName.Name, err)
			}
			if n.OutputVar != nil {
				if err := interp.env.Context.DeclareVariable(n.OutputVar.Name, result, false, n.Span); err != nil {
					_ = interp.env.Context.SetVariable(n.OutputVar.Name, result)
				}
			}
			return nil
		}
	}

	// TODO: Implement full skill resolution (import-based skills, etc.).
	interp.env.Log("warn", fmt.Sprintf("skill '%s' not found; skipping", n.SkillName.Name))
	return nil
}

// ---------------------------------------------------------------------------
// evaluatePipeExpression
// ---------------------------------------------------------------------------

func (interp *Interpreter) evaluatePipeExpression(n *ast.PipeExpression) RuntimeValue {
	current := interp.evaluateExpression(n.Input)

	for _, op := range n.Operations {
		switch op.Operator {
		case "map":
			current = interp.pipeMap(current, op)
		case "filter":
			current = interp.pipeFilter(current, op)
		case "reduce":
			current = interp.pipeReduce(current, op)
		case "pmap":
			current = interp.pipePmap(current, op)
		default:
			interp.env.Log("warn", fmt.Sprintf("unknown pipe operator: %s", op.Operator))
		}
	}

	return current
}

func (interp *Interpreter) pipeMap(input RuntimeValue, op *ast.PipeOperation) RuntimeValue {
	items := toSlice(input)
	results := make([]any, 0, len(items))

	for _, item := range items {
		interp.env.Context.PushScope()
		if op.ItemVar != nil {
			_ = interp.env.Context.DeclareVariable(op.ItemVar.Name, item, false, op.Span)
		}

		var result RuntimeValue
		for _, stmt := range op.Body {
			result = interp.evaluateExpression(stmt)
		}
		results = append(results, result)

		interp.env.Context.PopScope()
	}

	return results
}

func (interp *Interpreter) pipeFilter(input RuntimeValue, op *ast.PipeOperation) RuntimeValue {
	items := toSlice(input)
	var results []any

	for _, item := range items {
		interp.env.Context.PushScope()
		if op.ItemVar != nil {
			_ = interp.env.Context.DeclareVariable(op.ItemVar.Name, item, false, op.Span)
		}

		var result RuntimeValue
		for _, stmt := range op.Body {
			result = interp.evaluateExpression(stmt)
		}

		if isTruthy(result) {
			results = append(results, item)
		}

		interp.env.Context.PopScope()
	}

	return results
}

func (interp *Interpreter) pipeReduce(input RuntimeValue, op *ast.PipeOperation) RuntimeValue {
	items := toSlice(input)
	if len(items) == 0 {
		return nil
	}

	accumulator := items[0]

	for i := 1; i < len(items); i++ {
		interp.env.Context.PushScope()
		if op.AccVar != nil {
			_ = interp.env.Context.DeclareVariable(op.AccVar.Name, accumulator, false, op.Span)
		}
		if op.ItemVar != nil {
			_ = interp.env.Context.DeclareVariable(op.ItemVar.Name, items[i], false, op.Span)
		}

		var result RuntimeValue
		for _, stmt := range op.Body {
			result = interp.evaluateExpression(stmt)
		}
		accumulator = result

		interp.env.Context.PopScope()
	}

	return accumulator
}

func (interp *Interpreter) pipePmap(input RuntimeValue, op *ast.PipeOperation) RuntimeValue {
	items := toSlice(input)
	results := make([]any, len(items))

	var wg sync.WaitGroup
	for i, item := range items {
		wg.Add(1)
		go func(idx int, val any) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					// Swallow panics in parallel map; store nil.
					results[idx] = nil
				}
			}()

			// NOTE: This shares the interpreter context which is not goroutine-safe.
			// A production implementation would use isolated sub-interpreters.
			interp.env.Context.PushScope()
			if op.ItemVar != nil {
				_ = interp.env.Context.DeclareVariable(op.ItemVar.Name, val, false, op.Span)
			}

			var result RuntimeValue
			for _, stmt := range op.Body {
				result = interp.evaluateExpression(stmt)
			}
			results[idx] = result

			interp.env.Context.PopScope()
		}(i, item)
	}

	wg.Wait()
	return results
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// interpolateString evaluates an InterpolatedString's parts into a single string.
func (interp *Interpreter) interpolateString(parts []ast.Node) string {
	var sb strings.Builder
	for _, part := range parts {
		switch p := part.(type) {
		case *ast.StringLiteral:
			sb.WriteString(p.Value)
		case *ast.Identifier:
			val, err := interp.env.Context.GetVariable(p.Name)
			if err != nil {
				sb.WriteString(p.Name)
			} else {
				sb.WriteString(runtimeValueToString(val))
			}
		default:
			sb.WriteString(runtimeValueToString(interp.evaluateExpression(part)))
		}
	}
	return sb.String()
}

// executeBody executes a slice of statements in a new scope.
func (interp *Interpreter) executeBody(body []ast.Node) error {
	interp.env.Context.PushScope()
	defer interp.env.Context.PopScope()

	for _, stmt := range body {
		if err := interp.executeStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

// executeBodyWithBreakContinue executes body statements, catching break and
// continue control flow. Returns a sentinel error with message "break" for
// break, nil for continue, and other errors as-is.
func (interp *Interpreter) executeBodyWithBreakContinue(body []ast.Node) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			if cf, ok := r.(ControlFlow); ok {
				switch cf.Type {
				case "break":
					retErr = fmt.Errorf("break")
					return
				case "continue":
					retErr = nil
					return
				}
			}
			panic(r) // re-panic for anything else
		}
	}()

	for _, stmt := range body {
		if err := interp.executeStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

// pushBlock pushes a block name onto the block stack for context tracking.
func (interp *Interpreter) pushBlock(name string) {
	interp.currentBlockStack = append(interp.currentBlockStack, name)
}

// popBlock pops the top block name from the block stack.
func (interp *Interpreter) popBlock() {
	if len(interp.currentBlockStack) > 0 {
		interp.currentBlockStack = interp.currentBlockStack[:len(interp.currentBlockStack)-1]
	}
}

// getPropertyValue finds a property by name in a slice.
func getPropertyValue(props []*ast.Property, name string) ast.Node {
	for _, p := range props {
		if p.Name.Name == name {
			return p.Value
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Value conversion helpers
// ---------------------------------------------------------------------------

// runtimeValueToString converts any runtime value to its string representation.
func runtimeValueToString(v RuntimeValue) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == math.Trunc(val) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case *SessionResult:
		return val.Output
	case []any:
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(data)
	case map[string]any:
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// toInt converts a runtime value to an integer.
func toInt(v RuntimeValue) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case string:
		// Try to parse as a number.
		var f float64
		if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
			return int(f)
		}
		return 0
	default:
		return 0
	}
}

// toSlice converts a runtime value to a slice of any.
func toSlice(v RuntimeValue) []any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []any:
		return val
	case string:
		// Split string into characters.
		chars := make([]any, 0, len(val))
		for _, ch := range val {
			chars = append(chars, string(ch))
		}
		return chars
	default:
		// Wrap single values in a slice.
		return []any{v}
	}
}

// isTruthy determines whether a runtime value is considered truthy.
func isTruthy(v RuntimeValue) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		lower := strings.ToLower(strings.TrimSpace(val))
		return lower != "" && lower != "false" && lower != "0" && lower != "nil" && lower != "null"
	case float64:
		return val != 0
	case int:
		return val != 0
	case int64:
		return val != 0
	case []any:
		return len(val) > 0
	case map[string]any:
		return len(val) > 0
	default:
		return true
	}
}

// truncateString truncates a string to maxLen characters, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
