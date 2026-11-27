#!/usr/bin/env bash
# check-stub-plan-paths.sh — every source-file path in docs/ui/v3/V3-STUB-PLAN.md
# must resolve to a tracked file (repo root, frontend/src/app/, or bare .go anywhere).
set -euo pipefail
cd "$(dirname "$0")/.."

plan=${STUB_PLAN:-docs/ui/v3/V3-STUB-PLAN.md}
if [[ ! -f $plan ]]; then
  echo "check-stub-plan-paths: plan file not found: $plan" >&2
  exit 1
fi

tracked=$(git ls-files)
checked=0
fail=0
missing=()

# Extract unique path-shaped tokens ending in .ts, .tsx, .go, or .mjs (backticked or plain),
# skipping any non-space run that contains glob characters (*, ?, [).
paths=$(awk '{ for (i = 1; i <= NF; i++) if ($i !~ /[*?\[]/) print $i }' "$plan" \
  | grep -oE '[A-Za-z0-9_./*-]+\.(ts|tsx|go|mjs)\b' \
  | sort -u || true)

while IFS= read -r p; do
  [[ -z $p ]] && continue
  # Skip globs.
  case "$p" in
    *\**|*\?*|*\[*) continue ;;
  esac
  # Skip generated-file patterns.
  case "$p" in
    *.gen.ts|*.gen.tsx|*.gen.go|*.gen.mjs) continue ;;
  esac
  # Skip paths outside the repo.
  case "$p" in
    '~/'*|/*|http://*|https://*) continue ;;
  esac

  checked=$((checked + 1))

  if grep -qxF "$p" <<<"$tracked"; then
    continue
  fi
  if grep -qxF "frontend/src/app/$p" <<<"$tracked"; then
    continue
  fi
  # Bare filenames (no slash): .go files may live anywhere in the repo; .ts/.tsx under frontend/src/app/.
  if [[ $p != */* ]]; then
    if [[ $p == *.go ]] && grep -qE "/${p//./\\.}$" <<<"$tracked"; then
      continue
    fi
    if [[ $p == *.ts || $p == *.tsx ]] && grep -qE "^frontend/src/app/.*/${p//./\\.}$" <<<"$tracked"; then
      continue
    fi
  fi

  missing+=("$p")
  fail=1
done <<<"$paths"

if (( fail != 0 )); then
  echo "check-stub-plan-paths: missing tracked source file(s) in $plan:" >&2
  for m in "${missing[@]}"; do
    echo "  $m" >&2
  done
  exit 1
fi

echo "check-stub-plan-paths: $checked path(s) in $plan resolved to tracked files"
exit 0
