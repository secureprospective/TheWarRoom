# APP-SCHED-10

Continues `APP-SCHED-9-evidence.md` from 61c50a5c6 on `build/SCHED-1`. Same 148-row cohort,
same disposable copy of the SQLite profile, no production writes and no MFL submission.

## Item 1 — not on any NFL roster that week is an evidence-backed zero

`source.UnrosteredEvidence` = **"on no NFL roster that week"**. A player no NFL team listed
that week cannot have played, so the week is a recorded zero rather than a pending gap. It is
admitted only when all of this holds:

- The deciding **dated membership snapshot** is the newest held historical depth snapshot at
  most seven days before that week's earliest held game — the same `depthAt` window the
  schedule-bye zeros use. No current affiliation is carried backwards.
- That snapshot **covers every team the season's membership history knows**. A snapshot short
  of a team cannot say a player is on none of them. For 2026 each of the seven deciding
  snapshots lists all 32 teams, matching the season history's 32.
- The player is **on no team in it**, and the season's history **lists him somewhere**: a
  player it never lists is outside the file's coverage, so he stays pending rather than
  becoming a zero by the file's silence.
- He took **no snap that week** on any team. A snap means he was on a roster whatever the
  snapshot says.
- The existing refusals still apply: any play event that week wins, and an identity that is
  not a GSIS id stays pending.

Such a proof names **no team and no game**, because the player belonged to neither.
Validation now requires identity and period alone for it and refuses one that does name a
team or game; admission no longer looks for an end-of-game record it could not have. Scoring
is untouched — the zero arrives as an ordinary recorded-zero player week on the same path as
the bye and snap-coverage zeros.

**Why the membership file alone is not treated as a roster file.** Across weeks 1–7, 30 to 46
players a week who have stat rows are absent from that week's deciding snapshot (2.9%–4.3%),
so absence from it is not by itself proof of absence from a roster. Those players are already
excluded, because a stat row or a snap refuses the proof before it is made. What the chain
above adds is the two cases the file could still get wrong, and both are refused: a player the
feed does not cover at all, and a player whose membership the file dropped for one week.
Independent corroboration on the 73 cohort rows admitted: **none** appears in both adjacent
weeks' snapshots on the same team (no one-week dropout), and **none** appears on any team's
injury report for that week (an injury report is itself roster evidence).

**Re-check of every `unavailable` row in `APP-SCHED-9-absent.tsv` (198 occurrences):**

| Disposition | Occurrences |
|---|---:|
| on no NFL roster that week | 73 |
| complete snaps + pbp, no snap (newly admitted, items 2 and 3) | 34 |
| still pending | 84 |
| identity gap (MFL 0835, unchanged) | 7 |

Cross-check: MFL holds **no `playerScores` row at all** for any of the 198, so MFL counts
every one of them as nothing. Each zero admitted here agrees with MFL's own treatment.

Of the 84 still pending, **55** are players the 2026 membership history never lists on any
team in any snapshot (Mattison, Shenault, Claypool, Peppers, Milano, Baker, Tre'Davious White
and others). They are almost certainly unsigned all season, but the file that would prove it
does not cover them. The exact missing evidence is **nflverse weekly rosters**
(`rosters_weekly_2026`): not held in the archive and not adapted. The remaining 29 are on a
roster that week per the deciding snapshot, with no participation evidence either way.

## Item 2 — the three absent Bills lines

None is an identity or lineup problem: all three are on their team's dated depth snapshot, all
three are listed **Questionable** (not Out), none has a stat row, and none has a snap row, so
each was inactive. Each was blocked by the snap-coverage completeness gate, for two distinct
reasons, both fixed.

| Row | Team file | Gate before | After |
|---|---|---|---|
| W4 Terry McLaurin | WAS special teams 310 counts over 28 plays | refused: 310 is not divisible by 11 | admitted: excess cannot come from an omission |
| W7 Calvin Ridley | TEN defense 771 over 70 plays; CLE special teams 298 over 27 | refused: neither divides by 11 | admitted: both are excesses |
| W6 Tyler Warren | IND special teams 309 over 28 plays (excess); **TEN** 307 over 28 (deficit) | refused: the whole game had to reconcile | admitted: TEN's short file is no evidence about an IND player |

