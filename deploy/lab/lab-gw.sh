#!/usr/bin/env bash
# Lab gateway: one tiny Alpine CT routes + NATs + serves DHCP for the lab server VLAN.
# Why: when the PVE host itself routes a VLAN it also bridges, netfilter's bridge pass
# fixes the NAT decision before the routed pass (proven 11-08: UNREPLIED, un-NATed source
# on vmbr0; DHCP marked INVALID). A real host has its router upstream; so does the lab now.
set -euo pipefail
[ "$(hostname)" = twr-lab ] || { echo "lab only" >&2; exit 97; }
GW=110 VLAN=51 NET=10.51.0
STAMP=$(date +%Y%m%d-%H%M%S)
[ ! -e /etc/pve/lxc/"$GW".conf ] || { echo "CT $GW exists" >&2; exit 1; }
base=$(find /var/lib/vz/template/cache -maxdepth 1 -name "alpine-3.23-default_*_amd64.tar.xz" -printf "%f\n" | sort -V | tail -1)

# 1. Host stops routing the lab VLAN: drop vmbr1.51 (vmbr1 stays), stop host dnsmasq.
cp -a /etc/network/interfaces /root/interfaces.bak-"$STAMP"
cp -a /etc/pve/firewall/cluster.fw /root/cluster.fw.bak-"$STAMP"
iptables -t nat -D POSTROUTING -s "$NET.0/24" -o vmbr0 -j MASQUERADE 2>/dev/null || true
awk -v s="auto vmbr1.$VLAN" -v t="iface vmbr1.$VLAN " '
  $0==s {skip=1; next}
  index($0,t)==1 {skip=1; next}
  skip && /^[ \t]/ {next}
  {skip=0; print}' /root/interfaces.bak-"$STAMP" | cat -s > /etc/network/interfaces
ifdown "vmbr1.$VLAN" 2>/dev/null || ip link del "vmbr1.$VLAN" 2>/dev/null || true
ifreload -a
systemctl disable --now dnsmasq
# Host serves no DHCP any more: remove that now-dead rule.
grep -v 'IN ACCEPT -p udp -dport 67' /root/cluster.fw.bak-"$STAMP" > /etc/pve/firewall/cluster.fw
pve-firewall compile >/dev/null; pve-firewall restart

# 2. Gateway CT: eth0 on the home LAN (DHCP), eth1 = lab VLAN gateway.
(umask 022; pct create "$GW" "local:vztmpl/$base" --hostname lab-gw \
  --unprivileged 1 --cores 1 --memory 128 --swap 0 --rootfs lab-thin:1 \
  --onboot 1 --startup order=1 --nameserver 1.1.1.1 \
  --net0 name=eth0,bridge=vmbr0,ip=dhcp,ip6=manual \
  --net1 name=eth1,bridge=vmbr1,tag=$VLAN,ip=$NET.1/24,ip6=manual)
pct start "$GW"
for _ in $(seq 1 30); do pct exec "$GW" -- wget -q --spider https://dl-cdn.alpinelinux.org/ 2>/dev/null && break; sleep 2; done
pct exec "$GW" -- sh -euc "
sed -i 's|http://|https://|g' /etc/apk/repositories
apk update -q && apk upgrade -q --available
apk add -q dnsmasq iptables
if apk info -e openssh-server >/dev/null; then rc-service sshd --ifstarted stop || true; apk del -q openssh-server; fi
echo 'net.ipv4.ip_forward = 1' > /etc/sysctl.d/90-lab-gw.conf
sysctl -p /etc/sysctl.d/90-lab-gw.conf
cat > /etc/dnsmasq.d/lab.conf <<EOF
port=0
interface=eth1
bind-interfaces
dhcp-range=$NET.100,$NET.150,12h
dhcp-option=option:router,$NET.1
dhcp-option=option:dns-server,1.1.1.1
EOF
iptables -F; iptables -t nat -F
iptables -P FORWARD DROP
iptables -A FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
# Lab servers never reach the home LAN (defence in depth behind the installer's CT firewall).
iptables -A FORWARD -i eth1 -d 192.168.0.0/16 -j DROP
iptables -A FORWARD -i eth1 -o eth0 -j ACCEPT
iptables -t nat -A POSTROUTING -s $NET.0/24 -o eth0 -j MASQUERADE
rc-update add iptables default; /etc/init.d/iptables save
rc-update add dnsmasq default; rc-service dnsmasq restart
"
echo "gateway up: $(pct exec "$GW" -- ip -4 -br addr | tr '\n' ' ')"
