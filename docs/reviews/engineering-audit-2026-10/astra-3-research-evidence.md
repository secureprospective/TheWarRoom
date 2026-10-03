# Sector 3: evidence for credibility, recency, priors and aging
Research date 2026-10-03. IN PROGRESS. Primary-source excerpts below are from pages fetched today; numeric HTTP statuses are reported only when the extension supplied them. Public article previews are not complete access to proprietary datasets. Blog analyses are labelled research leads, not validated calibration constants.

## E0. What would actually justify k
The brief's `Z=n/(n+k)` has Z=0.5 when n=k (algebra, not an empirical football finding). Under a stationary random-effects model with between-player variance tau² and per-opportunity noise sigma²/n, reliability is tau²/(tau²+sigma²/n), giving k=sigma²/tau². This is a conditional mathematical interpretation, NOT permission to turn any published correlation into k.

Three essential distinctions:
1. A study's minimum sample inclusion cutoff is not its estimated 50% reliability crossing. In particular, **250 pass-rush snaps in E2 is an inclusion filter, not k=250**.
2. Year-to-year correlation incorporates changing role, team, age and selection. It is neither the same as split-half reliability nor its square. Random within-season splits can share environment and overstate predictive reliability under a future role change.
3. The reliability of a raw production statistic versus a population mean does not automatically supply the optimal weight versus a player-specific scouting prior. Draft/Madden/college priors can be correlated with production. Both signals must estimate the same target on the same scale; an ordinal prospect rank and yards per attempt cannot be averaged directly.

**No publishable universal NFL position-specific k vector was established in this run.** The starting values in Sector 4 must remain labelled placeholders, not fitted or literature-validated estimates. Injury absences add no new trials; earlier seasons contribute discounted evidence, not zero-valued play outcomes.

## E1. QB: an apparent stabilization number with a methodological warning
**NFLGraphs (author handle; personal author identity not established), 2019-07-21, blog, “How many passes does it take for Passer Rating to ‘stabilize’?”**
URL fetched/readable: https://nflgraphs.wordpress.com/2019/07/21/how-many-passes-does-it-take-for-passer-rating-to-stabilize/

Population: **“10 seasons spanning 2009 to 2018”**; **“only QBs with 300 or more passing attempts”**; computes a passer-rating value for each throw and Cronbach's alpha as attempts accumulate.
Exact finding: **“In 2018, the threshold was crossed after only 50 pass attempts, while in 2014 it took until 110 pass attempts for alpha to reach 0.7. For the majority of the seasons in this study, 80 pass attempts was sufficient”**.

Do NOT set QB k to that number. The article says **“a correlation of .70 means an R-squared of 49%”** while interpreting alpha as reliability; those quantities are not interchangeable. It also uses a nonlinear/clipped passer-rating formula at the individual-play level, not simply aggregate NFL passer rating. This is a useful cautionary lead, not robust evidence for the requested reliability=0.5 point. No transferable completion%, CPOE, Y/A or interception-rate k established here.

## E2. Pass rush: strong relative reliability evidence, not a stabilization threshold
**Eric Eager, PFF, 2020-02-13, “True pass-rushing snaps and their importance to player evaluation.”** Original proprietary-data analysis; publicly returned article preview, not peer-reviewed paper.
URL fetched readable and raw (author/date metadata verified): https://www.pff.com/news/nfl-true-pass-rushing-snaps-importance-player-evaluation

Quoted table:
> Sack Rate / Pressure Rate / Grade/Play / Win Rate: **0.51 / 0.72 / 0.72 / 0.73**.
> “Snap threshold of **250 pass-rushing snaps per season**, yielding **n = 1282 season-pairs**.”

The restricted-period comparison says **“only look at data from 2012 to the present”** and gives **0.48 / 0.70 / 0.73 / 0.73**, **“Minimum 250 pass-rushing snaps, n = 711 players.”** Full baseline start year not stated in retrieved preview; do not invent it. “Present” is publication in 2020, not 2026.

