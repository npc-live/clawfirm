#!/usr/bin/env bash
# Build and launch the pi-go desktop app.
# Usage: ./scripts/run-desktop.sh [--rebuild-frontend]
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND_DIR="$REPO_ROOT/cmd/desktop/frontend"
OUT="/tmp/pi-go-desktop"

# -- Frontend -----------------------------------------------------------------
if [[ "${1:-}" == "--rebuild-frontend" ]] || [[ ! -f "$FRONTEND_DIR/dist/index.html" ]]; then
  echo "▸ Building frontend..."
  cd "$FRONTEND_DIR"
  npm install --silent
  npm run build --silent
  echo "  done."
fi

# -- Go binary ----------------------------------------------------------------
echo "▸ Building Go binary..."
cd "$REPO_ROOT"
CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
  go build -tags production -o "$OUT" ./cmd/desktop/
echo "  done → $OUT"

# -- Launch -------------------------------------------------------------------
echo "▸ Launching pi-go desktop..."
open "$OUT"
