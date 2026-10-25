#!/bin/sh
# Run inside the installed Alpine CT, never on the Proxmox host.
set -eu
umask 077
[ "$(id -u)" = 0 ] || { echo 'Run this as root in the CT console.' >&2; exit 1; }
# shellcheck source=/dev/null
. /etc/os-release
[ "$ID" = alpine ] && [ -f /etc/init.d/twr-server ] && [ -d /etc/twr ] || {
  echo 'Run this inside the installed league CT, not on the Proxmox host.' >&2; exit 1;
}
[ "$#" = 1 ] && [ -f "$1" ] || { echo 'Give the path to the new bundle twr-server binary.' >&2; exit 1; }
# The installed service executable is not replaced until backup, migration and verification pass.
install -m 755 "$1" /usr/local/bin/twr-server.next
su -s /bin/sh twr -c '/usr/local/bin/twr-server.next upgrade -db /var/lib/twr/server.db -check'
rc-service twr-tunnel --ifstarted stop
rc-service twr-server --ifstarted stop
# No automatic restart after a failure: a failed restore must never serve partial state.
save="/var/lib/twr/upgrades/$(date -u +%Y%m%dT%H%M%SZ)-$$"
install -d -m 700 -o twr -g twr /var/lib/twr/upgrades "$save"
cp -p /usr/local/bin/twr-server "$save/twr-server.before"
cp -Rp /etc/twr "$save/config"
# Keep the saved secrets private, even if the installed certificate is group-readable.
chmod -R go-rwx "$save"
backup=''
if [ -f /etc/twr/backup.json ]; then backup='-backup-config /etc/twr/backup.json'; fi
if ! su -s /bin/sh twr -c "/usr/local/bin/twr-server.next upgrade -db /var/lib/twr/server.db -save $save/league $backup"; then
  echo "Upgrade stopped. The server and Tunnel remain stopped. Follow UPGRADE.md; the saved set is $save." >&2
  exit 1
fi
mv /usr/local/bin/twr-server.next /usr/local/bin/twr-server
rc-service twr-server start
attempt=0
while [ "$attempt" -lt 30 ]; do
  code=$(curl --silent --cacert /etc/twr/origin.pem --max-time 2 -H 'CF-Connecting-IP: 127.0.0.1' \
    --output /dev/null --write-out '%{http_code}' https://127.0.0.1:8443/not-a-route) || code=000
  # 426 is also an application response: a future security floor may retire this headerless probe.
  case "$code" in
    404|426)
      rc-service twr-tunnel --ifstopped start
      printf '%s\n' "Server upgrade ready. The previous server, settings and league copy are in $save." \
        'Update the desktop apps next. Changes stay paused until a fresh off-site backup verifies.'
      exit 0 ;;
  esac
  attempt=$((attempt + 1))
  sleep 1
done
rc-service twr-server --ifstarted stop
echo "The new server did not become ready. Tunnel remains stopped. Follow UPGRADE.md; keep $save." >&2
exit 1
