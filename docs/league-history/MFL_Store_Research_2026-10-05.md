# What shape the MFL data should take for accuracy — research, 2026-10-05

Christopher's brief: "build a vector database of the MFL data, only the MFL data... before
starting, research the best shape for it for max accuracy." He has seen the Lab's data get less
accurate round after round, and suspects that is why the work keeps coming back as slop.

## 1. Bottom line

- **A vector database on its own would make the numbers less accurate, not more.** A vector
  database finds records that *mean something similar* to a question. Almost everything in this
  archive is numbers, IDs and dates, and vector search is weak on exactly those (section 4).
  Asking it "how many trades did the Cardinals make in 2021" returns the 20 most similar-looking
  records, not all of them, and it cannot add them up.
- **The accuracy problem is upstream of any database.** The Lab has never had a single checked
  layer of facts. Each round rebuilt a compiler that went straight from MFL's raw files to screen-
  ready numbers. Definitions drifted, nothing was checked against MFL's own totals, and outside
  data got mixed in (section 3).
- **The shape that works is a fact store first, with a meaning index on top.**
  - **The fact store** is one SQLite database holding MFL's records as MFL states them, one row per
    fact, each row traceable to the file it came from. Every number anyone quotes comes from here,
    by query. SQLite is built into Python and needs nothing new installed.
  - **The checks** compare every total against MFL's own figures. A mismatch stops the build.
  - **The meaning index** is the vector part. Every trade, contract, player season and franchise
    season gets a plain-English card, searchable both by meaning and by exact words. It
    **finds** things; it never **supplies** numbers. A hit hands back row IDs, and the numbers
    come from the fact store.
  - **Estimates** (wins above replacement, forecasts, market prices) sit in their own tables,
    marked as estimates, and never mix with facts.

This matches what the research says works (section 4) and what failed in rounds 1–4 (section 3).

## 2. What the archive actually holds

96 MB at `league-archive/raw/`, 2013–2026, 1,735 requests logged in `manifest.jsonl`.

| Kind | Where | Size | Notes |
|---|---|---|---|
| Transactions | `transactions.json` per season | 18,220 | 7,393 free-agent adds, 4,195 waivers, 2,634 trades, 470 proposals, 156 rejections, 156 revokes, 150 accepts, IR/taxi moves |
| Weekly rosters | `weekly/rosters_Wnn.json` | 17 weeks × 14 seasons | Barely change within a season before 2018 (MFL storage) |
| Weekly player scores | `weekly/playerScores_Wnn.json` | 17 × 14 | |
| Weekly projections | `weekly/projectedScores_Wnn.json` | 17 × 14 | MFL's own projections: unused so far |
| Standings | `leagueStandings.json` | 448 team-seasons | MFL's W-L, points for, **all-play record, potential points, lineup efficiency** |
| Rookie draft | `draftResults.json` | 2,104 picks | 2013 is the startup draft (1,696); **2014–2016 are empty at MFL** |
| Contracts | `salaries.json` | from 2016 (full from 2019) | salary, years, status, free-text contract notes |
| Cap charges | `salaryAdjustments.json` | 15,216 | Dead money, buyouts; descriptions are templated text |
| Pick ownership | `futureDraftPicks.json`, `assets.json` | per season | |
| Trade bait | `tradeBait.json` | 6 with text | |
| Players | `players.json` | ~2,700 per season | **Name, position, NFL team only: no birthdate or NFL draft slot** |
| Message board | `messageBoard.json` | **empty in every season** | |

**Free text is rare:** 329 transaction notes, 873 draft-pick notes (mostly "Pick traded from…"),
240 contract notes, and the templated cap descriptions. Everything else is structured.

## 3. Where the inaccuracy comes from (measured today)

1. **Outside data in the core.** Every player age in the Lab comes from the DynastyProcess list,
   not MFL. Age bands drive the forecast and every market class, so a non-MFL source sits under
   most of the findings. MFL has its own birthdates and NFL draft slots in `players&DETAILS=1`.
   The archive script already knows that request (`archive.py players`) but it was never run;
   `players_DETAILS1.json` exists for no season.
