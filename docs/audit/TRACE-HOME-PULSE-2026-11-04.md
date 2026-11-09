# Reader trace — Home / HQ / Pulse standings (2026-11-04)

Source: HQ-1 c196a523, soak run-0952 `func.txt`, frames in `soak/trace-home-hq-pulse-L1-L40.png`
(top-left Home loop 1 · top-right Home loop 40 · bottom-left HQ loop 40 · bottom-right Pulse Race loop 40).

## Standings: ONE live backend path (good)
Home (`shell/homeReads.ts` → `readNativePulse` in `shell/pulseReads.ts:74`), HQ (`commissioner/hqReading.ts:427`) and Pulse all send
`standings.read` → `commandcatalog/workflow_commands.go:83` → `leagueservice.TargetNativeStandings` (75% ran) →
`store/history.NativeStandings` (83%) / `readNativeStandings` (89%).

## Parallel standings code that never ran (cut candidates, Kind B)
- `app/commissioner/native_legality_api.go:85 TargetNativeStandings`: 0%. A second API wrapper for the same read as leagueservice.
- `app/season/view.go:192 Standings`, `:351 StandingsOrder`: 0%.
- `app/season/points_reading.go` (whole file, 0%), `app/season/playoffs_reading.go` (whole file, 0%).
- `app/now/season_view.go:22 newRaceView`: 0%. `app/race/api.go` TargetRace/TargetRaceRefresh/Build: 0%. But `app/race/held.go` + `refresh.go`
  ran at 70–100%, and `app/core/host_commissioner.go:109 NativeStandings` ran at 100%, so a third standings holder lives in race/core. Not yet traced which screen reads it.

## F3 revised: not two backends, it's Home's loading
- Loop 1 Home: 2-0, seed 1, OFFSEASON, while Pulse later shows the truth (Bills 7-1).
- **Loop 40 (about 2.5 h in) Home is broken:** stuck on "OFFSEASON · Reading the league…", record "—", and the franchise card shows the raw id
  **"franchise.01"** instead of Buffalo Bills. Every tile is "Reading…".
- **Loop 40 HQ:** "Not available yet", with the page empty, rendered inside the shared rail. In the probe (08:36) HQ ran in its own 222 px shell (F1).
  So HQ has two render paths and the late one shows nothing.
- **Loop 40 Pulse Race: correct** (Bills 7-1, seeds, playoff odds). Footer says "Standings through week 13: 128 games have a result, 80 do not yet"
  while the header says week 9. Check that the "through week" wording is right.

## Home loading: why it shows wrong data or stays stuck (code read, HQ-1)
- **H1 — "OFFSEASON" is a fallback, not a reading.** `shell/HomeBrief.tsx`: `phase = g || b.results?.bye === mine ? 'in' : 'off'`.
  Until Home finds the owner's game it labels the league OFFSEASON. Any slow or failed read shows "offseason" in week 9.
- **H2 — a stale success is never refreshed.** Home retries only when `transientErrors(b)` is true, and subscribes to season changes
  (`subscribeSeason`) only while retrying. A read that *succeeds* while history is still replaying (2-0 through an early week) stays
  for the life of the mount.
- **H3 — reads are serial.** `homeReads.ts:62` awaits the whole `readNativePulse` (directory → week → standings/bracket, `pulseReads.ts`)
  before the other tiles start. One slow call blocks every tile ("Reading…" everywhere, franchise name falls back to the raw id).
- Backend evidence that calls are slow: run-0952 `app.log` 09:56:40, a dozen `context deadline exceeded` in history/accounts reads and writes
  while the launch refresh ran; 10:04:30 `standings.read` and `odds.read` failed with `context deadline exceeded`.

## Stall timing (run-0952)
Stalled loops (≤11 of 33 distinct): 27 (11:40), 29 (11:48), 34 (12:09), 36 (12:18), 41–43 (12:39–12:48).
The hourly launch refreshes (10:56, 11:56, 12:57) don't line up, and the backend log is silent during the stalls.
Diagnostic run-1332 (DIAG=1): per-process CPU/RSS (Go vs WebKit) every 5 s; at the first stall, thread snapshot + Go goroutine dump.

