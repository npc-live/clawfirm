package types

// ThinkingLevel controls how much chain-of-thought reasoning the model performs.
type ThinkingLevel string

const (
	ThinkingLevelOff     ThinkingLevel = "off"
	ThinkingLevelMinimal ThinkingLevel = "minimal"
	ThinkingLevelLow     ThinkingLevel = "low"
	ThinkingLevelMedium  ThinkingLevel = "medium"
	ThinkingLevelHigh    ThinkingLevel = "high"
	ThinkingLevelXHigh   ThinkingLevel = "xhigh"
)

// ToolExecutionMode controls whether tools are executed sequentially or in parallel.
type ToolExecutionMode string

const (
	ToolExecutionSequential ToolExecutionMode = "sequential"
	ToolExecutionParallel   ToolExecutionMode = "parallel"
)

// SteeringMode controls how many queued steering messages are consumed per turn.
type SteeringMode string

const (
	// SteeringModeAll delivers all queued steering messages at once after the current turn.
	SteeringModeAll SteeringMode = "all"
	// SteeringModeOneAtATime delivers one steering message per turn, keeping the rest queued.
	SteeringModeOneAtATime SteeringMode = "one-at-a-time"
)

// FollowUpMode controls how many queued follow-up messages are consumed per agent stop.
type FollowUpMode string

const (
	// FollowUpModeAll triggers a new agent run with all queued follow-up messages at once.
	FollowUpModeAll FollowUpMode = "all"
	// FollowUpModeOneAtATime triggers one follow-up message per agent run, keeping the rest queued.
	FollowUpModeOneAtATime FollowUpMode = "one-at-a-time"
)

// ModelCost holds per-token pricing for a model (in USD per million tokens).
type ModelCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

// Model describes an LLM and its capabilities.
type Model struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Provider      string    `json:"provider"`
	BaseURL       string    `json:"baseUrl"`
	Reasoning     bool      `json:"reasoning"`
	InputTypes    []string  `json:"input"` // "text", "image", "audio"
	Cost          ModelCost `json:"cost"`
	ContextWindow int       `json:"contextWindow"`
	MaxTokens     int       `json:"maxTokens"`
}
