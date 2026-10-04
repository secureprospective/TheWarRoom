# Measure Dictionary

Generated from `internal/measures/*.csv` by `make measure-dictionary`. Do not edit it by hand: change the CSV files and regenerate.

Every number in `history.db` is stored as player · season · week · measure. Week 0 holds season-level values; season 0 holds player-level facts that belong to no period. A week-level number is stored only when it is not zero, so a week the player was on the field with no row for a count means zero. A measure is named for what it means, never for the source it came from. When several sources feed a measure, the first listed wins.

## exposure

| Measure | Grain | Unit | Positions | Meaning | Sources |
|---|---|---|---|---|---|
| `exposure.defense_snap_share` | week | fraction | DE DT LB CB S | Share of the team's defensive snaps the player was on the field for. | `nflverse` snap_counts.defense_pct |
| `exposure.defense_snaps` | week | snaps | DE DT LB CB S | Defensive snaps played in the game. | `nflverse` snap_counts.defense_snaps |
| `exposure.offense_snap_share` | week | fraction | QB RB WR TE | Share of the team's offensive snaps the player was on the field for. | `nflverse` snap_counts.offense_pct |
| `exposure.offense_snaps` | week | snaps | QB RB WR TE | Offensive snaps played in the game. | `nflverse` snap_counts.offense_snaps |
| `exposure.special_teams_snaps` | week | snaps | all | Special-teams snaps played in the game. | `nflverse` snap_counts.st_snaps |

## opportunity

| Measure | Grain | Unit | Positions | Meaning | Sources |
|---|---|---|---|---|---|
| `opportunity.air_yards` | week | yards | RB WR TE | Air yards on every pass thrown to the player, caught or not. | `nflverse` stats_player_week.receiving_air_yards |
| `opportunity.carries` | week | count | QB RB WR TE | Designed runs and scrambles. | `nflverse` stats_player_week.carries |
| `opportunity.fg_attempts` | week | count | K | Field goals attempted. | `nflverse` stats_player_week.fg_att |
| `opportunity.pass_attempts` | week | count | QB | Passes thrown, not counting sacks. | `nflverse` stats_player_week.attempts |
| `opportunity.passing_air_yards` | week | yards | QB | Air yards on every pass thrown, complete or not. | `nflverse` stats_player_week.passing_air_yards |
| `opportunity.pat_attempts` | week | count | K | Extra points attempted. | `nflverse` stats_player_week.pat_att |
| `opportunity.targets` | week | count | RB WR TE | Passes thrown to the player. | `nflverse` stats_player_week.targets |

## outcome

