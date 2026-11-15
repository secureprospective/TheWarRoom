#!/usr/bin/env bash
# check-source-branches.sh — where a league came from is not a kind of league (Christopher, 10-31).
# 1. Every Go line that reads a league's origin must be on the allowlist: sites that only record where data came
#    from, or choose initial data at creation. Behaviour is driven by settings, never by origin.
# 2. The hard-coded MFL league ID and the screens' MFL/app forks may only go down (ratchet).
set -euo pipefail
cd "$(dirname "$0")/.."

allow=docs/architecture/source-branch-allowlist.txt
base=docs/architecture/source-branch-baseline.txt
fail=0

hits=$(grep -rnE 'LeagueSource\(|Source(MFLImport|Native)\b|importedSource\(' --include=*.go internal \
  | grep -v '_test\.go:' | while IFS=: read -r file _; do
    printf '%s\n' "$file"; done | sort -u)
while read -r file; do
  [[ -z $file ]] && continue
  if ! grep -qxF "$file" <(grep -v '^#' "$allow"); then
    echo "check-source-branches: $file reads where a league came from and is not on $allow." >&2
    echo "  Drive the behaviour by a league setting instead; a league from MFL is a normal league." >&2
    fail=1
  fi
done <<<"$hits"

count() { grep -v '^#' "$base" | grep "^$1=" | cut -d= -f2; }
ids=$(grep -rn 'ingestion\.LeagueID' --include=*.go internal cmd | grep -v '_test\.go:' | wc -l)
forks=$(grep -rn 'native ?' frontend/src --include=*.ts --include=*.tsx | grep -v '\.test\.' | wc -l)
for pair in "league_id:$ids" "native_forks:$forks"; do
  name=${pair%%:*}; now=${pair#*:}; was=$(count "$name")
  if (( now > was )); then
    echo "check-source-branches: $name is $now, over the baseline of $was in $base. It only goes down." >&2
    fail=1
  elif (( now < was )); then
    echo "check-source-branches: $name is $now, under $was. Lock it in: write $name=$now to $base."
  fi
done
(( fail == 0 )) && echo "check-source-branches: origin reads all allowlisted; league_id=$ids native_forks=$forks"
exit $fail
