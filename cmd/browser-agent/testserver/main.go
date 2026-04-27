// testserver: mock eval server for browser-agent integration tests.
// It wraps agent-browser CLI as the JS executor, allowing browser-agent
// to be tested against a real browser without needing the Wails app running.
//
// Usage:
//
//	go run ./cmd/browser-agent/testserver &
//	browser-agent snapshot -i
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func main() {
	url := "https://example.com"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	// Open the URL in agent-browser so its daemon is running
	if err := agentBrowser("open", url); err != nil {
		log.Fatalf("agent-browser open %s: %v", url, err)
	}
	fmt.Printf("Opened %s in agent-browser\n", url)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/eval", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Script string `json:"script"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		result, err := evalViaAgentBrowser(req.Script)
		if err != nil {
			result = fmt.Sprintf("eval error: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"result": result})
	})

	addr := "localhost:9310"
	fmt.Printf("Mock eval server on http://%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func agentBrowser(args ...string) error {
	cmd := exec.Command("agent-browser", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func evalViaAgentBrowser(script string) (string, error) {
	// agent-browser eval --stdin reads script from stdin.
	// agent-browser JSON-encodes the JS return value, so we unwrap one layer.
	// e.g. JS returns `{"ok":true}` → agent-browser outputs `"{\"ok\":true}"` → we return `{"ok":true}`
	cmd := exec.Command("agent-browser", "eval", "--stdin")
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		return "", fmt.Errorf("%w: %s", err, stderr)
	}
	raw := strings.TrimSpace(string(out))
	// agent-browser wraps the result in JSON quotes — unwrap if it's a JSON string
	var inner string
	if json.Unmarshal([]byte(raw), &inner) == nil {
		return inner, nil
	}
	return raw, nil
}
