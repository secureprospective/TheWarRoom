# APP-SCHED-3 — played starters, 2026-11-01

**Partial delivery; score parity NOT established.** No scoring math, settings, standings overrides,
reported-score adapter or storage format changed. Local commits only; no git remote contact.

## Slice and guarantees

`BackgroundRefresh.Sync` / manual and watched reference refresh → `syncLeagueSchedules` →
validated `weeklyresults.Fetch` → importer `BackfillLineups` maps held franchise IDs and player
identities → generic `SyncLineups` → ordinary `LineupCorrect` through `Ledger.Execute` →
store authorization, revision fence, roster and season checks → existing ledger lineup reader →
existing ResultSources/NFL statistics → native results and standings.

Historical sync uses the existing commissioner's correction, not owner impersonation: owner submissions
cannot submit another franchise's lineup or change locked players. Reason and recorded import principal
are retained in the same lineup ledger. No imported-league condition was added to a reader or engine.
The only source-following decision remains in the importer at the acquisition boundary.

Each franchise-week is atomic. Mapping/shape failures occur before any append. Ordinary store refusals
leave that franchise-week unchanged and do not stop independent weeks; returned errors enumerate them.
Unchanged starter sets (order-insensitive) append nothing. Season mismatch is rejected before writes.
Launch reads the held player archive if no target snapshot exists; no extra daily players fetch.
An empty slot (0000) is omitted; an unknown real starter refuses mapping rather than silently disappearing.
A completely empty starting lineup cannot be represented by the existing lineup command and is refused.

Source-traced: callers, ordinary append authorization/roster checks, ledger folding, native result sources.
Exercised: table tests, store integration, simulated launch on a SQLite-backup copy. Independent head-brain
review and GUI click-through have not occurred.

## Copied-profile proof

SQLite backup API, read-only source `~/scratch/cos-profile/TheWarRoom`, destination
`.tmp/sched3/profile` within this worktree. Never opened `~/.config/TheWarRoom`, launched no GUI/binary.
Simulated launch uses the normal background reference sync callback, MFL GET exports and ordinary store
lineup environment. The already held player directory was restored as on an app session; no fabricated
player/stat/lineup source was installed. Clock: 2026-11-01 16:00 UTC.

- Before launch: 2 ledger entries; launch publishes schedule and imports 148 lineups → 152 entries.
- Repeated launch: stays 152 entries. 76 refused franchise-weeks remain unchanged.
- All 112 played games remain undecided; native regular-season standings show 0 decided / 208 pending.
- Bills: app 0–0 / PF 0 (no complete totals), MFL 6–1 / PF 1912.35. **FAIL**, not a valid zero-score comparison.

| Week | MFL franchise-weeks | Lineups recorded | Exactly equal | Within 0.01, not exact | Differ, complete | Unavailable |
|---|---:|---:|---:|---:|---:|---:|
| 1 | 32 | 15 | 0 | 0 | 0 | 32 |
| 2 | 32 | 15 | 0 | 0 | 0 | 32 |
| 3 | 32 | 18 | 0 | 0 | 0 | 32 |
| 4 | 32 | 22 | 0 | 0 | 0 | 32 |
| 5 | 32 | 23 | 0 | 0 | 0 | 32 |
| 6 | 32 | 24 | 0 | 0 | 0 | 32 |
| 7 | 32 | 31 | 0 | 0 | 0 | 32 |
| Total | 224 | 148 | 0 | 0 | 0 | 224 |

Complete per-franchise/week comparison: `APP-SCHED-3-comparison.tsv`. Nulls were never converted to zero.

### Top five

MFL order below is wins then PF calculated from weeklyResults, not a claim about MFL's tie-break ranking.
App order is its actual computed reading, presently tied/unscored.

| Place | App | W–L / PF | MFL (wins then PF) | W–L / PF |
|---|---|---|---|---|
| 1 | Buffalo Bills | 0–0 / 0 | Minnesota Vikings | 6–1 / 2086.49 |
| 2 | New England Patriots | 0–0 / 0 | Washington Commanders | 6–1 / 2073.58 |
| 3 | New York Jets | 0–0 / 0 | Buffalo Bills | 6–1 / 1912.35 |
| 4 | Miami Dolphins | 0–0 / 0 | Arizona Cardinals | 6–1 / 1722.33 |
| 5 | Los Angeles Chargers | 0–0 / 0 | Los Angeles Rams | 6–1 / 1665.41 |

## Causes, not invented numerical differences

