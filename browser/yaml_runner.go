package browser

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ── YAML types ───────────────────────────────────────────────────────────────

// AdapterYAML is the top-level schema for a YAML adapter file.
type AdapterYAML struct {
	Platform   string                `yaml:"platform"`
	LoginURL   string                `yaml:"login_url"`
	LoginCheck LoginCheck            `yaml:"login_check"`
	Commands   map[string]CommandDef `yaml:"commands"`
}

// LoginCheck indicates how to verify a logged-in state.
type LoginCheck struct {
	Cookie string `yaml:"cookie"`
}

// CommandDef describes a single command with named args and execution steps.
type CommandDef struct {
	Args  []string `yaml:"args"`
	Steps []Step   `yaml:"steps"`
}

// Step is a single automation step parsed from YAML.
// Exactly one field should be non-zero.
type Step struct {
	Open           string          `yaml:"open,omitempty"`
	Click          any             `yaml:"click,omitempty"`
	Fill           *FillStep       `yaml:"fill,omitempty"`
	TypeRich       *FillStep       `yaml:"type_rich,omitempty"`
	Wait           any             `yaml:"wait,omitempty"`
	Eval           string          `yaml:"eval,omitempty"`
	Capture        *CaptureStep    `yaml:"capture,omitempty"`
	Upload         *UploadStep     `yaml:"upload,omitempty"`
	Screenshot     string          `yaml:"screenshot,omitempty"`
	Extract        *ExtractDef     `yaml:"extract,omitempty"`
	Return         []ReturnRow     `yaml:"return,omitempty"`
	Assert         *AssertStep     `yaml:"assert,omitempty"`
	Key            string          `yaml:"key,omitempty"`
	KeyboardInsert string          `yaml:"keyboard_insert,omitempty"`
	InsertText     string          `yaml:"insert_text,omitempty"`
	WaitUntil      *WaitUntilStep  `yaml:"wait_until,omitempty"`
}

type FillStep struct {
	Selector string `yaml:"selector"`
	Value    string `yaml:"value"`
}

type CaptureStep struct {
	Name string `yaml:"name"`
	Eval string `yaml:"eval"`
}

type UploadStep struct {
	Selector string `yaml:"selector"`
	File     string `yaml:"file"`
}

type ExtractDef struct {
	Selector string         `yaml:"selector"`
	Fields   map[string]any `yaml:"fields"`
}

type ReturnRow struct {
	Field string `yaml:"field"`
	Value string `yaml:"value"`
}

type AssertStep struct {
	Eval    string `yaml:"eval"`
	Message string `yaml:"message,omitempty"`
}

type WaitUntilStep struct {
	Eval     string `yaml:"eval"`
	Timeout  int    `yaml:"timeout,omitempty"`  // ms, default 120000
	Interval int    `yaml:"interval,omitempty"` // ms, default 2000
}

// ── Load ─────────────────────────────────────────────────────────────────────

// LoadAdapterYAML reads and parses a YAML adapter file.
func LoadAdapterYAML(path string) (*AdapterYAML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var a AdapterYAML
	if err := yaml.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &a, nil
}

// ── Runner ───────────────────────────────────────────────────────────────────

