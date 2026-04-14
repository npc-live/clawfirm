package browser

import (
	"context"
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

// ProgressFunc is called during execution to report progress to the frontend.
// Each call replaces the previous progress text shown to the user.
type ProgressFunc func(text string)

// RunYAMLCommand loads a YAML adapter and executes the given command.
// When healer is non-nil, failed steps (selector not found, assert, wait_until)
// will attempt auto-healing via an LLM before returning an error.
// onProgress may be nil; when set, it emits SSE updates so the user can see
// what's happening in real time.
func RunYAMLCommand(ctx context.Context, adapterPath, commandName string, argValues []string, cdpPort int, healer *HealerConfig, onProgress ProgressFunc) ([]map[string]any, error) {
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

	progress := func(text string) {
		if onProgress != nil {
			onProgress(text)
		}
	}

	// Ensure Chrome is running with CDP enabled.
	progress(fmt.Sprintf("🌐 Connecting to Chrome (port %d)...", cdpPort))
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

	// Verify login state before running any steps.
	if adapter.LoginCheck.Cookie != "" {
		progress(fmt.Sprintf("🔑 Checking %s login...", adapter.Platform))
		if err := verifyLoginWithRetry(ctx, cdpClient, adapter, progress); err != nil {
			return nil, err
		}
	}

	// Connect agent-browser to the CDP port directly (not a per-tab WebSocket
	// URL). This lets agent-browser pick/manage the tab internally and avoids
	// "Session with given id not found" errors from stale ws URLs.
	exec := NewStepExecutor(fmt.Sprintf("%d", cdpPort))
	exec.Connect()
	globalExec = exec // allow interpolate() to evaluate JS expressions

	log.Printf("browser: connected to %s", adapter.Platform)
	log.Printf("browser: running %s %s %s", adapter.Platform, commandName, strings.Join(argValues, " "))
	progress(fmt.Sprintf("▶ Running %s/%s...", adapter.Platform, commandName))

	// Execute steps.
	var extractedRows []map[string]any
	var returnRows []ReturnRow
	steps := cmdDef.Steps

	for i, step := range steps {
		progress(fmt.Sprintf("▶ Running %s/%s — step %d/%d...", adapter.Platform, commandName, i+1, len(steps)))
		err := executeStep(ctx, step, vars, exec, cdpClient, &extractedRows, &returnRows, healer, adapterPath, adapter.Platform, commandName, i, steps, progress)
		if err != nil {
			return nil, err
		}
	}

	// Save session cookies for future restoration.
	if _, err := CaptureSession(cdpClient, adapter.Platform); err != nil {
		log.Printf("browser: warning: failed to save session for %s: %v", adapter.Platform, err)
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

// verifyLoginWithRetry checks login state. If not logged in, it opens the
// login URL in the browser and polls for the cookie to appear, sending SSE
// progress updates so the user knows what's happening.
func verifyLoginWithRetry(ctx context.Context, client *CDPClient, adapter *AdapterYAML, progress ProgressFunc) error {
	cookieName := adapter.LoginCheck.Cookie

	if hasLoginCookie(client, cookieName) {
		log.Printf("browser: %s login verified (cookie %s found)", adapter.Platform, cookieName)
		return nil
	}

	// Cookie not found — try restoring a saved session.
	log.Printf("browser: %s cookie %s not found, attempting session restore...", adapter.Platform, cookieName)
	restored, err := RestoreSession(client, adapter.Platform)
	if err != nil {
		log.Printf("browser: warning: session restore failed for %s: %v", adapter.Platform, err)
	}

	if restored && hasLoginCookie(client, cookieName) {
		log.Printf("browser: %s session restored successfully", adapter.Platform)
		return nil
	}

	// Not logged in — open the login URL and wait for the user to log in.
	progress(fmt.Sprintf("⚠️ %s: not logged in. Opening %s — please log in, I'll wait...", adapter.Platform, adapter.LoginURL))
	log.Printf("browser: %s not logged in, opening login URL and waiting...", adapter.Platform)

	// Navigate the tab to the login URL so the user can log in.
	client.Navigate(adapter.LoginURL, 3000)

	// Poll for login cookie (up to 5 minutes).
	loginTimeout := 5 * time.Minute
	pollInterval := 3 * time.Second
	deadline := time.Now().Add(loginTimeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}

		if hasLoginCookie(client, cookieName) {
			progress(fmt.Sprintf("✅ %s login detected! Continuing...", adapter.Platform))
			log.Printf("browser: %s login detected after waiting", adapter.Platform)
			// Save session for future use.
			if _, err := CaptureSession(client, adapter.Platform); err != nil {
				log.Printf("browser: warning: failed to save session: %v", err)
			}
			return nil
		}

		remaining := time.Until(deadline).Truncate(time.Second)
		progress(fmt.Sprintf("⏳ Waiting for %s login... (%s remaining)", adapter.Platform, remaining))
	}

	return fmt.Errorf(
		"%s: login timed out after %s. Cookie %q not found.\n"+
			"Please log in at %s and retry.",
		adapter.Platform, loginTimeout, cookieName, adapter.LoginURL)
}

// hasLoginCookie checks if the named cookie exists in the browser.
func hasLoginCookie(client *CDPClient, cookieName string) bool {
	cookies, err := client.GetAllCookies()
	if err != nil {
		return false
	}
	for _, c := range cookies {
		if c.Name == cookieName {
			return true
		}
	}
	return false
}

func executeStep(
	ctx context.Context,
	step Step,
	vars map[string]string,
	exec *StepExecutor,
	cdpClient *CDPClient,
	extractedRows *[]map[string]any,
	returnRows *[]ReturnRow,
	healer *HealerConfig,
	adapterPath, platform, commandName string,
	stepIndex int,
	allSteps []Step,
	progress ProgressFunc,
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
			if err := mustOKOrHeal(ctx, exec.Click(sel), "click failed: "+sel, healer, step, exec, adapterPath, platform, commandName, stepIndex, allSteps, vars); err != nil {
				return err
			}
		case map[string]any:
			if text, ok := v["text"].(string); ok {
				t := interpolate(text, vars)
				log.Printf("  → click text=%q", t)
				if err := mustOKOrHeal(ctx, exec.ClickText(t), "text not found: "+t, healer, step, exec, adapterPath, platform, commandName, stepIndex, allSteps, vars); err != nil {
					return err
				}
			} else if sel, ok := v["selector"].(string); ok {
				s := interpolate(sel, vars)
				log.Printf("  → click selector=%q", s)
				if err := mustOKOrHeal(ctx, exec.Click(s), "click failed: "+s, healer, step, exec, adapterPath, platform, commandName, stepIndex, allSteps, vars); err != nil {
					return err
				}
			}
		}

	case step.Fill != nil:
		sel := interpolate(step.Fill.Selector, vars)
		val := interpolate(step.Fill.Value, vars)
		log.Printf("  → fill %q", sel)
		if err := mustOKOrHeal(ctx, exec.Fill(sel, val), "fill failed: "+sel, healer, step, exec, adapterPath, platform, commandName, stepIndex, allSteps, vars); err != nil {
			return err
		}

	case step.TypeRich != nil:
		sel := interpolate(step.TypeRich.Selector, vars)
		val := interpolate(step.TypeRich.Value, vars)
		log.Printf("  → type_rich %q", sel)
		if err := mustOKOrHeal(ctx, exec.TypeContentEditable(sel, val), "type_rich failed: "+sel, healer, step, exec, adapterPath, platform, commandName, stepIndex, allSteps, vars); err != nil {
			return err
		}

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
			if err := waitSelectorWithTimeout(exec, sel, 30000); err != nil {
				log.Printf("  → wait WARNING: %v", err)
			}
		case map[string]any:
			if sel, ok := v["selector"].(string); ok {
				s := interpolate(sel, vars)
				log.Printf("  → wait selector=%q", s)
				if err := waitSelectorWithTimeout(exec, s, 30000); err != nil {
					log.Printf("  → wait WARNING: %v", err)
				}
			}
		}

	case step.Eval != "":
		js := interpolate(step.Eval, vars)
		log.Printf("  → eval ...")
		r := exec.Eval(js)
		if !r.OK {
			log.Printf("  → eval WARNING: %s", r.Error)
		}

	case step.Capture != nil:
		js := interpolate(step.Capture.Eval, vars)
		log.Printf("  → capture %s", step.Capture.Name)
		r := exec.Eval(js)
		if !r.OK {
			log.Printf("  → capture WARNING: eval failed for %s: %s", step.Capture.Name, r.Error)
		}
		val := fmt.Sprintf("%v", r.Value)
		if val == "<nil>" {
			val = ""
			log.Printf("  → capture WARNING: %s is nil", step.Capture.Name)
		}
		vars[step.Capture.Name] = val

	case step.Upload != nil:
		sel := interpolate(step.Upload.Selector, vars)
		f := interpolate(step.Upload.File, vars)
		log.Printf("  → upload %q ← %s", sel, f)
		if err := mustOKOrHeal(ctx, exec.Upload(sel, f), "upload failed: "+sel, healer, step, exec, adapterPath, platform, commandName, stepIndex, allSteps, vars); err != nil {
			return err
		}

	case step.Screenshot != "":
		p := interpolate(step.Screenshot, vars)
		log.Printf("  → screenshot → %s", p)
		exec.Screenshot(p)

	case step.Extract != nil:
		log.Printf("  → extract %q", step.Extract.Selector)
		rows := runExtract(exec, step.Extract, vars)
		if len(rows) == 0 && healer != nil {
			log.Printf("  → extract returned 0 rows, attempting heal...")
			if progress != nil {
				progress(fmt.Sprintf("🔧 Healing empty extract at step %d...", stepIndex+1))
			}
			failure := StepFailure{
				AdapterPath: adapterPath, Platform: platform,
				CommandName: commandName, StepIndex: stepIndex,
				Step: step, Preceding: precedingSteps(allSteps, stepIndex),
				Remaining: remainingSteps(allSteps, stepIndex),
				Error: fmt.Sprintf("extract %q returned 0 rows", step.Extract.Selector), Vars: vars,
			}
			if healed, action, _ := healStep(ctx, healer, failure, exec); healed && action != nil && action.Selector != "" {
				// Retry extract with the healed selector.
				healedDef := &ExtractDef{Selector: action.Selector, Fields: step.Extract.Fields}
				rows = runExtract(exec, healedDef, vars)
				log.Printf("  → extract (healed) → %d rows", len(rows))
			}
		}
		if len(rows) == 0 {
			log.Printf("  → extract WARNING: 0 rows returned")
		}
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
			if healer != nil {
				if progress != nil {
					progress(fmt.Sprintf("🔧 Healing assert failure at step %d...", stepIndex+1))
				}
				failure := StepFailure{
					AdapterPath: adapterPath, Platform: platform,
					CommandName: commandName, StepIndex: stepIndex,
					Step: step, Preceding: precedingSteps(allSteps, stepIndex),
					Remaining: remainingSteps(allSteps, stepIndex),
					Error: msg, Vars: vars,
				}
				if healed, _, healErr := healStep(ctx, healer, failure, exec); healed {
					// Re-evaluate the assertion after healing.
					r2 := exec.Eval(js)
					if r2.Value != nil && r2.Value != false && r2.Value != "" {
						log.Printf("  → assert OK (healed)")
						break
					}
				} else if healErr != nil {
					log.Printf("healer: %v", healErr)
				}
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
			errMsg := fmt.Sprintf("wait_until timed out after %ds", timeout/1000)
			if healer != nil {
				if progress != nil {
					progress(fmt.Sprintf("🔧 Healing wait_until timeout at step %d...", stepIndex+1))
				}
				failure := StepFailure{
					AdapterPath: adapterPath, Platform: platform,
					CommandName: commandName, StepIndex: stepIndex,
					Step: step, Preceding: precedingSteps(allSteps, stepIndex),
					Remaining: remainingSteps(allSteps, stepIndex),
					Error: errMsg, Vars: vars,
				}
				if healed, _, healErr := healStep(ctx, healer, failure, exec); healed {
					log.Printf("     condition met (healed)")
					break
				} else if healErr != nil {
					log.Printf("healer: %v", healErr)
				}
			}
			return fmt.Errorf(errMsg)
		}
		log.Printf("     condition met")
	}
	return nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

