# Legacy NFL — Application Modules Specification
**Version:** 1.0 — June 2026
**Status:** Design document. Each module is a separate build session.

---

## Overview

The application is built in layers. The scoring engine and data pipeline are the foundation. Modules are built on top of that foundation. Each module solves a specific problem. Together they form a complete GM intelligence tool.

---

## Module 1: 32-Team Asset Rankings

**What it does:** Processes all 32 rosters through the scoring engine and returns a single league-wide ranked asset list. Also produces per-team views.

**Inputs:**
- MFL roster and contract data (all 32 teams)
- MFL player database (positions, ages)
- External: RAS scores, snap counts, injury status

**Outputs:**
- Global ranked list of all rostered players by Adjusted Score
- Per-team ranked roster view
- Position-filtered views (e.g., all DTs league-wide ranked)
- Cap efficiency view (Adjusted Score per dollar of salary)

**Key design decisions:**
- Rankings update on a defined schedule (daily during offseason, after each game week in-season)
- Scouting layer weight is configurable for testing (default 3-5%)
- Cap tier boundaries are configurable pending calibration

---

## Module 2: Weekly Power Rankings

**What it does:** Ranks all 32 teams by combined roster strength, recent performance, and schedule context. Updates weekly.

**Inputs:**
- MFL standings (record, total points, all-play record)
- MFL weekly scoring history
- Scoring engine output (roster strength metric)
- NFL schedule (upcoming opponent strength)

**Methodology (to be built and tested):**
- Weighted blend of: recent scoring trend, total points, roster Adjusted Score, strength of schedule
- Weights are configurable and testable
- Not purely record-based — a 6-7 team with elite roster assets should rank differently than a 6-7 team depleted by injury

**Outputs:**
- Weekly power rankings with movement indicators (up/down from prior week)
- Rationale line for each team's ranking
- Trend visualization over the season

---

## Module 3: Matchup Score Predictions

**What it does:** Projects weekly scoring for each active matchup using historical player performance, opponent defensive strength, and current roster/injury state.

**Inputs:**
- MFL player scoring history by week
- MFL scoring settings (confirmed against rulebook)
- MFL weekly lineups (once set)
- Injury and depth chart data from RotoWire / Ourlads
- Opponent defensive rankings by position (from NFL Next Gen Stats, PFF)

**Methodology (to be built and tested):**
- Baseline: average recent scoring per player (last 4-6 weeks weighted)
- Adjustment: opponent defensive strength vs. position
- Adjustment: injury/snap count flags
- Adjustment: home/away (relevant for lineup decisions — not directly scored in this league)
- Output a projected score range (floor/ceiling) not just a point estimate

**Outputs:**
- Projected score for each side of each matchup
- Confidence range per projection
- Key variance players (boom/bust indicators)
- Start/sit suggestions for flex decisions

---

## Module 4: In-App Transaction System

**What it does:** Replaces the Proboards forum workflow for all transaction types. Bid submission, validation, clock management, and resolution happen inside the application.

**Transaction types covered:**
- UFA free agent bidding
- RFA offer sheets and match decisions
- Waiver claims
- Trade submissions and DOT voting
- Trade block management
- Franchise tag applications
- Contract extensions
- Contract restructures
- Contract buyouts

**Key capabilities:**
- Real-time bid validation against all rulebook constraints before submission
- Automatic 24-hour clock management per bid
- Snipe detection at 20-hour mark with automatic increment enforcement
- Dead cap auto-calculation on waiver posts
- Trade deadline hard block (timestamp enforcement)
- Trade format validation before DOT voting opens
- Cap and roster compliance check on every transaction type
- DOT vote tracking (3 approve / 3 veto to resolve)
- League-wide transaction feed (all active threads visible to all GMs)

**Phase 1 (Current target):** Full transaction management inside the application. Winning transactions generate correctly formatted output. GM or commissioner executes on MFL manually.

**Phase 2 (Future):** Authenticated MFL write access. Application submits winning transactions directly to MFL on behalf of the authenticated user.

**Phase 3 (Long-term):** Computer-use / screen-interaction layer. Click-through automation on MFL UI. Codex-style execution. Noted in scope — not a near-term build target.

---

## Module 5: Free Agency Intelligence

**What it does:** Surfaces the free agent pool with ranking engine scores, bid history, and positional need context for each team.

**Inputs:**
- MFL free agent list (players not on any roster)
- Scoring engine output for each free agent
- MFL team rosters (to calculate positional depth by team)
- Active bid threads

**Outputs:**
- Ranked free agent pool by Adjusted Score
- Per-position free agent rankings
- Team need overlay (flags positions where a team has thin depth)
- Active bid tracker with clock status for all open players
- Historical bid data (what comparable players have gone for)

---

## Module 6: Rookie Draft Intelligence

**What it does:** Supports rookie draft preparation and real-time draft execution with valuation, prospect scouting, and team need analysis.

**Inputs:**
- Prospect rankings and scouting data (The Draft Network, PFF, DLF, FantasyPros)
- RAS scores from ras.football
- Breakout age and school tier data (Matt Waldman RSP, Sharp Football — Christopher supplies annually)
- MFL team rosters (positional need calculation)
- Fixed rookie slot pricing from rulebook

**Outputs:**
- Pre-draft prospect rankings with scouting layer applied
- Per-pick value at each slot (what is expected vs. what is available)
- Team need analysis — absolute positional deficits override global scarcity rank per rulebook
- Real-time draft board (updates as picks are made)
- Post-draft valuation — how each team's draft class ranks

**Key rule:** If a team shows an absolute deficit at a position (e.g., only one active RB), team need temporarily overrides global positional scarcity rank in recommendations.

---

## Module 7: Trade Analyzer

**What it does:** Evaluates proposed trades against valuation data and surfaces the analysis for both teams and the DOT.

**Inputs:**
- Scoring engine output for all players in the trade
- MFL draft pick values (historical draft class performance)
- Both teams' cap situations post-trade
- Both teams' roster compositions post-trade
- League-wide positional scarcity context

**Outputs:**
- Side-by-side value comparison of assets exchanged
- Cap impact analysis for both teams
- Roster need alignment (does each team get what they need?)
- Historical comparable trades from league history
- DOT-facing summary report

**Critical note:** The trade analyzer surfaces information. It does not make DOT decisions. The NYG/WAS veto was a judgment call about league competitive health — a perpetual rebuild pattern that no valuation metric captures. The analyzer supports DOT. It does not replace them.

---

## Module 8: Commissioner Dashboard

**What it does:** Gives commissioners and DOT members a single view of league health, active issues, and pending decisions.

**Inputs:**
- All transaction feeds
- Power rankings
- Cap compliance status all 32 teams
- Roster compliance status all 32 teams
- Active DOT votes
- Flagged issues (teams approaching cap, roster limits, etc.)

**Outputs:**
- League health overview
- Active transaction queue (bids, claims, trades pending)
- DOT vote management interface
- Cap and roster alert flags
- Trade deadline status

---

## Future Modules (Noted in Scope)

**Historical Analytics:** Season-over-season performance tracking, GM record analysis, franchise history.

**Franchise Valuation:** Long-term asset value scoring — not just current season production but dynasty value accounting for age, contract length, and positional scarcity.

**Offseason Planner:** Cap space projection, contract expiration calendar, free agency target list builder.

**Mobile Interface:** Full mobile-optimized version. GMs manage the league from their phones — the Proboards forum threads show this is already how they play. The application needs to meet them there.
