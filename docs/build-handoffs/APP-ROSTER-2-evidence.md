# APP-ROSTER-2 — partial receipt-history prerequisite

**Incomplete. This is not effective-dated roster admission or transaction replay.**

## Implemented slice

`commissioner.NativeAssetsAsOf` reads the existing receipt-dated ledger at an explicit
instant at or after the seed's `RosterAt`. It retains create/seed evidence even when
received later than its observation; excludes subsequent entries before replay;
preserves revision order for equal timestamps; and returns the source revision.
Compensation is replayed over the selected history, not the final active-entry map.
An instant before the seed observation is refused rather than returning today's roster.
Zero instants and missing ledgers are refused. Existing reading/admission callers are
unchanged; this new API has no production caller yet.

This is deliberately a bounded prerequisite, **not** the brief's effective-order
projector. The existing `NativeAssetsReading(..., at)` still uses `at` for correction
readings, not roster filtering. No timestamped roster command has been enabled.

### Responsibility and data-flow trace

| Boundary | Contract / owner / effects |
| --- | --- |
| `NativeAssetsAsOf` | Already decoded records + instant → copied selection; no writes or external I/O. |
| `seedNativeAssets` | Validate and clone observation evidence; refuses malformed/missing history. |
| `NativeAssetsReading` → `projectNativeOverrides` | Replay selected receipt history with existing ordinary projectors and compensation checks. Errors retain context. |
| Existing admission, lineup and migration callers | Not rewired; their behavior and limitations remain unchanged. |

Tests exercise inclusive release/reversal instants, before/after transitions, a late
seed receipt, cached compensation replay, current-reading equality, ledger-body
immutability and refusal without a guessed snapshot. This does not establish effective
ordering, backdating authorization, lineup-at-lock membership or importer idempotency.

## Ruling carried forward

The seed is the roster at its observation time, not at receipt time. Pre-observation
moves need backward-consistent reconstruction. For a dropped player whose historical
contract is unknown, use the league's default unknown-acquisition contract and flag
**"contract from import, unverified"** for commissioner review. This rule is authorized
by APP-ROSTER-2 but **not implemented** here. No fabricated contract, source-specific
league branch, store-clock backdating, raw SQL or lineup membership bypass was added.

## Open steps

1. Optional effective time ≤ receipt time on ordinary awards/cuts/trades/reserve and
   `correct.roster`; enforce commissioner/migration authority for backdating and expose
   both times in the ledger.
2. Generic effective-order projection, tie-broken by revision, including backward
   reconstruction around the seed observation. The new receipt-history API alone is
   insufficient; do not wire it into pre-seed lineup import as a substitute.
3. Membership at each player's week kickoff in `CheckLineupAppend`.
4. Translate validated TRADE/FREE_AGENT/IR/TAXI transactions through ordinary commands;
   LOAD_ROSTERS remains baseline evidence. Establish stable retry identities.
5. Backup-API copied-profile replay proof, repeat-zero proof and 32-franchise comparison.

## Checks and profile proof

- Local implementation commit: `f1249898`.
- Targeted `go test -race -json`: **34 test/subtest PASS, 0 FAIL** across six passing
  packages; ten passes are the new tests/subtests. Two migration packages had no
  matching tests and are not counted as test evidence. Selection below.
- Plain `make build`: **PASS**, packaged successfully; binary not launched. No frontend
  source changed; standalone `tsc` was not run.
- Commit hooks: golangci-lint-full, gitleaks and ifaceguard **PASS**. No bypasses.
- Independent review and GUI verification: **not performed**.

```text
-run 'Test(NativeAssetsAsOf|LineupCommandBoundary|RecordedLineup|OnlyTheSeatedOwner|CorrectionProjection|CorrectionsDecode|MarketClosedBooksRoundTrip|RealTransactions|SyntheticTransactions|MigrationLineupSync|LineupSetMustBeLegal|LineupBelongs|ImportedCutPricing)'
```

Packages: `./internal/commissioner/...`, `./internal/ingestion/transactions`,
`./internal/leaguemigration/...`, `./internal/store/history`.

Fresh SQLite backup-API copies of both requested scratch-profile databases were made
inside `.tmp/roster2/profile`; no live-config profile was opened. The disposable
harness reused the already archived bodies and existing validators. Unlike the old
measurement harness, this run used the real receipt clock, not `SetClock`. No network
request or transaction replay occurred.

```text
Validated archived transactions: 895
League season=2026 initial_records=2
Ordinary sync pass=1 recorded=148 refused=76
Ordinary sync pass=2 recorded=0 refused=76
Bills week=1 recorded=true
Bills week=2 recorded=true
Bills week=3 recorded=true
Bills week=4 recorded=true
Bills week=5 recorded=true
Bills week=6 recorded=true
Bills week=7 recorded=true
```

**Refused before/after: 76 → 76.** Exact reasons for all 76 are in
`docs/build-handoffs/APP-ROSTER-2-refusals.txt`. Every remaining refusal is the ordinary
active-roster membership check; none was suppressed. The existing lineup importer
repeat appends zero; **transaction-replay idempotence is unproven**. This run supplied
no kickoff schedule, so it is not a lineup-at-lock proof.

No transaction has been imported. End-of-replay equality for 32 franchises is
**unproven**. The earlier baseline remains 28/32 equal, with these four differences
(not remeasured as a replay result): reference adds 10976 to 0003, 14467 to 0015 and
15263 to 0018, and removes 14467 from 0025. Its authoritative detail remains
`docs/build-handoffs/APP-ROSTER-1-baseline.tsv`.

Both `.tmp/roster1/profile` and `.tmp/roster2/profile` were checked for open users and
deleted after the commands completed. Text logs remain ignored under `.tmp/roster2/`.
No git remote contact, real MFL request or real roster move was made.

## Footprint and debt

Physical lines / Unicode code points including LF, baseline `332cb055` to the
implementation commit: affected production file `native_reading.go` **99 → 127 lines,
3502 → 4649 code points**; new handwritten tests **98 lines / 2992 code points**.
This is an addition, not a debloat claim. Documentation is excluded from these counts.

All five open steps above remain. No new product ruling is needed: Christopher has
already authorized the generic extension. There is no demonstrated external blocker;
the requested implementation is unfinished. Do not treat this commit as APP-ROSTER-2
acceptance or wire receipt-history filtering in place of effective-date reconstruction.
