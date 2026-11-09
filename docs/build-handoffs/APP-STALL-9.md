# APP-STALL-9

Branch `build/STALL-9`, base `930002e6`. **Desktop gate: FAIL on Control Room only (≈300% while the screen is open);
every other screen < 100% from lap 3, Pulse Now 141% in lap 2; RSS PASS; data PASS.**

## Changes
1. **Stat-queue read is withdrawn when its caller goes away.** Wails bindings carry no request context, so the read ran
   under the 120 s server timeout and the Control Room started a new one on every ledger revision.
   - `TargetStatQueue(league, season, request)`: a non-empty client-chosen request name (≤ 64 letters/digits/dashes)
     registers the read's cancel; new binding `TargetCancelRead(request)` withdraws it (`leagueservice/read_cancel.go`).
     A withdrawal that arrives before its read starts is kept, so that read starts cancelled. Hosted keys are scoped
     to the principal. The single-flight (`shareReading`) already stops a joined reading only when no caller waits.
   - `StatQueueReading` checks `ctx` between items.
   - Frontend: `readRoom(league, port, signal)` names the read; `ControlRoom` aborts on unmount and when a newer revision
     re-reads. The act tray passes `''` (unchanged behaviour).
   - Measured: Home 20 s after leaving Control Room went from 263% to 0%.
2. **Per-ledger score index and reading instant** (`commissioner/score_index.go`). `ledgerFold` now holds the score
   changes by (player, franchise) in ledger order and the ledger's latest `RecordedAt`; `withProposal` extends both in O(1).
   `nativePlayerScore` reads from them instead of `scoreChangesFrom` / `ReadingInstant` whole-ledger scans.
   Equivalence: `TestScoreIndexMatchesLedgerScan` (DeepEqual vs scan, all owners/weeks, active/reversed, decode error;
   instant vs `ReadingInstant` at several clocks incl. zero).
3. **Top remaining cause (re-profile): the result-flip check re-scored every franchise's week on each trial ledger.**
   For a statistics decision only the franchises starting the decided player are re-scored (`flipBase.rescored`); the
   rest keep their score, and `compareWeeks` reads only games with a touched side. The price and the flip share one
   trial copy per item. Equivalence: `TestRescoredFlipMatchesFullWeek` (JSON-identical to full re-scoring; flip, unknown,
   bye and unstarted cases; a mutation that skips one touched franchise fails it).

## Desktop gate (DISPLAY=:5, 4 laps, CPU 20–25 s after arrival, final tree, no profiling hook)
| screen | lap1 | lap2 | lap3 | lap4 |
|---|---|---|---|---|
| hq | 410 | 16 | 36 | 0 |
| pulse-now | 180 | 141 | 0 | 0 |
| pulse-race | 311 | 0 | 0 | 0 |
| home | 10 | 7 | 6 | 11 |
| warroom | 0 | 0 | 0 | 24 |
| tradefloor | 0 | 0 | 0 | 26 |
| control | 299 | 317 | 339 | 315 |
RSS 1474–1710 MB. Lineup stacks 0, owner refusals 0, StatQueueReading stacks at exit 1 (Control Room was the last screen).

## Remaining (why FAIL)
- Control Room prices the whole queue while open, again on every ledger revision. Remaining per-item cost: `scoreTrial`
  copies the 16.8k-record ledger per item (GC), and trial `Replay` (`cachedReplay` O(n) key checks + `Active` clone) in
  `RecordedLineupFor`, `FranchiseScoreChanges`, `LineupSyncPending` for the touched franchises. Next: cache priced items
  per (held revision, item, evidence digest) across readings, or price items lazily (visible rows only).
- `TestEveryCommandDoorRoleParity` now fails: the new `TargetCancelRead` door is not in the frozen
  `internal/commandcatalog/testdata/doors-before.json` (293 rows). Sol is regenerating that fixture in `.worktrees/fix`
  (`TestFrozenDoorMetadata`), so it was not edited here; add the row (and 293→294) when merging.

## Tests (`go test -race -p 2`, counts include subtests)
commissioner 622 PASS; leagueservice 176 PASS; commandcatalog 44 PASS; history 512 PASS / 2 FAIL; composition 511 PASS /
4 FAIL; app/commissioner 1971 PASS (`TestNativeCatalogPublication` then fixed, re-run PASS). Known base failures:
`TestNativeResultsStandingsWire`, `TestLedgerAuditAdoptsExistingHistoryWithoutChangingReceipts`, `TestFrozenDoorMetadata`,
`TestNativeLeagueWireMatchesGolden`, `TestNativeScoreWireFixtureMatchesGo`. New: `TestEveryCommandDoorRoleParity` (above).
Frontend: vitest control+commissioner 56 files / 562 PASS; `tsc --noEmit` clean; `nativegen -check` fresh. `make build` OK.
