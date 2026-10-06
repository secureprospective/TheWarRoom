# RESUME: TheWarRoom UI target and roadmap (2026-10-06, planning session)

## 1. What we are doing
- **Planning only.** We are designing TheWarRoom's target UI/UX well before the infrastructure
  under it. The target is the definition of done. Today's app screens are a **testing harness**
  that retires as the target screens go live.
- **Where:** repo `~/work/TheWarRoom`, branch `session/ui-target` (cut from main 2026-10-06). Never work on main.
- **Kept separate (Christopher, 2026-10-06):** the Legacy NFL Lab (`session/league-history-pwa`) is his own league research tool. It is not part of TheWarRoom and its work is never mixed into this branch; ideas may flow from it, code and commits do not.
- **Christopher's build rule: "think like an onion".** Start with the most frequent places people
  operate from. Build the core systems a little at a time, piecemeal, until the core is solid,
  then layer. Expect to revise the target as we learn.

## 2. Agents and harnesses
- **The research brief** (written by Claude): `~/fleet/briefs/warroom-endpoint-ia-2026-10-06.md`.
- **Sol (gpt-6.1-sol, through Bee/pi):** `~/fleet/briefs/warroom-endpoint-ia-bee-2026-10-06.md`.
  - 1,351 lines, 442 occurrence rows.
  - Machine rows: `~/fleet/runs/warroom-endpoint-ia-2026-10-06/bee-warroom-rows.json`.
- **GLM 5.3 (nvidia, through Bee/pi):** `~/fleet/briefs/warroom-endpoint-ia-bee-pass2-2026-10-06.md`.
  - 956 lines, 259 canonical rows.
  - Session `~/.pi/agent/sessions/--home-chris--/2026-10-06T16-17-33-782Z_01a11201-*.jsonl`.
- **The MFL menu map** (accepted 10-05): `~/fleet/runs/bee-mfl-ui-map-2026-10-04/`. The menu tree
  is `reports/S8-menu-tree.md`; the PDF is `deliverable/MFL-UI-map-2026-10-05.pdf`.

## 3. Status
| Item | State |
|---|---|
| MFL menu tree digested and shown to Christopher | done |
| Research brief written; both answers back and digested | done |
| Roadmap, target spec, registry; UI Direction §21 amended | committed on `session/ui-target` |
| Visual rough draft: `docs/ui/target-draft/TheWarRoom_Target_Draft_2026-10.html` (8 tabs) | committed; **approved by Christopher "with joy"** |
| Sol ↔ GLM row reconciliation | open: a rough name-match left 187 Sol rows unmatched, mostly aliases. Assigned to ring 0 |

## 4. Artifacts (written 2026-10-06, uncommitted)
- `docs/build-handoffs/UI_Target_Roadmap_2026-10.md` (175 lines)
  - The onion rings 0–4 with their gates; gap closure; harness retirement; definition of done.
  - Brainstorm record: Gameday Default/Classic views; the startup-draft hybrid.
- `docs/ui/Target_UI_Spec_2026-10.md` (132 lines)
  - Rulings, consensus skeleton, the four channels, the organization tree, card families, the
    move envelope, new concepts, open questions.
