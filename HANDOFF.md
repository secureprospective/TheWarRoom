# Handoff

## Baton
Bee → Christopher / ClaudeBox — 2026-11-01, `build/ROSTER-1`.
APP-ROSTER-8 acceptance remains blocked. Local commits only; no remote contact.

## APP-ROSTER-8 recheck
- No production changes from `cd66522c1`. Fresh backup-API copy: all 895 rows
  validate; both replay calls refuse the same unobserved March 7 consumed pick.
  Zero appended; complete ledger unchanged. 55.083/57.481 ms are early refusal,
  not completed importer time or successful idempotence.
- Fresh focused race batch: 16 test/subtest PASS, 0 FAIL, 0 SKIP. No new lineup
  measurements or build; 76/107, 28/32 and Bills outcomes below are prior evidence.
- Still no permitted `syncPlayedLineups` hook at this base. Need permission for
  the actual refresh body or the correct integration base, plus checked original
  pick evidence or a historical-pick ruling. No unused setting or hidden sync.
- Full recheck evidence: `docs/build-handoffs/APP-ROSTER-8-evidence.md`.
  Own disposable profile/harness removed after liveness checks.

## Where it stands
- Blank contracts now enter by ordinary roster correction; salary/year are null,
  not defaulted. Current-roster jobs say “Contract from import — fill it in”.
  Ordinary contract edit resolves the job; compensation restores it. Monetary
  readings/admission preserve uncertainty. The unused default-contract rule is gone.
- Generic pre-observation reconstruction works in the real-store synthetic test:
  invert checked membership evidence, then baseline + earlier moves in one ordinary
  atomic batch. LOAD_ROSTERS membership changes matter; reserve observations are
  commissioner status corrections, not current owner eligibility decisions.
- Prepared reserve batching is implemented with held facts and ordinary checks,
  rollback, prepared outcomes and repeat-without-write tests.
- Full 895-row archive replay still appends zero: a March 7 trade names consumed
  original pick `FP_0004_2026_3`, absent from the observed current-slot identities.
  Refusal: `internal/leaguemigration/mflsource/reconstruction.go:128`.
- Fresh-copy refusals remain 76 → 76 schedule-free, 107 → 107 kickoff-aware.
  Membership/status equality remains 28/32 (0003/0015/0018/0025 differ). Bills 1–7
  recorded schedule-free; kickoff-aware Bills 7 still fails. Explicit archived
  week 8: 0 refusals schedule-free, 32 kickoff-aware. No GUI/live acceptance.
- Targeted batches: 193 distinct Go test/subtest PASS; frontend 66 PASS. Final
  plain make build and commit hooks passed. No full suite or make verify.
- Commits: `9aecbebc6`, `5c212545a`, `a2b74949c`, `6bdc4de69`,
  `c8991085e`, `cd66522c1`. Evidence and every exact remaining refusal:
  `docs/build-handoffs/APP-ROSTER-7-evidence.md`,
  `docs/build-handoffs/APP-ROSTER-7-refusals.txt`.
- Twelve backup-API DB copies, temporary baseline source snapshot and one-job
  harnesses removed after process/open-file checks. Text logs remain in
  `.tmp/roster7/`. Own temporary stash dropped; other stashes/cache left alone.

## Next move / blocked on
- Obtain checked original-franchise → current-slot evidence for consumed 2026
  picks, starting with 0004/round 3, or a commissioner ruling for historical pick
  representation. Do not guess identity or split a player out of its trade.
- Import still lacks transaction invocation (`internal/leaguemigration/service.go:150`).
  Automatic migration actor + transaction-follow wiring is not delivered. This
  base has no `syncPlayedLineups` body or `schedule.follow`; existing refresh at
  `internal/composition/migration_app.go:79` is outside the allowed edit boundary.
- Finish durable off-roster missing-contract review: current jobs enumerate only
  rostered players, and contract edit requires a roster player.
- Then successful full replay/repeat-zero, 32/32 equality and kickoff-aware Bills 7.
  Current transaction timings are early refusal, not completed importer time.

## Tried and rejected
No skipped traded assets, guessed draft origins, fabricated contracts/football
facts, selected-pick safeguard bypass, source-kind authority, raw ledger writes,
remote/MFL call or protected-file workaround. No successful replay claim based
on zero appends after refusal.
