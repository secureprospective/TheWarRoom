#!/bin/sh
# CT-only spare-host restore (RESTORE.md). The restore key is typed here and passed to
# twr-server in memory; it is never written to disk. Nothing on the Proxmox host changes.
set -eu
umask 077
[ "$(id -u)" = 0 ] || { echo 'Run this as root in the CT console.' >&2; exit 1; }
[ -d /etc/twr ] && [ -d /var/lib/twr ]
stage=/var/lib/twr/restore
# The working folder holds a decrypted copy of the league; it never outlives this run.
# A failed restore leaves the fresh install running as it was.
cleanup() { stty echo 2>/dev/null || true; rm -rf "$stage"; rc-service twr-server --ifstopped start || true; }
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
rm -rf "$stage"
printf 'R2 S3 endpoint (hostname only): '; IFS= read -r endpoint
printf 'Backup bucket name: '; IFS= read -r bucket
printf 'NEW R2 Access Key ID: '; IFS= read -r access
printf 'NEW R2 Secret Access Key (hidden): '
stty -echo 2>/dev/null || true; IFS= read -r secret; stty echo 2>/dev/null || true; printf '\n'
printf 'Restore key, the line beginning AGE-SECRET-KEY-1 (hidden): '
stty -echo 2>/dev/null || true; IFS= read -r key; stty echo 2>/dev/null || true; printf '\n'
rc-service twr-server --ifstarted stop
# Secrets travel over stdin, never argv or files. The service user does the work.
printf '%s\n' "$endpoint" "$bucket" "$access" "$secret" "$key" |
  jq -Rn '[inputs] | {endpoint:.[0],bucket:.[1],access_key:.[2],secret_key:.[3],restore_key:.[4]}' |
  su -s /bin/sh twr -c "/usr/local/bin/twr-server restore -db /var/lib/twr/server.db -stage $stage"
unset secret key
install -m 600 -o twr -g twr "$stage/ready/authenticator.key" /etc/twr/authenticator.key
install -m 600 -o twr -g twr "$stage/ready/backup.json" /etc/twr/backup.json
mv "$stage/ready/server.db" /var/lib/twr/server.db
rm -f /var/lib/twr/server.db-wal /var/lib/twr/server.db-shm
rc-service twr-server start
printf '%s\n' 'Restore installed. Changes and sign-ins stay paused until the first backup of the restored' \
  'league uploads and verifies, usually within five minutes. Then every owner signs in again.'
