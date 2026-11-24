# Official 2026 week-1 fixtures (synthetic)

Used by `TestSeasonInABoxSpike` (`cmd/twr/season_in_a_box_test.go`) to pin official statistics with no network.

- `stats_player_week_2026.csv.gz`: nflverse-shaped weekly statistics. Two rows, both in game `2026_01_DAL_NYG`: QB `00-0000099` (franchise.01's starter, 250 passing yards, 2 passing TDs) and QB `00-0005002` (franchise.04's starter, the 2027 rookie the box drafts; every stat `0`). Every other stat column is `0`; every `_list` column is empty.
- `stats_player_week_2026_corrected.csv.gz`: the same file with passing yards 250 -> 262 (the stat correction); the second row is unchanged.
- `play_by_play_2026.csv.gz`: one pass play (250 yards) and one `END GAME` play for the same game.

Synthetic playoff files used by `TestSeasonInABoxPlayoff` (`cmd/twr/season_box_playoff_test.go`), covering weeks 1–3 for the four seeded teams' starters (`00-0000099`, `00-0000102`, `00-0006001`, `00-0005002`) in synthetic games `2026_01_DAL_NYG`, `2026_02_DAL_NYG`, and `2026_03_DAL_NYG`:

- `stats_player_week_2026_playoff_w1.csv.gz` and `play_by_play_2026_playoff_w1.csv.gz`: week-1 regular-season stats.
- `stats_player_week_2026_playoff_w2.csv.gz` and `play_by_play_2026_playoff_w2.csv.gz`: cumulative weeks 1–2 (semi-finals).
- `stats_player_week_2026_playoff_w3.csv.gz` and `play_by_play_2026_playoff_w3.csv.gz`: cumulative weeks 1–3 (championship final).

The header of the weekly file is copied from `internal/fantasyscore/source/testdata/e5_weeks.csv.gz` (2025 data). The rest is made up. These are not real 2026 statistics.
