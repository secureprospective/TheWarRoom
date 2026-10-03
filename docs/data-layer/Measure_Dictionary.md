# Measure Dictionary

Generated from `internal/measures/*.csv` by `make measure-dictionary`. Do not edit it by hand: change the CSV files and regenerate.

Every number in `history.db` is stored as player · season · week · measure. Week 0 holds season-level values. A measure is named for what it means, never for the source it came from. When several sources feed a measure, the first listed wins.

## outcome

| Measure | Grain | Unit | Positions | Meaning | Sources |
|---|---|---|---|---|---|
| `outcome.fantasy_points` | season | points | all | Fantasy points for the season under this league's scoring, as MFL computes them. | `mfl` playerScores.score |

## Sources

A source is lost when its latest load failed and it has had no successful load within its window.

| Source | Name | Status | Lost after | Fetched from |
|---|---|---|---|---|
| `mfl` | MyFantasyLeague API (league 14432) | active | 7 days | `myfantasyleague.com` |
| `nflverse` | nflverse data releases | active | 14 days | `github.com/nflverse/nflverse-data/` |
| `dynastyprocess` | DynastyProcess player id crosswalk | active | 14 days | `raw.githubusercontent.com/dynastyprocess/` |
| `cfbd` | CollegeFootballData API | active | 30 days | `api.collegefootballdata.com` |
| `madden` | EA Madden ratings | active | 30 days | `ratings-api.ea.com` |
