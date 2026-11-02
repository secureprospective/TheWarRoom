# APP-ROSTER-10 — proof completed; ten-minute worker budget still fails

Base `dd987f494`; final runtime `2ee267af8`. Local only: no remote, live configuration,
real MFL request/move, source DB write, clock substitution or protected-file edit.

## Delivered

- Plain `make build` passed before proof, regenerating missing historical-pick and
  transaction-follow bindings (`590f4caf0`). Existing Wails generic-discovery warnings
  remain; the production binary was packaged, never launched on this machine.
- First complete held-transaction importer: **3,338 appended in 17m50.422515204s**;
  repeat **0 in 781.710043ms**. CPU samples attributed 41.10% to
  `commissioner.rememberNativeContracts`: copying every rostered player before
  every correction, even when only one player's membership changed.
- `2ee267af8` limits single-player capture to that player on the named side.
  Initial capture and multi-player operations retain full capture. Ordinary
  authorization, validation, preparation, append, contract restoration and atomic
  recovery are unchanged. No timeout increase or weakened checks.
- A new backup-API replay on that candidate completed: **3,338 appended in
  10m43.484829317s**, repeat **0 in 569.330031ms**. Both full measurements include
  held draft lookup, translation/reconstruction and the entire ordinary importer;
  CPU profiling was enabled in both. This is about 40% faster, but **43.485 seconds
  over** the protected ten-minute lifetime, before lineup import even begins.

**Worker remains blocked:** `internal/composition/migration_app.go:161` bounds the
worker to ten minutes; `internal/store/history/migration_import.go:73` commits only
at the end of the atomic batch. A direct completed call without that deadline is
not proof the production worker can commit it. No production worker/GUI acceptance
is claimed and the timeout was not raised.

## Final copied-profile proof

SQLite backup API copied all three DBs from the authorized scratch profile into
worktree-local disposable profiles. The followed league began with two records.
All **895** transaction rows validate. After transaction replay the followed
ledger has **3,340** records. Two independent backup-API snapshots of that completed
ledger provide schedule-free and kickoff-aware lineup runs; the transaction import
is not redundantly run once per kickoff mode. All jobs ran one replay at a time.

The input bodies match previous evidence: players `4c060f47…`, transactions
`9375de87…`, weekly results `61fc1e31…`, roster reference `ce0c0c0b…`.
Only held week-7/8 kickoffs are supplied for the historical/in-play proof. No
kickoffs for weeks 1–6 were invented. Baseline counts/times below are inherited,
measured APP-ROSTER-7 results, **not a fresh before run in this session**.

| Requirement | Schedule-free | Kickoff-aware |
|---|---|---|
| Refused before → after | **76 → 75** | **107 → 77** |
| First historical appended | 149 | 147 |
| First full historical lineup call | 10m25.269960987s | 10m31.033218601s |
| Prior first historical call | 40.540s | 33.141s |
| Repeat appended / refused | **0 / 75** | **0 / 77** |
| Repeat full lineup call | 3m15.367646231s | 3m30.078635924s |
| Bills weeks 1–7 | **all recorded** | **all recorded, including 7** |
| Explicit archived week 8 appended / refused | **32 / 0** | **32 / 0** |
| Explicit week 8 first time | 1m35.94870252s | 1m43.63441875s |
| Explicit week 8 repeat | **0 / 0**, 179.190629ms | **0 / 0**, 177.985077ms |
| Final membership/status equality | **32/32** | **32/32** |
| Final followed-league records | 3,521 | 3,519 |
| Actual missing-contract / unverified-pick jobs | **2 / 0** | **2 / 0** |

Historical lineup runtime has **regressed materially** with the expanded ledger;
transaction replay acceleration is not an overall refresh-speed success. These
are completed direct importer calls, not app-launch or live Pulse measurements.
Week 8 was explicitly selected from held starters, without changing source scored
flags or claiming final results. Each historical refusal list is byte-identical
between first and repeat. Every exact remaining refusal is retained in
`APP-ROSTER-10-refusals.txt`.

### Remaining reasons

Both modes have the same **74 week-1–6** refusals: the named starter is not on that
franchise's active roster in the view available to the check. With no held kickoff
for those weeks the ordinary check uses the current roster, not a guessed date.
The history now exists, but missing dated football facts do not become known merely
because transactions were imported. Some source selections may also be invalid;
no claim that supplying kickoffs necessarily clears all 74.

