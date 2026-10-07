# Legacy NFL Lab

Fourteen seasons of Legacy NFL trades and drafts, rebuilt as a dated ledger and valued in one
currency (wins above replacement). It answers decision questions:

- where the league misprices kinds of assets;
- when each position is worth drafting, and whether to draft it or buy it;
- what every trade was worth on the day and what it delivered since.

Every answer carries its uncertainty and a check against what actually happened. It exists to
inform how TheWarRoom is built; it is not part of TheWarRoom.

**Since 2026-10-06 the Lab reads only the facts database below.** `compile/build_lab.py` builds
`data/lab.json` from `data/mfl.db`: no outside player data, MFL's own birthdates, US Eastern days.

## The facts database (`store/`)

One SQLite file, `data/mfl.db`, built only from the MFL archive:

- **facts:** MFL's records as MFL states them, one row per fact, each row traceable to its file;
- **checks:** every total proved against MFL's own figures, with each difference explained or
  listed for review;
- **cards:** one plain-English card per trade, offer, draft pick, player season, team season and
  season, found by exact words and by meaning. A card hands back the rows it came from; numbers
  always come from the facts.

```sh
python3 store/build.py                      # facts, checks, note readings, cards (about 20 seconds)
.venv/bin/python store/index.py             # meaning search, bge-base (about 30 minutes on the CPU, once per build)
.venv/bin/python store/ask.py find "trades where a team sent cap space" --year 2014-2016
.venv/bin/python store/ask.py sql "SELECT * FROM team_season WHERE fid = '0025'"
python3 store/factexam.py                   # answers checked against MFL's own web pages
.venv/bin/python store/exam.py              # does search find the right card?
python3 -m unittest store/test_store.py     # about 100 seconds
```

`.venv/` is the project's private Python environment (`sqlite-vec`, `fastembed`); the facts,
checks and cards need nothing beyond Python. `data/review.md` lists everything waiting for
Christopher's judgement. The research behind this shape is in
`docs/league-history/MFL_Store_Research_2026-10-05.md`.

| To change… | Edit |
|---|---|
| A fact table | `store/schema.sql` and the matching step in `store/load.py` |
| What a term means (regular season, all-play, team season, trade asset) | `store/views.sql` |
| A check against MFL's figures | `store/checks.py` |
| How a trade note was read | `store/note_readings.csv` (then rebuild) |
| What a card says | `store/cards.py` |
| The search (word and meaning ranking, filters) | `store/ask.py`; the model in `store/index.py` |

Open the Lab on the Beelink at **http://localhost:8765/**. It installs as an app from the browser's
address bar and works offline once it has loaded once.

The method, with sources, is in
`docs/league-history/League_Lab_Method_Research_2026-10-05.md`.

## Running it

From this folder (`tools/league-history`):

```sh
python3 store/build.py            # first: the facts database, after the archive is refreshed (about 20 seconds)
python3 compile/build_lab.py      # then the Lab data from it (about 40 seconds; the backtest is most of it)
python3 serve.py                  # serve on http://localhost:8765/ (loopback only)
```

The server only answers on this machine. It only serves the app and the compiled data, never
the raw archive.

## Checks

```sh
python3 -m unittest compile/test_build_lab.py         # data, valuation and ledger checks (about 75 seconds)
node --test tests/engine.test.mjs                     # the browser engine against the compiler
node tests/screens.mjs http://127.0.0.1:8765/ <dir>   # every screen renders, no errors, screenshots
node tests/interact.mjs http://127.0.0.1:8765/ <dir>  # clicks, filters, drawers, no team lens, offline
```

The two browser checks use Brave headless and the `playwright-core` copy pnpm has cached on
this machine (or `$PLAYWRIGHT_CORE`).

## What is where

| To change… | Edit |
|---|---|
| Wins above replacement, the replacement level, all-play, the currency check | `Wins` in `compile/value.py` |
| The forecast (features, folds, calibration, point-in-time fits) | `Forecast` in `compile/value.py` |
| Pick values by slot (local-linear smoothing, each pick judged without its own class) | `Picks`, `local_linear` in `compile/value.py` |
| Valuing an asset on a given day, contract cost, the five-season frame | `Valuer`, `contract_cost` in `compile/value.py` |
| The market model (price per expected win, anchor, discount, ridge) | `fit_prices`, `market`, `DISCOUNT`, `ANCHOR` in `compile/value.py`; mirrored in `app/js/engine.js` |
| The backtest ("would it have worked?") and the delivered records | `backtest` in `compile/value.py` |
| The proof that the move history is complete | `compile/ledger.py` |
| The buy and sell lists, the going rate, the calendar, small samples, draft judging | `app/js/engine.js` |
| Any wording a GM reads: positions, age bands, times of year, draft bands | `app/js/model.js` |
| A screen | `app/js/views/<screen>.js` (market, draft, tradelog, method) |
| Top bar (seasons), routing | `app/js/main.js` |
| Drawer, trade cards, range bars | `app/js/ui.js` |
| Phases, standings at the time, contracts, pick slots | `compile/build_lab.py` |

