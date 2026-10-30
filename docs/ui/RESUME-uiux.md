# TheWarRoom UI/UX groundwork: RESUME (2026-10-30, compact-safe #3)

## 1. What we are doing
We are building HTML mocks screen by screen, each one approved by Christopher from his screenshots.
- Franchise HQ v10 and War Room v6 are DONE.
- NOW: **League Pulse**. Christopher's rule is one tab at a time, from easiest to hardest.
  - Archive: DONE (good).
  - Live: DONE ("pretty stinking cool").
  - **NEXT: Race, then Matchups.**

Repo and branch:
- Beelink `chris@192.168.1.191:/home/chris/work/TheWarRoom`, worktree `.worktrees/uiux`, branch `session/ui-ux-groundwork`.
- Local only, NOT pushed. SSH key `/root/.ssh/beelink`.
- The main checkout belongs to another session (`session/league-history-pwa`). **Don't touch it.**

## 2. Harness
- **Pulse mock source** lives in the scratchpad. Build order:
  `head + base.css + pulse.css + </style></head> + pulse.body.html + pulse.js1 js2 js3 js4 js6 js5`
  - **js6 must come BEFORE js5.** js5 holds the init and the wiring. js6 holds Archive v2 and overrides renderArch through hoisting.
  - Build command:
    `{ printf '<!doctype html>\n<html lang="en">\n<head>\n<meta charset="utf-8">\n<title>…</title>\n<style>\n'; cat base.css pulse.css; printf '\n</style>\n</head>\n'; cat pulse.body.html pulse.js1 pulse.js2 pulse.js3 pulse.js4 pulse.js6 pulse.js5; } > pulse-mock-vN.html`
  - If the scratchpad is lost, take everything from the committed `docs/ui/mocks/league-pulse-mock-v7.html`.
- **Shots:** `$SCRATCH/shot.sh <abs html> <abs png> 1920,1080 "go('race')"`. Console errors print to stdout. Also check at 2550,1352.
- **Testing live ticks:** pass `"go('live');tick();tick()"` as the shot's JS.
- **Publish:** scp the mock to Beelink `~/Downloads/` and to the worktree `docs/ui/mocks/league-pulse-mock-vN.html`, then commit with the Co-Authored-By trailer.
- **"check SS":** `ls -1t /home/chris/content/feedback/ | head`. The last one read is `2026-10-30_13-20.png` (the Live layout request).
- **Terra opinion pass:**
  - Write the brief to `~/fleet/briefs/purge/<name>.md`.
  - Launch: `ssh -f … 'rm -f ~/fleet/runs/terra-uiux/completion.txt; TERRA_WT=uiux TERRA_BRIEF=<name>.md TERRA_THINKING=high nohup ~/fleet/bin/launch-terra-reorg.sh >/dev/null 2>&1 &'`.
  - Done when `completion.txt` exists.

## 3. Status
| Item | State |
|---|---|
| Franchise HQ v10, War Room v6 | APPROVED |
| League Pulse reorg (Terra opinion + my edits): 4 tabs Matchups · Live · Race · Archive, no new rail node | APPROVED |
| Archive (v2): franchise history, eras by owner, win% bars + rank-of-32 + stamps (DIV TITLE + best result), hover tip, season story, playoff run, WHO WAS HERE lineage / HOW WE GOT HERE moves, trophy shelf → full-screen TROPHY CASE drawer, FRANCHISE/LEAGUE toggle | "in a good place", may revisit later |
| Live (v3–v6): field + follow bar on top; center = raised MY MATCHUP card (16 v 16, OFFENSE/DEFENSE groups, compact stat shorthand, solid TD tags first, clock chip, brown football = on field by possession); right = IN THIS GAME (natural height) + PLAY-BY-PLAY fills the rest; ball/LOS slide + dashed gain trail on new play | APPROVED |
| v7 jet-black Pulse palette (neutral black, soft warm-white text ~84%), truer-green field | just shipped, awaiting his look |
| Race (v1: standings + my road left; center PLAYOFF PATH bigs/odds/scenarios ↔ POWER; drawers TIEBREAKERS/BRACKET/PRODUCTION) | NEXT. Not reviewed yet in the one-at-a-time pass |
| Matchups (v1: game list, score header, across/classic) | after Race |

## 4. Artifacts
Worktree `docs/ui/`:
- Mocks: `mocks/league-pulse-mock-v1..v7.html` (v7 is current).
- Research: `research/league-pulse-reorg-opinion-2026-10-30.md`.

Commits on the branch: bd9a2b47 v1 → e27521ac v2 → acf63970 v3 → ee90a5a4 v4 → 1e22d5b7 v5 → 08f8e9f2 v6 → 75f45479 v7 (plus this resume).

## 5. Current bug
None.

## 6. Refuted — do not retry
- **Lightening a raised card's fill ("smoked out"):** REJECTED. Cards keep the page colour; lift comes only from the light border, the hard black shadow and the hazard top edge. This is saved in memory.
- **Narrow 10-column seed ladder in Race:** it truncated names and duplicated the standings. Replaced by a 2-column odds list.
- **Long prose stat lines** ("16 car, 71 yds"): too busy. Use the shorthand "16-71 rush | 0-23 rec" with the TD tag FIRST so it never clips.
- **Orange or yellow border for the matchup card:** I chose white instead, because orange means warning or action elsewhere.
- **Everything earlier still applies:** no PRICED OUT-style owner judgments, no emoji, no big pale buttons, no page scroll, names never truncated in narrow columns.

## 7. Decisions (Christopher, 10-30)
- **Live is the main game-day / TV screen.** My matchup goes center, side by side. Play-by-play of the selected NFL game auto-fills the right column below "in this game", which is never cut off.
- **On-field football:** offense players get it when their team has the ball; defense players get it when the opponent has the ball.
- **Field animation:** the light version is done. A full play animation is for later.
- **League Pulse palette = jet black.** The War Room stays smoky and HQ stays cool grey.
- **Archive:**
  - It defaults to the user's team.
  - Eras by owner explain "how did we get here".
  - The trophy case is a small shelf that opens to a full card; the information leads.
  - Animated trophies come later.
- **Still OPEN from before:**
  1. Write "center card = exception to inspector-only" into the rule sheet?
  2. Should "one-of" bids be allowed?
  3. `UI_Direction_v2` still lacks the War Room and Pulse rulings.

## 8. Ledger
Everything is committed on the local branch. Nothing pushed; pushing hasn't been discussed.

## 9. Next actions
1. On "we are back": start the **Race** tab pass. Screenshot the v7 race view (`go('race')`), do a slop pass in jet black, and propose the improvements. Christopher steers each tab.
2. Then **Matchups**: re-check it under jet black. Live already shows my matchup, so Matchups should focus on the whole league's week.
3. Later, add the War Room and Pulse rulings to `UI_Direction_v2`.

## 10. Environment
- Nothing running. Terra is idle.
- Headless Chrome profile reaped.
- Beelink filing check passes.

## 11. Honest status
Mocks only; all data is illustrative. The stat shorthand and the on-field logic are mock logic. In the real build they need the live feed's possession flag and stat lines.