Implication: pressure rate is more repeatable than sack rate among qualifying pass rushers. Applies directionally to DT/DE/edge roles, not a calibrated DT-vs-DE split and not automatically off-ball LB. PFF explains scheme/double-team and opponent/quick-throw confounds. An unblocked pressure and a positively graded rush are not identical. Free PBP sacks/QB hits do not recreate PFF pressures/wins or pass-rush opportunities.

## E3. WR opportunity rate versus yardage efficiency
**Chase Stuart, Football Perspective, 2014-07-21, original analysis/blog, “Yards per Route Run, Yards per Target, and Targets per Route Run.”**
URL fetched readable and raw (byline/date verified): https://www.footballperspective.com/yards-per-route-run-yards-per-target-and-targets-per-route-run/

Population quotation: **“From 2007 to 2012, there were 344 wide receivers who saw at least 40 targets in Year N, and then played for the same team and saw at least 40 targets in Year N+1.”** This is a selected same-team consecutive-season sample; the article acknowledges survivorship concerns.
Exact fitted formulas:
> `N+1 YPRR = 0.843 + 0.474 * Yr N YPRR (R^2 = 0.21)`
> `N+1 Yd/Tar = 5.84 + 0.28 * Yr N Yd/Tar (R^2 = 0.08)`
> `N+1 TPRR = 0.062 + 0.671 * TPRR (R^2 = 0.41)`

Author's conclusion: **“Targets Per Route Run ... the most consistent from year to year.”** These are regression slopes and R², not k or split-half correlations; 40 targets is selection, not stabilization. Evidence supports separating earning targets from efficiency after the target. WR findings do not establish identical TE/RB coefficients. No free complete routes feed verified in Sector 2.

## E4. Coverage: important football skill can be statistically unstable
**Eric Eager and George Chahrouri, PFF, 2018-06-22, original industry analysis, “Examining the value of receiver and coverage positions in today's NFL.”**
URL fetched readable/raw, author/date metadata verified: https://www.pff.com/news/pro-pff-forecast-examines-value-of-coverage-and-receiving

The study says **“all quarterbacks seasons since 2006”** and player-level analyses require **“30 or more targets at a given position in both seasons.”** Exact coverage sample count and end-season enumeration not present in retrieved text.
Key quoted findings:
> “For deep safeties and box safeties/linebackers, EPA allowed at the team level was correlated at a rate roughly 0.13, while at the player level there was no correlation (**-0.015 and 0.03**, respectively).”
> “PFF coverage grades for box safeties and linebackers were slightly more stable (**0.159 team level, 0.208 player level**), but the same wasn’t true for safeties (**0.077, 0.056**).”

For outside/slot CBs the article describes individual EPA correlations as small; no unextracted chart values are invented. This is not proof coverage is unimportant, nor proof no one has coverage skill. It is evidence against high-confidence ranking from one season of allowed outcomes. PFR passer-rating-allowed is not PFF coverage grade or EPA, so no numeric equivalence is licensed. Team versus individual and target alignment versus roster position must stay distinct. No CB/S stabilization k obtained.

## E5. Tackling: primary IDP split-period evidence, with the denominator trap exposed
**Quang Nguyen, Ruitong Jiang, Meg Ellingwood and Ronald Yurko, Scientific Reports, 2025-01-16, “Fractional tackles: leveraging player tracking data for within-play tackling evaluation in American football.” Peer-reviewed original study.**
Fetched readable full text: https://pmc.ncbi.nlm.nih.gov/articles/PMC11739690/ and https://www.nature.com/articles/s41598-025-85993-1 . A subsequent raw PMC fetch hit a browser challenge; the earlier readable full text and publisher page were available, without login.

