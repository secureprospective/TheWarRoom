#!/usr/bin/env bash
# Lab only: one up/down cycle against server CT <ctid> while stress.py runs from OUTDIR.
# Usage: stress-cycle.sh <ctid> <kill9|pctstop|restart> OUTDIR
# Bursts 20 s so writes are in flight, takes the server down, checks the stopped database
# (integrity, head, confirmed-action reconcile), brings it back and waits until it answers.
set -euo pipefail
[[ $(hostname) == twr-lab ]] || { echo 'lab only' >&2; exit 97; }
ct=$1 kind=$2 out=$3
x() { pct exec "$ct" -- env PATH=/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "$@"; }
n=$(( $(find "$out" -maxdepth 1 -name "check-*.txt" | wc -l) + 1 ))
echo burst > "$out/phase"; sleep 20
t0=$(date +%s.%N)
case $kind in
  kill9)   x pkill -9 -f '^/usr/local/bin/twr-server hosted' ;;
  pctstop) pct stop "$ct" ;;
  restart) x rc-service twr-server stop </dev/null >/dev/null ;;
esac
echo pause > "$out/phase"
if [[ $kind == pctstop ]]; then
  pct start "$ct"
  for _ in $(seq 60); do x rc-service twr-server status </dev/null >/dev/null 2>&1 && break; sleep 1; done
  x rc-service twr-server stop </dev/null >/dev/null 2>&1 || true
fi
# supervise-daemon respawns after kill -9; stop whatever came back so the check reads a stopped database.
if [[ $kind == kill9 ]]; then
  sleep 2; x rc-service twr-server stop </dev/null >/dev/null 2>&1 || true
fi
x pgrep -f '^/usr/local/bin/twr-server hosted' >/dev/null && { echo 'server still running' >&2; exit 1; }
x su -s /bin/sh twr -c 'stresscheck -db /var/lib/twr/server.db' > "$out/check-$n-$kind.txt"
python3 "$(dirname "$0")/stress.py" reconcile "$out" "$out/check-$n-$kind.txt" > "$out/reconcile-$n-$kind.json"
x rc-service twr-server start </dev/null >/dev/null
x rc-service twr-tunnel start </dev/null >/dev/null 2>&1 || true
for _ in $(seq 120); do
  code=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H 'Content-Type: application/json' -H 'X-TWR-Protocol: 1.1' \
    --data '{}' "https://twr-lab-$ct.secureprospective.com/v1/native/ledger" || true)
  [[ $code =~ ^4 ]] && break; sleep 1
done
t1=$(date +%s.%N)
echo steady > "$out/phase"
x /usr/local/libexec/twr-status </dev/null > "$out/status-$n-$kind.txt" 2>&1 || true
printf '%s\n' "{\"cycle\":$n,\"kind\":\"$kind\",\"down_s\":$(echo "$t1 - $t0" | bc),\"answer\":\"$code\"}" >> "$out/cycles.jsonl"
cat "$out/reconcile-$n-$kind.json"
