#!/bin/sh
# CT-only league move, old server side (RESTORE.md E). Start writes the league's move file and keeps
# the league readable but unchangeable here; finish archives it here for 30 days once the new server
# has it; cancel lets it change here again.
set -eu
umask 077
[ "$(id -u)" = 0 ] || { echo 'Run this as root in the CT console.' >&2; exit 1; }
printf 'League to move (its id, for example native.mfl.12345): '; IFS= read -r league
case "$league" in ''|*[!A-Za-z0-9._-]*) echo 'That is not a league id; nothing was changed.' >&2; exit 1 ;; esac
printf 'Type 1 to start the move, 2 to finish it (the new server has the league), or 3 to cancel it: '; IFS= read -r step
case "$step" in
  1) install -d -m 700 -o twr -g twr /var/lib/twr/outgoing
     command="export-league -league '$league' -out /var/lib/twr/outgoing/$league-$(date -u +%Y%m%dT%H%M%SZ)" ;;
  2) command="finish-move -league '$league'" ;;
  3) command="cancel-move -league '$league'" ;;
  *) echo 'Nothing was changed.' >&2; exit 1 ;;
esac
rc-service twr-server --ifstarted stop
status=0
su -s /bin/sh twr -c "/usr/local/bin/twr-server $command -db /var/lib/twr/server.db" || status=$?
rc-service twr-server start
exit "$status"
