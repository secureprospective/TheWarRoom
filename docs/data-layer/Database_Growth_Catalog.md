# Database Growth Catalog

What will make the SQLite files hold up as real leagues fill them and reads become the bottleneck.
A living list: add an entry when a pressure is found, update it when measured. Never guess a
threshold; measure it with the opt-in big-league fixture against the budgets below.

Settings today (`internal/db/pools.go:24`): WAL, one writer and up to 10 readers per file,
`foreign_keys(ON)`, `recursive_triggers(ON)`, `synchronous(FULL)`. 51 production tables, 6 named indexes.

Measured 2026-10-22 on a copy of the real `history.db` (375 MB): 179 archived bodies, 460 MB
unzipped in total, the largest 97.95 MB (nflverse play-by-play 2025 CSV).

**Budgets (Christopher, 2026-10-23):** a page opens in under 1 s; a save finishes in under 0.5 s.

| # | Pressure | Symptom | How to measure | Trigger | Smallest fix | Status |
|---|---|---|---|---|---|---|
| D1 | Read speed | Some native reads re-parse the whole ledger; cached native reads instead clone/fold it | `BenchmarkLeague/thick3/D1People`; growth `BenchmarkLeaguePeopleGrowth` | page > 1 s | keep a folded state per league and fold only new events | open: 0.49 s on thick3; passes 1 s until ~10k records (store F12, partial) |
| D2 | Read speed | One week of results re-read every held stats copy, scanned its rows and replayed the ledger for each starter | `BenchmarkLeague/thick3/D2WeekWarm` and `D2WeekCold` | page > 1 s | read and index the season's files once per reading; fold the ledger once | fixed on `session/weekly-results-speed`: 81 s → 0.95 s warm |
| D3 | Read speed | Archive lookup is a substring scan over `fetch_log.url` | `BenchmarkLeagueArchive/thick3/D3{NoMatch,FirstAccept,RejectAll}`; prints `EXPLAIN QUERY PLAN` | page > 1 s | match on an indexed source or URL key | open: 3–17 ms, well inside budget (F14, partial) |
| D4 | Growth | Contract previews seed one franchise per generation in `whatif.db`; old generations remain | `BenchmarkLeagueWhatIf/thick3/D4Season` (735 previews, file bytes and rows) | file growth per week | delete sessions nothing references | open: 15 ms a preview, 190 MB after a season (F15) |
| D5 | Integrity | Native scores depend on which archived copy of a stats file was read | replay a past week after removing one copy | any change in a past result | pin the copy each scored week used | open (F9) |
| D6 | Growth | `history.db` holds every fetch body for good (224 MB of archive) | `raw_archive` count and sum of `size` per month | file size or backup time over budget | a retention rule for bodies nothing pins | open (new) |
| D7 | Integrity | An archived body unzipped with no limit | — | — | `archive.Gunzip(body, size)`: the body must match its recorded size, with a 256 MiB ceiling | fixed on `session/hardening-method` (83070a6a); all 179 real rows match |
| D8 | Many leagues | One file serves every league; one writer per file | write wait time with several active leagues | writes queue past budget | decide one file vs one per league at the client/server panel | open (panel) |
| D9 | Write growth | Every save re-reads and replays all records (`internal/store/history/commissioner.go:105–118`) | `BenchmarkLeagueWrite/records={1000…50000}`; `make bench-league-build` prints seconds per season | save > 0.5 s | close the books each season: a season-summary event carries state forward; a differential test proves it decides exactly as full replay, with full replay as the fallback | open: fails budget past ~3k records (about one season) |

Fixed earlier (in source on main `d3dabe7a`): F1–F8, F10, F11. See `docs/build-handoffs/Hardening_Plan_2026-10.md`.

## Kept generated fixture

`make bench-league` (tag `leaguebench`, `-benchmem`, `-count=5`) builds the fixture if missing, then runs
every bench. `make test`, `make verify` and CI never enable the tag. No database binary is committed.

- **thick3:** 32 franchises; a seed-only 2021 startup (1,696 players), then the three busiest real seasons,
  2022–2024 (1,483 + 1,680 + 1,534 MFL transactions). 7,398 native events; builds in ~35 min.
- **Cache:** `~/logs/TheWarRoom/bigleague-fixtures/thick3-<key>.db`; the key is SHA256 over every
  `history/bigleague*_test.go`, so editing any of them rebuilds. An exclusive lock stops concurrent builds.
- **Growth benches** (`BenchmarkLeagueWrite`, `BenchmarkLeaguePeopleGrowth`) seed 1k–50k records in one
  transaction; `TestBigLeagueWriteSeed` proves the seed byte-equal to fully authorized commands.
- Every fixture move goes through `ExecuteCommissionerWithFacts` with production checks; no raw INSERTs.

### Assumptions

- **ASSUMED:** MFL's combined waiver/free-agent rows split half claims, half bids; each drop is a cut.
- **ASSUMED:** trade assets are players, spread evenly; the commissioner reviews (offer → accept → review).
- **ASSUMED:** rookie picks use `draft.assign`/`draft.insert`; rosters and cap are relaxed so historical
  move counts run without invented cuts. Players and positions are synthetic.
- **SKIPPED:** IR, TAXI, LOAD_ROSTERS and waiver-processing rows have no native equivalent.
- **SYNTHETIC D2:** 640 starters; 24 held copies of the season stats file (9 distinct), 11,520 rows each.
  D2 cost scales with the copies held; the real count per season is not yet measured.
- **SYNTHETIC D3:** 10,000 fetches / 2,000 bodies of ~550 bytes; URL shape assumed.
- **ASSUMED D4:** a season's what-ifs = 2023 player adds + trades (419 + 316 = 735).
- **ASSUMED D9:** the timed save is a `rules.set cap.amount`, the simplest legal write.

## Baselines 2026-10-23

Beelink (Ryzen 9 6900HX, 16 threads), Go 1.26.10, idle machine. Medians of five runs.

**thick3 (7,398 records)**

| Item | Time | Memory per run |
|---|---:|---:|
| D1 people page | 0.49 s | 0.38 GB |
| D2 week results, warm (before fix) | 81 s | 180 GB |
| D2 week results, cold (before fix) | 86 s | 183 GB |
| D2 week results, warm (after fix) | 0.95 s | 0.81 GB |
| D2 week results, cold (after fix) | 4.1 s | 3.6 GB |
| D3 no match / first accept / reject all | 2.8 / 4.4 / 16.7 ms | 744 B / 44 KB / 43 MB |
| D4 a season of what-ifs (735) | 11.2 s (15 ms each) | 0.77 GB; file 190 MB |

D2 cold includes parsing nine distinct CSV bodies once after a restart; later readings are warm.
D3 plan: `SCAN f` · `SEARCH r USING INDEX sqlite_autoindex_raw_archive_1 (sha256=?)` · `USE TEMP B-TREE FOR ORDER BY`.

**Growth (records in the ledger)**

| Records | One save (D9) | People page (D1) |
|---:|---:|---:|
| 1,000 | 0.15 s | 0.11 s |
| 5,000 | 0.69 s | 0.48 s |
| 10,000 | 1.56 s | 0.87 s |
| 20,000 | 2.95 s | 1.71 s |
| 50,000 | 7.14 s (17 GB) | 3.93 s (10 GB) |

Both are linear: time doubles as records double. A real league adds about 2,500 records a season.
