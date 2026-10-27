# Handoff

## Baton
Bee — 2026-10-27, `session/mode1-mirror`. F3a changes uncommitted; do not push.

## Where it stands
- Native starts now use roster O-Line setting, not preset; consistency validation and requested regression tests added. Build, vet, targeted tests and lint pass; touched-package race tests pass in two batches.
- The requested consistency rule breaks existing single-field O-Line changes and can make existing mixed-rule histories unreadable. The old history scoring test failed when setting `scoring.lineUnit` on Legacy; changing that test fixture to Default32 made the requested suite pass but does not fix the regression.
- Default32 creation is still restricted to 32 franchises by `internal/commissioner/creation.go`; a 12-team empty-preset league with O-Line settings and a synthetic historical 12-team Default32 record both start with 32 lots.

## Next move
Christopher/ClaudeBox: decide an atomic multi-field O-Line setting transition and compatibility policy for previously recorded mixed states before committing. Existing `RulesSet` changes one field at a time; `internal/leaguerules/resolve.go` validates complete current readings. No safe way to transition from all-off to all-on with the requested invariant and existing command interface.

## Blocked on
No commit: never commit broken code. The required validation conflicts with existing settings workflow and historical read compatibility.

## Tried and rejected
- Changing only the line-unit scoring field: existing history test failed with the new consistency error.
- Rewriting the scoring test to use an already-consistent preset: verifies scoring but does not repair existing leagues or the transition.

## Debt left
GUI verification, transition/compatibility design, and inherited generic seed descriptions. The full targeted race batch timed out after leaguerules, commissioner and history passed; app/commissioner and leagueservice subsequently passed separately.
