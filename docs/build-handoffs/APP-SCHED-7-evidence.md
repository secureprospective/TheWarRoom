# APP-SCHED-7 — measurement at 1207f2da

2026-11-01; build/SCHED-1. No feature/settings/position-queue changes. No git remote contact.
SQLite backup API copied all three databases from the authorized scratch profile into
`.tmp/sched7/profile`; neither installed profile nor source databases were written.
Existing archive parsers/backfill recorded 148 lineups; 76 roster-incompatible franchise-weeks
remain excluded. Ordinary SyncScoring acquired 2026 play-by-play; ProcessStatFeed recorded
selected revisions on the copy. NativeResults and NativeStandings were read afterward using
the real composition player-directory environment and league-position precedence.
The cohort is exactly the previously recorded 148 keys, not all 224 franchise-weeks.

## Comparison

`APP-SCHED-7-comparison.tsv`: 148 rows. Exact compares integer fixed-point scores;
within .01 excludes exact; differ requires a complete native total. Missing is never zero.

| Week | Recorded | Exact | Within .01 | Differ | Unavailable |
|---|---:|---:|---:|---:|---:|
| 1 | 15 | 1 | 0 | 2 | 12 |
| 2 | 15 | 2 | 0 | 1 | 12 |
| 3 | 18 | 4 | 0 | 2 | 12 |
| 4 | 22 | 4 | 0 | 2 | 16 |
| 5 | 23 | 6 | 0 | 0 | 17 |
| 6 | 24 | 3 | 0 | 1 | 20 |
| 7 | 31 | 4 | 0 | 1 | 26 |
| **Total** | **148** | **24** | **0** | **9** | **115** |

Previous part 5 **TSV actually contains 4 exact / 29 differ / 115 unavailable**.
Its prose total (4/33/111), repeated in the handoff, was an arithmetic reporting error;
its per-week table agrees with its TSV. Completeness is unchanged, not four weeks worse.
League positions improve complete-score parity; they do not supply missing evidence.

### Top five complete differences (absolute delta; app minus MFL)

| Week / franchise | MFL | App | Delta | Observed attribution |
|---|---:|---:|---:|---|
| 1 NY Giants | 208.20 | 179.70 | -28.50 | -26 highest-starter penalty (Za'Darius Smith); Roquan Smith -2.50 residual |
| 6 Atlanta | 214.15 | 217.65 | +3.50 | Cooper DeJean +2; D.J. Reader +1.50 residuals |
| 2 Pittsburgh | 146.28 | 142.78 | -3.50 | Jeffery Simmons -2.50; Jessie Bates -1 residuals |
| 4 Arizona | 294.90 | 292.90 | -2.00 | Chuck Clark -2 residual |
| 3 Tampa Bay | 186.30 | 184.30 | -2.00 | Mike Evans -2 residual |

Giants lineup actually contains 18 starters: offense 6/8, defense 11/12, total 18/21;
native configured penalty zeroes the 26-point highest scorer. Other four lineups are valid.
These player-level residuals are observed attribution, **not established upstream stat-versus-rule
root causes**: the held MFL weeklyResults gives player points, not the stat-credit breakdown.
Native item tracing: Roquan 5 solos/6 assists/1 INT/1 PD/1 QB hit/1 INT return yard;
DeJean 6 solos/1 PD/1 TFL; Reader 5 solos/2 assists/1 TFL;
Simmons 3 solos/3 assists/3 QB hits/1 sack/1 TFL;
Bates 4 solos/2 assists; Clark 4 solos/3 assists;
Evans 3 receptions/34 yards/1 TD/1 reception >=20 yards.
No positional penalty explains these residuals, and no rate mismatch is proved.
Denver W4 Schultz remains 2 receptions/9 yards = 2.90 versus MFL 3.90;
its underlying missing credit remains unresolved, not silently diagnosed as a rule error.

**Bills W1–7:** MFL **6–1, PF 1912.35**; native **unavailable (0/7 complete weeks)**.
Displayed 0–0/PF 0 is no counted evidence, not an observed season result.

## Absent starter evidence

`APP-SCHED-7-absent.tsv`: 479 starter occurrences (also 479 distinct player-weeks):
472 missing weekly rows plus 7 identity refusals. This includes starters whose individual
statistics are missing even if the lineup penalty route produces a usable franchise total.

| Classification | Occurrences | Evidence / limit |
|---|---:|---|
| Did-not-play: bye candidates | 36 | Held MFL 2026 nflByeWeeks + player-directory NFL team; historical team affiliation not independently verified |
| Did-not-play: inactive confirmed | 0 | No held exact-week game-day inactive list established |
| Did-not-play: injured / Out report | 53 | nflverse injuries_2026.csv, exact GSIS + season + REG week, report_status=Out |
| Feed coverage gap: participation held | 141 | nflverse snap_counts_2026 observations have positive offense/defense/special-team snaps; scoring-week row absent |
| Unresolved absence | 242 | No positive snaps, matching bye, or exact-week Out evidence; cannot honestly choose inactivity versus feed gap |
| Identity gap | 7 | 0835 Zion Childress; canonical external-ID table has no GSIS link |
| **Total** | **479** | **472 absent rows + 7 identity gaps** |

The 36 byes are candidates, not 36 verified historical affiliations. Out is an exact-week
report designation, not an independently audited participation record. A positive snap proves
participation but does not prove a lost feed row: a player may have no scoring-counting statistics.
Neither missing snaps nor missing weekly statistics establishes inactivity. No speculative
inactive classifications, alias, or zeroes were inserted. Full root-cause classification remains
unverified for 242, and historical team confirmation remains for the 36 bye candidates.

**Existing in-app sources, no build needed for this measurement:**
`internal/ingestion/nflschedule/byes.go` parses MFL nflByeWeeks;
nflverse injury ingestion already stores exact-week `availability.game_status` observations
(Out), and snap-count ingestion already stores exposure snap observations.
`internal/ingestion/injuries` also parses MFL current status, used by IR/taxi eligibility,
but a current injury/retired designation cannot establish W1–7 nonparticipation retrospectively.
These are candidates for an evidence-backed did-not-play marker; none currently makes missing
native player-week statistics a verified zero. A game-day inactive list was not established.

## Execution / debt

- Disposable composition measurement PASS, 208.44 s (non-race); 148 backfills, acquisition,
  automatic stat ledger, all seven native results and native standings. Exit 0.
- Follow-up native player-item trace PASS under `-race`, 33.82 s, eight player-week readings.
- Initial race measurement hit the external 600 s deadline during lineup adoption; interrupted,
  then resumed on the disposable copy with ordinary idempotent backfill. Not reported as a pass.
- One-use harnesses retained only in ignored `.tmp/sched7`, removed from production/test tree.
  No full build/suite repeated for docs-only work; no GUI validation or independent review.
- Debt: unresolved absences and historical bye affiliation; verified 0835 identity; nine score
  residuals/penalty parity (including Schultz); 76 historical-roster refusals; slow adoption/readings;
  deferred player-position queue integration; existing live/advanced-feed and source-health debt.
