# Memory debloat log — APP-MEM-1

**Measurement only; no memory fixes authorized or applied.** Locked screens precede cleanup.
Base: `4cd127dd`, `build/INTEGRATE-1102`; isolated branch `build/MEM-1`.
Production `CI=true make build`, Go 1.26.10, WebKitGTK 4.1, display :5.
Evidence root: `~/scratch/mem1-profiles/` (private, intentionally not committed).
All MB below are **MiB** (pprof's MB = 2^20 bytes), sampled estimates, not database sizes.

## Method and limits

Copied config and data with `cp -a --reflink=auto` into one disposable profile;
original files were never opened by this desktop. Recreated that profile from the same source
for the limit run. Checked `pgrep -x thewarroom` before each launch; stopped only the launched
PID, via the runner's `$!`. Home remained open for five minutes before navigation.
`baseline/rss.csv` samples desktop PID RSS and VmHWM every 5 s; child WebKit processes are
**not included**. The opt-in diagnostic writes twenty heap profiles and twenty MemStats rows,
every 15 s through minute five. Each profile contains both in-use and cumulative-allocation
samples. No forced GC, memory limit, cache policy, scoring or result changes in application code.

Heap profiles report allocation stacks, **not a retaining-object graph**; live cache attribution
combines stacks, source inspection and persistence in later profiles. GC accounting can lag by two
cycles; MemStats HeapAlloc and pprof inuse_space need not match. A 15-second sample cannot identify
the precise subsecond RSS high-water allocation. Do not subtract the profile total from VmHWM and
call the result native memory. Tests/lint and other fleet work shared the host during these runs;
wall-time comparisons are observational, not a controlled benchmark.

## Startup peak and heap/RSS gap

- First five minutes: sampled RSS peak **2,305.1 MiB at 40 s**, VmHWM **2,415.7 MiB**.
- Closest post-peak heap: `baseline/heap-003.pprof` (45 s), **715.6 MiB** sampled in-use,
  **11,822.4 MiB cumulative allocations**. Same-time RSS **2,085.7 MiB**.
- At that sample MemStats: HeapAlloc **826.0**, HeapInuse **860.6**, HeapIdle **1,286.1**,
  HeapReleased **340.7**, Sys **2,201.3 MiB**, NumGC **106**.
  RSS minus HeapInuse is **1,225.1 MiB**: **945.4 MiB idle but unreleased heap** explains most;
  another **279.7 MiB** lies outside that heap-residency approximation (runtime metadata/stacks,
  SQLite/native GTK/WebKit mappings, shared/file pages). Released/idle accounting is not an exact
  OS residency map. `baseline/smaps-steady.txt` supplies a later OS cross-check, not peak proof.
- Largest sampled in-use profile: `baseline/heap-005.pprof` (75 s), **876.1 MiB**; instantaneous
  HeapAlloc **1,318.0 MiB** illustrates the GC/sample lag. Allocation churn already totals
  **44,621.7 MiB**. Do not confuse this later heap peak with the earlier RSS peak.
- At five minutes: `baseline/heap-020.pprof`, **607.3 MiB** sampled in-use; HeapAlloc **706.3**,
  HeapInuse **794.3**, idle **1,354.2**, released **893.0 MiB**, NumGC **201**;
  RSS around **1,483.4 MiB**, including approximately **461.3 MiB** idle/unreleased heap.
- Subsequent screen traversal reached sampled RSS **2,875.0 MiB**, VmHWM **3,051.3 MiB**.
  This is **outside the requested five-minute profiling window**; no profile proves its owners.
  Navigation readings/ledger projections are candidates, not an established attribution.
- This copy did **not** reproduce the old “2262 players valued” engine run: startup logged
  “engine: not due” after unchanged signals. These numbers establish this real-profile launch,
  not the old OOM's exact allocation path.

## Causes ranked by retained MB near startup RSS peak

Evidence filenames are relative to the evidence root. Values refer to `baseline/heap-003.pprof`
unless another sample is named. Cumulative call-tree figures include children; rows explicitly
separate children to avoid adding the same bytes twice. Suggested fixes are deferred hypotheses.

| Cause / lifetime | Where (file:function) | MiB near RSS peak; later evidence | Profile evidence | Suggested preserving fix | Risk / preservation check |
|---|---|---:|---|---|---|
| Per-player/week stat maps, labels and kick-event slices; **steady** | `internal/fantasyscore/source/weekly.go:RawWeek.Convert` | **302.9 flat**; still 302.9 at 300 s. Quantities insertion 185.3; combined event/label construction 75.5; distance slice 38.0 | `baseline/heap-003.pprof`, `list-003-RawWeek.Convert.txt`, `top-020.txt` | Compact fixed stat storage; avoid duplicating immutable display labels per week; retain exact zero/missing and distance semantics | Medium/high: golden scoring, kicking distances, missing/zero and mutation isolation |
| Whole historical depth materialization; **startup temporary** | `internal/fantasyscore/source/depth.go:ReadDepthHistory`, called by `internal/store/history/player_absence_roster.go:heldDepthHistory` | **154.3 cumulative**, including 80.8 row slice, 49.0 Parquet string assignment and reflect allocations; absent from leading steady owners | `baseline/heap-003.pprof`, `list-003-ReadDepthHistory.txt` | Consume existing bounded `ScanDepthHistory` into only required dated snapshots, rather than retaining every row | High: historical membership, seven-day cutoff, complete-team coverage and no future affiliation leakage |
| Per-game full-roster absence expansion; **startup temporary** | `internal/store/history/player_absence_roster.go:heldNonParticipation` | **56.3 flat**, 55.2 at roster append; 218.1 cumulative includes depth above, so **do not add 218.1** | `baseline/heap-003.pprof`, `list-003-heldNonParticipation.txt` | Derive absence proofs per relevant game/week with bounded intermediate roster storage | High: no fabricated zeros, bye/unrostered evidence and snapshots must be identical |
| Independent whole-ledger Record clones; **mixed steady + transient** | `internal/commissioner/replay_cache.go:WithReplayCache` and clone callers (`slices.Clone[Record]`) | **47.1 flat** across Record clone stacks; 58.9 at 75 s, 11.8 at 300 s. Generic clone attribution is shared, not all charged to WithReplayCache | `baseline/heap-003.pprof`, `list-005-slices.Clone.txt`, `top-020.txt` | Reduce independently copied outer record arrays where revision ownership can safely be shared | High: caller mutation isolation, replay invalidation and bounded revision retention |
| Decoded league-event stream; **steady** | `internal/store/history/events.go:scanEvents`, `internal/store/history/events_cache.go:wholeStream` | **43.2 cumulative scanEvents** (includes 10.5 direct, JSON strings and decode structures); persists at 300 s | `baseline/heap-003.pprof`, `top-020.txt` | Share immutable decoded revisions / smaller indexes; preserve deep-copy guarantees at mutable edges | High: event ordering, rollback/restore detection, concurrency and cache bounds |
| CSV record strings retained by decoded facts; **steady** | `internal/fantasyscore/source/weekly.go:readRecords` → `encoding/csv.(*Reader).readRecord` | **42.5 flat**, 42.0 at 300 s | `baseline/heap-003.pprof`, `top-020.txt` | Measure backing-string retention before selecting fields/copying identifiers; discard nonneeded fields after validation | Medium: complete source validation, identifiers, precedence and archived evidence |
| PlayerWeek outer slices; **steady** | `internal/fantasyscore/source/weekly.go:ReadWeeks.func1` | **17.5 flat** (320.3 cumulative includes 302.9 stat maps above) | `baseline/heap-003.pprof`, `top-020.txt` | Bound needed seasons/weeks only when all consumers' histories are proven; compact per-week representation | High: historical scoring, dynasty inputs and evidence availability |
| Time-filtered ledger copies; **startup/read transient** | `internal/commissioner/rules_season.go:RecordsThrough` | Not leading at RSS-nearest sample; **106.0 flat at later heap peak** | `baseline/heap-005.pprof`, `list-005-RecordsThrough.txt` | Reuse safely immutable prefixes or an index for time-cutoff reads instead of repeated whole-record copies | High: restore/correction chronology, stable ordering and current-vs-historical rules |
| Explicit recorded-zero stat maps; **steady** | `internal/fantasyscore/source/absence.go:emptyRecordedLine` | Arrives after RSS peak: **49.1 at 75 s and 300 s** | `baseline/heap-005.pprof`, `top-020.txt` | Compact zero representation distinct from missing evidence, or immutable shared data with copy-on-write | High: evidence-backed zero must never become missing or permit shared mutation |
| Play augmentation maps; **steady** | `internal/fantasyscore/source/plays.go:AttachPlays` | Arrives later: **25.1 at 75 s and 300 s** | `baseline/heap-005.pprof`, `top-020.txt` | Compact representation aligned with weekly stat storage; retain event detail | High: credited plays, blocked kicks, duplicate sources and golden scores |

### Allocation churn (not additional retained memory)

`baseline/alloc-top-005.txt`: **44,621.7 MiB** total allocation by 75 s. Flat owners:
`internal/fantasyscore/source/play_credits.go:creditColumns` **12,941.2 MiB**,
`defensiveCredits` **6,054.2 MiB**; `internal/fantasyscore` `countLine` **3,901.2 MiB**;
`RecordsThrough` **3,628.6 MiB**; `maps.clone` **3,047.4 MiB**.
These are repeated short-lived builders/clones, not 29 GiB concurrently live.
Suggested later investigation: construct per-decode immutable credit tables once, reuse per-score
rule metadata, avoid repeated ledger-prefix copies. Preserve call ownership and all scoring goldens;
no package globals or unbounded memoization. Risk medium/high. Existing `list-005-*` captures
in-use line attribution; `alloc-list-005-*` captures allocation-space attribution for the top three.

## GOMEMLIMIT experiment

Same production binary and rebuilt disposable profile, **one** `GOMEMLIMIT=1500MiB` run:

| Measure | Default | 1500MiB |
|---|---:|---:|
| Startup (0–300 s) sampled peak RSS | 2,305.1 MiB | 1,720.2 MiB |
| Startup VmHWM | 2,415.7 MiB | 1,859.7 MiB |
| Navigation sampled peak RSS | 2,875.0 MiB | 1,749.5 MiB |
| Whole-run VmHWM | 3,051.3 MiB | 1,922.3 MiB |
| RSS at 300 s | 1,483.4 MiB | 1,626.8 MiB |
| Five-minute sampled live heap | 607.3 MiB | 618.8 MiB |
| NumGC at 300 s | 201 | 226 |
| Desktop user + system CPU through 300 s | 233.81 s | 223.17 s |
| Startup season ready (logged wall duration) | 21.459 s | 17.296 s |

Evidence: `limited/rss.csv`, `limited/memstats.csv`, `limited/desktop.log`,
`limited/heap-020.pprof`, `limited/top-020.txt`, and both runs' navigation captures.
Startup high-water fell **556.0 MiB / 23.0%**; whole-run high-water fell **1,128.9 MiB / 37.0%**.
No slowdown observed in startup readiness or aggregate desktop CPU in this single run;
GC count rose by 25 (12.4%). Five-minute RSS was actually **143.4 MiB higher**, so the limit
is not a universal steady-state improvement. Identical live-heap ownership persists.

**A soft Go limit can help transient retained heap/RSS on this copy**, but is not an RSS cap,
does not bound native or child-process memory, and does not remove the retained objects.
Concurrent fleet work, warmed OS caches, fresh live responses and navigation pacing confound
causal timing claims. UI latency was not instrumented; screenshots show routes opened, not a
performance SLA or correct data on every screen. The limit is an experiment environment variable
only, not a shipped default or an applied memory fix.

## Reproduce analysis

```sh
# From the isolated worktree; E is the private evidence root above.
go tool pprof -top -inuse_space build/bin/thewarroom "$E/baseline/heap-003.pprof"
go tool pprof -top -sample_index=alloc_space build/bin/thewarroom "$E/baseline/heap-003.pprof"
go tool pprof -list RawWeek.Convert build/bin/thewarroom "$E/baseline/heap-003.pprof"
go tool pprof -list ReadDepthHistory build/bin/thewarroom "$E/baseline/heap-003.pprof"
go tool pprof -list heldNonParticipation build/bin/thewarroom "$E/baseline/heap-003.pprof"
```

The retained binary and SHA256 are in the evidence root so later source/build changes do not
silently change symbolization. Profiles may contain internal identifiers/function paths; keep private.
