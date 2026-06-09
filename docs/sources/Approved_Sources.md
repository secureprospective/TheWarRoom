# Legacy NFL — Approved Source Library
**Version:** 1.1 — June 2026
**Status:** Locked. All sources approved by Christopher Campbell. Madden added in document audit pass (v1.1).

---

## How This Library Is Used

These sources feed the application in three ways:
1. **Data pipeline inputs** — structured data pulled programmatically (stats, snap counts, depth charts)
2. **Scouting layer inputs** — film analysis, prospect grades, RAS scores
3. **Weekly intelligence** — injury news, player updates, IDP analysis

Each source is documented with its URL, what it provides, how frequently it updates, and its primary use in the engine.

---

## TIER 1 — Official & Authoritative Data

### NFL.com / Next Gen Stats
**URL:** nextgenstats.nfl.com
**Free access:** Yes — full
**What it provides:** Official NFL player tracking. Passing, rushing, receiving, and defensive advanced stats by week and season. Air yards, time to throw, completion probability, target separation, coverage responsibility metrics.
**Update frequency:** Day after each game. Live during games.
**Primary use:** Advanced stat context for scoring projections. Player tracking metrics for scouting layer calibration.

### Pro Football Reference
**URL:** pro-football-reference.com
**Free access:** Yes — core stats free. Stathead search tool has soft subscription.
**What it provides:** Historical statistics gold standard. Every player, every season, every box score since 1920. Career stats, experience years, games played.
**Update frequency:** Weekly in-season. Daily for transactions.
**Primary use:** Player experience years (practice squad eligibility). Career game counts. Historical production baselines.

### NFL.com Player Profiles
**URL:** nfl.com
**Free access:** Yes — full
**What it provides:** Official roster data, experience tracking, injury designations. The rulebook explicitly references NFL.com for experience-based minimum salaries and practice squad eligibility.
**Update frequency:** Daily.
**Primary use:** Source of truth for experience years and practice squad eligibility tracking. Rulebook-specified authority.

---

## TIER 2 — Advanced Stats & Analytics

### Pro Football Focus (PFF)
**URL:** pff.com
**Free access:** Partial — player grades and some advanced stats free. Deeper data (pass rush win rate, route grades, coverage grades) behind subscription.
**What it provides:** Play-by-play grading database. Player grades 0–100 on every snap. Both NFL and college.
**Update frequency:** Weekly in-season. Draft season for college grades.
**Primary use:** Scouting layer — player grade as a film consensus proxy. Depth charts with PFF grades for all 32 teams.

### Relative Athletic Score (RAS)
**URL:** ras.football
**Creator:** Kent Lee Platte
**Free access:** Yes — full
**What it provides:** Scores every NFL prospect 0–10 vs. positional history back to 1987. Combine and pro day measurements. The exact metric the scoring engine uses for the tiebreaker protocol and scouting layer.
**Update frequency:** Combine season (February/March). Pro day season (March/April). Updated as new data arrives.
**Primary use:** Direct engine input. RAS score is a first-class data field in the scoring engine output schema.

### EA Sports Madden NFL (Madden API / MAPI)
**URL:** Madden player attributes accessible via third-party aggregators and community-maintained APIs
**Free access:** Attributes widely documented; official EA API access varies by release
**What it provides:** Annual player attribute ratings on a 0–99 scale across all positions. Positional sub-attributes (Speed, Awareness, Throw Power, Catching, etc.) are the regulation inputs for Approach D Madden regulation in the scouting layer.
**Update frequency:** Annual (August EA release) + mid-season roster and attribute updates.
**Primary use:** Subjective expert claim regulation (Approach D). Madden sub-attributes serve as a computational check on subjective scouting claims from RSP, TDN, Sharp, Dynasty Nerds, and IDP Show. Analytical signals (PFF, NGS, IDP Guru) are self-regulating and do NOT pass through Madden regulation. Kicker data archived for potential future use (CAL-032) but does not affect Layer 4 output.

---

## TIER 3 — Weekly Fantasy & IDP Analysis

### FantasyPros
**URL:** fantasypros.com
**Free access:** Yes — core rankings, snap counts, depth charts, start/sit tools
**What it provides:** Consensus rankings from 100+ analysts. IDP rankings. Snap count data by position, team, and week. Depth charts. Gameday inactives.
**Update frequency:** Daily. Snap counts updated after each game week.
**Primary use:** Weekly snap count data. IDP consensus rankings. Gameday inactive lists.

### The IDP Guru
**URL:** idpguru.com
**Free access:** Partial — weekly IDP rankings and select articles free. Some premium content behind paywall.
**What it provides:** Six-time FantasyPros Most Accurate IDP Rankings winner. Weekly defensive player analysis, snap counts, scheme impact breakdowns, home stat crew tackle tracking (tackle credits vary by crew — directly relevant to IDP scoring).
**Update frequency:** Thursday rest-of-season rankings. In-season weekly.
**Primary use:** IDP weekly intelligence. Stat crew variance tracking — important for DT/DE tackle scoring accuracy.

### The IDP Show
**URL:** theidpshow.com
**Free access:** Yes — tiered IDP rankings by position free
**What it provides:** Weekly IDP rankings by position (DL, LB, DB). Film-informed defensive player analysis. Directly relevant to this league's True Position defensive scoring structure.
**Update frequency:** Weekly in-season. Pre-draft rankings in offseason.
**Primary use:** DL, LB, DB weekly rankings. Film-based defensive player evaluation.

