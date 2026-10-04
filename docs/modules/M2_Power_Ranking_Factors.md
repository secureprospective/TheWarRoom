# Power rankings — the factor table (research, 2026-10-04)

**Status:** decided 2026-10-04 and built on `session/power-ranking-factors` (Core Build Plan R10-1…R10-7).
The first pass below used a stand-in for the model's values; the re-run on the model's own Now and
Dyn (the last section) set the shipped numbers and changed two verdicts (#6's curve, #14).

## How each factor was judged
- **Data:** Legacy NFL's own history from MFL (league 14432), regular seasons 2021–2025 (13 weeks each,
  32 teams). For every team and week it holds the roster, who started and every player's score. Also
  standings, end-of-season rosters with salaries, and rookie draft results. That is 160 team-seasons.
- **This season factors:** after k weeks, how well does the factor predict the **rest of the regular
  season** (points per week and all-play win%)? The score is R² on a held-out season: fit on the other
  seasons, score the one left out, for each of 2022–2025. 1.0 is perfect; 0 is no better than guessing
  the average.
- **Franchise factors:** the same, against **next season** and the **mean of the next two**.
- **Roster value in the test** is a stand-in for the model's Now: each player's points per game so far
  this season, shrunk toward last season's. The model's Now is better than this stand-in, so the roster
  side is if anything *understated* here. The lineup solver matches MFL's own "optimal points" on 410 of
  416 team-weeks in 2021 (the misses are mid-season position changes).
- Evidence: `~/fleet/runs/warroom-power-rankings-2026-10-04/` (scripts, raw results, the MFL exports).

## What the board does today
- **This season** = roster value (Now, league points per game) z-scored, blended 60/40 with results.
  **The franchise** = Dyn roster value alone.
- The roster is the **whole roster summed** by default, or the top 21 by value regardless of position.
- **Defect found:** MFL sends all-play as `all_play_wlt: "89-4-0"` and `all_play_pct`. The reader looks
  for `all_play_w` / `all_play_l` / `all_play_t`, which MFL does not send. So all-play always reads 0-0
  and the board falls back to points for, even now that all-play is turned on in MFL.

## The table

R² columns: rest of season after 1 / 3 / 6 weeks (target: all-play win%), unless marked.

