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
