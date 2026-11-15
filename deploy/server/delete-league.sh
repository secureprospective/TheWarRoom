#!/bin/sh
# CT-only league removal (RESTORE.md D). The league leaves this server at once; its file is kept
# 30 days in /var/lib/twr/archive, then the server deletes it. Other leagues are untouched.
set -eu
umask 077
[ "$(id -u)" = 0 ] || { echo 'Run this as root in the CT console.' >&2; exit 1; }
printf 'League to delete (its id, for example native.mfl.12345): '; IFS= read -r league
case "$league" in ''|*[!A-Za-z0-9._-]*) echo 'That is not a league id; nothing was changed.' >&2; exit 1 ;; esac
printf 'Type DELETE %s to confirm: ' "$league"; IFS= read -r confirm
[ "$confirm" = "DELETE $league" ] || { echo 'Not confirmed; nothing was changed.' >&2; exit 1; }
rc-service twr-server --ifstarted stop
status=0
su -s /bin/sh twr -c "/usr/local/bin/twr-server delete-league -db /var/lib/twr/server.db -league '$league'" || status=$?
rc-service twr-server start
exit "$status"
