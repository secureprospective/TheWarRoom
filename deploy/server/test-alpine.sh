#!/usr/bin/env bash
# Disposable Alpine execution, never a real league or Cloudflare account.
set -euo pipefail
cd "$(dirname "$0")/../.."
[[ -f .scratch/server.tar.gz ]] || { echo 'Run bundle.sh .scratch/server.tar.gz first.' >&2; exit 1; }
fixtures=$(mktemp -d "$PWD/.scratch/release-test.XXXXXX")
scripts/release/.venv/bin/python deploy/server/test-release-fixtures.py .scratch/server.tar.gz "$fixtures/delivery"
fixtures=${fixtures#"$PWD/"}
image=alpine:3.23@sha256:ff7f0c727dd796aa36787bfcd9a64bf35a6ce72a73451b3359fde849b6703d63
name="twr-cs34-test-$$"
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
# Privileges apply only to this disposable container's namespaces. No host mounts
# except the read-only candidate; no host network/PID namespace or real secrets.
docker run -d --init --name "$name" --privileged -v "$PWD:/src:ro" "$image" sleep 600 >/dev/null
docker exec -e "DELIVERY=/src/$fixtures/delivery" "$name" sh -ec '
  mkdir -p /root/twr /run/openrc
  touch /run/openrc/softlevel
  tar xzf /src/.scratch/server.tar.gz -C /root/twr
  cp "$DELIVERY/root.json" /root/twr/root.json
  apk add --no-cache bash shellcheck
  shellcheck /src/deploy/server/proxmox.sh /src/deploy/server/install.sh /src/deploy/server/backup-setup.sh /src/deploy/server/restore.sh /src/deploy/server/upgrade.sh /src/deploy/server/reset-sign-ins.sh /src/deploy/server/status.sh /src/deploy/server/cloudflare.sh /src/deploy/server/test-alpine.sh /src/deploy/server/bundle.sh /src/deploy/server/twr-sandbox
  mkdir /tmp/host-tools
  cat > /tmp/host-tools/stub <<'\''STUB'\''
#!/bin/sh
printf "%s\n" "$0" >> /tmp/host-calls
exit 1
STUB
  chmod +x /tmp/host-tools/stub
  tools="pct pveam pvesm pve-firewall chronyc realpath grep mktemp rm awk sort tail ip cat sleep"
  for tool in $tools; do ln -s stub "/tmp/host-tools/$tool"; done
  # Every missing prerequisite is explained before any host command can run.
  for tool in $tools; do
    rm "/tmp/host-tools/$tool"
    : > /tmp/host-calls
    if PATH=/tmp/host-tools /bin/bash /src/deploy/server/proxmox.sh /src/.scratch/server.tar.gz > /tmp/host-result 2>&1; then exit 1; fi
    grep -F "Required Proxmox host tool is missing: $tool." /tmp/host-result
    [ ! -s /tmp/host-calls ]
    ln -s stub "/tmp/host-tools/$tool"
  done
  # No jq exists on this simulated host; preflight reaches the clock check.
  : > /tmp/host-calls
  if PATH=/tmp/host-tools /bin/bash /src/deploy/server/proxmox.sh /src/.scratch/server.tar.gz > /tmp/host-result 2>&1; then exit 1; fi
  grep -F "The Proxmox host clock is not synchronized" /tmp/host-result
  [ "$(cat /tmp/host-calls)" = /tmp/host-tools/chronyc ]
  # A prior rotation link must not prevent installation or repeated scheduling.
  mkdir -p /etc/periodic/hourly
  ln -s /etc/periodic/daily/logrotate /etc/periodic/hourly/twr-logrotate
  sh /root/twr/install.sh
  sed -n "/^ln -s/p" /root/twr/install.sh > /tmp/schedule-rotation
  sh /tmp/schedule-rotation
  sh /tmp/schedule-rotation
  # Exercise the deployed JSON writer with the allowed domain/token characters.
  mkdir /tmp/json-work
  sed -n "/^printf.*zone.*hostname.*api_token/,/input.json/p" /root/twr/proxmox.sh > /tmp/write-input
  zone=example.org hostname=league-1.example.org token=Abc_123-xyz replace=true work=/tmp/json-work sh /tmp/write-input
  jq -e '\''. == {zone:"example.org",hostname:"league-1.example.org",api_token:"Abc_123-xyz",replace:true}'\'' /tmp/json-work/input.json
  rc-service twr-server status
  # Real CT-only settings validation/install, but stub service restart to avoid external R2.
  mkdir /tmp/backup-tools
  printf "#!/bin/sh\nprintf \"%%s\\n\" \"\$*\" >> /tmp/backup-service-calls\n" > /tmp/backup-tools/rc-service
  chmod 755 /tmp/backup-tools/rc-service
  printf "{\"endpoint\":\"%s.r2.cloudflarestorage.com\",\"bucket\":\"league-backups\",\"access_key\":\"%s\",\"secret_key\":\"%s\",\"recipient\":\"age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p\"}\n" \
    "$(printf %032d 0)" "$(printf %032d 0)" "$(printf %064d 0)" > /root/twr/backup.json
  chmod 600 /root/twr/backup.json
  PATH=/tmp/backup-tools:$PATH sh /usr/local/libexec/twr-backup-setup > /tmp/backup-setup-result
  test "$(stat -c %a /etc/twr/backup.json)" = 600
  test "$(stat -c %U:%G /etc/twr/backup.json)" = twr:twr
  test ! -e /root/twr/backup.json
  grep -F "twr-server --ifstarted stop" /tmp/backup-service-calls
  grep -F "twr-server start" /tmp/backup-service-calls
  /usr/local/bin/twr-server backup-check -config /etc/twr/backup.json
  rm /etc/twr/backup.json
  # Spare-host restore refuses a bad key before touching state, then restarts the fresh install.
  before=$(sha256sum /etc/twr/authenticator.key /var/lib/twr/server.db)
  : > /tmp/backup-service-calls
  if printf "endpoint\nbucket\naccess\nsecret\nnot-a-key\n\n" | PATH=/tmp/backup-tools:$PATH sh /usr/local/libexec/twr-restore > /tmp/restore-result 2>&1; then exit 1; fi
  grep -F "restore key is not right" /tmp/restore-result
  grep -F "twr-server --ifstopped start" /tmp/backup-service-calls
  [ "$before" = "$(sha256sum /etc/twr/authenticator.key /var/lib/twr/server.db)" ]
  test ! -e /var/lib/twr/restore
  test ! -e /etc/twr/backup.json
  # The incident reset runs as the service user and keeps its files owned by twr.
  printf "alpine drill\n" | PATH=/tmp/backup-tools:$PATH sh /usr/local/libexec/twr-reset-sign-ins | grep -F "on 0 accounts"
  [ -z "$(find /var/lib/twr ! -user twr)" ]
  # A second install refuses without replacing authoritative state or keys.
  before=$(sha256sum /etc/twr/authenticator.key /var/lib/twr/server.db /etc/twr-release/root.json)
  if sh /root/twr/install.sh; then exit 1; fi
  [ "$before" = "$(sha256sum /etc/twr/authenticator.key /var/lib/twr/server.db /etc/twr-release/root.json)" ]
  [ "$(stat -c %a /var/lib/twr)" = 700 ]
  [ "$(stat -c %U:%G /var/lib/twr)" = twr:twr ]
  [ "$(readlink /etc/periodic/hourly/twr-logrotate)" = /etc/periodic/daily/logrotate ]
  for file in server.db server.db-wal server.db-shm server.log; do
    [ -f "/var/lib/twr/$file" ]
    [ "$(stat -c %a "/var/lib/twr/$file")" = 600 ]
  done
  [ "$(stat -c %a /etc/twr/authenticator.key)" = 600 ]
  printf "rotation probe\n" >> /var/lib/twr/server.log
  logrotate -f -s /tmp/rotation.state /etc/logrotate.d/twr
  [ "$(stat -c %a /var/lib/twr/server.log.1)" = 600 ]
  [ "$(stat -c %a /var/lib/twr/server.log)" = 600 ]
  rc-service crond status
  # Reuse the exact deployed namespace/mount options, replacing only its payload.
  cat > /tmp/probe <<'\''PROBE'\''
#!/bin/sh
exec /usr/local/libexec/twr-sandbox /bin/sh -ec '\''
    touch /var/lib/twr/write-probe
    if touch /usr/forbidden; then exit 1; fi
    if touch /etc/twr/forbidden; then exit 1; fi
    if test -e /root/twr; then exit 1; fi
    if test -e /etc/cloudflared; then exit 1; fi
    if mount -t tmpfs tmpfs /usr; then exit 1; fi
    test "$(id -u)" != 0
  '\''
PROBE
  chmod 755 /tmp/probe
  su -s /bin/sh twr -c /tmp/probe
  test -e /var/lib/twr/write-probe
  test ! -e /usr/forbidden
  test ! -e /etc/twr/forbidden
  # Host interface must not expose origin TLS, even inside the deployment network.
  ip=$(hostname -i | awk "{print \$1}")
  if curl --noproxy "*" --silent --connect-timeout 2 --insecure "https://$ip:8443/"; then exit 1; fi
  # Wrong trust is rejected; a direct client cannot skip the Tunnel peer contract.
  if curl --silent https://127.0.0.1:8443/not-a-route; then exit 1; fi
  status=$(curl --silent --cacert /etc/twr/origin.pem -o /dev/null -w "%{http_code}" https://127.0.0.1:8443/not-a-route)
  test "$status" = 400
  rc-service twr-server restart
  sleep 2
  rc-service twr-server status
  # The console dashboard (CHECKLIST.md) after a restart: the stale socket is replaced, the
  # server answers, and the faults this container really has (no backups, no Tunnel) show.
  test "$(stat -c %a:%U /var/lib/twr/status.sock)" = 600:twr
  sh /usr/local/libexec/twr-status > /tmp/status-result
  cat /tmp/status-result
  grep -F "OK       League server: Running and answering securely" /tmp/status-result
  grep -F "PROBLEM  Tunnel: Not connected to Cloudflare" /tmp/status-result
  grep -F "PROBLEM  Backups: Off-site backups are not set up" /tmp/status-result
  grep -F "OK       History: League history (0 entries)" /tmp/status-result
  grep -E "^(OK|PROBLEM) +Clock: " /tmp/status-result
  grep -E "^(OK|PROBLEM) +Disk: " /tmp/status-result
  if su -s /bin/sh twr -c "sh /usr/local/libexec/twr-status"; then exit 1; fi
  # A stopped server reads as a problem, not a blank screen.
  rc-service twr-server stop
  sh /usr/local/libexec/twr-status > /tmp/status-result
  grep -F "PROBLEM  League server: Not answering on this machine" /tmp/status-result
  grep -F "PROBLEM  Backups and history: Could not be read" /tmp/status-result
  rc-service twr-server start
  sleep 2
  test "$(stat -c %a:%U:%G /etc/twr-release/root.json)" = 644:root:root
  test "$(stat -c %a:%U:%G /var/lib/twr-release)" = 700:root:root
  # The real upgrade wrapper stops/starts OpenRC and verifies TLS. Only Tunnel is stubbed:
  # this disposable CT has no Cloudflare credentials and never contacts a real account.
  apk add --no-cache sqlite
  mkdir /tmp/upgrade-tools
  cat > /tmp/upgrade-tools/rc-service <<'\''SERVICE'\''
#!/bin/sh
if [ "$1" = twr-tunnel ]; then
  printf "%s\n" "$*" >> /tmp/upgrade-tunnel-calls
  exit 0
fi
exec /sbin/rc-service "$@"
SERVICE
  chmod 755 /tmp/upgrade-tools/rc-service
  rc-service twr-server stop
  sqlite3 /var/lib/twr/server.db "UPDATE server_release SET release=1;"
  chown twr:twr /var/lib/twr/server.db
  PATH=/tmp/upgrade-tools:$PATH sh /usr/local/libexec/twr-upgrade "$DELIVERY/good/server.tar.gz" "$DELIVERY/good" > /tmp/upgrade-result
  cat /tmp/upgrade-result
  grep -F "Server upgrade ready" /tmp/upgrade-result
  rc-service twr-server status
  saved=$(find /var/lib/twr/upgrades -name checkpoint.json)
  test -n "$saved"
  /usr/local/bin/twr-server checkpoint -db /var/lib/twr/server.db -check "$saved"
  test "$(sqlite3 /var/lib/twr/server.db "PRAGMA user_version")" = 12
  grep -F "twr-tunnel --ifstopped start" /tmp/upgrade-tunnel-calls
  # Freeze live DB writes so unchanged-state assertions measure the wrapper only.
  rc-service twr-server stop
  cat > /tmp/upgrade-tools/rc-service <<'\''SERVICE'\''
#!/bin/sh
printf "%s\n" "$*" >> /tmp/upgrade-service-calls
exit 0
SERVICE
  chmod 755 /tmp/upgrade-tools/rc-service
  for case in byte-flipped truncated script expired rollback lower-release no-pinned-root sandbox; do
    : > /tmp/upgrade-service-calls
    before=$(sha256sum /usr/local/bin/twr-server /var/lib/twr/server.db)
    pinned=$(sha256sum /etc/twr-release/root.json)
    delivery="$DELIVERY/$case"
    if [ "$case" = no-pinned-root ]; then
      delivery="$DELIVERY/good"
      mv /etc/twr-release/root.json /etc/twr-release/root.saved
    fi
    if PATH=/tmp/upgrade-tools:$PATH sh /usr/local/libexec/twr-upgrade "$delivery/server.tar.gz" "$delivery" > /tmp/upgrade-refusal 2>&1; then
      cat /tmp/upgrade-refusal; exit 1
    fi
    cat /tmp/upgrade-refusal
    test ! -s /tmp/upgrade-service-calls
    test "$before" = "$(sha256sum /usr/local/bin/twr-server /var/lib/twr/server.db)"
    if [ "$case" = sandbox ]; then
      grep -Fx sandbox-ok /var/lib/twr/candidate-marker
      rm /var/lib/twr/candidate-marker
    else
      test ! -e /usr/local/bin/twr-server.next
      test ! -e /var/lib/twr/candidate-marker
    fi
    if [ "$case" = no-pinned-root ]; then
      grep -F "reinstall from a verified bundle" /tmp/upgrade-refusal
      mv /etc/twr-release/root.saved /etc/twr-release/root.json
    fi
    test "$pinned" = "$(sha256sum /etc/twr-release/root.json)"
    echo "Upgrade refusal $case: PASS"
  done
  echo "Upgrade signed good bundle: PASS"
  sqlite3 /var/lib/twr/server.db "UPDATE server_release SET release=999;"
  before=$(sha256sum /var/lib/twr/server.db)
  : > /tmp/upgrade-service-calls
  if PATH=/tmp/upgrade-tools:$PATH sh /usr/local/libexec/twr-upgrade "$DELIVERY/newer-file/server.tar.gz" "$DELIVERY/newer-file" > /tmp/upgrade-refusal 2>&1; then exit 1; fi
  grep -F "Reinstall the newer server bundle" /tmp/upgrade-refusal
  test "$before" = "$(sha256sum /var/lib/twr/server.db)"
  test ! -s /tmp/upgrade-service-calls
  sqlite3 /var/lib/twr/server.db "UPDATE server_release SET release=5;"
  rc-service twr-server start
  sleep 2
  sh /src/deploy/server/test-cloudflare.sh
'
echo 'Alpine install, signed upgrade/refusals, OpenRC, TLS, sandbox and dashboard: PASS'