Population: tracking supplied for **“12,486 plays across 136 games during the first nine weeks of the 2022 NFL season”**. Analysis is a **subset: running-back run plays**, not all NFL tackles. Stability section: **“first four weeks and last five weeks.”** Table 3 compares **totals**, not tackle rates per snap:

|Metric, quoted Table 3|All positions|DB|DL|LB|
|---|---|---|---|---|
|Total fractional tackles|0.69 (0.65, 0.73)|0.57 (0.47, 0.65)|0.57 (0.48, 0.66)|0.73 (0.66, 0.80)|
|“Total tackles + assists/2”|0.59 (0.54, 0.65)|0.46 (0.34, 0.57)|0.51 (0.40, 0.61)|0.64 (0.54, 0.73)|

Parentheses are reported 95% CIs. The study's combined comparator is **tackles + assists/2**, NOT Legacy NFL scoring and not necessarily a vendor's combined tackles field. DB pools CB/S; DL pools DT/DE. Exposure and role contribute to correlations in totals; these values do not locate a 0.5 **rate** reliability crossing or imply “four games = k”. Tracking-based slowing/contact credit can measure contributions omitted by box-score tackles. No current free full tracking feed was verified in Sector 2, so this is conceptual evidence, not an immediately ingestible metric.

**Supporting industry context (not a k study):** Jonathon Macri, PFF, 2024, “Using 2024 NFL defensive schemes for IDP safety projections,” https://www.pff.com/news/fantasy-football-using-2024-nfl-defensive-schemes-for-idp-safety-projections (article retrieved). Its alignment-specific discussion supports distinguishing box/slot/deep safety deployment. No numeric coefficient from this article is used here without needing its full denominator protocol.

## E6. Kicker shrinkage has direct NFL evidence, but not the proposed k formula
**Jason A. Osborne and Richard A. Levine, Journal of Sports Analytics 3(2), 129–146, 2017, “Shrinkage estimation of NFL field goal success probabilities.”**
Publisher abstract URL https://journals.sagepub.com/doi/abs/10.3233/JSA-16140 returned HTTP 403. Author-proof PDF https://pdfs.semanticscholar.org/5813/adf733479f8535dfe3471d92228c06804c61.pdf fetched and extracted; page-local web `answer` extraction supplied formulas/quotes below. Page numbers refer to that proof (1–18), not final journal pagination.

Abstract p.1: **“demonstrate the desired variance-reduction, both in and out of sample”**, with **“ranking NFL kickers from 1998 to 2014”**. Data include playoffs, not XP; exclusions include all-make/all-miss careers. Model is distance-conditioned complementary-log-log kicker ability plus league empirical frequency, not raw FG%.

The displayed midpoint formula (p.7) is `p_mid(d)=0.5*p_kicker(d)+0.5*p_league(d)`. The exponential alternative (p.8) weights individual estimate by `1-exp(-n_k/a)`; reported full-data optimum **a=670**. Neither is `n/(n+k)`, and **670 must not become k**. These quoted formulas/numbers are the paper's, not proposed application settings.

Table 6 (pp.9–10) reports cross-validation average Hosmer–Lemeshow statistics: CLL **20.12**, empirical frequency **16.70**, midpoint **7.06**, exponential(a=670) **7.75**, Bayesian CLL **17.35**. Those are calibration statistics, NOT RMSE or reliability. Quote: **“the midpoint and exponential shrinkage estimators are generalizing to out-of-sample data better than either the CLL or EF components by themselves.”** Crucial limitation: exponential parameter selection was not repeated inside cross-validation. The distinct empirical-Bayes Bayesian CLL model is not the midpoint estimator and did not dominate this comparison. Stadium effects matter; distance/environment adjustment precedes declaring kicker skill.

Direction supported: shrink noisy kicker rates and control for kick distance/context. Exact new-season k, XP prior, and kicking age slope: not established. New-era long-range attempts and PAT changes also limit transport from this historical sample.

