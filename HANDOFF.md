# Handoff

## Baton
Bee → ClaudeBox/reviewer — 2026-10-24. Repeat-restore fix on `session/bfix`; no merge authorized.

## Where it stands
- Unique 0600 per-file migration copies; retain five per DB, including legacy fixed-name copies.
- Matched-set captures now create DB copies 0600. Matched retention policy unchanged.
- Repeated old-schema matched restore/app initialization passes twice, preserving data and both rounds of safety copies.
- Regression tests initially failed as expected. Full verification found a history test assuming the old filename; it now uses migration findings. Final `make verify` passes: full Go race suite, lint, vulnerability/binding checks, 1,415 frontend tests, build and CSP checks.
- Rationale and evidence: `docs/build-handoffs/BlockB_Phase6_Backups_2026-10.md`.

## Next move
Read-only Claude review/mutant, CI and Christopher's throwaway-profile live repeat-restore gate. Open PR title: “Fix repeat restore bricking startup”. Do not merge.

## Blocked on
Reviewer and live gate, not implementation. No production data touched.

## Tried and rejected
Removing per-file copies: store initialization/offline tools also migrate outside matched-set startup, so those callers still need their own safety copy.
