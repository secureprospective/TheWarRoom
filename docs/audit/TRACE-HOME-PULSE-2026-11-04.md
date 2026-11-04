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
