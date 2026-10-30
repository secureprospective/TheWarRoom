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
[ "$#" = 2 ] && [ -f "$1" ] && [ -d "$2" ] || {
  echo 'Give the signed server bundle and its metadata folder.' >&2; exit 1;
}
[ -f /etc/twr-release/root.json ] || {
  echo 'No pinned release root; reinstall from a verified bundle.' >&2; exit 1;
}
# Only the installed executable supplies the release floor and verifies delivery bytes.
release=$(/usr/local/bin/twr-server version | sed -n 's/^League server release \([0-9][0-9]*\):.*/\1/p')
case "$release" in ''|*[!0-9]*) echo 'Repair the installed server from a verified bundle.' >&2; exit 1;; esac
# The verifier and snapshot share one private cache; serialize concurrent console upgrades.
[ "$(stat -c %a:%u:%g /var/lib/twr-release)" = 700:0:0 ] || {
  echo 'Repair the release cache ownership and private permissions.' >&2; exit 1;
}
mkdir /var/lib/twr-release/upgrade.lock 2>/dev/null || {
  echo 'Wait for the other upgrade to finish before trying again.' >&2; exit 1;
}
trap '
  rm -f /var/lib/twr-release/upgrade.lock/bundle.tar.gz /var/lib/twr-release/upgrade.lock/twr-server
  rmdir /var/lib/twr-release/upgrade.lock
' EXIT
trap 'exit 1' HUP INT TERM
verified=$(/usr/local/bin/twr-server verify-release -installed-release "$release" \
  -bundle "$1" -metadata "$2") || exit 1
# Freeze the authenticated bytes, then recheck the digest before unpacking anything.
# This closes replacement of the delivery file between verification and extraction.
snapshot=/var/lib/twr-release/upgrade.lock/bundle.tar.gz
cp "$1" "$snapshot"
digest=$(printf '%s\n' "$verified" | sed -n 's/^verified release [0-9][0-9]* sha256 \([0-9a-f]*\)$/\1/p')
[ -n "$digest" ] && [ "$digest" = "$(sha256sum "$snapshot" | cut -d ' ' -f 1)" ] || {
  echo 'The bundle changed; ask your release provider for a fresh signed bundle and metadata.' >&2; exit 1;
}
# Extract only the executable, never archive-selected paths or deployment scripts.
tar -xzOf "$snapshot" ./twr-server > /var/lib/twr-release/upgrade.lock/twr-server
install -m 755 /var/lib/twr-release/upgrade.lock/twr-server /usr/local/bin/twr-server.next
su -s /bin/sh twr -c '/usr/local/libexec/twr-sandbox /usr/local/bin/twr-server.next upgrade -db /var/lib/twr/server.db -check'
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
if ! su -s /bin/sh twr -c "/usr/local/libexec/twr-sandbox /usr/local/bin/twr-server.next upgrade -db /var/lib/twr/server.db -save $save/league $backup"; then
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
