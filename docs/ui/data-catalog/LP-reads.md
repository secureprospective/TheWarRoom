# League Pulse → reads (part 1: Now and Race)

Target: `docs/ui/mocks/league-pulse-mock-v14.html` (LOCKED). Regions are the rows of
`terra-LP.md`. "Read" is the existing native read that feeds the region — every one is an
existing command in `internal/commandcatalog/native_reads.go`; nothing new was added to the
engines, the models or the stores. The Archive tab and the drawers are out of scope for part 1.

## Wired

| Region | Read | Field |
|---|---|---|
| Shell › League label | `native.directory.read` | `name` |
| Shell › League week | `week.read` | `current.week`, else `nextWeek` |
| Shell › Leading shortcut | `results.read` | my `scores[].points` vs my opponent's |
| Strip › Week label, Previous/Next week | `week.read` | `schedule.games[].week` bounds the range |
| Strip › Slate final | `results.read` | games whose `status` is not `No result yet` |
| Strip › Slate pending | `results.read` | games whose `status` is `No result yet` |
| Strip › Freshness | `native.directory.read`, `results.read` | `revision`, `words` |
| Strip › My score, Opponent score | `results.read` | `scores[].points` |
| Matchups › Week header, Row teams, Row scores | `results.read` | `games[]` |
| Matchups › Row status | `results.read` | `status` + whether the week is the current one |
| Matchups › All / Mine / Live / Close filters | `results.read` | derived from `games[]` |
| Matchups › Section headers | `native.directory.read` | `franchises[].division` |
| Matchups › Empty filter | — | honest empty state |
| Matchups › Scoreboard names, scores, status | `results.read` | `games[]`, `scores[]` |
| Live › Sidebar mode, Matchup filters, Matchup list, Matchup empty | `results.read` | `games[]` |
| Live › Starters heading, Starter total, Starter team header | `results.read` | `scores[]` |
| Live › Starter slot, name, fantasy points | `results.read` | `scores[].starters[]` |
| Live › Starter NFL team | `lineup.read` | `starters[].team` |
| Live › Starter game chip | `lineup.read` | `starters[].locked` |
| Live › Starter projected points | `projections.read` | `players[].median` |
| Race › Standings view, Seed, Franchise, Division, Record, Points for | `standings.read`, `bracket.read` | `computed[]`, `fields[].seeds[]` |
| Race › Conference switch | `bracket.read` | `fields[].conference` |
| Race › Division header, Division leader | `standings.read` | `computed[].division`, `rank` |
| Race › My seed summary | `bracket.read` | my seat in `fields[].seeds[]`, `bye` |
| Race › My road week, opponent, record | `week.read`, `standings.read` | `schedule.games[]`, `computed[].record` |
| Race › Race mode, Race context | `bracket.read` | `rules.startWeek`, `rules.finalWeek`, `words` |

## No existing read (reported, not built)

Each of these renders a calm empty state naming what will fill it.

- **A live NFL game feed for an app league.** Shell › Live count. Strip › Slate timestamp,
  Slate live. Live › NFL filters, NFL groups, NFL row teams/scores/possession/clock,
  NFL exposure yours/theirs, Following chips, Game header away/home/clock, Field end zones /
  drive / ball / first down / possession / down-distance / last play / empty, Game rostered
  count, Game score contribution yours/theirs, Game players name/ownership/points/empty,
  Starter on field, Play-by-play clock/description/owner/points/empty/header, Tick
  notification, Starter compact stats. `internal/app/now` serves these for the MFL mirror from
  `leaguefeed`; no native read exposes them for an app league.
- **A win-probability model.** Strip › Win probability bar. Matchups › Row win probability,
  Scoreboard win chances. Live › matchup list win columns.
- **A season simulation.** Race › Playoff odds, My playoff odds, My bye odds, Magic number,
  Scenario own win / own loss / rival loss bye / rival loss change, Run-in projected seed /
  projected wins / opponent chance / odds / bubble / difficulty / tooltip, Projected bracket
  header / slots / opponent.
- **A power ranking for a league season.** Race › Power rank, Power rank row, Power movement,
  Power team, Power score, Power window badge, Power component ranks, Power season/dynasty rank
  and score, Power shift, Power peak year, Power vs record, Power horizon, Power explanation,
  Power weight (+ tooltip), Movers period / franchise / delta, Weekly / season / year mover
  reason, Season mover chart and tooltip, Season form / lineup / health / all-play / run-in
  components, Dynasty value / age / picks / cap / depth components. `standings.read` names a
  power rank among its tie-breakers and reports that none is held.
- **An all-play record and points against.** Race › My road chance (opponent strength),
  Inspector season all-play. `standings.read` carries `pointsFor` only.
- **Games back.** Race › Games back. The standings read does not carry it.
- **Projected franchise totals.** Matchups › Scoreboard projections, Starter projected total.
  `projections.read` is per player; no read totals a lineup.
- **Player counts by game state.** Matchups › Scoreboard player counts, Player game state.
  Needs the NFL feed above.
- **The Archive tab, the drawers, the inspector and the command hint.** Out of scope for part 1.
