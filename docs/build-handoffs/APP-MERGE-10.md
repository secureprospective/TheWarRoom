# APP-MERGE-10 — desktop heap spike

2026-11-02 · Bee · build/INTEGRATE-1102

**Partial acceptance.** The measured multi-gigabyte depth-decoder spike is repaired in
`001825e9`. The copied desktop survives ten minutes, logs 774 live players and no
`context deadline exceeded`, and War Room's shelf shows player names. **The requested
RSS ceiling does not pass:** several samples exceed 1.5 GB. Do not mark this task complete.

## Cause and evidence

- Two baseline copied-profile launches reproduced termination around 128–143 seconds.
  The first reached 6,307,408 KiB RSS. A temporary loopback-only pprof patch in `main.go`
  captured heaps/stacks; it was removed before the functional commit and final build.
  No memory cap was used.
- At the second spike, `source.ReadDepthHistory → parquet.Read[RawDepth]` accounts for
  **2,816.93 MiB (81.03%) of a 3,476.27 MiB sampled heap**. Its destination slice alone
  accounts for 1,650.63 MiB; parquet row buffers add another 724.59 MiB flat.
  The stack capture contains **21 `TargetResearchPlayer` calls**, decoding the same season.
- Regression trigger: **`cbad237c3`**, Home's FORM read, introduced
  `Promise.all(mineStarters.map(...research.player...))`. Each call reaches
  `PrepareResearchDepth → refreshResearchDepth → ParseDepth`. There was no shared
  in-flight depth acquisition. This is a merged-lane ancestor already present at
  `d809fd833`, not a new decoder change within `d809fd833..73a9fd15`.
- Whole-file decoder origin: `5a1c3299` used `contracts.ReadRows → parquet.Read`;
  `152274c8` moved it into `source.ReadDepthHistory`. That relocation did not introduce
  the allocation strategy. Attribution is from captured stacks and source history,
  not a binary bisect. The five-minute history worker cannot explain the two-minute spike.

## Fix and preserved flow

- `ScanDepthHistory` decodes and validates **256 rows per batch**, not a season-sized
  parquet row buffer. Research retains only the newest dated snapshot and retains its
  existing stable team/formation/slot/rank ordering. Invalid discarded older rows still
  reject the result; malformed input never becomes a successful partial board.
- Historical absence evidence continues to receive every validated dated membership row
  through `ReadDepthHistory`; it is not reduced to today's membership.
- `research.Held.PrepareDepth` joins overlapping player requests to one depth refresh.
  It holds no season/publication lock across I/O. A waiting request can cancel without
  cancelling the producer; completed attempts release the slot for later retries.
- Reviewed callers: Research's refresh and on-demand player reads share the same route;
  held absence evidence uses the historical reader. No credential, evidence-admission,
  scoring, follow-setting, timeout or publication rule was weakened. No dependency added.
- Only straight-line coordination changed in `research_runtime.go`; its source fingerprint
  was updated after reviewing that wiring. Boundarycheck passes with 170 frozen entries.
- Four affected production files: **417 → 479 physical lines**, **13,668 → 15,335 Unicode
  code points**, including LF. Tests/docs counted separately. No file exceeds 600 lines.

## Verification

Targeted race batches, one package at a time, `-p 2`; no full Go suite.

| Check | Result |
|---|---|
| ingestion/researchfeeds | 8 PASS |
| fantasyscore/source | 130 PASS, 4 fixture-gated SKIP |
| app/research | 88 PASS |
| composition Research/Depth selection | 27 PASS |
| Total tests/subtests | **253 PASS, zero failures; 4 SKIP** |
| 100,000-row allocation regression | Before 152,564,976 bytes; after about 68.3 MB; 80 MiB ceiling |
| Plain `make build` | PASS, final functional sources; no diagnostic patch |
| Commit hooks | golangci-lint-full, gitleaks, ifaceguard PASS; no bypass |
| Boundarycheck | PASS, 170 frozen functions/sources |
| `make lint` | **FAIL at inherited bloat ratchet**, described below |

