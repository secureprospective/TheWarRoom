# APP-MEM-1 — measurement complete

## Baton
Bee → reviewer, 2026-11-02, `build/MEM-1`, base `4cd127dd`. Local only; no remote operations.

## Outcome / flow
`internal/composition/run.go:runDesktop` starts the opt-in Platform diagnostic and joins it
on return. Unset environment: no worker/files/timers. Set `TWR_HEAPPROFILE_DIR`: one worker,
20 × 15-second samples through five minutes; private exclusive-create CSV/profile files,
errors logged without changing application results. Heap profiles carry both in-use and
allocation-space samples; no forced GC. Existing directory is allowed; existing filenames
are refused to preserve evidence. No memory fixes or business-behavior changes.

Results, ranked causes, source locations, preservation risks and reproduction commands:
`docs/debloat-log.md`. Startup VmHWM 2,415.7 MiB; weekly stat maps 302.9 MiB steady;
historical depth 154.3 MiB and absence roster 56.3 MiB temporary. Near-peak idle/unreleased
heap 945.4 MiB. One 1500MiB limit trial: startup VmHWM 1,859.7 MiB; no observed startup
slowdown, more GC, higher five-minute RSS. Navigation peaked outside profiling window:
3,051.3 MiB default versus 1,922.3 MiB limited; owners remain unproven there.

## Evidence / verification
- `CI=true make build`: PASS; real production binary launched twice on :5, copied data only.
- `go test -race ./internal/app/platform/...`: PASS, 49 tests/subtests including 3 new tests.
- Targeted composition race batch: PASS, 9 named tests (flag/version selection).
- `golangci-lint run ./...`: PASS, 0 issues; diff whitespace checks PASS.
- 40 heap profiles, 40 CSV memory rows, 162 five-second RSS samples retained privately,
  along with binary/hash, pprof top/list outputs, launch logs and navigation screenshots.
- Both desktops stopped by their own `$!` PID; disposable profile deleted; original config/data
  untouched. Shell tool blocked literal find-delete; scoped Python deletion used only for that
  disposable path. No repository/worktree moves or generated-binding edits.
- Full verify not claimed: inherited bloat gate has 29 tiny files versus baseline 24 (HEAD alone
  also has 29), and placement lacks the already-present ingestion/livestats package row.
  Neither condition was changed or bypassed. Root ratchet and source-branch checks PASS.

## Limits / next move
Review diagnostic/log, then integrate locally if accepted. No public profile artifacts in git.
This snapshot skipped the old 2262-player engine run because inputs were unchanged; it does not
prove that OOM's exact stack. Peak RSS lies between sampling instants; pprof is GC-lagged.
Child WebKit processes are excluded from PID RSS. Single limit trial is observational under
shared-host load, not a benchmark. Several screens expose existing unavailable/private/placeholder
states; navigation screenshots are not a feature correctness verdict. Defer cleanup until after
the locked-screen revamp; if needed later, profile navigation separately.
