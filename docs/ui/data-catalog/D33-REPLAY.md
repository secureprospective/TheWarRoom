# D33 replay: lineup delta of a trade (MODEL_METHODS_DRAFT §6)

Run: `go run ./tools/d33replay -dir ~/fleet/runs/d33-replay` (under 1 s). Data: the real league's
MFL exports (L=14432), 2023–2025, read-only: trades, weeklyResults for weeks 1–17 (starters,
MFL's optimal lineup, every rostered player's league-scored points), players and league starters.

Method under test = the shipped code: `lineupdelta.Compute` (the existing `powerrankings.Lineup`
fill), with each player valued at the D11 median (`projection.NewShape` + `Project`, defaults),
fit only on weeks before the trade. Here the median comes from league-scored weeks rather than
nflverse, because MFL lists the league's own points for every rostered player.

Trades judged: player trades in week windows 5–9 (4 weeks either side inside the 13-week regular
season) where both rosters still held the players the week before. 164 trade sides, 159 with a lineup change.

| Bar | Need | Result | |
|---|---|---|---|
| (i) predicted ≥ +2.0 PPG → team actually better | majority, sign test p ≤ .05 | the trade's own effect up in 30 of 41 (73%, p = .002) | PASS |
| | | MFL optimal points, 4 weeks after vs before: 22 of 41 (54%, p = .38) | not significant |
| (ii) calibration | \|r\| ≥ .4, slope .7–1.3 | the trade's own effect: r = .548, slope 1.06 | PASS |
| | | MFL optimal points change: r = .219, slope 1.87 | FAIL |
| (iii) model-"benched" starters lose starts next week (owners' real lineups) | ≥ 70% | 52 of 81 (64%); only players the owner was starting: 33 of 54 (61%) | FAIL |
| (iv) this-week and rest-of-season deltas agree in sign | 100% | 131 of 131 | PASS |

"The trade's own effect" = both rosters as of the week before the trade, best lineup at the next
4 weeks' actual scores, after minus before: it isolates the trade from every other move. The MFL
optimal change also carries every waiver claim, injury and other trade in those 8 weeks, which is
why it is noisy.

## What this means for the screen
- The PPG number is calibrated (slope ≈ 1): show it as is.
- Bar (iii) fails: owners keep starting about 4 in 10 of the players the best lineup would bench.
  So the tile must not say "X goes to your bench". Say "your best lineup would bench X". Not tuned:
  this measures owner behaviour, not the projection.

## Debt
- Only 41 sides predicted ≥ +2; 2023 trades have no prior seasons (thin projections).
- Picks in trades are ignored (D33 is lineups only).
- The IR return-week rule and the rookie tier fallback (§6 failure modes) are not built: an
  unprojected player counts at 0 and is named in `unprojected`.
