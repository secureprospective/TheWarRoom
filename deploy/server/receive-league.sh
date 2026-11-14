#!/bin/sh
# CT-only league move, new server side (RESTORE.md E). Takes the league from the old server's move
# file and prints one claim link per person. The move code is typed here and passed in memory only.
set -eu
umask 077
[ "$(id -u)" = 0 ] || { echo 'Run this as root in the CT console.' >&2; exit 1; }
incoming=/var/lib/twr/incoming
stage=/var/lib/twr/receive-league
cleanup() { stty echo 2>/dev/null || true; rm -rf "$stage" "$incoming"; rc-service twr-server --ifstopped start || true; }
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
printf 'Move file (its full path in this container, for example /root/native.mfl.12345.twrleague): '; IFS= read -r file
[ -f "$file" ] || { echo 'There is no file at that path; nothing was changed.' >&2; exit 1; }
printf 'Move code, the line beginning AGE-SECRET-KEY-1 (hidden): '
stty -echo 2>/dev/null || true; IFS= read -r code; stty echo 2>/dev/null || true; printf '\n'
rm -rf "$stage" "$incoming"
install -d -m 700 -o twr -g twr "$incoming"
install -m 600 -o twr -g twr "$file" "$incoming/league.twrleague"
rc-service twr-server --ifstarted stop
status=0
result=$(printf '%s\n' "$code" | su -s /bin/sh twr -c "/usr/local/bin/twr-server import-league -db /var/lib/twr/server.db -in $incoming/league.twrleague -stage $stage" 2>&1) || status=$?
unset code
# Claim links print as paths; this server's own web address goes in front of each.
host=$(jq -r '.ingress[0].hostname // empty' /etc/cloudflared/config.yml 2>/dev/null || true)
if [ -n "$host" ]; then result=$(printf '%s\n' "$result" | sed "s|/join#|https://$host/join#|"); fi
printf '%s\n' "$result"
exit "$status"