## F11 — The stall is the client (WebKit), triggered by Home (DIAG run-1332, 13:33–13:49 CT)
- Loop 1–3 responsive (33, 32, 27 of 33). Startup took 48 s with no migration, so the morning's early lag was startup DB contention, not the screens.
- Loop 4: the screen froze right after `home` → `home-brief` (13:45:20) and stayed frozen through Home, War Room and Trade Floor,
  catching up only sporadically after that. 7 of 33 distinct.
- Snapshot at 13:49:19 (`stall-threads.txt`): **WebKitWebProcess 92% CPU, 1.3 GB RES, 6:42 CPU time in 16 min. Go process 0% CPU, all threads sleeping.**
- WebKit memory jumps when Home mounts: 431 MB → 993 MB at 13:45 (loop 4 Home), and 599 → 1,195 MB at 13:37 (loop 2 Home).
- WebKit burns 24–56% CPU **all the time**, even when responsive. In a PWA on a phone that's battery drain and heat.
- Conclusion: the freeze is client-side rendering/JS, mainly Home. Go and the database were idle at the freeze.
  Not yet known: which Home tile or effect. Next: profile Home's render in a browser.

## F12 — Home's Transactions tile pulls and renders the WHOLE league ledger (leading suspect for F11)
- `shell/homeReads.ts:112` sends `commish.ledger.read` with `{ league }` only: no limit, no paging. The full ledger crosses the bridge on every
  Home mount (in a PWA that means across the network).
- `shell/homeTiles.tsx:125` renders `entries.slice().reverse().map(...)`: **every** entry becomes a `HomeAct` row. The tile footer read
  "+16,798 more in LEDGER" (soak frames), so about 16.8k DOM rows get built to show a few.
- The same reverse-the-whole-ledger logic also sits in `HomeBrief.tsx:170` (the card drawer): two copies.
- Matches F11: WebKit memory +560 MB on each Home mount, then the main thread pegged.

## F13 — Home's ticker animates forever
- `shell/home.css:323` `.hb .tkr{width:max-content; animation:tk 70s linear infinite}`, the "TOP 5 · WK n" strip. The content is duplicated
  (`HomeBrief.tsx:230` renders `tickerHtml` twice) so it can loop. It repaints all the time, even when idle → the baseline 25–55% WebKit CPU.
  (The Xvfb test box renders in software (llvmpipe), which magnifies this. A phone would still drain battery.)
- Other always-on loops: `draftroom/useDraftReading.ts:15` 1 s clock, `research/state.ts:33` polling, `pulse.css:353` blink, `style.css:320` skeleton pulse.

Not yet proven. The proof would be a throwaway build: cap the tile at the rows it shows, stop the ticker, rerun DIAG and compare WebKit CPU/RSS and stalls.

## Result: experiment run-1104-1415 (throwaway patch, reverted after)
Patch: Transactions tile draws the last 12 rows only (homeTiles.tsx:125); ticker animation off (home.css:323). The full ledger is still fetched.
| | Baseline run-1332 (unpatched) | run-1415 (patched) |
|---|---|---|
| Distinct frames per loop | 33, 32, 27, then STALL 7/33 (loop 4) | 32, 31, 24, 25, 25, 26, 27, 27 (8 loops, no stall) |
| WebKit CPU | 24–56% all the time, 92% at the stall | 7–13% |
| WebKit RSS | 431→993 MB on a Home mount, 1.3 GB at the stall | 209–333 MB, flat |
| Run status | 5 (stall) | 0 (30 min clean) |
**F12 + F13 PROVEN** as the cause of the client stall and the WebKit CPU/memory. Rendering the full ledger and the endless ticker did it.
The fetch alone (still the full ledger) costs little on the client. The fix still pages the read, because PWA owners pay for the transfer.
Loops 3+ at 24–27 distinct are a separate smaller gap (not a stall); not chased yet.

## Real fix verified: run-1104-1507 (commit 23ea625a applied to the soak build, then reverted)
The fix: a 12-row ledger page via commish.ledger.filtered.read last=12, one reverse, and the ticker plays once.
Result: status 0, 9 loops, distinct 33, 31, 26, 26, 26, 26, 27, 27, 27, no stall.
WebKit averaged 8% CPU and peaked at 357 MB RSS. The baseline was 24–56% CPU, 1.3 GB, and a stall in loop 4. App usable after 53 s.
