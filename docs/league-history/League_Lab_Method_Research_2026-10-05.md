# Legacy NFL Lab: how to rebuild the league's history and pull out what you can act on

Research pass, 2026-10-05. It replaces the earlier psychology research as the guide for the app.

**The question:**
- How do you rebuild a 14-season league history so it is trustworthy?
- How do you get the full picture out of it?
- How do you surface only the outliers that change a decision in the game?

---

## 1. What went wrong in rounds 1–3 (an honest diagnosis)

Tamara Munzner's model of visualization design (UBC, 2009) has four layers:

1. The real problem.
2. The data and task abstraction.
3. The picture.
4. The code.

A mistake at one layer carries into every layer below it. Rounds 1–3 were mostly work at layers 3 and 4 (pictures and code) on top of an unexamined layer 2. Our tests checked that charts render. They never checked that a chart answers a question you would act on.

| Symptom on screen | Underlying cause |
|---|---|
| Totals like "Fading took in 66.7 firsts". | Totals summed over 14 seasons. They are not per team or per season, so the size of the number means nothing. |
| "Price of a proven player" is a mean of 23 players in a two-week window. | No uncertainty is shown and the samples are tiny. Most of the differences are noise. |
| "Who drafts best" ranks 32 teams by hit rate on 15–70 picks. | A ranking of noise. Ranking small samples in a league table is a known mistake (Spiegelhalter, funnel plots). |
| 92% of 1st-round picks "hit". | The hit bar ("one weekly-starter season") is too low for a 32-team league with deep starting lineups, so it cannot separate good picks from bad. |
| "How picks pan out, season by season" falls off after season 5. | Composition bias: only the oldest classes reach season 6 or later. |
| Team stages come from raw standings rank. | Rank carries about ±3 wins of schedule luck (Elhabr, 2023). Stage labels inherit that noise. |
| Trade heat grid, era lines, era-share heat, flow heat, stacked bars. | Pictures of activity. None of them answer "what should I do". |
| The turn scatter is 43 unnamed dots. | No link to a team, a player or the present. |
| Pick value scale 1 / 0.66 / 0.44 … | Counts starter seasons and ignores how good the starter was or what he cost. |
| Price, draft and trade use three different measures. | **No common currency**, so you cannot compare a pick, a player and a contract. |
| Everything describes the past. | **No link to today**: no current state, no who-is-likely-to-do-what-next. |

The root cause is the last two rows. There is no single value scale, and no bridge from history to the present. Without those, every chart can only describe the past.

---

## 2. What the archive actually holds (verified 2026-10-05)

`league-archive/raw/<year>/`, 2013–2026, 97 MB. Files marked **unused** are not read by the current compiler.

