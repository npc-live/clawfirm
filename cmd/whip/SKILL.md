---
name: whipflow
version: 0.0.1
description: |
  Whipflow is an AI workflow automation tool for harness.farm.
  Write .whip files in flows/ to define multi-step AI workflows.
  The runtime handles execution, context passing, and tool permissions.

  Activate when: user wants to create or run a workflow, mentions whipflow,
  or needs to automate a multi-step AI task for harness.farm.

  If whipflow is not installed, offer to install it:
  npm i -g @harness.farm/whipflow
---

# Whipflow

Whipflow orchestrates AI workflows using the OpenProse DSL. Flows live in `flows/`
with the `.whip` extension and are run with `whipflow run flows/my-flow.whip`.

## CLI Commands

```bash
whipflow run <file.whip>        # execute a workflow
whipflow validate <file.whip>   # check syntax without running
whipflow compile <file.whip>    # show compiled form
whipflow install-skills         # install this skill into Claude Code
whipflow install-skills --force # overwrite existing
whipflow help
```

## Quick Start

```whip
agent worker:
  provider: "claude-code"
  model: sonnet
  tools: ["bash"]
  prompt: "You are a helpful assistant."

let result = session: worker
  prompt: "Echo hello world using bash"
```

### Ask user for input

```whip
ask project_name: "What is the project name?"
ask description:  "One sentence description?"

let result = session: worker
  prompt: "Create a README for {project_name}: {description}"
```

### Multi-step with context passing

```whip
let step1 = session: worker
  prompt: "Analyze the codebase structure"

let step2 = session: worker
  prompt: """
  Based on this analysis:
  {step1}

  Write a summary report.
  """
```

---

# OpenProse Language Reference

OpenProse is a DSL for orchestrating multi-agent workflows. Programs consist of
statements executed sequentially, with each `session` spawning a subagent.

## File Format

| Property | Value |
|----------|-------|
| Extension | `.whip` |
| Encoding | UTF-8 |
| Indentation | Spaces (Python-like) |
| Case sensitivity | Case-sensitive |

---

## Comments

```whip
# This is a standalone comment

session "Hello"  # Inline comment
```

- Comments begin with `#` and extend to end of line
- `#` inside string literals is NOT a comment
- Comments are stripped during compilation — agents never see them

---

## String Literals

### Single-line

```whip
session "Hello world"
session "Line one\nLine two"
session "She said \"hello\""
```

Escape sequences: `\\` `\"` `\n` `\t`

### Multi-line

```whip
session """
This is a multi-line prompt.
It preserves:
  - Indentation
  - Line breaks
  - All internal whitespace
"""
```

Opening `"""` must be followed by a newline. Content continues until closing `"""`.

### String Interpolation

```whip
let name = session "Get the user's name"
session "Hello {name}, welcome!"
```

- Use `{varname}` to embed variables
- Works in both single-line and multi-line strings
- Escape with `\{` for a literal `{`

---

## Import Statements

```whip
import "web-search" from "github:anthropic/skills"
import "code-analyzer" from "npm:@company/analyzer"
import "custom-tool" from "./skills/custom-tool"
```

Source types: `github:user/repo`, `npm:package`, `./local/path`

---

## Agent Definitions

```whip
agent name:
  provider: "claude-code"   # claude-code | claude | opencode | aider | custom:bin
  model: sonnet             # opus | sonnet | haiku
  tools: ["bash", "read", "write", "edit"]
  skills: ["skill-name"]
  prompt: "System prompt for this agent."
  permissions:
    read: ["*.md"]
    bash: deny
```

### Properties

| Property | Description |
|----------|-------------|
| `provider` | Which AI runtime to use |
| `model` | `sonnet`, `opus`, or `haiku` |
| `tools` | Tools the agent can use |
| `skills` | Imported skills assigned to this agent |
| `prompt` | System prompt / persona |
| `permissions` | Access control rules |

### Model Selection

| Model | Use Case |
|-------|----------|
| `haiku` | Fast, simple tasks |
| `sonnet` | Balanced; general purpose |
| `opus` | Complex reasoning; detailed analysis |

### Permissions

```whip
agent secure-agent:
  permissions:
    read: ["*.md", "*.txt"]
    write: ["output/"]
    bash: deny
    network: allow
```

Permission values: `allow`, `deny`, `prompt`, or array of glob patterns.

---

## Session Statement

The primary executable construct. Spawns a subagent to complete a task.

### Variants

```whip
# Simple inline prompt
session "Prompt text"

# With agent reference
session: agentName

# Named session
session analysis: agentName

# With property overrides
session: agentName
  prompt: "Override the default prompt"
  model: opus
```

### Properties

