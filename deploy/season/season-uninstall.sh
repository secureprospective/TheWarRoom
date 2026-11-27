#!/bin/sh
# Undo script for deploy/season/season-install.sh. Pure POSIX sh; run from the Beelink.
# Removes ONLY what season-install.sh added. Preserves /var/lib/twr and /var/backups/twr unless
# --purge-data is passed explicitly. Never touches /etc/sysctl.d/10-twr.conf or openssh-server.
# Usage:
#   season-uninstall.sh --target <user@host> [--purge-data] [--dry-run] [--christopher-ok]
# Exit: 0 done (or dry run), 1 a step failed, 2 bad usage, 3 refused by guard.
set -eu

APP_ROOT=${SEASON_APP_ROOT:-/opt/twr}
DATA_DIR=${SEASON_DATA_DIR:-/var/lib/twr}
BACKUP_DIR=${SEASON_BACKUP_DIR:-/var/backups/twr}
ETC_DIR=${SEASON_ETC_DIR:-/etc/twr}
INIT_DIR=${SEASON_INIT_DIR:-/etc/init.d}
LOGROTATE_DIR=${SEASON_LOGROTATE_DIR:-/etc/logrotate.d}
PERIODIC_DIR=${SEASON_PERIODIC_DIR:-/etc/periodic/hourly}
STAGE_DIR=${SEASON_STAGE_DIR:-/root/.twr-install-stage}
SERVICE="twr"

usage() {
  echo "usage: $0 --target <user@host> [--purge-data] [--dry-run] [--christopher-ok]" >&2
  exit 2
}

TARGET=""
PURGE_DATA=0
DRY_RUN=0
CHRISTOPHER_OK=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --target)
      [ "$#" -ge 2 ] || usage
      TARGET=$2
      shift 2
      ;;
    --purge-data)
      PURGE_DATA=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --christopher-ok)
      CHRISTOPHER_OK=1
      shift
      ;;
    -h | --help)
      usage
      ;;
    *)
      echo "unknown option: $1" >&2
      usage
      ;;
  esac
done

[ -n "$TARGET" ] || { echo "missing required --target <user@host> (never defaulted)" >&2; usage; }

valid_target() {
  case "$1" in
    '' | -* | *@*@* | *[!A-Za-z0-9._@-]*) return 1 ;;
    ?*@?*) ;;
    *) return 1 ;;
  esac
  t_user=${1%@*}
  t_host=${1#*@}
  case "$t_user" in '' | -* | *[!A-Za-z0-9._-]*) return 1 ;; esac
  case "$t_host" in '' | -* | *[!A-Za-z0-9._-]*) return 1 ;; esac
  return 0
}

valid_abs_path() {
  case "$1" in
    '' | / | [!\/]* | *..* | *//* | *[!A-Za-z0-9/._-]*) return 1 ;;
  esac
  return 0
}

valid_target "$TARGET" || { echo "invalid --target (expected user@host): $TARGET" >&2; exit 2; }
for p in "$APP_ROOT" "$DATA_DIR" "$BACKUP_DIR" "$ETC_DIR" "$INIT_DIR" "$LOGROTATE_DIR" "$PERIODIC_DIR" "$STAGE_DIR"; do
  valid_abs_path "$p" || { echo "paths must be plain absolute paths, not /: $p" >&2; exit 2; }
done

