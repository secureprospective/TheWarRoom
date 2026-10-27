# Handoff

## Baton
Bee — 2026-10-27, `session/revamp-E-edge-scoring`. E5 stopped on independent-study discrepancies; do not push without authorization.

## Where it stands
E2b is committed at 36ffee80. E5 is not complete. `scripts/e5_study.py` re-derives the p6 study totals; `TestE5Diagnostic` independently reads 2024 and 2025 nflverse and 2025 Sleeper caches, adapts and scores the five players, and logs stat breakdowns. It is a diagnostic, not the golden test. It requires `TWR_STUDY_CACHE` and `TWR_SLEEPER_STUDY_CACHE` and skips otherwise. No golden tolerance was weakened. Targeted diagnostic race test and `go build ./...` passed; no full suite, verify, or push.

## Next move
Investigate discrepancies before implementing the golden. Engine minus study: Hunt +0.173380, McCaffrey +0.359199, Hurts +0.541743, Allen +0.966704, Barkley +0.330065. Differences are in RTL and red-zone counts. Do not tune either side without a ruling on their evidence/rounding differences.

## Blocked on
E5 study and engine differ by more than 0.01. The owner ruled prior full-season positional rates, not weeks 1–13: 2024 RB 1106/11832, QB 78/2352, WR 62/430, TE 7/62. The earlier 0.09368635 vs 0.09386 discrepancy was weeks 1–13 vs full season; Default32's 2025 rates remain unchanged for 2026.

## Tried and rejected
Do not infer fantasy-roster membership or bye status from weekly statistics. Do not require GSIS for direct Sleeper→MFL, create fake identities, or inner-join away Sleeper-only weeks. Unknown IDs are not automatically irrelevant.

Debt left: E5 discrepancy and golden, E6, unknown Sleeper identity/roster coverage.
