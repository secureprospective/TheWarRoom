#!/bin/sh
# CT-only one-league restore (RESTORE.md C). Puts one league back from its own backup. Only that
# league's owners are signed out; every other league is untouched. The league's file before the
# restore is kept 30 days. The restore key is typed here and passed in memory, never written to disk.
set -eu
umask 077
[ "$(id -u)" = 0 ] || { echo 'Run this as root in the CT console.' >&2; exit 1; }
[ -s /etc/twr/backup.json ] || { echo 'Off-site backups are not set up on this server, so there is nothing to restore from.' >&2; exit 1; }
stage=/var/lib/twr/restore-league
# The working folder holds a decrypted copy of the league; it never outlives this run.
cleanup() { stty echo 2>/dev/null || true; rm -rf "$stage"; rc-service twr-server --ifstopped start || true; }
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
rm -rf "$stage"
printf 'League to restore (its id, for example native.mfl.12345): '; IFS= read -r league
case "$league" in ''|*[!A-Za-z0-9._-]*) echo 'That is not a league id; nothing was changed.' >&2; exit 1 ;; esac
printf 'Restore key, the line beginning AGE-SECRET-KEY-1 (hidden): '
stty -echo 2>/dev/null || true; IFS= read -r key; stty echo 2>/dev/null || true; printf '\n'
printf 'Restore from an older day? Type YYYY-MM-DD, or press Enter for the newest backup: '; IFS= read -r before
rc-service twr-server --ifstarted stop
# The bucket settings come from this server's own backup settings; secrets travel over stdin only.
{ cat /etc/twr/backup.json; printf '%s\n' "$key" "$before" | jq -R .; } |
  jq -s '{endpoint: .[0].endpoint, bucket: .[0].bucket, access_key: .[0].access_key, secret_key: .[0].secret_key, restore_key: .[1], before: .[2]}' |
  su -s /bin/sh twr -c "/usr/local/bin/twr-server restore-league -db /var/lib/twr/server.db -league '$league' -stage $stage"
unset key
