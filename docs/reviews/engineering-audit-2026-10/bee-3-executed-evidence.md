# Sector 3 — Executed evidence

Source reads remained against read-only `src/`; executable checks used a writable copy in `scratch-src/`. All DB analysis used `?mode=ro&immutable=1`. Both migration exercises ran only on copies in `migration-copies/`; neither supplied DB nor `~/.config/TheWarRoom/` was opened for writing.

## 3a. Build and tests

Environment used (as specified):

```sh
export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH
export GOCACHE=/home/chris/fleet/runs/warroom-dataflow-2026-10-03/.gocache
export GOLANGCI_LINT_CACHE=/home/chris/fleet/runs/warroom-dataflow-2026-10-03/.lintcache
export GOTMPDIR=/home/chris/fleet/runs/warroom-dataflow-2026-10-03/.gotmp
export GOFLAGS=-mod=readonly GOPROXY=off GOTOOLCHAIN=local
export GOMEMLIMIT=1500MiB
mkdir -p "$GOTMPDIR"
cd scratch-src
```

The supplied snapshot has no `frontend/dist`, required by `main.go`'s `//go:embed all:frontend/dist`. The four exact first attempts (`go build ./...`, `go vet ./...`, `go test -race -count=1 ./...`, `golangci-lint run ./...`) each stopped on `pattern all:frontend/dist: no matching files found`; this was the omitted generated embed artifact, not a code failure. To compile and test the Go sources without installing frontend dependencies or building/running a UI, I added a 54-byte `index.html` placeholder **only** under `scratch-src/frontend/dist/`, then reran the requested Go gates. No app binary was launched and no frontend build was run.

Final commands and outcomes, all from `scratch-src/` with the environment above:

| Command | Exit | Result |
|---|---:|---|
| `go build ./...` | 0 | All Go packages compile with the scratch-only embed placeholder. |
| `go vet ./...` | 0 | No diagnostics. |
| `go test -json -race -count=1 ./...` | 0 | 950 test events: 926 passed, 24 skipped, 0 failed. 51 packages: 47 pass; 4 report no test files. |
| `golangci-lint run ./...` | 0 | `0 issues.` |

Packages passing: root `github.com/secureprospective/TheWarRoom`; `internal/composition`, `db`, `domain`, `engine`, `engine/l4/defense`, `engine/l4/kicker`, `engine/l4/offense`, `harness`; `internal/ingestion`, `agetrajectory`, `collegedefense`, `collegeshare`, `crosswalk`, `kicking`, `league`, `leagueschedule`, `leaguestandings`, `madden`, `nflproduction`, `pfrcoverage`, `pfrpassrush`, `players`, `playerscores`, `ras`, `rosters`, `salaryadjustments`, `schedule`, `schooltier`, `touchshare`, `veteranfilm`; `internal/m2service`, `mfl`, `normalize`, `numeric`, `output`, `playerid`, `powerrankings`, `rankings`; `internal/scouting/assembly`; `internal/store/params`, `rulebook`, `state`; `internal/transactions`, `contracts`, `deadcap`, `freeagency`.

No-test-file packages: `internal/engine/l4/curve`, `internal/schema`, `internal/scouting`, `internal/transactions/acquisitions`. Full package/test event output is `logs/sector3-go-test-json-final.log`; build, vet and lint output is in `logs/sector3-*-final.log`. The 24 test skips are test-level skips, not failures. No `TWR_LIVE_*` variable was set and no network/live gate was run. The Go build used a dummy embedded page, so this proves Go compile/test/lint only—not a built frontend or working GUI.

## 3b. The copied data

The query/calculation script is `logs/sector3-analytics.py`; its complete output, including the 32-franchise cap totals for each copy, is `logs/sector3-analytics.txt`. Main queries use `PRAGMA table_info(season_scores)` to enumerate every INTEGER/REAL/NUMERIC column, then for each run `SELECT COUNT(column), MIN(column), MAX(column), COUNT(DISTINCT column) FROM season_scores`. Counts below are all 827 rows in release-path `thewarroom.db` and 1,299 rows in `thewarroom-dev.db`.

### Persisted numeric values and effective scoring

