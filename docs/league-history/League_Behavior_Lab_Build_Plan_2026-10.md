# League Behavior Lab: Build Plan

Date: 2026-10-05. Branch: `session/league-history-pwa`. Status: **built 2026-10-05**, serving at
http://localhost:8765/ and waiting for Christopher's review. Not committed. Decisions taken: owners not
tracked (franchise slots only), Beelink desktop only, Bee's explorer replaced. How to run, check
and change it: `tools/league-history/README.md`. Research basis:
`Dynasty_Market_Psychology_Research_2026-10.md` (same folder).

## Purpose

An interactive tool for asking questions of 14 seasons of Legacy NFL behaviour (32
teams, IDP, salary cap). The published research comes from 12-team offense-only
leagues, so it supplies only the *questions and levers*. Every answer comes from this
league's own data.

## What already exists, and the call on it

Bee built a seven-view "behavior explorer" on this branch earlier today
(`tools/league-history/`, serving on localhost:8765). It is still waiting for
Christopher's acceptance.
- **Worth keeping:** its archive compiler, its tests, and its verified totals
  (2,634 completed trades; 1,725 trades that include picks; 3,780 pick moves). Its
  data-quality rules are also right: unknown is not zero, and a franchise slot is not
  a manager.
- **Not fit for this purpose:** the interface is the same six-dropdown filter wall on
  every tab, followed by tables. It has one chart (the calendar heatmap) and a lot of
  jargon ("contextual deals before asset facets", "HHI", "pp"). It can tell you what a
  franchise received. It cannot answer behavioural questions, such as what teams
  sitting 10th–16th at week 8 do, or what a 1st costs in March versus October.

**Decision:** build the Lab as version 2 in the same folder.
- Reuse and extend Bee's compiler and its tests.
- Replace its interface.
- Keep Bee's own acceptance case (Arizona, round-1 picks received, March 2024: 2
  received, 1 sent) as a regression test.
- Retire the old views once the Lab reproduces every check they passed.

## What the archive actually holds (checked 2026-10-05)

- **Coverage:** 14 seasons (2013–2026; 2026 partial). Weekly rosters exist for every
  season, with salary, contract year, RFA/UFA status and extension text attached to
  each rostered player.
  - Contract fields are sparse in 2015–2016 and solid from about 2019. **To confirm in
    Stage 0.**
- **Transactions:** trades; trade proposals, accepts, rejections and revokes (only
  partly kept); free-agent adds; waiver claims (2013–16); taxi-squad and IR moves; and
  commissioner roster loads.
- **Salary adjustments:** cuts and dead money, timestamped.
- **Drafts:** rookie draft picks with timestamps (2017 onward); future-pick ownership
  snapshots; standings, schedules, weekly player scores and projections; rules; a
  message board.
- **League settings:** cap $115; roster 48, taxi squad 8, IR 12; last regular-season
  week is 13.
- **This league's calendar is its own:**
  - The trade window opens around Feb 15 – Mar 16.
  - The trade deadline falls between Oct 25 and Nov 6.
  - Trading then stops until the next season (as of 2023–2025, nothing until March).
  - The rookie draft has moved from May (2017) to mid-July (2024–2026). It is an
    email draft that runs about 10–16 days.
  - So the "December deadline frenzy" and "April pick peak" from the research do not
    exist here in that form.
- **Missing:** birthdates are not in the archived player file. Age has to come from the
  app's MFL player directory, which does carry birthdates. **To confirm in Stage 0.**
- **Missing:** there is no record of who managed which franchise, and when. Without
  that, results are about franchise slots, not people.
- **Oddities to explain before anyone trusts the numbers:**
  - March 2017: 143 trades recorded, but only 10 pass the clean-trade rule (a bulk
    load).
  - Washington: 597 trades, about twice the next franchise. It needs explaining.
  - Picks are not encoded in trades for 2013–2016.

## Architecture

1. **Facts as they were known at the time.** Every event is stored with the acting
   team's situation *at that moment*: record, standings rank, games from the playoff
   line, points rank, cap room, committed salary, contract load and roster age.
   Final-season results never leak into a decision; they appear only under an explicit
   **Hindsight** switch.
2. **Pricing without a market.** No historical KeepTradeCut values exist for this
   league, so assets are priced from inside it:
   - (a) What the league actually paid: exchange rates between picks and players.
   - (b) Production known at the time: trailing points above replacement at 32-team
     depth, by position, IDP included.
   - (c) Salary and years left on the contract.
   - What an asset produced *after* the trade is shown only under Hindsight.
3. **The league's own clock.**
   - Each year's trade window, deadline, rookie draft, cut timing and contract rollover
     are taken from that year's own events.
   - Any chart can line up years by calendar date, by days to the deadline, or by days
     to the rookie draft, so seasons compare fairly even though the draft moved two
     months.
