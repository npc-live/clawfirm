package runner

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v3"
)

// LoadScenario reads and parses a scenario YAML file.
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sc Scenario
	if err := yaml.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if sc.Name == "" {
		sc.Name = path
	}
	return &sc, nil
}

// Run executes a scenario and returns the result.
func Run(sc *Scenario, verbose bool) *Result {
	r := &Result{Scenario: sc.Name}
	start := time.Now()

	if err := run(sc, r, verbose); err != nil {
		r.Error = err.Error()
		r.Passed = false
	} else {
		r.Passed = allAssertionsPassed(r.Assertions)
	}
	r.DurationMs = time.Since(start).Milliseconds()
	return r
}

type conn struct {
	ws         *websocket.Conn
	mu         sync.Mutex
	eventsCh   chan EventRecord
	start      time.Time
	lastSendMs int64 // ms since scenario start of the last send step
}

func run(sc *Scenario, r *Result, verbose bool) error {
	dialer := websocket.Dialer{
		Proxy:            func(*http.Request) (*neturl.URL, error) { return nil, nil },
		HandshakeTimeout: 10 * time.Second,
	}

	var c *conn
	autoConnect := func(override *ConnectStep) error {
		if c != nil {
			return nil
		}
		server := sc.Server
		agent := sc.Agent
		sessionID := sc.SessionID
		if override != nil {
			if override.Server != "" {
				server = override.Server
			}
			if override.Agent != "" {
				agent = override.Agent
			}
			if override.SessionID != "" {
				sessionID = override.SessionID
			}
		}
		if server == "" {
			server = "ws://localhost:9988"
		}
		if sessionID == "" {
			sessionID = fmt.Sprintf("s-test-%d", time.Now().UnixMilli())
		}

		var wsURL string
		if agent != "" {
			wsURL = fmt.Sprintf("%s/ws/%s/%s", strings.TrimRight(server, "/"), agent, sessionID)
		} else {
			wsURL = fmt.Sprintf("%s/ws/%s", strings.TrimRight(server, "/"), sessionID)
		}
		if verbose {
			log.Printf("[scenario] connecting to %s", wsURL)
		}
		ws, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			return fmt.Errorf("connect %s: %w", wsURL, err)
		}
		c = &conn{
			ws:       ws,
			eventsCh: make(chan EventRecord, 256),
			start:    time.Now(),
		}
		go readLoop(c, verbose)
		return nil
	}

	for i, step := range sc.Steps {
		switch {
		case step.Connect != nil:
			if c != nil {
				c.ws.Close()
				c = nil
			}
			if err := autoConnect(step.Connect); err != nil {
				return fmt.Errorf("step %d connect: %w", i, err)
			}

		case step.Send != nil:
			if err := autoConnect(nil); err != nil {
				return err
			}
			msgType := step.Send.Type
			if msgType == "" {
				msgType = "message"
			}
			msg := map[string]string{"type": msgType}
			if step.Send.Content != "" {
				msg["content"] = step.Send.Content
			}
			if verbose {
				log.Printf("[scenario] send %s: %q", msgType, step.Send.Content)
			}
			c.mu.Lock()
			err := c.ws.WriteJSON(msg)
			c.lastSendMs = time.Since(c.start).Milliseconds()
			c.mu.Unlock()
			if err != nil {
				return fmt.Errorf("step %d send: %w", i, err)
			}

		case step.Wait != nil:
			time.Sleep(time.Duration(*step.Wait) * time.Millisecond)

		case step.Expect != nil:
			if c == nil {
				return fmt.Errorf("step %d expect: not connected", i)
			}
			timeout := time.Duration(step.Expect.Timeout) * time.Millisecond
			if timeout == 0 {
				timeout = 30 * time.Second
			}
			if verbose {
				log.Printf("[scenario] waiting for event %q (timeout %v)", step.Expect.Event, timeout)
			}
			found := false
			deadline := time.Now().Add(timeout)
			for !found && time.Now().Before(deadline) {
				remaining := time.Until(deadline)
				select {
				case ev, ok := <-c.eventsCh:
					if !ok {
						return fmt.Errorf("step %d expect: connection closed before %q", i, step.Expect.Event)
					}
					r.Events = append(r.Events, ev)
					if verbose {
						log.Printf("[scenario] <- %s %s", ev.Type, ev.Content)
					}
					if ev.Type == step.Expect.Event {
						found = true
					}
				case <-time.After(remaining):
					return fmt.Errorf("step %d expect: timeout waiting for %q", i, step.Expect.Event)
				}
			}

		case step.Disconnect:
			if c != nil {
				c.ws.Close()
				c = nil
				if verbose {
					log.Printf("[scenario] disconnected")
				}
			}

		case step.Assert != nil:
			var lastSendMs int64
			if c != nil {
				drainPending(c, r)
				c.mu.Lock()
				lastSendMs = c.lastSendMs
				c.mu.Unlock()
			}
			results := evaluate(step.Assert, r.Events, lastSendMs)
			r.Assertions = append(r.Assertions, results...)
		}
	}

	// Drain any remaining events.
	if c != nil {
		drainPending(c, r)
		c.ws.Close()
	}

	return nil
}

// readLoop reads server events and pushes them to eventsCh.
func readLoop(c *conn, verbose bool) {
	defer close(c.eventsCh)
	c.ws.SetReadDeadline(time.Now().Add(120 * time.Second))
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		c.ws.SetReadDeadline(time.Now().Add(120 * time.Second))

		var m map[string]any
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		ev := EventRecord{
			TimeMs:     time.Since(c.start).Milliseconds(),
			Type:       str(m["type"]),
			Content:    str(m["content"]),
			StopReason: str(m["stop_reason"]),
		}
		select {
		case c.eventsCh <- ev:
		default:
		}
	}
}

// drainPending moves all buffered events to the result without blocking.
func drainPending(c *conn, r *Result) {
	for {
		select {
		case ev, ok := <-c.eventsCh:
			if !ok {
				return
			}
			r.Events = append(r.Events, ev)
		default:
			return
		}
	}
}

func str(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func allAssertionsPassed(aa []AssertResult) bool {
	for _, a := range aa {
		if !a.Passed {
			return false
		}
	}
	return true
}
