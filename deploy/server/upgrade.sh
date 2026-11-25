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
first_load=${TWR_FIRST_LOAD:-0}
case "$first_load" in
  0 | 1) ;;
  *) echo 'TWR_FIRST_LOAD must be 0 or 1.' >&2; exit 1 ;;
esac
if [ "$first_load" = 1 ]; then
  [ -d /var/lib/twr ] && [ ! -L /var/lib/twr ] || {
    echo 'First-load mode requires the installed state directory.' >&2; exit 1;
  }
  [ -z "$(find /var/lib/twr -mindepth 1 -maxdepth 1 -print -quit)" ] || {
    echo 'First-load mode requires an empty /var/lib/twr; nothing was changed.' >&2; exit 1;
  }
  if rc-service twr-server status >/dev/null 2>&1; then
    echo 'First-load mode requires twr-server to be stopped; nothing was changed.' >&2; exit 1;
  fi
  runlevel_services=$(rc-update show default 2>&1) || {
    echo 'Could not confirm twr-server startup status; nothing was changed.' >&2; exit 1;
  }
  case "$runlevel_services" in
    *twr-server*) echo 'First-load mode requires twr-server to be unscheduled; nothing was changed.' >&2; exit 1 ;;
  esac
fi
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
# Extract only the executable here, never archive-selected paths; the console tools follow from the
# same frozen bytes only once the upgrade has succeeded.
tar -xzOf "$snapshot" ./twr-server > /var/lib/twr-release/upgrade.lock/twr-server
install -m 755 /var/lib/twr-release/upgrade.lock/twr-server /usr/local/bin/twr-server.next
if [ "$first_load" = 0 ]; then
  su -s /bin/sh twr -c '/usr/local/libexec/twr-sandbox /usr/local/bin/twr-server.next upgrade -db /var/lib/twr/server.db -check'
fi
rc-service twr-tunnel --ifstarted stop
if [ "$first_load" = 0 ]; then
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
fi
mv /usr/local/bin/twr-server.next /usr/local/bin/twr-server
# The console tools and service files come from the same verified bundle (ruling 2026-11-14), so an
# upgraded server has every tool a fresh install has. Fixed names only, never archive-selected paths;
# nothing from the bundle runs here. Each lands beside its target and is renamed over it, so a tool
# that is running, this one included, keeps reading its old copy.
while read -r source target; do
  tar -xzOf "$snapshot" "./$source" > "$target.next" 2>/dev/null && [ -s "$target.next" ] || {
    rm -f "$target.next"; continue;
  }
  chown root:root "$target.next"
  chmod 755 "$target.next"
  mv "$target.next" "$target"
done <<'TOOLS'
backup-setup.sh /usr/local/libexec/twr-backup-setup
upgrade.sh /usr/local/libexec/twr-upgrade
restore.sh /usr/local/libexec/twr-restore
reset-sign-ins.sh /usr/local/libexec/twr-reset-sign-ins
restore-league.sh /usr/local/libexec/twr-restore-league
delete-league.sh /usr/local/libexec/twr-delete-league
move-league.sh /usr/local/libexec/twr-move-league
receive-league.sh /usr/local/libexec/twr-receive-league
status.sh /usr/local/libexec/twr-status
twr-sandbox /usr/local/libexec/twr-sandbox
twr-server.initd /etc/init.d/twr-server
cloudflared.initd /etc/init.d/twr-tunnel
TOOLS
if [ "$first_load" = 1 ]; then
  rc-service twr-tunnel --ifstopped start
  cat <<'NEXT'
First-load server update ready. twr-server remains stopped and unscheduled; /var/lib/twr is still empty.
Copy the desktop history.db and owner-only commissioner.json to /root/seed, then run in this CT:
  twr-server league-load -db /var/lib/twr/server.db -mirror /root/seed/history.db \
    -mfl-league <MFL league number> -football-pin /etc/twr/football.sha256 < /root/seed/commissioner.json
league-load sets ownership on every path it creates; no chown command is needed.
  rc-update add twr-server default
  rc-service twr-server start
Then complete the R2 backup setup in BACKUPS.md and check twr-status.
NEXT
  exit 0
fi
rc-service twr-server start
attempt=0
while [ "$attempt" -lt 30 ]; do
  code=$(curl --silent --cacert /etc/twr/origin.pem --max-time 2 -H 'CF-Connecting-IP: 127.0.0.1' \
    --output /dev/null --write-out '%{http_code}' https://127.0.0.1:8443/not-a-route) || code=000
  # 426 is also an application response: a future security floor may retire this headerless probe.
  case "$code" in
    404|426)
      rc-service twr-tunnel --ifstopped start
      # Names are restricted below; ls | sort provides the specified UTC name ordering.
      # shellcheck disable=SC2012
      old_sets=$(ls /var/lib/twr/upgrades | sort)
      set_count=$(printf '%s\n' "$old_sets" | grep -Ec '^[0-9]{8}T[0-9]{6}Z-[0-9]+$' || true)
      if [ "$set_count" -gt 5 ]; then
        remove_count=$((set_count - 5))
        printf '%s\n' "$old_sets" | grep -E '^[0-9]{8}T[0-9]{6}Z-[0-9]+$' | while IFS= read -r name; do
          [ "$remove_count" -gt 0 ] || break
          [ "$name" = "${save##*/}" ] && continue
          rm -rf -- "/var/lib/twr/upgrades/$name"
          remove_count=$((remove_count - 1))
        done
      fi
      printf '%s\n' "Server upgrade ready. The previous server, settings and league copy are in $save." \
        'Only the newest 5 saved sets are kept; older ones were removed.' \
        'Changes stay paused until a fresh off-site backup verifies.'
      exit 0 ;;
  esac
  attempt=$((attempt + 1))
  sleep 1
done
rc-service twr-server --ifstarted stop
echo "The new server did not become ready. Tunnel remains stopped. Follow UPGRADE.md; keep $save." >&2
exit 1