- `docs/ui/endpoint-registry.csv` (263 rows: GLM's 259 + X-01..X-04)
  - Columns: id, source, name, kind, gm_question, when, gravity, clock, data, placement, verb,
    disposition, ring, status=spec.
  - Built by `~/fleet/runs/warroom-endpoint-ia-2026-10-06/build-registry.py`. Its input path
    reads GLM's file; its output path is an argument.
- `docs/ui/UI_Direction_Document.md` §21: three rulings plus version 1.1.

## 5. Current bug
None. This was a planning session.

## 6. Settled; do not retest
- **Free live NFL feeds,** probed with curl 10-06 (no keys):
  - ESPN `site.api.espn.com/apis/site/v2/sports/football/nfl/summary?event=` gives drives and
    plays, each with text, down/distance, yardLine and possession, plus the box score and win
    probability.
  - Yahoo `api-secure.sports.yahoo.com/v1/editorial/s/scoreboard?leagues=nfl&week=N&season=Y` gives
    the live down, distance, start_yardline, team_in_possession and last_play; `/s/boxscore/<gid>`
    works too.
  - Fox `api.foxsports.com/bifrost/v1/nfl/event/<id>/data?apikey=<public web key, see ~/fleet/runs/warroom-endpoint-ia-2026-10-06/feeds.md>`
    gives pbp by drive and key plays.
  - **No usable feed:**
    - CBS serves HTML only.
    - feeds.nfl.com returns 403 and needs credentials from DM.ProductOps@nfl.com.
    - Sleeper has live stat lines but no plays.
    - The free API tiers are too small.
  - None of the feeds is licensed. Facts are free (NBA v. Motorola). Official play-by-play comes
    only through Genius Sports.
- **Headshots:** ESPN and Sleeper keyed by crosswalk espn_id / sleeper_id. Fox has no ID in the
  crosswalk. Photos are copyrighted: personal use only.
- **Auction scenario on Legacy's 2025 scoring:**
  - The weakest pool slots (Pitts 120, Chase/Pickens 207, value over replacement) are worth less
    than the best players left for the snake (Garrett 247, Brooks 236).
  - Injured stars (Burrow, Daniels) fall out of a pool ranked by season total.

## 7. Decisions (Christopher, 2026-10-06)
1. **Two clicks** = ≤2 in-app clicks to a ready move with MFL opened. The hold is a gesture,
   typing is free, and MFL's own form steps are counted separately.
2. **Inspector:** 320px at rest; `inspector.expand` widens it to ~480px as an overlay.
3. **Neutral badges:** position, tier and posture are monochrome glyphs (amends §7.3).
4. **The onion build** (above).
5. **Gameday matchup:** a Default view (formation, laid "across the line": my offense against
   their defense) and a Classic view (a list). All leagues get both; Legacy uses Classic. **Do
   not build it until he briefs the how and why.** The default ruleset idea is 11 defenders
   against 6 offense plus an OL visual; scoring later.
6. **Startup draft (brainstorm):**
   - Round 1 is an auction, rounds 2–25 a snake. It is his candidate default; owners liked it.
   - The pool is the top 8 QB, 5 RB, 5 WR and 2 TE.
   - Claude's proposed rules: book price as the minimum bid, random order, the pool ranked by
     points per game with a games minimum, a round-2 lottery, sealed second-price bids.
   - "We will talk about this later."

## 8. Ledger state
- Branch `session/ui-target` (from main) holds this RESUME and the whole planning set, including the
  design board. Not pushed, not merged.
- The Lab (`session/league-history-pwa`) is separate. Its rounds 4–9b are parked uncommitted on
  that branch, Christopher's call. Never stage Lab files on this branch.

## 9. Next actions
1. **Ring 0**, when the token budget allows:
   - Reconcile Sol's 442 rows into the registry.
   - Start gap closure: the mflmap owner-form capture with no submits; league rules from
     allRules.json, O=09 and By-Laws; ask Christopher about waiver mechanics, DOT, windows, taxi/IR
     and Victory Points.
   - Begin the walking skeleton: tokens, card base, command registry, endpoint registry with the
     route test, the fixture → live provider pattern, the league-year clock, the envelope on
     fixtures. Use the design board as the visual reference.
2. Still queued, his call: remove NFL contract data (`RESUME-2026-10-04-after-power-rankings.md`).

## 10. Environment
- The Lab server runs on 127.0.0.1:8765 (PID 1254811).
- Claude-OS is running; leave it running.
- Bee/pi sessions are Christopher's; do not kill them.

## 11. Honest status
- The target is revision 0, an honest merge of two strong answers.
- **Unproven:**
  - 8 MFL action targets are unmapped.
  - The registry's ring assignments are a first cut.
  - Neither model nor Claude has seen any target screen rendered.
- No build has started.
