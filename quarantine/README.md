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

`ScoreLeague`, kept by this pass, went in the next one (below).

To bring the harness back: move `frontend/src/` and the Go files back to their old paths, restore the
deleted methods from `session/purge-tom`, run `make bindings`, and restore `TargetMount.tsx`,
`state.ts` (`harness`), the two registry commands and the buttons from the same branch.

## Layer 1–4 engine (moved 2026-10-20, branch `session/purge-engine`)

`ScoreLeague` was the only writer of the board run, and no screen called it once the harness left.
Power rankings, Race, the pool, Lab and the draft reports read the model run, which
`engine_refresh.go` writes on its own, so nothing on a live screen moved. Under `go/`, original
content:

- `internal/rankings` (M1: the board run), `internal/composition` (engine inputs from the stores),
  `internal/scouting` and `internal/scouting/assembly` (the Layer 4 profiles), and the fetchers only
  they used: `internal/ingestion/ras`, `pfrcoverage`, `schooltier`. Tests included; none had testdata.
- `m1_scouting.go` (`buildScoutingDirectory` and its merges), as it was before this branch.
- `internal/engine/pipeline.go` (L1 hygiene, the L1–L6 pipeline, the L6 tiebreaker builder) and
  `capscaling.go` (L5). `internal/engine/pipeline_test.go` here is the original less five tests that
  stayed live in `internal/engine/decay_test.go` (`approxEq`, decay, cushion, tie order, peak ages).

Deleted from live files (recover them from git, `session/purge-harness`): `ScoreLeague`,
`ScoreLeagueResult`, `proxyLabel` (`m1_app.go`); `runModel` (`model_app.go`, whose `runModelLocked`
moved into `engine_refresh.go`); `ingestion.Share`, `ingestion.IntCell`, `model.CollegeMeasures`;
the board-run read in source health (`healthInput.boardRun`, `missing`, the "Board running reduced"
alarm and the "Running reduced" / "Feeding the board" row text). Source health now words a measured
source's row by the engine run, the path it already took when no board run existed.

Copied, not moved: `composition.PeakLimit` is now also `engine.PeakLimit` (`cmd/fit` reads it).

Kept on purpose, unreached (until the leftovers pass, below): `internal/engine/l4` (its settings table is the `l4.*` params the
Data and sources parameter editor lists, R7; the rubric math in it is unreached), the engine's types
(history's `run_scores` rows store `engine.Result`), and history's board-run store side
(`WriteRun`, `RunScores`, `scoring_runs`, `run_scores`): existing databases keep their board runs.

To bring the engine back: move the `go/` paths above back (drop the `quarantine/go/` prefix), delete
`engine.PeakLimit` or point composition at it, restore the deleted functions and the source-health
board read from `session/purge-harness`, and run `make bindings`. The `rasFetchTimeout` and
`cfbdEnvVar` constants now live in `signals_app.go`; drop them from the restored `m1_scouting.go`.

## Leftovers (moved 2026-10-20, branch `session/purge-leftovers`)

What the engine pass kept but nothing reached. Under `go/`, original content unless noted.

### Layer 4 rubric math

- `internal/engine/l4/rubric.go` (`Rubric`, `New`, `Rubrics`, `Rubric.Apply`, `breakout`) and
  `curve.go` (`Component.Apply`, `interp`, `curved`, `given`, `athleticLift`, `NeutralNorm`).
- Their tests, `golden_test.go` and `rubric_test.go`; the latter less `TestKnobsCoverTheOnMechanics`,
  which stayed live in `internal/engine/l4/settings_test.go`.
- The live package keeps `Defaults`, `Knobs` and the types they need (`Settings`, `Component`,
  `Breakpoint`, `Weights`, `Curves`), now all in `settings.go`. `params/defaults.go` seeds the `l4.*`
  params from them, and their descriptions now carry "(inert, no run reads it)", which the parameter
  editor shows in each Scouting adjustment area heading.
- `engine.CushionGuard.Slow`, whose only caller was the rubric: first deleted from `types.go`, then
  the whole file moved here (below). Its test, `TestCushionGuardSlow`, is back in
  `internal/engine/pipeline_test.go` here.

To bring the rubric back: move the two files and their tests back, drop the types they declare from
`settings.go`, move `internal/engine/types.go` back (below), and remove `rubricInert` from
`params/defaults.go`.

### History's board-run store side

Deleted from live files (recover them from git, `session/purge-engine`), not quarantined, since the
files keep live code: `history.WriteRun`, `prepareRun`, `validScores`, `scoreArgs`, `insertScoreSQL`,
`scoreCols`, `insertBoardRunSQL`, `runTable`, `Runs`, `RunScores`, `RunScore`, `scores`, the `Score`
and `NewRun` types and the `RunBoard` / `RunRebalance` kinds (`internal/store/history/runs.go`).
`writeRun`, `LatestRun` and `PreviousRun` now read and write `model_runs` only, and `NewModelRun`
feeds `pending` directly. The board-run tests in `history_test.go` were deleted; their validation and
missing-measure cases now run against `WriteModelRun`.

The schema is unchanged: `scoring_runs` and `run_scores` stay in `history.go`, the append-only
triggers and the JSON-column migration, and existing databases keep their board-run rows. No
migration was added and no table dropped.

With `engine.Result` unread, `internal/engine/types.go` had no live code left: it is under
`go/internal/engine/types.go` as it was on `session/purge-engine` (`CushionGuard.Slow` included), and
`TestTiebreakerRanksAbove` joined `TestCushionGuardSlow` in the quarantined `pipeline_test.go`. The
package doc moved to `decay.go`.

To bring the board run's store side back: restore `runs.go`, `modelruns.go` and
`history_test.go` in `internal/store/history` from `session/purge-engine`, and move `types.go` back.
