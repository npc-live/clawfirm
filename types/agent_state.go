package types

// AgentState is a read-only snapshot of the agent's current state.
type AgentState struct {
	Model     Model
	IsRunning bool
	Messages  []Message
	Error     string
}
