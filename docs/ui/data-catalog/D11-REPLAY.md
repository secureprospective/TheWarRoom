# D11 native projections — replay proof (2026-10-31)

Method: docs/ui/data-catalog/MODEL_METHODS_DRAFT.md §7 steps 1–4 (weighted weekly history, p10/p50/p90,
shrink toward the position distribution, `-cap` = step-4 matchup factor). Injury modifiers (step 5) not replayed.
Harness: `go run ./tools/d11replay -dir <cache with stats_player_week_2023-2025.csv.gz>` (~5 s). It calls the shipped `research.NewShape`/`research.Project` (internal/app/research/projection.go), so these numbers are the app code.
Scoring: leaguerules.Default32Scoring() through fantasyscore.Score — the league's own rules.
Walk-forward: every REG week 2–18 of 2024 and 2025, using only weeks before it. Scored on players who
recorded a stat line that week (DNP weeks are not in the sample — availability is D14/D15's job).

## Defaults (half-life 8, prior weight 1/3, MIN_SCORES 8)
| pos | n | coverage p10–p90 | Spearman median vs actual | thin-history RMSE shrunk / raw |
|---|---|---|---|---|
| QB | 1255 | 77.8% | 0.429 | 10.03 / 10.44 |
| RB | 3060 | 83.4% | 0.638 | 6.47 / 6.69 |
| WR | 4658 | 83.2% | 0.535 | 4.86 / 5.22 |
| TE | 2367 | 86.1% | 0.533 | 4.74 / 4.49 |
| PK | 1022 | 77.9% | 0.085 | 4.85 / 5.60 |
| DT | 3044 | 84.8% | 0.270 | 4.87 / 5.46 |
| DE | 2737 | 83.5% | 0.337 | 5.43 / 6.12 |
| LB | 5872 | 85.1% | 0.551 | 5.17 / 5.59 |
| CB | 3757 | 85.2% | 0.393 | 5.93 / 6.72 |
| S  | 3810 | 84.6% | 0.515 | 6.86 / 7.66 |
| ALL | 31582 | 84.0% | | |

## Against the bars
- (i) Coverage 75–92%: **PASS** pool-wide (84.0%) and at every position.
- (ii) Spearman ≥ 0.45 at QB/RB/WR/TE: **PASS RB/WR/TE; QB FAILS at 0.43.** Reported honestly per the method.
  PK (0.09) and the IDP spots are weak, as the method expected; no bar set for them.
- (iii) Shrinkage beats raw for < 8 games: **PASS at 9 of 10 positions; TE fails** (4.74 vs 4.49).
- League-level ROS-vs-final-PF bar: **NOT RUN** (needs franchise rosters as of week 1; follow-up).

## Dial sweep (prior 0.5/1.0, half-life 4/16, both)
Nothing moves QB above 0.434 (half 16), and coverage stays 83.7–84.0%. Keep the method defaults.

## Matchup factor (step 4), opponent's points allowed to the position, last 8 games, rank → [1−cap, 1+cap]
| cap | QB ρ | RB ρ | WR ρ | TE ρ | ALL coverage |
|---|---|---|---|---|---|
| 0 | 0.429 | 0.638 | 0.535 | 0.533 | 84.0% |
| 0.10 (method) | 0.430 | 0.639 | 0.535 | 0.534 | 83.8% |
| 0.20 | 0.406 | 0.636 | 0.528 | 0.529 | 83.4% |
The matchup step adds nothing measurable at 0.10 and hurts at 0.20. It does not fix QB.

## Read
Ship steps 1–3 with the QB and TE weakness shown, not hidden. Drop step 4 (matchup): replay shows no gain, so
it would be added complexity with no value (or ship it at cap 0, off). Likely QB cause (unproven): backups who
take a few snaps count as "played", and starter changes drive QB weekly rank — that is an availability
question (D14/D15 + depth chart), not a projection question.
