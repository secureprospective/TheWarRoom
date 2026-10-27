#!/usr/bin/env bash
# prepush.sh — the local pre-push gate. It checks only what the pushed commits change; GitHub's verify
# workflow runs the full suite on every push, and the merge waits for it.
#   docs only        → nothing to run
#   Go changed       → lint, then shuffled tests on the changed packages and their importers (-race runs in CI)
#   frontend changed → type check, frontend tests, build and bundle budget
# With no usable range (a new branch, a deleted ref) it falls back to the full `make verify`.
set -euo pipefail

from=${PRE_COMMIT_FROM_REF:-}
to=${PRE_COMMIT_TO_REF:-HEAD}
zero=0000000000000000000000000000000000000000
if [[ -z "$from" || "$from" == "$zero" ]] || ! git cat-file -e "$from^{commit}" 2>/dev/null; then
  base=$(git merge-base "$to" origin/main 2>/dev/null || true)
  [[ -n "$base" ]] || exec make verify
  from=$base
fi

mapfile -t changed < <(git diff --name-only "$from" "$to")
go_changed=() front=0 build=0
for f in "${changed[@]}"; do
  case "$f" in
    frontend/*) front=1 ;;
    *.go|go.mod|go.sum|.golangci.yml|.bloat-baseline|tools/*|internal/*/testdata/*|*/testdata/*) go_changed+=("$f") ;;
    Makefile|scripts/*.sh|scripts/*.mjs|.pre-commit-config.yaml) build=1 ;; # the gate's own scripts; CI checks .github and the rest
  esac
done
[[ $build -eq 0 ]] || exec make verify
if [[ ${#go_changed[@]} -eq 0 && $front -eq 0 ]]; then
  echo "prepush: no code changed in $from..$to; the full suite runs in CI"
  exit 0
fi

if [[ ${#go_changed[@]} -gt 0 || $front -eq 1 ]]; then
  # Embedded assets and generated bindings must exist before any Go package that embeds them compiles.
  make bindings >/dev/null
fi

if [[ ${#go_changed[@]} -gt 0 ]]; then
  make lint
  module=$(go list -m)
  mapfile -t dirs < <(printf '%s\n' "${go_changed[@]}" | xargs -n1 dirname | sort -u)
  pkgs=()
  for d in "${dirs[@]}"; do
    [[ -d "$d" ]] || continue
    [[ "$d" == "." ]] && pkgs+=("$module") || pkgs+=("$module/${d#./}")
  done
  # Packages that import a changed package can break too; go list -deps over ./... finds them.
  mapfile -t affected < <(go list -f '{{.ImportPath}}{{range .Deps}} {{.}}{{end}}' ./... |
    awk -v want="$(printf '%s ' "${pkgs[@]}")" 'BEGIN{n=split(want,w," "); for(i=1;i<=n;i++) set[w[i]]=1}
      { for(i=1;i<=NF;i++) if ($i in set) { print $1; break } }')
  if [[ ${#affected[@]} -gt 0 ]]; then
    go test -shuffle=on -timeout 30m "${affected[@]}"
  fi
fi

if [[ $front -eq 1 ]]; then
  (cd frontend && pnpm tsc --noEmit && pnpm test && pnpm run build && node scripts/bundle-budget.mjs)
fi
