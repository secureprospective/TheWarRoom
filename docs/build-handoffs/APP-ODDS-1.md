# APP-ODDS-1: win %, power and playoff odds from the app's own league data

Branch `build/ODDS-1` (from `73a9fd15`), worktree `~/work/TheWarRoom/.worktrees/odds` on Claude-OS.

## 1. The map: where each number was computed, what fed it, why it was MFL-only

Line numbers are at `73a9fd15`.

| Number | Screen before | Computed in | Fed by | Why only MFL |
|---|---|---|---|---|
| WIN % (matchup) | `frontend/src/app/shell/homeTiles.tsx:197-199` hardcoded `WIN % · Not available yet`; its hover reads "computed only for the MFL mirror's Now board (M-018)" | `internal/powerrankings/lineup.go:87` `WinProbability`, called only by `internal/app/now/nfl_reading.go:335` | MFL live `leaguefeed` side points on the mirror's Now board | Its only caller reads MFL feed types; no native read called it |
| POWER | `homeTiles.tsx:46` tile `missing: 'No power ranking read for an app league'`; `HomeBrief.tsx:122` and `:326` hardcoded `—`; `PulseRace.tsx:55` "A power ranking is not held" | `internal/composition/power_rankings_app.go:41` → `internal/app/scoring/power.go:51` `Source.Build` → `internal/m2service/m2service.go:107` `BuildBoard` (`buildBlendInputs` :217, `parseStanding` :355) → `powerrankings.Blend` | `[]leaguestandings.RawStanding` from `composition/mfl_feed_cache.go:27` `standingsOrArchive` (MFL fetch or MFL archive) plus history model runs | The inputs are MFL ingestion types fetched with `a.core.MFLClient` / `ingestion.LeagueID` (`race_app.go:34,92,100`). `engine/race/race.go:61` `Line.Power` is "MFL's power rating" |
| PLAYOFF ODDS | `homeTiles.tsx:223` hardcoded `—`; `PulseRace.tsx:169` "need a season simulation this league does not run yet" | No odds or simulation existed anywhere. The nearest thing was `internal/m2service/outlook.go:88` `projectedWins` (expected wins only) | MFL standings plus MFL `leagueschedule.RawScheduleWeek` | Same MFL-typed inputs, and no playout at all |

Native engines that already existed and are now wired: `powerrankings` `Count` / `ExpectedPoints` / `WinProbability` / `Blend` / `AutoRosterWeight` / `GameSpread`; `lineupdelta.Rules` (native lineup rules → lineup fill); `engine/competition` `RankStandings` (the league's tie-breaker line) and `SeedField` (the bracket's field rules); the commissioner's `NativeStandingsReading` / `bracketReader` (rules, conferences, hand seeds); and the prepared native value read `appvalue.Reading.NowPPG`.

## 2. What was built (all leagues; there is no MFL/imported branch)

- `internal/commissioner/odds_reading.go` `NativeOddsReading`:
  - **Expected score** per franchise = `ExpectedPoints(best legal lineup of NowPPG, PF, games)`. This is the same formula the MFL outlook uses.
  - **Win %** for each game in the first open week = `WinProbability(expected home, expected away)`.
  - **Power** = `Blend` of lineup value with results (all-play %, or PF/maxPF when no week was all-played) at `AutoRosterWeight(games)`. These are the M2 season-view rules.
  - **Playoff odds**: 2,000 play-outs of every open regular-season game.
    - Each side's score is drawn as N(expected, GameSpread/√2), so the margin has exactly the spread `WinProbability` reads (tested).
    - Each played-out season is ranked with `RankStandings` under the league's criteria.
    - A group the full line leaves level is broken by a coin flip drawn from the recorded `seed`.
    - The field is seeded with `SeedField` under the bracket's own rules (per-conference size, division winners, wild cards, hand seeds).
    - The runs are split into 8 fixed streams, so results are identical on any machine.
  - **Holds back with words, never zeros**:
    - Values missing or still preparing: no power, win % or odds.
    - Schedule published for fewer weeks than the regular season: no odds.
    - Playoff rules needing a ruling: no odds.
    - Commissioner hand standings: no odds.
- `internal/store/history` `Store.NativeOdds`. `internal/leagueservice/odds_api.go` `TargetNativeOdds` admits like `standings.read` and reads values through the prepared value read (franchise `""` = every franchise).
- Command `odds.read`: league-view, G0, gm/commish/admin, workflow + desktop doors. It is added to the catalog, bindings, release matrix and door fixtures. Generated surfaces were refreshed.
- Frontend:
  - `readNativePulse` reads `odds.read`.
  - Home THIS WEEK shows WIN % and the bar, only for the tile's week.
  - Home RACE shows PLAYOFF ODDS and POWER.
  - The POWER tile is live.
  - The header and inspector show POWER.
  - Pulse Race shows SEEDS PWR/ODDS columns, MAKE PLAYOFFS · sim, field-row odds bars and the POWER table.

## Verification

- `-race`, one package at a time with `-p 2` (`go test -list` count = top-level PASS):
  - commissioner: 8/8 (12 sub)
  - store/history: 24/24
  - leagueservice: 4/4
  - commandcatalog: 12/12
  - app/commissioner: 12/12 (416 sub)
  - serverhttp: 4/4 (180 sub)
  - composition door tests: 6/6 (38 sub)
- `vitest src/app`: 180 files, 1837 tests passed.
- `tsc --noEmit`: clean.
- `go run ./tools/boundarycheck`: passes. Only the `#source` rows of `app/core/host_commissioner.go` and `composition/command_bindings.go` were re-recorded, after review; both changes are straight-line wiring.
- `make build`: passes.

## Debt left

1. **Speed**: `odds.read` takes about 285–296 ms on the 32-team fixture at load 2.9 (standings alone about 32 ms) and about 516 ms at load 6. That is over the 200 ms budget. `RankStandings` × 2,000 dominates. Options, each Christopher's call:
   - Cache by ledger revision plus stat-evidence revision. Not done, because a revision-only key could serve stale odds.
   - Precompute after each final.
   - Drop to 1,000 runs (±1.6 pts).
2. **Partly decided weeks**: in a week where some games are already decided, the all-play of the played-out games counts only the simulated teams.
3. **`power_rank` tie-breaker**: `competition/standings_rank.go` still skips it ("no power rank is held"). The native power now exists, but wiring it into standings changes the league's rule outcome and was left alone.
4. **Not built**: PROJECTED seed, first-round-bye chance, magic number, scenarios, run-in opponent strength, and the offseason POWER · DYN (franchise view).
5. **Coin flip**: the play-outs record their flip seed. The live standings have no recorded coin-flip ledger entry for a real final-day level tie; `RankStandings` still keeps league order.
6. **No composition-level test** calls `TargetNativeOdds` through a real app. It is covered at store, service and doors level.
