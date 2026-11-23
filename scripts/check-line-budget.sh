#!/usr/bin/env bash
# check-line-budget.sh — the line-budget guard (draft: not wired into CI or the Makefile yet).
# Counts lines of TRACKED files (git ls-files + wc -l) by area and compares each area with
# docs/architecture/line-budget-baseline.txt (rows: area<TAB>lines; # lines are comments).
#   - An area ABOVE its baseline fails (exit 1) unless the change carries a written reason:
#       LINE_BUDGET_REASON="why"                     (env var, any non-blank text), or
#       docs/architecture/line-budget-reasons/<branch>.md   (non-blank file; <branch> is the
#                                                     current branch with "/" -> "-", so
#                                                     session/x -> session-x.md)
#   - An area BELOW its baseline passes and prints a hint to lock it in: the ratchet only goes down.
#   - "--write" rewrites the baseline from today's counts (put it in the same commit as the change).
# Areas (the same as the docs/architecture/line-budget-baseline.txt rows):
#   go-prod (total), go-prod/root, go-prod/cmd/<name>, go-prod/internal/<name>, go-prod/<top dir>
#   go-test            *_test.go anywhere (outside testdata/)
#   go-testdata        any file under a testdata/ directory
#   frontend-src       frontend/src/** minus the tests and generated file below
#   frontend-tests     frontend/src/** matching *.test.* / *.spec.* / __tests__/ / fixtures/
#   frontend-generated frontend/src/app/registry/endpoints.gen.ts (counted on its own)
#   docs               docs/** and top-level *.md
#   other              every other tracked text file (scripts, deploy, config, frontend tooling, ...)
# Skipped: binary files (fonts, images, archives) and lock files (go.sum, pnpm-lock.yaml,
# package-lock.json, yarn.lock).
set -euo pipefail
cd "$(dirname "$0")/.."

