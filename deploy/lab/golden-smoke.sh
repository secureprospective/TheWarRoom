#!/usr/bin/env bash
# Prove the golden template: boot a server-shaped CT from it, check network, packages and no sshd; remove it. Lab only.
set -euo pipefail
[ "$(hostname)" = twr-lab ] || { echo "lab only" >&2; exit 97; }
ID=9001
T=alpine-3.23-twr-golden_20261108_amd64.tar.gz
cd /var/lib/vz/template/cache && sha256sum -c $T.sha256
[ ! -e /etc/pve/lxc/$ID.conf ] || { echo "CT $ID exists" >&2; exit 1; }
t0=$(date +%s.%N)
(umask 022; pct create $ID local:vztmpl/$T --hostname twr-smoke --unprivileged 1 --features nesting=1 \
  --cores 2 --memory 1024 --swap 0 --rootfs lab-thin:4 --onboot 0 --nameserver 1.1.1.1 \
  --net0 name=eth0,bridge=vmbr1,tag=51,ip=dhcp,ip6=manual,firewall=1 >/dev/null 2>&1)
pct start $ID
up=0; for _ in $(seq 1 40); do pct exec $ID -- wget -q --spider https://dl-cdn.alpinelinux.org/ 2>/dev/null && { up=1; break; }; sleep 0.5; done
t1=$(date +%s.%N)
[ $up = 1 ] || { echo "FAIL: no network"; exit 1; }
echo "create→online: $(echo "$t1 - $t0" | bc) s"
c() { pct exec $ID -- "$@"; }
echo "release: $(c cat /etc/alpine-release)  ip: $(c ip -4 addr show eth0 | awk '/inet /{print $2}')"
echo "rc default: $(c rc-status default 2>/dev/null | awk 'NR>1{print $1}' | tr '\n' ' ')"
echo "listening TCP: $(c netstat -tln 2>/dev/null | awk 'NR>2{print $4}' | tr '\n' ' ')"
# shellcheck disable=SC2016 # expands inside the CT
c sh -c '! apk info -e openssh-server >/dev/null && ! [ -e /usr/sbin/sshd ]' && echo "sshd: absent"
c sh -c 'ls /etc/ssh/ssh_host_* 2>/dev/null' && echo "FAIL host keys present" || echo "host keys: none"
# shellcheck disable=SC2016 # expands inside the CT
c sh -c 'for p in openrc bubblewrap openssl ca-certificates curl jq logrotate busybox-openrc; do apk info -e $p >/dev/null || echo MISSING $p; done; echo "installer pkgs: checked"'
c apk -q update && echo "upgradable: $(c apk -q version -l '<' | wc -l)"
pct stop $ID; pct destroy $ID; echo "smoke CT removed"
