#!/bin/sh
# Run only inside a fresh Alpine CT, with this directory and both binaries in /root/twr.
set -eu
umask 077
cd /root/twr
[ "$(id -u)" = 0 ] && [ "$(uname -m)" = x86_64 ]
# shellcheck source=/dev/null
. /etc/os-release
[ "$ID" = alpine ] && [ "${VERSION_ID%%.*}" = 3 ]
[ ! -e /etc/twr ] || { echo 'An installation already exists; refusing to replace keys or state.' >&2; exit 1; }
apk add --no-cache openrc bubblewrap openssl ca-certificates curl jq logrotate busybox-openrc
# Proxmox templates may already have SSH enabled. Console is the only admin door.
if apk info -e openssh-server >/dev/null; then
  rc-service sshd --ifstarted stop
  apk del openssh-server
fi
addgroup -S twr
adduser -S -D -H -s /sbin/nologin -G twr twr
addgroup -S twr-tunnel
adduser -S -D -H -s /sbin/nologin -G twr-tunnel twr-tunnel
install -d -m 700 -o twr -g twr /var/lib/twr
install -d -m 750 -o root -g twr /etc/twr
install -d -m 750 -o root -g twr-tunnel /etc/cloudflared
install -d -m 755 /usr/local/bin /usr/local/libexec
install -m 755 twr-server cloudflared /usr/local/bin/
install -m 755 twr-sandbox /usr/local/libexec/
install -m 755 twr-server.initd /etc/init.d/twr-server
install -m 755 cloudflared.initd /etc/init.d/twr-tunnel
install -m 600 twr-logrotate /etc/logrotate.d/twr
# Run rotation hourly; logs and rotated copies stay private inside the CT.
ln -sf /etc/periodic/daily/logrotate /etc/periodic/hourly/twr-logrotate
rc-update add crond default
rc-service crond start
install -m 640 -o root -g twr endpoint-registry.csv /etc/twr/
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
  -keyout /etc/twr/origin.key -out /etc/twr/origin.pem -days 36500 \
  -subj /CN=localhost -addext 'subjectAltName=DNS:localhost,IP:127.0.0.1'
openssl rand -out /etc/twr/authenticator.key 32
# Key reads deliberately require owner-only mode in twr-server.
chown twr:twr /etc/twr/origin.key /etc/twr/authenticator.key
chmod 600 /etc/twr/origin.key /etc/twr/authenticator.key
chgrp twr /etc/twr/origin.pem
chmod 640 /etc/twr/origin.pem
install -m 640 -o root -g twr-tunnel /etc/twr/origin.pem /etc/cloudflared/origin.pem
install -m 600 -o twr -g twr /dev/null /var/lib/twr/server.db
# An unprivileged CT shares the host's clock. Never run a fake NTP client with -x:
# it cannot synchronize the kernel state that hosted admission checks.
install -m 755 backup-setup.sh /usr/local/libexec/twr-backup-setup
install -m 755 upgrade.sh /usr/local/libexec/twr-upgrade
install -m 755 restore.sh /usr/local/libexec/twr-restore
install -m 755 reset-sign-ins.sh /usr/local/libexec/twr-reset-sign-ins
install -m 755 status.sh /usr/local/libexec/twr-status
rc-update add twr-server default
# No SSH, HTTP admin listener or public metrics service is installed. The status
# dashboard is a console command reading an owner-only socket (CHECKLIST.md).
rc-service twr-server start
# Check TLS and an actual application response, not just an open socket.
attempt=0
while [ "$attempt" -lt 30 ]; do
  status=$(curl --silent --cacert /etc/twr/origin.pem --max-time 2 -H 'CF-Connecting-IP: 127.0.0.1' \
    --output /dev/null --write-out '%{http_code}' https://127.0.0.1:8443/not-a-route) || status=000
  if [ "$status" = 404 ]; then
    echo 'Next, complete the R2 backup setup in BACKUPS.md; league changes stay paused until backups verify.'
    echo 'Then move your league to the server from the desktop app.'
    echo 'Replacing a lost server instead? Run sh /usr/local/libexec/twr-restore now (RESTORE.md).'
    exit 0
  fi
  attempt=$((attempt + 1))
  sleep 1
done
echo 'League service did not become ready; Tunnel remains stopped.' >&2
exit 1
