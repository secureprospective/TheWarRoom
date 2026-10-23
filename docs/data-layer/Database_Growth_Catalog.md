# Database Growth Catalog

What will make the SQLite files hold up as real leagues fill them and reads become the bottleneck.
A living list: add an entry when a pressure is found, update it when measured. Never guess a
threshold; measure it with the opt-in big-league fixture against a read time budget.

Settings today (`internal/db/pools.go:24`): WAL, one writer and up to 10 readers per file,
`foreign_keys(ON)`, `recursive_triggers(ON)`, `synchronous(FULL)`. 51 production tables, 6 named indexes.

Measured 2026-10-22 on a copy of the real `history.db` (375 MB): 179 archived bodies, 460 MB
unzipped in total, the largest 97.95 MB (nflverse play-by-play 2025 CSV).

| # | Pressure | Symptom | How to measure | Trigger | Smallest fix | Status |
|---|---|---|---|---|---|---|
| D1 | Read speed | Some native reads re-parse the whole ledger; cached native reads instead clone/fold it | `make bench-league`: `BenchmarkLeague/{legacy,double}/D1People` (`CommissionerPeople`, full decode) | a read over its budget | keep a folded state per league and fold only new events | open (store F12, partial) |
| D2 | Read speed | Scoring reads/decompresses held season CSVs for each starter; parsing is cached by body hash | `make bench-league`: `BenchmarkLeague/{legacy,double}/D2WeekWarm` and `D2WeekCold` | a week's scoring over budget | parse each distinct copy once and key it by sha256 | open (F13, partial) |
| D3 | Read speed | Archive lookup is a substring scan over `fetch_log.url` | `make bench-league`: `BenchmarkLeagueArchive/{legacy,double}/D3{NoMatch,FirstAccept,RejectAll}`; prints `EXPLAIN QUERY PLAN` for the production constant | fallback reads over budget | match on an indexed source or URL key | open (F14, partial) |
| D4 | Growth | Contract previews seed one franchise per generation in `whatif.db`; old generations remain | `make bench-league`: `BenchmarkLeagueWhatIf/{legacy,double}/D4Season` (734 previews, file bytes and normalized rows) | file growth per week | delete sessions nothing references | open (F15) |
| D5 | Integrity | Native scores depend on which archived copy of a stats file was read | replay a past week after removing one copy | any change in a past result | pin the copy each scored week used | open (F9) |
| D6 | Growth | `history.db` holds every fetch body for good (224 MB of archive) | `raw_archive` count and sum of `size` per month | file size or backup time over budget | a retention rule for bodies nothing pins | open (new) |
| D7 | Integrity | An archived body unzipped with no limit | — | — | `archive.Gunzip(body, size)`: the body must match its recorded size, with a 256 MiB ceiling | fixed on `session/hardening-method` (83070a6a); all 179 real rows match |
| D8 | Many leagues | One file serves every league; one writer per file | write wait time with several active leagues | writes queue past budget | decide one file vs one per league at the client/server panel | open (panel) |

Fixed earlier (in source on main `d3dabe7a`): F1–F8, F10, F11. See `docs/build-handoffs/Hardening_Plan_2026-10.md`.


## Kept generated fixture

`make bench-league` enables the `leaguebench` build tag, `-benchmem`, and `-count=5`.
Neither `make test`, `make verify`, nor default CI enables it. No database binary is committed.
Helpers live in existing test packages (`internal/store/history`, `internal/app/contracts`);
placement is package-based, so no new SYSTEM_MAP/placement row is needed.
Native fixture files are reused across calibration/count repetitions within each benchmark binary.
The target prints its scratch directory and exports native final snapshots for the D4 binary.

The census is Claude's measurement of the real MFL archive's per-season `transactions.json`
and `draftResults.json`, supplied in the 2026-10-23 operator brief, **not** the old history DB.
`bigleague_volume_test.go` retains the per-season census and separate moved-player counts.
Legacy is 32 franchises, 2013–2026; double continues that same league through 2040, repeating
14 seasons' move demand. Every 2026 category is rounded independently after multiplication
by 1500/809. Unknown rookie classes use 170 picks. Startup's 1,696 players are seeded once;
the repeated 2013 season uses a normal 170-player class, not another startup.

Native writes use `ExecuteCommissionerWithFacts`, recorded owner seats, deterministic request
IDs and UTC clocks, and production computed outcomes. There are no raw ledger INSERTs.
The smoke test builds twice, compares serialized native event bytes, and exercises rollover,
cut/claim/award, bid/award, offer/accept/review, decline/withdraw, and draft assignment/insertion.
This is deterministic **native event content**, not a promise of identical SQLite files:
registry initialization and production what-if timestamps are not fixed by the native clock.

### Explicit workload assumptions

- **ASSUMED:** the brief supplies combined WAIVER/FREE_AGENT rows but no separate type counts.
  Half the player adds, bounded by available measured drops, use waiver claims; the rest use
  free-agent bids. Every measured drop is a separate `roster.cut`; every window is processed.
  Both add paths therefore append their real claim/bid and processing events, not one invented
  event per MFL row. No claim is made that this is the measured WAIVER/FREE_AGENT proportion.