4. **Contending vs rebuilding is a dial, not a fixed label.** The rule is visible and
   based on standing and points at the time. You can drag its thresholds.
5. **In-browser query engine.**
   - The Python compiler (standard library only) produces compact column-oriented
     data.
   - A small, dependency-free filtering engine in the page recounts everything on each
     click, with a budget of under 100 ms.
   - Every chart is also a filter: brush a date range, click a team or a position, and
     every other view recounts.
   - D3 is bundled into the app (free, works offline) for drawing only.
6. **App delivery.**
   - It installs and runs offline as a PWA.
   - The whole lens lives in the URL, and saved lenses are named.
   - Every number drills down to the trades behind it.
   - The existing hardened local server is reused.
   - The generated private data stays out of git, as now.

## The levers (one global bar of chips, not a dropdown wall)

| Lever | Options |
|---|---|
| Time | Season range; brush a window on the calendar; line up years by date, deadline or rookie draft |
| Who | Franchise(s); trading partner; manager (once the owner map exists) |
| Team state at the time | Contending / bubble / rebuilding (thresholds adjustable); cap room band; roster age |
| Picks | Round; years out; the original owner's standing at the time of the trade |
| Players | Position group (QB/RB/WR/TE, DL/LB/DB, K); age band; salary band; contract years left; production tier at the time |
| Trade shape | 1-for-1; consolidation (2+ for 1); picks only; salary dump; 3+ assets |
| Evidence | Clean trades / everything recorded; Hindsight on or off |

## Views

1. **Pulse.** Each season is a strip of league days, and you choose what to count:
   trades, picks moved, adds, cuts, dead money, taxi or IR moves. Overlays mark that
   year's actual window, deadline and draft. This shows when the league moves.
2. **Market.** The exchange-rate explorer:
   - What a 1st, 2nd or 3rd bought, by phase of the year.
   - Whether a future 1st's price follows its original owner's record (the "mid"
     mistake).
   - Which way picks and players flow between contending and rebuilding teams, week
     by week.
3. **Teams in time.** All 32 teams plotted by standing and points, played forward
   week by week, with each team's buying or selling marked. The deadline appears as a
   moment in that animation.
4. **Cap & contracts.**
   - When cuts and dead money happen.
   - Cap room versus buying.
   - Dumps of expiring contracts.
   - Extensions, taxi-squad and IR use.
5. **Draft at 32.** How often each pick slot pays off, by round and position, using
   *this league's scoring* (IDP included). It also compares what was paid to acquire
   picks with what those picks returned.
6. **Franchises (managers once the owner map exists).**
   - Each one's habits against the league baseline: timing, partners, trade shapes,
     and how it behaves from each team state.
   - A 32-team partner network.
7. **Questions.** The 11 research hypotheses, plus anything the data turns up that is
   unique to this league. Each one is a one-click lens with a one-sentence answer, its
   chart and its sample size.
8. **Ledger.** Every number opens the trades behind it. Search any player.

## Stages and gates

| Stage | Work | Gate |
|---|---|---|
| 0. Data truth | Explain the 2017 bulk load and Washington's volume; confirm contract coverage by year, birthdate source, pick encoding and how much of the proposal history was kept; derive each year's windows | A data-facts note; totals match Bee's verified counts |
| 1. Fact compiler | Tables for trades, trade legs, team-weeks as they stood at the time, roster-weeks with contracts, cuts and dead money, adds, drafts, and pick lineage (original owner → each holder → the player taken → his output) | Tests that check the totals hold together, including that each pick's chain of owners is complete |
| 2. Engine and levers | Filtering engine, lever bar, URL lenses, ledger drill-down | Every click recounts in under 100 ms; Bee's Arizona case passes |
| 3. Pulse and Market | | Screenshots reviewed at desktop and phone widths |
| 4. Teams in time; Cap & contracts | | Same |
| 5. Draft at 32 | | Same |
| 6. Franchises / managers and network | Uses the owner map | Same |
| 7. Questions board | Hypotheses answered, each with its sample size | Each answer reproducible from its lens |
| 8. Finish | Offline PWA check; Claude-OS browser check; Bee review when GPT budget allows (R4); old views retired | Christopher accepts; merge only on his go |

## Needs from Christopher

1. **Owner map:** who ran each franchise, and from when to when. This is what turns
   "franchise slot" into "manager." Without it, the manager psychology stays at the
   franchise level.
2. **Where he will use it:** the Beelink desktop only (works now), or also phone and
   other machines on the home network (needs private HTTPS on the LAN).
3. **Veto, if wanted,** on replacing Bee's explorer interface.
