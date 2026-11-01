# Handoff

## Baton
Bee → ClaudeBox / review — 2026-11-01, build/ROSTER-1. Local commits only.
APP-ROSTER-3 admission + post-seed as-of projection implemented; checks completed.

## Where it stands
- `5f7a14916`: optional effective timestamps on ordinary roster moves, commissioner
  authority for past moves, future refusal, durable effective + real receipt times.
- `c770a6e00`: inclusive effective-time projection ordered by effective then revision;
  historical market preparation, compensation, reserve ties and authority tests.
- Final commissioner/history race batch: 994 test/subtest PASS, 1 inherited FAIL,
  4 fixture/config SKIP. Only failure: `TestSinglePoolStartupDraftIsUnchanged`,
  expected 90ca21fd230f0aa6, got 8a6936145d37309e; reproduced on starting production
  files at 0fd3b6c4. Existing tests/expectations were not edited.
- Focused race batch: 119 PASS / 0 FAIL. Three behavioral mutants rejected.
  Plain make build PASS; both code commits' hooks PASS. Binary not launched.
- Evidence: docs/build-handoffs/APP-ROSTER-3-evidence.md; ignored logs: .tmp/roster3/.
  No independent review, GUI test, full verify, frontend or generated-binding change.

## Next move / blocked on
- Review this slice; investigate inherited startup digest failure separately.
- Pre-seed reconstruction remains open: seed = observation baseline. Unknown historical
  acquisition contracts use league defaults, flagged "contract from import, unverified"
  for commissioner review. No new ruling needed; not implemented in this scoped run.
- Then lineup membership at kickoff and ordinary-command transaction translation;
  importer/lineups/contracts untouched here. Refused 76 and 32-team replay equality
  are not remeasured or claimed fixed. Replay idempotence still unproven.

## Tried and rejected
- Receipt-only history cannot answer backdated roster questions; replaced with
  effective selection/order. Existing bid/claim/offer dependencies still apply.
- No raw SQL, store-clock backdating, source-specific behavior, invented contracts
  or lineup membership bypass. Invalid dependent timelines roll back.
- No fetch/push: initial git pull stopped because branch lacks upstream.
