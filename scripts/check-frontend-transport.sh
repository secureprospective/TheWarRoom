#!/usr/bin/env bash
# check-frontend-transport.sh: the frontend talks to the server only over the hosted transport.
# Since PR-H (T710, 11-24) Wails is gone, so the old per-feature adapter rules collapse to one:
# no file under frontend/src (tests included) may import frontend/wailsjs or reach window.go.
set -euo pipefail
cd "$(dirname "$0")/.."
hits=$(grep -rnE "wailsjs|window\.go\b|window as \{ ?go\b" frontend/src --include='*.ts' --include='*.tsx' || true)
if [ -n "$hits" ]; then
  echo "check-frontend-transport: desktop bindings are gone; use the hosted transport instead:" >&2
  echo "$hits" >&2
  exit 1
fi
echo "check-frontend-transport: no desktop bindings in frontend/src"
