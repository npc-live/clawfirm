// browser-agent: agent-browser compatible CLI for WKWebView via Clawfirm eval server.
// Commands are pixel-level compatible with agent-browser, but use JS eval instead of CDP.
// Requires Clawfirm app running (eval server on localhost:9310).
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "snapshot":
		err = cmdSnapshot(args)
	case "click":
		err = cmdClickCmd(args)
	case "dblclick":
		err = cmdDblclickCmd(args)
	case "hover":
		err = cmdHoverCmd(args)
	case "focus":
		err = cmdFocusCmd(args)
	case "fill":
		err = cmdFillCmd(args)
	case "type":
		err = cmdTypeCmd(args)
	case "press", "key":
		err = cmdPressCmd(args)
	case "check":
		err = cmdCheckCmd(args)
	case "uncheck":
		err = cmdUncheckCmd(args)
	case "select":
		err = cmdSelectCmd(args)
	case "scroll":
		err = cmdScrollCmd(args)
	case "scrollintoview", "scrollinto":
		err = cmdScrollIntoViewCmd(args)
	case "get":
		err = cmdGetCmd(args)
	case "is":
		err = cmdIsCmd(args)
	case "wait":
		err = cmdWaitCmd(args)
	case "eval":
		err = cmdEvalCmd(args)
	case "find":
		err = cmdFindCmd(args)
	case "--help", "-h", "help":
		printUsage()
	case "--version", "-V":
		fmt.Println("browser-agent (Clawfirm WKWebView) compatible with agent-browser CLI")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func cmdSnapshot(args []string) error {
	fs := flag.NewFlagSet("snapshot", flag.ExitOnError)
	interactive := fs.Bool("i", false, "interactive elements only (recommended)")
	compact := fs.Bool("c", false, "compact output")
	depth := fs.Int("d", 0, "limit depth")
	selector := fs.String("s", "", "scope to CSS selector")
	fs.Parse(args)

	result, err := takeSnapshot(snapshotOptions{
		Interactive: *interactive,
		Compact:     *compact,
		Depth:       *depth,
		Selector:    *selector,
		Reset:       true,
	})
	if err != nil {
		return err
	}

	if result.Title != "" || result.URL != "" {
		fmt.Printf("Page: %s\nURL: %s\n\n", result.Title, result.URL)
	}
	fmt.Println(result.Snapshot)
	return nil
}

func cmdClickCmd(args []string) error {
	fs := flag.NewFlagSet("click", flag.ExitOnError)
	fs.Parse(args)
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: click @eN")
	}
	return cmdClick(stripRef(fs.Arg(0)))
}

func cmdDblclickCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: dblclick @eN")
	}
	return cmdDblclick(stripRef(args[0]))
}

func cmdHoverCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: hover @eN")
	}
	return cmdHover(stripRef(args[0]))
}

func cmdFocusCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: focus @eN")
	}
	return cmdFocus(stripRef(args[0]))
}

func cmdFillCmd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: fill @eN \"text\"")
	}
	return cmdFill(stripRef(args[0]), args[1])
}

func cmdTypeCmd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: type @eN \"text\"")
	}
	return cmdType(stripRef(args[0]), args[1])
}

func cmdPressCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: press KEY (e.g. Enter, Control+a, Escape)")
	}
	return cmdPress(args[0])
}

func cmdCheckCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: check @eN")
	}
	return cmdCheck(stripRef(args[0]))
}

func cmdUncheckCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: uncheck @eN")
	}
	return cmdUncheck(stripRef(args[0]))
}

func cmdSelectCmd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: select @eN value [value...]")
	}
	return cmdSelect(stripRef(args[0]), args[1:])
}

func cmdScrollCmd(args []string) error {
	direction := "down"
	amount := 300
	if len(args) >= 1 {
		direction = args[0]
	}
	if len(args) >= 2 {
		n, err := strconv.Atoi(args[1])
		if err == nil {
			amount = n
		}
	}
	return cmdScroll(direction, amount)
}

func cmdScrollIntoViewCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: scrollintoview @eN")
	}
	return cmdScrollIntoView(stripRef(args[0]))
}

func cmdGetCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: get text|html|value|attr|title|url [@eN] [attr]")
	}
	sub := args[0]
	switch sub {
	case "title":
		v, err := cmdGetTitle()
		if err != nil {
			return err
		}
		fmt.Println(v)
	case "url":
		v, err := cmdGetURL()
		if err != nil {
			return err
		}
		fmt.Println(v)
	case "text":
		if len(args) < 2 {
			return fmt.Errorf("usage: get text @eN")
		}
		v, err := cmdGetText(stripRef(args[1]))
		if err != nil {
			return err
		}
		fmt.Println(v)
	case "html":
		if len(args) < 2 {
			return fmt.Errorf("usage: get html @eN")
		}
		v, err := cmdGetHTML(stripRef(args[1]))
		if err != nil {
			return err
		}
		fmt.Println(v)
	case "value":
		if len(args) < 2 {
			return fmt.Errorf("usage: get value @eN")
		}
		v, err := cmdGetValue(stripRef(args[1]))
		if err != nil {
			return err
		}
		fmt.Println(v)
	case "attr":
		if len(args) < 3 {
			return fmt.Errorf("usage: get attr @eN attrName")
		}
		v, err := cmdGetAttr(stripRef(args[1]), args[2])
		if err != nil {
			return err
		}
		fmt.Println(v)
	default:
		return fmt.Errorf("unknown get subcommand: %s", sub)
	}
	return nil
}

func cmdIsCmd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: is visible|enabled|checked @eN")
	}
	sub := args[0]
	ref := stripRef(args[1])
	var v bool
	var err error
	switch sub {
	case "visible":
		v, err = cmdIsVisible(ref)
	case "enabled":
		v, err = cmdIsEnabled(ref)
	case "checked":
		v, err = cmdIsChecked(ref)
	default:
		return fmt.Errorf("unknown is subcommand: %s", sub)
	}
	if err != nil {
		return err
	}
	fmt.Println(v)
	return nil
}

func cmdWaitCmd(args []string) error {
	fs := flag.NewFlagSet("wait", flag.ExitOnError)
	text := fs.String("text", "", "wait for text to appear")
	shortText := fs.String("t", "", "wait for text (short)")
	urlPat := fs.String("url", "", "wait for URL pattern")
	shortURL := fs.String("u", "", "wait for URL pattern (short)")
	timeout := fs.Int("timeout", 30000, "timeout in ms")
	fs.Parse(args)

	waitText := first(*text, *shortText)
	waitURL := first(*urlPat, *shortURL)

	switch {
	case waitText != "":
		return cmdWaitText(waitText, *timeout)
	case waitURL != "":
		return cmdWaitURL(waitURL, *timeout)
	case fs.NArg() > 0:
		arg := fs.Arg(0)
		// numeric → sleep ms; @eN → wait for ref
		if n, err := strconv.Atoi(arg); err == nil {
			return cmdWaitMs(n)
		}
		if strings.HasPrefix(arg, "@") {
			return cmdWaitElement(stripRef(arg), *timeout)
		}
		return fmt.Errorf("invalid wait argument: %s", arg)
	default:
		return fmt.Errorf("usage: wait <ms> | wait @eN | wait --text \"...\" | wait --url \"...\"")
	}
}

func cmdEvalCmd(args []string) error {
	fs := flag.NewFlagSet("eval", flag.ExitOnError)
	b64 := fs.String("b", "", "base64 encoded script")
	fs.Parse(args)

	var result string
	var err error

	if *b64 != "" {
		result, err = cmdEvalBase64(*b64)
	} else if fs.NArg() > 0 {
		result, err = cmdEval(fs.Arg(0))
	} else {
		return fmt.Errorf("usage: eval \"expr\" | eval -b <base64>")
	}
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}