var tmplRe = regexp.MustCompile(`\{\{(.+?)\}\}`)

// interpolate replaces {{varName}} with values from vars.
// For expressions containing operators (===, ?, +, etc.), it evaluates
// them as JavaScript in the browser via evalExpr (if set).
func interpolate(template string, vars map[string]string) string {
	return tmplRe.ReplaceAllStringFunc(template, func(match string) string {
		expr := strings.TrimSpace(match[2 : len(match)-2])
		// Simple variable lookup first.
		if v, ok := vars[expr]; ok {
			return v
		}
		// If it looks like an expression (has operators), evaluate as JS.
		if isExpression(expr) {
			return evalExpression(expr, vars)
		}
		return match
	})
}

// isExpression returns true if the template expression contains JS operators
// and is not a simple variable name.
func isExpression(expr string) bool {
	for _, op := range []string{"===", "==", "!==", "!=", "&&", "||", "?", "+", "-", ">", "<"} {
		if strings.Contains(expr, op) {
			return true
		}
	}
	return false
}

// evalExpression evaluates a JS expression with variables substituted.
// Falls back to the raw expression on any error.
func evalExpression(expr string, vars map[string]string) string {
	// Build a JS snippet that declares all vars and evaluates the expression.
	var sb strings.Builder
	sb.WriteString("(function(){")
	for k, v := range vars {
		vJSON, _ := json.Marshal(v)
		fmt.Fprintf(&sb, "var %s=%s;", k, string(vJSON))
	}
	fmt.Fprintf(&sb, "return (%s);", expr)
	sb.WriteString("})()")

	// Use a throwaway executor on the current page (eval-only, no connection needed
	// because the caller's executor is already connected). We use a global to avoid
	// threading the executor through interpolate's simple signature.
	if globalExec != nil {
		r := globalExec.Eval(sb.String())
		if r.OK && r.Value != nil {
			return fmt.Sprintf("%v", r.Value)
		}
	}
	return "{{" + expr + "}}"
}

