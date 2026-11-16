# Alpha 0.10 — stress test (line 14 of RELEASE-0.10.md)
**Ruled by Christopher 2026-11-16:** load is the gate. Speed is measured and written down as the baseline; it
improves as the project goes on and does not hold up 0.10.

## Setup
- A throwaway container on the test-server VM, sized like the real one: 1 GiB memory, 2 cores, no swap.
- The 0.10 server bundle (`twr-server version` says 0.10.0), a 32-franchise league, backups switched on.
- 32 owners, 80 devices (the tool's built-in spread: 3 devices for 16 owners, 2 for the other 16), driven by `deploy/lab/stress.py`.
- The run is `deploy/lab/stress-schedule.sh`: steady load, burst, steady, with 15 forced kill / container-stop /
  service-restart cycles mixed in. About 3 hours. `stress-sample.sh` records memory, disk and CPU every 10 seconds.

## The gate — every line must pass
1. No crash and no out-of-memory kill outside the 15 scripted ones.
2. No 502 or other gateway error while the server is up.
3. Every confirmed action lands exactly once (`stress.py reconcile` reports zero lost, zero doubled).
4. After each of the 15 forced stops the server answers again within 2 minutes, by itself.
5. The database passes its integrity check after every restart.
6. Backups keep completing during load; changes are never paused for lack of a backup.
7. Live scores are never older than two poll intervals while a game window is being replayed or is live.
8. Memory does not climb steadily across the run (end-of-run resting memory within 10% of the first steady phase).

## Measured and recorded, not a gate
- Screen-open and confirmed-click times at 8, 32 and 64 devices: p50, p95, p99 and worst.
  The standing budget is 200 ms screen / 100 ms click. Any miss is listed as named work for a later release.
- Peak memory, peak CPU, database and log growth.

Results go in `docs/roadmap/RELEASE-0.10-STRESS-RESULT.md`: one pass/fail per gate line, then the speed table.

## History
The 2026-11-08 run on an older build passed durability (15 cycles, 72 of 72 actions exactly once) and failed speed
(about 1 s screens at 8 devices; saturation and 502s at 32 or more). Nothing has been measured on 0.10 yet.
