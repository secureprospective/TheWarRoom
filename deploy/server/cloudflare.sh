#!/bin/sh
# Called by install.sh; the API token is never placed in a command argument.
set -eu
umask 077
mode=install
if [ "$1" = --rollback ]; then mode=rollback; shift; fi
input=$1
zone=$(jq -er '.zone' "$input")
hostname=$(jq -er '.hostname' "$input")
valid_public_host() {
  case "$1" in '' | .* | *. | *..* | *[!a-z0-9.-]*) return 1 ;; esac
  [ "${#1}" -le 253 ] || return 1
  printf '%s\n' "$1" | awk -F. '
    NF < 2 { exit 1 }
    {
      for (i = 1; i <= NF; i++) {
        if (length($i) == 0 || length($i) > 63 || substr($i, 1, 1) == "-" || substr($i, length($i), 1) == "-") exit 1
      }
    }
    NF == 4 {
      ip = 1
      for (i = 1; i <= NF; i++) if ($i !~ /^[0-9]+$/ || $i + 0 > 255) ip = 0
      if (ip) exit 1
    }
  '
}
if ! valid_public_host "$zone" || ! valid_public_host "$hostname"; then
  echo 'The zone and public hostname must be bare lowercase DNS names.' >&2
  exit 1
fi
case "$hostname" in *."$zone") ;; *) echo 'The public hostname must be inside the configured zone.' >&2; exit 1 ;; esac
header=$(mktemp)
reply=$(mktemp)
payload=$(mktemp)
created=0
cleanup() {
  result=$?
  trap - EXIT HUP INT TERM
  if [ "$result" != 0 ] && [ "$created" = 1 ]; then rollback || true; fi
  rm -f "$header" "$reply" "$payload"
  exit "$result"
}
trap cleanup EXIT HUP INT TERM
jq -er '"Authorization: Bearer " + .api_token' "$input" > "$header"
api() {
  method=$1 path=$2
  shift 2
  status=$(curl --fail-with-body --silent --show-error --connect-timeout 10 --max-time 60 \
    -X "$method" -H @"$header" -H 'Content-Type: application/json' \
    "https://api.cloudflare.com/client/v4/$path" "$@" --output "$reply" --write-out '%{http_code}') || {
    [ "$method" = DELETE ] && [ "$status" = 404 ] && return 0
    return 1
  }
  jq -e '.success == true' "$reply" >/dev/null
}
rollback() {
  ids=/etc/twr-cloudflare.ids
  [ -e "$ids" ] || return 0
  account=$(jq -er '.account' "$ids")
  tunnel=$(jq -r '.tunnel // empty' "$ids")
  zone_id=$(jq -er '.zone' "$ids")
  dns=$(jq -r '.dns // empty' "$ids")
  failed=0
  if [ -n "$dns" ]; then
    if api DELETE "zones/$zone_id/dns_records/$dns"; then
      jq 'del(.dns)' "$ids" > "$reply"
      cat "$reply" > "$ids"
    else failed=1; fi
  fi
  if [ -n "$tunnel" ]; then
    if api DELETE "accounts/$account/cfd_tunnel/$tunnel"; then
      jq 'del(.tunnel)' "$ids" > "$reply"
      cat "$reply" > "$ids"
    else failed=1; fi
  fi
  if [ "$failed" = 0 ]; then
    rm -f "$ids" /etc/cloudflared/credentials.json /etc/cloudflared/config.yml /etc/twr/public-host
  else
    echo 'Cloudflare cleanup could not finish; retained resource identifiers for recovery.' >&2
  fi
  return "$failed"
}
rollback
if [ "$mode" = rollback ]; then exit; fi
api GET "zones?name=$zone&status=active"
zone_id=$(jq -er '.result | select(length == 1) | .[0].id' "$reply")
account=$(jq -er '.result[0].account.id' "$reply")
api GET "zones/$zone_id/dns_records?name=$hostname"
old_dns=
old_tunnel=
if ! jq -e '.result | length == 0' "$reply" >/dev/null; then
  # Spare-host restore (RESTORE.md) may take over only a lost TheWarRoom server's record.
  jq -e '.replace == true' "$input" >/dev/null || {
    echo 'That hostname is already in use; nothing was replaced.' >&2; exit 1;
  }
  foreign() { echo 'That hostname does not point to a TheWarRoom server; nothing was replaced.' >&2; exit 1; }
  old_dns=$(jq -er '.result | select(length == 1) | .[0] | select(.type == "CNAME") | .id' "$reply") || foreign
  old_tunnel=$(jq -er '.result[0].content | capture("^(?<id>[0-9a-f-]{36})[.]cfargotunnel[.]com$").id' "$reply") || foreign
  api GET "accounts/$account/cfd_tunnel/$old_tunnel"
  jq -e '.result.name | startswith("twr-")' "$reply" >/dev/null || foreign
  jq -e '.result.deleted_at != null or (((.result.connections // []) | length) == 0 and (.result.status == "down" or .result.status == "inactive"))' "$reply" >/dev/null || {
    echo 'The old server is still connected to Cloudflare. Only replace a server that is really gone; nothing was replaced.' >&2; exit 1;
  }
  jq -e '.result.deleted_at == null' "$reply" >/dev/null || old_tunnel=
