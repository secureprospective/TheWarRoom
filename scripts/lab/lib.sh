#!/usr/bin/env bash
# lib.sh (target: lab) — shared helpers for the Lab scripts. Source it; do not run it.
# Everything here talks to CT 260 only. Never the Season server.

# guard_id <league>: practice leagues only. The real native.rehearsal-* leagues are never touched.
guard_id() {
  case ${1:-} in
    native.rehtest-?*) return 0 ;;
    *) echo "REFUSED: '${1:-}' is not a practice league. Only native.rehtest-* ids may be changed here." >&2; return 1 ;;
  esac
}

# lab_init: needs LAB_SSH_CONF (ssh config with the alias `lab`).
lab_init() {
  LABSSH=(ssh -F "${LAB_SSH_CONF:?set LAB_SSH_CONF to the Lab ssh config}" lab)
  SRV="su -s /bin/sh twr -c"
  LAB_STOPPED=0
}

# lab_server stop|start
lab_server() {
  "${LABSSH[@]}" pct exec 260 -- rc-service twr-server "$1" >/dev/null
}

lab_stop() { LAB_STOPPED=1; lab_server stop; }
lab_start() { lab_server start && LAB_STOPPED=0; }

# lab_restart_on_exit: use as `trap lab_restart_on_exit EXIT`. Starts the Lab server again if a script
# stopped it and did not start it. Keeps the script's own exit status.
lab_restart_on_exit() {
  local rc=$?
  if [ "${LAB_STOPPED:-0}" = 1 ]; then
    echo "Lab server was left stopped; starting it again." >&2
    lab_server start || echo "COULD NOT START THE LAB SERVER. Run: ssh lab pct exec 260 -- rc-service twr-server start" >&2
  fi
  exit "$rc"
}

# lab_twrserver <twr-server args...>: runs twr-server as user twr on CT 260 (server must be stopped).
# Stdin is passed through. Arguments must not contain single quotes.
lab_twrserver() {
  "${LABSSH[@]}" "pct exec 260 -- $SRV '/usr/local/bin/twr-server $*'"
}