// globalExec is set during RunYAMLCommand so interpolate can evaluate expressions.
var globalExec *StepExecutor

// mustOKOrHeal checks a step result. When healer is nil, it logs a warning
// (preserving original behavior). When healer is set, it attempts auto-healing.
func mustOKOrHeal(
	ctx context.Context,
	r StepResult, msg string,
	healer *HealerConfig,
	step Step,
	exec *StepExecutor,
	adapterPath, platform, commandName string,
	stepIndex int,
	allSteps []Step,
	vars map[string]string,
) error {
	if r.OK {
		return nil
	}
	if healer == nil {
		log.Printf("WARNING: %s: %s", msg, r.Error)
		return nil
	}
	failure := StepFailure{
		AdapterPath: adapterPath,
		Platform:    platform,
		CommandName: commandName,
		StepIndex:   stepIndex,
		Step:        step,
		Preceding:   precedingSteps(allSteps, stepIndex),
		Remaining:   remainingSteps(allSteps, stepIndex),
		Error:       fmt.Sprintf("%s: %s", msg, r.Error),
		Vars:        vars,
	}
	healed, _, healErr := healStep(ctx, healer, failure, exec)
	if healed {
		return nil
	}
	if healErr != nil {
		log.Printf("healer: %v", healErr)
	}
	return fmt.Errorf("%s: %s", msg, r.Error)
}

