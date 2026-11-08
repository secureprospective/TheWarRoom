#!/usr/bin/env bash
# Lab only: the whole stress run against CT <ctid>: steady -> burst -> steady -> 15 up/down cycles
# (5 each of kill9, pctstop, restart, interleaved, under load) -> steady -> stop. Status every 10 min.
# Usage: stress-schedule.sh <ctid> STATE OUTDIR   (detach it; it runs for hours)
set -uo pipefail
ct=$1 state=$2 out=$3 dir=$(cd "$(dirname "$0")" && pwd)
base=https://twr-lab-$ct.secureprospective.com
mark() { echo "$(date +%s) $*" >> "$out/timeline.txt"; }
x() { pct exec "$ct" -- env PATH=/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "$@"; }
( while [[ $(cat "$out/phase" 2>/dev/null) != stop ]]; do
    { date +%s; x /usr/local/libexec/twr-status </dev/null 2>&1 | grep -E '^(OK|PROBLEM)' | cut -c1-120; } >> "$out/status-log.txt"
    sleep 600; done ) &
echo steady > "$out/phase"; mark steady1
python3 -u "$dir/stress.py" run "$base" "$state" "$out" > "$out/run.log" 2>&1 &
sleep 2100
echo burst > "$out/phase"; mark burst; sleep 600
echo steady > "$out/phase"; mark steady2; sleep 1800
for i in 1 2 3 4 5; do
  for kind in kill9 pctstop restart; do
    mark "cycle $kind start"; "$dir/stress-cycle.sh" "$ct" "$kind" "$out" >> "$out/cycles.log" 2>&1; mark "cycle $kind end rc=$?"
    sleep 300
  done
done
echo steady > "$out/phase"; mark steady3; sleep 1800
echo stop > "$out/phase"; mark stop; sleep 60
x rc-service twr-server stop </dev/null >/dev/null 2>&1
x su -s /bin/sh twr -c 'stresscheck -db /var/lib/twr/server.db' > "$out/check-final.txt"
python3 "$dir/stress.py" reconcile "$out" "$out/check-final.txt" > "$out/reconcile-final.json"
x rc-service twr-server start </dev/null >/dev/null; x rc-service twr-tunnel start </dev/null >/dev/null 2>&1
mark done