## Things the facts database taught us (built in and checked)

- MFL shows every date in US Eastern time; a date taken from MFL's timestamps in UTC can be a
  day late (C.J. Stroud went 1.04 on June 8, 2023 at 9:27 p.m. Eastern).
- MFL's standings count points over every week it scored, playoffs included. Win-loss is the
  regular season, except in 2013, when the playoff games were counted too.
- The regular season is the weeks in which every team had an opponent: weeks 1-12 through 2020,
  1-13 from 2021. MFL's own "last regular week" setting is wrong for past seasons.
- All-play counts every week in which all teams scored, playoffs included; it then matches MFL
  exactly.
- Home teams got 3 extra points in 2014-2015 regular-season games and in playoff games.
- MFL lists a player on IR or taxi twice in a weekly roster (once as ROSTER).
- MFL repeats one team's 2018 games as standalone copies, and uses a "BYE" stand-in team in 2013-14.
- Trade offers in the archive are only the Cardinals': MFL shows offers to the owners involved,
  and the archive was pulled with the Cardinals' login.
- Some players' names change between seasons (Darius/Shaquille Leonard); cards carry every name.
- Picks and cap money in 2014-2016 trades are only in the notes. A pick that moved in a trade
  without a note is not recorded anywhere.
- Rebuilding next to a stale write-ahead log damages the database; the build now refuses while
  anything is using it.

## Data sources

- `league-archive/raw/` (read-only): the 2013–2026 MFL pull.
- `data/mfl.db`: the facts database built from it (`store/`); the Lab compiler reads only this.
  MFL's own detailed player lists (`players_DETAILS1.json`, pulled 2026-10-05) give birthdates
  and NFL draft slots. The DynastyProcess list (`data/db_playerids.csv`) is no longer read.

`data/` is private and not in git.

## Things the data taught us (built into the compiler and checked by tests)

**Archive quirks**
- MFL's current-year pick codes count from zero: `DP_3_20` is round 4, pick 21.
- Picks are only recorded in trades from 2017; contracts are fully recorded from 2019.
- 361 trade records have only one side. They are not valued.
- MFL marks a skipped or forfeited rookie pick with the player `----`.
- Each season's salary file lists every earlier cut that still charges that season's cap.
- MFL's stored weekly rosters before 2018 barely change within a season, so the ledger proof
  covers 2018 on. From then, 100% of week-to-week team changes are explained by recorded moves.

**The currency**
- Wins above replacement from started players track all-play win rate at 0.97 across 384
  team-seasons. All-play records strip out about ±2.4 wins a season of schedule luck.

**The forecast**
- A straight-line forecast overrates older players: they delivered 76–90% of it, because
  retirements count as zero. Calibrating by position group and age band fixes this on average.
- For an in-season forecast, the last full forecast is worth about 6 weeks of new evidence
  (tested).

**Pick values**
- A plain kernel average underrates the top draft slots, because nothing comes before pick 1;
  every early 1st then looks like a win. Local-linear smoothing removes that.

**Trades**
- The league's trades balance best when each later season counts 0.75 of the one before.
  Salary barely moves trade prices: $1M a season is worth 0.007 wins.
- On single trades the model is a weak guide. In the point-in-time backtest the favoured side
  came out ahead about half the time, and lopsided-looking trades delivered about 10% of their
  gap. The edges are in kinds of asset across many trades.
- Against the going rate (what the median traded asset costs per delivered win, 2017–2026), the
  league sells linebackers (39%) and defensive linemen (68%) too cheap, and pays too much for
  next-draft 1sts (294%) and next-draft 3rds and later (207%). All four hold in 2017–21 and in
  2022–26; every other kind is priced about right or too new to judge.
- A 1st a year before its draft trades at about half its draft-year price per expected win
  (45%, then 47%).
- In 2017–21 a 1st bought 2.4 times the player wins in season that it bought in the offseason;
  since 2022 that gap has all but closed (1.2 times).
- Contenders (top 10 at the time) and rebuilders (bottom 10) trade at even value.

**The draft**
- Within a round, position barely changes a pick's return. Across all rounds, defensive linemen
  return about 70% of their slot and tight ends about 80%; kickers return well above theirs.

## History

- **2026-10-05, round 5 (the facts database):** Christopher saw the Lab's numbers drifting and
  asked for a vector database of the MFL data, MFL only. Research showed vectors alone would lose
  accuracy on numbers, so the build is a checked facts database with meaning search on top
  (`docs/league-history/MFL_Store_Research_2026-10-05.md`). The Lab's compiler was then moved
  onto it (2026-10-06): the same trades, picks and team seasons; days now Eastern, 2013's regular
  season 13 weeks, ages from MFL. Trade values moved by at most 0.1 win; market prices by under 1%.
