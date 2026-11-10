# TheWarRoom — UI-to-mocks RESUME (compact-safe #6, 2026-11-10 ~12:05 CT) — WAR ROOM TRADE DESK

## 0. FIRST THING AFTER COMPACT
Pulse + rail are MERGED (PR #223, squash, main 57c86968, Christopher merged after his look on :0). We are now on
the WAR ROOM TRADE DESK (mock docs/ui/mocks/warroom-mock-v6.html). Christopher approved my plan (section 9) and said:
"this page has a nice skeleton, but if you feel something needs to be tweaked, added, or reworked, do not let me
stop you ... Make sure you cover the bases before presenting anything to me to limit turns on slop you missed."
So: build steps 2-3 (then onward), self-review each against the mock on the real league BEFORE showing him, show
fewer, fuller results. On "we are back": continue at section 9 step 2. Nothing is running.

## IN-FLIGHT
Nothing. Test app stopped. Xvfb :5 left up on Claude-OS. No app on his :0 (I stopped it to use the profile).

## 1. What we are doing
Build the five screens to the locked mocks on the NATIVE league. Done: rail, League Pulse. Now: War Room trade desk,
"the biggest selling point" (his words): it must beat KeepTradeCut (KTC) by using the league's REAL rules:
two lenses (worth to you vs worth to them), real lineup effect, cap legality. KTC = crowd sentiment, one generic
number 0-9999, a curve that penalises many-for-one; we do not have crowd sentiment and will not fake it.
- Work on Claude-OS (`ssh claudeos`, always `< /dev/null`): worktree ~/work/TheWarRoom/.worktrees/ui, branch
  `session/war-room-trade-desk` off main 57c86968. Claude-OS has NO git remote. To push: `git bundle create
  ~/scratch/x.bundle 57c86968..session/war-room-trade-desk`, scp via CT105 to the Beelink ~/scratch, then in the
  Beelink worktree ~/work/TheWarRoom/.worktrees/ui (MINE; currently on session/ui-to-mocks, switch it with
  `git fetch <bundle> branch:branch` + checkout) push with PATH/GOTOOLCHAIN=go1.26.10 set (pre-push hook ~1 min;
  frontend/dist must be built: pnpm run build). gh is ~/.local/bin/gh on the Beelink. NEVER touch
  .worktrees/website (Tom's landing page work) or the main checkout.
- MERGE: the auto-mode check refuses my `gh pr merge` ("merge without review"). Christopher merges, or tells me.
- Env: export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.10 GOMEMLIMIT=1500MiB
  GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false" GOLANGCI_LINT_CACHE=$PWD/.golangci-cache
  VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1. Commit: -c user.name=secureprospective
  -c user.email=secureprospective@users.noreply.github.com, -F message file, trailers Co-Authored-By: Claude Fable
  5.2 <noreply@anthropic.com> + Claude-Session line. Pre-commit runs golangci-lint-full (~3-5 min on Go changes).

## 2. Harness
I do the work myself. Edits: python replace scripts in the scratchpad, scp to claudeos:/tmp/twr-*.py, run, rm by name.
- `make build` (~20 s). `UIP=ui-profile2 sh ~/scratch/ui-desk.sh` launches on :5 (real league, fresh import). Wait
  ~150 s. W=$(xdotool search --onlyvisible --name "The War Room"|head -1); windowmove 0 0, windowsize 1920 1080;
  scrot -o. Stop: kill $(pgrep -x thewarroom) (never pkill -f).
- Click points on :5 (1920x1080): Home (86,168) War Room (86,201) Franchise HQ (86,235) League Pulse (86,268)
  Control Room (86,302). War Room tabs: TRADE DESK (280,22) MARKET (378,22) DRAFT ROOM (476,22) PLANS (570,22).
  Partner select (1418,303). POCKET (278,1052) NEGOTIATIONS (375,1052). On his :0 everything sits ~63 px lower.
- His screen: launch the same way with DISPLAY=:0 XAUTHORITY=$HOME/.Xauthority, then
  wmctrl -i -r $W -b add,maximized_vert,maximized_horz. His display is 1920x1080. Leave Brave alone.
- Show him: live on :0, or scp -i /root/.ssh/beelink to chris@192.168.1.191:~/Downloads + file:// path.
- "check ss" = newest files in Beelink /home/chris/content/feedback (red marks + notes).
- Render a mock state: headless chromium on Claude-OS on a temp copy of the mock with a state script appended.

## 3. Gates
| Gate | State |
|---|---|
| PR #223 Pulse + rail | merged, CI green |
| War Room parser fix (commit on new branch) | tsc clean, build passes, War Room opens on the real league |
| Full suite / push / PR for the trade desk | not started |
PUSH-GATE LESSONS (cost four round trips on #223): before pushing run on Claude-OS: `make verify` (govulncheck now
installed); wire snapshots regenerate with TWR_WRITE_WIRE=1 go test ./internal/store/history (when a read gains a
field); no source line over 110 chars (grep -rnE "^.{111,}$" frontend/src/app --include=*.ts --include=*.tsx);
`as` must not start a wrapped line. CI flakes seen: internal/mfl TestDo_HostRouting (real network), mac install job
timeout; `gh run rerun <id> --failed`.

## 4. What exists (do not rebuild; engines are done, expose and wire)
Frontend skeleton (frontend/src/app/shell): WarRoom.tsx (140 lines), WarRoomTrade.tsx (122), WarRoomHeader.tsx,
WarRoomInspector.tsx, WarRoomDrawers.tsx, warroomTradeReads.ts, warroomInspectorReads.ts (already calls value.read),
warroomReads.ts, tradeBuilderState.ts, warroom.css, trade.css. Working today: shelves, drag to trays, partner
select, trade.offer PREVIEW (legality/cap fields), lineupdelta.read (both sides, +x/wk, enters/leaves/benched),
header Best 21 + rank, value by age, cap runway. Showing "Not available yet": YOU/THEM value columns, tray totals,
both lenses, verdict, TUNE IT, partner needs/spare/lean, needs-next-season ("Not built yet"), + VERSION, pocket.
Backend reads: value.read -> appvalue.Reading {players[{player,franchise,position,nowPPG,dynastyPPG,age,ageKnown,
available,words}], best{value,rank,of,starterN}, age bands} (internal/appvalue; scope league+franchise+season);
lineupdelta.read; trade.offer (preview/execute); native.assets.read; native.cap.check; standings.read; trades.read;
pick value model internal/pickvalue (Model.Pick(year,round,slot,teams), Selection) and app/market pick_history.go:
NOT yet confirmed served for a native league; age arc internal/app/market/age_arc.go.
NOT in the backend at all: direction/posture (contend/retool/rebuild), lenses, tune-it, pocket/versions.

## 5. Current bug
None open.

## 6. Established: do not redo
- War Room error on the real league was the frontend snapshot parser (commissioner/hostedReadings.ts) rejecting
  backend fields: snapshot.pickHistory, snapshot.sides, pick.compensatory, pick.correction. Fixed, committed.
- App-league reads route through shell/localWorkflow.ts or TargetWorkflow (commandcatalog/workflow_commands.go,
  closed workflowOutput list). Add a direct line in localWorkflow.ts for a bound read; do not widen workflowOutput.
- table class `pwr` in pulse.css is a grid ROW style: never put it on a table (caused the Power "slop").
- Importing an existing league is a re-sync; the ledger refuses deletes; use a clean profile instead.

## 7. Decisions (Christopher, 11-10 unless dated)
- APPROVED: add a new ledger event "owner set their direction" (contend/retool/rebuild). Ruled 10-31: the owner
  picks it, the lens follows it, the label is public to the league (win now / retool / rebuild).
- AGREED: a partner who has not picked shows OUR read from record + roster age, labelled as ours
  ("our read · they have not set it").
- GOOD: lens = a blend of the two values the engine already has (contend leans nowPPG, rebuild leans dynastyPPG,
  retool between); nothing new fitted. I owe him the exact weights with real examples from his league before it
  ships.
- Free hand to tweak/add/rework the desk; cover the bases before presenting; limit his turns.
- Crowd sentiment (KTC's strength): left out, not faked.
- Pulse leftovers PARKED for a smaller model (Hermes task list, SUPPORT, added 11-10). Tiebreak wording and Race
  fixes accepted ("Pulse is in a good place").
- Standing: address him as Christopher; outcome first, short plain replies; times Central; post ETAs; ping when I
  need him; excellence, lean clean code; mock is the gate; never invent states; don't hunt for issues.

## 8. Ledger state
Branch session/war-room-trade-desk: one commit (parser fix) + this resume. Not pushed. Untracked: .golangci-cache/.

## 9. PLAN (approved 11-10), gated; self-review against the mock before each show
1. DONE: War Room opens on the real league (parser fix).
2. NEXT: values on every block: YOU and THEM columns + tray totals from value.read (picks included: first confirm
   how pick value reaches a native league; if it needs a new read, add it additively). Mock scale is 0-100 per
   asset ("77 to you, 42 to them"): find how appvalue numbers map to that scale (model.Level/Score) before choosing.
3. Top strip to the mock: direction buttons, Best 21 + rank, value by age, NEEDS · NEXT SEASON (QB ok / RB thin /
   DE hole), cap runway.
4. Direction event (approved) + the two lenses + verdict line; bring him weights with real examples.
5. Their read: record · cap room, needs, spare, one-line lean.
6. Inspector card: contract, value arc, in your lineup, fills need, notes.
7. TUNE IT: smallest changes that keep it acceptable to both.
8. Pocket, versions A/B/C, send, negotiations.
Proof each step: desk numbers equal the engine's own read; screenshot on the real league.
Stop rule: any OTHER new ledger event, schema change or permission change -> tell him first.

## 10. Seen on the real league's trade desk (11-10 first load; fix while building, these are evidence not hunting)
- Shelf blocks show no age and no value (the two right-hand cells are empty).
- Header: "window Lineup" reads wrong; first age band is labelled "≤1" (should be 23 and under); Direction shows
  three buttons + "Not set" in a wide empty box; "Not built yet" under NEEDS.
- The bottom calendar strip shows ten old events all marked "passed" in two rows and covers TUNE IT; the mock
  reserves that strip (UI direction: reserve the space only, don't build it).
- SEND / SAVE TO POCKET sit bottom right; POCKET and NEGOTIATIONS pulls bottom left.
- Right inspector: "Pick a player to see their details" in an otherwise empty column.
- Pulse findings list from the earlier resume is in git history (docs/IMPORT-UI-RESUME.md at 57c86968 section 10).

## 11. Honest status
The trade desk is a working skeleton with the calculator parts empty. Only the load error is fixed. No lens math
exists yet. Live NFL field/drive/play-by-play in Pulse has never been seen during a real game (Thursday night is
the first chance).
