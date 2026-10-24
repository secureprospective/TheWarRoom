#!/bin/sh
# Called by install.sh; the API token is never placed in a command argument.
set -eu
umask 077
mode=install
if [ "$1" = --rollback ]; then mode=rollback; shift; fi
input=$1
zone=$(jq -er '.zone' "$input")
hostname=$(jq -er '.hostname' "$input")
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
    rm -f "$ids" /etc/cloudflared/credentials.json /etc/cloudflared/config.yml
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
jq -e '.result | length == 0' "$reply" >/dev/null || {
  echo 'That hostname is already in use; nothing was replaced.' >&2; exit 1;
}
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
    ingress:[{hostname:$hostname,path:"^/v1/.*$",service:"https://127.0.0.1:8443",
      originRequest:{caPool:"/etc/cloudflared/origin.pem",originServerName:"localhost"}},
      {service:"http_status:404"}]}' > /etc/cloudflared/config.yml
jq -nc --arg name "$hostname" --arg target "$tunnel.cfargotunnel.com" \
  '{type:"CNAME",name:$name,content:$target,proxied:true,ttl:1}' > "$payload"
api POST "zones/$zone_id/dns_records" --data-binary @"$payload"
# Keep non-secret identifiers for console incident response, not the API token.
dns=$(jq -er '.result.id' "$reply")
jq --arg dns "$dns" '.dns=$dns' /etc/twr-cloudflare.ids > "$reply"
cat "$reply" > /etc/twr-cloudflare.ids
chown -R root:twr-tunnel /etc/cloudflared
chmod 750 /etc/cloudflared
chmod 640 /etc/cloudflared/*