| Numeric column | Release-path min–max; distinct | Dev min–max; distinct |
|---|---|---|
| `season` | 2026–2026; 1 | 2026–2026; 1 |
| `scoring_config_id` | 1–1; 1 | 1–1; 1 |
| `base_points` | 0–456.1; 651 | 0–476.75; 890 |
| `age_pull` | 0.700109–1; 200 | 0.699824–1; 331 |
| `film_effective` | 1–1; 1 | 0.973148–1.049575; 657 |
| `film_raw` | 0.5–0.5; 1 | 0.4–0.954691558; 668 |
| `ras_effective` | 0.921414–1.077913; 597 | 0.921003–1.078258; 867 |
| `breakout_effective` | 0.980470–1.019530; 550 | 0.980470–1.019530; 838 |
| `combined` (L4) | 0.922214–1.096765; 740 | 0.932549–1.146005; 1,126 |
| `scouting_adjusted` | 0–461.192568; 761 | 0–483.108482; 1,114 |
| `cap_multiplier` | 0.85–1.15; 3 | 0.85–1.15; 3 |
| `adjusted_score` | 0–457.106126; 762 | 0–460.656004; 1,115 |
| `tb_is_veteran` | 1–1; 1 | 0–1; 2 |
| `tb_ras` | 0.449256–10; 655 | 0.395025–10; 955 |
| `tb_scarcity_rank` | 0–0; 1 | 0–0; 1 |

`season` and `scoring_config_id` are constant batch metadata. Engine-output constants are more consequential: release-path `film_effective` is exactly neutral 1.0 for all 827 rows and `film_raw` exactly neutral 0.5; `tb_scarcity_rank` is zero for every row in both copies. These are facts about stored test-exercised batches, not a fresh scoring run. The dev batch has varied film values.

Zero-base / final-score arithmetic:

```sql
SELECT COUNT(*) AS rows,
       SUM(base_points=0) AS zero_base,
       SUM(adjusted_score=0) AS zero_adjusted
FROM season_scores;
```

Release-path: 827 rows, 65 zero-base (7.86%), 65 zero-adjusted. Dev: 1,299 rows, 184 zero-base (14.16%), 184 zero-adjusted. No row with BasePoints=0 escaped zero AdjustedScore; the multiplier cannot move those players off zero.

To quantify ranking influence, the script computes Spearman rank correlation between `base_points` and stored `adjusted_score` using average ranks for ties. It then reranks the same records with L4 set to 1 (`base_points × age_pull × cap_multiplier`) and keeps the persisted L6 tiebreak fields fixed, using the exact ordering in `output.Store.Scores`: score descending, veteran, RAS, scarcity, MFL ID. This isolates L4's change to the primary score while retaining the stored tie order.

| Copy | Spearman ρ: AdjustedScore vs BasePoints | L4-neutral mean absolute rank shift | Maximum | Rows whose rank changes |
|---|---:|---:|---:|---:|
| `thewarroom.db` | 0.989117 | 6.6626 places | 47 | 683 / 827 |
| `thewarroom-dev.db` | 0.992003 | 9.5643 places | 79 | 1,044 / 1,299 |

The rank correlation is very high, consistent with the prior-year points proxy driving the ordering; L4 is nevertheless not a rounding error in ranks: the counterfactual moves hundreds of rows and up to 47/79 places. This does not make L4 an independent measurable: it remains a multiplier, and zero-base players remain zero. `season_scores` has no `position` column and neither copy persists the MFL player directory, so per-position L4 spreads could not be established without a network fetch (prohibited here).

### Current-season cap totals, joins, and orphans

`state.loadCellCap` in `src/internal/store/state/helpers.go` reads current-season `PAID` `contract_years.salary_cents`, applies the rulebook taxi/IR percentage by roster status, and rounds each player contribution via `domain.RoundToNearest10k`. Both stored rulebook payloads omit `IncludeTaxiWithSalary` / `IncludeIRWithSalary`; `capPercent("")` defaults to 100%. Neither roster set has IR rows. Every current PAID salary cell is already $10,000-aligned, so the per-player snap does not change the total. The query shape is:

```sql
SELECT r.franchise_id, r.mfl_id, r.roster_status, cy.salary_cents
FROM contract_years cy
JOIN rosters r
  ON r.league_id=cy.league_id AND r.mfl_id=cy.mfl_id AND r.season=:state_season
WHERE cy.league_id='14432' AND cy.league_year=:state_season
  AND cy.year_status='PAID';
```

The per-player granularity check was:

```sql
SELECT r.season, count(*) AS rostered,
       SUM(CASE WHEN cy.salary_cents % 1000000 <> 0 THEN 1 ELSE 0 END) AS non_10k_aligned_cells
FROM rosters r
LEFT JOIN contract_years cy
  ON cy.league_id=r.league_id AND cy.mfl_id=r.mfl_id
 AND cy.league_year=r.season AND cy.year_status='PAID'
WHERE r.season=(SELECT season FROM season_phases ORDER BY rowid DESC LIMIT 1)
GROUP BY r.season;
```

