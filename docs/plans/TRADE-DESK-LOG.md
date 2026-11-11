# Trade desk: the math, where it was, where it is

**By:** Christopher Campbell and Claude · **Date:** 2026-11-11 · **Status:** passable; needs its own session.

Code: `internal/tradelens/tradelens.go` (the math), `internal/app/market/trade_lens.go` (league data in), `tools/fit` (the keep share). PR #227.

## The idea

Replace generic trade calculators with a read built from this league's own rules, rosters and drafts. Every asset is worth what it adds to a franchise, seen through that owner's direction. Two depth pieces never add up to a star.

## The math today

**1. Lens.** Each owner's direction sets how much "this season" counts against "long term":

| Direction | Now weight |
|---|---|
| Contend | .65 |
| Retool | .50 |
| Rebuild | .30 |

`blend = w × now + (1 − w) × dynasty`. The weights are the league ruling (tested with owners). `now` and `dynasty` come from the existing player model; the desk fits nothing of its own there.

**2. Replacement level.** Per position: rank every valued player in the league by `blend`; the floor is the best one left after the league has taken as many as it rosters at that position. Roster room draws the line, not the lineup.

**3. Lead that lasts.** `over = keep[pos] × max(0, blend − floor[pos])`. Keep is the share of a lead that carries into the next season, fitted from history (slope of next season's points per game on this season's, 6+ games both years):

| QB | RB | WR | TE | K | DT | DE | LB | CB | S |
|---|---|---|---|---|---|---|---|---|---|
| .496 | .806 | .797 | .811 | .185 | .675 | .661 | .743 | .671 | .634 |

**4. Draft supply.** `drafted[pos]` = the average `over` of every player this league has taken in its own drafts at that position. It is what a pick spent on that position delivers. Real league, retool lens: CB 1.5, DE 1.6, DT .6, LB 3.0, PK 1.0, QB 2.5, RB 2.1, S 3.1, TE 1.1, WR 2.0.

**5. Player worth.** `worth = max(0, over − drafted[pos])`. A position every draft restocks is cheap; one it rarely fills is dear.

**6. Pick worth.** `worth = Weight × (best comp group's average over)`. Comp groups are the players this league really took where the pick sits, then each later round's (a pick is never worth less than a later one). `Weight = 1.2 × discount^(seasons out)`. The 1.2 is what this league's owners paid for picks against what they added in 2026 trades. No draft-supply subtraction and no roster bar: a pick is the supply itself. Unslotted picks are placed by the original team's record today.

**7. Roster count.** `count = best N player worths + every held pick's worth`, N being the league's starter count.

**8. The read.** For each side: `net = count(after) − count(before)`, under that side's own lens. Under ±0.05 is level. Both level or better: "both win". You fine, they lose: they balk. You lose: bad. Tune suggestions try every single trim, ask or sweetener the other side would still take.

**9. Unset direction.** Our read from record: 60%+ wins is contend, 35% or less is rebuild, older-than-median roster at .500+ is contend, else retool.

## Where it was

| Step | What it did | Why it moved on |
|---|---|---|
| 1 | Plain blend, best N, no replacement level | Depth and stars read alike; picks had no value |
| 2 | Picks valued from the league's drafted players, 1.2 market scale, slots guessed from records | Positions all priced alike |
| 3 | Positional scarcity at the lineup line | Backups worth nothing; deep positions overpriced |
| 4 | Replacement at the roster line, fitted keep share, picks priced without the holder's roster bar | Positions the draft restocks every year still too dear |
| 5 | Draft supply subtracted from players (today) | Reads 1 to 3 pass |

Tried and refuted, do not retest: draft supply as a multiplier on `over`; the draft floor applied to pick comps as well as players. Both failed the reads.

## Where it is

Four real league proposals, graded by Christopher against his own read:

| Read | Desk (you / them) | Grade |
|---|---|---|
| 1 | −3.2 / +2.6 | A+ |
| 2 | 0.0 / 0.0, both win | A+ |
| 3 | −2.6 / +0.9 | A−, within error |
| 4 | +15.8 / −16.0 | Miss |

## Open for the dedicated session

1. **Read 4: backup and veteran quarterbacks.** The player model values a backup at his points per game when he plays, so the desk overprices him. Likely a player-model fix (playing-time share), not a desk fix.
2. **League-size scale.** Logged, not built.
3. **Scarcity stays out of power rankings.** Ruled: it belongs in the trade desk and draft value only. Power rankings rework is separate.
4. **The 1.2 pick market scale** rests on one season of trades. Recheck with more.
5. **More reads.** Four is a small test. Bring more real proposals with a human grade each.
