# Handoff

## Baton
Bee → ClaudeBox / policy clarification and implementation — 2026-11-01,
`build/ROSTER-1`. Local evidence commit only; APP-ROSTER-5 incomplete.

## Where it stands
- No production code changed or transaction appended. New evidence and exact
  refusals: `docs/build-handoffs/APP-ROSTER-5-{evidence.md,refusals.txt}`.
- Schedule-free copied-profile baseline: 148 appended / 76 refused in 40.640 s;
  repeat 0 / 76 in 10.446 s. Bills 1–7 recorded. Not launch timings.
- Fresh copy with held week-7/8 kickoffs: 117 appended / 107 refused in 32.165 s;
  repeat 0 / 107 in 9.955 s. All 32 week-7 lineups refuse pre-seed history,
  including Bills. Weeks 1–6 have no held kickoffs in this harness.
- Explicit archived week-8 starters: without kickoffs 32 appended / 0 refused;
  with held kickoffs 0 / 32, all pre-observation history refusal. Archived week-8
  results are unscored and ordinary historical backfill skips them.
- Three post-seed transactions explain the four original franchise differences;
  equality remains 28/32 baseline, not replay proof. Details in evidence.
- Targeted existing race tests: 26 PASS, 0 FAIL. Plain make build PASS; not launched.
  No implementation, independent review or GUI acceptance. Copies/harnesses deleted.

## Next move / blocked on
- Clarify unknown-contract salary/term: contracts settings define maximum years and
  experience minimums, not an unknown-acquisition default. Is fallback one year at
  lowest minimum, or experience-specific? No value was guessed.
- Implement authorized generic pre-seed reconstruction and contract review intake;
  current guards: `internal/commissioner/native_reading.go:40` and
  `internal/commissioner/correction_projector.go:46`. These are implementation gaps,
  not a claim that the task cannot be built.
- Translate 895 validated rows through ordinary commands, stable retries, import and
  independent sync setting; prove equality/repeat-zero/Bills/week 8 with kickoffs.
- Keep changes below backfill/migration boundary for integration cherry-pick;
  protected composition files untouched.

## Tried and rejected
- No source branch, raw SQL ledger writes, receipt-clock backdating, lineup bypass,
  fabricated default contract, or treating unscored week 8 as final results.
- 76 refusals without kickoff facts is not a complete schedule-aware baseline.
- No git remote contact or MFL requests. Pre-existing untracked `.golangci-cache/`
  untouched. One heavy gate STOP was accidentally followed by the targeted race
  batch; logged as a workflow error in evidence. Later heavy checks passed.
