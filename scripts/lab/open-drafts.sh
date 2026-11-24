#!/usr/bin/env bash
# open-drafts.sh (target: lab) — opens both practice drafts. Run only after every human has claimed a seat
# (seat-link.sh): the computer drafts each open seat within minutes. Needs TWR_LAB_KIT and a finished cycle.sh.
set -euo pipefail
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib.sh
. "$HERE/lib.sh"
D=${TWR_LAB_KIT:?set TWR_LAB_KIT to the Lab kit folder}; cd "$D"
for lg in default twin; do guard_id "native.rehtest-$lg" || exit 1
  cfg=$D/cfg; [ $lg = twin ] && cfg=$D/cfg-twin
  XDG_CONFIG_HOME=$cfg ./apply.sh "$D/rehearsal-rehtest-$lg/cmd/30-draft-open.json"
done
echo "both drafts open at $(date -u +%T) UTC (target: lab)"