- **2026-10-06, round 9b (offers against acceptance):** Christopher asked for a chart of trade offers
  against acceptance by time of year. MFL shows a franchise only its own offers, so all 470 in the
  archive involve the Cardinals; he chose to build it on those. The compiler now writes `offers`
  (`offer_outcomes` in `compile/build_lab.py`): each proposal matched to its ending (148 accepted,
  138 rejected, 146 withdrawn, 8 expired, 30 not matched and left out of acceptance rates), with its
  stretch of the year and season. The trade-log screen shows offers a season per stretch, stacked by
  ending, with the acceptance rate; sent, received or both; the seasons, time-of-year and team
  filters apply. Measured: about a third accepted, no stretch different beyond chance.
- **2026-10-06, round 9 (how the teams trade):** Christopher: the trade log should "get far deeper
  into behavioral data from the teams ... like trading stocks ... granular on teams, and trends with
  contenders, in the hunt, and rebuilding." The Trade log screen is now "How the teams trade"
  (`app/js/teams.js`): every valued trade split into two team sides, each tagged with the team's tier
  that day (contender 1-10, in the hunt 11-22, rebuilding 23-32) and what it did (bought for now,
  built for later, like for like). Filters: tier, time of year, what it did, team. Tier cards, the
  tiers' habits by season (with an earlier-vs-later-half test), who trades with whom, a sortable
  table of all 32 teams, a team in depth (season by season with its finish, how it trades against
  the league, what it sends and takes by kind, its partners), then the log itself. Measured: no tier
  wins its trades on the day beyond chance; contenders bought for now more in 2022-26 (27%) than
  2017-21 (22%), and built for later more too (24% against 18%), both beyond chance.
- **2026-10-06, round 8 (draft timing rebuilt):** Christopher: "the numbers are so tight its hard to
  really understand where those ineffeciencies are. A timing on the draft and player value is where
  the core focus should be." The screen now measures picks as a percent of their slot (a ratio
  with a range, `app/js/draftlab.js`) instead of ±wins. Filters: seven draft stretches (each bar is
  what a pick there returns), positions, the player's NFL draft round, judging window. Two league
  maps: each position by season after the draft (year 1 to 5) and by draft stretch. Choosing a
  position opens its own draft: where to take it (with "plays like pick N", read off the draft's
  own value curve), when it pays, what decides it (NFL round, age), how often it becomes a starter
  against the same picks overall, and draft it or buy it (the trade market's cost for proven
  players of the position against the round the picks were spent in). Measured: defensive linemen
  pay late (48% of slot in year 1, 100% by year 4) and return 10% of slot from NFL rounds 4-7;
  defensive backs become starters 65% of the time against 50% for the same picks.
- **2026-10-06, round 7 (filters on the market):** Christopher called the buy and sell lists the
  best part and asked for more control over time of year and assets, with the calendar blended
  into the filters. The month chart became the time-of-year filter: the league year in eight
  stretches set by each season's own dates (NFL draft, rookie draft, kickoff, week 4, deadline),
  each showing trades a season. Asset chips cover positions, pick round and draft, with players
  together or split by age. The lists, the pick clock and the contenders line reprice for the
  choice, against a going rate taken from everything traded in that stretch. The 1st-for-players
  rates table was retired; the same question is now answered by narrowing the time of year.
  Rows too uncertain to judge on delivery show what GMs paid instead.
  Same day, round 7b, after Christopher: "the filter should filter the data field, not just the
  locked paths". Choosing an asset now opens that asset's own market above the league lists:
  why (what GMs pay per forecast win against the typical asset, what share of the forecast it
  delivers, who takes it on), when it is cheapest (all eight stretches, marked only beyond
  chance, each clickable to its trades), against everything else (cost per delivered win as a
  multiple of every other kind, ranges from the fit's full covariance), and what came back for it.
  The league-year counts follow the chosen assets.
- **2026-10-06, round 6 (the league view):** Christopher: still hard to read, and he needs to see
  the league, not his own team. The team selector and every Cardinals highlight are gone. The
  market screen is now a buy list and a sell list (each kind of asset's cost per delivered win
  against the going rate, shown only when it clears chance) and a calendar band: trades by
  month, what a 1st buys in players by time of year, the pick clock, contenders against
  rebuilders. Layout chosen by Christopher from three on the Impeccable decision page.
  Meaning search uses bge-base-en-v1.5, chosen by the search exam (2026-10-06): combined with exact
  words it put the right card first for 116 of 135 questions and in the top 10 for all 135, against
  97 and 130 for exact words alone; bge-small reached 107/135, nomic 90/121. Questions about
  structure (deadline trades, quarterbacks traded for a 1st) are for `ask.py sql`, not search.
- **2026-10-05, round 4 (method rebuild):** Christopher judged rounds 1–3 to look good but not
  help him act. The app was rebuilt on the method research:
  - a dated ledger and one currency;
  - point-in-time valuation, a market model and a backtest;
  - shrinkage and funnel limits on every small sample.

  League year, Cycles, Cap & contracts, Franchises, the clock and the old Market and Rookie
  draft screens were retired. The round 2–3 build is in
  `~/fleet/runs/league-lab-2026-10-05/snapshot-before-method/`.
- **Round 2:** the League year and a shared league clock. The pre-change build is in
  `snapshot-before-refocus/`.
- **Round 1:** replaced the seven-view explorer Bee built (recoverable from commit `94bcf88`).
