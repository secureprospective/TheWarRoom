# RESUME — Legacy NFL Lab (2026-10-06, rounds 5-9b: facts database, league view, market filters, draft timing, how the teams trade, offers)

## 1. What we are doing
- **Round 5 brief (Christopher):** the Lab's data was "getting less and less accurate"; build a
  vector database of the MFL data, MFL only; research the best shape for accuracy first.
- **Research verdict** (`docs/league-history/MFL_Store_Research_2026-10-05.md`): vectors alone are
  weak on numbers. Build a checked facts database with meaning search on top. Christopher
  approved this shape, an MFL player-detail pull plus "everything else we should have", a review
  list for unreadable trade notes, and freezing the Lab's screens until they are rebuilt on the
  store.
- **Where:** repo `~/work/TheWarRoom`, branch `session/league-history-pwa`,
  `tools/league-history/store/`; database `tools/league-history/data/mfl.db` (private, gitignored).
  Christopher's franchise: Arizona Cardinals `0025` (never highlighted: the Lab is league-wide).
- **Round 6 brief (Christopher, 2026-10-06):** "Its still hard to use, hard to read and understand
  ... focus less on the Arizona Cardinals, and more on the broader league behaviours. The
  ineffeciencies need to be market and calendar wide ... I need to SEE the legue, not through the
  lens of my team, I already have that". Done under the Impeccable skill (PRODUCT.md, decision
  page, finish review).

- **Round 7 brief (Christopher, 2026-10-06):** the buy/sell lists are "the best part ... where I
  want more control on filtering the time of year, and the assets in question. Blend [the
  calendar] into the filtering methods". Built: a filter band above the lists (league year in
  eight stretches with trades a season; asset chips; players together or by age). Lists, pick
  clock and contenders reprice; the 1st-for-players rates table is retired.

## 2. Agents + harnesses
Claude, plus Impeccable's shipped subagents for round 6: `impeccable-finish-reviewer` (two rounds,
both "fix", all items fixed) and `impeccable-documenter` (in flight, section 5). Skill launcher:
`~/.claude/skills/impeccable/scripts/impeccable`. Private Python environment `tools/league-history/.venv` (sqlite-vec 0.1.9, fastembed);
models cached in `data/models/`.

## 3. Gates

