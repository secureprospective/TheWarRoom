# APP-STALL-1 — handoff (2026-11-02)

Branch `build/STALL-1` off `build/INTEGRATE-1102` @ 7371e620.

## Cause (evidence)
- SIGQUIT dump of PID 203866 (7371e620): no deadlock. The only busy app goroutine was
  `appvalue.Preparation.Run → Source.Scopes → leagueRecords → history.Store.Events → scanEvents → json.Unmarshal`.
  That is the 1 s value-producer loop (d2b70b16, 2026-11-01). Neither suspect commit touches it.
- The `native.mfl-14432-2026` commissioner ledger grew from about 150 events (11-01) to 17,118 events
  (18.8 MB of JSON). Almost all are distinct `commish.stat.queue.approve` entries from `system.stat-feed`
  (9,623 at 17Z, 344 more during the 13:07 run). Every second, the producer re-decoded the whole ledger twice
  (discover and sweep), pinning the CPU and starving the history read pool. Home's 13:08 reads then hit
  `read linked identities: context deadline exceeded`, and Home read only once, so those errors stayed.
- Bisect: 73a9fd15 on the same profile showed the same signature (450–520% CPU, no engine line). It was then
  OOM-killed at 6.4 GB anon RSS, which is the depth-decode spike that 001825e9 fixed. So neither
  001825e9 nor 7371e620 caused this.
- "Engine run … players valued" is absent by design. The profile's last model run is 17:37:46Z, and there is no
  `loads` row with added>0 after it, so `EngineDue` is false and `CheckEngine` returns silently. A fresh
  copy of ~/f2click behaves the same way, so the "players valued within 3 min" check cannot pass until an input loads.

## Fix
- `internal/store/history/events_cache.go`: `Events(…, 0)` keeps a decoded prefix per (league, stream)
  and reads only the appended tail. The ledger is append-only (triggers). The prefix is dropped if the held
  newest id is no longer at its revision (restored or replaced DB). Callers get deep copies. The cache is
  capped at 100k events. The same events reach `Records`, so valuation inputs are unchanged.
- Home (`HomeBrief.tsx`, `homeReads.ts`): when a tile's error is transient (deadline/timeout/startup/not ready),
  Home re-reads on `target:season` and on a backoff of 5/10/20…60 s, merging over the brief on screen.
  Final errors are not retried.

## Verification
- Fresh copy of ~/f2click, 10 min, 30 s samples: RSS 1.59–1.92 GB (max 1921 MB at 30 s). 0 "deadline exceeded".
  Home tiles (Transactions, alerts, research) loaded.
- `go test -race -p 2 -run 'WholeStream|LeagueEvent|Commissioner|Native|Records' ./internal/store/history`: 70/70 PASS.
- vitest `src/app/shell/homeBrief.test.tsx`: 25/25 PASS (2 new: transient retry, final error not retried).
- `make build`: passes.

## Debt
- CPU is still about 290% instantaneous during a live week. A dump at 3 min shows `Scopes → ReadNativeBase →
  governance.Replay` over 17k records every 1 s, and `DueWorker.ProcessDueReserves → NativeReserveCheckReading`
  replaying the same ledger. Fix candidates: gate the producer's pass on the history revision plus params and
  as-of (needs a design call on as-of freshness), or keep governance replay incremental. Stat approvals
  also deserve a compaction/snapshot policy (they are about 99% of the ledger).
- The engine skip is silent; a one-line "engine not due" log would have saved this investigation.
- The Home right tile column is still clipped under the right inspector. `.hb .work{grid-template-columns:minmax(0,1fr)}`
  did not fix it, so it was reverted. Not cheap; needs layout work.
