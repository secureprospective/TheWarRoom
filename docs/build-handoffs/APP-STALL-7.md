# APP-STALL-7

Branch `build/STALL-7`, base `955a0d04`. Continued Sol's uncommitted work, then Tom. **Desktop gate: FAIL (idle CPU).**

## Cause and changes
- HQ "Could not refresh your league": logged error (new `slog.Error` in `TargetWorkflow`) is
  `bids.read`/`moves.read` → "you may act only for your own franchise". It's an authorization refusal, not a deadline.
  Sol's frontend fix (`canManage` = viewer owns the franchise) stops HQ's hosted room requesting owner-only data
  for unowned franchises. Server checks are unchanged, so authorization is intact. 28 such refusals still appear per run,
  coming from another screen's reads. They no longer block HQ.
- Lineup: `LineupLedger` (per held revision, keyed by rule-boundary interval + principals) shares the
  `NativeBase`; bench uses roster membership only; decision exceptions folded once per reading.
  NativeLineupReading stacks: 0 in both final dumps.
- Stat queue: `RulesMemo` shared across pricing; `RulesMemo.ruleLedger` resolves a proposed
  `stat.queue.approve/reject` trial against its held (replay-cached) ledger; `ResultFlipCheck` uses the reading's memo
  via an optional `RulesInForce()` on sources.
- `localWorkflow` joins duplicate in-flight reads.

## Evidence (desktop, MNF live, 4 laps, 20 s after arrival; run 2 = final tree)
hq 435→239–248% · pulse-now 392→253–258% · pulse-race 46 first lap, then 313–354% · home 49–67% · warroom 50–71% (417 once) ·
tradefloor 47–54% · control 209–218%. RSS 1375–1823 MB (PASS). Data showed on HQ/Pulse within the first lap (screens in `~/scratch/stall7-shots/`).

Final dump hot paths remaining:
1. `appvalue.(*Preparation).sweep` → `HeldCommissionerRecords` → `transactionRecords` → `deepClone` of the full 16.8k-record ledger, on every sweep. Also `ReadNativeBase`→`RulesNow` full fold.
2. `StatQueueReading.price` → per item `foldLedger(scoreTrial(...))` (full Replay + stat-effect fold per proposal), re-run on each live capture.

## Tests
- Targeted `-race -p 2`: commissioner 34 PASS / 0 FAIL (incl. new `TestRulesMemoProposedStatMatchesFullResolution`,
  `TestLineupLedgerByteEquivalent`); history 18 PASS / 1 FAIL — `TestNativeLineupWire`, **fails identically on base
  955a0d04** (fixture lacks `fieldSources:{reception:""}`), and our output is byte-identical to base's.
- Frontend: 3 files, 20 PASS. golangci-lint targeted: 0 issues. `make build` OK.

## Debt / next
- Preparation sweep must read shared read-only records (not `transactionRecords` deep clone), memoised per revision.
- Stat-queue pricing needs an incremental trial fold (held fold + one proposal) or per-revision caching of priced items.
- Stale `native-lineup-wire.json` on base.
