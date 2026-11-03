# APP-FEED-1

Base bc95f10e; branch build/FEED-1. Local only; no desktop launched or real profile read.

## Causes / fixes

1. Sleeper URL omitted `/v1`. Live curl: old
   `https://api.sleeper.com/stats/nfl/regular/2026/8` → 404, 206 B HTML;
   `https://api.sleeper.com/v1/stats/nfl/regular/2026/8` → 200, 485272 B,
   2,080 keyed stat rows, the shape already supported. Corrected embedded URL.
   Full captured-body test exposed a second blocker: negative yardage averages
   rejected the entire response. Allow signed averages, still reject negative counts.
   Captured body now runs through actual gap acquisition and conversion.
2. Stats worker fetched ESPN successfully but discarded scoreboard clocks. The live
   adapter only refreshed inside legacy schedule windows, so empty windows meant an
   empty board forever. Publish the validated worker board to Now's holder before
   identity/stat acquisition, serialized with live reads under LiveGameMu. Clone on
   publication, preserve same-week details, ignore older publications, reset on week
   rollover. EffectiveWindows fills missing windows from that checked board, retaining
   final detection and the six-hour stale-game bound. Pulse strip and field now share
   one live read; removed the hardcoded clocks placeholder and distinguish loading /
   unavailable from no game. No new provider.
3. Week 8 is current: live ESPN JSON explicitly says season 2026/type 2/week 8.
   The frontend gets its week from `week.read` (`schedule_owner.go` →
   leagueweek.LineupWeek), not standings. `NativeStandingsReading` explicitly counts
   through min(regularWeeks, published weeks): 13 is the season's regular-season
   scope, NOT a current-week clock. 4 + 204 = 208 = 13 × 16; the header's 0 + 16
   is week 8 only. No wiring defect found; retained the intentional different scopes,
   rather than truncating standings/open games and breaking odds. Live league DB
   counts were not independently reread (profile access prohibited).

## Flow / preservation

ESPN HTTP → RawScoreboard validation → parsed games → stats worker Board callback →
Now holder (LiveGameMu + clone) → live adapter windows / LiveReading → shared Pulse
read → strip and field. Failed board acquisition never publishes; identity failure
cannot discard clocks. Follow/summary routes and existing polling cadence remain.
Sleeper HTTPS → existing RawSleeperWeek / RawLiveSleeperStats validation → raw stat
lines → existing ESPN-first gap merge → held live evidence. No fantasy point totals,
MFL score fallback, source ladder or commissioner override changes. DropTotals stays.

## Evidence

- Real ESPN curl → 200, 234792 B; 14 events, event 401873036 in progress Q2 15:00.
- Exact public captures + decompressed SHA256s: `internal/app/now/testdata/SOURCE.md`.
- `go test -list` first; targeted `go test -race -p 2`:
  now **69 PASS**, livegame **33 PASS**, source **242 PASS / 4 opt-in SKIP**,
  composition native-results IPC **1 PASS**. Counts include subtests.
- Captured ESPN test exercises stats poll → publication → live refresh with NO legacy
  windows, including identity outage; sees Active, 14 games and exact Q2 clock.
- Window tests prove no duplicate windows, final stop, stale six-hour bound.
- Publication tests prove copy isolation, detail retention, stale rejection and rollover.
- Frontend Pulse live/reads: **14 PASS**; `pnpm tsc --noEmit` PASS.
- Affected-package golangci-lint: **0 issues**.
- `make build VERSION=dev COMMIT=FEED-1 BUILD_DATE=FEED-1`: PASS on final production
  tree. Wails emits inherited generic Sourced binding warnings; build exits 0.

## Debt / next verification

No GUI per brief. Reviewer should confirm Pulse's strip and field both show the live
clock after startup and following a game still loads plays; no live screenshot claimed.
Standings footer wording remains potentially confusing (season scope, not current week).
No private league profile or config touched. No git remote operations.
