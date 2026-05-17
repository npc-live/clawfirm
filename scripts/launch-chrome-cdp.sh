#!/usr/bin/env bash
# launch-chrome-cdp.sh — Safely launch Chrome with CDP on port 9222.
#
# Mirrors the logic in app/app.go:BrowserLaunchChrome():
#   1. If CDP is already reachable on the target port, exit immediately (reuse).
#   2. Otherwise, launch a NEW Chrome instance with a dedicated --user-data-dir.
#   3. NEVER kill existing Chrome processes.
#
# Usage:
#   ./scripts/launch-chrome-cdp.sh          # default port 9222
#   CDP_PORT=9333 ./scripts/launch-chrome-cdp.sh

set -euo pipefail

CDP_PORT="${CDP_PORT:-9222}"
PROFILE_DIR="${HOME}/.social-cli/chrome-profile"

if [ ! -d "$PROFILE_DIR" ]; then
  PROFILE_DIR="${HOME}/.clawfirm/chrome-profile"
  mkdir -p "$PROFILE_DIR"
fi

# ── Step 1: Check if CDP is already reachable ────────────────────────────────
if curl -s --max-time 2 "http://127.0.0.1:${CDP_PORT}/json/version" >/dev/null 2>&1; then
  echo "CDP already active on port ${CDP_PORT} — reusing existing Chrome."
  curl -s "http://127.0.0.1:${CDP_PORT}/json/version" | python3 -c "
import sys, json
info = json.load(sys.stdin)
print(f\"Browser: {info.get('Browser', 'unknown')}\")
print(f\"WebSocket: {info.get('webSocketDebuggerUrl', 'N/A')}\")
" 2>/dev/null || true
  exit 0
fi

# ── Step 2: Find Chrome binary ───────────────────────────────────────────────
CHROME_BIN=""
case "$(uname -s)" in
  Darwin)
    for candidate in \
      "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
      "/Applications/Chromium.app/Contents/MacOS/Chromium" \
      "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"; do
      if [ -x "$candidate" ]; then
        CHROME_BIN="$candidate"
        break
      fi
    done
    ;;
  Linux)
    for name in google-chrome chromium-browser chromium; do
      if command -v "$name" >/dev/null 2>&1; then
        CHROME_BIN="$(command -v "$name")"
        break
      fi
    done
    ;;
esac

if [ -z "$CHROME_BIN" ]; then
  echo "Error: Chrome/Chromium not found." >&2
  exit 1
fi

# ── Step 3: Launch Chrome alongside any existing instance ────────────────────
echo "Launching Chrome (CDP port ${CDP_PORT}, profile ${PROFILE_DIR})..."
"$CHROME_BIN" \
  --remote-debugging-port="${CDP_PORT}" \
  --user-data-dir="${PROFILE_DIR}" \
  --no-first-run \
  --no-default-browser-check \
  --disable-blink-features=AutomationControlled \
  &>/dev/null &

# ── Step 4: Poll for CDP readiness (up to 5s) ───────────────────────────────
for i in $(seq 1 10); do
  sleep 0.5
  if curl -s --max-time 1 "http://127.0.0.1:${CDP_PORT}/json/version" >/dev/null 2>&1; then
    echo "Chrome CDP ready on port ${CDP_PORT}."
    curl -s "http://127.0.0.1:${CDP_PORT}/json/version" | python3 -c "
import sys, json
info = json.load(sys.stdin)
print(f\"Browser: {info.get('Browser', 'unknown')}\")
print(f\"WebSocket: {info.get('webSocketDebuggerUrl', 'N/A')}\")
" 2>/dev/null || true
    exit 0
  fi
done

echo "Error: Chrome launched but CDP not ready after 5s." >&2
exit 1
