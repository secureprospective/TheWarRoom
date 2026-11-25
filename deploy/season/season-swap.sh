#!/usr/bin/env bash
# Saturday-night swap on the Season server (Alpine, OpenRC). Read deploy/season/README.md first.
# Usage: season-swap.sh <version-dir-name> [--rollback] [--christopher-ok] [--dry-run]
#   <version-dir-name>  a folder under $APP_ROOT/versions/ on the server
#   --christopher-ok    Christopher has approved this exact run; lets it run outside the window
#   --dry-run           print every step and command; send nothing to the server
# Exit: 0 done (or dry run), 1 a step failed, 2 bad usage, 3 refused by the guard (no server contact).
set -euo pipefail

SEASON_HOST="root@192.168.1.162"
APP_ROOT="${SEASON_APP_ROOT:-/opt/twr}"
DATA_DIR="${SEASON_DATA_DIR:-/var/lib/twr}"
SERVICE="twr"
PORT="5199"
BACKUP_DIR="${SEASON_BACKUP_DIR:-/var/backups/twr}"
SSH_OPTS=(-o BatchMode=yes -o ConnectTimeout=10)

usage() {
  echo "usage: $0 <version-dir-name> [--rollback] [--christopher-ok] [--dry-run]" >&2
  exit 2
}

VERSION=""
CHRISTOPHER_OK=0
DRY_RUN=0
ROLLBACK=0
for arg in "$@"; do
  case "$arg" in
    --christopher-ok) CHRISTOPHER_OK=1 ;;
    --dry-run) DRY_RUN=1 ;;
    --rollback) ROLLBACK=1 ;;
    -*) echo "unknown option: $arg" >&2; usage ;;
    *)
      [ -z "$VERSION" ] || { echo "give one version name only" >&2; usage; }
      VERSION="$arg" ;;
  esac
done
[ -n "$VERSION" ] || usage

# Sets NOW_DOW (1=Mon .. 7=Sun), NOW_HOUR (0-23) and NOW_TEXT, all in America/Chicago time.
read_clock() {
  if [ -n "${SEASON_NOW:-}" ]; then
    if ! [[ "$SEASON_NOW" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}$ ]]; then
      echo "SEASON_NOW must look like 2026-11-28T19:00 (Chicago time)" >&2
      exit 2
    fi
    NOW_TEXT="$SEASON_NOW (SEASON_NOW test override, Chicago time)"
    NOW_DOW=$(date -d "${SEASON_NOW%%T*}" +%u)
    NOW_HOUR=${SEASON_NOW:11:2}
  else
    NOW_TEXT=$(TZ=America/Chicago date '+%a %F %H:%M %Z')
    case "$NOW_TEXT" in
      *CST*|*CDT*) ;;
      *) echo "cannot read America/Chicago time (is tzdata installed?)" >&2; exit 1 ;;
    esac
    NOW_DOW=$(TZ=America/Chicago date '+%u')
    NOW_HOUR=$(TZ=America/Chicago date '+%H')
  fi
  NOW_HOUR=$((10#$NOW_HOUR))
}

print_approval() {
  echo "Exact command for Christopher to approve (run it only after he says yes):"
  echo "  $0 $VERSION $([ "$ROLLBACK" -eq 0 ] || echo --rollback) --christopher-ok"
}

# The guard runs before anything else touches the server.
read_clock
IN_WINDOW=0
if [ "$NOW_DOW" = 6 ] && [ "$NOW_HOUR" -ge 18 ] && [ "$NOW_HOUR" -le 23 ]; then IN_WINDOW=1; fi
# A real swap always needs his OK; a dry run inside the window does not. Outside the window only his OK
# (an emergency he approved) lets even a dry run through.
if [ "$CHRISTOPHER_OK" -eq 0 ] && { [ "$DRY_RUN" -eq 0 ] || [ "$IN_WINDOW" -eq 0 ]; }; then
  print_approval
  echo "Season server swaps need Christopher's OK each time (--christopher-ok); the window is" >&2
  echo "Saturday 18:00-23:59 Chicago time, and it is now $NOW_TEXT. Nothing was sent to the server." >&2
  exit 3
fi
print_approval

if ! [[ "$VERSION" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]] || [[ "$VERSION" == *..* ]]; then
  echo "version name must be a plain folder name (letters, digits, . _ -): $VERSION" >&2
  exit 2
fi

for path in "$APP_ROOT" "$DATA_DIR" "$BACKUP_DIR"; do
  [[ "$path" =~ ^/[A-Za-z0-9/_-]+$ ]] && [ "$path" != / ] || {
    echo "Paths must be absolute plain paths, not the filesystem root." >&2; exit 2;
  }
done
STAMP=$(date -u +%Y%m%dT%H%M%SZ)-$$
[ "$ROLLBACK" -eq 0 ] || STAMP="$VERSION"

step() { echo ""; echo "== $*"; }

# remote <command>: print it as ssh; run it on the Season server unless --dry-run.
# shellcheck disable=SC2029  # on purpose: values are built here and sent as one command
remote() {
  echo "   ssh $SEASON_HOST '$1'"
  if [ "$DRY_RUN" -eq 0 ]; then
    ssh "${SSH_OPTS[@]}" "$SEASON_HOST" "$1"
  fi
}

print_rollback() {
  echo "  $0 $STAMP --rollback --christopher-ok"
}

die() {
  echo "FAILED: $1" >&2
  echo "Recovery command (inspect server state before approving):" >&2
  print_rollback >&2
  exit 1
}

# The same on-host transaction is exercised offline through a local ssh stub.
PAYLOAD="APP_ROOT='$APP_ROOT'
DATA_DIR='$DATA_DIR'
BACKUP_DIR='$BACKUP_DIR'
VERSION='$VERSION'
STAMP='$STAMP'
ROLLBACK='$ROLLBACK'
$(cat "$(dirname "$0")/season-data.sh")"
step "1/6 Validate recovery inputs"
step "2/6 Stop service before touching data"
step "3/6 Back up or stage the recorded backup"
step "4/6 Preserve data and change version link"
step "5/6 Start service"
step "6/6 Health check"
remote "$PAYLOAD" || die "transaction failed; inspect the printed step; service may be stopped"
echo "Rollback command (restore pre-swap data and version):"
print_rollback
