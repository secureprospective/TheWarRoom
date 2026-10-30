# TheWarRoom UI/UX groundwork: RESUME (2026-10-30, compact-safe #5)

## 1. What we are doing
We build HTML mocks one screen at a time, and Christopher approves each from his screenshots.
- DONE: Franchise HQ v10, War Room v6, League Pulse v14, **Home v5 (approved "really good", palette SLATE)**.
- **NEXT: Control Room mock** (commissioner area). Ruled scope: **both** day-to-day commissioner jobs (approvals, rulings, corrections) **and** league setup/settings. **Co-commissioner gets near-parity** (same screen; matches K3 rulings memory).

Where it lives:
- Beelink `chris@192.168.1.191:/home/chris/work/TheWarRoom`, worktree `.worktrees/uiux`, branch `session/ui-ux-groundwork`. Local only, NOT pushed. SSH key `/root/.ssh/beelink`.
- Main checkout belongs to another session (`session/league-history-pwa`). Don't touch it.
- Commit with `GOTOOLCHAIN=go1.26.10`, never `--no-verify`.

## 2. Harness
- Scratchpad `/tmp/claude-0/-root/ceeceeaf-c38e-4b3f-bddd-137d1de3f99a/scratchpad/`:
  - Home source: `base.css + home.css`, `home.body.html`, `home.js` then `home2.js` (home2 wires in capture phase, must be last). Assemble: `{ head; cat base.css home.css; …; cat home.body.html; <script>; cat home.js home2.js; }`.
  - Pulse source: `pulse.css pulse.body.html pulse.js1 js2 js3 js4 js6 js7 js5` (js5 last).
  - Lost scratchpad → take from committed `docs/ui/mocks/*.html`.
- **New screen (Control Room)**: reuse `base.css`; copy shell (rail, tabs, inspector, tray, bdrawer, sdrawer, card overlay `#scrim/#card` + `openCard(html,sz)`) from `home-mock-v5.html`.
- Shots: `$SCRATCH/shot.sh <abs html> <abs png> 1920,1080 "<js>"`; JS must not contain `#`; also check 2550,1352; `rm -rf $SCRATCH/hcr` after.
- **Class-collision trap**: base.css owns `.r .mine .wire .mv .tl .pin .grab .sel` etc. Seen 3 times (`.r`, `.mine`, `.wire`). Prefix new classes.
- Publish: scp to Beelink `~/Downloads/` + worktree `docs/ui/mocks/`, commit with trailers, give `file:///home/chris/Downloads/...`.
- "check SS": `ls -1t /home/chris/content/feedback/ | head`; last read `2026-10-30_15-15.png`.
- Sol opinion pass: brief in `~/fleet/briefs/purge/<name>.md`, launch `ssh -f … 'SOL_WT=uiux SOL_BRIEF=<name>.md nohup ~/fleet/bin/launch-sol-reorg.sh >/dev/null 2>&1 &'`; done when `~/fleet/runs/sol-uiux/completion.txt` exists; report in `stdout.log`. Christopher asked for Sol's 2nd opinion on Home because it was technically hard; consider the same for Control Room.
- Do NOT grep the session JSONL with `.\{400\}` regex — it hung >120s.

## 3. Status
| Item | State |
|---|---|
| HQ v10, War Room v6, Pulse v14 | APPROVED |
| Home v5 (Brief · Depth charts · NFL stats, career card, slate) | APPROVED |
| UI_Direction_v2: Home section | committed 4ef214d8 / 80780b36 |
| Rulings below (1,2,4) in UI_Direction | committed with this resume |
| Control Room | NEXT, not started |

## 4. Artifacts
- Worktree `docs/ui/mocks/`: `franchise-hq-mock-v10.html`, `warroom-mock-v6.html`, `league-pulse-mock-v14.html`, `home-mock-v5.html` (current; v1–v4 history), `rail-mock-v1.html`.
- `docs/ui/UI_Direction_v2_2026-10-30.md` = ruling sheet. **Read before designing Control Room.**
- Commits this window: 5dd688cb v1, a5182e92 v2, a30a7da4 v3, 12917a18 v4, 4ef214d8 v5+rulings, 80780b36 slate.
- Sol Home opinion: `~/fleet/runs/sol-uiux/stdout.log` (Beelink).

## 5. Current bug
None.

## 6. Refuted — do not retry
- Lightened card fills; text on bars/hatching (WIRE tag on stripes was unreadable → solid block); rank sparklines in tables; 1–16 fixed chart scale; LIVE tags; one win % per game; truncated names; long prose stat lines; orange card border; invented owner-judgment states; page scroll; emoji.
- Home: jet-black background (rejected); Decisions Due as a default tile (demoted to add-tray); one-name-per-position ticker (must be top 5 per position, broadcast style); stacking/rotating alert band.

## 7. Decisions (10-30)
All prior ones are in UI_Direction_v2. New this window:
- Home = morning brief + personal dashboard + league front page, customizable; alerts push in regardless of settings (one-row band, all screens).
- Home palette SLATE.
- Control Room = both jobs + setup/settings; co-commish near-parity.
- **R1** Race defaults to Power Rankings in the offseason: YES.
- **R2** Window labels shown league-wide: YES.
- **R4** Center cards are an approved exception to inspector-only; more areas will use them.
- **R3 one-of bids: NO** — a bid is a bid; all open bids count against cap; cancel only when legally outbid. Recorded in UI_Direction.

## 8. Ledger
Everything committed on the local branch; nothing pushed; pushing not discussed.

## 9. Next actions
1. On "we are back": read this file + UI_Direction_v2.
2. (done) R3 ruled NO.
3. Control Room: read memories `project_warroom_k3_rulings`, `project_warroom_commish_full_control`, `project_warroom_commish_surface_process`, `project_warroom_cli_ai_actors`, `project_warroom_rules_are_league_data`; then propose a one-screen layout with subtabs (jobs vs setup), confirm, optionally Sol 2nd opinion, then build `control-room-mock-v1.html`.

## 10. Environment
- Nothing running. Sol idle. Filing check exit 0. Scratchpad reaped (632K).

## 11. Honest status
Mocks only; all data illustrative. Engine work not built: win %, run-in projection, dynasty components, news/depth feeds (source TBD), FFB stat scoring view, saved layouts server-side.