// RunYAMLCommand loads a YAML adapter and executes the given command.
func RunYAMLCommand(adapterPath, commandName string, argValues []string, cdpPort int) ([]map[string]any, error) {
	adapter, err := LoadAdapterYAML(adapterPath)
	if err != nil {
		return nil, err
	}
	cmdDef, ok := adapter.Commands[commandName]
	if !ok {
		names := make([]string, 0, len(adapter.Commands))
		for k := range adapter.Commands {
			names = append(names, k)
		}
		return nil, fmt.Errorf("unknown command %q; available: %s", commandName, strings.Join(names, ", "))
	}

	// Build variable map from positional args.
	vars := make(map[string]string)
	for i, name := range cmdDef.Args {
		if i < len(argValues) {
			vars[name] = argValues[i]
		}
	}

	// Ensure Chrome is running with CDP enabled.
	if err := ensureChromeRunning(cdpPort); err != nil {
		return nil, err
	}

	// Open a new tab in the default browser context so that automation
	// can access the user's existing cookies and login sessions (required
	// for posting to social media platforms).
	cdpClient, err := NewTab(cdpPort)
	if err != nil {
		return nil, fmt.Errorf("open new tab: %w", err)
	}
	defer cdpClient.Close()

	// Connect agent-browser to the CDP port directly (not a per-tab WebSocket
	// URL). This lets agent-browser pick/manage the tab internally and avoids
	// "Session with given id not found" errors from stale ws URLs.
	exec := NewStepExecutor(fmt.Sprintf("%d", cdpPort))
	exec.Connect()

	log.Printf("browser: connected to %s", adapter.Platform)
	log.Printf("browser: running %s %s %s", adapter.Platform, commandName, strings.Join(argValues, " "))

	// Execute steps.
	var extractedRows []map[string]any
	var returnRows []ReturnRow

	for _, step := range cmdDef.Steps {
		if err := executeStep(step, vars, exec, cdpClient, &extractedRows, &returnRows); err != nil {
			return nil, err
		}
	}

	// Build output.
	if extractedRows != nil {
		return extractedRows, nil
	}
	if returnRows != nil {
		out := make([]map[string]any, len(returnRows))
		for i, r := range returnRows {
			out[i] = map[string]any{
				"field": interpolate(r.Field, vars),
				"value": interpolate(r.Value, vars),
			}
		}
		return out, nil
	}
	return nil, nil
}

