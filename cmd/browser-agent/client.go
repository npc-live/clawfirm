package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

var httpClient = &http.Client{Timeout: 35 * time.Second}

// evalURL returns the eval server URL, reading BROWSER_AGENT_PORT env var.
func evalURL() string {
	if port := os.Getenv("BROWSER_AGENT_PORT"); port != "" {
		return "http://localhost:" + port + "/api/eval"
	}
	return "http://localhost:9310/api/eval"
}

// evalResult is what the eval server returns.
type evalResult struct {
	Result string `json:"result"`
}

// eval sends a JS script to the Wails WKWebView eval server and returns the result string.
func eval(script string) (string, error) {
	body, _ := json.Marshal(map[string]string{"script": script})
	resp, err := httpClient.Post(evalURL(), "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("eval server unreachable at %s (is Clawfirm running?): %w", evalURL(), err)
	}
	defer resp.Body.Close()
	var r evalResult
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return r.Result, nil
}

// evalJSON sends JS and parses the JSON result.
func evalJSON(script string, out any) error {
	raw, err := eval(script)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return fmt.Errorf("bad JS result %q: %w", raw, err)
	}
	return nil
}

// jsResult is the common success/error envelope returned by command JS.
type jsResult struct {
	OK    bool   `json:"ok"`
	Value any    `json:"value"`
	Error string `json:"error"`
}

// runCommand executes a one-liner JS command and checks for errors.
func runCommand(script string) error {
	var r jsResult
	if err := evalJSON(script, &r); err != nil {
		return err
	}
	if r.Error != "" {
		return fmt.Errorf("%s", r.Error)
	}
	return nil
}

// stripRef strips the leading @ from a ref like @e1 → e1.
func stripRef(ref string) string {
	if len(ref) > 0 && ref[0] == '@' {
		return ref[1:]
	}
	return ref
}