| # | Factor | What it measures | Have the data? | Evidence | Verdict |
|---|--------|------------------|----------------|----------|---------|
| **This season — results side** |||||
| 1 | **All-play win%** | Wins if you played all 31 teams every week: strength without schedule luck | Yes, MFL standings (now on) — **but the reader is broken** | 0.56 / 0.69 / **0.76**: the best results signal once 4+ weeks are in | **Fix the reader.** It is meant to be the results side already |
| 2 | Points for | Points per week | Yes | 0.58 / 0.71 / 0.76. Best for predicting rest-of-season *points* (0.77 at 6 weeks) | Keep as the fallback before all-play exists |
| 3 | Potential points | Points from the best possible lineup each week | Yes (`pp`) | Best in weeks 1–2 (0.63 vs 0.58), equal or worse after week 3. Adding it to roster + points for gains +0.02 in week 1, nothing after | Display column only (shown today) |
| 4 | Head-to-head record | Actual W-L | Yes | 0.23 / 0.33 / 0.56: the worst signal by far | Display only (as today) |
| **This season — roster side** |||||
| 5 | **Roster as a legal lineup** | The best 21 the league's lineup rules allow (1 QB, 1–3 RB, 2–5 WR … 2–4 of each defensive spot, 12 defenders) | Yes: rules in the league export, values in the model run | Lineup 0.59 / 0.62 / 0.67; flat top-21 0.59 / 0.61 / 0.65; **whole-roster sum (today's default) 0.57 / 0.61 / 0.64, worst every week**. Next season: lineup beats sum too | **Adopt.** Replace sum and flat top-N. A 4th QB or a 9th WR adds nothing on Sunday |
| 6 | **Roster weight that shrinks as games are played** | How much to trust the roster vs results as the season goes on | Yes (weeks played is in the standings) | The best mix gives the roster 58% after week 1, 42% after week 2, 32% after week 3, 29% after week 4, 18% after week 6. That fits *roster weight = 1.5 ÷ (1.5 + weeks played)*, the same credibility rule the player model uses. Today's fixed 60% is right only in week 1 | **Adopt** (Christopher, 1A). Refit on the real Now: **4 ÷ (4 + weeks)** (last section) |
| 7 | Depth (best 10 bench players) | Cover for byes and injuries | Yes | Adds nothing on top of lineup + results: −0.01 to −0.02 every week | **Reject** as a score input. Bye and injury cover already shows up in points scored |
| 8 | Lineup efficiency (points ÷ potential) | Start/sit skill | Yes (`eff`) | First half → second half r = 0.34–0.51; season to season r = 0.23. Adds nothing on top of points for | Display only |
| 9 | Recent form (last 3 weeks) | Momentum | Yes | Adds nothing (±0.003) | **Reject** |
| 10 | Schedule luck (record − all-play) | Lucky or unlucky schedule so far | Yes | Does not carry forward: first half → second half r = −0.17 to 0.24 | Display only. Useful to read, useless to rank |
| 11 | Remaining strength of schedule | How hard the rest of the slate is | Yes (league schedule) | Predicting the rest-of-season *record*: 0.54 → 0.57 (+0.03). The spread is real (sd 15.7 points a week) | Not strength. Belongs in a **projected record / playoff odds** column if you want one |
| 12 | Availability this week | Starters out (injury, bye) | MFL `injuries` export, not loaded; no past injury data to test | Untested | Park. It belongs in a this-week view, not season strength |
| 13 | Offense / defense split | Where a team's points come from (12 of 21 starters are defense) | Yes (`op`, `dp`) | Not a predictor; context | Display only |
| **The franchise (multi-season)** |||||
| 14 | Dyn value as a legal lineup | Same as #5 on dynasty values | Yes | Stand-in: as #5. **Real Dyn: the whole roster predicts better** (last section) | **Superseded:** the franchise view counts the whole roster by default |
| 15 | Roster age profile | Value-weighted age of the 21 starters | Yes (birthdates) | **Biggest franchise effect found:** next season +0.07 R² (r −0.31 with what lineup + points miss), mean of next two +0.13. **But the stand-in has no age curve, and Dyn does** | **Add** (Christopher, 4B). On the real Dyn it still adds: weight 0.5 (last section) |
| 16 | Cap room | $ under the cap ($125M) | Yes (MFL standings `salary`, the league mirror) | Next season: r +0.17 with what lineup + points miss, +0.01 R²; two seasons out: +0.06, −0.01 R² | Display only. 2026 room ranges $0–$73.5M (median $4.3M) |
| 17 | Expiring contracts | Share of starter value in a final contract year | Yes | r −0.18, but scores worse held out (0.24 → 0.19) | Display only |
| 18 | Draft capital | Rookie picks owned (next 1–2 drafts) | History yes (draft results); current needs new ingestion (MFL `futureDraftPicks`) | No signal: r +0.07 next season, −0.07 two seasons out | Display only, if wanted. Rookie picks barely move a 21-starter, 48-man roster within 2 years |
| 19 | Dead cap | Money owed for released players | Yes (salary adjustments) | Part of cap room (#16) | Fold into the cap room column |

## What this says, in short
1. **Fix all-play first.** It is a bug, and it is the best results signal we have.
2. **Two changes earn their place in the score:** count the roster as the lineup the rules allow (#5 and #14),
   and let results take over from the roster as weeks are played (#6).
3. **Everything else is context, not score:** cap room, luck, schedule, efficiency, picks, the
   offense/defense split. Several are already MFL columns.
4. **One open test:** age for the franchise view, with the real Dyn values, before deciding anything.

## Sources consulted
- MFL league 14432 exports, 2021–2025 (`weeklyResults`, `leagueStandings`, `rosters`, `draftResults`,
  `league`, `players`).
- ffsimulator (ffverse, open source): season simulation by bootstrapping weekly scores, optimal
  lineups, all-play and potential points. The method is the model for a playoff-odds column; its
  rankings input (expert consensus) is not usable here (benchmark-only rule).
- Getty, Li, Yano, Gao, Hosoi, "Luck and the Law: Quantifying Chance in Fantasy Sports and Other
  Contests", SIAM Review 60(4), 2018: skill dominates over a season, single weeks are noisy.
- "Measuring manager performance in fantasy football" (R-bloggers, Dec 2023): all-play against the
  record for schedule luck, and lineup efficiency averaging out across seasons. Both agree with #8 and #10.

## Re-run on the model's own values (2026-10-04, after Christopher's calls)

Christopher's calls: 1A (automatic roster weight, the slider overrides), 2B (lineup by default, keep
the toggle), 3B (cap room with dead cap, schedule luck, projected record), 4B (add age now).

The model was re-run as of each past week: every player on a Legacy NFL roster after k weeks of
2022–2025, valued with the shipped params and that season's MFL positions
(`scratch_power_test.go`, 49,629 player-weeks). Same held-out-season scoring as above. The params
were fitted on data through 2025, so the roster side has a slight head start here; it touches the
roster weight, which comes out a little high if anything.

| Question | Result | Shipped |
|---|---|---|
| Roster count, this season | Rest-of-season R² after 1 / 3 / 6 weeks: legal lineup 0.64 / 0.69 / 0.71; top-21 by value 0.61 / 0.66 / 0.66; whole roster 0.57 / 0.63 / 0.63. Before any game: 0.62 / 0.57 / 0.54 | Legal lineup by default; top-N and the whole roster stay on the toggle |
| Roster weight by weeks played | Best roster share 82%, 70%, 58%, 46%, 35% after weeks 1, 2, 3, 4, 6. Of m ÷ (m + weeks), m = 3 to 6 fit best (mean r 0.866–0.867, against 0.865 for today's fixed 60/40 and 0.862 for m = 1.5) | m = 4: 80% after week 1, 50% after week 4, 29% after week 10 |
| Roster count, franchise | Next season / mean of next two: whole roster 0.29 / 0.12; top-21 0.27 / 0.12; legal lineup 0.27 / 0.10. A dynasty roster's bench grows into starters | The whole roster by default; the toggle still offers the others |
| Age on top of Dyn | Whole-roster Dyn plus its value-weighted age: 0.29 → 0.32 next season, 0.12 → 0.15 over two. Fitted weight against Dyn (both z): 0.42 next season, 0.55 over two | Age at 0.5 of Dyn's weight, younger better; the age of the players the count includes |
| Projected record | A game's win chance Φ((a − b) ÷ σ), each team's expected score = the auto weight on its lineup's Now + the rest on its points per game. Best σ 50 points (Brier 0.165 against 0.25 for a coin flip, 4,480 games); projected remaining wins miss by 1.2 on average | σ = 50 |

Evidence: `~/fleet/runs/warroom-power-rankings-2026-10-04/` (`power2.py`, `power3.py`, `results3.txt`,
`results4.txt`, `values.csv`).