2. **Same name, different number.** The Lab's "points for" disagrees with MFL's standings in
   **416 of 448** team-seasons, and its all-play rate in **93**. Neither is a bug. The Lab counts the
   regular season (13 weeks) and MFL counts all 17 (Packers 2024: Lab 3,530 against MFL 4,702;
   all-play .876 against .890). But nothing on screen says so, and a GM checking against MFL sees
   wrong numbers.
3. **Recorded facts left on the floor.** The Lab treats trades before 2017 as "picks not recorded"
   and does not value them. But **197 trades in 2014–2016 carry notes recording the picks and cap
   that moved**: "Broncos also add 2015 1st KC", "4.5 m to den", "Cowboys also receive 2016 NYG 3rd,
   2016 NYG 4th...". Those are MFL records of what happened; they just need reading.
4. **Estimates presented next to facts.** A trade card shows "0.84 expected" beside a date and a
   player name, in the same type. Expected wins sit on a forecast, which sits on wins above
   replacement, which sits on a replacement level: three model layers. When a layer changed
   between rounds, every number above it moved, which reads as the data getting less accurate.
5. **No fixed layer of facts.** `lab.json` is built straight for the screens, in columns that
   change each round (round 4 dropped the moves, team-weeks and player-season tables outright).
   There is nothing stable to check against, so every round re-derived the basics and could
   re-break them.
6. **Unused MFL truth.** MFL's own all-play record, potential points (the best lineup a team could
   have started) and efficiency are in the standings file. The Lab computed its own versions instead
   of using MFL's.

## 4. What the research says

