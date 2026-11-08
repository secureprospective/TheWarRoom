#!/usr/bin/env bash
# Build the TWR golden Alpine template on the lab PVE (VM 503 twr-lab). Lab only.
# Output: local:vztmpl/alpine-3.23-twr-golden_<date>_amd64.tar.gz (+ .manifest, .sha256)
# Holds NO per-server secrets: install.sh still mints keys, DB and release pin per server.
set -euo pipefail
[ "$(hostname)" = twr-lab ] || { echo "lab only" >&2; exit 97; }
ID=9000
CACHE=/var/lib/vz/template/cache
DATE=$(date +%Y%m%d)
NAME=alpine-3.23-twr-golden_${DATE}_amd64.tar.gz
# Same base selection rule as proxmox.sh.
pveam update >/dev/null
base=$(pveam available --section system | awk '$2 ~ /^alpine-3\.23-default_.*amd64.tar.xz$/ {print $2}' | sort -V | tail -1)
[ -n "$base" ] || { echo "no alpine 3.23 base" >&2; exit 1; }
[ -f "$CACHE/$base" ] || pveam download local "$base"
# Only ever reuse ID 9000 if it is our own builder.
if [ -e /etc/pve/lxc/$ID.conf ]; then
  grep -q '^hostname: twr-golden-build$' /etc/pve/lxc/$ID.conf || { echo "CT $ID is not the builder" >&2; exit 1; }
  pct stop $ID 2>/dev/null || true
  pct destroy $ID
fi
# Mirror proxmox.sh's container shape exactly.
(umask 022; pct create $ID "local:vztmpl/$base" --hostname twr-golden-build \
  --unprivileged 1 --features nesting=1 --cores 2 --memory 1024 --swap 0 \
  --rootfs lab-thin:4 --onboot 0 --nameserver 1.1.1.1 \
  --net0 name=eth0,bridge=vmbr1,tag=51,ip=dhcp,ip6=manual)
pct start $ID
net=0
for i in $(seq 1 30); do pct exec $ID -- wget -q --spider https://dl-cdn.alpinelinux.org/ 2>/dev/null && { net=1; break; }; sleep 2; done
[ $net = 1 ] || { echo "FAIL: builder has no network (check lab-gw CT 110)" >&2; exit 1; }
pct exec $ID -- sh -euc '
sed -i "s|http://|https://|g" /etc/apk/repositories
apk update -q
apk upgrade -q --available
# The exact package set install.sh adds, so install.sh is a no-op for apk.
apk add -q openrc bubblewrap openssl ca-certificates curl jq logrotate busybox-openrc
if apk info -e openssh-server >/dev/null; then rc-service sshd --ifstarted stop || true; apk del -q openssh-server; fi
rm -f /etc/ssh/ssh_host_* /root/.ash_history
rm -rf /var/cache/apk/* /tmp/* /var/tmp/*
find /var/log -type f -exec truncate -s 0 {} +
'
# Proof gates: no sshd; any openssh piece present must be at or past the sshole fix (CVE-2026-123158).
pct exec $ID -- sh -euc '
! apk info -e openssh-server >/dev/null || { echo "FAIL: openssh-server present"; exit 1; }
for p in $(apk info | grep "^openssh" || true); do
  v=$(apk info -e -v "$p" | sed "s/^$p-//")
  [ "$(apk version -t "$v" 10.2_p1-r1)" != "<" ] || { echo "FAIL: $p $v below fixed 10.2_p1-r1"; exit 1; }
done
echo "alpine $(cat /etc/alpine-release)"
'
pct exec $ID -- apk info -v | sort > "$CACHE/$NAME.manifest"
# BusyBox tar: keep the mount-point dirs, never their live (virtual) contents.
pct exec $ID -- sh -c 'printf "%s\n" "./proc/*" "./sys/*" "./dev/*" "./run/*" "./tmp/*" "./lost+found" "./tar.exclude" > /tar.exclude'
pct exec $ID -- tar -C / -X /tar.exclude --numeric-owner -czf - . > "$CACHE/$NAME.part"
# Archive proof: mount points present and empty, no exclude file, sshd absent.
list=$(tar -tzf "$CACHE/$NAME.part")
for d in ./proc/ ./sys/ ./dev/ ./etc/apk/world; do grep -qx "$d" <<<"$list" || { echo "FAIL: $d missing from archive" >&2; exit 1; }; done
if grep -qE '^\./(proc|sys|dev)/.+|^\./tar\.exclude$|usr/sbin/sshd$' <<<"$list"; then echo "FAIL: archive has live/forbidden entries" >&2; exit 1; fi
echo "archive entries: $(wc -l <<<"$list")"
pct stop $ID
pct destroy $ID
mv "$CACHE/$NAME.part" "$CACHE/$NAME"
(cd $CACHE && sha256sum "$NAME" > "$NAME.sha256")
ls -l "$CACHE/$NAME"*
echo "openssh pieces:"; grep '^openssh' "$CACHE/$NAME.manifest" || echo "  none"
echo "packages: $(wc -l < "$CACHE/$NAME.manifest")"
