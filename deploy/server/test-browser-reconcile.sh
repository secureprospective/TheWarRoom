#!/bin/sh
# Isolated regression tests for the install-time Tunnel/browser reconciliation.
set -eu
script_dir=$(CDPATH='' cd -- "$(dirname "$0")" && pwd)
status_script=${1:-"$script_dir/status.sh"}
work_root=${TMPDIR:-"$script_dir/.scratch"}
made_work_root=no
if [ ! -d "$work_root" ]; then mkdir -p "$work_root"; made_work_root=yes; fi
tmp=$(mktemp -d "$work_root/browser-reconcile.XXXXXX")
cleanup() {
  rm -rf "$tmp"
  [ "$made_work_root" = no ] || rmdir "$work_root"
}
trap cleanup EXIT HUP INT TERM

make_config() {
  jq -n --arg host "$1" --arg path "$2" \
    '{ingress:[{hostname:$host,path:$path,service:"https://127.0.0.1:8443",originRequest:{originServerName:"localhost"}},{service:"http_status:404"}]}'
}

config="$tmp/config.yml"
host_file="$tmp/public-host"
make_config league.example.org '^/(v1/.*|join|password)?$' > "$config"
chmod 640 "$config"
output=$(sh "$status_script" --prepare-browser "$config" "$host_file")
printf '%s\n' "$output" | grep -Fq 'added /app ingress and recovered the public hostname'
if printf '%s\n' "$output" | grep -Fq 'league.example.org'; then exit 1; fi
jq -e '.ingress[0].path == "^/(v1/.*|join|password|app(/.*)?)?$" and .ingress[0].originRequest.originServerName == "localhost" and .ingress[0].service == "https://127.0.0.1:8443" and .ingress[1].service == "http_status:404"' "$config" >/dev/null
grep -Fqx 'league.example.org' "$host_file"
test "$(stat -c '%a' "$host_file")" = 644
if [ "$(id -u)" = 0 ]; then test "$(stat -c '%U:%G' "$host_file")" = root:root; fi
before=$(sha256sum "$config" | cut -d ' ' -f 1)
sh "$status_script" --prepare-browser "$config" "$host_file" >/dev/null
test "$before" = "$(sha256sum "$config" | cut -d ' ' -f 1)"

# A hostless tunnel config still gets the ingress fix; the browser door stays off without a hostname.
unknown_config="$tmp/unknown-config.yml"
unknown_host="$tmp/unknown-public-host"
make_config '' '^/(v1/.*|join|password)?$' > "$unknown_config"
output=$(sh "$status_script" --prepare-browser "$unknown_config" "$unknown_host")
printf '%s\n' "$output" | grep -Fq 'Browser app disabled: no valid public hostname'
test ! -e "$unknown_host"
jq -e '.ingress[0].path == "^/(v1/.*|join|password|app(/.*)?)?$"' "$unknown_config" >/dev/null

# First-load upgrades start the Tunnel before the server: reconcile ingress without writing a host.
first_load_config="$tmp/first-load-config.yml"
first_load_host="$tmp/first-load-public-host"
make_config league.example.org '^/(v1/.*|join|password)?$' > "$first_load_config"
sh "$status_script" --prepare-ingress "$first_load_config" "$first_load_host" >/dev/null
jq -e '.ingress[0].path == "^/(v1/.*|join|password|app(/.*)?)?$"' "$first_load_config" >/dev/null
test ! -e "$first_load_host"

# Unknown route syntax is left untouched and never prevents the service starting.
unsupported_config="$tmp/unsupported-config.yml"
unsupported_host="$tmp/unsupported-public-host"
make_config league.example.org '^/custom$' > "$unsupported_config"
before=$(sha256sum "$unsupported_config" | cut -d ' ' -f 1)
output=$(sh "$status_script" --prepare-browser "$unsupported_config" "$unsupported_host")
printf '%s\n' "$output" | grep -Fq 'first Tunnel ingress path is not a supported route pattern'
test ! -e "$unsupported_host"
test "$before" = "$(sha256sum "$unsupported_config" | cut -d ' ' -f 1)"
printf '%s\n' 'Browser ingress reconciliation tests: PASS'
