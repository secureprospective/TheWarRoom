# Handoff

## Baton
ClaudeBox / Christopher — 2026-10-27, `session/revamp-E-edge-scoring`.
Revamp E part 1 stopped at the brief's RTL design checkpoint; no scoring code changed.
Do not push or rebase this branch.

## Where it stands
- Base remains D at `391933f3`. `git pull --ff-only` refused: this branch has no upstream;
  no alternate merge/pull was attempted.
- Read D's scorer, held nflverse adapter, pinned evidence/revision path and the p6/TFL studies.
- Independently recounted 2025 PBP conversions (REG weeks 1–13): Hunt 17/11,
  McCaffrey 29/1, Hurts 12/8, Keenan Allen 23/1. Printed conversion PPG reproduce p6.
- Reproduced 2025 RB rate 0.09386238312155358; half-strength RTL: Hunt +2.562503,
  Barkley −3.566352. These are research calculations, not engine golden results.
- Targeted baseline: `GOTOOLCHAIN=go1.26.10 GOMAXPROCS=4 go test -race
  ./internal/fantasyscore/ ./internal/fantasyscore/source/ ./internal/leaguerules/`:
  all three packages passed (1.392s / 4.829s / 4.760s).
- Local reproduction/test output is under `.scratch/revampE/` (ignored).

## Next move
Approve a season-settlement contract before adding RTL: season scope/completeness,
league-position cohort, rational coefficient, evidence digest and a distinct seasonal
result. Then implement all five rulings and the new golden evidence together.
Prefer a separate season settlement over silently rescoring earlier matchups.

## Blocked on
Current `fantasyscore.Score` receives one player's week only. Held revisions may
contain arbitrary week subsets; no finalized-season/completeness contract supplies
season carries or the season's positional league rate. The studies use weeks 1–13,
not a complete NFL season. A weekly implementation would need explicit cross-week
revision/correction ownership rather than an inferred current-season rate.

## Tried and rejected
- Per-event RTL: expressly overruled by Christopher.
- Treating a held partial-season cohort as final: gives the wrong eligibility/rate.
- Rounding −0.31242 to the existing four-place Decimal rate: changes the ruling.
- Claiming unchanged p6 totals as the new golden: ruled PBP red-zone counts and rate
  RTL intentionally differ from p6's Sleeper red-zone/event RTL totals. Need separate
  frozen-study reproduction and explicitly adjusted reference fixtures, never tuned expectations.
