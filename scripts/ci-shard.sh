#!/usr/bin/env bash
# Split composition tests round-robin; greedily balance the other packages by CI duration.
# Usage: scripts/ci-shard.sh SHARD TOTAL (zero-based); DRY=1 prints commands only.
set -euo pipefail
shard=${1:?shard required} total=${2:?total required}
[[ $shard =~ ^[0-9]+$ && $total =~ ^[1-9][0-9]*$ ]] && (( shard < total )) || {
  echo "invalid shard/total" >&2; exit 1;
}
weights="$(dirname "$0")/ci-shard-weights.tsv"
module=$(go list .)
root=$module/internal/composition
test_list=$(go test -list '^Test' "$root")
mapfile -t tests < <(printf '%s\n' "$test_list" | grep '^Test' | LC_ALL=C sort)
pick=()
for i in "${!tests[@]}"; do
  if (( i % total == shard )); then pick+=("${tests[$i]}"); fi
done

# Missing packages get the median recorded weight. Ties in weight and load use
# package name and shard number, respectively, so every runner builds the same map.
median=$(cut -f2 "$weights" | sort -n | awk '{v[NR]=$1} END {if (NR % 2) print v[(NR+1)/2]; else print (v[NR/2]+v[NR/2+1])/2}')
packages=$(go list ./...)
assignments=$(printf '%s\n' "$packages" | awk -v root="$root" -v default="$median" '
  NR == FNR {weight[$1]=$2; next}
  $0 != root {print $0, ($0 in weight ? weight[$0] : default)}
' "$weights" - | LC_ALL=C sort -k2,2nr -k1,1 | awk -v total="$total" '
  {best=0; for (i=1; i<total; i++) if (load[i]<load[best]) best=i;
   print $1, $2, best; load[best]+=$2}
')
mapfile -t pkgs < <(printf '%s\n' "$assignments" | awk -v shard="$shard" '$3 == shard {print $1}')
composition_weight=$(awk -v root="$root" -v default="$median" '$1 == root {found=1; print $2} END {if (!found) print default}' "$weights")
predicted=$(printf '%s\n' "$assignments" | awk -v shard="$shard" -v base="$composition_weight" -v total="$total" 'BEGIN {sum=base/total} $3 == shard {sum+=$2} END {printf "%.3f", sum}')
echo "shard $shard/$total: ${#pick[@]} composition tests, ${#pkgs[@]} packages, predicted ${predicted}s"
if (( ${#pick[@]} > 0 )); then
  ${DRY:+echo} go test -race -timeout 60m -run "^($(IFS="|"; echo "${pick[*]}"))\$" "$root"
fi
if (( ${#pkgs[@]} > 0 )); then
  ${DRY:+echo} go test -race -timeout 60m "${pkgs[@]}"
fi
