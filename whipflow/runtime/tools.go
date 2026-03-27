// Package runtime implements the WhipFlow runtime, including the tool registry
// and built-in tool definitions.
package runtime

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ToolParameter describes a single parameter accepted by a tool.
type ToolParameter struct {
	Type        string   // JSON schema type (e.g. "string", "number", "boolean")
	Description string   // human-readable description
	Enum        []string // allowed values; nil if unrestricted
}

// ToolParameters describes the full parameter schema for a tool.
type ToolParameters struct {
	Type       string                   // always "object"
	Properties map[string]ToolParameter // parameter definitions keyed by name
	Required   []string                 // names of required parameters
}

// ToolDefinition defines a tool that can be registered and executed.
type ToolDefinition struct {
	Name        string                                       // unique tool name
	Description string                                       // human-readable description
	Parameters  ToolParameters                               // parameter schema
	Handler     func(args map[string]any) (RuntimeValue, error) // execution handler
}

// ToolCallResult captures the result of a single tool call.
type ToolCallResult struct {
	Name      string         // tool that was called
	Arguments map[string]any // arguments passed
	Result    RuntimeValue   // return value
}

// ToolExecutionLog records a tool execution with timing and error information.
type ToolExecutionLog struct {
	Name      string         // tool that was called
	Arguments map[string]any // arguments passed
	Result    RuntimeValue   // return value
	Err       error          // error, if any
	Timestamp int64          // Unix milliseconds when execution started
	Duration  int64          // execution duration in milliseconds
}

// ToolRegistry manages tool registration, lookup, and execution logging.
type ToolRegistry struct {
	tools map[string]ToolDefinition
	log   []ToolExecutionLog
}

// NewToolRegistry creates a new ToolRegistry pre-populated with all built-in tools.
func NewToolRegistry() *ToolRegistry {
	r := &ToolRegistry{
		tools: make(map[string]ToolDefinition),
	}
	for _, t := range builtinTools {
		r.Register(t)
	}
	return r
}

// Register adds a tool to the registry, overwriting any existing tool with the
// same name.
func (r *ToolRegistry) Register(tool ToolDefinition) {
	r.tools[tool.Name] = tool
}

// Get returns the tool definition for the given name. The second return value
// indicates whether the tool was found.
func (r *ToolRegistry) Get(name string) (*ToolDefinition, bool) {
	t, ok := r.tools[name]
	if !ok {
		return nil, false
	}
	return &t, true
}

// GetAll returns a slice of all registered tool definitions.
func (r *ToolRegistry) GetAll() []ToolDefinition {
	all := make([]ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		all = append(all, t)
	}
	return all
}

// Execute runs the named tool with the given arguments. It records the
// execution in the log and returns the result. If the tool is not found, an
// error is returned.
func (r *ToolRegistry) Execute(name string, args map[string]any) (RuntimeValue, error) {
	td, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	start := time.Now()
	result, err := td.Handler(args)
	duration := time.Since(start).Milliseconds()

	r.log = append(r.log, ToolExecutionLog{
		Name:      name,
		Arguments: args,
		Result:    result,
		Err:       err,
		Timestamp: start.UnixMilli(),
		Duration:  duration,
	})

	return result, err
}

// GetExecutionLog returns a copy of the execution log.
func (r *ToolRegistry) GetExecutionLog() []ToolExecutionLog {
	out := make([]ToolExecutionLog, len(r.log))
	copy(out, r.log)
	return out
}

// ClearLog removes all entries from the execution log.
func (r *ToolRegistry) ClearLog() {
	r.log = r.log[:0]
}

// ---------------------------------------------------------------------------
// Built-in tools
// ---------------------------------------------------------------------------

// builtinTools contains all tools that are registered by default.
var builtinTools = []ToolDefinition{
	calculateTool(),
	getCurrentTimeTool(),
	randomNumberTool(),
	stringOperationsTool(),
	readTool(),
	writeTool(),
	bashTool(),
	editTool(),
}

// ---------------------------------------------------------------------------
// calculate
// ---------------------------------------------------------------------------

