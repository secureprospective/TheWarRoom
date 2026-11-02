# APP-STALL-4 — handoff (2026-11-02)

Branch `build/STALL-4` off `build/INTEGRATE-1102` @ c6bd263a. **Result: the pile-up is fixed (bounded memory,
39 goroutines, no stacked readings), but the screens still FAIL. Readings now finish, and every scoring
reading fails with a scoring error that the stall had been hiding (below).**

## Changed
**A — server single-flight** (`internal/store/history/reading_flight.go`, `internal/commissioner/reading_view_key.go`)
- `shareReading` joins concurrent identical `NativeResults`, `NativeStandings`, `NativeOdds`, `NativeBracket`,
  `StatQueue` and the rookie-draft `FinalStandings`. Key = kind + args + ledger mark (len + last entry) +
  `commissioner.ReadingViewKey(ctx)`. The odds key includes a sha256 of the supplied values. The ledger mark
  means a read made after a write never joins a flight that started before it.
- `ReadingViewKey`: recipient (principal, at) plus the principals-map pointer. A hosted request (credential
  admission) or a token-audit request is never shared.
- Joiners get the flight's result and clock instant. The flight is detached from any one caller's
  cancellation and is cancelled when its last waiter leaves. Nothing is kept after completion.
- The bodies are unchanged (renamed `read*`). No scoring code was touched.

**C — frontend no-overlap** (`frontend/src/app/data/query.ts`, the one shared helper)
- `joinRead`: reads of the same reader and scope never overlap. A request that arrives while a read is
  running queues at most one read after it (`overlap: 'after'`, for change events), and later requests join
  that queued read. Mounts and polls use `'join'`. The map is shared across query instances and holds
  nothing once reads settle.
- `poll(read, ms)`: the next read is scheduled from completion, and unmount stops it. Used by
  `startLivePolling` (HQ/Pulse live, Now update) and the draft room. A visibility change joins the running look.
- `useHQRead` keeps showing the last value for the same scope while it re-reads, so a slow read no longer
  blanks HQ.
- Three tests asserted the old overlapping contract (a fast second read overtaking a slow first one):
  `lineups.test`, `FranchiseHQ.test` and `now.test`. I rewrote them to the new contract: the change is read
  after the slow read, and the newer reading wins. `livegame.test` now uses async timers.
- Held-season key narrowing (debt #2): **not done**, because it was not the bottleneck in this run.

## Desktop (`make build`, copied profile, DISPLAY=:5, 1920x1080, every screen every ~45 s, 10 min)
| t | 30s | 60 | 90 | 120 | 180 | 240 | 300 | 360 | 420 | 480 | 540 | 600 (nav stopped) |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CPU % | 169 | 760 | 780 | 758 | 779 | 789 | 790 | 787 | 775 | 720 | 647 | 57 |
| RSS MB | 1473 | 2730 | 2660 | 3319 | 3966 | 3626 | 3417 | 2973 | 2869 | 2779 | 2892 | 1569 |

Idle, 20 s after arriving: Pulse 132% / 1510 MB, HQ 396% / 1699 MB. STALL-3 for comparison: 6.4 GB and still
climbing at 3.5 min. RSS is now bounded and falls back, but peaks pass 2.5 GB. CPU stays high while
navigating because each visit legitimately starts new distinct readings (each one re-scores the season).
Goroutine dump at 10 min (`~/scratch/stall4/dump1.log`): 39 goroutines and 0 scoring readings in flight. The
hot path was `nativeRecords → cachedTransactionRecords → deepClone` (the ledger is cloned per reading).

## Per screen (`~/scratch/stall4-shots/`)
- Home, War Room, Control: render. **PASS**.
- HQ lineup/bench/this-week: "Not available yet". **FAIL**.
- Pulse Now / Race standings + playoff odds: **FAIL**, but the reading now completes and returns
  `score Stevens, Drew: read native score: count held player statistics: pat_missed_or_blocked is counted
  from its parts; do not record it` (`internal/fantasyscore/kinds.go:38`). Held statistics now carry a
  derived stat id that the scoring engine refuses. This is in scoring/data (likely Sol's SCORE-1/2 or the
  held files), so I did not touch it.

## Tests
`go test -list` first. `store/history -race -p 2`: TestShareReadingRunsConcurrentIdenticalReadingsOnce,
TestShareReadingNeverJoinsDifferentViewers, TestShareReadingCancelsOnlyWhenEveryCallerLeaves,
TestSharedStandingsMatchUnsharedReading (byte-identical JSON), and the two HeldSeasonCache equivalence
tests: **6 PASS, 0 FAIL**. Frontend vitest for data/now/shell/draftroom/race: **643 PASS, 0 FAIL** (74 files),
including the new "never overlaps a slow read" and "polls from completion" tests. `tsc` clean.
`golangci-lint` 0 issues on history and commissioner. `make build` passes. No full suite.

## Debt
1. **Blocker:** the `pat_missed_or_blocked` scoring refusal fails every Native* reading on this league.
2. The per-reading ledger deepClone dominates CPU once the pile-up is gone. Share an immutable records
   slice per revision.
3. The held-season key narrowing (STALL-3 debt #2) is still open.
4. Joiners share result slices and maps by reference. The callers are read-only today, which is fine, but
   a mutating caller would corrupt them.
5. A `make build` from a worktree needs its own `node_modules` (`CI=true`). A symlinked one makes pnpm abort.
