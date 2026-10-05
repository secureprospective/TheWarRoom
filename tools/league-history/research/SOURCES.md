# Behavioral analytics research — sources and triage

Research date: 2026-10-05. Scope: league managers/GMs, as Christopher clarified.
Primary pages/full texts were inspected, not just search-result summaries. Sources
support methods or hypotheses; none validates a turnkey personality score for a
32-team salary-cap IDP dynasty league. “No matching study found” is a search result,
not a claim that no such study exists.

## Evidence classes

- **Research:** inspectable academic work; still distinguish its population from this league.
- **Methods:** established analytical/interaction technique, adapted here by engineering judgment.
- **Commercial:** product or vendor-authored observational research, not independently replicated.
- **Analogue:** related domain, not evidence of a specific GM's motive.

## S1 — Actual fantasy-football draft behavior

**Michael D. Lee and Siqi Liu (2022).** “Drafting strategies in fantasy football:
A study of competitive sequential human decision making.” *Judgment and Decision
Making*, 17(4), 691–719.

https://doi.org/10.1017/S1930297500008901

Verified full article at Cambridge Core. Abstract: “based on 1350 leagues” from the
2017 NFL season; choices in some circumstances were influenced by the immediately
preceding competitor choice. The paper found no evidence of irrational influence
from NFL-team preferences and little/no evidence of handcuffing in that sample.

**Adopt:** draft sequence, available-pool context, positional selection distributions,
conditional response to a run. **Adapt:** a dynasty rookie draft is not this study's
initial-roster draft; IDP and roster rules alter the pool and demand. **Reject:** importing
redraft archetypes or assuming NFL-team fandom bias before measuring it locally.

## S2 — Skill and planning across fantasy seasons

**Joseph D. O'Brien, James P. Gleeson and David J. P. O'Sullivan (2021).**
“Identification of skill in an online game: The case of Fantasy Premier League.”
*PLOS ONE*, 16(3), e0246698.

https://journals.plos.org/plosone/article?id=10.1371/journal.pone.0246698

Verified full article. It describes persistent cross-season performance correlation,
planning/decision patterns, and template teams. The detailed 2018/19 analysis has
901,912 identified managers, colloquially described as the top million. FPL has
manager-to-game transfers and its own prices/chips, not dynasty bilateral trades.

**Adopt:** distinguish repeated decisions from individual noisy outcomes; examine
consistency and change over seasons. **Adapt:** compute local, rule-aware behavior
and outcomes separately. **Reject:** claiming its coefficients or captain/chip metrics
apply to this league, or that a single championship proves trading skill.

## S3 — Draft-market overconfidence and time preference

**Cade Massey and Richard H. Thaler (2013).** “The Loser's Curse: Overconfidence
vs. Market Efficiency in the National Football League Draft.” *Management Science*.

https://faculty.wharton.upenn.edu/wp-content/uploads/2013/08/massey---thaler---losers-curse---management-science-july-2013.pdf

Full PDF downloaded and text inspected. Section 4 studies 1983–2008 NFL pick trades.
From 1,078 trades it excludes player-containing and inconsistent exchanges, leaving
314 current-year-only trades and 94 current/future-year trades. Section 4.2 reports
a model-implied 136% annual discount rate and discusses “gain a round by waiting a
year.” That is an NFL sample/model result, not a dynasty discount to install here.

**Adopt:** distinguish current from future picks, count direction and horizons, and
study clean pick-only exchanges separately from mixed packages. **Defer:** estimating
this league's revealed pick-price curve until bilateral exchanges, identities, slot
uncertainty and sample adequacy are audited. **Reject:** importing the NFL chart,
136% discount or a universal “trade down is always right” rule.

## S4 — The objective function changes the interpretation

**Ryan S. Brill and Abraham J. Wyner.** “The Loser's Curse and the Critical Role
of the Utility Function.” Read the accessible 2025 arXiv v4 full text; RePEc lists
the journal version in *The American Statistician*, 80(1), 15–30 (2026).

https://arxiv.org/html/2411.10400v4

https://ideas.repec.org/a/taf/amstat/v80y2026i1p15-30.html

