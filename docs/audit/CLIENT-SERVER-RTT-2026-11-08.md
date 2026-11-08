# Client/server step 1 — real-tunnel RTT, desktop (2026-11-08)

Plan: [CLIENT-SERVER-PLAN-2026-11-05](../plans/CLIENT-SERVER-PLAN-2026-11-05.md), step 1 gate: record the real-tunnel RTT
distribution on home broadband (desktop and modern phone), compare with the 100 ms confirmed-action budget, report every breach.
**This file covers the desktop only. The phone run is still open.**

## Setup
- Server: lab CT 201 (`deploy/lab/twr-up`), bundle `twr-server-LAB-ONLY-r6-376f15431`, golden template `alpine-3.23-twr-golden_20261108`,
  real `proxmox.sh` install, real Cloudflare Tunnel `twr-lab-201.secureprospective.com`. No league loaded.
- Client: Claude-OS (Debian 13 VM on the Beelink), home broadband, curl HTTP/2. Tools: `deploy/lab/rtt-desktop.sh`, `deploy/lab/rtt-stats.py`.
- **app** = `/v1/not-a-route`: Cloudflare edge → Tunnel → origin TLS → app router (app 404 with its CSP header). No league work: the network floor.
- **edge** = `/cdn-cgi/trace`: answered by Cloudflare's edge. It separates broadband→edge from edge→lab.
- **warm** = one reused connection, sequential requests (first request excluded). **cold** = new process each time: DNS + TCP + TLS + request.
- Taken 2026-11-08 ~07:35 CT (13:35 UTC), Sunday morning.

## Results (ms)

| run | n | p50 | p95 | p99 | max | breaches > 100 ms |
|---|---:|---:|---:|---:|---:|---|
| app-warm | 499 | 32.4 | 34.9 | 44.9 | 48.8 | 0 (0.0%) |
| app-cold | 60 | 93.2 | 104.1 | 108.2 | 110.6 | **8 (13.3%)**: 103, 104, 105, 107, 111 among them |
| edge-warm | 499 | 13.6 | 15.2 | 22.2 | 165.1 | 1 (0.2%): 165 |
| edge-cold | 60 | 72.6 | 80.1 | 96.9 | 119.0 | 1 (1.7%): 119 |

## What it means (no success redefined)
- **Warm connection: the floor fits the budget.** p99 is 45 ms, which leaves about 55 ms of the 100 ms confirmed-action budget for server work and rendering.
  Tunnel + origin add about 19 ms over the edge (p50 32.4 vs 13.6).
- **Cold connection: the floor alone breaches.** 13.3% of fresh connections exceed 100 ms before any server work, so every
  confirmed action on a cold connection will be over budget once server time is added. This is a physical condition (DNS + TCP + TLS to the edge),
  not something to tune away by redefining the budget.
- Single edge-warm outlier (165 ms) shows broadband jitter exists. One sample in 499 is not a pattern, but long runs should expect it.

## Escalation for Christopher's ruling
Cold-connection breaches are certain under the current budget. Options to rule on (not decided here):
1. Keep connections warm. The plan's topic SSE stream already holds a connection open while a screen is up; confirmed actions then ride a warm connection.
   Measure the first action after app start / long idle separately and report it as its own number.
2. Accept that "first action after idle" is outside the 100 ms budget and report it separately (that would be a ruling, not a redefinition by us).

## Still open in step 1
- Phone run (S25 over CDP) — needs the phone connected to the Beelink.
- Contract lock: hosted `runNative` + command catalog, locked screens and CLI/MCP reaching the same service with identical outcomes/rejections.
