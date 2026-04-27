package main

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// cmdJS wraps a ref-based JS command with error envelope.
func refJS(ref, body string) string {
	return fmt.Sprintf(`(function(){
var el=window.__agentRefs&&window.__agentRefs[%q];
if(!el)return JSON.stringify({error:'ref %s not found'});
%s
return JSON.stringify({ok:true});
})()`, ref, ref, body)
}

// reactClickJS tries React fiber onClick first (required for WKWebView where el.click()
// does not propagate to React's root-delegated event system), then falls back to el.click().
const reactClickJS = `
el.scrollIntoView({block:'nearest',inline:'nearest'});
el.dispatchEvent(new MouseEvent('mouseover',{bubbles:true}));
el.dispatchEvent(new MouseEvent('mousedown',{bubbles:true,cancelable:true,buttons:1}));
el.dispatchEvent(new MouseEvent('mouseup',{bubbles:true,cancelable:true}));
(function(target){
  var fKey=Object.keys(target).find(function(k){return k.startsWith('__reactFiber');});
  if(fKey){
    var props=target[fKey].memoizedProps;
    if(props&&typeof props.onClick==='function'){
      props.onClick({type:'click',bubbles:true,cancelable:true,target:target,currentTarget:target,
        preventDefault:function(){},stopPropagation:function(){},nativeEvent:{}});
      return;
    }
  }
  target.click();
})(el);`

func cmdClick(ref string) error {
	return runCommand(refJS(ref, reactClickJS))
}

func cmdDblclick(ref string) error {
	return runCommand(refJS(ref, `
el.scrollIntoView({block:'nearest',inline:'nearest'});
el.dispatchEvent(new MouseEvent('dblclick',{bubbles:true,cancelable:true,detail:2}));`))
}

func cmdHover(ref string) error {
	return runCommand(refJS(ref, `
el.scrollIntoView({block:'nearest',inline:'nearest'});
el.dispatchEvent(new MouseEvent('mouseover',{bubbles:true}));
el.dispatchEvent(new MouseEvent('mousemove',{bubbles:true}));`))
}

func cmdFocus(ref string) error {
	return runCommand(refJS(ref, `el.focus();`))
}

func cmdFill(ref, value string) error {
	return runCommand(refJS(ref, fmt.Sprintf(`
el.focus();
if(typeof el.select==='function') el.select();
el.value=%s;
el.dispatchEvent(new Event('input',{bubbles:true}));
el.dispatchEvent(new Event('change',{bubbles:true}));
(function(target,val){
  var fKey=Object.keys(target).find(function(k){return k.startsWith('__reactFiber');});
  if(fKey){
    var props=target[fKey].memoizedProps;
    if(props&&typeof props.onChange==='function'){
      props.onChange({target:target,currentTarget:target,type:'change',
        preventDefault:function(){},stopPropagation:function(){},nativeEvent:{}});
    }
  }
})(el,%s);`, jsonStr(value), jsonStr(value))))
}

func cmdType(ref, text string) error {
	return runCommand(refJS(ref, fmt.Sprintf(`
el.focus();
el.value=(el.value||'')+%s;
el.dispatchEvent(new Event('input',{bubbles:true}));
el.dispatchEvent(new Event('change',{bubbles:true}));`, jsonStr(text))))
}

func cmdPress(key string) error {
	parts := strings.Split(key, "+")
	mainKey := parts[len(parts)-1]
	modifiers := parts[:len(parts)-1]

	ctrlKey := contains(modifiers, "Control") || contains(modifiers, "Ctrl")
	metaKey := contains(modifiers, "Meta") || contains(modifiers, "Command")
	altKey := contains(modifiers, "Alt")
	shiftKey := contains(modifiers, "Shift")

	keyCode := keyCodeFor(mainKey)

	script := fmt.Sprintf(`(function(){
var opts={bubbles:true,cancelable:true,key:%s,code:%s,keyCode:%d,which:%d,ctrlKey:%v,metaKey:%v,altKey:%v,shiftKey:%v};
document.activeElement.dispatchEvent(new KeyboardEvent('keydown',opts));
document.activeElement.dispatchEvent(new KeyboardEvent('keypress',opts));
document.activeElement.dispatchEvent(new KeyboardEvent('keyup',opts));
return JSON.stringify({ok:true});
})()`,
		jsonStr(mainKey), jsonStr("Key"+strings.ToUpper(mainKey[:1])+mainKey[1:]),
		keyCode, keyCode, ctrlKey, metaKey, altKey, shiftKey)

	return runCommand(script)
}