| Measure | Grain | Unit | Positions | Meaning | Sources |
|---|---|---|---|---|---|
| `outcome.assisted_tackles` | week | count | DE DT LB CB S | Tackles shared with a teammate. | `nflverse` stats_player_week.def_tackle_assists |
| `outcome.completions` | week | count | QB | Passes completed. | `nflverse` stats_player_week.completions |
| `outcome.defensive_tds` | week | count | DE DT LB CB S | Touchdowns scored on defense. | `nflverse` stats_player_week.def_tds |
| `outcome.fantasy_points` | season | points | all | Fantasy points for the season under this league's scoring, as MFL computes them. | `mfl` playerScores.score |
| `outcome.fg_made_0_19` | week | count | K | Field goals made from 0–19 yards. | `nflverse` stats_player_week.fg_made_0_19 |
| `outcome.fg_made_20_29` | week | count | K | Field goals made from 20–29 yards. | `nflverse` stats_player_week.fg_made_20_29 |
| `outcome.fg_made_30_39` | week | count | K | Field goals made from 30–39 yards. | `nflverse` stats_player_week.fg_made_30_39 |
| `outcome.fg_made_40_49` | week | count | K | Field goals made from 40–49 yards. | `nflverse` stats_player_week.fg_made_40_49 |
| `outcome.fg_made_50_59` | week | count | K | Field goals made from 50–59 yards. | `nflverse` stats_player_week.fg_made_50_59 |
| `outcome.fg_made_60_plus` | week | count | K | Field goals made from 60 or more yards. | `nflverse` stats_player_week.fg_made_60_ |
| `outcome.fg_missed_0_19` | week | count | K | Field goals missed from 0–19 yards. | `nflverse` stats_player_week.fg_missed_0_19 |
| `outcome.fg_missed_20_29` | week | count | K | Field goals missed from 20–29 yards. | `nflverse` stats_player_week.fg_missed_20_29 |
| `outcome.fg_missed_30_39` | week | count | K | Field goals missed from 30–39 yards. | `nflverse` stats_player_week.fg_missed_30_39 |
| `outcome.fg_missed_40_49` | week | count | K | Field goals missed from 40–49 yards. | `nflverse` stats_player_week.fg_missed_40_49 |
| `outcome.fg_missed_50_59` | week | count | K | Field goals missed from 50–59 yards. | `nflverse` stats_player_week.fg_missed_50_59 |
| `outcome.fg_missed_60_plus` | week | count | K | Field goals missed from 60 or more yards. | `nflverse` stats_player_week.fg_missed_60_ |
| `outcome.forced_fumbles` | week | count | DE DT LB CB S | Fumbles forced. | `nflverse` stats_player_week.def_fumbles_forced |
| `outcome.fumble_recoveries` | week | count | DE DT LB CB S | Opponent fumbles recovered. | `nflverse` stats_player_week.fumble_recovery_opp |
| `outcome.fumbles_lost` | week | count | QB RB WR TE | Fumbles lost to the defense, on any play. | `nflverse` stats_player_week.fumbles_lost_total |
| `outcome.interceptions` | week | count | DE DT LB CB S | Passes intercepted. | `nflverse` stats_player_week.def_interceptions |
| `outcome.interceptions_thrown` | week | count | QB | Passes intercepted. | `nflverse` stats_player_week.passing_interceptions |
| `outcome.passes_defended` | week | count | DE DT LB CB S | Passes broken up or intercepted. | `nflverse` stats_player_week.def_pass_defended |
| `outcome.passing_epa` | week | epa | QB | Expected points added on the player's dropbacks, by the nflverse model. | `nflverse` stats_player_week.passing_epa |
| `outcome.passing_first_downs` | week | count | QB | First downs gained by the player's passes. | `nflverse` stats_player_week.passing_first_downs |
| `outcome.passing_tds` | week | count | QB | Touchdown passes. | `nflverse` stats_player_week.passing_tds |
| `outcome.passing_yards` | week | yards | QB | Passing yards. | `nflverse` stats_player_week.passing_yards |
| `outcome.pat_made` | week | count | K | Extra points made. | `nflverse` stats_player_week.pat_made |
| `outcome.qb_hits` | week | count | DE DT LB CB S | Hits on the quarterback as or after he threw. | `nflverse` stats_player_week.def_qb_hits |
| `outcome.receiving_epa` | week | epa | RB WR TE | Expected points added on targets, by the nflverse model. | `nflverse` stats_player_week.receiving_epa |
| `outcome.receiving_first_downs` | week | count | RB WR TE | First downs gained on catches. | `nflverse` stats_player_week.receiving_first_downs |
| `outcome.receiving_tds` | week | count | RB WR TE | Receiving touchdowns. | `nflverse` stats_player_week.receiving_tds |
| `outcome.receiving_yards` | week | yards | RB WR TE | Receiving yards. | `nflverse` stats_player_week.receiving_yards |
| `outcome.receptions` | week | count | RB WR TE | Passes caught. | `nflverse` stats_player_week.receptions |
| `outcome.rushing_epa` | week | epa | QB RB WR TE | Expected points added on carries, by the nflverse model. | `nflverse` stats_player_week.rushing_epa |
| `outcome.rushing_first_downs` | week | count | QB RB WR TE | First downs gained on carries. | `nflverse` stats_player_week.rushing_first_downs |
| `outcome.rushing_tds` | week | count | QB RB WR TE | Rushing touchdowns. | `nflverse` stats_player_week.rushing_tds |
| `outcome.rushing_yards` | week | yards | QB RB WR TE | Rushing yards. | `nflverse` stats_player_week.rushing_yards |
| `outcome.sacks` | week | count | DE DT LB CB S | Sacks, with a shared sack counted as a half. | `nflverse` stats_player_week.def_sacks |
| `outcome.sacks_taken` | week | count | QB | Times sacked. | `nflverse` stats_player_week.sacks_suffered |
| `outcome.safeties` | week | count | DE DT LB CB S | Safeties. | `nflverse` stats_player_week.def_safeties |
| `outcome.solo_tackles` | week | count | DE DT LB CB S | Tackles made alone. | `nflverse` stats_player_week.def_tackles_solo |
| `outcome.tackles_for_loss` | week | count | DE DT LB CB S | Tackles behind the line of scrimmage. | `nflverse` stats_player_week.def_tackles_for_loss |
| `outcome.yards_after_catch` | week | yards | RB WR TE | Receiving yards gained after the catch. | `nflverse` stats_player_week.receiving_yards_after_catch |