The allocation test failed before the fix (152.6 MB also exceeds the final 80 MiB budget).
Coverage also checks latest-snapshot equivalence to the real CSV, invalid older-row rejection,
full dated membership retention, malformed parquet, coalescing 21 overlapping requests,
waiter cancellation without producer cancellation, and subsequent retry availability.

`make lint` encountered an independently present ignored `.worktrees/odds` tree and reported
provenance 1 / baseline 0 and tiny files 60 / baseline 24. A tracked-source-only comparison
of `73a9fd15` and `001825e9` gives **provenance 0 → 0, tiny files 28 → 28**, comment share
5% → 5%; even the starting tracked tree exceeds the tiny-file baseline. No other agent's
worktree or ratchet was edited. The gate remains failing, not bypassed or called passing.

## Ten-minute copied-profile desktop

Launched 11:54:15 CT using the final functional sources built before their commit (binary
stamp `73a9fd15-dirty`). Only documentation/fingerprint bookkeeping followed the build;
compiled functional source matches `001825e9`. Config/data/cache all point to a disposable
scratch profile copied from APP-MERGE-9's existing copy. Real profile untouched.

RSS is the desktop parent from `ps`, converted to **decimal GB**; WebKit children are not
included. Collected every second, with the requested 30-second samples below. Profiling,
GODEBUG and GOMEMLIMIT were absent. Targeted composition tests and commit hooks overlapped
part of the early measurement; the desktop remained the only task desktop throughout.

| Elapsed | RSS GB |
|---:|---:|
| 0:00 | 0.011 |
| 0:30 | 1.305 |
| 1:00 | 1.894 |
| 1:30 | 1.476 |
| 2:00 | 1.682 |
| 2:30 | 1.853 |
| 3:00 | 1.469 |
| 3:30 | 1.306 |
| 4:00 | 1.507 |
| 4:30 | 1.396 |
| 5:00 | 1.487 |
| 5:30 | 1.362 |
| 6:00 | 1.768 |
| 6:30 | 1.510 |
| 7:00 | 1.386 |
| 7:30 | 1.311 |
| 8:00 | 1.364 |
| 8:30 | 1.304 |
| 9:00 | 1.265 |
| 9:30 | 1.390 |
| 10:00 | 1.342 |

- One-second sampling caught an earlier startup peak of **2,592,948 KiB = 2.655 GB** at
  eight seconds. Even ignoring that transient, the table fails the ceiling.
- Process remained live beyond 10 minutes (observed at 621 seconds); stopped deliberately
  afterward. No task desktop remains running.
- Log: **774 live players from 14 games**, season 2026/week 8; **zero** occurrences of
  `context deadline exceeded`. Signals: 23 files, zero failed. No new engine run was logged
  on this already-refreshed profile; the existing run is not counted as a new one.
- **War Room MY SHELF names PASS**, independently visible: Flowers, Zay; Metellus, Josh;
  Latu, Laiatu; Warren, Tyler; Phillips, Andru. No player-name code change was needed.
  Home also visibly shows named TOP 5 players and live scores.

## Debt / next move

- **RSS acceptance remains open.** Streaming alone removed the fatal spike but still exceeded
  the budget; coalescing improved the run without making every sample fit. Further profiling
  of the repaired build, especially its first 20 seconds and steady-state native reads, is
  required before promising <1.5 GB. Do not substitute a Go memory limit.
- Baseline cumulative profiling also showed substantial churn outside depth acquisition:
  `maps.clone` 73,251 MiB of 179,675 MiB allocated by 120 seconds, with governance/native
  replay, repeated evidence reconstruction and valuation preparation in the callers. This is
  evidence for the next profile, not proof of the repaired build's remaining peak source.
- Existing backup-entry and Wails model-alias warnings remain. Bloat gate debt remains above.
- Local commits only; no pull/push to the bundle remote, no full-suite run, no real profile edits.

Evidence: `.task-evidence/app-merge-10/` (untracked): baseline/fixed logs, one-second RSS,
heap-top reports, failing-before test, targeted JSON results, build/hook/lint outputs.
Raw profiles, stacks and screenshots remain in disposable operator scratch, not git.
