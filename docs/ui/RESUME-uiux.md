# TheWarRoom UI/UX groundwork: RESUME (2026-10-30, compact-safe #4)

## 1. What we are doing
We build HTML mocks one screen at a time, and Christopher approves each from his screenshots.
- DONE: Franchise HQ v10, War Room v6, League Pulse v14.
- **NEXT: the Home screen mock.** After that comes Control Room (commissioner area; it gets its own focused pass).

Where it lives:
- Beelink `chris@192.168.1.191:/home/chris/work/TheWarRoom`, worktree `.worktrees/uiux`, branch `session/ui-ux-groundwork`.
- Local only, NOT pushed. SSH key `/root/.ssh/beelink`.
- The main checkout is another session's (`session/league-history-pwa`). Don't touch it.
- Commit with `GOTOOLCHAIN=go1.26.10`, never `--no-verify`.

## 2. Harness
- **Pulse source** lives in the scratchpad (`/tmp/claude-0/-root/ceeceeaf-c38e-4b3f-bddd-137d1de3f99a/scratchpad/`):
  - Build order: `base.css + pulse.css`, then `pulse.body.html`, then `pulse.js1 js2 js3 js4 js6 js7 js5`.
  - js5 (init and wiring) must be LAST.
  - js7 is Race v2. It overrides renderStand/renderRace by hoisting.
  - If the scratchpad is lost, take everything from the committed `docs/ui/mocks/league-pulse-mock-v14.html`.
- **A new screen (Home)** should reuse `base.css` (the shared shell: rail, tabs, inspector, pulls). Copy the shell markup from the pulse or HQ mock.
- **Shots:** `$SCRATCH/shot.sh <abs html> <abs png> 1920,1080 "<js>"`.
  - Console errors print to stdout.
  - The JS must NOT contain `#`, because sed uses it as the delimiter. Use `querySelectorAll('.cls')` instead.
  - Also check at 2550,1352.
  - `rm -rf $SCRATCH/hcr` afterwards.
- **Publish:** scp to Beelink `~/Downloads/` and to the worktree `docs/ui/mocks/`, commit with the trailers, and give him the `file:///home/chris/Downloads/...` path.
- **"check SS":** `ls -1t /home/chris/content/feedback/ | head`. The last one read is `2026-10-30_14-14.png`.
- **Terra opinion pass:** brief in `~/fleet/briefs/purge/<name>.md`, launched with `ssh -f … TERRA_WT=uiux TERRA_BRIEF=<name>.md TERRA_THINKING=high nohup ~/fleet/bin/launch-terra-reorg.sh`. Done when `~/fleet/runs/terra-uiux/completion.txt` exists.

## 3. Status
| Item | State |
|---|---|
| Franchise HQ v10, War Room v6 | APPROVED |
| League Pulse v14: Live · Race · Archive | APPROVED ("in a good place") |
| UI_Direction_v2: War Room and Pulse sections added | committed b1f5a448 |
| Home mock | NEXT, not started |
| Control Room mock | after Home |

## 4. Artifacts
- Worktree `docs/ui/`:
  - Mocks: `franchise-hq-mock-v10.html`, `warroom-mock-v6.html`, `league-pulse-mock-v14.html` (current), `rail-mock-v1.html`.
  - Research: `research/*.md`.
  - `UI_Direction_v2_2026-10-30.md` (81 lines) is the ruling sheet. **Read it before designing Home.**
- Recent commits: eb22e0c0 v8, ee3a6695 v9, 2939359f v10, e334b7f0 v11, 14aa4173 v12, 9fdfe95f v13, b1f5a448 v14 + UI_Direction.

## 5. Current bug
None.
- Known mock-only gap: the SCHEDULE drawer data doesn't match the Race run-in schedule (SCH). This is cosmetic and was not raised.

## 6. Refuted — do not retry
- Lightening a card fill ("smoked out"). Keep the page colour inside cards.
- A rank sparkline inside the power table. Rank charts live in WHY IT MOVED.
- Score text sitting on the bar.
- A 1–16 fixed scale on the rank charts. It flattens the movement, so zoom each chart to the team's own range.
- LIVE tags in the matchup list. Show the win % per team, coloured text only.
- One win % per game. Show one per team.
- A narrow seed ladder that truncates names.
- Long prose stat lines.
- An orange border on the card.
- Invented owner-judgment states.
- Page scroll.
- Truncated names.
- Emoji.

## 7. Decisions (10-30)
All of them are in UI_Direction_v2 (Look/Space/Rail/Interaction/HQ/War Room/League Pulse). Highlights:
- Matchups is not a tab. Live's left list holds the matchups, and clicking one fills the center card.
- Power Rankings is first-class, because the offseason is most of the year.
- The schedule (run-in) drives the projected seed and the bubble flag.

**OPEN, ask him:**
1. Race default view by phase (power rankings in the offseason?).
2. Window labels ("win now", "rebuilding") shown league-wide?
3. Write "center card = exception to inspector-only" into the rule sheet?
4. Should "one-of" bids be allowed?

## 8. Ledger
Everything is committed on the local branch. Nothing is pushed; pushing hasn't been discussed.

## 9. Next actions
1. On "we are back": read `docs/ui/UI_Direction_v2_2026-10-30.md` and this file.
2. Start Home.
   - Ask Christopher the goal of Home first; never infer the session goal.
   - Then propose a one-screen layout that pulls from HQ / War Room / Pulse: NOW line, this-week matchup, decisions due, race snapshot, league moments.
   - Then build `home-mock-v1.html` on `base.css`.
3. Raise the 4 open rulings when there's a natural pause.

## 10. Environment
- Nothing is running. Terra is idle.
- Filing check exit 0.
- Old PNGs and superseded mocks were reaped from the scratchpad.

## 11. Honest status
These are mocks only, and all the data is illustrative. The engine math needed is not built yet:
- win % per game
- run-in projection
- dynasty components
- on-field possession flag
