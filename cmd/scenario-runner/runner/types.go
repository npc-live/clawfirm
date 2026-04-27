package runner

// Scenario is the parsed YAML scenario file.
type Scenario struct {
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	Server      string  `yaml:"server"`      // ws://host:port (optional, overridden by -server flag)
	Agent       string  `yaml:"agent"`       // agent name (optional, uses default route)
	SessionID   string  `yaml:"session_id"`  // fixed session ID (optional, auto-generated if empty)
	Steps       []Step  `yaml:"steps"`
}

// Step is one action in the scenario.
type Step struct {
	// connect: establish WebSocket connection
	Connect *ConnectStep `yaml:"connect,omitempty"`

	// send: send a message to the server
	Send *SendStep `yaml:"send,omitempty"`

	// wait: sleep N milliseconds
	Wait *int `yaml:"wait,omitempty"`

	// expect: wait for a specific event type (blocks until received or timeout)
	Expect *ExpectStep `yaml:"expect,omitempty"`

	// disconnect: close the WebSocket connection
	Disconnect bool `yaml:"disconnect,omitempty"`

	// assert: verify a condition on collected events
	Assert *AssertStep `yaml:"assert,omitempty"`
}

// ConnectStep connects to the gateway.
// If omitted the runner auto-connects before the first send step.
type ConnectStep struct {
	// Override server/agent/session for this connection only.
	Server    string `yaml:"server,omitempty"`
	Agent     string `yaml:"agent,omitempty"`
	SessionID string `yaml:"session_id,omitempty"`
}

// SendStep sends a client message to the server.
type SendStep struct {
	// Type: "message" (default) | "abort"
	Type    string `yaml:"type,omitempty"`
	Content string `yaml:"content,omitempty"`
}

// ExpectStep waits for a specific server event.
type ExpectStep struct {
	// Event type to wait for: "done" | "error" | "delta" | "thinking"
	Event   string `yaml:"event"`
	Timeout int    `yaml:"timeout,omitempty"` // ms, default 30000
}

// AssertStep checks collected events for a condition.
type AssertStep struct {
	// GotDone: received at least one "done" event
	GotDone *bool `yaml:"got_done,omitempty"`
	// GotResponse: received at least one non-empty "delta" event
	GotResponse *bool `yaml:"got_response,omitempty"`
	// NoError: no "error" events received
	NoError *bool `yaml:"no_error,omitempty"`
	// DoneWithin: "done" event received within N milliseconds of the last "send"
	DoneWithin *int `yaml:"done_within,omitempty"`
	// MaxGap: no gap > N ms between consecutive server events
	MaxGap *int `yaml:"max_gap,omitempty"`
}

// Result is the output from running a scenario.
type Result struct {
	Scenario   string        `json:"scenario"`
	Passed     bool          `json:"passed"`
	DurationMs int64         `json:"duration_ms"`
	Error      string        `json:"error,omitempty"`
	Assertions []AssertResult `json:"assertions,omitempty"`
	Events     []EventRecord  `json:"events,omitempty"`
}

// AssertResult is the outcome of one assertion.
type AssertResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
}

// EventRecord captures one server event with timing.
type EventRecord struct {
	TimeMs int64  `json:"time_ms"` // ms since scenario start
	Type   string `json:"type"`
	Content string `json:"content,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}