## E7. Credibility and empirical Bayes: what transfers, what does not
**Lawrence D. Brown, Annals of Applied Statistics 2(1), 113–152, 2008, “In-season prediction of batting averages: A field test of empirical Bayes and Bayes methodologies.”**
https://arxiv.org/abs/0803.3697 fetched readable/raw; author metadata verified. Primary paper abstract studies **“a single season (2005)”**, earlier-season batting used to predict remaining season. Quote: **“In all situations the poorest performing choice is the naïve predictor which directly uses the current average to predict the future average.”** It also says the nonparametric empirical-Bayes procedure did less well on homogeneous subsets, where familiar alternatives improved.

Transfer: pooling noisy observations can improve genuinely future predictions; population choice changes which estimator wins. Non-transfer: baseball at-bats are not NFL snaps and this abstract supplies no football k. See E6 for direct NFL evidence that shrinkage method choice can change outcomes.

**Bühlmann specifically:** no primary sports application establishing this exact engine's n/(n+k) with player-specific scouting priors was verified. Search found general credibility references; https://encyclopediaofmath.org/wiki/Credibility_theory fetched HTTP 502. Do not mislabel Brown or the kicker midpoint procedure as a verified Bühlmann implementation. The conditional variance derivation in E0 is transparent algebra, not a citation claim.

**Weighted-history caution (derivation):** for independent equally noisy observations with weights w, variance of a weighted mean is proportional to `sum(w²)/(sum(w))²`; its sampling effective count is `(sum(w))²/sum(w²)`, not simply `sum(w)`. Recency-discounted snaps in the brief may instead intentionally represent aging/forgetting. That is a model choice requiring temporal validation, not automatic classical reliability. Rescaling all weights leaves the mean unchanged but changes Z unless k is rescaled too.

## E8. Recency: a defensible benchmark, no validated football-wide constants
### E8a. Original Marcel specification (baseball, not NFL evidence)
**Tom Tango / TangoTiger, “The 2004 Marcels,” Tango on Baseball, 2004-03-10; original methodological blog/forum post.** https://tangotiger.net/archives/stud0346.shtml fetched. Readable extraction was incomplete (547 chars); raw fetch recovered the actual article and discussion.
Original passage: **“Weight each season as 5/4/3. 2003 counts as ‘5’ and 2001 counts as ‘3’.”** Regression component: **“1200 PA for each player (2 weights x 600 PA).”** This describes batting data 2001–2003 projecting 2004, not NFL season weights. Later pitcher comment uses different weights; even Marcel is not one universal sport-free recipe.

Candidate relative weights `[1,0.8,0.6]` are just 5/4/3 divided by five (our arithmetic). They are a transparent comparison baseline, **not evidence-validated NFL defaults**. Use exposure-weighted rates, not equal weighting of a tiny injured season and a full season. MLB pseudo-PA and age corrections cannot be transplanted into football.

### E8b. Football evidence for recency/development, without exact weights
**Timo Riske, PFF, 2020-08-26, “How important is rookie performance when it comes to predicting career performance?”** https://www.pff.com/news/nfl-how-important-is-rookie-performance-when-it-comes-to-predicting-career-performance fetched readable/raw (byline/date verified). Public preview describes drafted cohorts **“between 2005 and 2015”**, comparing first-year-to-second-year WAR against later early-career transitions, with and without draft status. Quote: **“when forecasting a player going into Year 3, we should weigh the second year higher, especially when there is a large discrepancy between Year 1 and Year 2 performance.”** This is not an exact coefficient or a proven point at which the scouting prior stops mattering. Position-specific results are beyond the returned preview.

