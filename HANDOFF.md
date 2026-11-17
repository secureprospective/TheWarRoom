# Handoff

- Baton: Bee, 2026-11-17.
- State: Job 5 is blocked before implementation. `store/personal` reads `personal_entries`, but that table exists only in `internal/db/main_schema.go`; hosted startup opens/migrates `server.db` with `history.LeagueSchemaVersion()` and does not initialize the main schema. `history`'s shared file is server-global, while `personal_entries` keys only by `mine`, so reusing it there also collides across leagues. Draft blocker PR #328 is open; CI is green (docs-only).
- Next: Christopher must approve per-league private persistence and its schema migration. The task forbids a new table/migration, so no feature code was kept.
- Blocked on: storage design / migration approval.
- Tried and rejected: attaching `personal.New(files.shared)` without a schema change (fresh hosted DB has no table; server-global franchise IDs are not league-scoped). Calling `Store.Initialize` is not viable: it applies the main schema to a hosted database using a different schema version.