func calculateTool() ToolDefinition {
	return ToolDefinition{
		Name:        "calculate",
		Description: "Evaluate a simple arithmetic expression (+, -, *, /, %)",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParameter{
				"expression": {Type: "string", Description: "Arithmetic expression to evaluate"},
			},
			Required: []string{"expression"},
		},
		Handler: func(args map[string]any) (RuntimeValue, error) {
			expr, _ := args["expression"].(string)
			if expr == "" {
				return nil, fmt.Errorf("calculate: expression is required")
			}
			result, err := evalArithmetic(expr)
			if err != nil {
				return nil, fmt.Errorf("calculate: %w", err)
			}
			return result, nil
		},
	}
}

// evalArithmetic evaluates a simple arithmetic expression supporting +, -, *,
// /, % and parentheses over floating-point numbers. It uses a recursive-descent
// parser.
func evalArithmetic(expr string) (float64, error) {
	p := &arithParser{input: expr}
	p.skipSpaces()
	result, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	p.skipSpaces()
	if p.pos < len(p.input) {
		return 0, fmt.Errorf("unexpected character at position %d: %c", p.pos, p.input[p.pos])
	}
	return result, nil
}

type arithParser struct {
	input string
	pos   int
}

func (p *arithParser) skipSpaces() {
	for p.pos < len(p.input) && p.input[p.pos] == ' ' {
		p.pos++
	}
}

func (p *arithParser) parseExpr() (float64, error) {
	return p.parseAddSub()
}

func (p *arithParser) parseAddSub() (float64, error) {
	left, err := p.parseMulDivMod()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpaces()
		if p.pos >= len(p.input) {
			break
		}
		op := p.input[p.pos]
		if op != '+' && op != '-' {
			break
		}
		p.pos++
		right, err := p.parseMulDivMod()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

func (p *arithParser) parseMulDivMod() (float64, error) {
	left, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpaces()
		if p.pos >= len(p.input) {
			break
		}
		op := p.input[p.pos]
		if op != '*' && op != '/' && op != '%' {
			break
		}
		p.pos++
		right, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		switch op {
		case '*':
			left *= right
		case '/':
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left /= right
		case '%':
			if right == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			left = math.Mod(left, right)
		}
	}
	return left, nil
}

func (p *arithParser) parseUnary() (float64, error) {
	p.skipSpaces()
	if p.pos < len(p.input) && p.input[p.pos] == '-' {
		p.pos++
		val, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		return -val, nil
	}
	if p.pos < len(p.input) && p.input[p.pos] == '+' {
		p.pos++
		return p.parseUnary()
	}
	return p.parsePrimary()
}

func (p *arithParser) parsePrimary() (float64, error) {
	p.skipSpaces()
	if p.pos >= len(p.input) {
		return 0, fmt.Errorf("unexpected end of expression")
	}

	// Parenthesised sub-expression.
	if p.input[p.pos] == '(' {
		p.pos++ // consume '('
		val, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		p.skipSpaces()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		p.pos++ // consume ')'
		return val, nil
	}

	// Number literal.
	start := p.pos
	if p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9' || p.input[p.pos] == '.') {
		for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9' || p.input[p.pos] == '.') {
			p.pos++
		}
		num, err := strconv.ParseFloat(p.input[start:p.pos], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", p.input[start:p.pos])
		}
		return num, nil
	}

	return 0, fmt.Errorf("unexpected character: %c", p.input[p.pos])
}

// ---------------------------------------------------------------------------
// get_current_time
// ---------------------------------------------------------------------------

func getCurrentTimeTool() ToolDefinition {
	return ToolDefinition{
		Name:        "get_current_time",
		Description: "Returns the current time in the requested format",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParameter{
				"format": {
					Type:        "string",
					Description: "Output format: iso, unix, or readable",
					Enum:        []string{"iso", "unix", "readable"},
				},
			},
			Required: []string{"format"},
		},
		Handler: func(args map[string]any) (RuntimeValue, error) {
			format, _ := args["format"].(string)
			now := time.Now()
			switch format {
			case "iso":
				return now.Format(time.RFC3339), nil
			case "unix":
				return now.Unix(), nil
			case "readable":
				return now.Format("Monday, January 2, 2006 3:04:05 PM MST"), nil
			default:
				return now.Format(time.RFC3339), nil
			}
		},
	}
}

