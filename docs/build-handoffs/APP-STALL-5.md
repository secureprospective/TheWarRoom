# APP-STALL-5 — handoff (2026-11-02)

Branch `build/STALL-5` off `build/INTEGRATE-1102` @ 1dd4ad82. **Result: the per-reading ledger copy is gone,
but the screens still FAIL.** With 7b684d65 the readings no longer error out. They now run the full season
scoring, and that is too slow and too heavy: RSS reached 5.3 GB and CPU stays at about 750% when idle.

## Changed
- **Debt #2 (ledger copy):** `nativeRecords` now reads `sharedTransactionRecords`. Every reading of one revision
  shares the cached snapshot. The caller owns only the outer slice (`slices.Clone`), and nested data is read-only
  by contract. The old path also deep-copied the attached replay memo and projection on every reading.
  Write, preview, import and stat-feed paths still use `transactionRecords`, which deep-copies as before.
- **Debt #4 (joined results):** `shareReading` gives the caller that started a flight its value. Each joiner gets its
  own `deepClone`, so no caller can change what another caller reads.
- Tests (`shared_records_test.go`): TestSharedRecordsCallerCannotChangeAnother (rewrite, reverse and append on
  one result, then the next caller reads an identical ledger), TestSharedRecordsStayUnchangedByReadings
  (Standings, Results and Bracket leave the snapshot DeepEqual), TestJoinedReadingsCannotChangeEachOther.
- **Debt #3 (held-season key narrowing):** not done.
- **Item 3 (held parts totals):** not needed. No `pat_missed_or_blocked`, `xpa` or `rush_rec_yd` in any held archive
  (f2click copy: 2789 archives, after gunzip), and no "do not record it" in the desktop log. Scoring is untouched.

## Desktop (`make build`, copy of ~/f2click config+data at `~/scratch/stall5/prof`, DISPLAY=:5, 1920x1080, 10 min)
| t s | 25 | 55 | 115 | 175 | 235 | 295 | 355 | 415 | 475 | 535 | 595 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| CPU % | 250 | 608 | 659 | 699 | 760 | 586 | 670 | 668 | 674 | 596 | 753 |
| RSS MB | 1901 | 3052 | 3330 | 4243 | 4679 | 4585 | 4688 | 4680 | 4817 | 4806 | 5317 |

Idle, 20 s after arriving: HQ 759% / 5389 MB, Pulse Now 771% / 5148 MB, Pulse Race 756% / 5290 MB.

Goroutine dump at 10.5 min (`~/scratch/stall5-dump1.log`): 76 goroutines, 15 distinct flights, and waiters
blocked for more than a minute. The stacks are 22 `TargetWorkflow → commandcatalog.workflowReads` (odds,
draft, lineup) and 14 `NativeDraft → draftStandings → FinalStandingsReading → countSeason → franchiseWeekScore →
nativeFacts/PlayerScore` (season re-scored per reading). There are 11 `NativeLineup` stacks. No deepClone
appears.

## Per screen (`~/scratch/stall5-shots/`, r0–r9 + idle-*)
- HQ lineup/bench/this-week: "Not available yet". **FAIL**.
- Pulse Now matchups: "Reading the week…". **FAIL**.
- Pulse Race standings + playoff odds: "Reading the standings…/playoffs…". **FAIL**.
- No error text on any screen. Home, War Room and Control render.

## Tests
`go test -list` first. `store/history -race -p 2 -count=1`, 14 tests (shared records ×2, joined, record cache ×3,
native records, share reading ×3, shared standings, held season ×2, closed books ×2): **14 PASS, 0 FAIL**.
`make build` passes (CI=true, own node_modules). No full suite.

## Debt
1. **Blocker:** a full season score per reading. Command-palette workflow reads (`TargetWorkflow`) fan out odds,
   draft and lineup readings with distinct keys, so the flights cannot join them. Each one runs `countSeason`
   over every franchise week. The fix is a per-revision season-score cache, shared by standings, odds, final
   standings and draft order. Also check why workflow reads fire on screen visits at all.
2. Held-season key narrowing (STALL-3 #2) is still open.
3. The shared-snapshot contract is enforced by tests, not by types. A new reader that writes nested record data
   would corrupt other readers.
