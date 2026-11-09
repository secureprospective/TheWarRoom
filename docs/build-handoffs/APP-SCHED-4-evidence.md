# APP-SCHED-4 — partial delivery, 2026-11-01

**Automatic scoring acquisition is extended; complete league scores and parity are NOT delivered.**
No scoring rates, arithmetic, league settings, reported-score substitutes or player exceptions changed.

## Three-line finding

1. Signals acquires nflverse weekly CSVs and DynastyProcess identities; livegame holds ESPN live/detail feeds. Complete play-by-play/Sleeper/advanced-rushing adapters exist in pinned football imports, not the ordinary held-stat path.
2. The copied profile holds 7,447 weekly NFL rows for 2026 W1–7 (latest successful fetch 2026-11-01 07:11 UTC), but no 2026 play-by-play; it does hold older 2025 play-by-play. Part 3 found 630 absent starter player-weeks and 462 unresolved positions across all 224 franchise-weeks.
3. Neither wrong season nor failed weekly acquisition explains this: Signals never acquires play-by-play, and ordinary history reads only weekly files. The profile has run weekly acquisition repeatedly; doing that again cannot supply the missing suite.

## Delivered flow and preserved boundaries

`BackgroundRefresh.Sync` / manual / watched refresh → existing `syncLeagueSchedules` →
`SyncScheduledScoring` (before the reference-source fence, for any published NFL league season) →
existing archived transport + weekly ingestion + validated compressed nflverse plays → existing
load stamps. Signals' ordinary lock serializes shared acquisition. Season requests share cached bodies;
current files are due after the existing 20-hour window. A missing archived transport is refused.

`history.heldSeason` → newest weekly file plus newest usable plays → cloned lines + unchanged
`source.AttachPlays` → combined digest / later source instant → additional completed revision.
Original weekly-only revisions remain unchanged and available. Malformed or coverage-incompatible
plays are skipped for an earlier usable archived copy; if none exists, the read reports the error.
Expanded releases are capped at 256 MiB and network bodies at 64 MiB. No storage schema/dependency added.
Native results and the stat-change queue both see the additional revision. Pinned football behavior stays.

An acquisition-time regression exposed a safety defect: a new complete revision's lack of gaps hid the
**counted older revision's** gaps. The scorer now also retains the selected revision's missing-evidence
labels, without changing any arithmetic, prices or revision selection. Its regression fails without
that fix (falsely says Complete / 1.5 points) and passes with it (Incomplete / partial points only).

**Remaining policy boundary:** `commissioner.countedVersion` defaults to revision 0 and changes through
the ordinary approved stat-change flow. Downloading complete newer evidence does not approve it.
Automatically promoting a newer revision, deleting original revisions or using MFL totals would change
that existing contract; none was done. Head-brain decision needed before claiming automatic complete scoring.

## Copied-profile exercise

SQLite backup API from the explicitly authorized scratch profile to `.tmp/sched4/profile`, all within
this worktree. Never opened the installed profile or launched the production binary. Restored the held
player directory through its existing parser; installed ordinary lineup environment. Ran acquisition,
repeat acquisition, normal launch-sync and native W1–7 result readings. Repeated on final production code.
The one-use composition harness was removed after searching retained callers.

- Cold scoring acquisition: **1.472 s** (weekly file already fresh, plays newly fetched); repeat **0.003 s**.
- Final repeat: **0.003 / 0.002 s**; exactly **one distinct 2026 play-by-play body** after both runs.
- New archived compressed body: **7,433,966 bytes**. History DB **461,660,160 → 469,966,848 bytes**,
  growth **8,306,688 bytes (7.92 MiB)** including schedule/lineup acquisition and repeated fetch logs.
- Whole first proof, including live schedule/lineup acquisition and seven result readings: **126.83 s**;
  repeated final proof **39.32 s**. These are NOT page-read latency measurements. Overall sync speed is
  not certified fast; the added scoring fetch/cache is fast on this copy.
- All **148** previously recorded franchise-weeks remain recorded; 76 roster-incompatible lineups remain
  refused by the ordinary check, excluded from the comparison. No roster bypass or historical fiction.

## Recorded-lineup-only comparison

“Within” excludes exact; “differ” requires a complete total. Missing totals never become zero.

| Week | Recorded | Before exact / within .01 / differ / unavailable | After exact / within .01 / differ / unavailable |
|---|---:|---|---|
| 1 | 15 | 0 / 0 / 0 / 15 | 0 / 0 / 0 / 15 |
| 2 | 15 | 0 / 0 / 0 / 15 | 0 / 0 / 0 / 15 |
| 3 | 18 | 0 / 0 / 0 / 18 | 0 / 0 / 0 / 18 |
| 4 | 22 | 0 / 0 / 0 / 22 | 0 / 0 / 0 / 22 |
| 5 | 23 | 0 / 0 / 0 / 23 | 0 / 0 / 0 / 23 |
| 6 | 24 | 0 / 0 / 0 / 24 | 0 / 0 / 0 / 24 |
| 7 | 31 | 0 / 0 / 0 / 31 | 0 / 0 / 0 / 31 |
| Total | 148 | **0 / 0 / 0 / 148** | **0 / 0 / 0 / 148** |