Output: release-path `2027 | 831 | 0`; dev `2026 | 1375 | 0`. Aggregating the cells using the source function's percentage + per-player rounding gives 32 franchises and complete current PAID coverage in both copies:

| Copy | State season | Rosters / current PAID cells | Current PAID missing roster | Ledger-derived cap total | Difference vs `contracts.annual_salary_cents` sum |
|---|---:|---:|---:|---:|---:|
| Release-path | 2027 | 831 / 831 | 0 | $2,399,890,000 | −$7,300,000 across all teams |
| Dev | 2026 | 1,375 / 1,375 | 0 | $3,514,740,000 | $0 |

`CapUsed` is not a separately stored column: the runtime derives it from `contract_years`, so the exact queried ledger formula is the app's cap truth. The release-path ledger total differs from the older/base `contracts.annual_salary_cents` totals on three franchise IDs: `0001` by +$1,000,000, `0002` by −$5,000,000, and `0011` by −$3,300,000; the other 29 totals agree. All 32 dev totals agree. This is a difference between ledger cap cells and the legacy/base salary field in a test-exercised copy, not evidence that the engine's ledger calculation disagrees with itself. The full per-franchise arithmetic is in `logs/sector3-analytics.txt`.

Orphan check in the script counts each table row whose `mfl_id` or `franchise_id` has no matching value in `rosters` (which stores only the current state snapshot). Current same-season `rosters` without a matching `contracts` row, `contracts` without a same-season roster, and current-season PAID cells without a roster are all **0** in both copies; all 32 franchise IDs appear in roster state. Historical ledger and status rows naturally refer to cut/expired players no longer in the current roster:

| Table, ID absent from current roster | Release-path rows | Dev rows |
|---|---:|---:|
| `contract_year_changes.mfl_id` | 833 | 6 |
| `contract_years.mfl_id` | 813 | 4 |
| `dead_cap_ledger.mfl_id` | 6 | 2 |
| `player_status_events.mfl_id` | 397 | 2 |
| `season_scores.mfl_id` absent from any current roster | 0 | 2 |
| Rows with `franchise_id` absent from any current roster | 0 | 0 |

These ledger/status absences are historical by design, not broken current-roster joins. A separate season-key comparison demonstrates the release-path test state mismatch:

```sql
WITH state_season AS (
  SELECT season FROM season_phases ORDER BY rowid DESC LIMIT 1
), score_season AS (
  SELECT DISTINCT season FROM season_scores
)
SELECT (SELECT season FROM state_season) AS state_season,
       (SELECT group_concat(season) FROM score_season) AS score_season,
       (SELECT count(*) FROM season_scores) AS score_rows,
       (SELECT count(*) FROM season_scores s WHERE NOT EXISTS (
          SELECT 1 FROM rosters r WHERE r.mfl_id=s.mfl_id AND r.season=s.season
       )) AS scores_without_same_season_roster,
       (SELECT count(*) FROM rosters r WHERE r.season=(SELECT season FROM state_season)
          AND NOT EXISTS (SELECT 1 FROM season_scores s
                          WHERE s.mfl_id=r.mfl_id AND s.season=r.season)
       ) AS current_roster_without_same_season_score,
       (SELECT count(*) FROM contract_years cy
          WHERE cy.league_year=(SELECT season FROM state_season)
            AND cy.year_status='PAID'
            AND NOT EXISTS (SELECT 1 FROM rosters r
                            WHERE r.mfl_id=cy.mfl_id AND r.league_id=cy.league_id
                              AND r.season=cy.league_year)
       ) AS current_paid_cells_without_roster;
```

Release-path state is 2027 but all 827 score rows are season 2026: 827 score rows have no roster row for their score season; all 831 current 2027 roster rows have no 2027 score row; **0** current 2027 PAID cells lack a roster. Dev state and output are both 2026: 2 saved score IDs have no roster in any season, and 78 current-roster IDs have no score row for 2026 (the output table is not a complete roster projection). The release-path mismatch is the same test-exercised year split traced in S2-F3, now confirmed against every score row.

### Parameters and app logs

Both copies have the same five `param_defaults` and zero overrides:

```sql
SELECT param_key, position, default_val, min_val, max_val
FROM param_defaults ORDER BY param_key, position;
```

