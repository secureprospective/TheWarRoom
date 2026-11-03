# APP-STALL-8

Branch `build/STALL-8`, base `bc95f10e`. **Desktop gate: FAIL (idle CPU on Control/HQ/Pulse); RSS PASS; data PASS.**

## Changes
1. **Value sweep clone** (`appvalue`, `history`, `composition`): the preparation producer reads the shared,
   read-only per-revision snapshot (`Store.SharedCommissionerRecords`) instead of a deep clone of the 16.8k-record
   ledger. The producer also skips a scope while store revision (`PRAGMA data_version`) and params are unchanged,
   its pass is still in force, and less than 30 s has passed (`recheck`). Scope discovery is throttled the same way.
   UI reads compute their own key, so they show "preparing" rather than a stale pass.
   The `preparation_test` change-subtests now bump the revision with the records, as any real commit does.
2. **Stat-queue pricing** (`commissioner`):
   - `RulesMemo.fold` extends the held fold for a proposed approve/reject (`withProposal`), with no full re-fold per item.
   - `Replay` reuses the held replay cache for such a trial (`cachedTrialReplay`).
   - `ResultFlipCheck`'s directory, schedule, rules and per-week "before" results are read once per queue reading (`flipBase`).
   - **Leak fixed:** STALL-7's `RulesMemo.trials` map kept every proposal's whole-ledger copy alive for the reading.
     It caused 4.1 GB in-use and RSS of 5–6.5 GB after Control Room. Now only the latest trial is kept.
   - Equivalence: `TestTrialFoldMatchesFullFold` (DeepEqual vs `foldLedger`, held fold unchanged, 1 full fold),
     `TestTrialReplayMatchesFullReplay`, `TestStatQueuePricingMatchesStandalonePricing` (points + flip JSON identical to standalone pricing).
3. **HQ arrival spike**: a CPU profile showed it is the value sweep, standings `countSeason` (held-season build)
   and `AgeArc` (playerrecord `Book`/`Features`). The sweep part is fixed by (1). The rest is a one-time arrival load.
4. **Owner-only reads**: the 28 refusals came from `readNativeHQ` → `moves.read` (HQ and War Room header) and from
   `bids.read`/`waivers.read` (Home, War Room market) for a franchise the viewer does not own. They are now gated by
   `owner = people.yourFranchise === mine` at the client. The server is unchanged, so authorization is intact.
   Gate run: **0** refusals (was 28/32).
5. **`TestNativeLineupWire`**: `fieldSources` is intended (SCORE-3 field provenance). The fixture was regenerated
   (`TWR_WRITE_WIRE=1`); the diff is only added `fieldSources`.

## Desktop gate (DISPLAY=:5, MNF live, 4 laps, CPU 20–25 s after arrival, final tree)
| screen | lap1 | lap2 | lap3 | lap4 |
|---|---|---|---|---|
| hq | 384 | 175 | 165 | 138 |
| pulse-now | 342 | 156 | 157 | 195 |
| pulse-race | 0 | 207 | 187 | 152 |
| home | 11 | 10 | 11 | 11 |
| warroom | 0 | 17 | 16 | 224 |
| tradefloor | 0 | 0 | 10 | 0 |
| control | 166 | 182 | 142 | 148 |
RSS 1394–1813 MB (PASS). Lineup stacks 0. STALL-7 for comparison: hq 239–248, pulse-now 253–258, control 209–218.

## Remaining (why FAIL)
The Control Room's `stats.queue.read` is a detached flight that keeps pricing for minutes after the screen is left,
so its cost lands on whichever screen comes next. The profile now shows per-starter linear scans of the ledger in trial scoring:
`scoreChangesFrom` (~11%), `ReadingInstant` (~11%), and `cachedReplay` key checks. Next steps:
- index score changes per (player, season, week) per held revision;
- memo `ReadingInstant` per ledger;
- or cache priced items per (revision, item, evidence digest, live mark) so live updates re-price only changed items.

## Tests
- Targeted `go test -race -p 2`: commissioner 620 PASS; appvalue 16 PASS; history 512 PASS / 2 FAIL; composition 505 PASS / 3 FAIL
  (counts include subtests). The five failures (`TestNativeResultsStandingsWire`,
  `TestLedgerAuditAdoptsExistingHistoryWithoutChangingReceipts`, `TestFrozenDoorMetadata`,
  `TestNativeLeagueWireMatchesGolden`, `TestNativeScoreWireFixtureMatchesGo`) **fail identically on base bc95f10e**
  (stale goldens/schema hashes, migration v13 clash).
- Frontend: vitest (hqReading + shell) 50 files / 379 PASS, including the new owner-gate case. `tsc --noEmit` clean.
- `make build` OK.
