# Handoff

- Baton: Bee, 2026-11-17.
- State: Job 4 stopped before implementation. Hosted league DBs use `history.MigrateLeague`; they do not include `mainPersonalDDL` / `personal_entries`, initialized only for the local desktop DB. No hosted franchise-private persistence exists to reuse. Draft PR #323 is open; CI passed on rerun.
- Next: Christopher decides the required server persistence/schema migration. Keep queue and ADP/tier unavailable until then; hosted `marketfeeds` ADP is not populated (only local app/core captures those feeds).
- Blocked on: task forbids new DB tables/migrations and directs a DRAFT PR if one is needed. A private queue cannot safely use league cache or silently add storage. `tidy-check` also reports six dangling Chromium symlinks in unrelated `/home/chris/scratch/bee-resume-v14-*` folders; left untouched outside authorized paths.
- Tried and rejected: no code/tests changed. Source-traced personal DDL, hosted migrations, server routing; desktop-mode checks passed. Initial CI shard 1 failed two MFL tests with `CloseIdleConnections`/retry symptoms; rerun of failed CI checks passed.
