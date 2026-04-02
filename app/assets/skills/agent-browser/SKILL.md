---
name: agent-browser
description: >-
  Browser automation skill using the agent-browser CLI. Use this skill when the
  user asks you to interact with websites, fill forms, scrape pages, take
  screenshots, test web apps, or perform any browser-based task. Trigger on
  phrases like "open this URL", "click the button", "fill the form",
  "screenshot the page", "scrape the site", "test the login flow", or
  "browse to".
---

# agent-browser

A browser automation skill powered by the `agent-browser` CLI. It controls Chrome/Chromium via CDP and uses compact element refs for fast, token-efficient interactions.

## Prerequisites

Install `agent-browser` if not already available:

```bash
npm install -g agent-browser && agent-browser install
```

Or via Homebrew: `brew install agent-browser`

## Core Workflow

Every browser automation follows this pattern:

1. **Open** a URL: `agent-browser open <url>`
2. **Snapshot** interactive elements: `agent-browser snapshot -i`
3. **Interact** using refs from the snapshot: `agent-browser click @e1`, `agent-browser fill @e2 "text"`
4. **Re-snapshot** after any page change to get fresh refs

Commands persist via a background daemon — chain them with `&&` in a single bash call or run them sequentially.

## Quick Examples

### Navigate and extract text

```bash
agent-browser open https://example.com
agent-browser snapshot -i
agent-browser get text @e1
```

### Fill a form

```bash
agent-browser open https://example.com/login
agent-browser snapshot -i
# Identify the form fields from snapshot output, then:
agent-browser fill @e3 "user@example.com"
agent-browser fill @e4 "password123"
agent-browser click @e5
agent-browser wait --load networkidle
agent-browser snapshot -i
```

### Take a screenshot

```bash
agent-browser open https://example.com
agent-browser wait --load networkidle
agent-browser screenshot ./capture.png
# Or full page:
agent-browser screenshot --full ./full-page.png
```

### Scrape data

```bash
agent-browser open https://example.com
agent-browser snapshot -i
agent-browser get text body
```

## Key Commands

| Command | Description |
|---------|-------------|
| `open <url>` | Navigate to URL |
| `snapshot -i` | Get interactive elements with @refs |
| `click @ref` | Click an element |
| `fill @ref "text"` | Clear field and type text |
| `type @ref "text"` | Type without clearing |
| `select @ref "value"` | Select dropdown option |
| `check @ref` / `uncheck @ref` | Toggle checkbox |
| `press Enter` | Press a key |
| `scroll down 500` | Scroll the page |
| `get text @ref` | Get element text |
| `get url` | Get current URL |
| `get title` | Get page title |
| `screenshot [path]` | Capture screenshot |
| `wait @ref` | Wait for element |
| `wait --load networkidle` | Wait for network idle |
| `wait --text "Success"` | Wait for text to appear |
| `close` | Close browser session |

## Important Rules

1. **Always snapshot before interacting** — refs don't exist until you snapshot
2. **Re-snapshot after navigation** — refs are invalidated when the page changes
3. **Re-snapshot after dynamic changes** — clicking a button that opens a dropdown requires a new snapshot to see the new elements
4. **Use `wait` for slow pages** — `wait --load networkidle` before snapshot on heavy pages
5. **Close sessions when done** — `agent-browser close` to prevent leaked processes

## References

For the full command reference, read `references/commands.md`.
For snapshot/ref details, read `references/snapshot-refs.md`.
For authentication patterns, read `references/authentication.md`.
For session management, read `references/session-management.md`.
