#!/usr/bin/env bash
# apply.sh <change.json>... : preview, get the commissioner's confirmation when asked (labcommish.py), execute. Lab only.
set -uo pipefail
D=${TWR_LAB_KIT:?set TWR_LAB_KIT to the Lab kit folder}; export XDG_CONFIG_HOME=${XDG_CONFIG_HOME:-$D/cfg}; T=$D/bin/twr
for f in "$@"; do
  echo "=== $(basename "$f")"
  p=$($T preview "$f" 2>&1) || { echo "PREVIEW FAILED: ${p:0:400}"; exit 1; }
  req=$(python3 -c 'import json,sys;print(json.loads(sys.stdin.read()).get("confirmRequest",""))' <<<"$p" 2>/dev/null)
  if [ -n "$req" ]; then
    printf '%s\n' "$req" > "$D/confirm.req"; code=$(python3 -I "$D/labcommish.py" confirm "$D/confirm.req") || { echo "CONFIRM FAILED: $code"; exit 1; }
    out=$($T execute "$f" --confirmation "$code" 2>&1) || { echo "EXECUTE FAILED: ${out:0:400}"; exit 1; }
  else
    out=$($T execute "$f" 2>&1) || { echo "EXECUTE FAILED: ${out:0:400}"; exit 1; }
  fi
  echo "ok${req:+ (confirmed)}"
done
