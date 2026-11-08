#!/usr/bin/env python3
"""Summarise 'kind ms' samples: n, p50/p95/p99/max, and every breach of the budget (default 100 ms)."""
import sys

budget = float(sys.argv[1]) if len(sys.argv) > 1 else 100.0
groups = {}
for line in sys.stdin:
    parts = line.split()
    if len(parts) == 2:
        groups.setdefault(parts[0], []).append(float(parts[1]))

def pct(xs, p):
    xs = sorted(xs)
    k = (len(xs) - 1) * p / 100
    lo, hi = int(k), min(int(k) + 1, len(xs) - 1)
    return xs[lo] + (xs[hi] - xs[lo]) * (k - lo)

print(f"{'run':<12}{'n':>5}{'p50':>9}{'p95':>9}{'p99':>9}{'max':>9}  breaches>{budget:.0f}ms")
for kind, xs in groups.items():
    br = [x for x in xs if x > budget]
    print(f"{kind:<12}{len(xs):>5}{pct(xs,50):>9.1f}{pct(xs,95):>9.1f}{pct(xs,99):>9.1f}{max(xs):>9.1f}  "
          f"{len(br)} ({100*len(br)/len(xs):.1f}%)" + (f" worst: {', '.join(f'{x:.0f}' for x in sorted(br)[-5:])}" if br else ""))
