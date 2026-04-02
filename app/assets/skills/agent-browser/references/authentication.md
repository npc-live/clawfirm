# Authentication Patterns

## Basic Login Flow

```bash
agent-browser open https://app.example.com/login
agent-browser snapshot -i
agent-browser fill @e1 "$USERNAME"
agent-browser fill @e2 "$PASSWORD"
agent-browser click @e3
agent-browser wait --load networkidle
agent-browser snapshot -i
```

## Import from Existing Browser

Start Chrome with CDP, log in manually, then capture state:

```bash
# User starts Chrome with: --remote-debugging-port=9222
agent-browser --auto-connect state save ./auth.json
```

## Persistent Profile

```bash
agent-browser --profile /path/to/chrome-profile open https://app.example.com
# All cookies, IndexedDB, service workers persist automatically
```

## Session Persistence

```bash
# Auto-saves/restores cookies + localStorage by name
agent-browser --session-name myapp open https://app.example.com
```

## Save and Restore State

```bash
# Save after login
agent-browser state save ./auth-state.json

# Restore in future runs
agent-browser state load ./auth-state.json
agent-browser open https://app.example.com/dashboard
```

## Reusable Auth Pattern

```bash
STATE_FILE="/tmp/auth-state.json"

if [[ -f "$STATE_FILE" ]]; then
    agent-browser state load "$STATE_FILE"
    agent-browser open https://app.example.com/dashboard
else
    agent-browser open https://app.example.com/login
    agent-browser snapshot -i
    agent-browser fill @e1 "$USERNAME"
    agent-browser fill @e2 "$PASSWORD"
    agent-browser click @e3
    agent-browser wait --load networkidle
    agent-browser state save "$STATE_FILE"
fi
```

## Security Notes

- Add state files to `.gitignore` (they contain session tokens)
- Use environment variables for credentials, never hardcode
- Delete auth files after automation completes
- `--remote-debugging-port` is a security risk on untrusted machines
