# Quarantine

Files the purge (2026-10-20, branch `session/purge`) judged stale but did not delete outright.
Each file keeps its original path under `quarantine/`. Nothing here is built, tested or linked
from live docs. Christopher reviews; whatever survives review moves back, the rest is deleted.

## Docs: process history (moved 2026-10-20)

Completed session handoffs, superseded RESUME files, review and panel transcripts, GLM runs,
wireframe exploration output and superseded planning. The specs, designs and rules they led to
stay in `docs/`. Selection rule: a record of how a decision was reached, not the decision itself.

- `docs/build-handoffs/handoffs/`: per-session build briefs, June to October, all shipped.
- `docs/build-handoffs/RESUME-*.md`: compaction resumes, superseded by later ones.
- `docs/build-handoffs/reviews/`, `docs/transactions/*GLM*`, `*Panel*`: reviewer prompts and outcomes.
- `docs/build-handoffs/Layer4_PreBuild_Audit.md`: the June pre-build audit.
- `docs/reviews/engineering-audit-2026-10/`: the Oct 3 audit, followed by `Deep_Audit_2026-10-19.md`.
- `docs/ui/wireframes/session-*/glm-direction*`, `session-0-test/`, the three research digests: wireframe-track inputs.
  The session-b/c wireframe HTML stays in `docs/` because `style.css` cites it as its source.
- `docs/roadmap/FILM_Calibration_Planning.md`, `Versioning_and_Releases_Planning.md`: superseded by the DECIDED docs.

Deleted outright, not quarantined: `docs/ui/registry/__pycache__/` (committed Python bytecode).

## Go: opt-in harness tests (moved 2026-10-20)

Whole test files that run only when a TWR_* environment variable points at a local archive or
snapshot; CI never runs them. `quarantine/go.mod` keeps them out of `go build ./...` and `go test ./...`.
To bring one back, move it to its old path (drop the `quarantine/go/` prefix).

- `archive_smoke_test.go` (TWR_ARCHIVE_PARITY, TWR_ARCHIVE_SMOKE, TWR_MFL_PAGES)
- `data_sources_measure_test.go` (TWR_ARCHIVE_MEASURE_DIR/DB)
- `engine_benchmark_test.go` (TWR_ENGINE_BENCH_DIR)

Kept on purpose: `internal/store/history/replay_bench_test.go` (Block C replay fix yardstick), the
`*_live_test.go` feed checks (the only drift check against real MFL/CFBD/nflverse data), and files
that mix opt-in and always-on tests.

## Alpha harness UI (moved 2026-10-20, branch `session/purge-harness`)

The original Alpha UI, reached from the target shell's "Harness" button, and the Go bindings only
it called. The target UI replaced every screen it had.

Frontend, under `frontend/src/` (same paths as before):

- `App.tsx`, `components/**` (rankings, power rankings, feed, calendar, transactions, home, inspector,
  shell, board primitives) and `store/**` (the Zustand stores): 29 files, about 6,700 lines.
- Two pieces the target UI used were copied out before the move: the `FreshnessState` union is now
  `frontend/src/app/data/freshness.ts`, and `isTypingTarget` is now `frontend/src/app/shell/keys.ts`.
  The full originals stay in `components/board/` here.

Go, under `go/` (whole files with no live code left; original content, bindings included):

- `leagueschedule_app.go` (`GetLeagueSchedule`), `m1_player_score_app.go` (`GetPlayerScore`),
  `rulebook_app.go` (`GetLeagueSetting`, `SetLeagueSettingOverride`), `transactions_feed_app.go` (`GetFeed`).

Deleted from live files (recover them from git, `session/purge-tom`): `AppInfo` (`version.go`);
`ExecuteTransaction`, `PreviewTransaction`, `GetCurrentPhase`, `runTransaction` and their DTOs
(`transactions_app.go`); `GetCalendarEvents` (`transactions_calendar_app.go`); `GetRoster`,
`GetFreeAgentPool`, `GetFranchises`, `GetLegalOps`, `capDeltaDTOs`, `resolveDirectory`
(`m4_app.go`); `GetRankings`, `latestBoard`, `priorRanks`, `rankRows`, `RankRow` (`m1_app.go`);
`joinModel` (`model_app.go`); `GetPowerRankings` (`m2_app.go`); `RefreshLeague` (`refresh_app.go`,
folded into `refreshScope("league")`); `localFreshness` (`freshness.go`). The removed switch: the
`harness.open` / `target.open` commands, the shell `harness` field, the two buttons, and the lazy
mount in `frontend/src/TargetMount.tsx`.

What was left of `m4_app.go` (`m4Timeout`) moved into `m2_app.go`, and of
`transactions_calendar_app.go` (`calendarEvent`) into `transactions_app.go`, so the bloat ratchet's
tiny-file count stays at its baseline.

Kept on purpose: `ScoreLeague`, the only writer of the board run; see `SYSTEM_MAP.md`, Built but not wired.

To bring the harness back: move `frontend/src/` and the Go files back to their old paths, restore the
deleted methods from `session/purge-tom`, run `make bindings`, and restore `TargetMount.tsx`,
`state.ts` (`harness`), the two registry commands and the buttons from the same branch.