**Ruben Chung, Brown University undergraduate independent research, “Forecasting NFL Wide Receiver Touchdowns with a Temporal Linear Regression Model,” undated document hosted under a December 2025 path.** https://wsb.wharton.upenn.edu/wp-content/uploads/2025/12/CHUNG_Penn-Forecasting-NFL-Wide-Receiver-Touchdowns-with-a-Temporal-Linear-Regression-Model1.pdf fetched/extracted through page-local answer mode. Paper pp.7/10 describes WR seasons **1990–2024**, training **1990–2010**, test **2011–2024**. It uses previous-year features and two-year rolling means, but **no exact season recency-weight vector was recoverable**. p.18: **“No injury or situational awareness”** and **“Limited positional scope.”** Not peer-review verified; no reported performance score is used as validation for our model.

### E8c. Numeric-looking NFL weights that fail the adoption gate
**Nick (surname not established), The Box Score, 2026-05-12, original product blog, “Stop averaging the last three years for every position.”** https://www.the-box-score.com/nfl/articles/recency-weights-learned retrieved.
Quoted table, last year through fourth prior year:
- RB **68%, 20%, 10%, 3%**
- WR **66%, 27%, 7%, 1%**
- TE **64%, 21%, 8%, 8%**
- QB **42%, 24%, 22%, 12%**
- DB **40%, 24%, 20%, 17%**

The text says **“16 years of data”** but supplies neither exact seasons, player inclusion rules, outcome definition nor reproducible holdout protocol on the fetched page. Rounding makes several rows sum to more than unity. **Lead only; do not adopt.** “DB film holds up” is not established by a fantasy projection claim. The difference versus Marcel cannot be resolved empirically from this disclosure; they are different sports/targets/methods, not two peer-reviewed estimates of the same parameter.

## E9. Scouting priors: strength, independence and useful lifetime
### E9a. Draft capital: real broad signal, not individual certainty
**Bryce Hadley, Jun Woo Kim, Marshall Magnusen and Kyoung Tae Kim, Frontiers in Sports and Active Living, 2025-09-10, “Redefining the draft pick valuation in the National Football League.” Peer-reviewed research report.**
https://www.frontiersin.org/journals/sports-and-active-living/articles/10.3389/fspor.2025.1628223/full fetched readable/raw, authors verified in citation metadata.
Population quote: **“4,996 distinct draft picks spanning from 1993 to 2012.”** The page later says 1994–2012 while calling it a twenty-year period: internal date inconsistency, not silently corrected here. It excludes observations missing outcomes. Aggregated-pick models report **“R2 of 0.90 for weighted approximate value (wAV), 0.82 for games played (GP), and 0.90 for season started (ST)”** after averaging by draft slot. In contrast, player-level models report **“approximately 0.31 ... 0.22 ... 0.31”**.

Use: draft capital is a defensible position-conditioned starting prior for opportunity/career value. Do NOT sell aggregated-slot R² as prediction accuracy for an individual rookie, transfer NFL trade chart prices into fantasy value, or extrapolate pooled results as separate DT/DE/LB/CB/S calibrations. This is not an estimate of increment after observed NFL production. E8b tests that question conceptually but accessible results do not establish a numeric expiration schedule.

### E9b. Athletic testing: conflicting studies, not a licence for a universal RAS multiplier
**Frank E. Kuzmits and Arthur J. Adams, Journal of Strength and Conditioning Research 22(6), 1721–1727, November 2008, “The NFL Combine: Does It Predict Performance in the National Football League?”**
https://doi.org/10.1519/JSC.0b013e318185f09d fetched publisher abstract successfully after PubMed extraction failed.
Population: QB/RB/WR drafted **“1999-2004.”** Outcomes include draft order, salary/games in subsequent years and position statistics. Quote: **“no consistent statistical relationship between combine tests and professional football performance, with the notable exception of sprint tests for running backs.”** This is not proof every athletic trait is useless.

