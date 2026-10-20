# HANDOFF

- **Baton:** Claude/reviewer — 2026-10-19; block B phase 7 is partial.
- **Where it stands:** `533f2aa7` removes prefix replay from rollover reads and repeated reversal-chain walks from ledger labels, privacy and permission checks. General rollover selection is O(n log n), not strictly linear. `cc15814e` caches immutable decoded ledger prefixes by league/revision (including transactional append/preview/draft reads), and native league lists until the event-table watermark changes. Time-dependent projections are not cached. `328d9a83` adds exact-URL weekly archive reads; `3bea2774` caches parsed weekly bodies by hash, including failures, and clones public player-week maps.
- **Verification:** unchanged commissioner and history suites pass with `go test -race -count=1` (96.0 s and 109.8 s). Restoring the old ledger walkers fails the new 1,001-entry/two-second budget at 57.95 s. Disabled cache hits, substring exact lookup and broken rollover toggles each fail their regression tests. All commit hooks and scoped golangci-lint pass. `make lint` still stops at the inherited tiny_files ratchet (23 vs 22); baseline untouched. Root unchanged; root suite not run.
- **Replay gate:** `TWR_REPLAY_N=10000 go test -run TestReplayCost -v -timeout=30m ./internal/store/history` passes in 1026.24 s (GOMAXPROCS=8, GOMEMLIMIT=1500MiB). The timeout override is necessary for the total append loop; the plain default-ten-minute invocation is not proven to finish. Steps count reversals; the ledger also contains its creation.

| Step | Append ms | Ledger read ms |
|---:|---:|---:|
| 1000 | 16.860 | 16.017 |
| 2000 | 91.959 | 66.163 |
| 3000 | 91.811 | 96.881 |
| 4000 | 128.072 | 120.956 |
| 5000 | 80.207 | 64.064 |
| 6000 | 82.552 | 67.611 |
| 7000 | 102.118 | 77.214 |
| 8000 | 144.899 | 115.636 |
| 9000 | 151.421 | 100.055 |
| 10000 | 144.388 | 122.656 |

- **Next move:** review shared-read-only cache assumptions and memory growth (no eviction yet), reversal privacy/authorization equivalence, and worst-case rollover complexity. Finish projection caching and unchanged-input hourly work. Both are still outstanding.
- **Blocked on:** phase 3's transactional migration runner is still absent; neither requested index was added. Its prior quarantine/schema decision remains open. F15 is not implemented: simulation rows share append-only ledger triggers with durable state. Approve isolated disposable simulation storage (recommended) versus migration-backed, simulation-only deletion before pruning; preserve published sessions and legacy transaction state either way.
- **Tried/rejected:** no trigger bypass, ad-hoc migration/index DDL, guessed cleanup policy or revision-only cache of time-dependent projections. No production profile, network feed, push, merge or root code changes.