- **Embeddings are poor with numbers.** Thirteen widely used embedding models "generally struggle
  to capture numerical details accurately". Their example: "grew by 2%" and "grew by 20%" land
  close together ([Revealing the Numeracy Gap, 2025](https://arxiv.org/abs/2509.05691); earlier,
  [Wallace et al., ACL 2019](https://aclanthology.org/P19-1329/)). In this league, a $2M contract
  and a $20M contract would look alike.
- **Even the best retrieval misses.** Anthropic's own tests on text: embeddings alone missed the
  right passage 5.7% of the time; adding exact-word search (BM25) plus context brought that to 2.9%,
  and reranking to 1.9%. Exact-word search matters for identifiers (their example is error code
  "TS-999"; here it is player IDs, `FP_0025_2026_1`, "2015 1st KC")
  ([Contextual Retrieval](https://www.anthropic.com/news/contextual-retrieval)). A database query
  misses 0% of matching rows.
- **For tables, query the tables.** Structured retrieval-augmented generation keeps tables in a
  database and lets the model write queries. A vector store is a companion for fuzzy questions,
  chosen per question ([LlamaIndex SQLAutoVectorQueryEngine, via the TabEmbed survey](https://arxiv.org/pdf/2605.04962);
  [ICE/NYSE structured RAG: 96% execution match](https://www.zenml.io/llmops-database/text-to-sql-system-with-structured-rag-and-comprehensive-evaluation)).
  On mixed text and tables, hybrid search (exact words plus meaning) did best
  ([T²-RAGBench](https://arxiv.org/abs/2506.12071)).
- **A layer of meaning on top of the tables triples accuracy.** GPT-4 answered questions on a raw
  enterprise SQL schema correctly 16% of the time, and 54% when the same data carried a defined
  meaning layer (what each entity is and how they relate)
  ([data.world benchmark, Sequeda et al. 2023](https://arxiv.org/abs/2311.07509v1)). For us, that
  means named views like `trade_asset` and `team_season_official` with documented definitions,
  not raw MFL columns.
- **History needs "as of" on every fact.** Temporal knowledge stores record when each fact became
  true and when it stopped, and never delete. That is how they answer "what did the Cardinals'
  roster look like on the deadline in 2021" correctly
  ([Graphiti/Zep temporal model](https://blog.getzep.com/beyond-static-knowledge-graphs/)). Those
  that model events in order gain up to 18 points of accuracy on time questions
  ([DyG-RAG and related, 2025](https://arxiv.org/html/2510.16715v1)).
- **Decisions need checked models, not retrieval.** For a high-stakes decision task, plain
  retrieval and query-augmented LLMs were "comparably miscalibrated"; trained, calibrated models
  were not ([Deterministic Decisions for High-Stakes AI, 2026](https://arxiv.org/abs/2606.29280)).
  The Lab's market and draft findings belong in tested estimate tables, not in search results.
- **It can all run on the Beelink, free.** SQLite's built-in full-text search (FTS5) plus the
  `sqlite-vec` extension gives exact-word and meaning search in the same file, merged by rank
  ([Simon Willison, hybrid search with SQLite](https://simonwillison.net/2024/Oct/4/hybrid-full-text-search-and-vector-search-with-sqlite/)).
  Small open embedding models run on a CPU: EmbeddingGemma (308M), nomic-embed-text (137M) and
  Qwen3-Embedding-0.6B ([2026 roundup](https://d-central.tech/local-embedding-models/)). The Beelink
  has 16 cores and about 9 GB free memory, no usable GPU. About 50,000 short cards embed in well
  under an hour, once.

## 5. The recommended shape

```
league-archive/raw/            MFL's files, never edited                (exists)
        │  load: no arithmetic, no outside data
        ▼
mfl.db  FACTS                  one row per MFL fact, MFL's values verbatim,
        │                      each row tagged with its source file
        │  checks against MFL's own totals: a mismatch stops the build
        ▼
        VIEWS                  named, documented: what a "trade asset" or
        │                      "team season" is, with definitions in plain words
        ├─▶ CARDS + INDEX      one plain-English card per trade, contract,
        │                      player season, franchise season, draft pick;
        │                      found by exact words and by meaning;
        │                      returns row IDs, never numbers
        └─▶ ESTIMATES          wins above replacement, forecasts, prices:
                               separate tables, versioned, each with its check
```

**Fact tables, with MFL's own names and IDs:**
- `franchise_season`: name, division, and MFL's standings verbatim (W-L, points for, all-play,
  potential points, efficiency).
- `player`: from MFL's detailed list (birthdate, NFL draft year and slot, college).
- `player_week`: score, MFL projection, injury status.
- `roster_week`: who was on which team, started, benched, on IR or taxi, by week.
- `matchup_week`: each game's scores.
- `transaction` and `transaction_asset`: every trade, add, drop, waiver bid, IR and taxi move,
  proposal, rejection, revoke and accept. One row per asset that moved, with a timestamp.
- `pick`: every draft pick from creation to use, with each change of owner and who it became.
- `contract_year`: salary, years left and status per player per season, plus MFL's note text.
- `cap_charge`: the salary adjustments.
- `trade_note`: the free-text notes, plus what was read out of them (section 6, step 3) and how
  sure the reading is.

**Every row records** its source file, the MFL timestamp, and the date it became true (and stopped
being true, where that applies). That lets any question be asked "as of" a date.

**Cards** are written by code from the facts, never by a model: "2021-10-30, deadline week:
Cardinals sent Pick 2022 1st (own) and DeAndre Hopkins to Rams for ..." Each card carries its row
IDs. This is the vector database. It is where "find me trades like this one" or "every time
someone dumped cap before the deadline" gets answered, and the numbers are then re-read from the
facts.

## 6. How accuracy gets proved, in build order

1. **Pull MFL's detailed player list** (`archive.py players`: 14 requests, one per season; MFL asks
   for at most one player-database pull a day). Then drop DynastyProcess entirely.
2. **Load the fact tables** with no arithmetic. Test: counts per season equal the counts in the raw
   files, file by file.
3. **Read the 2014–2016 trade notes.** First by fixed patterns ("2015 1st KC", "4.5 m to den"), then
   every note that does not match goes on a review list for Christopher. Nothing is guessed.
4. **Reconcile against MFL's own totals**, which stop the build when they fail:
   - W-L and points for equal MFL's standings for all 448 team-seasons.
   - Week-to-week roster changes are explained by recorded moves (100% from 2018, as the Lab
     already proved).
   - Each season's cap total equals MFL's salary column.
   - Every pick's chain of owners ends at the team that used it.
5. **Write the views and their definitions,** then the cards and the index, then rebuild the
   estimates on top. The Lab's screens then read from this, not from a private compiler.
6. **A question test set:** 50 questions with answers checked by hand against MFL's website
   ("Cardinals' record in 2019", "who held pick 2023 1.04 on draft day", "every trade with a
   linebacker under 25"). The store must answer all 50 exactly before the screens are rebuilt.

## 7. What is new to install

Nothing for steps 1–4: SQLite 3.46 and Python are already here. For the meaning index: the
`sqlite-vec` extension and one small embedding model, run on the CPU in a private Python
environment inside the project. Both are free and offline.

## 8. What the build found (2026-10-05, same day)

Built as section 5 describes: `tools/league-history/store/`, database `data/mfl.db`.

**Checked against MFL's own figures** (`store/checks.py`, results in `data/review.md`):

| Check | Compared | Agree | Explained | For review |
|---|---|---|---|---|
| Win-loss records match MFL standings | 448 | 432 | 12 (2013 counted playoffs) | 4 |
| Points for match MFL standings | 448 | 448 | | |
| Potential points match MFL standings | 160 | 160 | | |
| All-play records match MFL standings | 160 | 160 | | |
| Team scores equal their starters | 7,168 | 6,657 | 505 (3-point home bonus) | 6 |
| Lineup scores equal MFL player scores | 202,954 | 202,954 | | |
| Roster changes are recorded moves (2018 on) | 4,814 | 4,811 | | 3 |
| Draft picks were used by their last owner (2017 on) | 1,692 | 1,643 | | 49 |
| Rostered players have an MFL birthdate | 5,223 | 5,167 | | 56 |

**Checked against MFL's web pages** (`store/factexam.py`): 2,291 of 2,295 answers (records,
points, every draft pick's team, player and day, birthdates, NFL draft slots) match what MFL
shows. The 4 that differ are the four records MFL's standings count one game more than was
played; they are on the review list as possible commissioner adjustments.

**What changed in the data because of it:**
- Ages now come from MFL's own player lists (pulled 2026-10-05); no outside data remains.
- Dates are US Eastern, as MFL shows them. They had been UTC, a day late for evening events.
- Season-by-season player names (MFL renamed some players) and every alias on the cards.
- All 260 trade notes read into structured transfers (`store/note_readings.csv`): 423 clear
  items, 17 worked out from other trades, 22 ambiguous, 7 unreadable. None is confirmed until
  Christopher confirms it.
- The trade-offer history covers only the Cardinals' own offers (MFL's visibility rule).
- 2013–2015 pick ownership, calendar, message board and polls are not in the archive, and will
  stay out. MFL serves them only to an owner of those seasons' leagues (51719, 47710, 21225), and
  Christopher's login holds no leagues before 2016 (checked 2026-10-05; his cookie works for
  2016 on and was deleted after the check). His call: leave it. Pick trades from those seasons
  rest on the trade-note readings.
- Christopher does not know the cause of the four record differences or the 2019 week 6–8 point
  penalties; they stay on the review list as "differs from MFL, cause unknown".

**Meaning search, chosen by exam (2026-10-06)** (`store/exam.py`, 135 one-answer questions written
from the facts, plus 8 many-answer questions): right card first / in the top 10.

| Method | First | Top 10 | Many-answer (of 80) |
|---|---|---|---|
| Exact words only | 97 | 130 | 38 |
| Words + bge-base-en-v1.5 | **116** | **135** | **48** |
| Words + bge-small-en-v1.5 | 107 | 135 | 39 |
| Words + nomic-embed-text-v1.5 (quantised) | 90 | 121 | 36 |

bge-base is kept as `card_vec`; the other two were dropped. Search with no filter uses the index's
own nearest-neighbour lookup (same distances, about 16 times faster). Questions that are really
filters (trades in week 8 or 9, quarterbacks traded for a 1st) score poorly by any search and
belong to SQL over the facts.

**The Lab moved onto the facts database (2026-10-06).** `compile/build_lab.py` now reads only
`data/mfl.db`. Same 2,634 trades, 10,316 traded assets, 1,689 draft picks and 448 team seasons as
before; days are Eastern, 2013's regular season is 13 weeks (was 17), ages come from MFL's
birthdates (13 traded players' ages moved by more than 0.2 years, up to 2). Trade values moved by
at most 0.1 win and market prices by under 1%. All Lab tests, the engine test and the screen and
click-through tests pass; "How it's measured" now shows the store's checks against MFL.