## prior

| Measure | Grain | Unit | Positions | Meaning | Sources |
|---|---|---|---|---|---|
| `prior.bench` | player | count | all | 225-pound bench press repetitions at the combine. | `nflverse` combine.bench |
| `prior.broad_jump` | player | inches | all | Broad jump at the combine. | `nflverse` combine.broad_jump |
| `prior.college_carries` | season | count | QB RB WR TE | Carries in the college season. | `cfbd` player_season.rushing.CAR |
| `prior.college_completions` | season | count | QB | Passes completed in the college season. | `cfbd` player_season.passing.COMPLETIONS |
| `prior.college_conference` | season | text | all | The college conference of the player's team that season. | `cfbd` player_season.conference |
| `prior.college_fg_attempts` | season | count | K | Field goals attempted in the college season. | `cfbd` player_season.kicking.FGA |
| `prior.college_fg_made` | season | count | K | Field goals made in the college season. | `cfbd` player_season.kicking.FGM |
| `prior.college_interceptions` | season | count | DE DT LB CB S | Interceptions in the college season. | `cfbd` player_season.interceptions.INT |
| `prior.college_interceptions_thrown` | season | count | QB | Passes intercepted in the college season. | `cfbd` player_season.passing.INT |
| `prior.college_pass_attempts` | season | count | QB | Passes thrown in the college season. | `cfbd` player_season.passing.ATT |
| `prior.college_passes_defended` | season | count | DE DT LB CB S | Passes defended in the college season. | `cfbd` player_season.defensive.PD |
| `prior.college_passing_tds` | season | count | QB | Touchdown passes in the college season. | `cfbd` player_season.passing.TD |
| `prior.college_passing_yards` | season | yards | QB | Passing yards in the college season. | `cfbd` player_season.passing.YDS |
| `prior.college_qb_hurries` | season | count | DE DT LB CB S | Quarterback hurries in the college season. | `cfbd` player_season.defensive.QB HUR |
| `prior.college_receiving_tds` | season | count | RB WR TE | Receiving touchdowns in the college season. | `cfbd` player_season.receiving.TD |
| `prior.college_receiving_yards` | season | yards | RB WR TE | Receiving yards in the college season. | `cfbd` player_season.receiving.YDS |
| `prior.college_receptions` | season | count | RB WR TE | Passes caught in the college season. | `cfbd` player_season.receiving.REC |
| `prior.college_rushing_tds` | season | count | QB RB WR TE | Rushing touchdowns in the college season. | `cfbd` player_season.rushing.TD |
| `prior.college_rushing_yards` | season | yards | QB RB WR TE | Rushing yards in the college season. | `cfbd` player_season.rushing.YDS |
| `prior.college_sacks` | season | count | DE DT LB CB S | Sacks in the college season. | `cfbd` player_season.defensive.SACKS |
| `prior.college_solo_tackles` | season | count | DE DT LB CB S | Solo tackles in the college season. | `cfbd` player_season.defensive.SOLO |
| `prior.college_tackles_for_loss` | season | count | DE DT LB CB S | Tackles for loss in the college season. | `cfbd` player_season.defensive.TFL |
| `prior.college_team` | season | text | all | The college the player played for that season. | `cfbd` player_season.team |
| `prior.college_team_carries` | season | count | QB RB WR TE | The player's college team's carries that season, summed over its players: the denominator of his share. | `cfbd` team_season.rushing.CAR |
| `prior.college_team_interceptions` | season | count | DE DT LB CB S | The player's college team's interceptions that season, summed over its players: the denominator of his share. | `cfbd` team_season.interceptions.INT |
| `prior.college_team_passes_defended` | season | count | DE DT LB CB S | The player's college team's passes defended that season, summed over its players: the denominator of his share. | `cfbd` team_season.defensive.PD |
| `prior.college_team_passing_yards` | season | yards | QB | The player's college team's passing yards that season, summed over its players: the denominator of his share. | `cfbd` team_season.passing.YDS |
| `prior.college_team_receiving_yards` | season | yards | RB WR TE | The player's college team's receiving yards that season, summed over its players: the denominator of his share. | `cfbd` team_season.receiving.YDS |
| `prior.college_team_receptions` | season | count | RB WR TE | The player's college team's receptions that season, summed over its players: the denominator of his share. | `cfbd` team_season.receiving.REC |
| `prior.college_team_rushing_yards` | season | yards | QB RB WR TE | The player's college team's rushing yards that season, summed over its players: the denominator of his share. | `cfbd` team_season.rushing.YDS |
| `prior.college_team_sacks` | season | count | DE DT LB CB S | The player's college team's sacks that season, summed over its players: the denominator of his share. | `cfbd` team_season.defensive.SACKS |
| `prior.college_team_tackles_for_loss` | season | count | DE DT LB CB S | The player's college team's tackles for loss that season, summed over its players: the denominator of his share. | `cfbd` team_season.defensive.TFL |
| `prior.college_team_total_tackles` | season | count | DE DT LB CB S | The player's college team's total tackles that season, summed over its players: the denominator of his share. | `cfbd` team_season.defensive.TOT |
| `prior.college_total_tackles` | season | count | DE DT LB CB S | Total tackles in the college season. | `cfbd` player_season.defensive.TOT |
| `prior.draft_pick` | player | pick | all | The overall pick the player was drafted with. | `nflverse` draft_picks.pick |
| `prior.draft_round` | player | round | all | The round the player was drafted in. | `nflverse` draft_picks.round |
| `prior.draft_year` | player | season | all | The year the player was drafted. | `nflverse` draft_picks.season |
| `prior.forty` | player | seconds | all | 40-yard dash at the combine. | `nflverse` combine.forty |
| `prior.height` | player | inches | all | Height. | `nflverse` players.height |
| `prior.shuttle` | player | seconds | all | 20-yard shuttle at the combine. | `nflverse` combine.shuttle |
| `prior.three_cone` | player | seconds | all | Three-cone drill at the combine. | `nflverse` combine.cone |
| `prior.vertical` | player | inches | all | Vertical jump at the combine. | `nflverse` combine.vertical |
| `prior.weight` | player | pounds | all | Weight measured at the combine. | `nflverse` combine.wt |

