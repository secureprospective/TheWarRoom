#!/usr/bin/env bash
# seat-link.sh <league> (target: lab) — prints ONE new join link for a practice league, to the terminal only.
# Stops the Lab server, rotates the link, starts the server again (even on failure). The old link stops working.
# Never redirect this script's output to a file or log. Needs TWR_LAB_KIT and LAB_SSH_CONF.
set -euo pipefail
[ $# -eq 1 ] || { echo "usage: $0 native.rehtest-<name>" >&2; exit 2; }
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib.sh
. "$HERE/lib.sh"
guard_id "$1" || exit 1
D=${TWR_LAB_KIT:?set TWR_LAB_KIT to the Lab kit folder}
[ -t 1 ] || { echo "REFUSED: output is not a terminal; the link is never written to a file." >&2; exit 1; }
lab_init
trap lab_restart_on_exit EXIT
lab_stop
lab_twrserver join-link -db /var/lib/twr/server.db -league "$1" -rotate < "$D/lab-commish.json"
lab_start
echo "(Lab server started again. Open the link on your device and pick a franchise.)"