Durable franchise-week detail: `APP-SCHED-4-comparison.tsv`. Before uses part 3's retained readings,
filtered to recorded lineups; after uses actual native results on the copied profile, not a surrogate sum.
There are no complete differences to rank. Largest reported totals among eligible rows, still blocked:

| Week | Franchise | MFL | App | Causes among its starters |
|---|---|---:|---|---|
| 7 | Washington | 341.38 | unavailable | 18 counted revisions lack plays; 3 positions unresolved |
| 7 | Minnesota | 330.20 | unavailable | 18 counted revisions lack plays; 3 positions unresolved |
| 5 | Buffalo | 328.10 | unavailable | 16 counted revisions lack plays; 4 positions; 1 absent week |
| 2 | Buffalo | 306.95 | unavailable | 16 counted revisions lack plays; 4 positions; 1 absent week |
| 5 | San Francisco | 304.81 | unavailable | 19 counted revisions lack plays; 2 positions unresolved |

Across these 148 recorded lineups, final starter readings report **2,304** incomplete counted revisions,
**472** absent player-weeks, **300** unresolved positions, **7** unmatched identities. These count starter
occurrences, not unique player-weeks, and are not comparable directly to part 3's all-starters unique counts.
No numeric/stat-by-stat setting mismatch can yet be established. Settings were not changed.

### Identity finding

The unmatched starter is **0835 — Zion Childress**, W1–7. The held MFL directory lists that custom low ID,
name, position S and DAL. A fresh DynastyProcess download during this investigation lists Zion Childress
under **17372**, GSIS **00-0040044**, not under 0835. This is a **candidate**, not verified identity:
the existing crosswalk deliberately does not infer identity from names or silently alias reused low IDs.
It would be false to call the real NFL player intrinsically unmappable. No verified crosswalk for 0835
was established, and no hard-mapped exception or name-only link was installed. The updated crosswalk
was diagnostic only, not manually ingested as a workaround.

## Verification and limits

- Targeted Go race runs: **623 top-level / 1,004 named PASS**, **9 skipped**, **1 failure**, deduplicated.
  Packages: sources (38 named pass), history (414), source adapter (84), commissioner (421), its market
  subpackage (34), composition (13). No GUI/full verify/independent review.
- The failure is `TestSinglePoolStartupDraftIsUnchanged`: actual digest `8a6936145d37309e`, expected
  `90ca21fd230f0aa6`. Reproduced against a **clean local git archive of starting f5366b23** inside
  `.tmp/sched4/baseline`, with the same digest. Inherited; not silently fixed or suppressed.
- Final focused acquisition/scorer/launch race tests pass. Ordinary native launch cold-acquires both
  weekly data and plays, a second launch makes no additional scoring requests, and held reads expose
  complete evidence plus original incomplete revisions. Covers a native league, not only reference imports.
- Acquisition tests also observe cancellation, failed-season isolation, damaged-body rejection, no fresh
  stamp on invalid data, retry and preservation of the earlier usable plays after a damaged fetch.
- Plain **`make build` PASS**, final production code. No version overrides, production launch or remote git.
- Commit hooks: golangci-lint, gitleaks and ifaceguard pass. Initial hook caught forbidden output in the
  disposable harness; fixed and removed it before successful commit. Initial launch test exposed typed-nil
  transport; the API now requires the archived transport explicitly and refuses nil.
- Whole affected Go files vs f5366b23, physical lines: production **1,172 → 1,356**; tests **260 → 469**.
  No reduction claimed. Raw proof, build and race logs remain under `.tmp/sched4`.

## Debt / required next move

1. Head brain: resolve ordinary counted-revision completeness/approval policy without changing math or
   silently approving corrections. New complete evidence is held and visible, not automatically counted.
2. Complete ordinary acquisition/attachment of Sleeper-owned edge stats and advanced rushing; preserve
   nflverse final authority. Existing ESPN live display polling was not changed or turned into a player-stat
   scoring adapter. The requested full live-to-final scoring suite remains incomplete.
3. Obtain verified identity aliases and historically appropriate true positions from the approved data
   flows. Missing NFL weekly rows remain absence, not fabricated inactivity/zero. None of these gaps fixed.
4. A league with no schedule before sync receives its first schedule during reference backfill; scoring
   acquisition sees that publication on the next ordinary sync. Immediate first-publication acquisition
   is not yet proven. Acquisition errors are logged/retried; source-health UI integration remains debt.
5. Re-run the 148-row comparison only after those gaps are resolved. No complete parity or numerical
   settings findings today. Existing historical-roster refusals, empty-lineup representation and the
   inherited startup draft digest failure remain separate debt.
