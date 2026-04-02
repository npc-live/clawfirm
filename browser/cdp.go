// Package browser provides a CDP (Chrome DevTools Protocol) client, session
// management, and a YAML-driven step executor for browser automation.
package browser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Cookie represents a browser cookie from the CDP Network domain.
type Cookie struct {
	Name       string  `json:"name"`
	Value      string  `json:"value"`
	Domain     string  `json:"domain"`
	Path       string  `json:"path"`
	Secure     bool    `json:"secure"`
	HTTPOnly   bool    `json:"httpOnly"`
	Expiration float64 `json:"expirationDate,omitempty"`
}

// CDPClient wraps a single Chrome tab via WebSocket.
type CDPClient struct {
	conn    *websocket.Conn
	mu      sync.Mutex
	pending sync.Map // id → chan cdpResponse
	idSeq   atomic.Int64
	done    chan struct{}
}

type cdpResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *cdpError       `json:"error"`
}

type cdpError struct {
	Message string `json:"message"`
}

// NewCDPClient connects to a Chrome tab via WebSocket.
func NewCDPClient(wsURL string) (*CDPClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cdp: dial %s: %w", wsURL, err)
	}
	c := &CDPClient{conn: conn, done: make(chan struct{})}
	go c.readLoop()
	return c, nil
}

func (c *CDPClient) readLoop() {
	defer close(c.done)
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			ID     int64           `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *cdpError       `json:"error"`
		}
		if json.Unmarshal(raw, &msg) != nil || msg.ID == 0 {
			continue // event, not response
		}
		if ch, ok := c.pending.LoadAndDelete(msg.ID); ok {
			ch.(chan cdpResponse) <- cdpResponse{Result: msg.Result, Error: msg.Error}
		}
	}
}

// Send sends a CDP command and waits for the response.
func (c *CDPClient) Send(method string, params any) (json.RawMessage, error) {
	id := c.idSeq.Add(1)
	ch := make(chan cdpResponse, 1)
	c.pending.Store(id, ch)

	payload := struct {
		ID     int64  `json:"id"`
		Method string `json:"method"`
		Params any    `json:"params,omitempty"`
	}{ID: id, Method: method, Params: params}

	data, _ := json.Marshal(payload)
	c.mu.Lock()
	err := c.conn.WriteMessage(websocket.TextMessage, data)
	c.mu.Unlock()
	if err != nil {
		c.pending.Delete(id)
		return nil, fmt.Errorf("cdp: write: %w", err)
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("cdp: %s", resp.Error.Message)
		}
		return resp.Result, nil
	case <-c.done:
		return nil, fmt.Errorf("cdp: connection closed")
	}
}

// Navigate navigates the tab to a URL and waits waitMs milliseconds.
func (c *CDPClient) Navigate(url string, waitMs int) error {
	_, err := c.Send("Page.navigate", map[string]any{"url": url})
	if err != nil {
		return err
	}
	if waitMs > 0 {
		time.Sleep(time.Duration(waitMs) * time.Millisecond)
	}
	return nil
}

// Eval evaluates a JavaScript expression and returns the result.
func (c *CDPClient) Eval(expression string) (any, error) {
	raw, err := c.Send("Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
	})
	if err != nil {
		return nil, err
	}
	var res struct {
		Result struct {
			Value any    `json:"value"`
			Type  string `json:"type"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}
	return res.Result.Value, nil
}

// GetAllCookies returns all browser cookies.
func (c *CDPClient) GetAllCookies() ([]Cookie, error) {
	raw, err := c.Send("Network.getAllCookies", nil)
	if err != nil {
		return nil, err
	}
	var res struct {
		Cookies []Cookie `json:"cookies"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}
	return res.Cookies, nil
}

// SetCookies sets cookies in the browser.
func (c *CDPClient) SetCookies(cookies []Cookie) error {
	for _, ck := range cookies {
		_, err := c.Send("Network.setCookie", map[string]any{
			"name":     ck.Name,
			"value":    ck.Value,
			"domain":   ck.Domain,
			"path":     ck.Path,
			"secure":   ck.Secure,
			"httpOnly": ck.HTTPOnly,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// DispatchKeyEvent sends a key event via the Input domain.
func (c *CDPClient) DispatchKeyEvent(eventType, key string, code string, keyCode int, modifiers int) error {
	_, err := c.Send("Input.dispatchKeyEvent", map[string]any{
		"type":                  eventType,
		"key":                   key,
		"code":                  code,
		"windowsVirtualKeyCode": keyCode,
		"modifiers":             modifiers,
	})
	return err
}

// InsertText inserts text via the Input domain (works inside shadow DOM).
func (c *CDPClient) InsertText(text string) error {
	_, err := c.Send("Input.insertText", map[string]any{"text": text})
	return err
}

// Close closes the WebSocket connection.
func (c *CDPClient) Close() {
	c.conn.Close()
}

// ── Tab discovery ────────────────────────────────────────────────────────────

// TabInfo represents a Chrome tab from the /json endpoint.
type TabInfo struct {
	Type        string `json:"type"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	WebSocketURL string `json:"webSocketDebuggerUrl"`
}

// ListTabs returns all open tabs from the CDP /json endpoint.
func ListTabs(cdpPort int) ([]TabInfo, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/json", cdpPort))
	if err != nil {
		return nil, fmt.Errorf("cdp: cannot connect to port %d: %w", cdpPort, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var tabs []TabInfo
	if err := json.Unmarshal(body, &tabs); err != nil {
		return nil, fmt.Errorf("cdp: invalid /json response: %w", err)
	}
	return tabs, nil
}

// ConnectTab connects to the tab at the given index (page tabs only).
func ConnectTab(cdpPort, tabIndex int) (*CDPClient, error) {
	tabs, err := ListTabs(cdpPort)
	if err != nil {
		return nil, err
	}
	var pages []TabInfo
	for _, t := range tabs {
		if t.Type == "page" {
			pages = append(pages, t)
		}
	}
	if tabIndex >= len(pages) {
		return nil, fmt.Errorf("cdp: no page tab at index %d (have %d)", tabIndex, len(pages))
	}
	return NewCDPClient(pages[tabIndex].WebSocketURL)
}

// NewTab opens a new tab and returns a CDPClient connected to it.
func NewTab(cdpPort int) (*CDPClient, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(mustReq("PUT", fmt.Sprintf("http://127.0.0.1:%d/json/new", cdpPort)))
	if err != nil {
		return nil, fmt.Errorf("cdp: new tab: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var tab TabInfo
	if err := json.Unmarshal(body, &tab); err != nil {
		return nil, fmt.Errorf("cdp: invalid new-tab response: %w", err)
	}
	return NewCDPClient(tab.WebSocketURL)
}

func mustReq(method, url string) *http.Request {
	req, _ := http.NewRequest(method, url, nil)
	return req
}