**Divisibility by eleven was the wrong invariant.** A unit's rounded shares hold its play count
independently of the counts, so the play count is recovered from them and must be the only one
every share fits. Totals then deviate from eleven-per-play in both directions: of 216
team-game units, offense is exact 209 times, defense 185, special teams 162; the rest are
+1/+2/+3 excesses (20 units) and −1 deficits (51 units), plus a handful of large deficits
(−6, −11, −13, −19, −70) that are plainly omitted players. An **excess cannot be produced by
leaving a player out** — omission only ever subtracts — so it is admitted as an over-manned
play. A **deficit stays refused**: a short-handed play and an omitted one-snap player look
identical in the aggregate, and that is exactly what the gate exists to catch.

**The opponent's file was the wrong scope.** Whether a player took a snap for his team is
answered by his team's rows. Requiring both halves of a game to reconcile let one team's short
file withhold the other team's proven absences. Each team's file is now judged on its own, and
a team whose rows are missing from the file still certifies nothing. Replacing that guard, a
file holding more than eleven players who took every play of a unit is refused: no more than
eleven can have, so those are somebody else's rows. Real 2026 team-games peak at seven
full-time players on offense and six on defense, so the cap only catches a file that is not
one team's.

Effect: complete NFL games rise from **47 to 57 of 108** under the recovered-play-count rule
alone, a superset of the 47 — nothing that reconciled before stops — and per-team judgement
adds the rest of the 34 newly admitted zeros.

## Item 3 — the 15 half-point differences

Every residual is a whole multiple of 0.5 and every cent part matches exactly, so each one is
a whole number of units of one stat under one rule, not a rate or rounding error. The league's
own held `TYPE=rules` body gives the relevant rates, and MFL stacks a position's block on top
of the general block, which the app reproduces rule by rule:

| Stat | MFL event | QB RB WR TE PK | LB | DE DT | CB S |
|---|---|---:|---:|---:|---:|
| Tackle | TK | 1.5 | 1.5 | 2.5 | 2.0 |
| Assisted tackle | AS | 1.0 | 1.0 | 1.5 | 1.0 |
| Pass defensed | PD | 2.5 | 2.5 | 2.5 | 3.0 |
| Tackle for loss | TKL | 2.5 | 2.5 | 2.5 | 2.5 |

Those rates are why the residuals quantise at 0.5: one tackle re-read as an assist is 0.5 for
an LB, and 1.5/1.0/2.5/2.0/3.0 in combination produce every observed value.

**The app does not misread the feed.** Read through the app's own per-stat attribution
(`store.NativePlayerScore`, which names each stat, its recorded quantity and the rule that
priced it) for all 46 differing player-weeks, the app's play-by-play-derived counts of
tackles, assists, sacks, passes defensed and tackles for loss agree with **nflverse's own
aggregated weekly file** — an independent nflverse product — on **46 of 46 rows, exactly**.
The difference is between MFL's stat provider and nflverse, in tackle/assist attribution,
with no systematic direction: 26 of the 46 rows have MFL higher and 20 lower (the 0.01 rounding row
is one of the 26).
Per the brief, ours stands and the difference is reported.

Per-row stat-by-stat diagnosis is `APP-SCHED-10-stat-differences.tsv`: the app's full stat
line, the residual, and the stat shift on MFL's side that accounts for it under the rule
named above. 39 of the 46 land on tackles and assists; the others are:

| Row | Residual | Stat and rule |
|---|---:|---|
| W1 Will Reichard, PK | MFL +1.50 | MFL credits the kicker a tackle (TK ×1.5); nflverse credits none |
| W3 Mike Evans, WR | MFL +2.00 | 20 receiving yards (CY ×0.1), 2 receptions (CC ×1) or a two-point catch (C2 ×2) — the residual does not single one out |
| W4 Hunter Henry, TE | MFL +1.00 | one reception (CC ×1) or 10 receiving yards (CY ×0.1) |
| W4 Dalton Schultz, TE | MFL +1.00 | as above |
| W4 Tarheeb Still, CB | MFL +2.00 | one tackle (TK ×2.0 at CB) |
| W5 Tyrone Tracy, RB | MFL −2.00 | MFL counts a fumble we do not (FU ×−2) |
| W7 Markquese Bell, S | MFL +3.50 | one tackle for loss (TKL ×2.5) and one assist (AS ×1.0) |

