# Valuation review — outputs first

Read-only triage, 2026-10-27. **1,450 rostered players; 57 candidates**, not 57 proven defects.
No valuation, fitted parameter, scoring-rule or real-profile changes. No MFL requests.

## Source and reproduction

Source: the real `~/.config/TheWarRoom/history.db` and `thewarroom.db`, copied with SQLite's
backup API through read-only connections. Copies, raw exports, extracted features, exact effective
parameters, replay output and source hashes stay in the ignored `.scratch/value-board-reviewed/`.
The board is `.scratch/value-board-reviewed/warroom-value-board.html` (open directly in a browser).
The operator restricted writes to this worktree, so neither `/tmp` snapshots nor the requested
Downloads destination were used.

This is **current code on held data**, not live October 27 outcomes:

- Season 2026; observations through **2026-10-05 18:44:31 UTC**.
- Current league points: October 5, 18:44:24 UTC; 2025 points: October 3, 22:44:46 UTC.
- Rosters: October 5, 18:43:52 UTC; MFL player directory: October 5, 19:59:41 UTC.
- The kept `~/logs/TheWarRoom/bigleague-fixtures/thick3-*.db` has no observations/model runs;
  it cannot supply these outcomes. The profile has **no archived free-agent export**; zero free
  agents were valued, rather than inventing eligibility from unowned directory entries.
- Fresh shipped defaults plus the profile's overrides (none): 3 seasons, discount 0.75.
  Historical model run #3 is an old build, not a substitute for replaying current code.
- 253 current totals, 292 last-season totals and five birth dates are absent. Missing is not zero.
  The production runner set aside one mislinked history record (`17767`), as designed.

From the repo root, with Go 1.26.10 on PATH (Python 3.11+; no Python dependencies):

```sh
GOTOOLCHAIN=go1.26.10 GOFLAGS=-p=2 python3 scripts/value_board.py --out .scratch/value-board-review
```

Choose a **new output directory** on each run; existing snapshot files are deliberately not
replaced. `--profile` selects another held profile. `--out` must stay in this worktree. The
command produces the self-contained HTML and `report.md`, using no app startup or network
fallback. Source freshness remains whatever the selected profile actually holds.

## Actual flow and safety

| Responsibility | Contract / effects |
|---|---|
| `internal/composition/engine_refresh.go` → `internal/app/scoring/model.go` | The app's logged “players valued (rostered, free agents)” refresh calls `modelrun.Runner.Run`. |
| `scripts/value_board.py` | Backup copies → exact archived MFL exports and source-priority/as-of features. Native SQLite extraction follows `history.Features`' latest-per-source, then priority selection; the measures and lookback come from Go, not a second list or constant. No source writes. |
| `internal/modelrun/value_board_test.go`, tag `valueboard` | Validated replay request + production MFL parsers/normalizer → **production Runner.Run**. Existing test roster adapter supplies ownership; model-run writes are captured in memory. No persistent model run is written. |
| `internal/modelrun/modelrun.go` → `internal/model` | MFL positions, games and held league points → prior/production blend, age/survival projections, dynasty PPG. K currently uses `TodayRule`: last total × age pull ÷ 17. Diagnostics alongside K are still model diagnostics, not components of that substituted value. |
| `internal/m2service` | Reads latest stored `ModelScores`; sums Now/Dynasty PPG for teams. Salary and scarcity are not multipliers in these player values. |
| Python assessment → `scripts/value_board.html` | Current outputs → three deterministic candidate checks → escaped HTML. Plain local search, candidate filter and expandable contracts/inputs; no fetch, storage or mutation controls. |

Direct Go SQLite reads under `-race` timed out twice at five minutes; even an index on a disposable
copy did not make that route practical. Native SQLite exports the same selected features, then
Go replays the **unchanged runner under `-race`** in about 17 seconds. No gate was disabled.
`Engine_Specification.md` is historical layer-1–4 design; the current implementation and
Core Build Plan's Stage 7/8 amendments describe the active path.

## Checks and counts

Ranks are within position among valued players, with competition ranks for ties. Points rank is
this season's **total league fantasy points**, not generic NFL scoring or a projection. Actual PPG
and its rank are also visible to distinguish missed games from per-game performance. A dynasty
rank need not match a short season's points rank. Candidates are ordered by number of checks,
then absolute rank gap; this order is a reading queue, not a severity model.

| Check | Simple trigger | Players |
|---|---|---:|
| Rank gap | Known current total (explicit zero included); ≥20 rank places apart; either rank ≤20 | 53 |
| Zero-valued producer | Positive current points, points rank ≤20, dynasty PPG ≤0 | 5 |
| Older over producer | Older player value rank ≤30 above someone ≥4 years younger with >25% more total points in **both** seasons | 2 |

Older thresholds: QB 34, RB 29, WR 31, TE 32, K 36, DT/DE 31, LB/CB 30, S 31. These are review
thresholds, not fitted parameters. Counts overlap: 60 check hits, **57 unique players**. No RB
triggered these checks; that is not evidence all RB values are correct.

No run-jump flag: stored runs #2/#3 changed engine, parameters and inputs together; a difference
would not isolate an unexplained jump. No cross-position flag: positional lineup scarcity is not
in these player outputs, and PPG by itself does not establish replacement-level value.

## Top 20 candidate reads

