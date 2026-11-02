# APP-MERGE-8 — review fixes and suite failures

2026-11-02 · Bee · build/INTEGRATE-1102 · local only, no remote/full suite.
Baseline d809fd833; resumed after items 1, 4, 5 were committed.

## Disposition and commits

| Item | Result / evidence | Commit |
|---|---|---|
| 1 | DONE: any absent salary/year/status marks ContractMissing, blanks terms and emits null salary; explicit zero stays known. Import table covers blank, each partial case and zero. Existing commissioner missing-contract jobs route exercised separately. | 7c6c3aa53 |
| 4 | DONE: one bounded history worker visits every eligible league; cancellation/second-league tests pass. Existing first-launch five-minute delay and ten-minute run timeout retained. | 74ef37bf0 |
| 5 | DONE: schedule and transaction following are independent; all four follow combinations exercised. | 21f8aa6df |
| 6 | DONE: checked scoreboard event season/week/type reaches live lines; unknown scope is not guessed, non-regular games are not filed as regular-season evidence. Poll scope captured once. Held reads match each line's scope instead of relabelling it to poll week. Cross-season/week tests pass. | 2ad87c55f |
| 7 | DONE: successful empty acquisition uses SetLiveStats' clear path and signals readers. Missing/malformed board, network/identity/box-score failure preserves held evidence. Empty events array is valid; absent events is not. | 9e4aac7ee |
| 8 | DONE: calendar drag extracts the league civil date and adds civil days, including Tokyo midnight and Chicago DST tests. Unknown timezone cannot produce a guessed date; it opens the existing rule editor instead. | e90d9640b |
| 9 | DONE: ran nativegen normally and with -surface doors; generated schemas/arguments current. | 01bfa974e |
| 10 | DONE: fixed workflow assertion, hosted clock mismatch, and regenerated only explained fixture changes. New assertion pins the surviving worker award after rollover. | dd7d9b75a |
| 6 bindings | Plain make build regenerated Game season/week/type bindings, committed without hand editing. | ba015b734 |
| 3 | REPORT COMPLETE, no source edit; findings below remain architectural debt. | this report |

## Relevant flow / preservation

Checked ESPN scoreboard → game scope → checked box score → source-ID match → live evidence.
The poll owns its final-game cache; History owns replacement/clearing and exact-scope reads.
Acquisition failure does not replace held evidence. Existing ten-minute final reread remains.
No engine/model package edits, source-based league behaviour additions or boundary exemptions.

Hosted owner read → StoredHost.CommissionerNow → History.Now → the same clock used to
record acts. Default production clock remains time.Now; tests can advance reads, writes and
workers together. The as-of projector remains intact, rather than admitting future acts.

## Fixture explanations (established BEFORE regeneration)

- workflow-before.json was NOT regenerated. Commit 9baccd3d7 deliberately added value.read
  and froze 44 commands; the test still expected 43. Updated only that assertion.
- Initially rollover awards changed from one award to empty. This was a bug, not accepted
  fixture drift: store/worker clocks advanced three minutes while StoredHost used wall time.
  The as-of projection excluded the future-dated award. Shared clock fixes visibility; the
  test now asserts player 6001 was awarded to franchise.02 after rollover.
- Complete normalized candidate diffs then showed ONLY added schedule.follow and
  transactions.follow rule rows in both fixtures, plus their values in rollover's captured
  rules. These settings were introduced by dc3191373 and 91e32e292. Defaults are empty
  schedule reference / transactions off. schedule.regularWeeks remains 13: a preceding new
  row merely moved it. Award content did not change after the clock fix.
- Regeneration also refreshes volatile instants, digests, revisions and countdowns covered
  by the existing normalizer; no normalizer changes. Raw candidate diffs were retained in
  .task-evidence/app8-{hosted,rollover}-diff.txt.

## Item 3: origin checks, report only

### internal/commissioner/native_seed.go: checkSeedSource (original lines 347–364)