**Masaru Teramoto, Chad L. Cross and Stuart E. Willick, Journal of Strength and Conditioning Research 30(5), 1379–1390, 2016, “Predictive Value of National Football League Scouting Combine on Future Performance of Running Backs and Wide Receivers.”**
https://doi.org/10.1519/JSC.0000000000001202 fetched publisher text (methods/results available). PubMed https://pubmed.ncbi.nlm.nih.gov/27100168/ separately returned incomplete/challenge content, not used as if it delivered the abstract.
Population: **“between 2000 and 2009 (N = 276 for RBs and N = 447 for WRs)”**; career outcomes through 2013. Complete-case and outlier filtering reduce actual regression samples.
Key quoted results:
- RB: **“After adjusting for #G in the first 3 years and Draft, 10-Y was the only significant predictor”**, uniquely **“9.2% of the variance”** in first-three-year yards/attempt.
- WR: **“Vertical jump alone could uniquely explain 3.7% of the variance in the career Y/R”**.

Disagreement with Kuzmits/Adams is plausibly explained by different cohorts, covariate adjustment, missing-data selection and rate outcomes rather than overall NFL success. Neither tests official RAS, this app's z-score substitute, tackles/coverage skill, or incremental value after multiple NFL seasons. Conditioning on future games played also makes these explanatory regressions different from a purely pre-draft forecast. Do not treat a one-drill partial association as a validated all-position RAS weight.

**Timo Riske, PFF, 2021-06-03, “Investigating athleticism and its effect on aging curves.”** https://www.pff.com/news/nfl-investigating-athleticism-and-its-effect-on-aging-curves retrieved preview, byline/date present. For combine-based normalized athletic scores since 2006, quote: **“correlation ... to NFL performance (measured by the PFF WAR rank within the position) is a positive 0.17.”** The later aging subset requires **“at least 1,000 snaps before their age-27 season.”** This is a custom score and selected established-player subset; neither equals validated official RAS. Directional modest athletic signal, no usable decay-to-zero year from the public preview.

### E9c. College production: direct TE evidence, weaker transport to share/breakout/IDP
**Jason Mulholland and Shane T. Jensen, JQAS 10(4), 381–396, 2014, “Predicting the draft and career success of tight ends in the National Football League.”**
Publisher https://www.degruyterbrill.com/document/doi/10.1515/jqas-2013-0134/html returned HTTP 405. University-hosted primary PDF https://repository.upenn.edu/bitstreams/003a752e-ee1c-4cb5-8860-f8595cb5dd67/download fetched via page-local answer mode.
Journal p.382: **“each tight end that participated in the NFL Combine or was selected in the NFL draft between 1999 and 2013”**, **315** total, **223** drafted. Different outcome models use different cohorts; these are not all mature careers.
Relevant p.392 draft-order-controlled finding: **“College yards per reception and college receptions were significant in the NFL Career Score model, college yards was significant in the NFL Career Score per Game model”**. Qualification p.395: **“We have also not accounted for the number of opportunities available to a player while in college.”**

This supports some college production information beyond draft selection at TE. It does NOT validate college team-share, universal breakout-age thresholds, school-tier bonuses or the same weights for defense. Blocking/playing-time information is absent. Broad-jump importance in unconditioned models does not survive as a universal conclusion once draft capital is controlled.

### E9d. Breakout age and production share: definition is established, portable predictive weight is not
**Ahaan Rungta, PlayerProfiler, 2022-08-16 (modified August 30), “2022 Breakout Finder: A Guide with Research Results.” Industry product/blog report, not independent validation.**
https://www.playerprofiler.com/article/2022-breakout-finder-guide/ fetched readable/raw; metadata verified. Definition: **“the age in which a receiver reaches a 20-percent dominator rating for the first time.”** It defines dominator through touchdown/yardage share. Model uses college metrics and fantasy breakout targets, reporting random-forest GINI importance. No held-out, leakage-audited marginal coefficient for breakout age conditional on draft capital and production was established from the fetched article.

Use as a WR prospect feature hypothesis, with exact definition and missingness recorded. Do not propagate its receiver threshold to QB, RB, TE or IDP as if validated. Feature importance is not a causal effect or blend weight. DLF's https://dynastyleaguefootball.com/2018/04/17/wide-receiver-breakout-age-by-draft-round/ returned HTTP 403; search-reported hit rates from that page are **not adopted**.

