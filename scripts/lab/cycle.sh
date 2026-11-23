#!/usr/bin/env bash
# cycle.sh (target: lab) — delete + recreate Tom Test / Tom Test Twin with tomlab as commissioner, set up, open both drafts.
set -euo pipefail
D=${TWR_LAB_KIT:?set TWR_LAB_KIT to the Lab kit folder}; cd "$D"
S=(ssh -F "${LAB_SSH_CONF:?set LAB_SSH_CONF to the Lab ssh config}" lab)
SRV="su -s /bin/sh twr -c"
"${S[@]}" pct exec 260 -- rc-service twr-server stop >/dev/null
for L in "native.tomtest-default|Tom Test" "native.tomtest-twin|Tom Test Twin"; do id=${L%%|*}; nm=${L#*|}
  "${S[@]}" "pct exec 260 -- $SRV '/usr/local/bin/twr-server delete-league -db /var/lib/twr/server.db -league $id'" </dev/null 2>&1 | tail -1 || true
  "${S[@]}" "pct exec 260 -- $SRV '/usr/local/bin/twr-server create-league -db /var/lib/twr/server.db -league $id -name \"$nm\" -preset default_32 -season 2026 -commissioner'" < lab-commish.json | head -1
  "${S[@]}" "pct exec 260 -- $SRV '/usr/local/bin/twr-server join-link -db /var/lib/twr/server.db -league $id -take franchise.01 -no-login'" </dev/null | tail -1
done
"${S[@]}" pct exec 260 -- rc-service twr-server start >/dev/null
for i in $(seq 1 20); do python3 -I labcommish.py helper native.tomtest-default >/dev/null 2>&1 && break; sleep 30; done
python3 -I labcommish.py helper native.tomtest-twin
for lg in default twin; do cfg=$D/cfg; [ $lg = twin ] && cfg=$D/cfg-twin; rm -rf "$cfg"; mkdir -p "$cfg"
  XDG_CONFIG_HOME=$cfg ./bin/twr connect --address "$(cat server.url)" --league native.tomtest-$lg < helper-native.tomtest-$lg.txt | tail -1
  W=$D/rehearsal-tomtest-$lg; rm -rf "$W"; WORK=$W DRY_RUN=1 bash ~/fleet/runs/twr-rehearsal-leagues-runbook-1122/setup/setup-commish.sh native.tomtest-$lg $lg >/dev/null
  python3 -c 'import json,sys;p=sys.argv[1];c=json.load(open(p));c["args"]["value"]=10_000_000_000;json.dump(c,open(p,"w"))' $W/cmd/20-draft-clock.json
done
Wd=$D/rehearsal-tomtest-default; Wt=$D/rehearsal-tomtest-twin
XDG_CONFIG_HOME=$D/cfg ./apply.sh $Wd/schedule-generate.json $Wd/schedule-publish.json $Wd/cmd/20-draft-clock.json
XDG_CONFIG_HOME=$D/cfg-twin ./apply.sh $Wt/cmd/05-pools-twin.json $Wt/schedule-generate.json $Wt/schedule-publish.json $Wt/cmd/20-draft-clock.json
XDG_CONFIG_HOME=$D/cfg ./apply.sh $Wd/cmd/30-draft-open.json
XDG_CONFIG_HOME=$D/cfg-twin ./apply.sh $Wt/cmd/30-draft-open.json
echo "both drafts open at $(date -u +%T) UTC"
