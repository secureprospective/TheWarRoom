# Handoff

- Baton: Bee, 2026-11-17.
- State: Added the schema-16 `private_entries` migration and Release 9 contract row. `personal.NewLeague` shares the existing store logic, scopes every query by the explicit franchise ID, and overrides `RawChange.Mine`; local desktop schema and API are unchanged. No hosted command or UI was added. `go test -race ./internal/store/history ./internal/store/personal ./internal/serverversion` and `make lint` passed after final edits.
- Next: Commit and push (pre-push still runs bindings, lint, build, and changed-package tests), update PR #325, and mark ready only after CI is green.
- Blocked on: None. Recheck PR #325 checks after the implementation push.
- Tried/rejected: Reusing Release 8 would edit an immutable historical row and violate `TestCurrentContract`; the approved next row is Release 9 / schema 16.