### E9e. Madden and consensus scouting
No primary result validating current Madden sub-attributes as incremental predictors of future individual production, conditional on known NFL production and draft capital, was established. The promising paper “Predicting plays in the National Football League” (Fernandes et al., 2020), https://journals.sagepub.com/doi/10.3233/JSA-190348, returned HTTP 403; its search description concerns play-call classification, not the requested player outcome. No borrowed accuracy number is used. Sector 1 also shows the actual code's m24 ratings are historical.

Likewise, no primary calibrated effect size/lifetime was established for a manually assembled consensus prospect board over and above actual draft capital. Scout consensus and draft capital are overlapping evidence, not independent votes. Madden may reflect already-observed production; combining both at face value risks double counting. Treat these as low-confidence priors with timestamped provenance, not ground-truth on-field measurements.

### E9f. Evidence by position and duration
- **QB:** draft capital broad cohort evidence; combine study gives no consistent general QB signal. No validated breakout-age or Madden coefficient. Increment beyond rookie production and expiration year unknown.
- **RB:** draft capital plus some acceleration signal in E9b; not proof of a composite RAS or college-share effect at a set weight. Receiving skill not measured by RB rushing Y/A outcome.
- **WR:** opportunity earning evidence E3; athletic effects outcome-dependent E9b; breakout/college-share exact marginal weight unknown E9d.
- **TE:** conditional college-production evidence E9c; blocking and usage omitted. Not an IDP analog.
- **DT/DE/LB/CB/S:** no fetched peer-reviewed validation of the app's particular college-share/breakout-age/Madden/RAS composites by true position. E9a pooled draft signal is the safer baseline, not high-confidence position-specific calibration.
- **K:** E6 supports context-adjusted NFL kicking and shrinkage; no validated RAS/breakout/Madden starting strength or duration.
For **every** position, the number of seasons until scouting stops adding information remains **could not establish**. Calendar tenure is a poor replacement for exposure and changing role, but that observation does not supply numeric k.

## E10. Age curves: multiple estimands, no universal flat 3% decay
### E10a. Why naïve aging curves mislead
**Timo Riske, PFF, 2021, “Investigating positional aging curves with PFF WAR.”** https://www.pff.com/news/nfl-investigating-positional-aging-curves-with-pff-war fetched preview/raw. Quote: **“The problem is survivorship bias.”** The article combines quality and playing time to incorporate injury, coaching decisions, front-office exit and retirement. Consequently its WAR-share-by-age curve is **not a pure on-field conditional skill decay**. Position plots/complete numeric peaks were not available in the returned preview.
E9b's athletic-aging follow-up states **“linemen also decline a bit later than players at other positions.”** That supports a qualitative distinction, not a DT/DE peak age or CB-versus-S slope. No verified numeric IDP age schedule is supplied.

Methodological primary lead fetched: **Michael Schuckers, Michael Lopez and Brian Macdonald, “What does not get observed can be used to make age curves stronger: estimating player age curves using regression and imputation,” arXiv 2021**, https://arxiv.org/abs/2110.14017 . Abstract supports regression/imputation for selection; no NFL parameter from it is adopted.

### E10b. QB rate-aging study (blog research, not a production default)
**Neil Paine, Football Perspective, 2013-08-06, “Another Quarterback Aging Curve Post (Adjusted Net Yards Per Attempt Edition).”**
https://www.footballperspective.com/another-quarterback-aging-curve-post-adjusted-net-yards-per-attempt-edition/ fetched readable/raw, byline/date verified. Uses era-relative ANY/A consecutive-year deltas and a cubic fit, requiring **“at least 15.1 dropbacks per game ... in back to back seasons.”** Exact underlying start/end cohort for the aging regression was not separately enumerated in fetched prose; do not assume the preceding since-merger leaderboard completely defines it.
Quote: **“quarterbacks peaks at age 27.”** Author contrasts a previous value-over-average approach giving a later peak (a different outcome). Table entries include age **27→28: -0.04**, **30→31: -0.14**, **35→36: -0.24** in relative ANY/A units, not percentages. Broad rise/plateau/accelerating decline is a hypothesis worth testing. Modern rushing QBs, selection and career exit are not resolved by that fit.

