#!/usr/bin/env bash
# oline-close.sh <league> <default|twin> (target: lab) — close the O-Line auction of a practice league.
# Run after the draft and after every O-Line bid is decided (the server refuses otherwise).
set -euo pipefail
[ $# -eq 2 ] || { echo "usage: $0 native.rehtest-<name> <default|twin>" >&2; exit 2; }
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib.sh
. "$HERE/lib.sh"
guard_id "$1" || exit 1
D=${TWR_LAB_KIT:?set TWR_LAB_KIT to the Lab kit folder}
cfg=$D/cfg; [ "$2" = twin ] && cfg=$D/cfg-twin
f=$D/oline-close-$2.json
sed -e "s/@LEAGUE@/$1/g" -e "s/@SLUG@/${1#native.}/g" "$HERE/commands/50-oline-close.json" > "$f"
XDG_CONFIG_HOME=$cfg "$HERE/apply.sh" "$f"