## availability

| Measure | Grain | Unit | Positions | Meaning | Sources |
|---|---|---|---|---|---|
| `availability.game_status` | week | text | all | The team's final injury designation for the game: Out, Doubtful or Questionable. | `nflverse` injuries.report_status |
| `availability.practice_status` | week | text | all | The player's last practice participation before the game. | `nflverse` injuries.practice_status |

## context

| Measure | Grain | Unit | Positions | Meaning | Sources |
|---|---|---|---|---|---|
| `context.birth_date` | player | text | all | Date of birth, as YYYY-MM-DD. | `nflverse` players.birth_date |
| `context.nfl_position` | player | text | all | The player's position as nflverse lists it, finer than the league's (OLB, NT, FS). | `nflverse` players.position |
| `context.rookie_season` | player | season | all | The first NFL season the player was on a roster. | `nflverse` players.rookie_season |
| `context.team` | week | text | all | The NFL team the player played the game for. | `nflverse` snap_counts.team |

## Feeds

Files the table-driven loader reads. A field named `<feed>.<column>` reads that column.

| Feed | Source | File | Seasons | Player id | Rows kept |
|---|---|---|---|---|---|
| `stats_player_week` | `nflverse` | `stats_player_week_{season}.csv` | 2021 on | `player_id` (gsis) | `season_type` is REG |
| `snap_counts` | `nflverse` | `snap_counts_{season}.csv` | 2021 on | `pfr_player_id` (pfr) | `game_type` is REG; `position` is not C, G, T, OT, OG, OL, LS, P |
| `injuries` | `nflverse` | `injuries_{season}.csv` | 2021 on | `gsis_id` (gsis) | `game_type` is REG |
| `players` | `nflverse` | `players.csv` | one file | `gsis_id` (gsis) | all |
| `draft_picks` | `nflverse` | `draft_picks.csv` | one file | `pfr_player_id` (pfr) | all |
| `combine` | `nflverse` | `combine.csv` | one file | `pfr_id` (pfr) | all |

## Sources

A source is lost when its latest load failed and it has had no successful load within its window.

| Source | Name | Status | Lost after | Fetched from |
|---|---|---|---|---|
| `mfl` | MyFantasyLeague API (league 14432) | active | 7 days | `myfantasyleague.com` |
| `nflverse` | nflverse data releases | active | 14 days | `github.com/nflverse/nflverse-data/` |
| `dynastyprocess` | DynastyProcess player id crosswalk | active | 14 days | `raw.githubusercontent.com/dynastyprocess/` |
| `cfbd` | CollegeFootballData API | active | 30 days | `api.collegefootballdata.com` |
| `madden` | EA Madden ratings | retired | 30 days | `ratings-api.ea.com` |
