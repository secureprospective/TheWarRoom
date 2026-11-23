#!/usr/bin/env bash
# prints latest pick per league every 60 s; exits when both show round 43 or nothing moved for 5 min
D=${TWR_LAB_KIT:?set TWR_LAB_KIT to the Lab kit folder}; declare -A last same
while :; do
  done_n=0
  for lg in default twin; do cfg=$D/cfg; [ $lg = twin ] && cfg=$D/cfg-twin
    top=$(XDG_CONFIG_HOME=$cfg $D/bin/twr ledger 2>/dev/null | python3 -c 'import json,sys
d=json.load(sys.stdin); r=d if isinstance(d,list) else next(v for v in d.values() if isinstance(v,list))
p=[x["what"] for x in r if "startup draft" in x["what"]]; print(p[0][:90] if p else "no picks")' 2>/dev/null)
    [ "$top" = "${last[$lg]:-}" ] && same[$lg]=$(( ${same[$lg]:-0} + 1 )) || same[$lg]=0; last[$lg]=$top
    echo "$(TZ=America/Chicago date +%H:%M) $lg: $top"
    case "$top" in *"round 43"*) done_n=$((done_n+1));; esac
    [ "${same[$lg]}" -ge 5 ] && { echo "STALLED $lg"; exit 1; }
  done
  [ $done_n = 2 ] && { echo "BOTH DONE"; exit 0; }
  sleep 60
done
