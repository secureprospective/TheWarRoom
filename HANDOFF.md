# Handoff

## Baton
Review — 2026-10-28, `session/fix-correctness`. No push requested or attempted.

## Where it stands
- `9da2c766`: reject another explicit playerScores YEAR during archive selection;
  Research totals sum exactly their displayed weekly scores at display precision.
- `4afc9ba8`: normalize M2 starter-rule positions through the existing player-directory
  mapping, fixing PK/K and EDGE/DE. Other inspected comparisons are listed in
  `docs/modules/M2_Power_Ranking_Factors.md`.
- Continuation found no unfinished tracked changes and required no further code edits.
  Both fixes have recorded failing-before evidence in `.tmp/bug1-before.log` and
  `.tmp/bug2-before.log`.

## Evidence and limits
- Continuation targeted race batch: 200 top-level PASS, two opt-in live tests skipped;
  composition Research slice: 14 PASS. Logs: `.tmp/continuation-targeted.json` and
  `.tmp/continuation-composition.json`.
- Prior full `make verify` on the same code exceeded ten minutes in Go tests and ended
  with 120 packages passing, one failing. Lint, vulnerabilities, binding freshness and
  TypeScript passed; frontend tests/build/CSP were not reached. See `.tmp/verify.log`.
  Not rerun on unchanged code; no all-green full-gate claim.
- The sole failure was the inherited long Unix-socket temporary path in
  `cmd/twr-server:TestFaultsShowOnPrivateDashboardOnly`. Its directly executed race
  binary passes with relative in-worktree TMPDIR: `.tmp/continuation-socket.log`.
- `.lintcache/` remains untracked local lint output. No GUI or live-source acceptance.

## Next move
Review fixes and run disposable-profile acceptance: Research totals must equal visible
weeks before/after source refresh plus engine work; AggLineup must count the required
kicker rather than a spare receiver. CI still owes a complete green gate.

## Blocked on
Full-gate Unix-socket path handling and unreached frontend checks; reviewer/live acceptance.

## Tried and rejected
- Matching only the league URL year: engine history uses another explicit score YEAR.
- Trusting independent YTD for displayed totals: partial or differently refreshed weeks
  can disagree. The visible weekly evidence owns this total.
- Kicker-only exception in the counter: normalize recognized rule aliases at the boundary.
