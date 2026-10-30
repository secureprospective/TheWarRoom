# TheWarRoom UI/UX groundwork: RESUME (2026-10-30, compact-safe #2)

## 1. What we are doing
We are laying the UI/UX groundwork for TheWarRoom: a "cinematic terminal" redesign, built as HTML mocks screen by screen, each one approved by Christopher from his screenshots.
- **Franchise HQ is done (v10).**
- **War Room is done (v6).** It has 4 tabs: Trade Desk, Market, Draft Room, Plans. Christopher's verdict on it: "you have nailed it" / "great work".
- **Next:** the next section that makes sense to me. Christopher delegated the choice: "after compaction we will move to the next section that makes sense to you".

Repo and machines:
- Repo: Beelink `chris@192.168.1.191:/home/chris/work/TheWarRoom`. The main checkout is on `session/league-history-pwa` and belongs to ANOTHER session. **Don't touch it.**
- My worktree: `.worktrees/uiux`, branch `session/ui-ux-groundwork`. It's local only and NOT pushed. SSH key `/root/.ssh/beelink`.

## 2. Agents + harnesses
- **Terra research and opinion passes:**
  - Write the brief to Beelink `~/fleet/briefs/purge/<name>.md`.
  - Launch with `ssh -f ... 'rm -f ~/fleet/runs/terra-uiux/completion.txt; TERRA_WT=uiux TERRA_BRIEF=<name>.md TERRA_THINKING=high nohup ~/fleet/bin/launch-terra-reorg.sh >/dev/null 2>&1 &'`.
  - It's done when `~/fleet/runs/terra-uiux/completion.txt` exists. Two runs today, both good, about 2–3 minutes each.
- **Headless shots:** `$SCRATCH/shot.sh <abs html> <abs png> <w,h> [js]` on CT105. The console prints. Check at 1920x1080 AND 2550x1352.
- **Mock source parts** (scratchpad, concatenated): `base.css` (Franchise HQ v10 CSS) + `room.part.html` (War Room CSS) + `body.part.html` + `js.part.html`. Build: `{ head; cat base.css room.part.html; </style></head>; cat body.part.html js.part.html; } > warroom-mock-vN.html`.
  - If the scratchpad is gone, take the CSS/JS out of the committed v6 instead.
- **JS gotcha:** later function declarations override earlier ones (hoisting). So any early init call (renderX) runs the NEW version before its `var`/`const` data exists. Put init calls at the end and use `var` for late data.
- **"check SS"** means read EVERY new file in Beelink `/home/chris/content/feedback/`. The last one checked is `2026-10-30_12-19.png` (the bid-button slop note). Ignore non-dated files there; they belong to another project.
- **Publish:** scp to Beelink `~/Downloads/` and to the worktree `docs/ui/mocks/`, then commit. Christopher opens `file:///home/chris/Downloads/<name>.html` in Brave.

## 3. Status
| Item | State |
|---|---|
| Franchise HQ | APPROVED v10 |
| War Room · Trade Desk (two lenses, versions, Pocket, negotiation threads/counters, Tune It, coloured send/get trays) | APPROVED v4–v6 |
| War Room · Market (target lanes, pool, hard cap limit, bids READY vs PLACED·FINAL, outbid → bid again) | APPROVED v5 |
| War Room · Draft Room (draft order R1–R4/ALL/MINE, traded picks via, live mock draft, on-the-clock bar, tier bands, queue auto-removal, center scouting card) | "nailed it" v5 |
| War Room · Plans (named plans, per-event checklists, links, fallback notes, DUE items, ranked needs, live summary of the 3 tabs, qualified cap, conflicts; NO execute buttons) | built v6, "great work" |
| League Pulse, Home, Control Room | NOT STARTED |
| Real React build | NOT STARTED (mocks only) |

## 4. Artifacts (worktree `docs/ui/`)
- `mocks/warroom-mock-v6.html` (current War Room), plus v1–v5.
- `mocks/franchise-hq-mock-v10.html`.
- `research/scouting-notes-research-2026-10-30.md` and `research/plans-tab-opinion-2026-10-30.md` (both by Terra).
- `UI_Direction_v2_2026-10-30.md`: the rule sheet. **NOT yet updated with today's War Room rulings** (see Next).
- Branch head: `2ac1840f`. Today's chain: 249a1021 v1 → 33c7ca71 v2 → defd9e63 v3 → 8019503a v4 → bd2e21d3 v5 → 2ac1840f v6.

## 5. Current bug
None.

## 6. Refuted — do not retry
- **"PRICED OUT" / "NEXT UP" states:** slop. They're owner judgments, not machinery. Only show states the system really produces.
- **Big pale full-width "PLACE BID" button:** slop. Use a framed, knurl, hard-shadow control (`.cmdbtn` / placebar).
- **Emoji (🔒):** off-motif. Use mono stamps like LOCK.
- **Tier board as narrow columns:** names truncate. Use horizontal tier bands.
- **Removing a placed bid:** wrong. Placed bids are final; only being outbid frees the money.
- **Everything from the Franchise HQ list still applies:** no stretched rows, no amber washes, no peeking drawers, data next to what it describes, mid-grey small marks.

## 7. Decisions (Christopher, 10-30)
- Trade Floor folds into the War Room ("smoky back room"). The left rail has 5 nodes and will mature during the build; he's holding rail ideas back, so don't propose rail changes.
- **Universal:** players, picks and names are physical draggable objects (mechanical block feel).
- The War Room uses a warmer, darker charcoal on purpose; Franchise HQ stays cool grey.
- Calendar strip at the bottom = reserved spot for future franchise-sim play controls (AI runs the other teams). Reserve only.
- Market: cap room is a HARD limit on total bids (any league).
- Draft Room: the center scouting card (private report, quick capture) is approved direction. Draft order auto-follows traded picks; it's projected during the season. Wiring is "to talk about later".
- Plans = summary + light functions only, never execution.
- Screens are for long hours: easy on the eyes, a place users want to stay. Colour stays a labelled accent.
- **OPEN (asked, no answer yet):**
  1. Write "center card = exception to inspector-only" into the rule sheet?
  2. "One-of" bids (bid on 3 RBs, only the max counts)? Currently every bid counts.

## 8. Ledger
Everything is committed on `session/ui-ux-groundwork` (local). Nothing pushed; push/PR not discussed.

## 9. Next actions
1. On "we are back": recommend and start the next section. My pick is **League Pulse** (live games / race / reports), the next node in my order. Its "Now" live-game view pairs with the same excitement grammar as the draft clock. Alternative: Home (dashboard) last, because it summarizes everything else.
2. Before building, update `UI_Direction_v2` with today's War Room rulings in one short section (cheap, and it keeps the system durable). Commit it.
3. Build loop: mock → shots at 1080 + 2550 → slop pass → publish → commit → "check SS".

## 10. Environment notes
- The Beelink filing check PASSES (re-run 10-30 ~17:35 UTC).
- Nothing running: Terra finished, and the headless Chrome profile was reaped.
- `.git-credentials` is still gone from the Beelink home (not mine). Flag it if a push fails.
- The task-list `cc task add` has no TheWarRoom section, so the task list is left untouched.

## 11. Honest status
Mocks only; all data is illustrative. Lens math, win odds, ADP and trends are made up. Drag is proven in the HTML mock only, not in WebKitGTK/Wails.