// waitSelectorWithTimeout polls for a CSS selector to appear, with a timeout (ms).
// Unlike exec.Wait(selector) which can hang indefinitely, this returns an error.
func waitSelectorWithTimeout(exec *StepExecutor, selector string, timeoutMs int) error {
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		r := exec.Eval(fmt.Sprintf("!!document.querySelector(%s)", jsonString(selector)))
		if r.OK && r.Value == true {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("selector %q not found after %ds", selector, timeoutMs/1000)
}

// precedingSteps returns the steps before stepIndex.
func precedingSteps(allSteps []Step, stepIndex int) []Step {
	if stepIndex > 0 {
		return allSteps[:stepIndex]
	}
	return nil
}

// remainingSteps returns the steps after stepIndex.
func remainingSteps(allSteps []Step, stepIndex int) []Step {
	if stepIndex+1 < len(allSteps) {
		return allSteps[stepIndex+1:]
	}
	return nil
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
// If not, it checks whether Chrome is already running without CDP (profile
// conflict risk) and either prompts the user or launches Chrome with the
// social-cli profile.
func ensureChromeRunning(cdpPort int) error {
	if isCDPReachable(cdpPort) {
		return nil
	}

	// If Chrome is already running without CDP, launching a new instance
	// would use a temporary profile (due to profile lock), losing all
	// login sessions. Abort with a helpful message instead.
	if isChromeProcessRunning() {
		home, _ := os.UserHomeDir()
		return fmt.Errorf(
			"Chrome is running but CDP is not enabled on port %d.\n"+
				"The existing Chrome holds the profile lock, so launching a new instance would use a temporary profile without your login sessions.\n\n"+
				"Please restart Chrome with:\n"+
				"  /Applications/Google\\ Chrome.app/Contents/MacOS/Google\\ Chrome --remote-debugging-port=%d --user-data-dir=%s/.social-cli/chrome-profile &\n\n"+
				"Or close Chrome entirely and retry (we will launch it with the correct flags).",
			cdpPort, cdpPort, home)
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

	log.Printf("browser: launching Chrome with profile %s", profileDir)
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

// isChromeProcessRunning checks if a Chrome process is already running.
func isChromeProcessRunning() bool {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("pgrep", "-x", "Google Chrome").Output()
		return err == nil && len(strings.TrimSpace(string(out))) > 0
	case "linux":
		out, err := exec.Command("pgrep", "-f", "chrome|chromium").Output()
		return err == nil && len(strings.TrimSpace(string(out))) > 0
	default:
		return false
	}
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
