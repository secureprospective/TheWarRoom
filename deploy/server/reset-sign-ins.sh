#!/bin/sh
# CT-only incident tool (RESTORE.md): signs every device and helper out. Passwords,
# authenticators, printed recovery codes and the league itself are unchanged.
set -eu
umask 077
[ "$(id -u)" = 0 ] || { echo 'Run this as root in the CT console.' >&2; exit 1; }
printf 'Reason, kept in the sign-in history: '; IFS= read -r reason
rc-service twr-server --ifstarted stop
status=0
printf '%s\n' "$reason" | su -s /bin/sh twr -c '/usr/local/bin/twr-server reset-sign-ins -db /var/lib/twr/server.db' || status=$?
rc-service twr-server start
exit "$status"
