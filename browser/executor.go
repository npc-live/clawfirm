package browser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// StepResult is the result of a single step execution.
type StepResult struct {
	OK    bool   `json:"ok"`
	Value any    `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

// StepExecutor maps step types to agent-browser CLI calls.
type StepExecutor struct {
	wsURL     string
	connected bool
}

// NewStepExecutor creates a new executor bound to a Chrome tab WebSocket URL.
func NewStepExecutor(wsURL string) *StepExecutor {
	return &StepExecutor{wsURL: wsURL}
}

// Connect connects to the Chrome tab (once per session).
func (e *StepExecutor) Connect() {
	if e.connected {
		return
	}
	e.run("connect", e.wsURL)
	e.connected = true
}

func (e *StepExecutor) Open(url string) StepResult      { return e.run("open", url) }
func (e *StepExecutor) Click(selector string) StepResult { return e.run("click", selector) }

func (e *StepExecutor) ClickText(text string) StepResult {
	js := fmt.Sprintf(`(function(){
		var el = [...document.querySelectorAll('*')].find(function(e){
			return e.textContent.trim() === %s && e.children.length === 0;
		});
		if(el){ el.click(); return true; }
		return false;
	})()`, jsonString(text))
	r := e.Eval(js)
	if r.Value == nil || r.Value == false {
		return StepResult{OK: false, Error: fmt.Sprintf("text not found: %q", text)}
	}
	return StepResult{OK: true}
}

func (e *StepExecutor) Fill(selector, value string) StepResult {
	js := fmt.Sprintf(`(function(){
		var el = document.querySelector(%s);
		if (!el) return false;
		el.focus();
		var nativeSet = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')
					 || Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, 'value');
		if (nativeSet && nativeSet.set) nativeSet.set.call(el, %s);
		else el.value = %s;
		el.dispatchEvent(new Event('input', { bubbles: true }));
		el.dispatchEvent(new Event('change', { bubbles: true }));
		return true;
	})()`, jsonString(selector), jsonString(value), jsonString(value))
	return e.Eval(js)
}

func (e *StepExecutor) TypeContentEditable(selector, value string) StepResult {
	js := fmt.Sprintf(`(function(){
		var el = document.querySelector(%s);
		if(!el) return false;
		el.focus();
		document.execCommand('selectAll', false, null);
		document.execCommand('insertText', false, %s);
		return el.textContent || el.value || true;
	})()`, jsonString(selector), jsonString(value))
	return e.Eval(js)
}

func (e *StepExecutor) Wait(msOrSelector string) StepResult {
	return e.run("wait", msOrSelector)
}

func (e *StepExecutor) PressKey(key string) StepResult {
	return e.run("press", key)
}

func (e *StepExecutor) Eval(js string) StepResult {
	r := e.run("eval", js)
	if r.OK && r.Value != nil {
		if m, ok := r.Value.(map[string]any); ok {
			if result, exists := m["result"]; exists {
				return StepResult{OK: true, Value: result}
			}
		}
	}
	return r
}

func (e *StepExecutor) Screenshot(path string) StepResult {
	if path != "" {
		return e.run("screenshot", path)
	}
	return e.run("screenshot")
}

func (e *StepExecutor) Upload(selector, filePath string) StepResult {
	return e.run("upload", selector, filePath)
}

func (e *StepExecutor) GetURL() string {
	r := e.run("get", "url")
	if m, ok := r.Value.(map[string]any); ok {
		if u, ok := m["url"].(string); ok {
			return u
		}
	}
	return ""
}

func (e *StepExecutor) Snapshot() string {
	r := e.run("snapshot")
	if s, ok := r.Value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", r.Value)
}

// run executes an agent-browser CLI command and returns parsed JSON result.
func (e *StepExecutor) run(args ...string) StepResult {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = shellQuote(a)
	}
	cmdStr := "agent-browser " + strings.Join(quoted, " ")

	cmd := exec.Command("/bin/bash", "-c", cmdStr)
	cmd.Env = append(os.Environ(), "AGENT_BROWSER_JSON=1")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return StepResult{OK: false, Error: errMsg}
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return StepResult{OK: true}
	}

	// Find last JSON line (agent-browser may emit warnings before JSON).
	lines := strings.Split(out, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if len(line) > 0 && line[0] == '{' {
			var parsed struct {
				Success bool   `json:"success"`
				Data    any    `json:"data"`
				Error   string `json:"error"`
			}
			if json.Unmarshal([]byte(line), &parsed) == nil {
				if !parsed.Success {
					return StepResult{OK: false, Error: parsed.Error}
				}
				return StepResult{OK: true, Value: parsed.Data}
			}
		}
	}
	return StepResult{OK: true}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
