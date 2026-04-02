# Session Management

## Named Sessions

Use `--session` for isolated browser contexts:

```bash
agent-browser --session auth open https://app.example.com/login
agent-browser --session public open https://example.com
```

Each session has independent cookies, localStorage, history, and tabs.

## Common Patterns

### Concurrent Scraping

```bash
agent-browser --session site1 open https://site1.com &
agent-browser --session site2 open https://site2.com &
wait

agent-browser --session site1 get text body > site1.txt
agent-browser --session site2 get text body > site2.txt

agent-browser --session site1 close
agent-browser --session site2 close
```

### A/B Testing

```bash
agent-browser --session variant-a open "https://app.com?variant=a"
agent-browser --session variant-b open "https://app.com?variant=b"

agent-browser --session variant-a screenshot /tmp/variant-a.png
agent-browser --session variant-b screenshot /tmp/variant-b.png
```

## Cleanup

```bash
agent-browser --session auth close   # Close specific session
agent-browser session list           # List active sessions
agent-browser close --all            # Close all sessions
```

## Best Practices

1. Name sessions semantically (e.g., `github-auth`, `docs-scrape`)
2. Always close sessions when done to prevent leaked processes
3. Add state files to `.gitignore`
4. Use timeouts for automated scripts: `timeout 60 agent-browser ...`