func executeStep(
	step Step,
	vars map[string]string,
	exec *StepExecutor,
	cdpClient *CDPClient,
	extractedRows *[]map[string]any,
	returnRows *[]ReturnRow,
) error {
	switch {
	case step.Open != "":
		u := interpolate(step.Open, vars)
		log.Printf("  → open %s", u)
		exec.Open(u)

	case step.Click != nil:
		switch v := step.Click.(type) {
		case string:
			sel := interpolate(v, vars)
			log.Printf("  → click %q", sel)
			mustOK(exec.Click(sel), "click failed: "+sel)
		case map[string]any:
			if text, ok := v["text"].(string); ok {
				t := interpolate(text, vars)
				log.Printf("  → click text=%q", t)
				mustOK(exec.ClickText(t), "text not found: "+t)
			} else if sel, ok := v["selector"].(string); ok {
				s := interpolate(sel, vars)
				log.Printf("  → click selector=%q", s)
				mustOK(exec.Click(s), "click failed: "+s)
			}
		}

	case step.Fill != nil:
		sel := interpolate(step.Fill.Selector, vars)
		val := interpolate(step.Fill.Value, vars)
		log.Printf("  → fill %q", sel)
		mustOK(exec.Fill(sel, val), "fill failed: "+sel)

	case step.TypeRich != nil:
		sel := interpolate(step.TypeRich.Selector, vars)
		val := interpolate(step.TypeRich.Value, vars)
		log.Printf("  → type_rich %q", sel)
		mustOK(exec.TypeContentEditable(sel, val), "type_rich failed: "+sel)

	case step.Wait != nil:
		switch v := step.Wait.(type) {
		case int:
			log.Printf("  → wait %dms", v)
			exec.Wait(strconv.Itoa(v))
		case float64:
			ms := int(v)
			log.Printf("  → wait %dms", ms)
			exec.Wait(strconv.Itoa(ms))
		case string:
			sel := interpolate(v, vars)
			log.Printf("  → wait selector=%q", sel)
			exec.Wait(sel)
		case map[string]any:
			if sel, ok := v["selector"].(string); ok {
				s := interpolate(sel, vars)
				log.Printf("  → wait selector=%q", s)
				exec.Wait(s)
			}
		}

	case step.Eval != "":
		js := interpolate(step.Eval, vars)
		log.Printf("  → eval ...")
		exec.Eval(js)

	case step.Capture != nil:
		js := interpolate(step.Capture.Eval, vars)
		log.Printf("  → capture %s", step.Capture.Name)
		r := exec.Eval(js)
		vars[step.Capture.Name] = fmt.Sprintf("%v", r.Value)

	case step.Upload != nil:
		sel := interpolate(step.Upload.Selector, vars)
		f := interpolate(step.Upload.File, vars)
		log.Printf("  → upload %q ← %s", sel, f)
		mustOK(exec.Upload(sel, f), "upload failed: "+sel)

	case step.Screenshot != "":
		p := interpolate(step.Screenshot, vars)
		log.Printf("  → screenshot → %s", p)
		exec.Screenshot(p)

	case step.Extract != nil:
		log.Printf("  → extract %q", step.Extract.Selector)
		rows := runExtract(exec, step.Extract, vars)
		*extractedRows = rows

	case step.Return != nil:
		*returnRows = step.Return

	case step.Assert != nil:
		js := interpolate(step.Assert.Eval, vars)
		r := exec.Eval(js)
		if r.Value == nil || r.Value == false || r.Value == "" {
			msg := step.Assert.Message
			if msg == "" {
				msg = "assertion failed: " + step.Assert.Eval
			}
			return fmt.Errorf(msg)
		}
		log.Printf("  → assert OK")

	case step.Key != "":
		k := interpolate(step.Key, vars)
		log.Printf("  → key %q", k)
		if k == "Enter" || k == "Control+Enter" {
			modifiers := 0
			if strings.HasPrefix(k, "Control") {
				modifiers = 2
			}
			cdpClient.DispatchKeyEvent("keyDown", "Enter", "Enter", 13, modifiers)
			cdpClient.DispatchKeyEvent("keyUp", "Enter", "Enter", 13, modifiers)
		} else {
			exec.PressKey(k)
		}

	case step.KeyboardInsert != "":
		text := interpolate(step.KeyboardInsert, vars)
		log.Printf("  → keyboard_insert %q", truncStr(text, 40))
		for _, ch := range text {
			s := string(ch)
			code := int(ch)
			cdpClient.DispatchKeyEvent("keyDown", s, "", code, 0)
			cdpClient.Send("Input.dispatchKeyEvent", map[string]any{
				"type": "char", "key": s, "text": s,
			})
			cdpClient.DispatchKeyEvent("keyUp", s, "", code, 0)
		}

	case step.InsertText != "":
		text := interpolate(step.InsertText, vars)
		log.Printf("  → insert_text %q", truncStr(text, 40))
		cdpClient.InsertText(text)

	case step.WaitUntil != nil:
		timeout := step.WaitUntil.Timeout
		if timeout <= 0 {
			timeout = 120000
		}
		interval := step.WaitUntil.Interval
		if interval <= 0 {
			interval = 2000
		}
		js := interpolate(step.WaitUntil.Eval, vars)
		log.Printf("  → wait_until (timeout: %ds)", timeout/1000)
		deadline := time.Now().Add(time.Duration(timeout) * time.Millisecond)
		resolved := false
		for time.Now().Before(deadline) {
			r := exec.Eval(js)
			if r.OK && r.Value != nil && r.Value != false && r.Value != "" && r.Value != 0.0 {
				resolved = true
				break
			}
			exec.Wait(strconv.Itoa(interval))
		}
		if !resolved {
			return fmt.Errorf("wait_until timed out after %ds", timeout/1000)
		}
		log.Printf("     condition met")
	}
	return nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

var tmplRe = regexp.MustCompile(`\{\{(.+?)\}\}`)

func interpolate(template string, vars map[string]string) string {
	return tmplRe.ReplaceAllStringFunc(template, func(match string) string {
		expr := strings.TrimSpace(match[2 : len(match)-2])
		if v, ok := vars[expr]; ok {
			return v
		}
		return match
	})
}

func mustOK(r StepResult, msg string) {
	if !r.OK {
		log.Printf("WARNING: %s: %s", msg, r.Error)
	}
}

func runExtract(exec *StepExecutor, def *ExtractDef, vars map[string]string) []map[string]any {
	fieldsJSON, _ := json.Marshal(def.Fields)
	selectorJSON, _ := json.Marshal(def.Selector)
	js := fmt.Sprintf(`(function(){
		var fields = %s;
		var results = [];
		document.querySelectorAll(%s).forEach(function(item, i) {
			var row = { index: i + 1 };
			Object.keys(fields).forEach(function(key) {
				var spec = fields[key];
				if (typeof spec === 'string') {
					var el = (spec === ':scope') ? item : item.querySelector(spec);
					if (!el) el = item;
					row[key] = el ? el.textContent.trim() : '';
				} else {
					var el2 = (spec.selector === ':scope') ? item : item.querySelector(spec.selector);
					if (!el2) el2 = item;
					row[key] = el2 ? (spec.attr === 'href' ? (el2.href || el2.getAttribute(spec.attr)) : el2.getAttribute(spec.attr)) || el2.textContent.trim() : '';
				}
			});
			if (Object.values(row).some(function(v){ return v && v !== i + 1; })) results.push(row);
		});
		return JSON.stringify(results);
	})()`, string(fieldsJSON), string(selectorJSON))

	r := exec.Eval(js)
	s, _ := r.Value.(string)
	if s == "" {
		return nil
	}
	var rows []map[string]any
	json.Unmarshal([]byte(s), &rows)
	return rows
}

func resolveTabWsURL(adapter *AdapterYAML, cdpPort int) (string, error) {
	tabs, err := ListTabs(cdpPort)
	if err != nil {
		return "", fmt.Errorf("cannot connect to Chrome CDP (port %d): %w\n\nPlease launch Chrome with: --remote-debugging-port=%d", cdpPort, err, cdpPort)
	}

	var pages []TabInfo
	for _, t := range tabs {
		if t.Type == "page" {
			pages = append(pages, t)
		}
	}
	if len(pages) == 0 {
		return "", fmt.Errorf("no page tabs open in Chrome")
	}

	// Prefer a tab already on the platform domain.
	parsed, _ := url.Parse(adapter.LoginURL)
	if parsed != nil {
		domain := strings.TrimPrefix(parsed.Hostname(), "www.")
		for _, t := range pages {
			if strings.Contains(t.URL, domain) && !strings.Contains(t.URL, "creator") {
				return t.WebSocketURL, nil
			}
		}
	}

	// Check saved session cookie.
	cookies, _ := LoadSession(adapter.Platform)
	if cookies != nil {
		for _, c := range cookies {
			if c.Name == adapter.LoginCheck.Cookie {
				return pages[0].WebSocketURL, nil
			}
		}
	}

	// No existing session — use first tab (caller must handle login).
	return pages[0].WebSocketURL, nil
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ensureChromeRunning checks if Chrome CDP is reachable on the given port.
// If not, it launches Chrome with the social-cli profile and waits for CDP.
func ensureChromeRunning(cdpPort int) error {
	if isCDPReachable(cdpPort) {
		return nil
	}

	chromePath := findChromePath()
	if chromePath == "" {
		return fmt.Errorf("Chrome not found; please launch Chrome with --remote-debugging-port=%d", cdpPort)
	}

	home, _ := os.UserHomeDir()
	profileDir := filepath.Join(home, ".social-cli", "chrome-profile")
	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		profileDir = filepath.Join(home, ".clawfirm", "chrome-profile")
	}

	cmd := exec.Command(chromePath,
		fmt.Sprintf("--remote-debugging-port=%d", cdpPort),
		"--user-data-dir="+profileDir,
		"--no-first-run",
		"--no-default-browser-check",
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch Chrome: %w", err)
	}
	go func() { _ = cmd.Wait() }()

	// Poll for CDP readiness.
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		if isCDPReachable(cdpPort) {
			return nil
		}
	}
	return fmt.Errorf("Chrome launched but CDP not ready on port %d after 10s", cdpPort)
}

func isCDPReachable(port int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/json/version", port))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func findChromePath() string {
	if runtime.GOOS == "darwin" {
		candidates := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}
