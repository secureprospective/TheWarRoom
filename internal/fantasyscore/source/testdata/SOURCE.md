# Held Legacy scoring proof

Only held data was read; no requests were made. Fixtures contain NFL statistics, public player
identities and fantasy totals, not contracts, salaries, franchise identities or owner information.

## Inputs and provenance

`history.db` was opened with SQLite `mode=ro&immutable=1`; its WAL was empty. Archive bodies were
inflated and their SHA256 verified before projection. `raw_archive` is defined in
`internal/store/history/history.go:63`; the configured weekly feed is `internal/measures/feeds.csv:2`.

| Held input | Uncompressed SHA256 |
|---|---|
| nflverse stats_player_week_2025.csv | e5e0615b3d96a3eaebfaee91e55afb4a4e7fe0caf057454177bcd7d6ad4bcfc2 |
| nflverse play_by_play_2025.csv | 8ce0001826f0f7b895b7a1068e4db7f43696c699db7064ccb1201855768fd06c |
| DynastyProcess db_playerids.csv | de241875e4ae91d43bbc7fbff682521da2abc9535028f952a35cb3305106c0a5 |
| Held 2025 players.json | b84e43f4585d39d9558372c4d0ad5bc40dda656cc3c42f1b4ff41e58ee46b423 |
| Held 2025 weekly playerScores_W01.json | 4907cfb1b88b0c61196e25123b516cfa9aec14f532020ae3d712946e16134b7d |
| Held 2025 weekly playerScores_W02.json | 88b54bb256a1c7bece77f61a8dab459743b16c46b6ae5c4106b40254e86bc662 |

Roster membership is selected from the already committed
`internal/ingestion/weeklyresults/testdata/weeklyResults-2025.json.gz` (starters AND bench).
True positions come from the held **2025** MFL directory, never nflverse's position label or
its coarse position group. The crosswalk joins string MFL IDs to GSIS IDs.
No rostered player with a nonzero held MFL total lacks a joined source stat row in these weeks.

`weeks.csv`: all 2,178 named regular-season player rows in 2025 weeks 1 and 2; all original columns.
Two source rows with blank player identity are excluded (not players, no nonzero rostered total).
`plays.csv`: all 5,527 plays for those weeks, projected onto the 20 columns the adapter consumes;
no distances or player credits are invented. These retain game/play identity and deleted/2-point flags.
`mfl.csv`: 879 week-1 and 904 week-2 rostered players with source stats; ten true positions,
including DT/DE/LB/CB/S. Exact held MFL point strings, including zero, remain unchanged.
`comparison.csv`: the same 1,783 identities with an independent Python Decimal reference total
in ten-thousandths and the held MFL total in ten-thousandths, plus every mismatch explanation.
The extraction, independent reference and mutation scripts are retained in the phase artifacts
(`c7a-scorer/`; `c7a-tom/plays-fixture.py` and `c7a-tom/reference.py` for the 20-column plays and
the returning-team rule).

## Stat mapping

All quantities are exact ten-thousandths. Source column headers are on `weeks.csv:1` and `plays.csv:1`.
The adapter preserves distances, not just threshold counts, so another commissioner can choose
another long-play threshold without fetching the source again.

| Catalogue ID(s) | Source column(s) / derivation |
|---|---|
| passing_td, passing_yards, interception_thrown, passing_two_point | passing_tds, passing_yards, passing_interceptions, passing_2pt_conversions |
| rushing_td, rushing_yards, rushing_attempt, rushing_two_point | rushing_tds, rushing_yards, carries, rushing_2pt_conversions |
| receiving_td, receiving_yards, reception, receiving_two_point | receiving_tds, receiving_yards, receptions, receiving_2pt_conversions |
| long_pass, long_rush, long_reception | Per-play passing_yards / rushing_yards / receiving_yards credited to their GSIS players; deleted and two-point plays excluded |
| fg_made_0_39 | fg_made_0_19 + fg_made_20_29 + fg_made_30_39 |
| fg_made_40_49, fg_made_50_59, fg_made_60_99 | Corresponding bands; fg_made_list verifies that 60+ really lies at 60–99 |
| fg_missed_0_29, fg_missed_30_99 | Corresponding missed bands; blocked attempts are also missed attempts, split using exact fg_blocked_list distances |
| punt_return_td, kickoff_return_td | Per-play touchdown and punt_attempt / kickoff_attempt, credited to td_player_id only when td_team is the return_team (a kicking team recovering its own kick has not returned it) |
| punt_return_yards, kickoff_return_yards | Same-named weekly columns |
| fumble_lost, own_fumble_recovery, opponent_fumble_recovery | fumbles_lost_total, fumble_recovery_own, fumble_recovery_opp |
| opponent_fumble_recovery_yards, forced_fumble | fumble_recovery_yards_opp, def_fumbles_forced |
| interception_return_td | Per-play touchdown + interception, credited to td_player_id only when td_team is the return_team |
| interception_caught / interception_caught_secondary, interception_return_yards, pass_defensed / pass_defensed_secondary, qb_hit | def_interceptions, def_interception_yards, def_pass_defended, def_qb_hits |
| blocked_fg, blocked_punt, blocked_extra_point | def_fg_blocks, def_punt_blocks, def_pat_blocks |
| tackle / tackle_line / tackle_secondary | def_tackles_solo + def_tackles_with_assist, recorded ONCE as tackle; scorer chooses true-position rate |
| assist / assist_line | def_tackle_assists, recorded ONCE as assist; scorer chooses true-position rate |
| sack, tackle_for_loss, safety | def_sacks (fractional), def_tackles_for_loss (direct), def_safeties |
| pat_made, pat_missed, pat_blocked | Same-named weekly columns; a blocked PAT is a missed extra point |
| fumbles_total, fumble_recovery_tds | Same-named weekly columns; kicking-team kick recovery TDs are added from held touchdown/scorer/team/kick flags, not treated as returns |
| Unused evidence | fumble_lost, fg_blocked, def_tds; no zero-point rules manufactured |