// cmdFindCmd implements: find text "Sign In" click | find role button click --name "Submit"
func cmdFindCmd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: find text|role|label|placeholder|alt|testid <value> [action]")
	}
	locatorType := args[0]
	locatorValue := args[1]
	action := "click"
	if len(args) >= 3 {
		action = args[2]
	}

	var findJS string
	switch locatorType {
	case "text":
		findJS = fmt.Sprintf(`
var found=null;
var _txt=%s;
var _all=document.querySelectorAll('button,a,input,label,span,div,p,h1,h2,h3,h4,h5,h6,li');
var _candidates=[];
for(var _i=0;_i<_all.length;_i++){if(_all[_i].textContent.trim()===_txt)_candidates.push(_all[_i]);}
if(_candidates.length){
  // prefer interactive leaf elements over container divs
  var _pref=['BUTTON','A','INPUT','LABEL'];
  found=_candidates.find(function(c){return _pref.indexOf(c.tagName)>=0;})||_candidates[_candidates.length-1];
}`, jsonStr(locatorValue))
	case "role":
		findJS = fmt.Sprintf(`
var roleName=%s;
var all=document.querySelectorAll('[role="'+roleName+'"],'+roleName);
var found=all.length>0?all[0]:null;`, jsonStr(locatorValue))
	case "label":
		findJS = fmt.Sprintf(`
var lbl=Array.from(document.querySelectorAll('label')).find(function(l){return l.textContent.trim()===%s;});
var found=lbl?lbl.control||lbl:null;`, jsonStr(locatorValue))
	case "placeholder":
		findJS = fmt.Sprintf(`var found=document.querySelector('[placeholder=%s]');`, jsonStr(locatorValue))
	case "alt":
		findJS = fmt.Sprintf(`var found=document.querySelector('[alt=%s]');`, jsonStr(locatorValue))
	case "testid":
		findJS = fmt.Sprintf(`var found=document.querySelector('[data-testid=%s]');`, jsonStr(locatorValue))
	default:
		return fmt.Errorf("unknown locator type: %s", locatorType)
	}

	var actionJS string
	switch action {
	case "click":
		actionJS = `found.scrollIntoView({block:'nearest'});
found.dispatchEvent(new MouseEvent('mouseover',{bubbles:true}));
found.dispatchEvent(new MouseEvent('mousedown',{bubbles:true,cancelable:true,buttons:1}));
found.dispatchEvent(new MouseEvent('mouseup',{bubbles:true,cancelable:true}));
(function(target){
  var fKey=Object.keys(target).find(function(k){return k.startsWith('__reactFiber');});
  if(fKey){var props=target[fKey].memoizedProps;
    if(props&&typeof props.onClick==='function'){
      props.onClick({type:'click',bubbles:true,cancelable:true,target:target,currentTarget:target,
        preventDefault:function(){},stopPropagation:function(){},nativeEvent:{}});
      return;}}
  target.click();
})(found);`
	case "fill":
		if len(args) < 4 {
			return fmt.Errorf("find fill requires a value: find %s %q fill \"text\"", locatorType, locatorValue)
		}
		actionJS = fmt.Sprintf(`found.focus();found.value=%s;
found.dispatchEvent(new Event('input',{bubbles:true}));
found.dispatchEvent(new Event('change',{bubbles:true}));
(function(t,v){var fk=Object.keys(t).find(function(k){return k.startsWith('__reactFiber');});
if(fk){var p=t[fk].memoizedProps;if(p&&typeof p.onChange==='function')
  p.onChange({target:t,currentTarget:t,type:'change',preventDefault:function(){},stopPropagation:function(){},nativeEvent:{}});
}})(found,%s);`, jsonStr(args[3]), jsonStr(args[3]))
	case "hover":
		actionJS = `found.dispatchEvent(new MouseEvent('mouseover',{bubbles:true}));`
	default:
		actionJS = `found.click();`
	}

	script := fmt.Sprintf(`(function(){
%s
if(!found)return JSON.stringify({error:'element not found: %s %s'});
%s
return JSON.stringify({ok:true});
})()`, findJS, locatorType, locatorValue, actionJS)

	return runCommand(script)
}

func first(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func printUsage() {
	fmt.Print(`browser-agent — agent-browser compatible CLI for WKWebView via Clawfirm

SNAPSHOT
  browser-agent snapshot            full accessibility tree
  browser-agent snapshot -i         interactive elements only (recommended)
  browser-agent snapshot -c         compact output
  browser-agent snapshot -d 3       limit depth to 3
  browser-agent snapshot -s "#main" scope to CSS selector

INTERACTIONS (use @refs from snapshot)
  browser-agent click @e1
  browser-agent dblclick @e1
  browser-agent fill @e2 "text"     clear and type
  browser-agent type @e2 "text"     type without clearing
  browser-agent press Enter         key press (e.g. Control+a, Escape, Tab)
  browser-agent hover @e1
  browser-agent check @e1
  browser-agent uncheck @e1
  browser-agent select @e1 "value"
  browser-agent scroll down 500     (up/down/left/right, default 300px)
  browser-agent scrollintoview @e1

GET INFORMATION
  browser-agent get text @e1
  browser-agent get html @e1
  browser-agent get value @e1
  browser-agent get attr @e1 href
  browser-agent get title
  browser-agent get url

CHECK STATE
  browser-agent is visible @e1
  browser-agent is enabled @e1
  browser-agent is checked @e1

WAIT
  browser-agent wait 2000              wait ms
  browser-agent wait @e1               wait for ref
  browser-agent wait --text "Success"  wait for text
  browser-agent wait --url "/dashboard" wait for URL

SEMANTIC LOCATORS
  browser-agent find text "Sign In" click
  browser-agent find role button click
  browser-agent find label "Email" fill "user@test.com"
  browser-agent find placeholder "Search" click
  browser-agent find testid "submit-btn" click

JAVASCRIPT
  browser-agent eval "document.title"
  browser-agent eval -b <base64-script>
`)
}