- Schedule-free additionally refuses week 7, franchise.04, **All, Erick**: currently
  IR, so not on the active roster. At the held week-7 cutoff he was active and this
  refusal disappears in the kickoff-aware run.
- Kickoff-aware additionally refuses week 7 **Kirk, Christian (15)**,
  **Terrell, A.J. (32)** and **Miller, Von (20)**. These are the ordinary first-kickoff
  rule, not missing pre-seed history: cutoff `2026-10-23T00:15:00Z`; Kirk and Terrell
  were IR then and were activated October 25 at 15:43:02Z and 02:48:42Z respectively;
  Miller was acquired October 25 at 01:56:39Z. All three source moves are validated
  and recorded. Do not silently replace the first-kickoff rule with Sunday/current
  membership to force these lineups through.

The former four franchise mismatches are repaired: 10976 is on 0003, 15263 on 0018,
and 14467 moved 0025 → 0015. Equality covers every player's membership and reserve
status for all 32 franchises, **not contract amounts or pick catalogue equality**.
The two actual jobs are `contract.franchise.03.10976` and
`contract.franchise.18.15263`, both “Contract from import — fill it in”. No salary or
years were invented. The held draft evidence produces zero unverified-pick jobs;
synthetic checked/unverified-pick paths remain covered by focused tests.

## Remaining CPU cost

Final CPU profile: 643.48s wall, 888.27s summed CPU samples (multiple runtime threads).
Percentages are cumulative/overlapping, **not additive wall-time shares**:

- Ordinary append checks: 43.79%; correction projection subset: 30.42%.
- `eventsInTransaction`: 22.17%, re-reading/decoding the entire growing prefix after
  each batch command (`migration_import.go:65`, `closed_books.go:81`).
- Governance replay: 15.29%; contract-history capture remains 13.69%, down from 41.10%.
- Background GC: 20.69%, overlapping allocation-heavy work above.

No second speculative optimization was stacked. Next speed work should preserve
ordinary admission and exact replay/retry/recovery behavior while addressing
measured prefix rereads and projection work. It needs another measured proof;
raising the timeout is not the fix.

## Verification / limitations

- Final focused `-race -count=1` batch: **52 test/subtest PASS, 0 FAIL, 0 SKIP**,
  over commissioner, history and mflsource. Discovery used `go test -list`.
  Includes contract capture, side isolation, zero/edited/missing terms, historical
  picks/reconstruction, Bills lock behavior, transaction sync/follow and atomic
  recovery. A final all-player-capture mutant fails selective/absent-player and
  side-isolation assertions; restored candidate passes again.
- Plain final `make build`: PASS; both production commits' hooks PASS;
  `git diff --check`: PASS. The first perf commit attempt hit the complexity gate;
  simplified the target selection, without suppression/bypass, then committed.
- No full suite, make verify, remote, independent review, GUI, actual worker deadline
  invocation, automatic refresh acceptance or real MFL move.
- One unoptimized lineup attempt and one pre-commit optimized transaction attempt
  were stopped before final acceptance, not reported as completed timings. The
  latter's atomic rollback was checked: followed league still had two records.
  Final tables use only completed candidate runs.
- Whole affected handwritten production file: 259 → 261 physical lines; tests:
  31 → 77; generated bindings: +86 lines. No reduction claim or new dependency.
- Own disposable DBs, raw bodies and one-job harnesses are removed after process /
  open-file checks. Text logs and CPU profiles remain ignored in `.tmp/roster10/`.
  Other agents' files and the pre-existing lint cache are untouched.

## Debt left

Ten-minute worker acceptance **fails**; initial replay plus first lineup import
cannot fit. Full-prefix ledger work also makes refused lineup retries expensive.
The 75/77 refusals remain explicit (74 lack held week-1–6 kickoff context; the three
week-7 cutoff conflicts are evidenced above). Durable off-roster missing-contract
review remains prior debt: current jobs enumerate rostered players only. Older
imported ledgers still need `transactions.follow` enabled through the ordinary
rule-change flow. Independent review, actual automatic worker and GUI/Pulse proof
remain unperformed.