| Property | Description |
|----------|-------------|
| `prompt` | Task instructions |
| `model` | Override agent model |
| `context` | Pass prior session outputs |
| `retry` | Auto-retry count on failure |
| `backoff` | `"none"` \| `"linear"` \| `"exponential"` |

---

## Variables & Context

### Let (mutable) / Const (immutable)

```whip
let draft = session "Write initial draft"
const config = session "Get configuration"

# Reassign let only
draft = session "Improve the draft"
  context: draft
```

### Context Property

```whip
let research = session "Research quantum computing"

# Single context
session "Write summary"
  context: research

# Multiple contexts
let analysis = session "Analyze findings"
session "Write final report"
  context: [research, analysis]

# Object shorthand (for parallel results)
parallel:
  a = session "Task A"
  b = session "Task B"

session "Combine results"
  context: { a, b }
```

### Ask (user input at runtime)

```whip
ask name: "What is your name?"
ask count: "How many items?"
```

---

## Composition Blocks

### do: (anonymous sequential block)

```whip
do:
  session "Step 1"
  session "Step 2"

let result = do:
  session "Gather data"
  session "Process data"
```

### Named Blocks (reusable)

```whip
block review-pipeline:
  session "Security review"
  session "Performance review"
  session "Synthesize reviews"

do review-pipeline
```

### Block Parameters

```whip
block review(topic):
  session "Research {topic} thoroughly"
  session "Summarize {topic} analysis"

do review("quantum computing")
do review("machine learning")
```

### Inline Sequence (Arrow)

```whip
session "Plan" -> session "Execute" -> session "Review"

let workflow = session "Draft" -> session "Edit" -> session "Finalize"
```

---

## Parallel Blocks

```whip
parallel:
  session "Security review"
  session "Performance review"
  session "Style review"
```

All branches start simultaneously. Program waits for all to complete.

### Named Parallel Results

```whip
parallel:
  security = session "Security review"
  perf     = session "Performance review"
  style    = session "Style review"

session "Unified review report"
  context: { security, perf, style }
```

### Join Strategies

```whip
# Race — return on first completion
parallel ("first"):
  session "Try approach A"
  session "Try approach B"

# Any N — wait for N successes
parallel ("any", count: 2):
  session "Attempt 1"
  session "Attempt 2"
  session "Attempt 3"
```

### Failure Policies

```whip
# Continue even if some branches fail
parallel (on-fail: "continue"):
  session "Task 1"
  session "Task 2"

# Ignore all failures
parallel (on-fail: "ignore"):
  session "Optional enrichment"
```

### Combining Modifiers

```whip
parallel ("first", on-fail: "continue"):
  session "Fast but unreliable"
  session "Slow but reliable"
```

---

## Fixed Loops

### Repeat

```whip
repeat 3:
  session "Generate a creative idea"

repeat 5 as i:
  session "Iteration {i} of 5"
```

### For-Each

```whip
let items = ["a", "b", "c"]
for item in items:
  session "Process: {item}"

for item, i in items:
  session "Item {i}: {item}"

# Inline array
for topic in ["AI", "climate", "space"]:
  session "Research this topic"
    context: topic
```

### Parallel Fan-Out

```whip
parallel for item in items:
  session "Process {item} concurrently"
```

---

## Unbounded Loops

Uses **discretion markers** (`**...**`) for AI-evaluated conditions.

```whip
# Loop until condition
loop until **the task is complete**:
  session "Continue working"

# Loop while condition
loop while **there are items to process**:
  session "Process next item"

# With safety limit
loop until **all bugs fixed** (max: 10):
  session "Find and fix a bug"

# With index variable
loop until **done** (max: 5) as attempt:
  session "Attempt {attempt}"

# Basic (always needs max to avoid infinite loop)
loop (max: 50):
  session "Process next item"
```

### Multi-line Conditions

```whip
loop until ***
  the document is complete
  all sections reviewed
  and formatting is consistent
***:
  session "Continue working"
```

---

## Pipeline Operations

```whip
let items = ["a", "b", "c"]

# Map — transform each element
let results = items | map:
  session "Process this item"
    context: item

# Filter — keep matching elements
let good = items | filter:
  session "Is this valid? Answer yes or no."
    context: item

# Reduce — accumulate into single result
let combined = items | reduce(summary, item):
  session "Add '{item}' to summary: {summary}"

# Parallel map
let results = tasks | pmap:
  session "Process concurrently"
    context: item

# Chaining
let result = topics
  | filter:
      session "Is this trending? Answer yes or no."
        context: item
  | map:
      session "Write a one-line pitch"
        context: item
```

Inside `map`, `filter`, `pmap` bodies, `item` is the current element.
Inside `reduce`, variables are named explicitly: `reduce(accVar, itemVar)`.

