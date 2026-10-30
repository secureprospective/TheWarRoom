#!/usr/bin/env bash
# Run inside a THROWAWAY Proxmox VM only (STAGING.md). Creates the isolated VLAN, DHCP, NAT,
# firewall and repo settings that proxmox.sh requires but never configures on a real host.
set -euo pipefail
vlan=${1:-51}
net=10.$vlan.0
rm -f /etc/apt/sources.list.d/pve-enterprise.sources /etc/apt/sources.list.d/ceph.sources
cat > /etc/apt/sources.list.d/pve-no-sub.sources <<EOF
Types: deb
URIs: http://download.proxmox.com/debian/pve
Suites: trixie
Components: pve-no-subscription
Signed-By: /usr/share/keyrings/proxmox-archive-keyring.gpg
EOF
apt-get -qq update
DEBIAN_FRONTEND=noninteractive apt-get -qq install -y dnsmasq tmux >/dev/null
systemctl disable --now dnsmasq >/dev/null 2>&1 || true
grep -q '^auto vmbr1$' /etc/network/interfaces || cat >> /etc/network/interfaces <<EOF

auto vmbr1
iface vmbr1 inet manual
	bridge-ports none
	bridge-stp off
	bridge-fd 0
	bridge-vlan-aware yes
	bridge-vids 2-4094

auto vmbr1.$vlan
iface vmbr1.$vlan inet static
	address $net.1/24
	post-up echo 1 > /proc/sys/net/ipv4/ip_forward
	post-up iptables -t nat -A POSTROUTING -s $net.0/24 -o vmbr0 -j MASQUERADE
EOF
ifreload -a
cat > /etc/dnsmasq.d/lab.conf <<EOF
interface=vmbr1.$vlan
bind-interfaces
dhcp-range=$net.100,$net.150,12h
dhcp-option=option:router,$net.1
dhcp-option=option:dns-server,1.1.1.1
EOF
systemctl enable --now dnsmasq
# An interface-qualified DHCP rule is not used: plain port rules are what this lab proved.
printf '%s\n' '[OPTIONS]' 'enable: 1' '' '[RULES]' 'IN ACCEPT -p tcp -dport 22' \
  'IN ACCEPT -p tcp -dport 8006' 'IN ACCEPT -p udp -dport 67' > /etc/pve/firewall/cluster.fw
systemctl restart pve-firewall
pve-firewall status
chronyc tracking | grep -E 'Leap status'
echo "Lab ready: bridge vmbr1, VLAN $vlan, DHCP $net.100-150."
