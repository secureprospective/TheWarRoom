# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users
Christopher, a GM in the Legacy NFL (a 32-team dynasty IDP salary-cap league on MFL). He already
knows his own team (the Arizona Cardinals, franchise 0025) inside out. He opens the Lab to see the
league: how the other 31 GMs price assets, and how that changes through the year.

## Product Purpose
Show where the league as a whole misprices assets, across kinds of asset and across the calendar,
so he can plan what to buy and sell and when. Success: in a few seconds he can say what the league
overpays for, what it underpays for, and which time of year moves the price, with how far to
trust each answer.

## Positioning
Every number comes from the league's own MFL record, checked against MFL's figures (the facts
database in `store/`), and is valued in one currency (wins above replacement). Prices come from
what this league's GMs actually traded, not from outside rankings.

## Operating Context
- Opened at a desk on a wide screen: a local web app on the Beelink (loopback, port 8765), offline
  capable. Desktop is the design target; phones are not a requirement.
- Use order (confirmed 2026-10-06): the league map first (what the league overpays for, and when);
  looking up a specific trade or pick second.
- Rebuilt from `python3 store/build.py` then `python3 compile/build_lab.py`.

## Capabilities and Constraints
- League-wide only (confirmed 2026-10-06): no "your team" selector and no team highlighting. The
  trade log keeps a plain team filter for lookups. Round 9 (2026-10-06): the trade log became "How the teams
  trade", every team in depth, still league-wide: no team is highlighted as his.
  The one exception, chosen by Christopher 2026-10-06: the offers chart, because MFL shows a team
  only its own offers; it is labelled as the Cardinals' offers.
- Measures: price per expected win for each kind of asset (against a next-draft 1st), wins
  delivered against the forecast at the time, the time of year, the draft against its slots.
- The market screen filters by stretch of the league year (each season's own draft, kickoff and
  deadline dates) and by asset (position, pick round and draft, age band); confirmed 2026-10-06.
  Narrow filters price fewer trades, so ranges widen; under 80 valued trades nothing is priced.
- The draft screen measures each pick as a percent of what its slot returned, filtered by draft
  stretch, position, NFL round and judging window, with a per-position view (confirmed 2026-10-06:
  draft timing and player value are the core focus).
- Trades are valued from 2017 (picks recorded on MFL); earlier seasons are context only.
- Small samples are shown with their uncertainty, never as findings.
- The archive has no owner names: a franchise is a slot, not a person.

## Evidence on Hand
- `data/mfl.db`: the facts database; `data/lab.json`: the compiled Lab data.
- Measured league patterns (2026-10-06, `~/fleet/runs/mfl-store-2026-10-05/league-view/measure.py`):
  next-draft 1sts are the dearest wins in the league in both halves of the history; linebackers
  and defensive linemen the cheapest; a 1st a year before its draft costs less than half its
  draft-year price; the in-season discount on players narrowed after 2021; contenders and
  rebuilders trade with each other at even value.

## Product Principles
- Lead with the finding in a plain sentence; the number and its range follow.
- Every element answers a question that changes a trade or draft decision.
- A pattern is shown as real only when it clears chance and holds in both halves of the history.
- The league, not one team.
- Numbers come from checked MFL facts; derived figures are named as such.
