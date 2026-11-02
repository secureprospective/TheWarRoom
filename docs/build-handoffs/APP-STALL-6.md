# APP-STALL-6

Branch `build/STALL-6`, base `a37ad596`. Continued the authorized uncommitted work. **Desktop gate FAIL.**

## Changes and flow

- `commissioner.franchiseWeekScore` delegates to an optional memo; the original scorer remains the uncached path. Store-owned `nativeSources` shares successful franchise-week scores across standings, odds, final standings, draft order, results and bracket.
- Key: league, season, durable ledger entry sequence, scoring rules, held archive/identity revision, player names/positions, franchise and lineup rules. Live evidence adds a per-week generation. Scores use the full ledger, not viewer-filtered records; no viewer-dependent score input was found.
- Eight LRU revisions retained. Concurrent cold requests join one calculation; waiters can cancel. Errors are not retained. Returned scores are deep copies. Hypothetical ledgers, replacement evidence/lineups without the ledger contract, and database-free transfer scoring bypass caching.
- Live captures, persisted chronology changes and restore invalidate live generations. A generation changed during calculation is not cached. Superseded cached live generations are removed.
- Player facts are read once per reading, shared by its key and scoring positions, instead of rebuilding the directory per starter.
- Held-season key now covers the season's registered weekly/snap/injury URLs, plays and historical depth. Bye and secondary-source blocker URLs remain conservatively global. Identity changes still invalidate. Unrelated archives no longer reconstruct absence proofs.

Flow: authorized reading → immutable ledger/facts and held revision → shared week score → caller-owned result. The store owns cache lifetime; commissioner code still owns scoring. Failed key acquisition falls back to scoring, not an empty reading.

## Workflow finding: the palette premise is false

`frontend/src/app/commands/CommandBar.tsx` → `search.ts` ranks an in-memory command index. It does not prefetch workflow readings. There is no palette workflow request to debounce or cancel.

Actual route: `AppLeagueMount.tsx` passes `localWorkflow` to Home/HQ/Pulse. `homeReads.ts`, `pulseReads.ts`, `hqReading.ts` and `hostedRoom.ts` request their screen data through that port. `localWorkflow.ts` calls `TargetWorkflow`, explicitly documented as the local counterpart of hosted workflow transport, not command-palette targeting. Disabling these calls on screen visits would remove the required data. No palette-only restriction or per-screen hack added. Screen-read cancellation/debouncing is not delivered under a false palette diagnosis.

## Tests and build

- `go test -list` first, targeted `go test -race -p 2`; no full suite.
- History cache/shared-record/live selection: **25 PASS, 0 FAIL**.
- Commissioner starter failures, standings table, golden odds and playoff odds: **16 PASS including subtests, 0 FAIL**.
- Cold/warm and uncached 32-team held-file fixture readings are byte-identical: results weeks 1–17, standings, odds, bracket, final standings and draft. Ledger/archive/live updates preserve cached/uncached equivalence.
- Caller mutation covers score pointers, starters and lineup verdict. Removing the hit deep copy makes this test fail with “one caller changed the score another read”; restored afterward. Twelve concurrent requests observe one calculation. Eviction, cancellation, trial bypass, facts reuse and database-free scoring tested.
- Narrowed held-key test preserves uncached bytes after an unrelated archive. Every weekly/plays/snap/injury/depth/bye/blocker dependency moves the key; existing identity and weekly-update equivalence tests pass.
- Targeted golangci-lint: **0 issues**. Diff whitespace check passes. Real `make build` passes with own node_modules, CI=true, stamp `stall6`. The `dev` stamp selects different databases and was not used for the selected-league gate.
- Inherited bloat failure: 29 tiny files versus baseline 24. HEAD and candidate have the identical 29-file set; baseline not weakened.

## Final desktop run

One copied profile only; originals untouched. DISPLAY=:5, 1920×1080, GOMEMLIMIT=1500MiB/GOMAXPROCS=8. Repeatedly visited Home, War Room, HQ, Trade Floor, Pulse Now/Race and Control for ten minutes, then measured idle after 20 seconds on each required screen. CPU is process CPU over five seconds; RSS is MiB (script labels MB).

| elapsed s | CPU % | RSS MiB |
|---:|---:|---:|
| 25 | 391 | 1387 |
| 55 | 670 | 1753 |
| 85 | 292 | 1680 |
| 115 | 724 | 1741 |
| 145 | 775 | 1782 |
| 175 | 771 | 1977 |
| 205 | 770 | 1863 |
| 235 | 780 | 1810 |
| 265 | 779 | 1840 |
| 295 | 796 | 1831 |
| 325 | 790 | 1847 |
| 355 | 744 | 1858 |
| 385 | 789 | 1842 |
| 415 | 745 | 1852 |
| 445 | 746 | 1857 |
| 475 | 743 | 1957 |
| 505 | 754 | 1895 |
| 535 | 711 | 1908 |
| 565 | 755 | 1992 |
| 595 | 763 | 2112 |

| idle screen | CPU % | RSS MiB | required data within 3 min |
|---|---:|---:|---|
| HQ | 769 | 2065 | FAIL: lineup, bench and this-week still unavailable at end |
| Pulse Now | 746 | 2039 | FAIL: matchups eventually contain data, after deadline |
| Pulse Race | 766 | 1960 | FAIL: standings and odds eventually contain data, after deadline |

RSS ceiling **PASS**: all 20 samples below 2.5 GB, sampled peak 2,112 MiB. Idle CPU **FAIL** on all three. Other screens render; that is not a claim every region's data passed.

Final dump: **0 countSeason stacks; 20 NativeLineupReading stacks**. Runnable work repeatedly calls `NativeBase.playersOf → ScopedDecisionExceptions → decisionException → Replay → cachedReplay`, cloning the active-entry map and hashing replay keys. This is the remaining observed blocker, not palette prefetch.

Evidence: `~/scratch/stall6/final-samples.txt`, `dump-final.log`, `history-tests.json`, `commissioner-final-tests.json`, `build.log`, `mutant.log`; screenshots `~/scratch/stall6-shots/f0-*` through `f8-*` and `final-idle-*`. Before-three-minute shots still lack the required data. Final Now/Race contain data; HQ does not.

ESPN held 842 players; Sleeper returned 404. No NFL game was in progress on the final Now screen; no live MNF-drive test claimed.

Earlier attempts: cache-only run stayed around 1.7–1.9 GB but failed CPU/readiness. An uncapped facts-reuse run exceeded 4 GB and was aborted; its dump identified held-season absence-proof reconstruction and lineup legality work. A restart encountered the previous process's still-held lock; no lock file was removed. The final run started after its holder exited.

Desktop stopped with the final dump. The execution tool blocked the requested disposable-profile cleanup, with no confirmation UI. Profile remains at `~/scratch/stall6/prof`; operator cleanup is still needed.

## Debt

1. Fold player-independent legality/decision-exception inputs once per ledger/reading, preserving results; join/cancel actual screen reads at their shared source. The season-score cache does not fix this remaining path.
2. The shared workflow transport is not cancellable on screen unmount. It is not command-palette prefetch.
3. Ledger nested contents remain read-only by contract, not by types. Cache safety assumes durable entries are immutable; hypothetical changes append entries.
4. External review remains required before integration. No remote git operations, scoring/valuation formula changes, source-kind branches, MFL roster moves or full-suite run.

This is an optimization, not a code-size reduction. Before the final database-free regression test, affected handwritten production was 10 files, 2,000 → 2,397 physical lines; tests were two files, 200 → 578 lines. Generated files/docs excluded.
