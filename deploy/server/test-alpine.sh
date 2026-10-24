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
  sh /root/twr/install.sh
  rc-service twr-server status
  # A second install refuses without replacing authoritative state or keys.
  before=$(sha256sum /etc/twr/authenticator.key /var/lib/twr/server.db)
  if sh /root/twr/install.sh; then exit 1; fi
  [ "$before" = "$(sha256sum /etc/twr/authenticator.key /var/lib/twr/server.db)" ]
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
  sh /src/deploy/server/test-cloudflare.sh
'
echo 'Alpine install, OpenRC start/restart, TLS, loopback-only bind and filesystem/capability sandbox: PASS'