Additional weekly analytics columns remain in the raw fixture; the adapter does not pretend every
EPA/share measure has a commissioner catalogue ID. The scorer accepts further recorded stat IDs
without code changes, lists uncatalogued stats as unused, and counts them only after rules are set.

## Approval and limits

nflverse stats and play-by-play are approved in `docs/sources/Approved_Sources.md:190-206`;
the crosswalk is approved at lines 217–221. That document explicitly preserves upstream terms
and is NOT a blanket redistribution licence. FTN's CC-BY-SA clause is not a licence for these
NFL stats. MFL is the existing reference integration (`SYSTEM_MAP.md`, External services), not
a newly contacted service; Approved_Sources has no explicit MFL redistribution licence.
No new dependency or licence assertion is introduced.

Weekly stats alone combine return TD kinds in special_teams_tds and defensive TD kinds in def_tds;
held play-by-play supplies the necessary event/player attribution. Weekly threshold-count columns
alone would not support arbitrary bonus thresholds. Source positions can be LB/SAF/DB/OLB and are
not the league's true-position authority. Made field goals from 60 through 99 yards share the final band; distances above 99 are refused.
The adapter rejects missing game coverage, duplicate player weeks/plays, malformed quantities,
unsupported made distances and count/list disagreements. Whole-game coverage is checked; callers
must still pin complete, authoritative source bodies rather than pass a partial game's plays.

MFL playerScores, liveScoring and weeklyResults contain fantasy totals, not NFL stat breakdowns:
`internal/ingestion/playerscores/fetcher.go:29`, `internal/ingestion/livescoring/fetcher.go:19`,
`internal/ingestion/weeklyresults/weeklyresults.go:27`. The main archive holds YTD playerScores
rather than individual weeks; the separate held league-archive files supply the weekly oracle.

## Parity under the league’s MFL scoring preset

Christopher ruled that Legacy NFL's preset copies its actual league scoring. The held 2026
`internal/ingestion/league/testdata/rules-2026.json` and 2025 settings have identical meaning:
JSON keys and position-rule arrays are reordered, with no changed event, position, range or points.
The equivalence proof and independent Python reference are in the `r4-rulings/` phase artifacts.
The reference reads held quantities and plays independently of Go and does not import the preset.

Legacy matches **1,775 / 1,783 (99.55%)**: week 1 **876 / 879**, week 2 **899 / 904**.
The former 538 preset/rounding differences are all resolved; **8** source/credit disagreements remain.
`TestHeldLegacyWeeks` checks every independent native total, unchanged MFL score string, residual
identity and cause. `comparison.csv` lists each residual, including the specific held plays rechecked.

All differences are commissioner settings: CB/S tackle, defended pass and interception rates;
PAT made/missed/blocked; interception/opponent-recovery yards at 0.1; all fumbles at −2 (lost-only
is retained but unused); recovery touchdowns at 6; 60–99-yard made field goals; own recovery earns
2 once per player-week (`Once`, switchable to each recovery); and stat contributions rounded to
hundredths, half away from zero (`scoring.roundStatPoints`, switchable off). Canonical quantities
are not modified by the counting setting. Scoring citations point to the MFL league settings.
The kickoff recovered by SEA in the end zone (`2025_02_SEA_PIT` play 3004) earns recovery TD points,
not kickoff-return points. Repeat attachment does not duplicate its credit.

### Eight residuals rechecked against the full held play-by-play

The full source SHA above was reverified before reading. The phase's `residual-evidence.json`
retains the structured player credits and descriptions; none was inferred from a name match.
Potential explanations below are not asserted as MFL's actual credits.

| Week / player | Native / MFL points | Held finding |
|---|---|---|
| 1 Percy Butler | 3 / 4 | NYG–WAS 236 solo punt tackle, 1662 shared kickoff tackle. Converting the assist to solo would add 1; MFL's credit is not held. |
| 1 Darnell Savage | 3 / 0 | CAR–JAX 3740 credits the defended pass. Nothing held explains MFL's zero. |
| 1 Nick Bellore | 3 / 2.5 | NYG–WAS 612 and 1233 both credit solo special-teams tackles. A different solo/assist split could explain −0.5, but is not held. |
| 2 Andru Phillips | 36 / 33.5 | NYG–DAL 1724 and 3848 credit two TFLs; 1724 also has a personal foul. One fewer TFL would explain −2.5, but the held play still credits it. |
| 2 Jamal Adams | 5.5 / 4.5 | LAC–LV 2429 credits a shared tackle. Removing one assist would explain −1, not justified by held evidence. |
| 2 Harold Landry | 17 / 15.5 | NE–MIA 1635 and 3091 credit shared tackles. One fewer assist would explain −1.5, not proved. |
| 2 Jaylinn Hawkins | 12 / 13 | NE–MIA 975 and 1693 credit shared tackles. One assist counted solo would explain +1, not proved. |
| 2 Drake London | 8.4 / 9.4 | ATL–MIN 832, 3262, 3325 record 18/10/21-yard receptions. 832 also credits his lost fumble and solo recovery tackle. The 21-yard bonus is already counted; no extra threshold or touchdown exists. |

No residual can be resolved without inventing a credit or changing source facts. MFL's credited
NFL-stat breakdown at the held correction time is the smallest missing evidence. No network was
contacted. No runtime scorer, adjustment queue, owner reading or scouting consumer is wired here.
