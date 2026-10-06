# RESUME — Legacy NFL Lab (2026-10-06, rounds 5-6: the MFL facts database, then the league view)

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
- **Impeccable documenter** (started about 10:10 on 2026-10-06): writes
  `tools/league-history/DESIGN.md` and `.impeccable/design.json`, nothing else. Alive if
  `/tmp/claude-1000/-home-chris/35b807e1-9d84-4024-b8c1-c4ae4b4ca27e/tasks/aed2177a26bcc61ca.output`
  is still changing (mtime). When done: check that both files exist and carry tokens; that closes
  the Impeccable FINISH line. If it died: rerun the documenter with the same packet (project root,
  artifact app/, the surface brief, PRODUCT.md, reference/document.md, write boundary = those two
  files).
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

## 8. Ledger state
Nothing from rounds 4-6 is committed except RESUME documents. Commit only on his go-ahead:
`git add tools/league-history docs/league-history docs/build-handoffs` (`.venv/` and `data/` are
ignored). Never `--no-verify`; no merge to main.

## 9. Next actions
Round 6 (2026-10-06), the league view, is built: no team lens anywhere. The market screen is a buy
list and a sell list against the going rate, then a calendar band. Christopher chose the layout on
the Impeccable decision page. Two finish-review rounds were scored "fix"; the third fix batch
(window marks must beat both other windows, no big-number pick clock, no split year spans,
12px seasons label) was applied and visually checked but not re-scored by the reviewer.
Product record: `tools/league-history/PRODUCT.md`; contract: `.impeccable/surfaces/app-index-html.md`.
0. Check the documenter (section 5); confirm DESIGN.md and `.impeccable/design.json` exist.
1. Report round 6 to Christopher (he has not seen the finished screen yet) and get his read.
2. His call: commit rounds 4-6 (`git add tools/league-history docs/league-history docs/build-handoffs`).
3. Open: the draft screen keeps its structure; its third summary still uses the old price unit
   (per expected win against a next-draft 1st). Bring it onto the going-rate unit if he wants
   the draft screen simplified the same way.
4. Optional, his call: the ~70 trade-note readings on `data/review.md`.

## 10. Environment
- The Lab server on 8765 (PID 1254811) serves the round-6 build at http://127.0.0.1:8765/.
- `.venv/bin/python` for anything that loads sqlite-vec or the models.

## 11. Honest status
The facts layer is proven against MFL two ways (its own totals and its web pages). Meaning search
is chosen by the exam (bge-base, hybrid). The Lab reads only the facts database. Round 6 is built and
tested; Christopher has not yet judged whether it is easier to read, which is the real test. The
trade-note readings are Claude's and unconfirmed.