### E10c. RB total-production study (survivorship-limited)
**Chase Stuart, Football Perspective, 2014-03-20, “A closer look at running back aging patterns Part II.”**
https://www.footballperspective.com/a-closer-look-at-running-back-aging-patterns-part-ii/ fetched readable/raw.
Quote: **“723 running backs since 1970 who had at least 150 carries in consecutive seasons and who were between 21 and 32 in the first of those two seasons.”** These are season-pair observations in the analysis, not independent long-career individuals. Outcome: rushing **yards total**, combining workload and skill. Quote: **“running backs peak at age 26, and then begin a steady decline”**; author explicitly says survivorship remains unresolved and likely understates late decline. It does not warrant a fixed rushing-efficiency multiplier or the same decline on receiving work.

### E10d. Receiver career-peak distribution, not a WR-only causal curve
**Chase Stuart, Football Perspective, 2020-08-22, “Update: Top Receiving Seasons By Age.”**
https://www.footballperspective.com/update-top-receiving-seasons-by-age/ fetched readable/raw.
Sample: **“141 players ... 7,000 career receiving yards.”** Conclusion: **“ages 25 to 29 are the best seasons for these receivers.”** Includes active players and pass catchers such as Gronkowski, so this is NOT an unselected WR-only cohort. Conditioning on career yardage selects successful careers. Useful broad mid/late-twenties prior hypothesis, no forecast decay slope.

### E10e. TE: selected fantasy-output peaks, not all-player talent aging
**Mike Braude, Apex Fantasy Leagues, 2026-04-07, “The Peak Age For An NFL Tight End.” Original industry/blog analysis.**
https://apexfantasyleagues.com/peak-age-nfl-tight-end/ fetched readable/raw; metadata verified. Includes seasons since 2000 through 2025, threshold **“11 PPR points per game”**, minimum **“seven games played”**, **“183 qualifying seasons.”** Quoted comparison: **“Average Age 27.38 ... 26.78”** with versus without the five most prolific qualifiers. Those are average ages of selected high-output seasons, not the estimated peak of the same TE's latent skill curve. The selected-sample change demonstrates sensitivity to elite survivors, not permission to set peakAge=27.38.

### E10f. K: a cautionary non-prospective aging study
**Pads of the Hands (personal byline not established), 2019, “How do NFL Kickers Age?” Blog analysis.**
https://padsofthehands.blogspot.com/2019/09/how-do-nfl-kickers-age.html fetched readable/raw. Caption specifies **1960–2018**, prose **“369 NFL kickers.”** Most useful quote: **“a predictive model could not know beforehand how long a kicker’s career will be.”** Its normalized-career curves condition on eventual career length. Do NOT import their late-career threshold into forward projections. No reliable kicker chronological-age decay parameter established; distance/era/context-adjusted ability and retention must be distinguished.

### E10g. IDP age gaps
DT, DE, LB, CB and S each lack a verified numeric position-specific decline/peak in the accessible evidence. The PFF qualitative lineman result supports not treating every position alike; it does not prove safeties age more slowly than corners or supply a linebacker cliff. https://statvault.org/leaders/nfl/snap-aging fetched only a generic site shell; search-supplied peaks were excluded. https://www.thieme-connect.com/products/ejournals/pdf/10.1055/a-1485-0031.pdf failed PDF structure extraction, and corresponding https://pubmed.ncbi.nlm.nih.gov/34395825/ was incomplete; no extracted combine-longevity coefficient is presented as an age curve.