- **ASSUMED:** trade assets are players, evenly divided across completed trades; review is by
  the commissioner (offer → accept → review). Lifecycle responses get independent legal
  offers; when responses outnumber proposals, supplementary offers are necessary. Acceptance
  does not move assets again without review. Expiry has no native append: unanswered offers
  expire by the production read-time clock after eight days.
- **ASSUMED:** native commissioner `draft.assign`/`draft.insert` represent rookie selections,
  with the real slot-pricing path. A veteran startup is a single held snapshot, not 1,696
  native rookie-pick commands. Synthetic rosters allow 0–10,000 players and a large cap so
  uncorrelated historical move counts can execute without inventing cuts to make space.
  Seed players have flat $1M contracts ending one year after the tier's final season; new bids
  use one-year $1M contracts. Player identities and positions are synthetic, not real rosters.
- **COUNTED AND SKIPPED:** legacy full-season IR 954, TAXI 661, LOAD_ROSTERS 1,590, plus
  PROCESS_WAIVERS/AUTO_PROCESS_WAIVERS 23. Double repeats these counts (1,908 / 1,322 /
  3,180 / 46). There is no direct native equivalent. Independent 2026 category rounding is
  retained rather than inventing an additional transaction to make rounded columns sum.
- **SYNTHETIC D2:** 32 × 20 distinct starters; each stats body has 18 weeks (11,520 rows).
  Legacy holds 24 fetches / 9 distinct bodies; double 48 / 18. All rows use an existing LB
  CSV template and validated production parsing; missing play-by-play stays missing, so the
  benchmark measures a real incomplete scoring pass, not fabricated complete game results.
  Injected lineup sources follow the existing native-results test seam; identity, archive,
  scoring, schedule and rules reads are production paths. Warm has cached CSV parsing; cold
  constructs a fresh Store, so ledger decoding and CSV parsing are both cold, pools warm.
- **SYNTHETIC D3:** exactly 10,000 fetches / 2,000 bodies and 20,000 / 4,000. Bodies are about
  550 bytes; 10% of URLs match. **ASSUMED URL shape:** registered nflverse weekly URLs with
  synthetic query suffixes, not a new inspection of the real archive copy. These results
  measure lookup density, not the cost of decompressing the real 97.95 MB maximum body.
- **ASSUMED D4:** N is a full 2026 season's player adds + completed trades, 532 + 202 = 734
  in each tier. Rotate franchises, filter the baseline roster exactly as contract previews
  do, retain current-season league adjustments, call `LoadWhatIf`, then `ProjectContract`
  for a $10k increase to one current-year salary cell. This measures the preview's durable
  write slice, not its read-only view rendering. No preview session is saved. Production
  seeding itself gives expired contracts a current-season paid cell. File size is SQLite
  allocated pages after WAL checkpoint; row metrics name all eight populated/pertinent tables.

## Baselines 2026-10-23

Machine: **Beelink** (`com`), Go 1.26.4, linux/amd64, GOMAXPROCS=8.
D3 below is the median of five samples with `-benchmem`. **Provisional:** native fixture generation
ran concurrently; repeat on an otherwise idle machine before setting budgets. Ratios
are double / legacy. Trigger cells above remain Christopher's budget decisions.

| Item | Legacy ns/op | Double ns/op | Time ratio | Legacy B/op | Double B/op | Legacy allocs/op | Double allocs/op |
|---|---:|---:|---:|---:|---:|---:|---:|
| D1People | UNVERIFIED | UNVERIFIED | — | — | — | — | — |
| D2WeekWarm | UNVERIFIED | UNVERIFIED | — | — | — | — | — |
| D2WeekCold | UNVERIFIED | UNVERIFIED | — | — | — | — | — |
| D3NoMatch (SYNTHETIC) | 4,151,662 | 8,730,549 | 2.103 | 744 | 744 | 24 | 24 |
| D3FirstAccept (SYNTHETIC) | 6,494,551 | 13,556,001 | 2.087 | 43,784 | 43,784 | 48 | 48 |
| D3RejectAll (SYNTHETIC) | 17,984,820 | 33,693,762 | 1.873 | 43,057,138 | 86,113,458 | 25,026 | 50,027 |
| D4Season (734 previews/op) | UNVERIFIED | UNVERIFIED | — | — | — | — | — |

D3 byte/allocation growth: NoMatch 1.000/1.000; FirstAccept 1.000/1.000;
RejectAll 2.000/1.999. Query plan is identical for both tiers:

```text
SCAN f
SEARCH r USING INDEX sqlite_autoindex_raw_archive_1 (sha256=?)
USE TEMP B-TREE FOR ORDER BY
```

Full-tier generation times and D4 retained-byte/row totals remain **UNVERIFIED** until the
complete `make bench-league` run finishes. Do not replace missing measurements with estimates.
The two-season smoke proves identical 266-event ledgers (2,459,787 serialized bytes each),
not full-tier determinism. Full successful fixtures must have 37,606 / 75,341 native
events including the two scoring-schedule commands. The separate D2 smoke scored 640 starters twice and observed exactly
9 cached CSV bodies after both the cold and warm pass; runtime was 194.09 seconds under load.
