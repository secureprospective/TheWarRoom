#!/usr/bin/env bash
# Disposable Alpine execution, never a real league or Cloudflare account.
set -euo pipefail
cd "$(dirname "$0")/../.."
[[ -f .scratch/server.tar.gz ]] || { echo 'Run bundle.sh .scratch/server.tar.gz first.' >&2; exit 1; }
image=alpine:3.23@sha256:ff7f0c727dd796aa36787bfcd9a64bf35a6ce72a73451b3359fde849b6703d63
name="twr-cs34-test-$$"
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
# Privileges apply only to this disposable container's namespaces. No host mounts
# except the read-only candidate; no host network/PID namespace or real secrets.
docker run -d --init --name "$name" --privileged -v "$PWD:/src:ro" "$image" sleep 600 >/dev/null
docker exec "$name" sh -ec '
  mkdir -p /root/twr /run/openrc
  touch /run/openrc/softlevel
  tar xzf /src/.scratch/server.tar.gz -C /root/twr
  apk add --no-cache bash shellcheck
  shellcheck /src/deploy/server/proxmox.sh /src/deploy/server/install.sh /src/deploy/server/backup-setup.sh /src/deploy/server/restore.sh /src/deploy/server/reset-sign-ins.sh /src/deploy/server/status.sh /src/deploy/server/cloudflare.sh /src/deploy/server/test-alpine.sh
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
  if printf "endpoint\nbucket\naccess\nsecret\nnot-a-key\n" | PATH=/tmp/backup-tools:$PATH sh /usr/local/libexec/twr-restore > /tmp/restore-result 2>&1; then exit 1; fi
  grep -F "restore key is not right" /tmp/restore-result
  grep -F "twr-server --ifstopped start" /tmp/backup-service-calls
  [ "$before" = "$(sha256sum /etc/twr/authenticator.key /var/lib/twr/server.db)" ]
  test ! -e /var/lib/twr/restore
  test ! -e /etc/twr/backup.json
  # The incident reset runs as the service user and keeps its files owned by twr.
  printf "alpine drill\n" | PATH=/tmp/backup-tools:$PATH sh /usr/local/libexec/twr-reset-sign-ins | grep -F "on 0 accounts"
  [ -z "$(find /var/lib/twr ! -user twr)" ]
  # A second install refuses without replacing authoritative state or keys.
  before=$(sha256sum /etc/twr/authenticator.key /var/lib/twr/server.db)
  if sh /root/twr/install.sh; then exit 1; fi
  [ "$before" = "$(sha256sum /etc/twr/authenticator.key /var/lib/twr/server.db)" ]
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
  sed "/  \/usr\/local\/bin\/twr-server hosted/,$ d" /usr/local/libexec/twr-sandbox > /tmp/probe
  cat >> /tmp/probe <<'\''PROBE'\''
  /bin/sh -ec '\''
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
  sh /src/deploy/server/test-cloudflare.sh
'
echo 'Alpine install, OpenRC start/restart, TLS, loopback-only bind, filesystem/capability sandbox and console dashboard: PASS'
