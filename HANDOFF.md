# Handoff

## Baton
Christopher — 2026-10-27, `session/revamp-D-scoring-engine`, part 8. Local only; do not auto-push.

## Where it stands
- Merged `origin/main` through `2eae4ebd` with merge commit `fab346b8`; no rebase or force-push.
- Kept main's SYSTEM_MAP rows and restored D's native-scoring doc and fantasyscore fixtures.
- Seed preparation checks request replay first, then Default32's own franchises/O-Line lots or
  held capture. Policy stays in app/commissioner; composition exposes a capture-only host port.
- Two boundary source fingerprints reviewed for this inward routing/port change; no policy
  allowances added. `make boundary`: 166 frozen functions/sources, no unreviewed changes.
- Focused race tests passed: Default32 start/retry/refusal, held capture/failure, native facts,
  and four composition seed tests (held assets, refresh/retry, missing inputs, generic IPC).
- Full `make verify VERSION=dev COMMIT=revamp-D-p8 BUILD_DATE=revamp-D-p8`: exit 0, 172 s,
  isolated worktree-local GOLANGCI_LINT_CACHE. Lint 0 issues; race suite passed (cached where
  valid; composition's preceding full race run passed in 1084.907 s); bindings/typecheck passed;
  frontend 142 files / 1704 tests passed, build/budget/CSP checks passed.
- Last meaningful gate line: `csp-browser-check: inline script, event handler, foreign script and eval blocked`.
  Literal final log line: `Exiting due to channel error.` Browser shutdown warnings did not fail the gate.

## Next move
Christopher pushes PR #187; check fresh CI and perform the outstanding live browser/feed gates.

## Blocked on
No push authorized. Real NFL roster-season feed acquisition and previous live-browser follow-ups
remain unverified; this part did not run live feeds or browser feature click-throughs.

## Tried and rejected
- Initial focused-package batch: native facts test failed; composition hit its default 10-minute
  timeout. Focused seed integrations subsequently passed; full gate uses its existing 60-minute timeout.
- Verify attempts: 131 s failed unused nolint (removed); 1296 s failed auto-merged facts test;
  7 s failed shared linter cache pointing at deleted revampB2 files. A fresh local lint cache
  fixed that environment issue without touching another worktree or weakening checks.
- Updated `TestHeldNativeFactsRequireDirectoryAndInjuryObservation`: exact count 1 → 33 because
  D adds 32 O-Line cards; now checks every card's name/position, preserving injury assertions.