read_clock() {
  if [ -n "${SEASON_NOW:-}" ]; then
    case "$SEASON_NOW" in
      [0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]) ;;
      *) echo "SEASON_NOW must look like 2026-11-28T19:00 (Chicago time)" >&2; exit 2 ;;
    esac
    now_date=${SEASON_NOW%%T*}
    now_time=${SEASON_NOW#*T}
    now_h=${now_time%%:*}
    now_m=${now_time#*:}
    now_h=${now_h#0}; NOW_HOUR=${now_h:-0}
    now_m=${now_m#0}; now_min=${now_m:-0}
    if [ "$NOW_HOUR" -gt 23 ] || [ "$now_min" -gt 59 ]; then
      echo "SEASON_NOW time out of range: $SEASON_NOW" >&2
      exit 2
    fi
    NOW_DOW=$(date -d "$now_date" +%u) || { echo "invalid SEASON_NOW date: $now_date" >&2; exit 2; }
    NOW_TEXT="$SEASON_NOW (SEASON_NOW test override, Chicago time)"
  else
    NOW_TEXT=$(TZ=America/Chicago date '+%a %F %H:%M %Z')
    case "$NOW_TEXT" in
      *CST* | *CDT*) ;;
      *) echo "cannot read America/Chicago time (is tzdata installed?)" >&2; exit 1 ;;
    esac
    NOW_DOW=$(TZ=America/Chicago date '+%u')
    now_h=$(TZ=America/Chicago date '+%H')
    now_h=${now_h#0}; NOW_HOUR=${now_h:-0}
  fi
}

print_approval() {
  cmd="$0 --target $TARGET"
  [ "$PURGE_DATA" -eq 0 ] || cmd="$cmd --purge-data"
  echo "Exact command for Christopher to approve (run only inside Saturday 18:00-23:59 Chicago after he says yes):"
  echo "  $cmd --christopher-ok"
}

read_clock
IN_WINDOW=0
if [ "$NOW_DOW" = 6 ] && [ "$NOW_HOUR" -ge 18 ] && [ "$NOW_HOUR" -le 23 ]; then
  IN_WINDOW=1
fi

if [ "$DRY_RUN" -eq 0 ]; then
  if [ "$CHRISTOPHER_OK" -eq 0 ] || [ "$IN_WINDOW" -eq 0 ]; then
    print_approval
    echo "Season server uninstall requires BOTH --christopher-ok and the Saturday 18:00-23:59" >&2
    echo "America/Chicago window; it is now $NOW_TEXT. Nothing was sent to any host." >&2
    exit 3
  fi
else
  if [ "$CHRISTOPHER_OK" -eq 0 ] && [ "$IN_WINDOW" -eq 0 ]; then
    print_approval
    echo "Season server uninstall dry runs outside Saturday 18:00-23:59 America/Chicago require" >&2
    echo "--christopher-ok; it is now $NOW_TEXT. Nothing was sent to any host." >&2
    exit 3
  fi
fi

print_approval

step() {
  echo ""
  echo "== $*"
}

remote() {
  printf '   ssh -o BatchMode=yes -o ConnectTimeout=10 %s %s\n' "$TARGET" "'$1'"
  if [ "$DRY_RUN" -eq 0 ]; then
    # shellcheck disable=SC2029  # intentional: payload is expanded locally and sent as one remote command
    ssh -o BatchMode=yes -o ConnectTimeout=10 "$TARGET" "$1"
  fi
}

step "1/4 Stop and unschedule OpenRC service $SERVICE"
STOP_CMD="set -eu
[ \"\$(id -u)\" = 0 ] || { echo 'Refused: must run as root on target.' >&2; exit 1; }
if [ -x '$INIT_DIR/$SERVICE' ]; then
  rc-service '$SERVICE' --ifstarted stop
  rc-update del '$SERVICE' default 2>/dev/null || true
fi"
remote "$STOP_CMD" || exit 1

step "2/4 Remove installed program, config, service script, and logrotate files"
REMOVE_CMD="set -eu
rm -f '$INIT_DIR/$SERVICE' '$LOGROTATE_DIR/twr' '$PERIODIC_DIR/twr-logrotate'
rm -rf '$APP_ROOT' '$ETC_DIR' '$STAGE_DIR'"
remote "$REMOVE_CMD" || exit 1

if [ "$PURGE_DATA" -eq 1 ]; then
  step "3/4 Purge state, backups, and twr service user (--purge-data)"
  DATA_CMD="set -eu
rm -rf '$DATA_DIR' '$BACKUP_DIR'
if id twr >/dev/null 2>&1; then
  deluser twr 2>/dev/null || true
fi
delgroup twr 2>/dev/null || true"
else
  step "3/4 Preserve state and backups (pass --purge-data to remove them)"
  DATA_CMD="set -eu
echo 'Preserved $DATA_DIR and $BACKUP_DIR (re-run with --purge-data to remove).'"
fi
remote "$DATA_CMD" || exit 1

step "4/4 Verify installed paths are removed (system files and /etc/sysctl.d/10-twr.conf untouched)"
CHECK_CMD="set -eu
for gone in '$APP_ROOT' '$ETC_DIR' '$INIT_DIR/$SERVICE' '$LOGROTATE_DIR/twr' '$PERIODIC_DIR/twr-logrotate' '$STAGE_DIR'; do
  [ ! -e \"\$gone\" ] && [ ! -L \"\$gone\" ] || { echo \"Still present: \$gone\" >&2; exit 1; }
done
echo 'Undo verification OK.'"
remote "$CHECK_CMD" || exit 1
