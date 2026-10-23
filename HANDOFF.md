# Handoff

## Baton
Claude — 2026-10-23. Nothing in flight; main is the latest state.

## Recently merged
- #108 AGENTS.md: tests run in batches, not before every commit.
- #109 Opt-in big-league benchmarks (`make bench-league`); budgets and baselines in
  `docs/data-layer/Database_Growth_Catalog.md`. Budgets: a page opens in under 1 s, a save in under 0.5 s.
- #110 Weekly results: 81 s to 0.95 s on the three-season league.

## Next
- D9: saves replay all history and miss the 0.5 s budget after about one season ("close the books").
- R2 follow-ups from the PR #101 live test (contracts tab before startup finishes; other-team labels).

## Working notes
- Commits over ssh on the Beelink need `GOTOOLCHAIN=go1.26.10` or the lint hook crashes.
- A fresh worktree needs `frontend/dist` built (pnpm install + build) before hooks run.