There are no complete app franchise totals, so a largest numerical score delta cannot honestly be ranked.
The largest MFL franchise-week totals with blocked app totals are:

| Week | Franchise | MFL | Why app total is unavailable |
|---|---|---:|---|
| 5 | Washington | 356.85 | Josh Downs is not on the currently held active roster; whole lineup refused |
| 7 | Washington | 341.38 | 18 starters need play-by-play; 3 need scoring positions |
| 3 | Minnesota | 331.15 | Jalen Davis is not on the currently held active roster; whole lineup refused |
| 7 | Minnesota | 330.20 | 18 starters need play-by-play; 3 need scoring positions |
| 5 | Buffalo | 328.10 | 16 starters need play-by-play, 4 need scoring positions, 1 absent player-week |

Read every real MFL starter through the existing native PlayerScore source, independently of whether its
lineup append was refused: 4,667 unique player-weeks. Results:

- 3,568 incomplete; 637 unavailable; 462 need a position ruling; **0 complete**.
- 4,660 player-weeks report missing held play-by-play (overlapping the other categories). Existing
  `fantasyscore/source/held.go` deliberately marks weekly-only evidence incomplete for long plays and
  return touchdowns. This affects weeks 1–7; it is not evidence that every whole weekly stats file is absent.
- 630 player-weeks (320 distinct players) absent from the held weekly statistics. Exact IDs, names and
  weeks are in `APP-SCHED-3-missing-stats.tsv`; absence is not treated as zero or inferred inactivity.
- Player 0835, Zion Childress: no matching NFL identity, weeks 1–7 (also in that TSV).
- 462 player-weeks cannot resolve a scoring position from their held statistics. No position setting
  or scoring rule was changed. E.g. Bills W1: Josh Metellus 14995, Andru Phillips 16703,
  Jalen Redmond 16927, Craig Woodson 17172. Zane Gonzalez 13347 lacks that weekly stat line.
- 76 historical lineups conflict with **today's** seeded roster, including historical starters now on
  IR, taxi, another franchise or no longer rostered. Exact first refusal for each franchise-week is
  in `APP-SCHED-3-refused-lineups.tsv`. One refusal may conceal additional conflicting starters.

No starter-to-app-player mapping failed against the held directory. NFL identity mapping is a separate
boundary; it remains missing for 0835. No plain import scoring-setting mapping bug was established.
Numeric stat-by-stat score parity cannot be proved until complete evidence and positions exist.

## Verification

Targeted Go `-race` batches (overlapping tests deduplicated): **18 top-level / 68 named PASS**, zero failures.

- `go test -race -json ./internal/leaguemigration/...`: 12 top-level / 53 named PASS.
- History batch: migration lineup sync, schedule import/backfill, rollback and changed publication:
  3 top-level / 12 named PASS. Covers idempotence, season isolation, authorization, unchanged state on
  failure and resuming a partially refused sync.
- Composition launch/schedule batch: 3 top-level / 3 named PASS. Covers launch with no target snapshot,
  played-only starters, omitted empty slot, ordinary ledger attribution and repeated launch.
- One-off copied-profile proof and player diagnostic harness: 2 PASS, not included above; removed after
  checking no retained callers. Evidence under `.tmp/sched3` (profile-proof.log, player-proof.log,
  player-scores.json, results-1.json through results-7.json, standings.json, mfl-weeks.json).
- Initial failures exposed batch prohibition and missing lineup environment; corrected by using normal
  single-command appends and installing the ordinary test environment, not weakening gates.
- Pre-commit golangci-lint, gitleaks and ifaceguard passed on both code commits.
- Plain `make build`: PASS. Build only; production binary was not launched. Wails emitted existing
  missing-struct discovery warnings but compiled and packaged successfully.

Whole affected handwritten Go file set vs 396d10ca, physical lines: production **113 → 313**;
tests **139 → 341**. No reduction claimed. No new package or dependency.

## Debt / next decision

1. Head brain must decide how historical roster membership reaches the **shared** lineup append check
   without pretending today's roster was its past roster or weakening owner submission checks. This
   run does not bypass the normal roster check or rewrite seeded rosters to make history pass.
2. Supply actual missing NFL evidence (including play-by-play) and resolve scoring positions through
   existing approved flows. Re-run parity after that; do not substitute MFL reported totals.
3. Existing ledger/results cannot represent a completely empty starting lineup. No storage or scoring
   contract was changed to solve that edge here.
4. MFL standings tie-break order, numeric per-stat parity, GUI behavior, full-suite and independent
   review remain unverified. No full make verify was repeated; the brief required targeted race/build.