| Source | What it gives | Used now? |
|---|---|---|
| `transactions.json` | 2,634 trades, 7,393 free-agent adds, 4,195 waiver moves (2013–16), 916 IR moves, 625 taxi moves, **470 trade proposals, 156 rejections, 156 revokes, 150 accepts** (2016–26, often with the proposer's own reasoning in the comments) | Trades only |
| `weeklyResults_WYTD.json` | Every matchup score. **Each team's starters, its bench and MFL's "optimal" lineup**, for 13 full seasons. | Scores only |
| `weekly/rosters_Wnn.json` | Weekly roster snapshots | Yes |
| `weekly/playerScores_Wnn.json` | Weekly player points | Yes |
| `weekly/projectedScores_Wnn.json` | Weekly projections | **Unused** |
| `salaries.json` | Contract per player per season: salary, years, status, extension text | **Unused** (only through `salaryAdjustments`) |
| `salaryAdjustments.json` | Dead money (re-lists old cuts; deduped) | Yes |
| `futureDraftPicks.json` | Who owns every future pick, as of each season's snapshot | **Unused** |
| `draftResults.json` | Rookie drafts | Yes |
| `tradeBait.json` | What each team says it will give up and wants | **Unused** |
| `assets.json` | Each franchise's players and picks | **Unused** |
| `rules.json` / `league.json` | Scoring, lineup requirements, cap | Partly |
| `auctionResults.json`, `accounting.json` | Empty: the league never ran an MFL auction or accounting | — |
| Owner names | Not in the archive. Franchise slots are the only identity available. | — |

**Missing for later work:** free-agent adds carry no bid amount, so FA spending cannot be priced. Owner changes cannot be detected.

---

## 3. The method, layer by layer

### Layer 1 — rebuild the league as a dated ledger ("as of that day")

Finance has a hard-won rule called **point-in-time data**: any past date must be reproducible exactly as it was known that day. Using later information when judging an earlier decision is **look-ahead bias**. In backtests it inflates results by several points. The fix is to keep two dates on every fact:

- when it happened;
- when it was knowable.

Every query then asks for the picture "as of" a date.

Applied here:
- **Asset ledger.** Every player, pick and contract gets a dated ownership history. It is built by replaying every transaction (trades, adds, drops, waivers, IR, taxi, draft) in time order.
- **Franchise state for every day.**
  - Roster, cap room and dead money.
  - Pick inventory, by year and round.
  - Record and all-play record (Layer 3).
  - Roster age and the value of the roster (Layer 2).
- **Two kinds of fields, kept apart:**
  - *At the time*: what was knowable that day (record, last season's points, age, contract).
  - *Hindsight*: what the asset later produced.

  The app labels which one a number is. Hindsight is never mixed into a judgment of the decision.
- **Proof of completeness.**
  - The replayed rosters must match the weekly roster snapshots, week by week.
  - The replayed pick ownership must match `futureDraftPicks`.
  - The app reports the match rate, and every gap is listed. This is the standard event-sourcing check (rebuild from events, check against snapshots). It is what makes the dataset trustworthy rather than merely large.

### Layer 2 — one currency: wins

The common sports-analytics answer to "a pick, a player and a contract are not comparable" is **value above replacement**:
- Woolner's VORP in baseball.
- Fantasy WAR (Fantasy Points; Dynasty Nerds).

**How it works:**
- **Replacement level.** Take this league's own lineup rules (IDP, 32 teams, the starter counts in `rules.json`). The replacement player at each position is the best one a team could expect to find for free that season. The 32 teams fill the lineup slots, and the replacement level is the next tier down.
- **Points to wins.**
  - A player's points above replacement, week by week, are run through the spread of weekly team scores in this league. The result is the extra matchups he would win.
  - That gives **wins above replacement (WAR)** for every player-season, in this league's own scoring.
- **Age.**
  - Expected future wins come from an aging curve built with the delta method: the same player, season to season.
  - It is corrected for survivor bias. Players who had a lucky year stay in the data and then "decline", which flatters the curve (Lichtman; Baseball Prospectus).
- **Contracts.**
  - Cap dollars are converted to wins at the league's going rate (the cap spent per win by contending teams).
  - A player's **surplus** = expected wins − contract cost. This is the Massey–Thaler surplus idea and Over The Cap's "value over APY".
- **Picks.**
  - A pick's value = the expected surplus at that draft slot, taken from this league's own draft history. That is wins on a cheap rookie contract.
  - Massey and Thaler found that the NFL overpays for top picks by about 2× relative to surplus. Whether this league does the same is an open question to test.
- **Market price.**
  - What the league actually pays for an asset is learned from its own trades. Each trade says that, in the two GMs' eyes, side A was worth about side B on that day.
  - Fitting a regularized regression across all 2,634 trades yields the league's implied price for an asset type (position × age × production tier × contract × pick round × time of year).
  - This is the "revealed preference" approach behind FantasyCalc (values from millions of real trades), and the hedonic-pricing approach used for footballer transfer fees.

With one currency, every decision reduces to a single comparison: **what the league charges versus what the asset returns.** The gap between the two is the edge.

### Layer 3 — separate skill from luck before calling anything a habit

- **Team state from all-play, not rank.**
  - All-play = your record if you had played every team every week.
  - Actual minus all-play is schedule luck, about ±3 wins a season (Elhabr 2023, Harstad).
  - Contending/rebuilding is judged from all-play, roster WAR, roster age, pick inventory and cap room. Rank alone is not used.
- **Lineup skill.** Points scored against MFL's optimal lineup, for 13 seasons. This is a pure manager-decision measure that the current app ignores.
- **Shrink every per-team rate toward the league before showing it.** This is empirical Bayes, the beta-binomial method (Efron & Morris 1975). It cut prediction error by about 65% against raw averages. A team with 15 picks and 73% hits is not a 73% drafter. Its honest estimate sits much nearer the league average.
- **Funnel plots instead of league tables.**
  - Plot each team's rate against how many decisions it rests on, with control limits that narrow as the sample grows (Spiegelhalter).
  - Only teams outside the funnel are real outliers.
- **Persistence test (reliability).**
  - A trait is only a habit if it repeats. Split each team's history into halves (odd and even seasons) and correlate them.
  - Reliability = true variance ÷ observed variance (Tango).
  - A trait with near-zero split-half correlation is noise and is not shown as a habit, however striking a single number looks.

### Layer 4 — which outliers matter

An outlier is shown only if it passes all four tests:

1. **Measured against an expectation.** Each outlier is a gap from a baseline: the same slot, the same state, the same time of year. A raw extreme is not enough. (Keim's visual analytics rule: "analyse first, show the important", then zoom, filter, details on demand.)
2. **Survives the noise.** It sits outside the funnel and its shrunken estimate still clears a meaningful size.
3. **Persists.** It repeats across time (Layer 3), so it predicts what happens next.
4. **Maps to a decision.** It changes one of the following:
   - who to call;
   - what to offer;
   - when to offer;
   - what to buy or sell;
   - what to stop doing yourself.

   This is the "positive deviance" idea: find the teams that do better with the same constraints, and copy the behavior.

**Guarding against false findings.** Every free filter is another comparison. Studies of interactive exploration find that up to 60% of "insights" are false (Zgraggen et al., in Northwestern's "garden of forking paths in visualization" work). Uncertainty displays reduce false discoveries. The app therefore:

- leads with a fixed set of questions that were computed in advance;
- shows an interval on every estimate;
- labels any number resting on few cases;
- treats free filtering as a way to drill into one result, not a way to go fishing.

### Layer 5 — counterparty profiles (the "who and when")

Research on MLB and NBA trade networks shows that front-office relationships and repeat pairings shape who trades with whom, though the effect is modest. For each franchise, the profile is:

- **Now:** state (Layer 3), cap room, picks owned, roster age, and the players it has listed as trade bait.
- **Persistent habits** (only those passing the reliability test):
  - when in the year it trades;
  - whether it pays or charges above the market price (Layer 2);
  - what it sells as its state changes;
  - its usual partners;
  - its proposal-to-acceptance pattern (from 470 proposals and 156 rejections).
- **What it is likely to do next.** Use the outside view (Kahneman's reference-class forecasting):
  1. Find past team-seasons in the same state, at the same point in the league year.
  2. Report what they did next.
  3. Weight that by this franchise's own persistent tendency.

  Base rates first, the particular team second.

### Layer 6 — judge decisions by process, then by outcome

Every trade gets two scores:

- **At the time:** market value received minus market value paid, on that day. This measures negotiation.
- **Hindsight:** wins actually produced minus wins given up. This measures negotiation plus luck.

Process and outcome are kept separate. Good decisions sometimes turn out badly (Duke; Mauboussin). A team that wins trades at the time, again and again, is a skilled negotiator. A team that only wins in hindsight was partly lucky.

---

## 4. What this means for the app

The app answers a short list of decision questions, in this order:

1. **Where is every team right now, and who is likely to buy or sell, what, in the coming weeks?** This uses the league clock: "we are at week 5; here is what teams in each state usually do between now and the deadline."
2. **What does the league charge for each kind of asset at this time of year, and what does it actually return?** A ranked list of what is underpriced and what is overpriced.
3. **For any team I'm about to deal with:** its state, needs, cap, picks and trade bait, plus its proven habits and its record at the time on trades. A dossier you open before you call.
4. **Draft:** what each slot is worth, what the league pays for it in trades, where the two disagree, and drafting skill shown with honest uncertainty.
5. **My own franchise:** my habits against the league's, judged at the time, and where they cost wins.

Everything else serves those five questions as details on demand. That includes the ledger, every record behind every number, and the reconciliation report. Anything that serves none of them is cut.

---

## Sources

- Munzner, *A Nested Model for Visualization Design and Validation* (2009): https://www.cs.ubc.ca/labs/imager/tr/2009/NestedModel/
- Brehmer & Munzner, *A Multi-Level Typology of Abstract Visualization Tasks* (2013): https://www.cs.ubc.ca/labs/imager/tr/2013/MultiLevelTaskTypology/
- Keim et al., visual analytics mantra (2006), as summarized in: https://arxiv.org/pdf/1404.4550
- Shneiderman, *The Eyes Have It* (1996): https://webspace.science.uu.nl/~telea001/uploads/VACourse/Shneiderman96.pdf
- Point-in-time data and look-ahead bias: https://arkolith.com/blog/point-in-time-data-explained · https://dev.to/tradevodata/your-backtest-saw-the-future-lookahead-bias-in-fundamental-data-measured-across-313562-rows-4cba
- Fantasy WAR: https://www.fantasypoints.com/nfl/articles/season/2021/fantasy-war-part-1-theory · https://www.dynastynerds.com/analytics/fantasy-football-wins-above-replacement-the-theory/
- Replacement level: https://fantasy.fangraphs.com/value-above-replacement-part-one · https://tht.fangraphs.com/replacement-level-theory-applied/
- Massey & Thaler surplus value: https://www.pff.com/news/nfl-revisiting-the-losers-curse-the-surplus-value-of-draft-picks · https://arxiv.org/html/2411.10400v3
- Over The Cap valuation: https://overthecap.com/?p=16661
- Aging curves and survivor bias: https://www.baseballprospectus.com/?p=59491 · https://hockey-graphs.com/2017/04/10/a-new-look-at-aging-curves-for-nhl-skaters-part-2/
- Revealed-preference trade values (FantasyCalc): https://www.parse.gl/brands/fantasycalc-com
- Hedonic pricing of players: https://lida.sport-iat.de/dfb/Record/4060897?lng=en
- Empirical Bayes / Efron–Morris: https://metricgate.com/docs/hierarchical-beta-binomial-baseball/ · https://andrewpwheeler.com/2018/07/23/sorting-rates-using-empirical-bayes/
- Funnel plots: https://www.stats.bris.ac.uk/R/web/packages/FunnelPlotR/vignettes/funnel_plots.html · https://arxiv.org/pdf/1810.12664
- Reliability and regression to the mean: https://blogs.fangraphs.com/a-new-way-to-look-at-sample-size · https://tht.fangraphs.com/it-makes-sense-to-me-i-must-regress
- Luck against skill in fantasy (all-play, ±3 wins): https://tonyelhabr.rbind.io/posts/fantasy-football-performance/ · https://www.footballguys.com/article/HarstadFiT7
- Forking paths in visual analytics: https://mucollective.northwestern.edu/project/forking-paths · https://mucollective.northwestern.edu/?p=527
- Positive deviance: https://ssir.org/book_reviews/entry/power_positive_deviance_richard_pascale_jerry_sternin_monique_sternin
- Reference-class forecasting: https://www.mckinsey.com.br/capabilities/strategy-and-corporate-finance/our-insights/daniel-kahneman-beware-the-inside-view
- Trade networks: https://resolve-he.cambridge.org/core/journals/network-science/article/do-nba-teams-avoid-trading-within-their-own-division/442B1F6003E0905C08F553641FAE0F13 · https://ideas.repec.org/a/sae/jospec/v15y2014i6p601-616.html
- Process against outcome: https://www.michaelmauboussin.com/s/MTYKexcerpt.pdf

---

## 5. What the foundation found (built and checked 2026-10-05)

Every number below is in the app, with its error range, on the "How it's measured" screen.

**The data and the currency**
- **Ledger.** Week-to-week team changes are 100% explained by recorded moves, every season from
  2018 on. Before 2018, MFL's stored weekly rosters barely change within a season, so those years
  can't be tested.
- **Currency.** Wins above replacement from started players track all-play win rate at 0.97.
- **Forecast.** Measured on players it never saw, it explains this share of next season's
  variation:
  - wide receivers 52%; quarterbacks 47%; running backs 43%;
  - linebackers 36%; tight ends 33%;
  - defensive linemen 28%; defensive backs 24%; kickers 20%.

**How this league trades**
- **Implied discount:** next season counts about 1.33× the season after.
- **Contracts:** barely priced, at 0.007 wins per $1M a season.
- **Bigger packages:** each extra asset in a deal is worth about 0.08 wins less than its
  expected wins say.

**Backtest: would it have worked?** 699 trades, 2018–2024, each valued using only earlier
seasons:
- The side favoured by expected wins came out ahead 52% of the time.
- Lopsided-looking trades delivered about 10% of the gap they showed.
- **Conclusion:** the league's consensus carries information the box score lacks. Edges exist
  across kinds of asset, not inside single deals.

**What a delivered win costs**, relative to a next-draft 1st (1.0), for trades from 2017 to
2026. Both halves of the history agree unless noted.
- **Linebackers:** about 0.18 at every age. The league charges about 0.28 per expected win, and
  they delivered about 1.55× their forecast.
- **Prime-age defensive backs:** 0.26.
- **Older tight ends:** 0.36.
- **Defensive linemen:** 0.28–0.39.
- **Next-draft 1sts:** 1.27, the dearest. They delivered 0.79× their forecast.
- **Young QBs, WRs and RBs:** delivered only 46–71% of their forecast.

**The draft (round 4 of the build)**
- Within any single round, position choice is within chance.
- Pooled across rounds:
  - Defensive linemen return about 70% of their slot and tight ends about 80% (beyond chance).
  - Kickers taken late beat their slot.
- **Paired with the market:**
  - Young tight ends (0.73 per expected win) and defensive backs (0.60) are what the league pays
    most for once drafted.
  - Linebackers are the cheapest wins to buy at every age.
  - So: draft what the market overpays for, and buy linebackers in trades.

**Cardinals**
- The one franchise outside the funnel: −0.08 expected wins per trade at the time, over 140
  trades. Hindsight is +0.06 per trade over 45 trades with two or more seasons played.
- Draft: −0.04 wins against the slot per pick over 56 judged picks, which is within chance.
