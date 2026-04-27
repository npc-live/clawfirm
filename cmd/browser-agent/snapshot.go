package main

import (
	"encoding/json"
	"fmt"
)

// snapshotOptions mirrors agent-browser's SnapshotOptions.
type snapshotOptions struct {
	Interactive bool
	Compact     bool
	Depth       int // 0 = unlimited
	Selector    string
	Reset       bool // reset refs before snapshot (default true per snapshot call)
}

type snapshotResult struct {
	Snapshot string `json:"snapshot"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Error    string `json:"error"`
}

func takeSnapshot(opts snapshotOptions) (snapshotResult, error) {
	optsJSON, _ := json.Marshal(map[string]any{
		"interactive": opts.Interactive,
		"compact":     opts.Compact,
		"depth":       opts.Depth,
		"selector":    opts.Selector,
		"reset":       opts.Reset,
	})

	script := fmt.Sprintf("(%s)(%s)", snapshotJS, string(optsJSON))

	var result snapshotResult
	raw, err := eval(script)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return result, fmt.Errorf("snapshot parse error: %w (raw: %q)", err, raw)
	}
	if result.Error != "" {
		return result, fmt.Errorf("%s", result.Error)
	}
	return result, nil
}

// snapshotJS is a self-invoking function that walks the DOM and returns an
// agent-browser compatible accessibility tree with @eN refs.
// It accepts an options object as its sole argument.
const snapshotJS = `function(opts) {
  if (opts.reset !== false) {
    window.__agentRefs = {};
    window.__agentNextId = 1;
  }
  if (!window.__agentRefs) window.__agentRefs = {};
  if (!window.__agentNextId) window.__agentNextId = 1;

  var refs = window.__agentRefs;
  var nextId = window.__agentNextId;

  var INTERACTIVE_ROLES = {
    button:1,link:1,textbox:1,checkbox:1,radio:1,combobox:1,listbox:1,
    menuitem:1,menuitemcheckbox:1,menuitemradio:1,option:1,searchbox:1,
    slider:1,spinbutton:1,'switch':1,tab:1,treeitem:1
  };
  var CONTENT_ROLES = {
    heading:1,cell:1,gridcell:1,columnheader:1,rowheader:1,
    listitem:1,article:1,region:1,main:1,navigation:1
  };
  var INVISIBLE_STRUCTURAL = {
    group:1,list:1,table:1,row:1,rowgroup:1,grid:1,treegrid:1,
    menu:1,menubar:1,toolbar:1,tablist:1,tree:1,directory:1,
    document:1,application:1,presentation:1,none:1
  };

  function getRole(el) {
    var r = el.getAttribute('role');
    if (r) return r.toLowerCase();
    var tag = el.tagName.toLowerCase();
    var type = (el.getAttribute('type') || '').toLowerCase();
    switch (tag) {
      case 'button': return 'button';
      case 'a': return el.hasAttribute('href') ? 'link' : 'generic';
      case 'input':
        if (type === 'hidden') return null;
        if (type === 'checkbox') return 'checkbox';
        if (type === 'radio') return 'radio';
        if (type === 'range') return 'slider';
        if (type === 'number') return 'spinbutton';
        if (type === 'search') return 'searchbox';
        if (type === 'button' || type === 'submit' || type === 'reset') return 'button';
        return 'textbox';
      case 'textarea': return 'textbox';
      case 'select': return el.multiple ? 'listbox' : 'combobox';
      case 'h1': return 'heading';
      case 'h2': return 'heading';
      case 'h3': return 'heading';
      case 'h4': return 'heading';
      case 'h5': return 'heading';
      case 'h6': return 'heading';
      case 'li': return 'listitem';
      case 'td': return 'cell';
      case 'th': return 'columnheader';
      case 'nav': return 'navigation';
      case 'main': return 'main';
      case 'article': return 'article';
      case 'section':
        return (el.hasAttribute('aria-label') || el.hasAttribute('aria-labelledby')) ? 'region' : 'generic';
      case 'img': return 'img';
      case 'summary': return 'button';
      case 'label': return 'LabelText';
      default: return 'generic';
    }
  }

  function getName(el) {
    var v = el.getAttribute('aria-label');
    if (v && v.trim()) return v.trim();
    var lblId = el.getAttribute('aria-labelledby');
    if (lblId) {
      var lbl = document.getElementById(lblId);
      if (lbl) { v = lbl.textContent.trim(); if (v) return v; }
    }
    var ph = el.getAttribute('placeholder');
    if (ph && ph.trim()) return ph.trim();
    var alt = el.getAttribute('alt');
    if (alt && alt.trim()) return alt.trim();
    if (el.id) {
      var forLbl = document.querySelector('label[for="' + el.id + '"]');
      if (forLbl) { v = forLbl.textContent.trim(); if (v) return v; }
    }
    return (el.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 200);
  }

  function getHeadingLevel(el) {
    var tag = el.tagName.toLowerCase();
    if (tag.length === 2 && tag[0] === 'h') {
      var n = parseInt(tag[1]);
      if (n >= 1 && n <= 6) return n;
    }
    var lvl = el.getAttribute('aria-level');
    if (lvl) return parseInt(lvl);
    return null;
  }

  function isCursorInteractive(el) {
    var style = getComputedStyle(el);
    var hasCursor = style.cursor === 'pointer';
    var hasClick = el.hasAttribute('onclick') || (el.onclick !== null && typeof el.onclick === 'function');
    var tabIdx = el.getAttribute('tabindex');
    var hasTab = tabIdx !== null && tabIdx !== '-1';
    var ce = el.getAttribute('contenteditable');
    var isEdit = ce === '' || ce === 'true';
    if (!hasCursor && !hasClick && !hasTab && !isEdit) return false;
    if (hasCursor && !hasClick && !hasTab && !isEdit) {
      var p = el.parentElement;
      if (p && getComputedStyle(p).cursor === 'pointer') return false;
    }
    var r = el.getBoundingClientRect();
    return r.width > 0 && r.height > 0;
  }

  function shouldSkip(el) {
    if (el.hidden) return true;
    if (el.getAttribute('aria-hidden') === 'true') return true;
    var p = el.parentElement;
    while (p) {
      if (p.getAttribute('aria-hidden') === 'true') return true;
      p = p.parentElement;
    }
    var s = getComputedStyle(el);
    return s.display === 'none' || s.visibility === 'hidden';
  }

  var lines = [];
  var root = document.body;
  if (opts.selector) {
    root = document.querySelector(opts.selector);
    if (!root) return JSON.stringify({error: 'selector not found: ' + opts.selector});
  }

  function walk(el, depth) {
    if (!el || el.nodeType !== 1) return;
    if (shouldSkip(el)) return;
    if (opts.depth && depth > opts.depth) return;

    var role = getRole(el);
    if (!role) return;

    var name = getName(el);
    var shouldRef = false;
    if (INTERACTIVE_ROLES[role]) {
      shouldRef = true;
    } else if (CONTENT_ROLES[role] && name) {
      shouldRef = true;
    } else if (role === 'generic' || INVISIBLE_STRUCTURAL[role]) {
      shouldRef = false;
      if (isCursorInteractive(el)) shouldRef = true;
    } else {
      shouldRef = isCursorInteractive(el);
    }

    var refId = '';
    if (shouldRef) {
      refId = 'e' + nextId++;
      refs[refId] = el;
    }

    var emitted = !opts.interactive || !!refId;
    if (emitted) {
      var indent = '';
      for (var d = 0; d < depth; d++) indent += '  ';
      var line = indent + '- ' + role;
      if (name) line += ' "' + name.replace(/\\/g, '\\\\').replace(/"/g, '\\"') + '"';

      var attrs = [];
      var lvl = getHeadingLevel(el);
      if (lvl) attrs.push('level=' + lvl);
      if (role === 'checkbox' || role === 'radio' || role === 'switch') {
        attrs.push('checked=' + !!el.checked);
      }
      var expanded = el.getAttribute('aria-expanded');
      if (expanded !== null) attrs.push('expanded=' + expanded);
      var selected = el.getAttribute('aria-selected');
      if (selected === 'true') attrs.push('selected');
      if (el.disabled) attrs.push('disabled');
      if (el.required) attrs.push('required');
      if (refId) attrs.push('ref=' + refId);

      if (attrs.length) line += ' [' + attrs.join(', ') + ']';

      if (el.value && el.value !== name &&
          (role === 'textbox' || role === 'combobox' || role === 'spinbutton' || role === 'searchbox')) {
        line += ': ' + el.value;
      }

      lines.push(line);
    }

    var childDepth = emitted ? depth + 1 : depth;
    for (var i = 0; i < el.children.length; i++) {
      walk(el.children[i], childDepth);
    }
  }

  walk(root, 0);
  window.__agentRefs = refs;
  window.__agentNextId = nextId;

  var output = lines.join('\n').trim();

  if (opts.compact && output) {
    var allLines = output.split('\n');
    var keep = [];
    for (var i = 0; i < allLines.length; i++) keep.push(false);
    for (var i = 0; i < allLines.length; i++) {
      if (allLines[i].indexOf('ref=') >= 0 || allLines[i].indexOf(': ') >= 0) {
        keep[i] = true;
        var myIndent = allLines[i].length - allLines[i].replace(/^ +/, '').length;
        for (var j = i - 1; j >= 0; j--) {
          var anc = allLines[j].length - allLines[j].replace(/^ +/, '').length;
          if (anc < myIndent) {
            keep[j] = true;
            if (anc === 0) break;
            myIndent = anc;
          }
        }
      }
    }
    var kept = [];
    for (var i = 0; i < allLines.length; i++) {
      if (keep[i]) kept.push(allLines[i]);
    }
    output = kept.join('\n');
  }

  if (!output) {
    return JSON.stringify({
      snapshot: opts.interactive ? '(no interactive elements)' : '(empty page)',
      title: document.title,
      url: window.location.href
    });
  }

  return JSON.stringify({snapshot: output, title: document.title, url: window.location.href});
}`
