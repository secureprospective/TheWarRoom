# RESUME — Legacy NFL Lab (2026-10-05, after round 4: the method rebuild)

## 1. What we are doing
- **What it is:** an app that rebuilds 14 seasons of Legacy NFL history (MFL league 14432) as a
  dated ledger and values every trade and pick in one currency: wins above replacement.
- **What it is for:** surfacing where the market misprices assets, and when to take each
  position in the draft.
- **Where it lives:**
  - It is **not** part of TheWarRoom.
  - Repo `~/work/TheWarRoom` on the Beelink, branch `session/league-history-pwa`, folder
    `tools/league-history/`.
  - Live at http://localhost:8765/ (Beelink only).
- **Christopher's franchise:** Arizona Cardinals (`0025`). The app highlights it by default; the
  choice can be changed and is stored in localStorage.

### Round history
- **Rounds 1–3:** calendar, clock and stages.
- **Round 4 (this one):** Christopher said rounds 1–3 "look cool" but were broken or not
  actionable. He asked for a rigorous research pass on reconstructing the dataset and surfacing
  the outliers that matter, and for the whole app to be built on it.
  - The research is in `docs/league-history/League_Lab_Method_Research_2026-10-05.md`.
    Section 5 holds the findings.
  - His answers:
    - Main job: market inefficiencies, long-range outliers, and draft timing by position.
    - Yes to a "my team" view (Cardinals).
    - The clock only flavours the data; it is not its own chart.
    - Build the foundation first, then two screens, then review.

**Status: foundation plus Market prices and Draft timing are built and verified. Waiting for his
review. Nothing committed for round 4.**

## 2. Agents + harnesses
Claude only, no subagents.

## 3. Gates (all run after the last change)

| Check | Result |
|---|---|
| `python3 -m unittest compile/test_build_lab.py` | 21 pass (about 75 s) |
| `node --test tests/engine.test.mjs` | 8 pass |
| `node tests/screens.mjs http://127.0.0.1:8765/ <dir>` | 4 screens ok |
| `node tests/interact.mjs http://127.0.0.1:8765/ <dir>` | 10 interactions pass |
| Christopher acceptance | **pending** |

## 4. Artifacts

**Compiler**
- `compile/value.py` (new): Wins, Forecast, Picks (local-linear), Valuer, `fit_prices`/`market`,
  backtest, `analyse`.
- `compile/ledger.py` (new): the reconciliation proof.
- `compile/build_lab.py`:
  - Prunes the moves, teamWeeks and playerSeasons outputs.
  - Adds the `delivered` table; `x*` columns on legs; `aNet`/`aRealNet`/`window`/`season` on
    trades; `wins`/`slotWins`/`group` on draft rows; `allPlay`/`luck` on team seasons; and
    `market`, `checks2`, `ledger`.
  - Schema is `league-lab-2`.

**App**
- `js/engine.js` (new): the JS port of the price model, shrinkage (DerSimonian–Laird), funnel
  limits, draft judging and position effects.
- `js/model.js` and `js/ui.js` rewritten; `js/main.js` simplified to a top bar with seasons and
  your team.
- Views: `market.js`, `draft.js`, `tradelog.js`, `method.js`.
- Removed: `clock.js`, `charts.js`, `lens.js`, `stats.js`, the calendar/cycles/contracts/
  franchises/about views, and vendor D3.
- Service worker version `lab-v3`.

**Data, backup and screenshots**
- Data: `data/lab.json`, 2,713,661 bytes, sha256 prefix `ab68d850fa046955` (gitignored). The build
  takes about 41 s; the backtest is most of it.
- Exploration scripts from round 4: `~/fleet/runs/league-lab-2026-10-05/method/`.
- Session transcript: `~/.claude/projects/-home-chris/35b807e1-9d84-4024-b8c1-c4ae4b4ca27e.jsonl`.
- Feedback memory saved: `analysis-tools-must-yield-decisions-not-pictures`.
- Backup of the round 2–3 build: `~/fleet/runs/league-lab-2026-10-05/snapshot-before-method/`.
- Screenshots: `~/fleet/runs/league-lab-2026-10-05/shots-v4/`.

## 5. Current bug
None open.

## 6. Settled facts — do not retest
**Currency and forecast**
- All-play correlation of started wins above replacement: 0.966.
- The in-season prior is worth 6 weeks of new evidence (tested 2–12).
- The straight-line forecast overrates older players. It is calibrated per position group, age
  band and horizon on out-of-fold predictions.

**Pick values**
- A kernel average underrates the top slots; local-linear smoothing fixes it. Early-1st band
  mean against slot is now 0.00.

**Market**
- Discount 0.75 fits best. Contracts are about 0.007 wins per $1M. Extra asset −0.08 wins.
- An additive premium model was degenerate (premiums scaled with value). Use the multiplicative
  price per expected win against the `pick1:next` anchor.
- "next" means the next rookie draft from the trade date, not `asset.year == season`.

**Backtest (point in time)**
- Trade-level edges don't materialise: the favoured side came out ahead 52% of the time; slope
  about 0.10.
- Classes do: linebackers deliver about 1.55× forecast; young QBs, WRs and RBs 0.46–0.71×; 1sts
  0.79×.

**Ledger**
- 100% of week-to-week changes explained from 2018. Pre-2018 MFL weekly rosters are near-static.

**Draft**
- Within-band position effects are within chance.
- Pooled across rounds: defensive linemen about 70% and tight ends about 80% of slot; kickers
  beat their slot.

**Cardinals**
- Outside the funnel at −0.08 expected wins per trade over 140 trades; hindsight +0.06 over 45.

## 7. Decisions (Christopher)
- Franchise slots only (the archive has no owner names). Beelink only. No jargon.
- Round 4 answers are in section 1.
- He wants thoughts before code on direction changes. Round 4 was an explicit build request.
- **Next round candidates** (his call):
  - a "my team" dossier screen;
  - per-team counterparty dossiers;
  - who is likely to sell now (current state from all-play, roster age, picks, cap, trade bait);
  - proposal and rejection history;
  - start/sit skill from MFL's optimal lineups.

## 8. Ledger state
- HEAD has the earlier RESUME commits. **The whole Lab is uncommitted.**
- Commit only on his go-ahead: `git add tools/league-history docs/league-history docs/build-handoffs`.
  Never `--no-verify`; no merge to main without his go.

## 9. Next actions
1. Get his reaction to round 4 in his browser.
2. Apply corrections, rerun the four checks, and screenshot into `shots-v5/`.
3. Build the next round he picks from the candidates in section 7.

## 10. Environment
- **Server:** PID 1254811, `python3 serve.py --port 8765` in `tools/league-history`, started
  with setsid. It serves from disk. No autostart after a reboot.
- **Headless checks:** `/usr/bin/brave-browser` plus pnpm's cached `playwright-core`.

## 11. Honest status
Built and verified headless; **not yet seen by Christopher.**

- **What is solid:** the class-level findings hold in both halves of the history.
- **Trade-level numbers:** shown with an explicit warning that single trades are a weak guide.
- **Draft cells:** small, and shrunk accordingly.
- **IDP cheapness:** may partly reflect risk aversion to volatile positions. The data says it
  paid off anyway (delivered above forecast).
