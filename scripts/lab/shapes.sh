#!/usr/bin/env bash
# shapes.sh (target: lab) — GATE part 2: the 10/12/16-team one-season leagues with computer owners.
#   setup   recreate native.rehtest-10/-12/-16 from the one_season_N presets, schedule weeks 13-18 (schedule.py),
#           10 s pick clock, open the startup drafts. The commissioner holds franchise.01 only for setup; its
#           turns time out to the computer, so every team is drafted by the computer within minutes.
#   release (after the drafts finish) revoke the setup helpers and free franchise.01: every team is computer-run.
# Never touches native.rehearsal-* or the Sunday practice leagues' scripts. Needs TWR_LAB_KIT and LAB_SSH_CONF.
set -euo pipefail
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
D=${TWR_LAB_KIT:?set TWR_LAB_KIT to the Lab kit folder}; cd "$D"
# shellcheck source=lib.sh
. "$HERE/lib.sh"
lab_init
DB=/var/lib/twr/server.db
trap lab_restart_on_exit EXIT
cmd=${1:-}; [ "$#" -eq 1 ] || { echo 'usage: shapes.sh setup|release' >&2; exit 2; }
for n in 10 12 16; do guard_id "native.rehtest-$n" || exit 1; done
case $cmd in
setup)
  lab_stop
  for n in 10 12 16; do id=native.rehtest-$n
    if ! out=$(lab_twrserver delete-league -db "$DB" -league "$id" </dev/null 2>&1); then case $out in *"is not on this server"*) ;; *) echo "STOPPED: could not delete $id: ${out:0:300}" >&2; exit 1;; esac; fi
    lab_twrserver create-league -db "$DB" -league "$id" -name "\"Shape $n\"" -preset "one_season_$n" -teams "$n" -season 2026 -commissioner < lab-commish.json | head -1
    lab_twrserver join-link -db "$DB" -league "$id" -take franchise.01 -no-login </dev/null >/dev/null
done
  lab_start
  for n in 10 12 16; do id=native.rehtest-$n
    ok=0; for i in $(seq 1 20); do if python3 -I labcommish.py helper "$id" >/dev/null 2>&1; then ok=1; break; fi; sleep 30; done
    [ "$ok" = 1 ] || { echo "helper failed for $id" >&2; exit 1; }
    cfg=$D/cfg-shape$n; rm -rf "$cfg"; mkdir -p "$cfg"
    XDG_CONFIG_HOME=$cfg ./bin/twr connect --address "$(cat server.url)" --league "$id" < "helper-$id.txt" >/dev/null
    W=$D/shape-$n; rm -rf "$W"; mkdir -p "$W"
    python3 -I "$HERE/schedule.py" --league "$id" --teams "$n" --out "$W"
    python3 - "$id" "$W" <<'PY'
import json,sys
league,w=sys.argv[1:]
def cmd(name,args,reason):
 return {'league':league,'command':name,'args':args,'reason':reason,'request':name.replace('.','-')+'-'+league.split('.',1)[1],'effective':'now','visibility':'league'}
for fn,doc in [('draft-clock.json',cmd('commish.rules.set',{'field':'draft.clock','value':10000000000},'Shape league draft clock')),('draft-open.json',cmd('draft.startup.open',{'draft':{'season':2026,'startup':True}},'Open shape league startup draft'))]:
 with open(w+'/'+fn,'w') as f: json.dump(doc,f,indent=1); f.write('\n')
PY
    XDG_CONFIG_HOME=$cfg ./apply.sh "$W/schedule-generate.json" "$W/schedule-publish.json" "$W/draft-clock.json" "$W/draft-open.json"
  done
  lab_stop
  for n in 10 12 16; do lab_twrserver refresh-nfl-schedule -db "$DB" -league "native.rehtest-$n" -season 2026 </dev/null 2>&1 | tail -1 || echo "WARNING: refresh-nfl-schedule failed for $n" >&2; done
  lab_start ;;
release)
  for n in 10 12 16; do python3 -I labcommish.py revoke "native.rehtest-$n"; done
  lab_stop
  for n in 10 12 16; do lab_twrserver join-link -db "$DB" -league "native.rehtest-$n" -release franchise.01 < lab-commish.json | tail -1; done
  lab_start ;;
*) echo 'usage: shapes.sh setup|release' >&2; exit 2;;
esac
