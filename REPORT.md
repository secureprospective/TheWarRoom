# Report
- Done 1: PASS. Fake 3-league pass fetched week 1 once; revised/pinned two leagues, skipped unpublished third; second pass added no digests. Removing memo failed: `unique fetched weeks = 2; fetches = map[1:2]`.
- Done 2: PASS. Real `leaguePins` concurrent read during swap returned old/new pin without error (`go test -race ... -run '^TestHostedNFLSchedule'`).
- Done 3: PARTIAL. Focused schedule tests and `make lint VERSION=dev COMMIT=sched BUILD_DATE=sched` pass. Required full race command timed out three times in existing `TestMirrorFetch*` migration tests (10-minute Go test timeout); `internal/store/history` passes/cached. New tests pass separately.
- Flow: enabled timer → bounded pass memo/fetch/archive → shared per-league revision → verified evidence → `leaguePins.setPin` atomic swap; per-league errors continue; context cancellation stops worker.
- Flag: `-nfl-schedule-refresh`; sandbox opt-in marker documented in `cmd/twr-server/README.md`.
- Tidy check: blocked by unrelated live AGY run PID 2470461 (canary-1124); did not stop/edit another agent's run or files.
- Files: `cmd/twr-server/{README.md,football_pins.go,hosted.go,refresh_nfl_schedule.go,refresh_nfl_schedule_timer.go,refresh_nfl_schedule_timer_test.go}`, `deploy/server/twr-sandbox`, `docs/architecture/line-budget-baseline.txt`, `docs/architecture/line-budget-reasons/session-pulse-schedtimer.md`.