func cmdCheck(ref string) error {
	return runCommand(refJS(ref, `
if(!el.checked){
  el.click();
  if(!el.checked){el.checked=true;el.dispatchEvent(new Event('change',{bubbles:true}));}
}`))
}

func cmdUncheck(ref string) error {
	return runCommand(refJS(ref, `
if(el.checked){
  el.click();
  if(el.checked){el.checked=false;el.dispatchEvent(new Event('change',{bubbles:true}));}
}`))
}

func cmdSelect(ref string, values []string) error {
	valsJSON, _ := jsonMarshal(values)
	return runCommand(refJS(ref, fmt.Sprintf(`
var vals=%s;
Array.from(el.options).forEach(function(o){
  o.selected=vals.indexOf(o.value)>=0||vals.indexOf(o.textContent.trim())>=0;
});
el.dispatchEvent(new Event('change',{bubbles:true}));`, valsJSON)))
}

func cmdScroll(direction string, amount int) error {
	var dx, dy int
	switch strings.ToLower(direction) {
	case "up":
		dy = -amount
	case "down":
		dy = amount
	case "left":
		dx = -amount
	case "right":
		dx = amount
	}
	return runCommand(fmt.Sprintf(`(function(){window.scrollBy(%d,%d);return JSON.stringify({ok:true})})()`, dx, dy))
}

func cmdScrollIntoView(ref string) error {
	return runCommand(refJS(ref, `el.scrollIntoView({block:'center',inline:'center'});`))
}

func cmdGetText(ref string) (string, error) {
	var r jsResult
	err := evalJSON(refJS(ref, `return JSON.stringify({ok:true,value:el.textContent.trim()});`), &r)
	if err != nil {
		return "", err
	}
	if r.Error != "" {
		return "", fmt.Errorf("%s", r.Error)
	}
	if s, ok := r.Value.(string); ok {
		return s, nil
	}
	return fmt.Sprintf("%v", r.Value), nil
}

func cmdGetHTML(ref string) (string, error) {
	var r jsResult
	err := evalJSON(refJS(ref, `return JSON.stringify({ok:true,value:el.innerHTML});`), &r)
	if err != nil {
		return "", err
	}
	if r.Error != "" {
		return "", fmt.Errorf("%s", r.Error)
	}
	if s, ok := r.Value.(string); ok {
		return s, nil
	}
	return "", nil
}

func cmdGetValue(ref string) (string, error) {
	var r jsResult
	err := evalJSON(refJS(ref, `return JSON.stringify({ok:true,value:el.value||''});`), &r)
	if err != nil {
		return "", err
	}
	if r.Error != "" {
		return "", fmt.Errorf("%s", r.Error)
	}
	if s, ok := r.Value.(string); ok {
		return s, nil
	}
	return "", nil
}

func cmdGetAttr(ref, attr string) (string, error) {
	var r jsResult
	err := evalJSON(refJS(ref, fmt.Sprintf(`return JSON.stringify({ok:true,value:el.getAttribute(%s)||''});`, jsonStr(attr))), &r)
	if err != nil {
		return "", err
	}
	if r.Error != "" {
		return "", fmt.Errorf("%s", r.Error)
	}
	if s, ok := r.Value.(string); ok {
		return s, nil
	}
	return "", nil
}

func cmdGetTitle() (string, error) {
	raw, err := eval(`JSON.stringify({ok:true,value:document.title})`)
	if err != nil {
		return "", err
	}
	var r jsResult
	if err := jsonUnmarshal([]byte(raw), &r); err != nil {
		return raw, nil
	}
	if s, ok := r.Value.(string); ok {
		return s, nil
	}
	return "", nil
}

