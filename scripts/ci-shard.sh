#!/usr/bin/env bash
# Runs one CI shard of the Go race suite: every TOTAL-th top-level test of the root package (the long
# pole, 26 min alone on a 4-core runner) plus every TOTAL-th other package, heaviest first.
# Usage: scripts/ci-shard.sh SHARD TOTAL
set -euo pipefail
shard=$1 total=$2
root=$(go list .)
mapfile -t tests < <(go test -list "^Test" . | grep "^Test" | sort)
pick=()
for i in "${!tests[@]}"; do
  if (( i % total == shard )); then pick+=("${tests[$i]}"); fi
done
# The slow packages lead the list so round-robin spreads them across shards.
heavy=(internal/store/history internal/labcompile internal/facts internal/commissioner internal/lab)
mapfile -t rest < <(go list ./... | grep -vxF "$root" | grep -vxF -f <(printf "$root/%s\n" "${heavy[@]}") | sort)
pkgs=()
all=($(printf "$root/%s\n" "${heavy[@]}") "${rest[@]}")
for i in "${!all[@]}"; do
  if (( i % total == shard )); then pkgs+=("${all[$i]}"); fi
done
echo "shard $shard/$total: ${#pick[@]} root tests, ${#pkgs[@]} packages"
if (( ${#pick[@]} > 0 )); then
  ${DRY:+echo} go test -race -timeout 60m -run "^($(IFS="|"; echo "${pick[*]}"))\$" .
fi
if (( ${#pkgs[@]} > 0 )); then
  ${DRY:+echo} go test -race -timeout 60m "${pkgs[@]}"
fi