| Player | Position | Value rank / points rank | Why review |
|---|---|---|---|
| Jadeveon Clowney | DE | 14 / 68 | 54 places above production; also above Dallas Turner, 23.6, despite >25% lower totals in both seasons. |
| Dominic Zvada | K | 37 / 11 | 38 current points, zero value; no last-season total. |
| Tyler Bass | K | 37 / 17 | 31 current points, zero value; no last-season total. |
| Myles Garrett | DE | 4 / 133 | 129 places above production; explicit zero points in one recorded game. |
| Paulson Adebo | CB | 20 / 119 | 99 places above production. |
| Brian Burns | DE | 17 / 109 | 92 places above production. |
| Nick Bosa | DE | 2 / 84 | 82 places above production. |
| Josh Uche | DE | 92 / 18 | Top-20 scorer sits 74 places lower by value. |
| Princely Umanmielen | DE | 90 / 17 | Top-20 scorer sits 73 places lower by value. |
| Nick Cross | S | 6 / 68 | 62 places above total-points rank; only two games, actual PPG 13.25. |
| Kamren Kinchens | S | 10 / 65 | 55 places above total-points rank; only two games, actual PPG 13.63. |
| Christian Gonzalez | CB | 18 / 71 | 53 places above production. |
| Hezekiah Masses | CB | 48 / 1 | Position's leading scorer sits 47 places lower by value. |
| Kalif Raymond | WR | 67 / 20 | Top-20 scorer sits 47 places lower by value. |
| Fernando Mendoza | QB | 3 / 49 | Top-three value with zero points and no recorded games. |
| L'Jarius Sneed | CB | 54 / 11 | Top-20 scorer sits 43 places lower by value. |
| Marcus Harris | CB | 46 / 5 | Top-five scorer sits 41 places lower by value. |
| Lukas Van Ness | DE | 46 / 6 | Top-ten scorer sits 40 places lower by value. |
| Chuck Clark | S | 49 / 9 | Top-ten scorer sits 40 places lower by value. |
| Minkah Fitzpatrick | S | 11 / 49 | 38 places above production. |

## Cause families to investigate, not tune yet

- **Data gaps / fallback — source-traced:** all five zero-valued producers are K: Zvada, Bass,
  Jason Sanders, Trey Smack and Drew Stevens. Each lacks a 2025 total. `TodayRule` discards the
  current-season blend, so 28–38 current points still produce zero. Check missing-versus-zero
  semantics and this fallback first; do not start by adjusting an age curve.
- **Recency / shrinkage — plausible:** DE retains 1/0.8/0.6 recency, k-now 5.627 games. Uche and
  Umanmielen's four current games receive only 41.5% weight; Garrett's one game receives 15.1%.
  Past evidence and priors can explain large short-season rank differences without a defect.
- **Age / prior / survival — review:** Clowney (33.5) and Khalil Mack (35.5) outrank Turner (23.6),
  who has 230 last-season / 83.75 current points versus 183.75/26.25 and 143.75/60. Mendoza's
  high prior and debut chance deserve a sitting-rookie review. No counterfactual isolates age alone.
- **Scoring rules — unproven:** totals are MFL's actual league points and scales use MFL positions.
  DE/CB/S dominate the candidates, but that does not prove a scoring-rule error.
- **Scarcity — outside these outputs:** no claim that raw cross-position PPG equals roster value.

## Evidence and remaining verification

Twelve Python tests exercise ties, missing versus explicit zero, check boundaries, both-season
age comparisons, HTML escaping, non-finite rejection, snapshot preservation/symlink rejection,
bounded archive decoding, exact export matching and source-priority/as-of selection. A Go regression
test rejects inferred zero totals from snap-only evidence while preserving explicit MFL zeros. Targeted Go
race tests (including the held free-agent parser path), real-data race replay and tagged lint pass. All **456,370 extracted features**
match the engine-created season/week feature views by SHA-256. Two complete generations produced
byte-identical replay JSON and HTML; both source DB files' SHA-256 values stayed unchanged.
Canonical-ID request validation also has a negative test: removing the guard lets `816` through
instead of requiring `0816`, and the test fails. The restored guard passes. Commit hooks run without bypasses.

Firefox headless exercised search, candidates-only, empty state and contract/age-pull expansion;
all 1,450 rows/ten positions were present. Desktop 1440×1000 and mobile 390×844 were visually
inspected; the table scrolls inside its own region. The current browser receipt is beside the board; the desktop/mobile captures are in
`.scratch/value-board-final/` (same layout, before the explicit-zero check added two candidates). The probe permits loopback reporting only in its test copy; the
actual artifact has `connect-src 'none'`.

Operator check: open the board, search “Zvada”, enable candidates-only, expand inputs: K rank
37 / points rank 11, 38 current points, zero value and last-season fallback should all agree.
Then search “Mendoza”: QB rank 3 / points rank 49 with explicit zero, not a missing total.

Latest archived MFL YTD exports (October 5, 20:00 UTC), decoded and checked with the production
`playerscores.RawScore.Validate`, agree with all 1,158 held 2025 totals and 1,197 held 2026 totals.
The other 292/253 are absent in both. This cross-check caught and removed five display-only zeros
that had been inferred from snaps; it did not change any engine value or candidate count.

Debt left: live/fresher outcomes, a held free-agent pool, independent external outcome spot-checks, and
owner adjudication of the candidates. Downloads publication awaits permission to write outside
this worktree. No valuation tuning or fleet/app deployment is part of this change.
