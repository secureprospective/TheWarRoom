#!/bin/sh
# Plan-only runbook for the CT 260 alpha cleanup review. It never changes server state.
set -eu

usage() {
  cat <<'EOF'
Usage: alpha-clean-plan.sh --dry-run --state-db /absolute/path/to/server.db

Prints the proposed Option A runbook. This command has no apply mode: the supported
server tools cannot delete immutable test-account identities. Run only against an
existing state DB or disposable fixture; it does not read database contents.
EOF
}

fail() {
  printf '%s\n' "$1" >&2
  exit 2
}

dry_run=no
state_db=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --dry-run)
      dry_run=yes
      shift
      ;;
    --state-db)
      [ "$#" -ge 2 ] || fail '--state-db needs a path'
      state_db=$2
      shift 2
      ;;
    --help)
      usage
      exit 0
      ;;
    *)
      fail "unsupported option: $1"
      ;;
  esac
done

[ "$dry_run" = yes ] || fail 'This runbook is plan-only; pass --dry-run.'
[ -n "$state_db" ] || fail '--state-db is required.'
case "$state_db" in
  /*) ;;
  *) fail '--state-db must be an absolute path.' ;;
esac
[ -f "$state_db" ] && [ ! -L "$state_db" ] || fail '--state-db must name an existing regular, non-symlink file.'

cat <<'EOF'
PLAN ONLY — no server command, account, session, seat or database was changed.
The supplied state DB path is only checked; the counts below refer to CT 260 evidence.
Chosen: Option A is the lower-risk route because it retains the imported league,
its 3,399-entry history and the commissioner's existing account identity.

1. Read Commissioner → People → Seats and record the exact 32 occupied franchises.
2. In Control Room, Preview backup now, then confirm backup.now. Require the result
   to say Cloudflare R2 and checked=true; save its timestamp. Then run twr-status and
   require Backups, Football reference backups and Sign-in backups checked in R2, plus
   History OK and the clean summary.
3. For each of the 32 confirmed franchises, preview and execute commish.seat.remove.
   Do not infer seat IDs or remove the commissioner. These are 32 new ledger events;
   with no concurrent league writes, history should advance from 3,399 to 3,431.
4. In the CT console run sh /usr/local/libexec/twr-reset-sign-ins. This revokes all
   device/helper sessions, including the commissioner's; have the commissioner sign
   in again. It preserves accounts, passwords, authenticators and the league.
5. Wait for the automatic sign-in backup (five-minute cadence, one-minute retries)
   to be checked in R2. Then verify People
   shows 32 vacant seats; require the expected history count, all three checked backup
   lines and a clean twr-status summary. Stop if any count or status differs.

STOP: account rows remain, and those users can sign in again. The supported seat
removal is not account deletion; reset-sign-ins is global. There is no supported
command to remove those account rows. No --apply mode is provided.

Rollback: use the saved franchise→principal mapping to re-seat any wrongly released
team with the commissioner's normal commish.seat.replace preview/execute workflow.
For broader recovery, RESTORE.md §C selects a league backup by day; use it only if
that day is the intended checkpoint and no valid writes would be lost. It signs out
league owners. If the day selector cannot identify the pre-change state, stop and ask.

Option B is not selected: the documented first-load import is for a new CT, not a
populated CT; prior job 1 imported 3,367 entries, not the current 3,399, and it would
bootstrap a new commissioner principal. Neither option currently meets every stated
constraint through supported commands. Get Christopher's approval before acting on
this partial Option A plan or changing account-retirement support.
EOF