// ---------------------------------------------------------------------------
// random_number
// ---------------------------------------------------------------------------

func randomNumberTool() ToolDefinition {
	return ToolDefinition{
		Name:        "random_number",
		Description: "Generate a random number between min and max",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParameter{
				"min":     {Type: "number", Description: "Minimum value (inclusive)"},
				"max":     {Type: "number", Description: "Maximum value (inclusive)"},
				"integer": {Type: "boolean", Description: "Whether to return an integer"},
			},
			Required: []string{"min", "max"},
		},
		Handler: func(args map[string]any) (RuntimeValue, error) {
			minVal, err := toFloat64(args["min"])
			if err != nil {
				return nil, fmt.Errorf("random_number: invalid min: %w", err)
			}
			maxVal, err := toFloat64(args["max"])
			if err != nil {
				return nil, fmt.Errorf("random_number: invalid max: %w", err)
			}
			if minVal > maxVal {
				return nil, fmt.Errorf("random_number: min (%v) must be <= max (%v)", minVal, maxVal)
			}

			integer, _ := args["integer"].(bool)

			if integer {
				lo := int64(math.Ceil(minVal))
				hi := int64(math.Floor(maxVal))
				if lo > hi {
					return nil, fmt.Errorf("random_number: no integers in range [%v, %v]", minVal, maxVal)
				}
				return lo + rand.Int64N(hi-lo+1), nil
			}
			return minVal + rand.Float64()*(maxVal-minVal), nil
		},
	}
}

// ---------------------------------------------------------------------------
// string_operations
// ---------------------------------------------------------------------------

func stringOperationsTool() ToolDefinition {
	return ToolDefinition{
		Name:        "string_operations",
		Description: "Perform common string operations",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParameter{
				"text": {Type: "string", Description: "Input text"},
				"operation": {
					Type:        "string",
					Description: "Operation to perform",
					Enum:        []string{"uppercase", "lowercase", "reverse", "length", "trim", "capitalize"},
				},
			},
			Required: []string{"text", "operation"},
		},
		Handler: func(args map[string]any) (RuntimeValue, error) {
			text, _ := args["text"].(string)
			operation, _ := args["operation"].(string)
			switch operation {
			case "uppercase":
				return strings.ToUpper(text), nil
			case "lowercase":
				return strings.ToLower(text), nil
			case "reverse":
				runes := []rune(text)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				return string(runes), nil
			case "length":
				return len([]rune(text)), nil
			case "trim":
				return strings.TrimSpace(text), nil
			case "capitalize":
				if text == "" {
					return "", nil
				}
				runes := []rune(text)
				runes[0] = unicode.ToUpper(runes[0])
				return string(runes), nil
			default:
				return nil, fmt.Errorf("string_operations: unknown operation: %s", operation)
			}
		},
	}
}

// ---------------------------------------------------------------------------
// read
// ---------------------------------------------------------------------------

func readTool() ToolDefinition {
	return ToolDefinition{
		Name:        "read",
		Description: "Read the contents of a file",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParameter{
				"path": {Type: "string", Description: "Path to the file to read"},
			},
			Required: []string{"path"},
		},
		Handler: func(args map[string]any) (RuntimeValue, error) {
			path, _ := args["path"].(string)
			if path == "" {
				return nil, fmt.Errorf("read: path is required")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read: %w", err)
			}
			return string(data), nil
		},
	}
}

// ---------------------------------------------------------------------------
// write
// ---------------------------------------------------------------------------

func writeTool() ToolDefinition {
	return ToolDefinition{
		Name:        "write",
		Description: "Write content to a file",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParameter{
				"path":    {Type: "string", Description: "Path to the file to write"},
				"content": {Type: "string", Description: "Content to write"},
				"append":  {Type: "boolean", Description: "Append to file instead of overwriting"},
			},
			Required: []string{"path", "content"},
		},
		Handler: func(args map[string]any) (RuntimeValue, error) {
			path, _ := args["path"].(string)
			if path == "" {
				return nil, fmt.Errorf("write: path is required")
			}
			content, _ := args["content"].(string)
			appendMode, _ := args["append"].(bool)

			if appendMode {
				f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return nil, fmt.Errorf("write: %w", err)
				}
				defer f.Close()
				if _, err := f.WriteString(content); err != nil {
					return nil, fmt.Errorf("write: %w", err)
				}
			} else {
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					return nil, fmt.Errorf("write: %w", err)
				}
			}
			return map[string]any{
				"success": true,
				"path":    path,
				"bytes":   len(content),
			}, nil
		},
	}
}

