<div align="center">

# TheWarRoom

**A front office for one dynasty league.**

TheWarRoom values every player in a 32-team dynasty IDP salary-cap league on MyFantasyLeague,
using two numbers fitted to the league's own history, and ranks the franchises on them.
It runs on your machine and shows its work.

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Wails v2](https://img.shields.io/badge/Wails-v2-DF0000)](https://wails.io)
[![React](https://img.shields.io/badge/React-20232A?logo=react&logoColor=61DAFB)](https://react.dev)
[![SQLite](https://img.shields.io/badge/SQLite-WAL-003B57?logo=sqlite&logoColor=white)](https://sqlite.org)
[![License](https://img.shields.io/badge/License-PolyForm%20Noncommercial-4B5563)](LICENSE)

</div>

![M2 power rankings, the franchise view](docs/ui/showcase/app-m2-franchise.png)

<sub>M2, the franchise view: each roster's summed dynasty value, ranked, with each team's move since the previous model run.</sub>

## The problem

A serious dynasty league outgrows the platform it runs on. MyFantasyLeague keeps the score,
the rosters and the contracts. It cannot tell you what a player is worth: this season, over
the next three, against a salary cap, under this league's own scoring, where defenders score
alongside the offense.

The usual answers are a spreadsheet or someone else's rankings, built for someone else's
league. TheWarRoom builds the answer from this league's data and keeps every step visible.

## Two numbers per player

Every rostered player gets two values, both in this league's fantasy points per game:

| | What it answers |
|---|---|
| **Now** | What he scores per game he plays, this season. |
| **Dyn** | What he is worth over this season and the next two, each later season counting 0.75 of the one before, weighted by his chance of being on the field. |

Both come from one rule, the **credibility blend**: trust a player's production in proportion
to how much of it there is.

```
value = Z · production + (1 − Z) · prior        Z = games / (games + k)
```

- **Production** is his league points per game, as a percentile among the regulars at his
  position, with recent seasons weighted more.
- **The prior** is what his pre-NFL facts predict: draft slot, combine, college production,
  age at entry. It never sees fantasy points, so it is an independent second opinion, not
  production counted twice.
- **k** sets how many games it takes before production outweighs the prior. It is fitted per
  position, because a kicker's season says far less about his next than a quarterback's does.
- **Age** enters once, through a fitted career arc per position. A 31-year-old running back
  and a 31-year-old quarterback with the same numbers are valued apart.
- **Rookies** are never zero. Until he plays, a rookie is valued on his prior and on the
  chance a player drafted where he was takes the field.

## Fitted, then checked on a season the fit never saw

The model's 400 values are fitted on the league's own scoring history, 2021–2025. Each
optional piece (the career arc, the recency weights, the survival model, the rookie debut
chance) is kept only if it beats its simpler alternative at predicting the 2025 season with
2025 held out. The fit is reproducible: three runs give byte-identical output.

Against the rankings the app already had (last season's points, adjusted for age), predicting
2025 from data through 2024:

- **Talent:** the model ranks the next season's players better at 9 of 10 positions
  (rank correlation at RB 0.76 against 0.53, at TE 0.80 against 0.56).
- **Rookies:** the model ranks the rookies who became regulars (0.41–0.86 by position). The
  old rankings score every rookie zero.
- **Kickers:** neither ranks them well. In this league's scoring a kicker's season barely
  predicts his next, and the report says so.

Every setting the engine reads, 509 of them with the fitted values and scouting weights, is
listed in the Engine Admin console, marked fitted or hand-set, and editable. A changed setting writes a new run, and
every earlier run stays readable.

![Engine Admin, the fitted WR values](docs/ui/showcase/app-admin-fitted.png)

## What it does today

- **M1, asset rankings.** Every rostered player, with Now and Dyn beside the app's original
  score, filterable by position and team, with each player's move since the last run.
- **M2, power rankings.** Two views. **This season** blends each roster's Now with the
  season's results (MFL's all-play record, or points for when the league reports none).
  **The franchise** ranks on roster Dyn alone.
- **Signals.** Each data feed with its coverage by season and position and its freshness.
  A lost source never stops the board: it runs on what still flows, and says so.
- **What-if transactions.** Trades, cuts, tags, extensions, restructures, buyouts and
  signings under the league's contract rules, with the cap computed to the cent before
  anything commits. MyFantasyLeague stays the record: a move made here is a plan until it is
  made there.

![M1 asset rankings with Now and Dyn](docs/ui/showcase/app-m1-assets.png)

## The data

Four free sources, read into one history:

| Source | What it supplies |
|---|---|
| [MyFantasyLeague](https://www.myfantasyleague.com) | The league: rosters, contracts, standings and its scoring of every player, each season since 2021 |
| [nflverse](https://github.com/nflverse/nflverse-data) | Weekly stats, snaps, injuries, the draft, the combine and advanced defensive stats |
| [DynastyProcess](https://github.com/dynastyprocess/data) | The player id crosswalk that joins the sources (124,906 links) |
| [CollegeFootballData](https://collegefootballdata.com) | College production and school tier |

History is one table of `player · season · week · measure`: about a million values from 2021
on, under 105 measures named for what they mean, never for the source that supplied them.
The model reads measures, so replacing a source changes no model code.

## How it is built

- **Local-first.** A Go engine with a desktop shell (Wails, React) on SQLite. The league's data
  stays on your machine.
- **A pure engine.** The scoring and model code touches no database, network or file, and a
  linter fails the build if it tries. Everything it scores arrives as input, the date included.
- **Append-only history.** Every run records the settings, inputs and engine version it used,
  and database triggers reject any edit to a past run.
- **Money in whole cents.** Salaries and cap figures are integers, so the cap never drifts.
- **Gated on real output.** Each stage of the build plan ends at a gate, run live on a
  separate virtual machine against a snapshot of the real database, with screenshots as
  evidence. The plan, every design decision and every gate record are in
  [`docs/build-handoffs/Core_Build_Plan_2026-10.md`](docs/build-handoffs/Core_Build_Plan_2026-10.md).

The system map is [`SYSTEM_MAP.md`](SYSTEM_MAP.md).

## Status

The core build (October 2026) is done: the data history, the fitted model, both numbers on
every player, and power rankings on them. It has one user, a GM in the league, and is not used
for in-season decisions before 2027. The known limits, on the record:

- In-season, Now leans a little too heavily on the current season's games.
- A rookie who has not played yet keeps his draft-day chance of taking the field.
- Kickers, as above.

Where it is going, beyond the core: [`docs/roadmap/Vision_2026.md`](docs/roadmap/Vision_2026.md).

## Running it

TheWarRoom is built for one league: the league id is `LeagueID` in
`internal/ingestion/schema.go`. To build it from source you need Go 1.26, Node with pnpm, and
the [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation).

```sh
make setup     # once per clone: the commit and push hooks
make build     # build/bin/thewarroom
make verify    # lint, race-enabled tests and the frontend build
```

Set `CFBD_API_KEY` to a free [CollegeFootballData](https://collegefootballdata.com/key) key for
the school-tier signal; without it, that one signal is skipped.

## Who builds it

Christopher Campbell sets the direction and makes every product call. Claude (Anthropic)
writes the code and makes the engineering calls, each one explained so Christopher can veto
it.

## License

**TheWarRoom is source-available, not open source: free to run and tinker with, never to
sell.**

- **Free for non-commercial use** under the
  [PolyForm Noncommercial License 1.0.0](LICENSE). Run it, fork it, self-host it, study it,
  share it.
- **Owned in full by SecureProspective LLC** (Texas). All rights reserved.
- **Tech Freedom Ministries holds a perpetual, irrevocable, free-use grant**, commercial use
  included, that survives any change of ownership. TFM is the inspiration for this project.
  See [`docs/licensing/TFM-Grant.md`](docs/licensing/TFM-Grant.md).
- **Commercial use and sale are reserved to SecureProspective.**
  [Contact SecureProspective](https://secureprospective.com) for terms.
- **Contributions are welcome** under the [Contributor License Agreement](CLA.md), which keeps
  ownership with SecureProspective while you keep the copyright to your own work. Start with
  [`CONTRIBUTING.md`](CONTRIBUTING.md).

The licensing picture in plain English:
[`docs/licensing/README-license-summary.md`](docs/licensing/README-license-summary.md).
