#!/usr/bin/env bash
# Lab only: CPU/alloc profiles of a twr-server built with -tags labpprof (pprof on the CT's loopback).
#   stress-profile.sh <ctid> STATE OUTDIR screens        each screen hammered alone for 20 s under a 20 s CPU profile
#   stress-profile.sh <ctid> STATE OUTDIR load <devices> a 60 s CPU profile + allocs after 60 s of `stress.py run`
# Profiles hold function names and counts only, never league rows. Run on the lab host.
set -uo pipefail
ct=$1 state=$2 out=$3 mode=$4 dir=$(cd "$(dirname "$0")" && pwd)
base=https://twr-lab-$ct.secureprospective.com
mkdir -p "$out"
pp() { pct exec "$ct" -- wget -qO- "http://127.0.0.1:6060/debug/pprof/$1"; }
if [[ $mode == screens ]]; then
  for s in $(python3 -c "import sys; sys.path.insert(0, '$dir'); import stress; print(' '.join(n for n, _ in stress.SCREENS))"); do
    pp "profile?seconds=20" > "$out/cpu-$s.pb.gz" &
    python3 "$dir/stress.py" hammer "$base" "$state" "$s" 20 >> "$out/hammer.jsonl"
    wait
  done
else
  n=$5 o=$out/load-$n; mkdir -p "$o"; echo steady > "$o/phase"
  STRESS_DEVICES=$n python3 -u "$dir/stress.py" run "$base" "$state" "$o" > "$o/run.log" 2>&1 &
  sleep 60
  pp "allocs" > "$o/allocs-0.pb.gz"
  pp "profile?seconds=60" > "$o/cpu.pb.gz"
  pp "allocs" > "$o/allocs-1.pb.gz"
  pp "heap?debug=1" | grep -E '^# (HeapAlloc|HeapInuse|NumGC|PauseTotalNs|NextGC|Sys) ' > "$o/heap.txt"
  sleep 60; echo stop > "$o/phase"; sleep 30; pkill -f "[s]tress.py run"
fi