Reads LeagueSource. Native origin permits recovery or enforces empty/native card startup;
import origin requires held players and prohibits O-Line cards in the snapshot. This does
NOT decode an external payload. The explanation that MFL cannot supply O-Line cards is a
real acquisition limitation, but ancestry is being used in a shared seed admission rule.
The empty-start versus held-start and recovery permissions are ordinary-league behaviour
switches, not merely provenance. An otherwise equivalent manually configured league cannot
select the imported starting-snapshot route through settings alone.

Replacement direction: explicit starting-assets acquisition/initialization choice available
to every league, with held-snapshot shape checks at that acquisition boundary. Existing
league.initialRosters selects draft/free agency, not held-snapshot acquisition, so it is NOT
an exact existing replacement. O-Line eligibility/lots should follow the existing O-Line
rules; MFL's inability to supply cards belongs to the MFL decoder/acquisition route, not to
a permanent prohibition on a league with that origin. New setting semantics need approval;
none were invented or implemented here.

### internal/commissioner/migration.go: CheckCutoverAppend (original lines 50–68)

Blocks reversion of a recorded Cutover, rejects non-import origins, requires seeding and
rejects duplicate cutovers. importedSource's legacy Source/Preset fallback is provenance
compatibility decoding. Using that provenance to decide who may stop following reference
data is an ordinary-league behaviour switch. Duplicate-act and seed checks are ledger
lifecycle checks, not decoding. The permanent no-revert rule is a locked product constraint,
not silently removed by this task.

Replacement direction: independent ordinary rule changes to schedule.follow and
transactions.follow, available regardless of origin, rather than an origin transition.
Any other remaining reference feed would need its own explicit follow setting, not a new
MFL/native mode. These two existing settings alone do not justify claiming every cutover
side effect/reversal policy is replaced. Preserve historical cutover provenance; obtain an
operator ruling before replacing its permanent/no-return semantics.

## Verification

Go 1.26.10, worktree-local lint cache; heavy-ok before heavy jobs. Discovery (`go test -list`)
preceded the final explicit package race batch; -timeout 30m (no repository-wide suite).

| Package | PASS tests/subtests |
|---|---:|
| internal/leaguemigration/mflsource | 45 |
| internal/app/mflref/migration | 8 |
| internal/app/now | 58 |
| internal/ingestion/livegame | 25 |
| internal/store/history | 466 |
| internal/leagueservice | 167 |
| internal/serverhttp | 309 |
| frontend/scripts/nativegen | 98 |
| internal/composition | 846 |
| **Total final batch** | **2,022** |

Four existing history opt-in skips: TestSleeperUnknownIDReport,
TestFullRealSeasonLoadsAsWeekParts, TestKeptBigLeagueIdentityMigration, TestReplayCost.
Separate discovered commissioner missing-contract correction/validation race run: 9 PASS.
Initial short live/history batch was interrupted by the command timeout; superseded by the
complete final batch above. Initial stale fixtures failed as expected; final batch passed
without update flags. Commit hooks passed after fixing complexity and accepting hook formatting.

- vitest src/app: 180 files / 1,833 tests PASS.
- pnpm tsc --noEmit: exit 0, repeated after generated bindings.
- boundarycheck: exit 0, 170 frozen functions/sources, no unreviewed changes.
- plain make build: exit 0; compiled and packaged, not launched.
- Evidence: .task-evidence/app8-final-{list.log,race.json},
  app8-contract-alert-race.json, app8-frontend-final.log,
  app8-tsc-after-bindings.log, app8-boundary-final.log, app8-build-final.log.

Handwritten affected production set versus d809fd833: 11 files, 2,157 → 2,290 physical
lines; 78,687 → 82,311 Unicode characters including LF. Tests/docs/generated output
excluded. This is a correctness change, not a claimed size reduction.

## Debt / acceptance still required

- Item 3's origin-based admission/cutover semantics remain deliberately unchanged.
- Operator full suite, independent review/mutant check and disposable-profile live acceptance
  remain unrun. Test Tokyo-midnight window drag lands on the following Tokyo calendar day;
  successful empty live poll must remove provisional evidence, offline poll must retain it.
- No live external feed calls or real league/profile use; no binary launched.
- Wails still emits inherited missing-model-alias diagnostics despite successful build.
- Existing unrelated .golangci-cache/ and .task-evidence/ remain untracked; no moved paths.
