#!/usr/bin/env bash
# Saturday-night swap on the Season server (Alpine, OpenRC). Read deploy/season/README.md first.
# Usage: season-swap.sh <version-dir-name> [--christopher-ok] [--dry-run]
#   <version-dir-name>  a folder under $APP_ROOT/versions/ on the server
#   --christopher-ok    Christopher has approved this exact run; lets it run outside the window
#   --dry-run           print every step and command; send nothing to the server
# Exit: 0 done (or dry run), 1 a step failed, 2 bad usage, 3 refused by the guard (no server contact).
set -euo pipefail

SEASON_HOST="root@192.168.1.162"
APP_ROOT="/opt/twr"
DATA_DIR="/var/lib/twr"
SERVICE="twr"
PORT="5199"
BACKUP_DIR="/var/backups/twr"
SSH_OPTS=(-o BatchMode=yes -o ConnectTimeout=10)

usage() {
  echo "usage: $0 <version-dir-name> [--christopher-ok] [--dry-run]" >&2
  exit 2
}

VERSION=""
CHRISTOPHER_OK=0
DRY_RUN=0
for arg in "$@"; do
  case "$arg" in
    --christopher-ok) CHRISTOPHER_OK=1 ;;
    --dry-run) DRY_RUN=1 ;;
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
  echo "  $0 $VERSION --christopher-ok"
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

STAMP=$(date -u +%Y%m%dT%H%M%SZ)
VERSION_DIR="$APP_ROOT/versions/$VERSION"
CURRENT="$APP_ROOT/current"
PREVIOUS="$APP_ROOT/versions/<previous-version-folder>"
BACKUP_FILE="$BACKUP_DIR/twr-data-$STAMP.tgz"
HEALTH_URL="http://127.0.0.1:$PORT/app/"
HEALTH_CMD="i=0; until curl -fsS $HEALTH_URL >/dev/null; do"
HEALTH_CMD+=" i=\$((i+1)); [ \$i -ge 15 ] && exit 1; sleep 2; done"
FLIPPED=0

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
  echo "  ssh $SEASON_HOST 'ln -sfn $PREVIOUS $CURRENT && rc-service $SERVICE restart'"
}

die() {
  echo "FAILED: $1" >&2
  if [ "$FLIPPED" -eq 1 ]; then
    echo "The link already points at $VERSION. Roll back with:" >&2
    print_rollback >&2
  fi
  exit 1
}

step "1/6 Back up the league data ($DATA_DIR) before anything changes"
remote "mkdir -p $BACKUP_DIR && tar -C $(dirname "$DATA_DIR") -czf $BACKUP_FILE $(basename "$DATA_DIR")" \
  || die "backup failed; nothing was changed"

step "2/6 Check the new version folder is on the server"
remote "test -d $VERSION_DIR" || die "$VERSION_DIR is not on the server; nothing was changed"

step "3/6 Read the current version, for rollback"
if [ "$DRY_RUN" -eq 0 ]; then
  echo "   ssh $SEASON_HOST 'readlink $CURRENT'"
  # shellcheck disable=SC2029  # on purpose: $CURRENT is a fixed path from the top of this file
  PREVIOUS=$(ssh "${SSH_OPTS[@]}" "$SEASON_HOST" "readlink $CURRENT") \
    || die "cannot read $CURRENT; nothing was changed"
  [ -n "$PREVIOUS" ] || die "$CURRENT is not a link; nothing was changed"
  echo "   previous version: $PREVIOUS"
else
  echo "   ssh $SEASON_HOST 'readlink $CURRENT'   (dry run: the real value is read here)"
fi

step "4/6 Flip the current link to the new version"
remote "ln -sfn $VERSION_DIR $CURRENT" || die "could not flip the link"
FLIPPED=1

step "5/6 Restart the service"
remote "rc-service $SERVICE restart" || die "restart failed"

step "6/6 Health check on the server (port $PORT, up to about 30 seconds)"
remote "$HEALTH_CMD" \
  || die "health check failed"

echo ""
if [ "$DRY_RUN" -eq 1 ]; then
  echo "Dry run: nothing was sent. A real run does the steps above."
else
  echo "Swap done: $VERSION is live and answered 200 on port $PORT."
fi
echo "Rollback command (flip back to the previous version):"
print_rollback
