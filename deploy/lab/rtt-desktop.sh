#!/usr/bin/env bash
# Plan step 1: real-tunnel RTT floor from a desktop on home broadband.
# Usage: rtt-desktop.sh <https-origin> [warm_n] [cold_n]   → raw samples (ms) on stdout as "warm <ms>" / "cold <ms>"
# Default /v1/not-a-route = full path: Cloudflare edge → Tunnel → origin TLS → app router (app 404). RTT_PATH=/cdn-cgi/trace RTT_CODE=200 = edge only. Floor, no league work.
set -euo pipefail
origin=$1; warm=${2:-300}; cold=${3:-50}
url="$origin${RTT_PATH:-/v1/not-a-route}"; want=${RTT_CODE:-404}
code=$(curl -s -o /dev/null -w '%{http_code}' "$url"); [ "$code" = "$want" ] || { echo "expected $want, got $code" >&2; exit 1; }
# Warm: one curl process, one reused connection, sequential requests (like a live client).
cfg=$(mktemp); trap 'rm -f "$cfg"' EXIT
for _ in $(seq "$warm"); do printf 'url = "%s"\noutput = "/dev/null"\n' "$url"; done > "$cfg"
curl -s --http2 -K "$cfg" -w 'warm %{time_total} %{num_connects} %{http_code}\n' |
  awk -v w="$want" '$3==0 && $4==w {printf "warm %.2f\n", $2*1000} $3!=0 && NR>1 {bad++} END {if (bad) print "note new-connections-mid-run", bad > "/dev/stderr"}'
# Cold: fresh process each time = DNS + TCP + TLS + request (first click after idle / app start).
for _ in $(seq "$cold"); do
  curl -s -o /dev/null -w 'cold %{time_total} %{http_code}\n' "$url" | awk -v w="$want" '$3==w {printf "cold %.2f\n", $2*1000}'
  sleep 0.2
done
