#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

expected=$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)
output=$(wails version)
actual=$(printf '%s\n' "$output" | sed -n '/^v[0-9][^[:space:]]*$/p')
if [[ -z "$expected" || "$actual" != "$expected" ]]; then
  echo "check-wails-version: CLI '${actual:-unknown}' != go.mod '$expected'" >&2
  echo "  Install: go install github.com/wailsapp/wails/v2/cmd/wails@$expected" >&2
  exit 1
fi
echo "check-wails-version: CLI matches go.mod ($expected)"
