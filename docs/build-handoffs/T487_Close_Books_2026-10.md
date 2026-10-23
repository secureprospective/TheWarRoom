# T487 initial investigation — historical evidence

These notes describe the initial investigation at base `b72eb6c8`, before the checkpoint implementation. Its unchanged-code measurements and rejected experiment are retained as historical evidence, not the implementation's before/after result. PR #113 contains the current implementation, schema change, verification and measurements.

## Source-traced slice

- `history.ExecuteCommissionerWithFacts` calls `transactionRecords` under the immediate append transaction. It incrementally caches decoded/validated records by revision, but deep-clones the whole cached ledger for callers. A cold cache validates every record.
- `checkCommissionerCommand` authorizes, checks request-key retries, then runs append checks. `CheckRulesAppend` resolves the trial ledger and checks current/proposed values separately. Full ledger identity, receipt metadata, privacy and old request keys must survive any summary.
- `commissioner.RulesInForce` already recognizes recorded rollovers: `rulesOfExplicitSeason` starts a new season from `Receipt.Plan.Rules` and reads only later changes; a closed season uses `Before.Rules`. `applyNativeRollover` similarly replaces native assets with the frozen plan. These are partial closed-books behavior, not a general replay checkpoint.
- `Replay` resolves compensation backwards, including reversed reversals, transaction corrections and draft restores. A season boundary alone does not prove that every older record can be dropped.
- `GovernanceState` still replays people events and activity from the entire ledger. `foldLedger` is currently a per-reading scoring memo (active entries, stat effects and provenance positions), not the whole state required by save validation.

## Acceptance gap

`seedLeagueWrite` creates a league in 2013, then writes only immediate `cap.amount` changes, microseconds apart. Both growth benchmarks use it. It contains **zero closed seasons** at every size. A closed-season-only optimization cannot establish bounded save/page time on these fixtures. Preserve these measurements, but add a separate several-closed-seasons fixture and compare summary-enabled and forced-full replay on the identical fixture. Do not silently add rollovers to the existing baseline and call the resulting numbers comparable.

Next implementation step: define which save/read folds can resume from a checked prefix while retaining the complete authoritative ledger for historical reads, compensation, privacy and request retries. Start with the existing rollover snapshots and `ledgerFold`; test missing, stale, mismatched and invalidated summaries against forced-full replay. This session did not implement or prove that contract. Broader storage/architecture decisions belong with ClaudeBox; Christopher must approve any schema change before merge.

## Experiment, rejected

Changed `governance.Replay` locally to mutate its private replay state instead of cloning for every event, and added a differential check against isolated `Apply`. Targeted race tests for governance, commissioner and history passed. A one-shot growth run still exceeded budgets (D9 at 50k: 9.76 s; D1: 5.51 s). The experiment and its test were reverted; it was not the approved summary design and did not solve the measured problem.

A CPU/allocation profile of that experimental 10k D9 run included fixture setup, so its percentages are not isolated save costs. It nevertheless showed substantial allocations in catalogue cloning, `ruleFields` and `DecodeSetting`, with `CheckRulesAppend` and cold `transactionRecords` prominent in the cumulative allocation view. It does not establish an exact production hotspot ranking.

## Final measurements: unchanged production code

Go 1.26.10, Ryzen 9 6900HX, default 16 Go processors. Command:

```sh
BENCH_LEAGUE_SCRATCH="$PWD/.scratch/t487" go test -tags leaguebench ./internal/store/history \
  -run '^$' -bench '^BenchmarkLeague(Write|PeopleGrowth)$' -benchtime=1x -count=5
```

Seconds, medians of five; supplied baseline versus this session's final **unchanged-code rerun**, not an improvement measurement:

| Records | D9 supplied | D9 rerun | D1 supplied | D1 rerun |
|---:|---:|---:|---:|---:|
| 1,000 | 0.15 | 0.331 | 0.11 | 0.129 |
| 5,000 | 0.69 | 1.328 | 0.48 | 0.515 |
| 10,000 | 1.56 | 2.512 | 0.87 | 3.223 |
| 20,000 | 2.95 | 5.080 | 1.71 | 2.724 |
| 50,000 | 7.14 | 12.235 | 3.93 | 10.972 |

Runs were noisy: D1 10k ranged 1.505–4.282 s; D1 50k 7.636–11.500 s. Machine idleness was not established. Do not attribute differences from the supplied baseline to source changes. These growth benches generate their own scratch databases; they do not use or rebuild the cached 35-minute thick3 fixture. No thick3 benchmark was run.

## Verification and missing evidence

- Frozen frontend install and frontend build passed before Go builds.
- Targeted `go test -race` on governance/commissioner/history passed on the subsequently reverted experiment; not evidence for a delivered summary.
- Final `go test -race ./...` failed: the unmodified `internal/composition` package exceeded Go's default 10-minute package timeout while multiple tests were running. No assertion failure established; no timeout change or second full-suite run made. Other reported packages, including commissioner, governance and history, passed.
- No closed-season differential test was delivered. Summary correctness, fallback behavior, save/page budgets and restart persistence remain unproven.
- No schema change. No GUI/live gate. No merge. No intentional live MFL requests or roster moves; tests used their configured fixtures/fakes and live opt-ins were not enabled.
- Raw local evidence is in ignored `.scratch/t487/`: `frontend.log`, `targeted.log`, `first-bench.log`, `profile.log`, `cpu.out`, `mem.out`, `final-bench.log`, `full-test.log`. These scratch paths are not committed artifacts.