Publisher page returned 403; its full text was not reviewed. The inspected preprint
argues that prioritizing elite/transformational-player probability can make behavior
that looks inefficient under expected surplus value appear rational.

**Adopt:** expose win-now versus longer-term objective and actual roster constraints.
**Reject:** calling an exchange irrational merely because a single value model dislikes
it. **Qualification:** use this as a counterargument to S3, not proof of local GM motives.

## S5 — Retention after bad acquisitions

**Roberto Pedace and Janet Kiholm Smith (2013).** “Loss Aversion and Managerial
Decisions: Evidence from Major League Baseball.” *Economic Inquiry*, 51(2), 1475–1488.
Accessible author manuscript inspected; distinguish draft from final publisher text.

https://economics.ucr.edu/wp-content/uploads/2019/10/Baseball_20110519_clean.pdf

https://doi.org/10.1111/j.1465-7295.2012.00463.x

The manuscript studies 15,880 player–team–year observations and contrasts new versus
continuing GMs. It expressly says, “We cannot know for sure what is in the mind of
the acquiring manager.” Retaining an acquired underperformer is evidence to examine,
not a direct reading of intent.

**Adapt:** acquisition source, observed retention/release, contemporaneous performance,
manager tenure and contract constraints. **Defer:** explanatory modeling until ownership
and acquisition/release episodes are validated. **Reject:** “stubborn,” “loss-averse” or
“sunk-cost-biased” labels derived from transaction counts alone.

## S6 — Complementary rosters rather than universal trade value

**Aaron Baughman et al. (2021/2022).** “Large Scale Diverse Combinatorial Optimization:
ESPN Fantasy Football Player Trades.” Abstract/method overview inspected; the complete
optimization implementation was not independently evaluated.

https://arxiv.org/abs/2111.02859

It describes valuations personalized to league rules/rosters, positional depth/slots,
and pairing complementary teams. The reported expert-evaluated trade-quality rates
are not demonstrated GM acceptance probabilities or evidence of dynasty success.

**Adopt:** compare positional needs and package structure under local rules.
**Defer:** automated trade recommendation until time-consistent needs/values exist.
**Reject:** copying its quantum/deep-learning complexity or marketing accuracy figures.

## S7 — Weighted trading relationships

**Alain Barrat, Marc Barthélemy and Alessandro Vespignani (2004).**
“Characterization and Modeling of weighted networks.” Full accessible text inspected.

https://arxiv.org/html/cond-mat/0408566v1

Section 2 defines strength and disparity: Y2(i) = sum_j (w_ij / s_i)^2. Equal weights
across k partners give approximately 1/k; a small set of dominant partners produces
larger concentration. Related foundation:

https://arxiv.org/html/cond-mat/0311416v1

**Adopt:** distinct counterparties, number of deals, partner shares, concentration and
its reciprocal effective-partner count. **Adapt:** completed deals form an undirected
relationship network; asset transfers can be directed. **Defer:** “unusually concentrated”
until a time/participation-preserving null is tested. **Reject:** collusion, friendship,
trust or negotiation-initiative claims inferred from a network edge.

## S8 — Coordinated visual analysis, not unrelated report tabs

**Jeffrey Heer and Ben Shneiderman (2012).** “Interactive Dynamics for Visual
Analysis.” *Communications of the ACM*, 55(4). Full PDF inspected.

https://idl.cs.washington.edu/files/2012-InteractiveDynamics-CACM.pdf

The taxonomy includes filtering, linked views, history and provenance. The Coordinate
section describes selecting items in one view to highlight/filter matching records
in others and comparing small multiples on common scales.

**Adopt:** one explicit analysis context shared by the calendar, manager comparison,
asset flows, partner matrix and evidence ledger. Preserve a visible peer baseline,
selection history and extraction of the supporting records. **Reject:** chart islands,
hidden filter scope and decorative graphs with no evidence drill-through.

## S9 — Overview, filtering, then evidence

**Ben Shneiderman (1996).** “The Eyes Have It: A Task by Data Type Taxonomy for
Information Visualizations.” PDF accessible; well-established interaction taxonomy.

https://www.cs.umd.edu/users/ben/papers/Shneiderman1996eyes.pdf

