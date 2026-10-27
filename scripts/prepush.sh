#!/usr/bin/env bash
# Quick local check; the full suite is CI's job.
# Docs only: skip. Go: bindings, lint, build all, test directly changed packages.
# Gate scripts or an unknown base: lint and build; CI runs the rest.
# Frontend: bindings, type check, tests, build and bundle budget.
set -euo pipefail

fallback=0
from=${PRE_COMMIT_FROM_REF:-}
to=${PRE_COMMIT_TO_REF:-HEAD}
zero=0000000000000000000000000000000000000000
if [[ -z "$from" || "$from" == "$zero" ]] || ! git cat-file -e "$from^{commit}" 2>/dev/null; then
  base=$(git merge-base "$to" origin/main 2>/dev/null || true)
  if [[ -n "$base" ]]; then from=$base; else fallback=1; fi
fi

changed=()
if [[ $fallback -eq 0 ]]; then
  mapfile -t changed < <(git diff --name-only "$from" "$to")
fi
go_changed=() front=0 build=0
for f in "${changed[@]}"; do
  case "$f" in
    frontend/*) front=1 ;;
    *.go|go.mod|go.sum|.golangci.yml|.bloat-baseline|tools/*|internal/*/testdata/*|*/testdata/*) go_changed+=("$f") ;;
    Makefile|scripts/*.sh|scripts/*.mjs|.pre-commit-config.yaml) build=1 ;; # the gate's own scripts; CI checks .github and the rest
  esac
done
if [[ $build -eq 0 && $fallback -eq 0 && ${#go_changed[@]} -eq 0 && $front -eq 0 ]]; then
  echo "prepush: no code changed in $from..$to; the full suite runs in CI"
  exit 0
fi

if [[ ${#go_changed[@]} -gt 0 || $front -eq 1 || $build -eq 1 || $fallback -eq 1 ]]; then
  # Embedded assets and generated bindings must exist before any Go package that embeds them compiles.
  make bindings >/dev/null
fi

if [[ ${#go_changed[@]} -gt 0 || $build -eq 1 || $fallback -eq 1 ]]; then
  make lint
  go build ./...
  if [[ $build -eq 1 || $fallback -eq 1 ]]; then
    echo "prepush: gate scripts changed or no usable base; CI runs the rest"
  fi
fi

if [[ ${#go_changed[@]} -gt 0 ]]; then
  module=$(go list -m)
  mapfile -t dirs < <(printf '%s\n' "${go_changed[@]}" | xargs -n1 dirname | sort -u)
  pkgs=()
  for d in "${dirs[@]}"; do
    [[ -d "$d" ]] || continue
    if [[ "$d" == "internal/composition" ]]; then
      echo "prepush: skipping internal/composition; CI runs it"
      continue
    fi
    if go list "./$d" >/dev/null 2>&1; then
      [[ "$d" == "." ]] && pkgs+=("$module") || pkgs+=("$module/${d#./}")
    fi
  done
  if [[ ${#pkgs[@]} -gt 0 ]]; then
    go test -shuffle=on -timeout 10m "${pkgs[@]}"
  fi
fi

if [[ $front -eq 1 ]]; then
  (cd frontend && pnpm tsc --noEmit && pnpm test && pnpm run build && node scripts/bundle-budget.mjs)
fi

echo "prepush: quick local check passed; CI runs the full suite"
