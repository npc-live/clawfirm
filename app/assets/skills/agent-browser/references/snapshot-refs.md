# Snapshot and Refs

Compact element references for token-efficient browser automation.

## How Refs Work

Traditional approach: Full DOM/HTML -> AI parses -> CSS selector -> Action (~3000-5000 tokens)

agent-browser approach: Compact snapshot -> @refs assigned -> Direct interaction (~200-400 tokens)

## The Snapshot Command

```bash
# Interactive snapshot (recommended)
agent-browser snapshot -i
```

### Output Format

```
Page: Example Site - Home
URL: https://example.com

@e1 [header]
  @e2 [nav]
    @e3 [a] "Home"
    @e4 [a] "Products"
    @e5 [a] "About"
  @e6 [button] "Sign In"

@e7 [main]
  @e8 [h1] "Welcome"
  @e9 [form]
    @e10 [input type="email"] placeholder="Email"
    @e11 [input type="password"] placeholder="Password"
    @e12 [button type="submit"] "Log In"
```

## Using Refs

```bash
agent-browser click @e6
agent-browser fill @e10 "user@example.com"
agent-browser fill @e11 "password123"
agent-browser click @e12
```

## Ref Lifecycle

Refs are invalidated when the page changes.

```bash
agent-browser snapshot -i          # @e1 [button] "Next"
agent-browser click @e1            # Triggers page change
agent-browser snapshot -i          # MUST re-snapshot! @e1 is now different
```

## Ref Notation

```
@e1 [tag type="value"] "text content" placeholder="hint"
```

### Common Patterns

```
@e1 [button] "Submit"                    # Button with text
@e2 [input type="email"]                 # Email input
@e3 [a href="/page"] "Link Text"         # Anchor link
@e4 [select]                             # Dropdown
@e5 [textarea] placeholder="Message"     # Text area
@e6 [checkbox] checked                   # Checked checkbox
```

## Iframes

Snapshots auto-detect and inline iframe content. Refs inside iframes work directly.

```bash
agent-browser snapshot -i
# @e2 [Iframe] "payment-frame"
#   @e3 [input] "Card number"
#   @e4 [button] "Pay"

agent-browser fill @e3 "4111111111111111"
agent-browser click @e4
```

## Best Practices

1. Always snapshot before interacting
2. Re-snapshot after navigation or dynamic changes
3. Use scoped snapshots for complex pages: `agent-browser snapshot @e9`
4. Scroll + re-snapshot if elements aren't visible
