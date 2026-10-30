# TheWarRoom UI/UX groundwork: RESUME (2026-10-30, compact-safe)

## 1. What we are doing
We are laying the UI/UX groundwork for TheWarRoom: a "command center / cinematic terminal" redesign, built as HTML mocks screen by screen, each one approved by Christopher from his screenshots. **Franchise HQ is done.** Next is a setup TALK with Christopher (he asked for it before more building), then Trade Floor.
- Repo (active): Beelink `chris@192.168.1.191:/home/chris/work/TheWarRoom`. The main checkout is on `session/league-history-pwa`, which belongs to ANOTHER session. **Do not touch it.**
- My worktree: `/home/chris/work/TheWarRoom/.worktrees/uiux`, branch `session/ui-ux-groundwork` (from origin/main 364379ba). Commits are local only and NOT pushed.
- SSH: `ssh chris@192.168.1.191` (key `/root/.ssh/beelink`).

## 2. Agents + harnesses
- GLM 5.3 free (`glm-5.3-flash-free`) via `~/fleet/bin/zen-ask.sh <model> <brief>` on the Beelink. It was used once to organise the rail tree. Its output is leads, not findings.
- Self-review loop: headless Chrome on CT105 (`google-chrome`). Script at `$SCRATCH/shot.sh <html> <png> <w,h> [js]` (it is in the scratchpad and will be lost; recreate it if needed). The script injects JS to set state, then screenshots. Check at 1920x1080 AND 2550x1352 (his monitor).
- "check SS" = read EVERY new file in Beelink `/home/chris/content/feedback/` (Flameshot savePath), not just the newest. Last one checked: `2026-10-30_10-36.png`.
- Mocks go to Beelink `~/Downloads/` and are opened in Brave as `file:///home/chris/Downloads/<name>.html`.

## 3. Status
| Item | State |
|---|---|
| Rail model | RULED: option 3 (fixed leaves + NOW line + My Shortcuts via Customize), opens on click |
| Franchise HQ: Lineup and roster | APPROVED (v10) |
| Franchise HQ: Contracts and cap | APPROVED (v8 RFA/UFA caps, release drawer) |
| Franchise HQ: My moves | APPROVED; now a queue control, not a tab (v10) |
| Bottom drawers Taxi/IR/What-if | APPROVED (v9 "YES YES YES") |
| L5 form bars | APPROVED, softened (e2836d69) |
| Trade Floor, War Room, League Pulse, Home, Control Room | NOT STARTED (my order) |
| Real build (React) | NOT STARTED: mocks only |

## 4. Artifacts
- Worktree `docs/ui/UI_Direction_v2_2026-10-30.md`: the living rule sheet. Read it first.
- Worktree `docs/ui/mocks/franchise-hq-mock-v10.html` (current), `rail-mock-v1.html`, `glm-rail-funnel-2026-10-30.md`.
- Beelink `~/Downloads/franchise-hq-mock-v10.html` (the copy he opens) plus older versions.
- Branch head: `e2836d69` (soften L5 bars). Chain: b04edd78 → 4e908065 → 0ce2b87d → 15c30070 → abf882d5 → e2836d69.

## 5. Current bug
None open.

## 6. Refuted — do not retry
- Stretching rows to fill tall screens (v4): he called it WORSE. Keep compact 30px rows and give spare space to real function.
- Amber-washed hazard stripes on the empty slot: he preferred the white stripes with high-contrast text; amber as an accent only (bar + OPEN label).
- Closed bottom drawer peeking its header above the tray: "stupid add, remove".
- Form bars at the far right edge: he didn't see them. Pack row data beside the name.
- Near-white small marks (form bars): "harsh on the eyes". Use mid-greys (~62% lightness).
- Hover-to-expand rail, and GLM's reorder-by-phase leaves: rejected.
- `google-chrome` does not exist on the Beelink (Brave only). Screenshot from CT105.

## 7. Decisions (Christopher, 10-30)
- Look: textured rubber grip / safety edge, never glow or bubbly. Subtle hover (1–2px, hard offset shadow, <80ms, transform/opacity only).
- No page scroll; drawers for detail; inspector for entities; tables scroll only inside their own frame.
- Every add or change needs a functional or aesthetic reason, never "just because". Function with form is welcome (matchup panel, win %, form bars, cap runway, ledgers all approved).
- Command-center doctrine: never second-guess a deliberate user move ("if they suck at commanding they lose").
- Position tags are drag handles; the lineup auto-adjusts (cascade + auto-fill).
- RFA = lime, UFA = signal yellow, as textured blocks plus a legend (classification hues kept off the 5 meaning colours).
- Drawers are sized to their content or carry more info; bottom drawers sit above the tray.
- Custom inspector drawers (MINE) are approved. Watch list = inspector tab; trade bait = asset-board filter.
- I take the wheel on screen order. Primary target 1920x1080. Excellence standard; an extra pass to kill slop is expected.

## 8. Ledger
Everything is committed on `session/ui-ux-groundwork` (local). Not pushed: pushing / opening a PR is not yet discussed with him.

## 9. Next actions
1. HAVE THE TALK: Christopher wants to discuss "a few things first so we are set up for future builds" before Trade Floor. Ask what's on his list; do not start building. Topics I'd raise if useful: how mocks become React (token sheet from the mocks → `style.css` vars; component list: drawer, pull tab, queue, form bars, hold bar); who builds (Terra/Tom per memory) and the brief format; push/PR for the branch; whether the mocks get a shared CSS file so each new screen inherits the system.
2. Then build Trade Floor (trade desk, offers, draft room) in the same loop: build → headless shots at 1080 and 2550 → slop pass → publish → commit.
3. Then War Room, League Pulse, Home, Control Room.

## 10. Environment notes
- The filing check on the Beelink PASSES. It notes `.git-credentials` and `.buzz` are gone since baseline; not mine, flag to Christopher if pushes from the Beelink fail.
- The Beelink pre-commit runs on docs commits (gitleaks passes).

## 11. Honest status
Mocks only. All data is illustrative. Win % has no engine behind it. Drag was proven in the HTML mock (he confirmed "works well"), not in WebKitGTK/Wails.
