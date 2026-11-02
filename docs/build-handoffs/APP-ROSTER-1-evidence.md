# APP-ROSTER-1 — baseline and blocker

**Incomplete. Item 1 investigated; items 2–3 not implemented.** No production code changed.
Local evidence only; no git remote contact, MFL requests or real roster moves.

## Three-line finding

Adds enter the commissioner ledger as bid/waiver awards or draft picks; drops are `roster.cut`; trades use the offer/accept/review commands and recorded outcomes.
IR/taxi/active changes use `roster.reserve`; commissioner roster corrections use `correct.roster`; replay projects these over the single recorded starting snapshot.
No roster-as-of-week support exists: lineup append reads the current projected roster, and native asset projection does not filter roster moves by the requested instant.

## Source-traced gap

- `internal/commissioner/lineup_week.go`: `CheckLineupAppend` calls `ReadNativeBase` at `now`, then `checkLineupRoster` before either owner or correction handling. A correction bypasses locks/legality refusal, **not** roster membership.
- `internal/commissioner/native_reading.go`: `nativeAssetsState` projects all active entries before using its optional `at` for correction readings. Passing a historical instant is not a historical roster read.
- `internal/commissioner/native_projector.go`: `projectNativeOverrides` walks revision order, with no roster effective-time cutoff. Merely passing a week kickoff to `NativeAssetsReading` will not fix this.
- `internal/commissioner/waiver_command.go`: `validateWaiverPlacement` requires `effective=now` for cuts and reserve changes. `trade_command.go` and `bid_command.go` likewise require current placement. The generic SQL event envelope has `effective_at`, but that is not sufficient support in the ordinary commands/projector.
- `internal/store/history/commissioner.go`: `appendAdmittedCommissioner` records real receipt time, validates ordinary commands, stores their outcomes and refreshes the recovery snapshot. Writing raw SQL or backdating the store clock would evade the ordinary admission/audit contract.
- `internal/commissioner/correction_projector.go`: a roster add needs a previously recorded contract; `correct.roster` cannot invent the contracts of players dropped before import. The transactions export identifies moves but supplies no historical contracts.

The brief's “engines done — wire only” assumption is not true for historical roster projection/admission. No safe existing history-import port was found in the traced route. This is an implementation/design gap, not a claim that the requested behavior is impossible.

**Escalation for ClaudeBox review:** define the generic effective-dated roster history/admission contract, including the initial historical baseline and what constitutes evidence of a dropped player's historical contract. Then translate validated transaction rows at the importer boundary and use that same generic projection for ordinary past-week lineups. No source-specific reader or lineup bypass is acceptable. No coordination call was made from this worktree.

## Copied-profile baseline (exercised)

SQLite backup API from the requested scratch profile into `.tmp/roster1/profile`, both databases; no live config profile opened. Existing archived bodies were decompressed and parsed through the existing handwritten validators. No network required. The copied history initially held two entries for the followed league (create and today's seed); two unrelated creates were left alone.

Validated season transaction feed: **895 rows**: 129 TRADE, 533 FREE_AGENT, 66 IR, 116 LOAD_ROSTERS, 51 TAXI. Transaction times run from February 23 through November 1, 2026. No transaction was imported.

The ordinary `mflsource.BackfillLineups` → `SyncLineups` → `MigrationLedger.Execute` path was exercised on the copy. The held player directory supplied names/team facts. This was a direct importer measurement, **not a launch/sync proof**; no kickoff schedule was supplied, so it proves correction admission/roster refusal, not owner-lock behavior.

- Initial race-enabled measurement hit its 600-second deadline after committing 122 accepted franchise-weeks; it did not finish. No race error was printed, but this run is **not a race PASS**.
- Non-race retry resumed from 124 total entries, recorded the remaining 26 accepted franchise-weeks and refused 76.
- Second completed pass recorded 0 and refused 76. Total accepted: **148/224**, unchanged behavior.
- Bills weeks **1, 2, 3, 4, 5, 6, 7 all recorded** through the ordinary path already; this does not establish the missing 76 other franchise-weeks were fixed.

```text
Validated archived transactions: 895
League season=2026 initial_records=124
Ordinary sync pass=1 recorded=26 refused=76
Ordinary sync pass=2 recorded=0 refused=76
Bills week=1 recorded=true
Bills week=2 recorded=true
Bills week=3 recorded=true
Bills week=4 recorded=true
Bills week=5 recorded=true
Bills week=6 recorded=true
Bills week=7 recorded=true
```

### Existing seed versus latest archived reference roster

`APP-ROSTER-1-baseline.tsv` compares membership/status for all 32 franchises, normalizing MFL `INJURED_RESERVE` to domain `IR` only. **28 equal, 4 mismatched.** This is **not a transaction replay-equality proof**, nor a live current-roster fetch.

- 0003: reference adds 10976.
- 0015: reference adds 14467.
- 0018: reference adds 15263.
- 0025: reference removes 14467.

The seed's roster time precedes the last archived transactions. No mismatch was forced away and no baseline was rewritten. Reconciliation must respect the seed observation time rather than treating the latest transaction export and seed as contemporaneous.

## Checks

Targeted `go test -race -json` over commissioner, transaction ingestion, migration and history, selecting the ordinary lineup/permission/correction/replay tests and transaction parser tests: **24 test/subtest PASS, 0 FAIL**. Two migration packages had no matching tests; their package PASS is not counted as additional test evidence. Command selection:

```text
-run 'Test(LineupCommandBoundary|RecordedLineup|OnlyTheSeatedOwner|CorrectionProjection|CorrectionsDecode|MarketClosedBooksRoundTrip|RealTransactions|SyntheticTransactions|MigrationLineupSync|LineupSetMustBeLegal|LineupBelongs|ImportedCutPricing)'
```

Plain `make build`: **PASS**, binary packaged. Wails printed generic-binding discovery warnings; exit status was zero. Binary was not launched. Raw local logs, disposable proof harness and copy remain under `.tmp/roster1/` (ignored, potentially private; not committed).

No implementation tests for items 2–3 exist because those items were not implemented. No production, handwritten-test or parallel-owned file changed. Independent review and GUI verification have not occurred.

## Debt left

All requested history importing and roster-at-lock behavior remain open. Refused count remains **76 → 76**, not zero. End-of-history replay equality is **unverified**. Required next work is the generic historical roster contract above, its tests and importer wiring; this evidence commit is not feature delivery.