| Key | Default | Code consumer |
|---|---:|---|
| `captier.cold_ceiling_pct` | 1.2 | consumed by L5 cap-tier boundary |
| `captier.hot_floor_pct` | 4.8 | consumed by L5 cap-tier boundary |
| `layer3.decay_rate` | 0.03 | consumed by L3 age decay |
| `cushion_guard.ras_threshold` | 8 | stored and editable; not consumed (DT threshold is literal 8.00) |
| `cushion_guard.reduction` | 0.1 | stored and editable; not consumed (DT reduction is literal 0.90) |

The code's five defined keys match these five rows; composition reads only the two cap tiers and L3 decay. The North Star's L2 scoring-value overrides, per-position L3 peak limits, L4 weights/curves, and L6 scarcity matrix have no defaults in either DB and are not represented as effective engine tunables. Source grep: `GetCapTiers`/`GetGlobal` call sites in `src/internal/composition/composition.go`; key declarations/defaults in `src/internal/store/params/types.go` and `defaults.go`; DT literals in `src/internal/composition/defaults.go`.

`find data/app-logs -type f -printf '%f %s bytes\\n'` returned ten files, each 0 bytes; count=10, nonempty=0, total bytes=0. There is no error, warning, migration, or timing content to attribute to the last runs. What happened on those launches is **could not establish** from logs. Christopher's feedback mentions seven empty files in the original config directory, but this audit did not access outside the permitted copied `data/app-logs/` directory.

## 3c. Open both copies under today's store code

A writable copy of each supplied database was made inside `migration-copies/`. The audit-only test `scratch-src/audit_migration_test.go` calls `db.Open` then the actual `App.initStoreFloor` (params, rulebook, state/migration DDL, Coordinator, and output stores); it never constructs an MFL client. Commands:

```sh
cp data/thewarroom.db migration-copies/release-full-open.db
cp data/thewarroom-dev.db migration-copies/dev-full-open.db
chmod u+w migration-copies/release-full-open.db migration-copies/dev-full-open.db
cd scratch-src
AUDIT_DB=/home/chris/fleet/runs/warroom-dataflow-2026-10-03/migration-copies/release-full-open.db \
  go test -v . -run '^TestAuditOpenCopiedLeagueDBThroughStoreFloor$'
AUDIT_DB=/home/chris/fleet/runs/warroom-dataflow-2026-10-03/migration-copies/dev-full-open.db \
  go test -v . -run '^TestAuditOpenCopiedLeagueDBThroughStoreFloor$'
```

Both passed. Each initialized 32 franchises, active rulebook version 1, and WAL mode. Because active rulebook rows and current-season state existed, store-floor initialization did not need an MFL source; the test has no MFL client and succeeded offline.

Before/after table and row-count comparison is in `logs/sector3-migration-diff.py` / `.txt`:

- Release-path copy: 16 tables → 21. State DDL added exactly `calendar_events`, `league_schedule_cache`, `standings_cache`, `trade_notes`, and `transaction_corrections`, each with 0 rows. Every pre-existing table's row count is unchanged.
- Dev copy: 21 tables → 21; no table added or removed, and every table row count is unchanged.
- Both copies retain their original two `state` migration markers: versions 1/2, `method='reconciled'`. No migration marker or data row was changed. This method means v1/v2's target state was already satisfied when stamped; it does not mean either transformation ran now. The feature-table `CREATE TABLE IF NOT EXISTS` DDL has no separate markers. No pre-migration backup was generated because no real money-schema migration work was pending.

**Result:** both copies open and initialize cleanly under the current store floor. Release-path startup creates the five missing current-code tables before IPC, so the raw-copy absence of `trade_notes` does not make normal post-initialization feed reads fail. This migration result says nothing about whether either database is a faithful live-league mirror; Sector 1c establishes that neither copy is trustworthy live state.

## Sector 3 findings

### S3-F1 — `HIGH` — Release-path output is from 2026 while the same copy's runtime state is 2027 (`BROKEN`)

The query above returns 827/827 saved score rows with no same-season roster row, and 831/831 current-state rosters without a same-season score. The active state has 831 2027 players, but `season_scores` holds 827 2026 rows. This is a demonstrated copy-level output/state key mismatch, not evidence that the real league is in 2027; it confirms S2-F3 and the test-exercised provenance from S1-F6.

### S3-F2 — `MEDIUM` — The persisted board remains highly correlated with the fantasy-points proxy (`PROXY`)

Average-tie Spearman rank correlation of AdjustedScore against BasePoints is 0.989117 (release-path) and 0.992003 (dev). The non-neutral L4 multipliers still move many rank slots, but 65/827 and 184/1,299 zero-base rows remain exactly zero. This quantifies, rather than replaces, S2-F1: a single proxy-backed score exists, not the two independent measurables Christopher chose.
