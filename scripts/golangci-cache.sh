#!/usr/bin/env bash
set -euo pipefail
if [[ -z "${GOLANGCI_LINT_CACHE:-}" ]]; then
  root=$(git rev-parse --show-toplevel)
  hash=$(printf '%s' "$root" | sha256sum | cut -c1-12)
  export GOLANGCI_LINT_CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/golangci-twr/$hash"
fi
exec golangci-lint "$@"
