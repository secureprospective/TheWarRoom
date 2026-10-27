#!/usr/bin/env python3
"""E5 oracle: p6/p5 study caches and prior-season full-season E3 rates (TFL/carries)."""
import sys
from collections import defaultdict

sys.path.insert(0, "/home/chris/fleet/runs/config-study-p6-2026-10-27")
import p6
import legacy_scorer as legacy

# Derived by TWR_RTL_YEAR=2024 go test -run '^TestRTLStudy$' -v ./internal/fantasyscore/source.
RATES = {"QB": 78 / 2352, "RB": 1106 / 11832, "WR": 62 / 430, "TE": 7 / 62}
PLAYERS = {"13138": "Hunt", "13130": "McCaffrey", "14783": "Hurts", "13589": "Allen", "13604": "Barkley"}
package = dict(p6.PACKAGES["A light"], DSNP=0.008, RTL=0)
rules = p6.rules_for(package)
c = p6.V.ctx(2025)
for pid, name in PLAYERS.items():
    pos = c["pos"][pid]
    breakdown = defaultdict(float)
    for week, event in c["byp"][pid].items():
        if not 1 <= week <= 13:
            continue
        for rule in rules:
            breakdown[rule["event"]] += legacy.score(event, pos, [rule])
        breakdown["RTL"] += -0.3124 * (event.get("RTL", 0) - RATES[pos] * event.get("RA", 0))
    print(f"{name} {pos} rate={RATES[pos]:.8f} total={sum(breakdown.values()):.6f}")
    print(" ".join(f"{k}={v:.4f}" for k, v in sorted(breakdown.items()) if abs(v) > 1e-9))