func cmdGetURL() (string, error) {
	raw, err := eval(`JSON.stringify({ok:true,value:window.location.href})`)
	if err != nil {
		return "", err
	}
	var r jsResult
	if err := jsonUnmarshal([]byte(raw), &r); err != nil {
		return raw, nil
	}
	if s, ok := r.Value.(string); ok {
		return s, nil
	}
	return "", nil
}

func cmdIsVisible(ref string) (bool, error) {
	var r jsResult
	err := evalJSON(refJS(ref, `
var rect=el.getBoundingClientRect();
var style=getComputedStyle(el);
var vis=rect.width>0&&rect.height>0&&style.display!=='none'&&style.visibility!=='hidden';
return JSON.stringify({ok:true,value:vis});`), &r)
	if err != nil {
		return false, err
	}
	if r.Error != "" {
		return false, fmt.Errorf("%s", r.Error)
	}
	if b, ok := r.Value.(bool); ok {
		return b, nil
	}
	return false, nil
}

func cmdIsEnabled(ref string) (bool, error) {
	var r jsResult
	err := evalJSON(refJS(ref, `return JSON.stringify({ok:true,value:!el.disabled});`), &r)
	if err != nil {
		return false, err
	}
	if r.Error != "" {
		return false, fmt.Errorf("%s", r.Error)
	}
	if b, ok := r.Value.(bool); ok {
		return b, nil
	}
	return false, nil
}

func cmdIsChecked(ref string) (bool, error) {
	var r jsResult
	err := evalJSON(refJS(ref, `return JSON.stringify({ok:true,value:!!el.checked});`), &r)
	if err != nil {
		return false, err
	}
	if r.Error != "" {
		return false, fmt.Errorf("%s", r.Error)
	}
	if b, ok := r.Value.(bool); ok {
		return b, nil
	}
	return false, nil
}

func cmdWaitMs(ms int) error {
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return nil
}

func cmdWaitText(text string, timeoutMs int) error {
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		raw, err := eval(fmt.Sprintf(`JSON.stringify({ok:true,value:document.body.innerText.indexOf(%s)>=0})`, jsonStr(text)))
		if err == nil {
			var r jsResult
			if jsonUnmarshal([]byte(raw), &r) == nil {
				if b, ok := r.Value.(bool); ok && b {
					return nil
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for text %q", text)
}

func cmdWaitURL(pattern string, timeoutMs int) error {
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		raw, err := eval(`JSON.stringify({ok:true,value:window.location.href})`)
		if err == nil {
			var r jsResult
			if jsonUnmarshal([]byte(raw), &r) == nil {
				if s, ok := r.Value.(string); ok && strings.Contains(s, pattern) {
					return nil
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for URL pattern %q", pattern)
}

func cmdWaitElement(ref string, timeoutMs int) error {
	// Wait for a ref to exist after re-snapshot
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		raw, err := eval(fmt.Sprintf(`JSON.stringify({ok:true,value:!!(window.__agentRefs&&window.__agentRefs[%q])})`, ref))
		if err == nil {
			var r jsResult
			if jsonUnmarshal([]byte(raw), &r) == nil {
				if b, ok := r.Value.(bool); ok && b {
					return nil
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for ref %s", ref)
}

func cmdEval(expr string) (string, error) {
	raw, err := eval(expr)
	if err != nil {
		return "", err
	}
	return raw, nil
}

func cmdEvalBase64(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	return cmdEval(string(decoded))
}

// helpers

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

func keyCodeFor(key string) int {
	switch strings.ToLower(key) {
	case "enter", "return":
		return 13
	case "tab":
		return 9
	case "escape", "esc":
		return 27
	case "backspace":
		return 8
	case "delete":
		return 46
	case "arrowup", "up":
		return 38
	case "arrowdown", "down":
		return 40
	case "arrowleft", "left":
		return 37
	case "arrowright", "right":
		return 39
	case "home":
		return 36
	case "end":
		return 35
	case "pageup":
		return 33
	case "pagedown":
		return 34
	case " ", "space":
		return 32
	case "a":
		return 65
	case "c":
		return 67
	case "v":
		return 86
	case "x":
		return 88
	case "z":
		return 90
	default:
		if len(key) == 1 {
			return int(key[0])
		}
		return 0
	}
}
