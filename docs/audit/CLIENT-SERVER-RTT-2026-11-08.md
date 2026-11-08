# Client/server step 1 — real-tunnel RTT, desktop + phone (2026-11-08)

Plan: [CLIENT-SERVER-PLAN-2026-11-05](../plans/CLIENT-SERVER-PLAN-2026-11-05.md), step 1 gate: record the real-tunnel RTT
distribution on home broadband (desktop and modern phone), compare with the 100 ms confirmed-action budget, report every breach.
Desktop and phone both measured 2026-11-08.

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

## Phone (S25, SM-S931U, Brave / Chrome 156, home Wi-Fi) — `deploy/lab/rtt-phone.cjs` over CDP from the Beelink
Measured inside one tab on the server origin (`/cdn-cgi/trace` hosts the page: the app 404's CSP correctly forbids page fetches),
sequential `fetch(no-store)` on the browser's warm connection. Tab opened and closed by the tool; phone tab count 24 before and after.
Taken ~08:05 CT.

| run | n | p50 | p95 | p99 | max | breaches > 100 ms |
|---|---:|---:|---:|---:|---:|---|
| phone app-warm | 300 | 64.7 | 90.1 | 111.3 | 161.0 | **8 (2.7%)**: 111, 111, 120, 130, 161 among them |
| phone edge-warm | 300 | 41.5 | 72.9 | 101.2 | 181.9 | 4 (1.3%): 101, 108, 117, 182 among them |
| phone first connection | 1 | 273 | | | | 1 — a single sample, not a distribution |

- The phone's Wi-Fi alone costs ~41 ms p50 to the edge (desktop: 13.6). Warm app requests **already breach 2.7% before any server work**;
  p95 90 ms leaves ~10 ms for the server at the 95th percentile.
- A true cold distribution on the phone needs closed sockets between samples (Chrome pools per origin); not measured. One first-connection sample: 273 ms.

## What it means (no success redefined)
- **Warm connection: the floor fits the budget.** p99 is 45 ms, which leaves about 55 ms of the 100 ms confirmed-action budget for server work and rendering.
  Tunnel + origin add about 19 ms over the edge (p50 32.4 vs 13.6).
- **Cold connection: the floor alone breaches.** 13.3% of fresh connections exceed 100 ms before any server work, so every
  confirmed action on a cold connection will be over budget once server time is added. This is a physical condition (DNS + TCP + TLS to the edge),
  not something to tune away by redefining the budget.
- Single edge-warm outlier (165 ms) shows broadband jitter exists. One sample in 499 is not a pattern, but long runs should expect it.

## Escalation for Christopher's ruling
Cold-connection breaches are certain under the current budget, and the phone breaches even warm. Options put to Christopher:
1. Keep connections warm. The plan's topic SSE stream already holds a connection open while a screen is up; confirmed actions then ride a warm connection.
   Measure the first action after app start / long idle separately and report it as its own number.
2. Accept that "first action after idle" is outside the 100 ms budget and report it separately (that would be a ruling, not a redefinition by us).

## Still open in step 1
- Contract lock: hosted `runNative` + command catalog, locked screens and CLI/MCP reaching the same service with identical outcomes/rejections.

## Ruling (Christopher, 2026-11-08)
Option 1 — **keep connections warm** (the live topic stream holds the connection open while a screen is up). The first action after app start
or long idle is measured and reported as its own number. The phone's warm-path breaches (Wi-Fi floor) are reported as measured, every time.