**Adopt:** overview → zoom/filter → details on demand, plus relate/history/extract.
This supports a calendar as an analysis controller, not just a file-navigation graphic.
It does not settle metric definitions or scientific validity.

## S10 — Proportions need an explicit inference model

**NIST Dataplot:** “Proportion Confidence Interval.” Documentation inspected.

https://www.itl.nist.gov/div898/software/dataplot/refman1/auxillar/propconf.htm

The documentation recommends Wilson/Jeffreys over naive normal intervals for small
binomial samples. That does not make repeated, dependent manager deals independent
Bernoulli trials.

**Adopt:** distinguish observed archive percentages from estimates of repeatable future
behavior; show numerator and denominator. **Adapt:** clustered/season-aware uncertainty
for modeled tendencies; label assumptions and unstable samples. **Reject:** “100% likely
to accept” from one successful exchange, and intervals that pretend unknown archive
coverage has been measured statistically.

## S11 — Commercial precedent: manager dossiers

**The Sunday Chronicle**, “Fantasy football manager analysis.” Product-authored guide
inspected; implementation, formulae and accuracy claims were not verified.

https://thesundaychronicle.app/guides/fantasy-football-manager-analysis/

The guide names drafting, lineup, trade/waiver activity and response to adversity,
organized in manager dossiers. Its “most complete”/Manager DNA assertions are marketing.

**Adopt as product precedent:** put behavior beside results and evidence in a dossier.
**Reject as science:** unsourced personality/DNA scoring and undefined “trade fairness.”
No proof of this product's MFL/32-team/IDP support was established.

## S12 — Commercial precedent: timing, not raw trade-volume worship

**Dynasty Trade Maker**, “Do Teams That Trade More Win Fantasy Football?”
Full noscript article inspected in the actual HTML after readable extraction failed.

https://dynastytrademaker.com/research/do-teams-that-trade-more-win

The vendor reports 167,265 trades from 6,976 completed Sleeper leagues, 3.07 trades
for champions versus 2.76 for others, identical medians of one, and different timing
shares. No dataset/code was independently audited. It also draws causal-sounding
conclusions about overpaying and titles that its displayed observational tables do
not establish.

**Adopt as hypotheses:** trading phase, contender/rebuilder context, deal direction.
**Reject:** numerical benchmarks and prescriptions for this league; playoff-week
activity cannot transfer to a league with a different trade deadline.

## S13 — Commercial corroboration, with selection bias

**Dynasty Dealmaker**, “We Analyzed 2.2 Million Dynasty Seasons. Here's How Champions
Trade Differently.” Vendor-authored full article inspected.

https://www.dynastydealmaker.com/blog/champions-trade-more

It describes roster-seasons, not 2.2 million independent leagues. Reported means are
3.42 versus 3.21 trades, alongside net pick flows and a U-shaped relationship with
final standing. The page has a minor 3.42/3.43 inconsistency and unverified causal
language. Final standing is known after decisions and cannot be a pretrade input.

**Adopt as questions to test locally:** whether contenders send picks, rebuilders receive
picks, and activity differs across competitive states. **Reject:** “spend picks to win”
as causal proof, universal rates or a norm to optimize toward.

## S14 — Practitioner writing, not a measured diagnosis

**PFF**, “Fantasy: Beware the Fallacy of the Sunk Cost.” Article inspected.

https://www.pff.com/news/fantasy-beware-the-fallacy-of-the-sunk-cost

**Use:** terminology and a testable retention question. **Do not use:** evidence that a
particular GM is irrational. Ownership attachment, acquisition cost, changing forecasts,
injuries and contract restrictions are different explanatory mechanisms.

## Not adopted / access limits

- Hobbs's professional-sports endowment-effect article was located at
  https://doi.org/10.1111/ecin.13102 but returned 403. Its full findings were not
  reviewed, so search-summary claims are not used as a foundation.
- Generic dynasty trade databases can illustrate available products, but no historical
  snapshot coverage, licensing, or matching 32-team cap-IDP valuation was established.
- No verified publication located in this pass supplies ready-made behavioral norms
  specifically for this league format. Local comparison and validation remain necessary.
- Literature-derived hypotheses are not findings about any of this league's managers.
