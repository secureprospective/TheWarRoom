#!/usr/bin/env bash
# bloat.sh — the bloat ratchet. Measures four signs of volume without value in non-test Go
# and fails if any rises above the committed baseline (.bloat-baseline). `--update` rewrites
# the baseline, but only downward: improvements are locked in, regressions are refused.
#
#   comment_pct   comment lines as a % of all lines
#   provenance    comment lines carrying review/agent history (belongs in commits and docs)
#   tiny_files    files with under 40 lines of code (a split made for size, not responsibility)
#   dupl          near-duplicate blocks found by golangci-lint's dupl (copies that should be data)
set -euo pipefail

baseline=.bloat-baseline
provenance_re='(GLM|DeepSeek|Gemini|Ornith|agy)|[Rr]eview (m|lead |finding |round )?[0-9#]|Round-[0-9]|Ship-[0-9]|Friction #[0-9]'

mapfile -t files < <(find . -name '*.go' ! -name '*_test.go' \
  -not -path './vendor/*' -not -path './tools/*' -not -path '*/node_modules/*' -not -path './frontend/*' | sort)

read -r total comments provenance tiny < <(awk -v re="$provenance_re" '
  FNR == 1 { if (NR > 1 && code < 40) tiny++; code = 0 }
  { total++ }
  /^[[:space:]]*\/\// { comments++; if ($0 ~ re) prov++; next }
  /[^[:space:]]/ { code++ }
  END { if (code < 40) tiny++; printf "%d %d %d %d\n", total, comments, prov + 0, tiny + 0 }
' "${files[@]}")

comment_pct=$(( 100 * comments / total ))
dupl=$(golangci-lint run --enable-only dupl --max-issues-per-linter=0 --max-same-issues=0 ./... 2>/dev/null | grep -c '(dupl)$' || true)

declare -A now=([comment_pct]=$comment_pct [provenance]=$provenance [tiny_files]=$tiny [dupl]=$dupl)
declare -A base=()
if [[ -f $baseline ]]; then
  while IFS='=' read -r k v; do [[ -n $k ]] && base[$k]=$v; done < "$baseline"
fi

fail=0
printf '%-12s %8s %8s\n' metric now baseline
for k in comment_pct provenance tiny_files dupl; do
  b=${base[$k]:-none}
  mark=''
  if [[ $b != none && ${now[$k]} -gt $b ]]; then mark='  ROSE'; fail=1; fi
  printf '%-12s %8s %8s%s\n' "$k" "${now[$k]}" "$b" "$mark"
done

if [[ ${1:-} == --update ]]; then
  if (( fail )); then echo "bloat: refusing to raise the baseline"; exit 1; fi
  for k in comment_pct provenance tiny_files dupl; do echo "$k=${now[$k]}"; done > "$baseline"
  echo "bloat: baseline updated"
  exit 0
fi
if (( fail )); then echo "bloat: a measure rose above the baseline — cut it back, don't raise the baseline"; exit 1; fi
echo "bloat: at or under baseline"
