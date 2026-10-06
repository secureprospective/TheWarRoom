# RESUME — Legacy NFL Behavior Lab (2026-10-05, after rounds 2 and 3)

## 1. What we are doing
An interactive app (the "Behavior Lab") so Christopher can study long-term GM habits in Legacy
NFL (32 teams, IDP, salary cap, MFL league 14432), as input for building a better TheWarRoom.
It is **not** part of TheWarRoom. Repo `~/work/TheWarRoom` on the Beelink, branch
`session/league-history-pwa`, folder `tools/league-history/`. Live at http://localhost:8765/
(Beelink only).

Rounds so far, all on his direction:
- **Round 1:** the first build. He judged it directionally right but overwhelming.
- **Round 2:**
  - The Calendar (renamed "League year") became the core, with a **league clock** shared with
    Market.
  - Cycles was turned into **team stages** (contending / rising / fading / rebuilding), with a
    switch to standings-only grouping.
  - New: price on the clock, and "when teams change direction".
  - Overview, Standings replay, Ask a question and four weak charts were cut.
- **Round 3 ("more emphasis on the rookie draft … who and why, benchmarking against the field,
  easy to understand, permit filtering"):** Rookie draft was rebuilt and moved into the main menu.

**Status: round 3 delivered and reported. Waiting for his reaction. Nothing committed except this
file.**

## 2. Agents + harnesses
Claude only, no subagents, no dispatches running.

## 3. Gates / status (all run after the last change)
| Check | Result |
|---|---|
| `python3 -m unittest compile/test_build_lab.py` | 14 pass (new: year-earlier rank, dead money counted once) |
| `node --test tests/engine.test.mjs` | 16 pass (new: stages, leans/turns, purchases, draft judging/benchmark) |
| `node tests/screens.mjs http://127.0.0.1:8765/ <dir>` | 8 screens ok |
| `node tests/interact.mjs http://127.0.0.1:8765/ <dir>` | 11 interactions pass (clock drag + pinning, turns, price, draft) |
| Christopher acceptance | **pending** |

## 4. Artifacts
- **App:** `tools/league-history/app/`
  - Main menu: League year (`views/calendar.js`), Market, Rookie draft, Cycles, Trade log.
  - Reference group: Cap & contracts, Franchises, About.
  - New module `js/clock.js`: the clock axis, the pinned strip, `clockSteps` and `clockCells`;
    `GUTTER = 92`.
  - Removed: `views/overview.js`, `views/standings.js`, `views/explore.js`, and `flow()` in
    charts.
- **Data** (private, gitignored): `data/lab.json`, 5,064,312 bytes, sha256 prefix
  `24be32e62a1fa4f6`; `lab.json.gz`, 683,543 bytes.
- **Backup** of the round-1 build: `~/fleet/runs/league-lab-2026-10-05/snapshot-before-refocus/`.
- **Screenshots:** `~/fleet/runs/league-lab-2026-10-05/shots/` (round 1), `shots-v2/` (round 2),
  `shots-v3/` (round 3).
- **Docs:** `tools/league-history/README.md` (updated with the "what is where" rows and lessons),
  research and plan docs in `docs/league-history/`.
- **Session transcript:** `~/.claude/projects/-home-chris/35b807e1-9d84-4024-b8c1-c4ae4b4ca27e.jsonl`.

## 5. Current bug
None open.

**Open question for Christopher:** are dead-money charges that are re-listed in later seasons'
files real per-season cap charges (rule: 35% of salary per year left)? The Lab assumes yes for
cap room. Events are counted once either way.

## 6. Settled facts — do not retest
- MFL `DP_r_p` current-year pick codes are zero-based (837/837). Arizona March 2024 = 0 received,
  1 sent.
- Picks are recorded in trades from 2017; contracts from 2019.
  - 361 one-sided records, excluded by default.
  - Washington's volume is real.
- Draft order ≠ reverse standings; a future pick is tied to a player only when its slot is
  proven.
- **Salary adjustment files re-list earlier cuts.**
  - 2015–18: every cut since 2015. From 2019: about 3 years back.
  - The compiler now dedupes events by (fid, ts, description, amount). An event's amount is the
    sum over the seasons that list it. `lab.dead` (cap room) is unchanged.
  - About 1,700 duplicate events removed (moves 26,232 → 24,532).
- **MFL player `----` in draft results** = a skipped or forfeited pick. 3 are dropped
  (draft 1,692 → 1,689).
- **Stages vs rank, in season, 2017+:**
  - Middle-ranked teams look neutral by rank alone (net per side +0.04).
  - Split by direction: rising teams buy (−0.145 net per side, 52% buy) and fading teams sell
    (+0.19, 49% sell).
- **Selling turn:** "first sell trade" was useless, because 122 of 145 in-season sellers had
  already sold in the offseason. The turn is therefore defined by **lean**: net pick value in the
  offseason (open→kickoff) against in season (kickoff→deadline). Results:
  - 24% of bought-or-held teams turned seller (43/182); half had turned by 27 days before the
    deadline.
  - 31% of sold-or-held teams turned buyer.
- **Price of a proven player (pick value per player):**
  - The median is stuck at 0.66 (a 2nd), so the chart uses the mean.
  - Older players get cheaper towards the deadline: 28+ goes 0.77 → 0.52; fading buyers pay
    0.55 at the deadline.
- **Draft benchmark ("same spot" = ±6 overall, every judged class):**
  - The league-wide gap is about 0.
  - NFL rounds 1–2 beat their spot by about +10 points; NFL rounds 6, 7 and undrafted fall short
    by about −13.
  - Over careers so far, 53% of starter seasons are spent on the drafting team (75% within the
    first 3 years).
- Last finished season = 2025 (`data.lastFinished`, from weeksDone ≥ regularWeeks).
- Playwright treats `stroke=transparent` lines as invisible: use `state: 'attached'`.
- Full-page screenshots draw sticky elements mid-page. That is a capture artifact, not a bug.

## 7. Decisions (Christopher)
- Franchise slots only; Beelink only; Bee's explorer replaced; no jargon.
- **Round 2 choices: 1a 2a+b 3b 4a.**
  - 1a: cuts as proposed.
  - 2a with b: stages are the default grouping, switchable to rank.
  - 3b: two screens sharing a pinned clock.
  - 4a: shared clock + price on the clock + selling turn first.
- He wants thoughts before code on direction changes. He said "lets talk before changing
  anything" at the first compact; round 3 was an explicit build request.

## 8. Ledger state
- Branch HEAD: this RESUME commit. **The whole Lab is uncommitted:**
  - new files untracked;
  - Bee's files staged as deletions;
  - `serve.py` modified.
- Commit only on his go-ahead:
  `git add tools/league-history docs/league-history docs/build-handoffs` (never `--no-verify`).
  Never merge to main without his go.

## 9. Next actions, in order
1. When he says "we are back":
   - read this file;
   - check the server: `ss -ltnp | grep 8765`. If it is down:
     `cd ~/work/TheWarRoom/tools/league-history && setsid -f python3 serve.py`.
2. Wait for or ask for his reaction to round 3 (the Rookie draft screen). Apply corrections he
   asks for, then rerun the four checks and re-screenshot into a new `shots-vN/`.
3. Leftovers he may pick, from earlier rounds:
   - the pick-value flow as a continuous band;
   - the weekly count of buyers against sellers;
   - one team's 14-year arc to replace the Cycles grid;
   - answering the dead-money question.
4. On his go-ahead: commit the Lab to the branch.

## 10. Environment notes
- Server: PID 1254811, `python3 serve.py --port 8765`, cwd `tools/league-history`, setsid. It
  serves files from disk, so rebuilt data and JS show on reload.
  - The service worker cache is `lab-v2` (network-first).
  - No autostart after a reboot.
- Rebuild the data with `python3 compile/build_lab.py` (about 4 s).
- Headless checks use `/usr/bin/brave-browser` and pnpm's cached playwright-core.

## 11. Honest status
Built and verified headless. **Not yet seen by Christopher in his own browser since round 2.**
Team draft report cards rest on 15–70 picks each, so differences of a few points are noise. Turns
rest on about 43 team seasons per direction. Findings are descriptive, not causal.