| Check | Result |
|---|---|
| `python3 store/build.py` (facts, 9 checks, note readings, cards) | all checks pass; ~20 s |
| `python3 -m unittest store/test_store.py` | 18 pass (~100 s) |
| `python3 store/factexam.py` (answers against MFL's web pages) | 2,291 / 2,295; the 4 are review items |
| `python3 store/exam.py --tables` (word search) | right card in top 10 for 130 / 135; first for 97 |
| `.venv/bin/python store/exam.py` (hybrid, bge-base, `card_vec`) | first for 116 / 135, top 10 for 135 / 135 |
| `python3 -m unittest compile/test_build_lab.py` (Lab on the store) | 22 pass |
| `node --test tests/engine.test.mjs` (9); `node tests/screens.mjs`; `node tests/interact.mjs` (7 steps) | all pass on the round-6 build |
| Impeccable detector (`impeccable detect --json`) | one finding (11.5px text), fixed |
| Impeccable finish review | round 1 fix (8 items), round 2 fix (5 items); all applied, third batch not re-scored |
| Round 7: `node --test tests/engine.test.mjs` (10), `tests/interact.mjs` (8 steps incl. filters), `tests/screens.mjs`, compiler 22 | all pass |
| Round 7 detector | advisory only (sizes/colours now recorded in DESIGN.md) |
| Round 7 finish review (Impeccable reviewer) | not run |
| Christopher's review of `data/review.md` | pending |

## 4. Artifacts
- `store/schema.sql`, `load.py` (facts, MFL only; dates US Eastern; season names), `views.sql`
  (regular season, all-play, team_season, trade_asset, offer, roster_move), `checks.py` (9 checks,
  `data/review.md`), `notes.py` + `note_readings.csv` (all 260 trade notes read: 423 clear,
  17 inferred, 22 ambiguous, 7 unreadable; replay check against drafts), `cards.py` (28,123 cards
  plus FTS5), `index.py` (embeddings into vec0 tables), `ask.py` (hybrid find plus read-only sql),
  `exam.py` (search exam), `factexam.py` (MFL page answer key, pages cached in `data/mfl-pages/`),
  `build.py`, `test_store.py`.
- Archive: `league-archive/raw/<year>/players_DETAILS1.json` for every season, plus
  `raw/allRules.json` (pulled 2026-10-05; `archive.py` now fetches allRules in `players` mode).
- Trade-note dump: `~/fleet/runs/mfl-store-2026-10-05/trade_notes_raw.txt`.
- README "facts database" section and history; research doc section 8 (findings).
- Memory: `league-numbers-come-from-checked-mfl-facts`.
- Round 6: `tools/league-history/PRODUCT.md`; surface brief with the direction contract
  `tools/league-history/.impeccable/surfaces/app-index-html.md` (seed key e4f51cb1, form "Buy list,
  sell list"); review captures `.impeccable/review/{desktop,user-1280,mobile,draft-desktop}.png`;
  measurement script `~/fleet/runs/mfl-store-2026-10-05/league-view/measure.py`.
- Round 6 code: `app/js/views/market.js` (rewritten), `app/js/engine.js` (`groupOf`,
  `leagueMarket`, `tradesByMonth`, `calendarRates`, `contenderTrades`; `teamEdges` removed),
  `main.js` (no team selector, offline status settles), `draft.js`/`tradelog.js`/`ui.js` (no team
  lens), `style.css` (`--faint` #7f90a6, market styles), `sw.js` cache `lab-v4`, tests updated.

## 5. In flight
- Nothing. The Impeccable documenter finished at 09:11 on 2026-10-06: `tools/league-history/DESIGN.md`
  (18,849 bytes) and `.impeccable/design.json` (26,403 bytes), so the Impeccable FINISH line is
  discharged. Unused CSS it flagged (`--raised-2`, `.badge`, `.pin-caution`) was removed.
- Earlier work is finished: the model queue (bge-base kept as `card_vec`, database vacuumed to 222 MB);
  the Lab compiler reads only `data/mfl.db`. Baselines and screenshots: `~/fleet/runs/mfl-store-2026-10-05/`.
- Claude-OS is running (left on by standing rule).

## 6. Settled — do not retest
- Points for = every week MFL scored (448/448). Win-loss = regular-season games, except 2013,
  which counted playoffs. Regular season = weeks where every team had an opponent. All-play = every
  week all teams scored (160/160). Home bonus +3.0 (2014-15 regular season, playoffs).
- MFL dates are US Eastern. The pre-2017 pick history in MFL's draft notes only covers trades made
  inside MFL. Offers in the archive are the Cardinals' only.
- Replacing the database beside a stale -wal corrupted it once: `retire()` now prevents it.
- MFL's transactions web page is empty without a login.
- League patterns (2026-10-06, both halves 2017-21 / 2022-26, each half's range must clear the
  going rate): linebackers sell cheap (39%, holds); next-draft 1sts (294%) and next-draft 3rds+
  (207%) are overpaid (hold); defensive linemen 68% overall but weaker lately (67% then 94%),
  so not "holds". Pick clock: a 1st a year out trades at ~45-47% of its draft-year price.
  In-season discount on players vs 1sts: clear in 2017-21 (5.2 vs 2.2 wins), gone since 2022.
  Contenders vs rebuilders: even (-0.02 ± 0.04). Only one time-of-year mark beats both other
  windows: next-draft 1sts fetch the most in season.
- A sell list must mark when the league pays most, not when it is cheapest (reviewer caught it).

## 7. Decisions (Christopher)
- Shape: facts first, vectors on top. MFL data only. Review list for unreadable notes.
- Round 6 (2026-10-06): the league map first, lookups second; league only, with a plain team
  filter kept in the trade log; desktop is the target. Layout: "Buy list, sell list".
- **2013-15 logged-in exports: leave it** (Christopher, after the check). His cookie works for 2016 on;
  his login holds no leagues in 2013-15, so MFL refused all 12. Cookie deleted. Do not ask again.
- He does not know the causes of the review items (4 records, 2019 penalties): they stay listed as
  "cause unknown".

- Round 7 (2026-10-06): the market filters by stretch of the year and asset; picking an asset must
  open that asset's own market (why, when, against what), not hide rows ("filter the data field").
- Round 8: draft timing and player value are the core focus; percent of slot, not ±wins.
- Round 9: the trade log is team behaviour (tiers by rank that day, postures), still league-wide.
- Round 9b: offers chart built on the Cardinals' offers ("Build it on your offers"), the one
  sanctioned exception to the no-team-lens rule, labelled as his offers.
- 2026-10-06 close: next session is planning only (MFL menu tree, then a TheWarRoom build plan).

## 8. Ledger state
Nothing from rounds 4-6 is committed except RESUME documents. Commit only on his go-ahead:
`git add tools/league-history docs/league-history docs/build-handoffs` (`.venv/` and `data/` are
ignored). Never `--no-verify`; no merge to main.

## 9. Next actions
**Christopher, 2026-10-06, closing the build day:** "Good work, this is fine for now: when we come back
from compaction we will examine the MFL menu tree and we will want to apply all of these features to
TheWarRoom, but not doing a build today. It will be a planning session for the build."

**NEXT SESSION = PLANNING ONLY. Write no app code, change no TheWarRoom build.**
1. Examine the MFL menu tree for league 14432: every page and report MFL offers (menus, report
   pages, options), so the plan maps each Lab feature to what MFL already shows and what it lacks.
   Use what is already cached first (`tools/league-history/data/mfl-pages/`, the store's
   `factexam.py` page list, `docs/data-layer/MFL_API_Specification.md`); fetch more pages only
   read-only, via headless Brave or on Claude-OS (no Claude-in-Chrome). MFL players endpoint: at
   most one call a day.
2. Plan applying the Lab's features to TheWarRoom (Go engine, Wails v2, React + Tailwind +
   Zustand, SQLite; see `CLAUDE.md`, `SYSTEM_MAP.md`, `docs/build-handoffs/Core_Build_Plan_2026-10.md`
   and its rulings R1-R12): the market (buy/sell lists, going rate, time-of-year and asset filters,
   per-asset panels), draft timing (percent of slot, stretches, NFL round, age, plays-like-pick,
   draft-or-buy), how the teams trade (tiers, postures, trends, team table, team panel), offers
   against acceptance; and the facts database (`store/`) as the data source. Decide where each
   lives (engine vs frontend), what ports from JS (`app/js/engine.js`, `draftlab.js`, `teams.js`),
   and the gates. Claude makes the architecture calls and explains them; Christopher decides
   product and priority. Write the plan as a document in `docs/build-handoffs/`.
3. Still open, his call: commit rounds 4-9b on `session/league-history-pwa`
   (`git add tools/league-history docs/league-history docs/build-handoffs`). Nothing from rounds
   4-9b is committed except RESUME documents.
4. Optional: finish review (Impeccable reviewer) for rounds 7-9b, never run; `data/review.md`.

### Build history of this session (rounds 7-9b, all 2026-10-06, all tested, Christopher approved each)
**Round 7 (filters) is built and tested; Christopher has not seen it.** Code: `engine.js`
(`leagueMarket` takes `phases` and `byAge`; `inScope`, `MIN_TRADES` 80/60, `tradesByPhase`,
`pickClock`, `paid` on rows; `tradesByMonth`/`calendarRates` removed), `model.js` `PHASES`,
`views/market.js` (filter band, `yearPicker`, `assetPicker`), `style.css` (`.flt`, `.yr-*`,
`.chip`), `sw.js` `lab-v5`. Captures: `~/fleet/runs/mfl-store-2026-10-05/filters/`.
Filter findings worth knowing: in season only LB (cheap) and RB (dear, fading) clear; next-draft
1sts drop to "too uncertain" there because few in-season-traded 1sts have played out (paid 312%).
Offseason adds QB as dear (143%, not in both halves). Winter alone (41 trades) is too few to price.
**Round 7b (same day):** Christopher: "its the same charts without new data ... when is the cheapest,
why, against what, players and other pick ... the filter should filter the data field". Built:
`engine.js` `fitPrices` returns `cov`; `assetOf`, `stretchPrices`, `assetProfile` (why/when/against/
back/buyers); `tradesByPhase` takes `assets`. `market.js` `profile()` panels above the league lists;
chosen rows outlined (`.focus`). `sw.js` `lab-v6`. Tests: engine 11, interact 8 steps (asset panel,
stretch drill-down), screens ok. Captures `filters/p-*.png`. Measured: next-draft 1sts cheapest during
the rookie draft (102%) and dearest rookie draft to kickoff (346%), both beyond chance; linebackers
cheap for both reasons (paid 55%, delivered 155%).
**Round 8 (same day): draft timing rebuilt.** Christopher: "much better" on 7b; then "take the same
kind of granular look and filtering on draft timing ... A timing on the draft and player value is
where the core focus should be." New `app/js/draftlab.js` (SEGMENTS, NFL_ROUNDS, AGE_BANDS,
draftScope, vsSlot, bySeason, slotCurve, playsLike, hitRate, positionDraft, draftMap); `views/draft.js`
rewritten (filter band, two maps, position panels; old grid/lean/pins removed; `priceList` no longer
used by a screen). `sw.js` `lab-v7` + draftlab in SHELL. Tests: engine 12, interact 8 steps (draft step
rewritten), screens ok, compiler OK; detector advisory only. Captures `filters/d-*.png`.
**Round 9 (same day): how the teams trade.** Christopher: "Good work" on round 8, then the trade log
should go "far deeper into behavioral data from the teams ... like trading stocks ... granular on teams,
and trends with contenders, in the hunt, and rebuilding." New `app/js/teams.js` (TIERS, POSTURES,
tradeSides cached on data, sidesInScope, behaviour, tierTrends, tierPairs, teamRows, teamProfile);
`views/tradelog.js` rewritten (title "How the teams trade"). `sw.js` `lab-v9` + teams.js in SHELL.
Tests: engine 13, interact 8 steps (trade-log step rewritten), screens ok, compiler OK, detector
advisory only. Captures `filters/t-*.png`. His own franchise appears as one row among 32 with no
highlight, per the league-only rule.
**Round 9b (same day): offers against acceptance.** Every offer in the archive involves 0025 (MFL shows
only a team's own offers); Christopher chose "Build it on your offers". `compile/build_lab.py`
`offer_outcomes` writes `offers` to lab.json (470; endings matched; 30 unknown); lab.json rebuilt and
verified: no other key changed. `model.js` decodes `offers`; `tradelog.js` `offersChart` (8 stretches,
stacked endings, acceptance rate, Sent/Received/Every offer). `sw.js` `lab-v10`. Tests: engine 14,
compiler OK, interact 8 steps (offers checks added), screens ok. Captures `filters/o-*.png`.
Christopher reviewed each round live: 7 ("much better" after 7b), 8 ("Good work"), 9 and 9b ("Good
work, this is fine for now"). Round 6's open item (the draft screen's old price unit) is gone: round 8
replaced that screen.

## 10. Environment
- The Lab server on 8765 (PID 1254811) serves the round-9b build at http://127.0.0.1:8765/.
- `.venv/bin/python` for anything that loads sqlite-vec or the models.

## 11. Honest status
The facts layer is proven against MFL two ways (its own totals and its web pages). Meaning search
is chosen by the exam (bge-base, hybrid). The Lab reads only the facts database. Rounds 7-9b are built,
tested (engine 14, compiler 22, interact 8 steps, screens) and seen by Christopher, who called it fine
for now. Not done: the Impeccable finish review for rounds 7-9b; the design record covers rounds 7-8
in prose, round 9's colours (posture #86bbf1/#b6a1e6/#3b4d63, tier ramp #c9d6e6/#6f86a3/#3b4d63,
offer endings) are not yet in DESIGN.md. The trade-note readings are Claude's and unconfirmed. Offers
exist only for the Cardinals (MFL limit), so the offers chart is one team's window, labelled so.
Nothing about TheWarRoom has been planned yet: that is the next session.
