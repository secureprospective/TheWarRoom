# Handoff

- Baton: Bee, 2026-11-17.
- State: Memory fix and regression test are committed on top of current `main` (#321). Four same-harness local 180s runs: before VmHWM 719,892 / 855,268 kB; after 608,376 / 622,140 kB; means 787,580 → 615,258 kB (21.9%). Local process RSS, not lab cgroup RSS. `go test -race ./internal/store/history/...` and `make lint` pass. Baseline pprof `alloc_space` attributed 24.67 GiB to the removed read copy; this line is absent after.
- Next: push and open a non-draft PR; include peak pprof top 5, benchmark table and RSS limitation in PR body. Do not merge.
- Blocked on: none.
- Tried and rejected: None.