fi
secret=$(openssl rand -base64 32)
printf '%s\n' "$secret" | jq -Rnc --arg name "twr-$(cat /etc/hostname)" \
  'input as $secret | {name:$name,tunnel_secret:$secret,config_src:"local"}' > "$payload"
api POST "accounts/$account/cfd_tunnel" --data-binary @"$payload"
tunnel=$(jq -er '.result.id' "$reply")
created=1
jq -n --arg account "$account" --arg tunnel "$tunnel" --arg zone "$zone_id" \
  '{account:$account,tunnel:$tunnel,zone:$zone}' > /etc/twr-cloudflare.ids
printf '%s\n' "$secret" | jq -Rn --arg account "$account" --arg tunnel "$tunnel" \
  'input as $secret | {AccountTag:$account,TunnelSecret:$secret,TunnelID:$tunnel}' > /etc/cloudflared/credentials.json
jq -n --arg tunnel "$tunnel" --arg hostname "$hostname" \
  '{tunnel:$tunnel,"credentials-file":"/etc/cloudflared/credentials.json",
    "metrics":"127.0.0.1:20241","no-autoupdate":true,
    ingress:[{hostname:$hostname,path:"^/(v1/.*|app(/.*)?|join|password|site\\.css|site\\.js|steel\\.js|og\\.png|fonts/[a-z0-9-]+\\.woff2)?$",service:"https://127.0.0.1:8443",
      originRequest:{caPool:"/etc/cloudflared/origin.pem",originServerName:"localhost"}},
      {service:"http_status:404"}]}' > /etc/cloudflared/config.yml
jq -nc --arg name "$hostname" --arg target "$tunnel.cfargotunnel.com" \
  '{type:"CNAME",name:$name,content:$target,proxied:true,ttl:1}' > "$payload"
if [ -n "$old_dns" ]; then
  # Repoint in place: the hostname moves to the new Tunnel without a gap.
  api PATCH "zones/$zone_id/dns_records/$old_dns" --data-binary @"$payload"
else
  api POST "zones/$zone_id/dns_records" --data-binary @"$payload"
fi
# Keep non-secret identifiers for console incident response, not the API token.
dns=$(jq -er '.result.id' "$reply")
jq --arg dns "$dns" '.dns=$dns' /etc/twr-cloudflare.ids > "$reply"
cat "$reply" > /etc/twr-cloudflare.ids
# The lost server's Tunnel credential may sit on a stolen disk; delete that Tunnel.
if [ -n "$old_tunnel" ] && ! api DELETE "accounts/$account/cfd_tunnel/$old_tunnel"; then
  echo "Delete the old Tunnel $old_tunnel in the Cloudflare dashboard (Networks, Tunnels)." >&2
fi
chown -R root:twr-tunnel /etc/cloudflared
chmod 750 /etc/cloudflared
chmod 640 /etc/cloudflared/*
printf '%s\n' "$hostname" > /etc/twr/public-host
chown root:twr /etc/twr/public-host
chmod 640 /etc/twr/public-host
