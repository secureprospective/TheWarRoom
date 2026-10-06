# RESUME — Legacy NFL Lab (2026-10-05, round 5: the MFL facts database)

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
  Christopher's franchise: Arizona Cardinals `0025`.

## 2. Agents + harnesses
Claude only. Private Python environment `tools/league-history/.venv` (sqlite-vec 0.1.9, fastembed);
models cached in `data/models/`.

## 3. Gates

| Check | Result |
|---|---|
| `python3 store/build.py` (facts, 9 checks, note readings, cards) | all checks pass; ~20 s |
| `python3 -m unittest store/test_store.py` | 18 pass (~100 s) |
| `python3 store/factexam.py` (answers against MFL's web pages) | 2,291 / 2,295; the 4 are review items |
| `python3 store/exam.py --tables` (word search) | right card in top 10 for 130 / 135; first for 97 |
| Meaning search and model choice | **in progress**: see section 5 |
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

## 5. In flight
- `data/index_all.sh` (PIDs 1528599/1528601 at compaction), started with setsid at 21:32, log `data/index_all.log`: embeds the cards with
  bge-base-en-v1.5 (table vec_bge_base), bge-small-en-v1.5 (vec_bge_small) and
  nomic-embed-text-v1.5-Q (vec_nomic), in turn; about 20 minutes each. Alive if
  `pgrep -f index_all.sh`. **When done:** run `.venv/bin/python store/exam.py`, pick the model with
  the best hybrid score, set `MODEL` and the `card_vec` table in `store/index.py` to it (or rename
  the table), and drop the others.
- **Never rebuild while it runs**: `load.retire` refuses anyway. Any card change means re-indexing.

## 6. Settled — do not retest
- Points for = every week MFL scored (448/448). Win-loss = regular-season games, except 2013,
  which counted playoffs. Regular season = weeks where every team had an opponent. All-play = every
  week all teams scored (160/160). Home bonus +3.0 (2014-15 regular season, playoffs).
- MFL dates are US Eastern. The pre-2017 pick history in MFL's draft notes only covers trades made
  inside MFL. Offers in the archive are the Cardinals' only.
- Replacing the database beside a stale -wal corrupted it once: `retire()` now prevents it.
- MFL's transactions web page is empty without a login.

## 7. Decisions (Christopher)
- Shape: facts first, vectors on top. MFL data only. Review list for unreadable notes. The Lab's
  screens are frozen until rebuilt on the store.
- **Pending from him:** the `MFL_USER_ID` cookie in `league-archive/.secrets/mfl_cookies.txt`, so
  2013-15 assets, calendar, message board and polls can be pulled (12 requests; then delete the
  file). He asked "how do you want the credentials"; the steps were given.

## 8. Ledger state
Nothing from rounds 4-5 is committed except RESUME documents. Commit only on his go-ahead:
`git add tools/league-history docs/league-history docs/build-handoffs` (`.venv/` and `data/` are
ignored). Never `--no-verify`; no merge to main.

## 9. Next actions
0. **First thing on "we are back" (Christopher asked for it): tell him what he must do to get the
   rest of the data.** It is one thing: the MFL login cookie, for the 12 exports from 2013-2015
   (league IDs 51719, 47710, 21225) that MFL serves only to a logged-in owner: assets (pick
   ownership), calendar, message board and polls. Steps to hand him:
   1. In Brave, logged into MyFantasyLeague, open the league page.
   2. Press F12, then Application, then Cookies, then the myfantasyleague.com entry; copy the
      value of `MFL_USER_ID`. Do not paste it into chat.
   3. In a Beelink terminal (replace PASTE_HERE):
      `d=~/work/TheWarRoom/league-archive/.secrets; mkdir -p $d && chmod 700 $d`
      `printf '.myfantasyleague.com\tTRUE\t/\tTRUE\t0\tMFL_USER_ID\t%s\n' 'PASTE_HERE' > $d/mfl_cookies.txt && chmod 600 $d/mfl_cookies.txt`
   4. Tell Claude "cookie's in".
   Then Claude: move the 12 saved error files aside (archive.py skips any file that is valid
   JSON, and the error bodies are valid JSON), run `python3 archive.py 2013 2014 2015`, load the
   results, and delete the cookie file. Everything else MFL offers is already archived or is not
   kept for past seasons (checked against MFL's full export list). Also optionally: his own
   memory of the review items (the 2019 penalties, the four extra results).
1. Finish the model comparison (section 5) and fix the choice in `store/index.py`.
2. Report to Christopher: the shape, the checks, the review-list highlights (the 4 records, the
   2019 weeks 6-8 point penalties for the Texans and Seahawks, the 49 picks, ~70 note readings).
3. When the cookie lands, pull the 12 logged-in exports, add them to the store, delete the cookie.
4. Then rebuild the Lab's screens on the store (his call on order).

## 10. Environment
- The Lab server on 8765 (PID 1254811) still serves the frozen screens.
- `.venv/bin/python` for anything that loads sqlite-vec or the models.

## 11. Honest status
The facts layer is proven against MFL two ways (its own totals and its web pages). Meaning search
is built but the model is not yet chosen by the exam. The trade-note readings are Claude's and
unconfirmed.
