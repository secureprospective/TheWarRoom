# Handoff

- Baton: Bee, 2026-11-17.
- State: Job 4 stopped before implementation. Hosted league DBs are separately migrated by `history.MigrateLeague`; they do not include `mainPersonalDDL` / `personal_entries`. The existing personal store initializes that table only in `MainSchema` for the local desktop DB. No hosted franchise-private persistence exists to reuse.
- Next: Christopher decides the required server persistence/schema migration. Keep queue and ADP/tier unavailable until then; hosted `marketfeeds` ADP is not populated (only local app/core captures those feeds).
- Blocked on: task forbids new DB tables/migrations and directs a DRAFT PR if one is needed. A private queue cannot safely use league cache or silently add storage.
- Tried and rejected: none; no code or tests changed. Source-traced `personal_entries` DDL, hosted league migrations, and server `history.Store` routing; desktop-modes checks passed.
