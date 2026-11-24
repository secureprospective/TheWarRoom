#!/usr/bin/env bash
# cycle.sh (target: lab) — one command prepares both practice drafts: deletes and recreates
# native.rehtest-default and native.rehtest-twin with the Lab commissioner, sets up the schedule and the
# 13-day pick clock, loads NFL kickoff dates. Drafts stay closed: claim seats, then open-drafts.sh.
# Never touches native.rehearsal-* (guard_id refuses). Needs TWR_LAB_KIT and LAB_SSH_CONF.
set -euo pipefail
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
D=${TWR_LAB_KIT:?set TWR_LAB_KIT to the Lab kit folder}; cd "$D"
# shellcheck source=lib.sh
. "$HERE/lib.sh"
lab_init
DB=/var/lib/twr/server.db
SETUP=~/fleet/runs/twr-rehearsal-leagues-runbook-1122/setup
trap lab_restart_on_exit EXIT

LEAGUES=("native.rehtest-default|Rehearsal Test|default" "native.rehtest-twin|Rehearsal Test Twin|twin")
for L in "${LEAGUES[@]}"; do guard_id "${L%%|*}" || exit 1; done

lab_stop
for L in "${LEAGUES[@]}"; do id=${L%%|*}; rest=${L#*|}; nm=${rest%%|*}
  # An id that is not on the server is fine; any other failure stops the run.
  if ! out=$(lab_twrserver delete-league -db $DB -league "$id" </dev/null 2>&1); then
    case $out in
      *"is not on this server"*) echo "$id: nothing to delete" ;;
      *) echo "STOPPED: could not delete $id: ${out:0:300}" >&2; exit 1 ;;
    esac
  else echo "${out:0:120}"; fi
  lab_twrserver create-league -db $DB -league "$id" -name "\"$nm\"" -preset default_32 -season 2026 -commissioner < lab-commish.json | head -1
  lab_twrserver join-link -db $DB -league "$id" -take franchise.01 -no-login </dev/null | tail -1
done
lab_start

for i in $(seq 1 20); do python3 -I labcommish.py helper native.rehtest-default >/dev/null 2>&1 && break; sleep 30; done
python3 -I labcommish.py helper native.rehtest-twin
for lg in default twin; do cfg=$D/cfg; [ $lg = twin ] && cfg=$D/cfg-twin; rm -rf "$cfg"; mkdir -p "$cfg"
  XDG_CONFIG_HOME=$cfg ./bin/twr connect --address "$(cat server.url)" --league native.rehtest-$lg < helper-native.rehtest-$lg.txt | tail -1
  W=$D/rehearsal-rehtest-$lg; rm -rf "$W"; WORK=$W DRY_RUN=1 bash "$SETUP/setup-commish.sh" native.rehtest-$lg $lg >/dev/null
done
Wd=$D/rehearsal-rehtest-default; Wt=$D/rehearsal-rehtest-twin
# League rule: 13-day pick clock (cmd/20-draft-clock.json as rendered, not overridden).
XDG_CONFIG_HOME=$D/cfg ./apply.sh "$Wd/schedule-generate.json" "$Wd/schedule-publish.json" "$Wd/cmd/20-draft-clock.json"
XDG_CONFIG_HOME=$D/cfg-twin ./apply.sh "$Wt/cmd/05-pools-twin.json" "$Wt/schedule-generate.json" "$Wt/schedule-publish.json" "$Wt/cmd/20-draft-clock.json"
# NFL kickoff dates need the published schedule and a stopped server. A failure here is a warning, not a stop.
lab_stop
for L in "${LEAGUES[@]}"; do id=${L%%|*}
  lab_twrserver refresh-nfl-schedule -db $DB -league "$id" -season 2026 </dev/null 2>&1 | tail -1 \
    || echo "WARNING: refresh-nfl-schedule failed for $id; kickoff times may be missing" >&2
done
lab_start
# The drafts are NOT opened here: the computer drafts every open seat within minutes, so seats are claimed
# first (seat-link.sh), then open-drafts.sh opens both. The 13-day league clock covers a seated owner's turn.
echo "both practice leagues ready, drafts closed, at $(date -u +%T) UTC (target: lab). Next: seat-link.sh, then open-drafts.sh"