**One residual is not a half point and is ours: 0.01 on W1 Chimere Dike.** His punt return
yards are −5 at 0.025 a yard = −0.125 points. The app rounds each stat to the cent away from
zero (`fantasyscore.roundCent`, `remainder <= -50`) and gets −0.13, total 1.12; MFL rounds the
half up and gets −0.12, total 1.13. Every other stat in the row matches. **Left as it is and
reported:** rounding direction is not a league setting today, the correction is a scoring
convention change for every league rather than a misread setting or feed, and this row's
team-week is already classified *within .01*, so it accounts for none of the 15 differences.
It is the only negative half-cent in the cohort.

## The same 148-row comparison

Full rows: `APP-SCHED-10-comparison.tsv`.

Replay: SQLite `.backup` of the SCHED-9 disposable profile into `.tmp/sched11/profile`, then
schedule + lineup backfill, `ProcessStatFeed`, `NativeResults` weeks 1–7 and `NativeStandings`,
held bodies only (no `SyncScoring`, network refused). PASS in 2754 s. No production writes.

| Comparison | SCHED-9 | SCHED-10 |
|---|---:|---:|
| exact | 61 | **82** |
| within .01 | 1 | 1 |
| differ | 15 | 20 |
| unavailable | 71 | **45** |

No row went backwards: every SCHED-9 exact or within-.01 row is unchanged. The five new
`differ` rows (W1, W3, W5 Bills; W4 Titans; W7 Cowboys) were `unavailable` before, and each
residual is a player already diagnosed in item 3 (Singleton W1/W5, Wilson W1, Latu W3, Bell
W4, Overshown W7), all tackle/assist attribution differences between MFL and nflverse. The
largest team residual is 4.00 (W7 Cowboys, Overshown).

Remaining unavailable starters: 85 "not reported this week" and 7 identity gaps (Childress, MFL 0835).

### Bills

| Week | MFL | App | |
|---:|---:|---:|---|
| 1 | 269.90 | 269.40 | differ −0.50 (Singleton −1.0, Wilson +0.5) |
| 2 | 306.95 | 306.95 | exact |
| 3 | 278.35 | 280.85 | differ +2.50 (Latu, one tackle at DE) |
| 4 | 248.90 | 248.90 | exact (McLaurin now a zero) |
| 5 | 328.10 | 327.10 | differ −1.00 (Singleton, one assist) |
| 6 | 233.35 | 233.35 | exact (Warren now a zero) |
| 7 | 246.80 | 246.80 | exact (Ridley now a zero) |

**Points for: app 1913.35 against MFL 1912.35**, +1.00, all from tackle/assist attribution. All
seven Bills weeks now score (0 of 7 in SCHED-9). **The record reads 3–1, not MFL's 6–1**:
standings decide only the 4 Bills games whose opponent has a score; the other 3 opponents
are still pending on starters not reported that week. The Bills' own side is done.

## Replay finding

The first attempt used the harness's `archive.Transport` with the default HTTP transport, so
`SyncScoring` fetched nflverse live and pulled 2026 week-8 stats. Week-8 play-by-play was not
held, so `ProcessStatFeed` failed outright at `internal/fantasyscore/source/plays.go:278`
("missing held play-by-play for 2026_08_CLE_PIT"), and weeks 1–7 could not be produced. One
week's missing source stops the whole board, which goes against R11. Not fixed here (that
path is outside this brief); it is recorded as debt.

## Debt left

- 45 unavailable comparison rows: 85 starter-weeks not reported (55 never on any membership
  snapshot; the missing evidence is nflverse `rosters_weekly_2026`, not held or adapted) and 7
  Childress identity gaps.
- Bills record 3–1 against 6–1 until three opponents' weeks resolve.
- 20 `differ` rows, all MFL-provider and nflverse stat attribution; ours stands. W1 Dike's 0.01 is a
  half-cent rounding convention (away from zero against half up), unchanged.
- R11: missing play-by-play for a new week aborts `ProcessStatFeed` (`plays.go:278`).
- Replay cost: about 46 min, against 169 s in SCHED-9.
