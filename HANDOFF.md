# Handoff

- Baton: Bee, 2026-11-17.
- State: PR #325 is open and ready on `session/pulse-privtable`. Commit `c8741ff0f` adds the schema-16 `private_entries` migration, Release 9 contract and franchise-scoped `personal.NewLeague` methods; local desktop schema/API and hosted commands/UI are unchanged. Tests, lint, pre-push and CI passed on that commit. `tidy-check.sh` is clean.
- Next: Await review; merge only on Christopher's go-ahead. Future hosted callers must pass the franchise ID from authenticated identity, never request data.
- Blocked on: Reviewer/merge decision only.
- Tried/rejected: Reusing Release 8 edits an immutable shipped contract and violates `TestCurrentContract`; added Release 9 / schema 16 as approved.