baseline_file=docs/architecture/line-budget-baseline.txt
branch=$(git branch --show-current 2>/dev/null || true)
reason_file="docs/architecture/line-budget-reasons/${branch//\//-}.md"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# classify: one "path<TAB>area" row per tracked countable file.
classify() {
  git ls-files | awk '
    function area_of(p,   n, parts, b) {
      if (p ~ /(^|\/)(go\.sum|pnpm-lock\.yaml|package-lock\.json|yarn\.lock)$/) return ""
      if (p ~ /\.(woff2?|ttf|otf|eot|png|jpe?g|gif|ico|webp|pdf|zip|gz|tar|bin|db)$/) return ""
      if (p ~ /(^|\/)testdata\//) return "go-testdata"
      if (p ~ /\.go$/) {
        if (p ~ /_test\.go$/) return "go-test"
        n = split(p, parts, "/")
        if (n == 1) return "go-prod/root"
        if (parts[1] == "cmd" || parts[1] == "internal") {
          if (n >= 3) return "go-prod/" parts[1] "/" parts[2]
          return "go-prod/" parts[1] "/(top)"
        }
        return "go-prod/" parts[1]
      }
      if (p ~ /^frontend\/src\//) {
        if (p == "frontend/src/app/registry/endpoints.gen.ts") return "frontend-generated"
        b = p
        sub(/.*\//, "", b)
        if (b ~ /\.(test|spec)\./ || p ~ /(^|\/)(__tests__|fixtures)\//) return "frontend-tests"
        return "frontend-src"
      }
      if (p ~ /^docs\// || p ~ /^[^\/]*\.md$/) return "docs"
      return "other"
    }
    { a = area_of($0); if (a != "") print $0 "\t" a }'
}

# measure: "area<TAB>lines" rows, sorted. "go-prod" is the total of every go-prod/* area.
measure() {
  classify > "$tmp/map.tsv"
  cut -f1 "$tmp/map.tsv" | tr '\n' '\0' | xargs -0 -r wc -l > "$tmp/wc.txt"
  awk -F'\t' '
    FNR == NR { area[$1] = $2; next }
    {
      if (!match($0, /^ *[0-9]+ /)) next
      p = substr($0, RLENGTH + 1)
      if (!(p in area)) next
      n = substr($0, 1, RLENGTH) + 0
      a = area[p]
      sum[a] += n
      if (a ~ /^go-prod\//) sum["go-prod"] += n
    }
    END { for (a in sum) printf "%s\t%d\n", a, sum[a] }
  ' "$tmp/map.tsv" "$tmp/wc.txt" | LC_ALL=C sort
}

measure > "$tmp/counts.tsv"

case ${1:-} in
  --write)
    # The baseline is itself a counted docs file: writing it can change its own line count, so write
    # until the counts it records are the counts it leaves behind.
    for _ in 1 2 3; do
      {
        echo "# line-budget-baseline.txt — per-area line budget (area<TAB>lines). Ratchet: only ever goes down."
        echo "# Written by scripts/check-line-budget.sh --write from git ls-files + wc -l. Do not edit by hand."
        cat "$tmp/counts.tsv"
      } > "$baseline_file"
      measure > "$tmp/after.tsv"
      cmp -s "$tmp/counts.tsv" "$tmp/after.tsv" && break
      mv "$tmp/after.tsv" "$tmp/counts.tsv"
    done
    echo "check-line-budget: wrote $baseline_file ($(wc -l < "$tmp/counts.tsv") areas)"
    exit 0
    ;;
  "") ;;
  *) echo "usage: scripts/check-line-budget.sh [--write]" >&2; exit 2 ;;
esac

[[ -f $baseline_file ]] || { echo "check-line-budget: $baseline_file is missing" >&2; exit 1; }

declare -A base=() now=()
while IFS=$'\t' read -r area lines; do
  base[$area]=$lines
done < <(grep -Ev '^(#|$)' "$baseline_file" || true)
while IFS=$'\t' read -r area lines; do
  now[$area]=$lines
done < "$tmp/counts.tsv"

areas=$(
  for a in "${!base[@]}" "${!now[@]}"; do printf '%s\n' "$a"; done | LC_ALL=C sort -u
)

over=()
under=()
while IFS= read -r a; do
  [[ -n $a ]] || continue
  b=${base[$a]:-0}
  c=${now[$a]:-0}
  if (( c > b )); then
    over+=("$a|$b|$c")
  elif (( c < b )); then
    under+=("$a|$b|$c")
  fi
done <<< "$areas"

if (( ${#over[@]} > 0 )); then
  reason_source=""
  if [[ -n ${LINE_BUDGET_REASON:-} && -n ${LINE_BUDGET_REASON//[[:space:]]/} ]]; then
    reason_source="LINE_BUDGET_REASON"
  elif [[ -f $reason_file && -n $(tr -d '[:space:]' < "$reason_file") ]]; then
    reason_source="$reason_file"
  fi
  if [[ -z $reason_source ]]; then
    echo "check-line-budget: ${#over[@]} area(s) are over the line budget in $baseline_file:" >&2
    for row in "${over[@]}"; do
      IFS='|' read -r a b c <<< "$row"
      printf '  %-28s budget %8s  now %8s  (+%s)\n' "$a" "$b" "$c" "$((c - b))" >&2
    done
    echo "  No written reason found for branch '${branch:-<detached>}'. Add one of:" >&2
    echo "    LINE_BUDGET_REASON=\"why this growth is worth it\" scripts/check-line-budget.sh" >&2
    echo "    $reason_file  (a non-blank file saying the same)" >&2
    echo "  Then raise the baseline in the same commit: scripts/check-line-budget.sh --write" >&2
    exit 1
  fi
  echo "check-line-budget: ${#over[@]} area(s) over budget, accepted by $reason_source:"
  for row in "${over[@]}"; do
    IFS='|' read -r a b c <<< "$row"
    printf '  %-28s budget %8s  now %8s  (+%s)\n' "$a" "$b" "$c" "$((c - b))"
  done
  echo "  Raise the baseline with scripts/check-line-budget.sh --write in the same commit."
fi

if (( ${#under[@]} > 0 )); then
  for row in "${under[@]}"; do
    IFS='|' read -r a b c <<< "$row"
    echo "check-line-budget: $a is under its budget ($c < $b). Lock it in: scripts/check-line-budget.sh --write (same commit)."
  done
fi

if (( ${#over[@]} == 0 && ${#under[@]} == 0 )); then
  echo "check-line-budget: all $(wc -l < "$tmp/counts.tsv") areas at budget"
fi
