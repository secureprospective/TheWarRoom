# Official 2026 week-1 fixtures (synthetic)

Used by `TestSeasonInABoxSpike` (`cmd/twr/season_in_a_box_test.go`) to pin official statistics with no network.

- `stats_player_week_2026.csv.gz`: nflverse-shaped weekly statistics. Two rows, both in game `2026_01_DAL_NYG`: QB `00-0000099` (franchise.01's starter, 250 passing yards, 2 passing TDs) and QB `00-0005002` (franchise.04's starter, the 2027 rookie the box drafts; every stat `0`). Every other stat column is `0`; every `_list` column is empty.
- `stats_player_week_2026_corrected.csv.gz`: the same file with passing yards 250 -> 262 (the stat correction); the second row is unchanged.
- `play_by_play_2026.csv.gz`: one pass play (250 yards) and one `END GAME` play for the same game.

The header of the weekly file is copied from `internal/fantasyscore/source/testdata/e5_weeks.csv.gz` (2025 data). The rest is made up. These are not real 2026 statistics.