---

## Error Handling

### Try/Catch/Finally

```whip
try:
  session "Attempt risky operation"
catch:
  session "Handle the error"
finally:
  session "Always clean up"

# Capture error context
try:
  session "Call external API"
catch as err:
  session "Log and handle"
    context: err
```

### Throw

```whip
throw "Precondition not met"

# Re-raise in catch block
try:
  try:
    session "Inner operation"
  catch:
    session "Partial handling"
    throw  # re-raises to outer catch
catch:
  session "Handle re-raised error"
```

### Retry

```whip
session "Call flaky API"
  retry: 3

session "Rate-limited API"
  retry: 5
  backoff: "exponential"
```

Backoff strategies: `"none"` (default), `"linear"`, `"exponential"`

---

## Choice Blocks

AI picks the best option based on criteria.

```whip
choice **the severity of issues found**:
  option "Critical":
    session "Stop deployment, fix critical issues"
  option "Minor":
    session "Log issues for later and proceed"
  option "None":
    session "Deploy immediately"
```

### Multi-line Criteria

```whip
choice ***
  which strategy is most appropriate
  given the current project constraints
***:
  option "MVP approach":
    session "Build minimum viable product"
  option "Full feature set":
    session "Build complete feature set"
```

---

## Conditional Statements

All conditions use discretion markers (`**...**`) for AI evaluation.

```whip
if **the code has security vulnerabilities**:
  session "Fix security issues"
elif **the code has performance issues**:
  session "Optimize performance"
else:
  session "Proceed with normal review"
```

### Multi-line Conditions

```whip
if ***
  the test suite passes
  and code coverage is above 80%
  and there are no linting errors
***:
  session "Deploy to production"
else:
  session "Fix issues before deploying"
```

---

## Complete Example

```whip
# Research + write + review workflow

agent researcher:
  provider: "claude-code"
  model: sonnet
  tools: ["bash", "read"]
  prompt: "You are a thorough research assistant."

agent writer:
  provider: "claude-code"
  model: opus
  tools: ["write", "edit"]
  prompt: "You write clear, concise documentation."

ask topic: "What topic should I research?"

# Parallel research from multiple angles
parallel:
  market  = session: researcher  prompt: "Research market landscape for {topic}"
  tech    = session: researcher  prompt: "Research technical aspects of {topic}"
  trends  = session: researcher  prompt: "Research current trends in {topic}"

# Write a comprehensive report
let report = session: writer
  prompt: """
  Write a comprehensive report on {topic}.

  Use these research inputs:
  - Market: {market}
  - Technical: {tech}
  - Trends: {trends}
  """

# Iterative improvement loop
loop until **the report is polished and publication-ready** (max: 3):
  report = session: writer
    prompt: "Review and improve this report:\n{report}"

# Final quality check
if **the report meets publication standards**:
  session: writer  prompt: "Format the final report for publication"
else:
  session: writer  prompt: "Flag issues and create an improvement plan"
```

---

## Skill Invocation

Invoke a Claude Code skill (`/<name>`) as a session step. Parameters are passed
as additional context after the skill name.

### Syntax

```whip
# Basic invocation
skill <name>

# With named parameters
skill <name> param1=<expr> param2=<expr>

# Capture output
skill <name> param=<expr> -> varname

# As an expression (in let/const)
let result = skill <name> param=<expr>
```

### Examples

```whip
# Run the /commit skill
skill commit

# Run /review-pr with a PR number
skill review-pr args="123"

# Run /simplify and capture output
let review = skill simplify

# Use a variable as parameter
let diff = session: researcher
  prompt: "Show me the git diff"

skill review-pr args=diff -> feedback

session: writer
  prompt: "Address this feedback:\n{feedback}"
```

### How It Works

- `skill foo` expands to a session that sends `/foo` to a `claude-code` agent
- Parameters are appended as `key:\nvalue` blocks after the skill name
- Output can be captured with `-> varname` or `let result = skill ...`
- Skills run with the default model and no extra tools/permissions

---

## Provider Config (.open-prose.json)

Place in the project root to configure providers:

```json
{
  "providers": {
    "claude-code": {
      "bin": "claude",
      "timeout": 300000
    },
    "my-tool": {
      "bin": "/usr/local/bin/my-tool",
      "args": ["--prompt"],
      "promptMode": "arg"
    }
  }
}
```

---

## Tips

- **One goal per session** — focused prompts produce better results
- **Use `ask`** for values that change per run
- **Name outputs with `let`** so you can reference them later
- **`loop until`** is ideal for iterative refinement tasks
- **`parallel`** for independent tasks that don't need each other's output
- **`try/catch`** around network calls or operations that might fail
- **`choice`** when the AI should decide which path to take
