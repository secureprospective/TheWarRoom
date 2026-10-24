#!/usr/bin/env bash
# One-command, interactive install from a prepared release bundle on a Proxmox host.
set -euo pipefail
umask 077
[[ $EUID == 0 ]] || { echo 'Run this installer as root on the Proxmox host.' >&2; exit 1; }
[[ $# == 1 && -f $1 ]] || { echo 'Provide one existing release bundle file.' >&2; exit 1; }
for tool in pct pveam pvesm pve-firewall chronyc realpath grep mktemp rm awk sort tail ip cat sleep; do
  command -v "$tool" >/dev/null || {
    echo "Required Proxmox host tool is missing: $tool. Nothing was changed; ask your host administrator for help." >&2
    exit 1
  }
done
# CTs inherit the host kernel clock; wait for real host synchronization.
chronyc waitsync 30 0.5 0 2 || {
  echo 'The Proxmox host clock is not synchronized. Nothing was changed; ask your host administrator for help.' >&2
  exit 1
}
pve-firewall status | grep -q 'Status: enabled/running' || {
  echo 'Proxmox firewall must already protect this host; refusing an unfiltered deployment.' >&2; exit 1;
}
bundle=$(realpath "$1")
read -rp 'Container number: ' ctid
read -rp 'Isolated network bridge [vmbr0]: ' bridge; bridge=${bridge:-vmbr0}
read -rp 'Isolated VLAN number: ' vlan
read -rp 'Container disk storage [local-lvm]: ' storage; storage=${storage:-local-lvm}
read -rp 'Template storage [local]: ' templates; templates=${templates:-local}
read -rp 'Cloudflare zone (example.org): ' zone
read -rp 'New public hostname (league.example.org): ' hostname
read -rsp 'Cloudflare API token (Tunnel Edit, DNS Edit, Zone Read): ' token; echo
[[ $ctid =~ ^[1-9][0-9]{2,8}$ && $vlan =~ ^[0-9]{1,4}$ ]]
(( vlan >= 1 && vlan <= 4094 ))
[[ $bridge =~ ^[a-zA-Z0-9_-]+$ && -d /sys/class/net/$bridge ]]
[[ $storage =~ ^[a-zA-Z0-9_-]+$ && $templates =~ ^[a-zA-Z0-9_-]+$ ]]
[[ $zone =~ ^[a-z0-9][a-z0-9.-]+\.[a-z]{2,}$ && $hostname == *."$zone" ]]
[[ $hostname =~ ^[a-z0-9][a-z0-9.-]+\.[a-z]{2,}$ && $token =~ ^[a-zA-Z0-9_-]+$ ]]
[[ ! -e /etc/pve/lxc/$ctid.conf && ! -e /etc/pve/firewall/$ctid.fw ]]
work=$(mktemp -d)
created=0
complete=0
cleanup() {
  rm -rf "$work"
  unset token
  if (( created && !complete )); then
    pct exec "$ctid" -- sh -ec 'if [ -f /root/twr-input.json ]; then rc-service twr-tunnel --ifstarted stop || true; sh /root/twr/cloudflare.sh --rollback /root/twr-input.json; fi' || true
    pct exec "$ctid" -- rm -f /root/twr-input.json 2>/dev/null || true
    pct set "$ctid" --onboot 0 || true
    pct stop "$ctid" || true
    echo "Install failed. Container $ctid is stopped and retained for diagnosis; no retry overwrites state." >&2
  fi
}
trap cleanup EXIT
# This bundle is trusted release input, not a league archive. Never extract it on the host.
pveam update
template=$(pveam available --section system | awk '$2 ~ /^alpine-3\.23-default_.*amd64.tar.xz$/ {print $2}' | sort -V | tail -1)
[[ -n $template ]] || { echo 'No Alpine 3.23 amd64 template is available.' >&2; exit 1; }
pveam download "$templates" "$template"
pct create "$ctid" "$templates:vztmpl/$template" --hostname "twr-$ctid" \
  --unprivileged 1 --features nesting=1 --cores 2 --memory 1024 --swap 0 \
  --rootfs "$storage:4" --onboot 0 --nameserver 1.1.1.1 \
  --net0 "name=eth0,bridge=$bridge,tag=$vlan,ip=dhcp,ip6=manual,firewall=1"
created=1
{
  cat <<'RULES'
[OPTIONS]
enable: 1
policy_in: DROP
policy_out: DROP
log_level_in: nolog
log_level_out: nolog

[RULES]
IN ACCEPT -p udp -sport 67 -dport 68
OUT ACCEPT -p udp -sport 68 -dport 67
OUT DROP -dest ::/0
RULES
  # Deny every connected host subnet too, including globally numbered management LANs.
  for subnet in 0.0.0.0/8 10.0.0.0/8 100.64.0.0/10 127.0.0.0/8 169.254.0.0/16 \
    172.16.0.0/12 192.0.0.0/24 192.168.0.0/16 198.18.0.0/15 224.0.0.0/4 240.0.0.0/4 \
    $(ip -4 route show scope link | awk '$1 ~ /^[0-9]+\./ {print $1}'); do
    printf 'OUT DROP -dest %s\n' "$subnet"
  done
  cat <<'RULES'
OUT ACCEPT -p udp -dest 1.1.1.1 -dport 53
OUT ACCEPT -p tcp -dest 1.1.1.1 -dport 53
OUT ACCEPT -p tcp -dport 443
OUT ACCEPT -p tcp -dport 7844
OUT ACCEPT -p udp -dport 7844
RULES
} > "/etc/pve/firewall/$ctid.fw"
pve-firewall compile >/dev/null
pct start "$ctid"
pct exec "$ctid" -- mkdir -m 700 /root/twr
pct push "$ctid" "$bundle" /root/twr.tar.gz --perms 0600
pct exec "$ctid" -- tar -xzf /root/twr.tar.gz -C /root/twr
# The validated domain/token alphabets contain no JSON escapes or control characters.
printf '{"zone":"%s","hostname":"%s","api_token":"%s"}\n' \
  "$zone" "$hostname" "$token" > "$work/input.json"
unset token
pct push "$ctid" "$work/input.json" /root/twr-input.json --perms 0600
pct exec "$ctid" -- sh -ec 'sed -i "s|http://|https://|g" /etc/apk/repositories; sh /root/twr/install.sh; sh /root/twr/cloudflare.sh /root/twr-input.json; /usr/local/bin/cloudflared --config /etc/cloudflared/config.yml tunnel ingress validate; rc-update add twr-tunnel default; rc-service twr-tunnel start'
# A real reboot, not merely writing an init file.
pct reboot "$ctid"
ready=0
for ((attempt=0; attempt<60; attempt++)); do
  if pct exec "$ctid" -- sh -ec 'rc-service twr-server status; rc-service twr-tunnel status; curl --fail --silent --max-time 2 http://127.0.0.1:20241/ready' >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 2
done
(( ready ))
pct exec "$ctid" -- rm -f /root/twr-input.json
pct set "$ctid" --onboot 1
complete=1
printf 'Installed: https://%s. Administration is Proxmox console only (pct enter %s).\n' "$hostname" "$ctid"
