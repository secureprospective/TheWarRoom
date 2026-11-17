# Handoff

- Baton: Bee, 2026-11-17.
- State: Commit `92be27e40` pushed. Draft PR #327 is open and unmerged: https://github.com/secureprospective/TheWarRoom/pull/327. Existing `native.reserve.check` now returns optional taxi experience/years remaining from its validated rule calculation; hosted HQ shows those columns only for held values. Desktop taxi UI is unchanged. IR remains unavailable because the hosted server holds no current injury/return feed. Race test for `internal/nativecheck`, all frontend tests (115 files / 886 tests), frontend typecheck, `make lint`, binding generation, and push-hook changed-package verification passed. No live server click-through was performed.
- Next: Review PR #327; decide how to provide current hosted injury/return facts. The nearest existing parser is MFL `TYPE=injuries`, but the governing plan has no ongoing MFL sync.
- Blocked on: `tidy-check` names six dangling Chromium symlinks in external `~/scratch/bee-resume-v14-{browser,render2}` folders. Left untouched because writes are outside this task's authorization.
- Tried/rejected: Do not infer injury/back from NFL injury eligibility, game state, or historical weekly practice/report statuses. nflverse has weekly status only, ESPN parser has game/lineup status, and hosted league-load does not ingest the MFL injuries export.
