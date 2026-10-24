# Client/server 12 — memory replay cache

## Flow and bounds

Service → existing history read/append transaction → snapshot's league revision → decoded
records and immutable replay memo → shared commissioner folds → authorized reading.
Credential admission and domain authorization still run on every request; no response,
identity policy, clock-dependent reading or evidence result is cached.

`history.cacheRecords` retains at most four leagues and 100,000 records **in total**, one
revision per league. Replacing a revision releases its old memo. Crossing either budget
clears retained entries; a single oversized revision works without retention. This is a
record-count bound, not a promise of an exact heap-byte ceiling for arbitrary payloads.
The existing decoded-record cache owns the memo, avoiding a second service cache/lifetime.

`commissioner.Replay` checks the complete compensation inputs, record count and revisions
before reusing a projection. Edited copies, trial appends, older SQLite snapshots and
corrections therefore replay or rebuild. Returned active maps are caller-owned. Draft
restore targets and seasons are included. Explicit authoritative replay clears the memo
along with closed books. Private fields never serialize. No schema or stored snapshot was
added; existing frozen rollover books remain unchanged.

## Exercised

- `go test -race ./internal/commissioner ./internal/store/history ./internal/leagueservice`
- `go test ./...` passes (the first shell invocation timed out; the complete rerun passes).
- `TestReplayCacheOwnershipAndInvalidation`: independent returned maps, modified records,
  truncation, append and concurrent reads.
- `TestReplayCacheTracksDraftRestoreSeason` and
  `TestReplayCacheCorrectionAndReversalOfReversal`: restore/correction target changes and
  reversal chains match full replay.
- `TestReplayCacheCorrectionsRolloverRestart`: cold and warm projections match authoritative
  events before/after rule compensation, rollover and physical database reopen; whole-DB
  fingerprints prove reads do not persist snapshots.
- `TestServiceReplayCacheColdWarmAndRestart`: real service/store ledger reads match direct
  engine reads after append, compensation and reopen.
- `TestReplayCacheRetentionBound`: league/record eviction and oversized bypass.
- Existing transaction-snapshot, admission, scoring and rollover tests remain in the targeted
  package batch. `make lint` passes with a worktree-local `GOLANGCI_LINT_CACHE`; shared cached
  analysis initially reported paths to deleted worktrees.

## 100k-event measurement

Command (Go 1.26.10, linux/amd64, Ryzen 9 6900HX, GOMAXPROCS=8, GOMEMLIMIT=1500MiB):

```sh
go test -tags leaguebench ./internal/store/history -run '^$' \
  -bench '^BenchmarkReplayCache100k$' -benchtime=20x -count=1
```

The existing bulk serializer creates exactly 100,000 valid rule-traffic events in a
throwaway database. Pools are physically closed/reopened. Restart includes open,
initialization and the first authoritative decode/cache build; OS filesystem caches are
not flushed. Warm samples include a real store transaction, caller-owned record clone and
`Replay`. Cold/full and cached projections must agree before timing.

| Measurement | Observed |
|---|---:|
| Warm mean | 248.4 ms/read |
| Warm p95 (20 samples) | 270.2 ms/read |
| Reopen and rebuild | 6.861 s |
| Retained heap delta after GC | 146,098,400 B |
| Allocated per warm read | 76,373,105 B; 200,531 allocations |

Retained heap includes decoded records, memo and reopened store, not just projection maps.
These are local synthetic measurements, not target-CT, HTTPS latency, a full-season scoring
benchmark, or a process cold-boot guarantee. The remaining clone/allocation cost is explicit;
this slice does not change ownership semantics to eliminate it. No evidence here justifies
additional persisted snapshots or declares their separate decision resolved.

## Plan adjustment

The cache lives at the shared history/replay boundary rather than wrapping service replies:
all existing service host methods already pass through it, including queued transaction
checks. This preserves current authorization/time/evidence checks without duplicating the
service read surface. No new package, placement/map change or binding change is needed.
Affected handwritten production files total 1,143 → 1,243 physical lines against the starting
HEAD, including both new cache files; tests/docs are excluded. This adds a bounded memo owner
and its invalidation contract rather than claiming a source-size reduction.
