#!/usr/bin/env bash
# check-placement.sh — the placement manifest check. Every root *.go file and every Go package
# under internal/ must have exactly one row in docs/architecture/placement.tsv, every row's path
# must exist, and every row's feature must be a slug in the SYSTEM_MAP.md feature table. Every
# path SYSTEM_MAP.md names in backticks must exist too (a `*` matches as a glob; `app/` means
# frontend/src/app/). Bash and coreutils only.
set -euo pipefail
cd "$(dirname "$0")/.."

manifest=docs/architecture/placement.tsv
map=SYSTEM_MAP.md
fail=0
bad() { echo "check-placement: $*" >&2; fail=1; }

declare -A feature_of=() known=()
while IFS=$'\t' read -r path feature rest; do
  [[ -z $path || $path == \#* || $path == path ]] && continue
  if [[ -z ${feature:-} || -n ${rest:-} ]]; then bad "$manifest: '$path' needs exactly one feature column"; continue; fi
  [[ -n ${feature_of[$path]:-} ]] && bad "$manifest: '$path' is listed twice"
  feature_of[$path]=$feature
  [[ -e $path ]] || bad "$manifest: '$path' does not exist"
done < "$manifest"

# Feature slugs are the backticked first cell of each SYSTEM_MAP.md table row.
while read -r slug; do known[$slug]=1; done < <(sed -n 's/^| `\([a-z][a-z0-9-]*\)`.*/\1/p' "$map")
(( ${#known[@]} > 0 )) || bad "$map: no feature rows found"
for path in "${!feature_of[@]}"; do
  [[ -n ${known[${feature_of[$path]}]:-} ]] || bad "$manifest: '$path' names feature '${feature_of[$path]}', not in $map"
done

for f in *.go; do
  [[ -n ${feature_of[$f]:-} ]] || bad "root file '$f' has no row in $manifest"
done
while read -r pkg; do
  [[ -n ${feature_of[$pkg]:-} ]] || bad "package '$pkg' has no row in $manifest"
done < <(find internal -name '*.go' -not -path '*/testdata/*' -printf '%h\n' | sort -u)

shopt -s nullglob
while read -r tok; do
  [[ $tok == *'<'* ]] && continue
  [[ $tok == */* || $tok =~ \.(go|md|json|csv|tsv|txt|html)$ ]] || continue
  [[ $tok == app/* ]] && tok=frontend/src/$tok
  if [[ $tok == *'*'* ]]; then matches=($tok); else matches=(); [[ -e $tok ]] && matches=("$tok"); fi
  (( ${#matches[@]} > 0 )) || bad "$map names '$tok', which does not exist"
done < <(grep -o '`[^`]*`' "$map" | tr -d '`' | sort -u)

(( fail == 0 )) || { echo "check-placement: fix the problems above; manifest rows are <path><TAB><feature>" >&2; exit 1; }
echo "check-placement: ${#feature_of[@]} paths placed in ${#known[@]} features"