### Dynasty Nerds
**URL:** dynastynerds.com
**Free access:** Partial — free IDP dynasty rankings, rookie mock drafts, weekly analysis. Premium tier for full platform.
**What it provides:** IDP dynasty rankings. Prospect film room previews. Rookie mock drafts. Weekly articles.
**Update frequency:** Weekly.
**Primary use:** IDP dynasty context. Rookie prospect previews. Offseason ranking updates.

### RotoBaller
**URL:** rotoballer.com
**Free access:** Yes — core content free
**What it provides:** Player news, injury updates, dynasty analysis, weekly rankings. Strong on IDP and dynasty.
**Update frequency:** Multiple times daily. Real-time injury updates.
**Primary use:** Player news feed. Dynasty analysis. IDP weekly content.

### RotoWire
**URL:** rotowire.com
**Free access:** Partial — news feed and depth charts free. Analysis behind subscription.
**What it provides:** Real-time injury news. Depth charts updated daily. IDP rankings. Free agent tracker.
**Update frequency:** Real-time for news. Daily for depth charts.
**Primary use:** Injury news feed. Depth chart monitoring. Real-time roster change tracking.

---

## TIER 4 — Scouting, Film & Prospect Evaluation

### The Draft Network
**URL:** thedraftnetwork.com
**Free access:** Yes — prospect profiles, scouting reports, TDN100 rankings
**What it provides:** Expert scouting reports on NFL Draft prospects. Positional rankings. Film breakdowns. The best free source for pre-draft scouting report language and trait evaluation.
**Update frequency:** Draft season (January through May). Year-round for breaking news.
**Primary use:** Prospect scouting reports. Film-based trait evaluation for rookie scouting layer.

### Pro Football Network
**URL:** profootballnetwork.com
**Free access:** Yes — full
**What it provides:** RAS tracking by combine class. Scouting reports. Mock drafts. Annual combine RAS tables by position updated through pro day season.
**Update frequency:** Draft season for RAS. Year-round for news.
**Primary use:** Companion RAS source. Combine context and position-group RAS comparisons.

### Dynasty League Football (DLF)
**URL:** dynastyleaguefootball.com
**Free access:** Yes — rookie rankings, dynasty trade values, ADP, offseason articles
**What it provides:** Long-running dynasty-specific site. Multi-analyst rookie rankings. Dynasty trade values. ADP data.
**Update frequency:** Draft season. Weekly in-season.
**Primary use:** Rookie rankings from multiple analysts. Dynasty trade value reference.

### KeepTradeCut (KTC)
**URL:** keeptradecut.com
**Free access:** Yes — full
**What it provides:** Crowdsourced dynasty trade values from millions of data points. Updated in real time by community activity. Market-consensus valuation of players and picks.
**Update frequency:** Real-time (crowdsourced).
**Primary use:** Market consensus check on player and pick values. Trade analyzer reference.

---

## TIER 5 — Weekly Operational Tools

### Ourlads
**URL:** ourlads.com
**Free access:** Yes — full
**What it provides:** The most consistently accurate free NFL depth charts available. Updated multiple times weekly. Specifically tracks practice squads and reserve designations.
**Update frequency:** Multiple times weekly. Daily during season.
**Primary use:** Depth chart monitoring. Practice squad tracking — directly relevant to this league's practice squad eligibility rules.

### FantasyPros Snap Counts
**URL:** fantasypros.com/nfl/reports/snap-counts
**Free access:** Yes — full
**What it provides:** Weekly snap count data broken down by position, team, and week. Both raw snaps and percentage of team snaps.
**Update frequency:** After each game week.
**Primary use:** Weekly snap count data for IDP and offensive player role verification.

### ESPN NFL Depth Charts
**URL:** espn.com/nfl
**Free access:** Yes — full
**What it provides:** Depth charts for all 32 teams updated throughout the season. Mike Clay dynasty rankings and IDP rankings freely accessible.
**Update frequency:** Throughout the week. Daily during season.
**Primary use:** Depth chart monitoring. Dynasty ranking reference.

---

## FILM ANALYSIS — Direct Access (Christopher Supplies)

### Matt Waldman's RSP
**URL:** mattwaldmanrsp.com
**Access:** Christopher has direct access. Annual RSP publication supplied year to year.
**What it provides:** Film-based skill position prospect analysis. Free blog content covers technique, route running, and player evaluation framework throughout the year. Annual Rookie Scouting Portfolio (RSP) is the paid deep-dive product.
**Update frequency:** Annual publication (April 1). Blog posts year-round.
**Primary use:** Breakout age data. Film-based trait evaluation for offensive skill positions. Scouting layer calibration.

### Sharp Football Analysis
**URL:** sharpfootballanalysis.com
**Access:** Christopher has subscription access.
**What it provides:** Rich Hribar's dynasty and rookie analysis. Post-draft rookie rankings. Free articles published through draft season and into the year.
**Update frequency:** Draft season. In-season dynasty analysis.
**Primary use:** Rookie dynasty rankings. Breakout age and prospect profile context for scouting layer.

---

## NOT APPROVED / OFF THE TABLE

### Proboards / legacynfl.boards.net scraping
Scraping violates Proboards Terms of Service. The forum is the source for historical human behavior documentation (done manually via screenshot and curation — see League History folder). Automated scraping is not permitted.

### Any source behind a full paywall with no free tier
The application is built on free and publicly accessible data. Paywalled sources are supplemented by Christopher's direct access where applicable (Waldman, Sharp) but are not programmatic data sources.
