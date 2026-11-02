# APP-STALL-3 — handoff (2026-11-02)

Branch `build/STALL-3` off `build/INTEGRATE-1102` @ 851cac6e. Commits: 6667891b, aed592bd, 1154b06c.
**Result: code is in and proven score-identical, but the desktop acceptance criteria FAIL.** The held-season
rebuild was one real cost. It was not what keeps HQ and Pulse empty.

## Changed (all in `internal/store/history`, none of Sol's SCORE-1 files touched)
1. `held_season_cache.go`: `heldSeason` builds once per (season, key), single-flight per season, keeps only the
   latest entry per season. Key = `count(*), max(fetch_id)` over `fetch_log` status 200 (the archive is
   append-only, so this is a superset of every body the season reads: weekly, plays, snaps, injuries, depth, byes,
   espn/sleeper blockers) + sha256 of the gsis/pfr directory. Live lines, stat overrides, rules and positions are
   applied per starter after the held season is read, so they are not inputs to it. The key is read before the
   build, so a mid-build change rebuilds on the next read. Errors are not cached. The old body is `readHeldSeason`.
   The indexed copy for many-player readings is built once with it.
2. The reading's gsis lookups (`nativeSources.SourceID`) come from the directory map read for the key, not
   one pooled query per starter.
3. Debt #2: `wholeStream` holds `eventsMu` only around the map. A reader stores its copy only if the entry
   did not move meanwhile.

## Equivalence
`TestHeldSeasonCacheMatchesUncachedReadings`: on the 32-team bracket league, scored from real archived weekly
files (no supplied evidence), JSON of `NativeResults` weeks 1–3 + `NativeStandings` is byte-identical between
cache off (per-call build, per-starter SourceID), cold cache and warm cache. Checked at three points: first
revision, after a new weekly revision (readings change), and after relinking a gsis id (forces a rebuild).
`TestHeldSeasonCacheBuildsOncePerRevision`: 16 concurrent standings readings → 1 build.

## Desktop (fresh copy `~/scratch/stall3/prof`, `make build`, DISPLAY=:5, 1920x1080, all screens every ~40 s)
Final build (1154b06c), sampled every 30 s: CPU 200 760 759 766 704 637 696 %; RSS 1504 2757 3622 4273 5477 5907
6370 MB. I stopped the run at 3.5 min because it was clearly failing. The run before (6667891b) was the same: 740% and
6.3 GB at 5.5 min. HQ lineup/bench/this-week "Not available yet". Pulse Now "Reading the week…". **FAIL.**

## Cause still standing (goroutine dump at 6 min, `~/scratch/stall3/dump3.log`)
Duplicate concurrent readings from the frontend's workflow polls: 13 `NativeStandings`, 7 `NativeResults`,
7 `FinalStandings` (rookie draft order), 6 `NativeOdds`, 5 `NativeBracket`, 3 `StatQueue→ResultFlipCheck`. All ran at
once, each re-scoring every week × 32 franchises × starters, most of them for 5+ minutes. Each takes longer than the
poll interval, so they stack up. Memory and CPU scale with the stack, not with any one reading. A build with
`go build` + pprof (non-wails frontend) never stacked up (≈900 MB, 23 goroutines), which also points at polling.

## Debt / decision needed
1. **Coalesce or memoize readings** (needs your call). Option A: single-flight identical in-flight readings keyed
   on (kind, league, season, week, ledger revision, held-season key, live book stamp, recipient view). Recipient
   and alias live in unexported ctx values in `commissioner`, so that key must come from commissioner. A joined
   caller sees the flight's clock instant. Option B: memo week scores across readings by the same revision
   key (touches `commissioner.weekResults`; `now` decides final vs provisional). Option C: frontend/host should not
   start a poll while its previous read is in flight.
2. The archive mark moves on every fetch (~1/min, mostly MFL/depth). Each move rebuilds the season once
   (single-flight). It could narrow to the season's URLs if that shows up.
3. Carried: STALL-2 debts 3–5.

## Tests
`store/history` `-race -p 2`, whole package on the final tree: 284 PASS, 4 SKIP, 0 FAIL (288 listed). `make build`
passes. Shots `~/scratch/stall3-shots/` (r* = real-build runs; p*/q* = pprof diagnostic build, not evidence).
