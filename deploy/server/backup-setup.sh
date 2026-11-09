#!/bin/sh
# CT-only private backup setup. Generate the age identity on your own computer, never here.
set -eu
umask 077
[ "$(id -u)" = 0 ]
[ -d /etc/twr ] && [ -d /var/lib/twr ]
input=/root/twr/backup.json
cleanup() { stty echo 2>/dev/null || true; rm -f "$input"; }
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
if [ ! -f "$input" ]; then
  printf 'R2 S3 endpoint (hostname only): '; IFS= read -r endpoint
  printf 'Backup bucket name: '; IFS= read -r bucket
  printf 'R2 Access Key ID: '; IFS= read -r access
  printf 'R2 Secret Access Key (hidden): '
  stty -echo; IFS= read -r secret; stty echo; printf '\n'
  printf 'Public age recipient (age1..., NEVER AGE-SECRET-KEY): '; IFS= read -r recipient
  # Pass secrets over stdin, not argv or logs.
  printf '%s\n' "$endpoint" "$bucket" "$access" "$secret" "$recipient" |
    jq -Rn '[inputs] | {endpoint:.[0],bucket:.[1],access_key:.[2],secret_key:.[3],recipient:.[4]}' > "$input"
fi
/usr/local/bin/twr-server backup-check -config "$input"
rc-service twr-server --ifstarted stop
install -m 600 -o twr -g twr "$input" /etc/twr/backup.json
rm "$input"
rc-service twr-server start
printf 'Backup settings saved privately. Changes stay paused until an encrypted R2 upload passes read-back verification.\n'
