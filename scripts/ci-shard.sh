#!/usr/bin/env bash
# Greedily balance packages across shards by recorded CI duration.
# Usage: scripts/ci-shard.sh SHARD TOTAL (zero-based); DRY=1 prints commands only.
set -euo pipefail
shard=${1:?shard required} total=${2:?total required}
[[ $shard =~ ^[0-9]+$ && $total =~ ^[1-9][0-9]*$ ]] && (( shard < total )) || {
  echo "invalid shard/total" >&2; exit 1;
}
weights="$(dirname "$0")/ci-shard-weights.tsv"

# Missing packages get the median recorded weight. Ties in weight and load use
# package name and shard number, respectively, so every runner builds the same map.
median=$(cut -f2 "$weights" | sort -n | awk '{v[NR]=$1} END {if (NR % 2) print v[(NR+1)/2]; else print (v[NR/2]+v[NR/2+1])/2}')
packages=$(go list ./...)
assignments=$(printf '%s\n' "$packages" | awk -v fallback="$median" '
  NR == FNR {weight[$1]=$2; next}
  {print $0, ($0 in weight ? weight[$0] : fallback)}
' "$weights" - | LC_ALL=C sort -k2,2nr -k1,1 | awk -v total="$total" '
  {best=0; for (i=1; i<total; i++) if (load[i]<load[best]) best=i;
   print $1, $2, best; load[best]+=$2}
')
mapfile -t pkgs < <(printf '%s\n' "$assignments" | awk -v shard="$shard" '$3 == shard {print $1}')
predicted=$(printf '%s\n' "$assignments" | awk -v shard="$shard" '$3 == shard {sum+=$2} END {printf "%.3f", sum}')
echo "shard $shard/$total: ${#pkgs[@]} packages, predicted ${predicted}s"
if (( ${#pkgs[@]} > 0 )); then
  ${DRY:+echo} go test -race -timeout 60m "${pkgs[@]}"
fi