// ---------------------------------------------------------------------------
// bash
// ---------------------------------------------------------------------------

func bashTool() ToolDefinition {
	return ToolDefinition{
		Name:        "bash",
		Description: "Execute a shell command",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParameter{
				"command": {Type: "string", Description: "Shell command to execute"},
				"cwd":     {Type: "string", Description: "Working directory for the command"},
				"timeout": {Type: "number", Description: "Timeout in milliseconds"},
			},
			Required: []string{"command"},
		},
		Handler: func(args map[string]any) (RuntimeValue, error) {
			command, _ := args["command"].(string)
			if command == "" {
				return nil, fmt.Errorf("bash: command is required")
			}

			timeoutMs := 30000.0 // default 30 seconds
			if t, err := toFloat64(args["timeout"]); err == nil && t > 0 {
				timeoutMs = t
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
			defer cancel()

			cmd := exec.CommandContext(ctx, "sh", "-c", command)

			if cwd, ok := args["cwd"].(string); ok && cwd != "" {
				cmd.Dir = cwd
			}

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					return nil, fmt.Errorf("bash: %w", err)
				}
			}

			return map[string]any{
				"stdout":   stdout.String(),
				"stderr":   stderr.String(),
				"exitCode": exitCode,
			}, nil
		},
	}
}

// ---------------------------------------------------------------------------
// edit
// ---------------------------------------------------------------------------

func editTool() ToolDefinition {
	return ToolDefinition{
		Name:        "edit",
		Description: "Edit file content with replace, insert, append, or prepend operations",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParameter{
				"path":      {Type: "string", Description: "Path to the file to edit"},
				"operation": {Type: "string", Description: "Edit operation", Enum: []string{"replace", "insert", "append", "prepend"}},
				"search":    {Type: "string", Description: "Text to search for (used by replace and insert)"},
				"content":   {Type: "string", Description: "New content to write"},
			},
			Required: []string{"path", "operation", "content"},
		},
		Handler: func(args map[string]any) (RuntimeValue, error) {
			path, _ := args["path"].(string)
			if path == "" {
				return nil, fmt.Errorf("edit: path is required")
			}
			operation, _ := args["operation"].(string)
			content, _ := args["content"].(string)
			search, _ := args["search"].(string)

			data, err := os.ReadFile(path)
			if err != nil && operation != "append" && operation != "prepend" {
				return nil, fmt.Errorf("edit: %w", err)
			}
			original := string(data)

			var result string
			switch operation {
			case "replace":
				if search == "" {
					return nil, fmt.Errorf("edit: search is required for replace operation")
				}
				if !strings.Contains(original, search) {
					return nil, fmt.Errorf("edit: search text not found in file")
				}
				result = strings.Replace(original, search, content, 1)

			case "insert":
				if search == "" {
					return nil, fmt.Errorf("edit: search is required for insert operation")
				}
				idx := strings.Index(original, search)
				if idx == -1 {
					return nil, fmt.Errorf("edit: search text not found in file")
				}
				insertPos := idx + len(search)
				result = original[:insertPos] + content + original[insertPos:]

			case "append":
				result = original + content

			case "prepend":
				result = content + original

			default:
				return nil, fmt.Errorf("edit: unknown operation: %s", operation)
			}

			if err := os.WriteFile(path, []byte(result), 0644); err != nil {
				return nil, fmt.Errorf("edit: %w", err)
			}

			return map[string]any{
				"success":   true,
				"path":      path,
				"operation": operation,
			}, nil
		},
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// toFloat64 converts a value that may be a JSON number (float64), int, int64,
// or numeric string to float64.
func toFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case string:
		return strconv.ParseFloat(n, 64)
	case nil:
		return 0, fmt.Errorf("value is nil")
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}
