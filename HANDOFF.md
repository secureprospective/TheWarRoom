# Handoff

- Baton: Bee, 2026-11-18. Bundle base includes Control Room PRs #347 and #350.
- State: D65 implementation is committed on `session/pulse-b5perm`; target is `bundle/b5`.
- Flow: native catalog → `history.Store` transaction → `timedCommandRole` → `AuthorizeRecords` / `mayRun` → ledger append → governance replay; `commish.people.read` exposes effective permissions. Grants are private ledger events; no table/migration.
- D65: six permission groups; whole-mutation-catalog classification; preview/execute permission command; replay/revocation; co-commissioners get six; vote close/tiebreak stay commissioner-only.
- Filelen: `internal/store/history/commissioner.go` is 769 lines (base 727); transaction-time deputy authorization stays beside the existing serialized append check so all writes share one gate.
- Verified: touched Go race packages; frontend 127 files/921 tests and TypeScript; generator freshness; `make lint`; `make bloat`.
- Slop fixed in #350: removed name-substring rule-board classification and restored owner guidance. D65: no unrelated slop found in scope.
- Tried/rejected: letting Change rules bypass field-level C-only checks; `TestCommissionerRulesPermissions` preserves the restriction for `league.teams`.
- PR: #355 is open against `bundle/b5`; push passed changed-package pre-push verification. CI/review pending.
- Open decisions untouched: snooze scope, deputy job visibility, written-request visibility, division names.
- Machine hygiene: tidy-check reports other fleet runs active; left them untouched.
- Next: head brain review/CI for #355; revise only if requested.
- Blocked: no D65 blocker; preset comparison, house rules and written requests still await hosted reads from #350.
