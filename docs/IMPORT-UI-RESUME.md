# TheWarRoom — Import + UI-to-mocks RESUME (compact-safe #5, 2026-11-10 ~09:40 CT)

## 0. FIRST THING AFTER COMPACT
Christopher's /compact-safe 11-10: "after compaction we will move to the next step of the plan". That is 9.5, the
batch point: make verify, bundle to the Beelink, push, PR, then launch on :0 for his real-screen look. Start it when
he says "we are back". 9.3, 9.4 and the tackle fix are DONE and reported. Still open with him: my call that the
Stat Changes "was" = the last live line. Nothing is running.
SEPARATE WORK, not mine: he will build the warroom.secureprospective.com website with Tom (Fable 5.2 on the
Beelink). I wrote Tom's prompt: Beelink ~/Downloads/TheWarRoom-WEBSITE-HANDOFF.md (copy /root/TheWarRoom-WEBSITE-
HANDOFF.md). Tom works in ~/work/TheWarRoom/.worktrees/website on session/website; when I push in 9.5 I make my own
NEW worktree and leave his alone.

## IN-FLIGHT
Nothing. The test app is stopped. Xvfb :5 (PID 1178155) is left up on Claude-OS for the next launch.

## 1. What we are doing
Build the five screens to the locked mocks on the NATIVE league, League Pulse first, import read folded into each
screen; then the lab server serves the app. Phone layout is out. Christopher 11-09: "most unique fantasy football
experience ever; act with purpose and conviction"; "this direction is good; excellence as the standard, lean and
clean code, precise execution with purpose".
- Handoff digested: Beelink ~/Downloads/TheWarRoom-IMPORT-UI-HANDOFF.md. North Star 2.1 (repo North_Star.md).
- Work on Claude-OS (`ssh claudeos`, always `< /dev/null`): worktree ~/work/TheWarRoom/.worktrees/ui, branch
  `session/ui-to-mocks` off main 8899a8bf. Claude-OS has NO git remote. To push: bundle the branch to the Beelink,
  fetch into a NEW worktree under ~/work/TheWarRoom/.worktrees/, push from there (pre-push runs the full suite,
  ~10 min, GOTOOLCHAIN=go1.26.10; a fresh worktree needs frontend/dist: pnpm install + build first). Squash merge
  only, after Christopher's live look.
- Env: export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.10 GOMEMLIMIT=1500MiB
  GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false" GOLANGCI_LINT_CACHE=$PWD/.golangci-cache
  VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1. Commit: -c user.name=secureprospective
  -c user.email=secureprospective@users.noreply.github.com, message from a file (-F), trailer
  Co-Authored-By: Claude Fable 5.2 <noreply@anthropic.com> + Claude-Session line. The pre-commit hook runs
  golangci-lint-full (~3-5 min); it has caught nilerr (use a bool predicate, not `x.Validate() != nil`) and
  appendAssign (use slices.Clone before append in tests).

## 2. Harness
I do the work myself. Edits: write files / python replace scripts in the scratchpad, scp to claudeos:/tmp, run,
remove BY NAME afterwards (a wildcard rm in /tmp is refused).
- `make build` (~20 s). `UIP=ui-profile2 sh ~/scratch/ui-desk.sh` launches on a profile. Wait ~140 s. Then
  W=$(xdotool search --onlyvisible --name "The War Room"|head -1); xdotool windowmove $W 0 0 windowsize $W 1920 1080;
  `scrot -o`. Stop: kill $(pgrep -x thewarroom). One chained ssh command does build+kill+launch+sleep+click+scrot.
- Profiles (real league data: stay on Claude-OS, delete when the job ends): ui-profile = OLD 11-01 seed league;
  ui-profile2 = FRESH import of MFL 14432 (the one to use). The app now opens straight on the app league, Home.
- Click points (app league, 1920x1080): Home (86,168) War Room (86,201) Franchise HQ (86,235) League Pulse (86,268)
  Control Room (86,302); an open node pushes later rows down ~27 px per leaf. Pulse prev-week arrow (260,75).
  MATCHUP BOX pull (300,1062), SCHEDULE pull (425,1062); schedule MY TEAM tab (630,436) in league view. Matchup rows
  in the left list start at y~245, ~61 px apart. Always screenshot before trusting.
- Show Christopher: scp -i /root/.ssh/beelink to chris@192.168.1.191:~/Downloads, give file:// path.

## 3. Gates
| Gate | State |
|---|---|
| vitest src/app/shell + src/app/data | 417/417 on e89c664c |
| tsc --noEmit | clean on e89c664c |
| Targeted go tests (player score Line, recorded starters, open-season projected field + row conference) | pass |
| pre-commit golangci-lint-full | passed on df01ea41, f17b8d4f, 9bc50015 and 599f2bad |
| make build | pass |
| make verify (full) | NOT run; once before the push |
| native_forks ratchet (19) | not lowered |
| Push / PR / Christopher's live look | not done (screenshots only) |

## 4. Artifacts
- Commits on session/ui-to-mocks: db668c16 Pulse slice 1; a75661a5 one rail; a9e364dc unusable stats id + recorded
  weeks parse; 05110385 pre-cutover lineups + recorded starters + slate/back button/matchup box; a8b4ab8a schedule
  drawer; df01ea41 starter stat lines; f17b8d4f Race drawers + projected seeds + row conference; 86b1d7f3 Race
  standings column; 9bc50015 Stat Changes drawer; ab414706 its regenerated bindings; e89c664c mock pass fixes; 599f2bad live tackles solo/assist; plus resume commits.
- Beelink ~/Downloads: twr-pulse-fresh-week{8,9}-1109.png, twr-pulse-fresh-week8-box-1109.png,
  twr-pulse-schedule-{league,myteam}-1109.png, twr-rail-{pulse,hq,control}-1109.png,
  twr-pulse-statline-{week9,box}-1110.png, twr-pulse-race-{race,tb,bk}-1110.png,
  twr-pulse-statchanges-week{9,8}-1110.png, twr-pulse-pass-{live9,sched,race,mock-race}-1110.png.

## 5. Current bug
None open.

## 6. Established: do not redo
- Starters DO land on a fresh import: current week at once, played weeks after a 5-minute launch delay. 288/288.
- Fixed and committed: pre-GSIS ids in the gsis slot (isGSIS in fantasyscore/source/held.go); parser accepts
  recordedResult; a LineupCorrect for a week before the recorded CutoverWeek skips the roster check
  (commissioner/lineup_week.go); RecordedStarters fills pre-cutover starters in the store's results.read only
  (not standings, not transfer), team total untouched.
- Week 8 starter points sum ~135.3 vs 134.8 recorded: app rules vs MFL's own scoring; the recorded total rules.
- App-league reads route through frontend shell/localWorkflow.ts (direct Wails bindings) or TargetWorkflow
  (commandcatalog/workflow_commands.go, closed workflowOutput type list). A read in neither says "this league action
  is not available here". Add a direct line in localWorkflow.ts for a bound read; do not widen workflowOutput.
- Importing an existing league is a RE-SYNC; league id native.mfl-<id>-<season>. The ledger refuses deletes; use a
  clean profile instead.
- Mock CSS for Pulse is fully ported in pulse.css (drawers, dtab, grid-sched, story, awards, bk already there).
  The bottom drawer is `.bdrawer` rendered inside `main.work`; pulls live in the tray (NativeLeaguePulse.tsx,
  `drawer` state 'box'|'sched', Esc closes). Mock: docs/ui/mocks/league-pulse-mock-v14.html.
- HQShell.tsx and WarRoom.tsx still carry their own rail copies for the non-embedded layout: collapse into
  ShellRail later, lower native_forks then.

## 7. Decisions (Christopher)
- 11-10: he gates step by step ("go ahead with the starter stat line", "Go" for Race drawers); report each step
  with screenshots, name the next one. Code quality and precise execution restated.
- 11-09: plan confirmed, Pulse first; clear pathway then execute; precise outputs; stay true to the mocks.
- 11-09 night: direction is good; excellence, lean and clean code; fix the lineup backfill then keep going on Pulse.
- My calls 11-10 (told him): tiebreak DETAIL is the engine's own long sentence (offered to cut down); crowded
  lineups keep two-line starter cells; standings points show two decimals; bracket headings from node ids.
- My calls (flag if he objects): >16 starters keep the single list with tight rows; recorded weeks show app-scored
  starter points beside the untouched recorded total; schedule grid shows last week through the next ones.

## 8. Ledger state
Everything committed (+ this resume). Not pushed. Beelink repo untouched. Untracked: only .golangci-cache/.

## 9. PLAN for the rest of this session (in order)
1. DONE 11-10 (commit df01ea41, not pushed): starter stat line
   (StarterScore.Line -> statLine.ts compactLine/boxLine; shots twr-pulse-statline-{week9,box}-1110.png). The league
   is now in week 10, so week 9 is one prev-arrow click. NEXT IS 9.2. Original note: Read how NativePlayerScore is built (commissioner PlayerScore source, fantasyscore StatLine
   Quantities); add a short stat line per starter to StarterScore in results.read (one field, built from the
   quantities already scored, no new fetch); parse it in nativePulse.ts; show it as the mock's `.ln` under each
   starter and as LINE columns in the matchup box. Go test + vitest case. Mock lines 1236-1240 and 1323-1330.
2. DONE 11-10 (commits f17b8d4f + 86b1d7f3, not pushed): TIEBREAKERS + BRACKET drawers (PulseRaceDrawers.tsx,
   shared PulseDrawer.tsx shell; pulseModel tieGroups/bracketRounds). Backend reads changed: bracket.read carries
   `projected` fields while the season is open (Fields stays empty), standings rows carry `conference`. Race opens on
   my own side; Race rail leaf click (55,325); pulls TIEBREAKERS (300,1062) BRACKET (425,1062). Shots
   twr-pulse-race-{race,tb,bk}-1110.png. PRODUCTION pull deferred (9.6). NEXT IS 9.3. Original note:
   Race drawers from reads already served: BRACKET (bracket.read is parsed already) and TIEBREAKERS (check what
   standings.read says about ties first). Pulls on the Race tab. Mock DRAWERS.race, lines 1345-1360.
3. DONE 11-10 (commit 9bc50015, not pushed): STAT CHANGES drawer. Backend: NativePlayerScore/StarterScore carry
   `revision` {stats[{stat,label,was,now}], was, now} (commissioner/score_revision.go) when the counted line is the
   final feed's and scores differently from the newest live (ESPN/Sleeper) line held; read only. Frontend:
   pulseModel statChanges/changePoints/changeVerdict, StatChangesDrawer in PulseNow.tsx, pull STAT CHANGES n at
   (545,1060) with the count; PulseScore.recorded. Results `changes` (commissioner adjustments) is still not shown:
   it is a different thing from the mock's drawer. NEXT IS 9.4 on his word. Original note: score changes are already in results.read (`changes`, currently dropped by the parser);
   line revisions exist in fantasyscore/source. Show first held vs latest. Mock line 1340.
4. DONE 11-10 (commit e89c664c): screenshot pass against the mock (mock rendered with headless chromium from a
   temp copy with a state script appended). Fixed: seg/filter buttons use `active` (CSS keys on aria-pressed);
   Escape closes only the Pulse drawer (capture, like RowMenu; the shell's Escape = inspector.close, which was
   hiding the inspector); MINE filter dropped; TO PLAY label. NOT built, needs data or his word (told him):
   Race right side = mock's RUN-IN table + projected bracket + "this week moves it" + bye %/magic number (app shows
   playoff-field cards); inspector extras (points against, all-play, streak, points by week, last 5, history);
   WEEK STORY, NFL BOX SCORE, PRODUCTION, ARCHIVE, schedule COMPACT tab, rail "LIVE leading" line; strip source
   line is long vs the mock's two short lines; matchup list shows a status tag where the mock shows win %.
4b. DONE 11-10 (commit 599f2bad): live tackle fix. ESPN SOLO -> tackle, assist = TOT - SOLO (AST wins if present);
   Sleeper live idp_tkl_solo -> tackle. Verified against the public ESPN summary and Sleeper stats feeds. Lines
   already held keep what was recorded, so week 9's Stat Changes rows stay; new live lines are right.
5. Batch point: make verify; bundle to the Beelink; push; PR; launch on :0 for Christopher's real-screen look
   (his display; never claim what he sees). Merge only after he confirms.
6. Deferred, needs his word or new backend: WEEK STORY (generated text), NFL BOX SCORE, PRODUCTION, ARCHIVE tab
   (history/eras/trophies). List what each needs and ask him to order them.
7. After Pulse: War Room trade desk (v6), Franchise HQ (v10), Home (v5), Control Room (v1); then Part C.
Stop rule: if a step needs a new ledger event, a schema change or a permission change, stop and tell him first.

## 10. Findings log (evidence seen; report, do not hunt)
- Week 9, my matchup: the last live line counts more tackles than the final feed for 9 of 13 changed starters
  (Chinn 10 -> 6, 24.0 -> 16.0 pts): looks like live = solo + assists, final = solo only. Told Christopher 11-10.
- Week 9 strip still says "0 with a result, 16 still to come" though every starter reads FINAL; the week-10 strip
  says "15 final" (ESPN's scoreboard is still last week's). The matchup box says "playing" for a 0.0 starter in a
  finished week. The no-lineup idle text repeats the same sentence for both teams. Schedule drawer shows 20 of 32
  rows without scrolling.
- Week 8 (recorded week) shows no stat changes: the live and final lines held at import score the same.
- The league is in week 10 now; week 9 still reads "0 with a result, 16 still to come / WAITING" (final stats not in).
- Pulse past-week view (week 9) drew without the right-hand inspector in one launch and with it in the next.
- Tie reasons name teams from both conferences (the tie is league-wide); the drawer shows this side's teams only.
- vitest prints "League Pulse read failed: TypeError ... reading 'API'" from one test; all tests pass.
- Before any import a new user sits in the OLD plain layout, not the mock shell.
- Import form: datetime fields pop a calendar over the field; Esc in the Control Room tray closes the whole tray.
- Stat feed writes ledger events every launch (commish.stat.queue.approve).
- Launch ~110-150 s at high CPU, 1.1-2.6 GB RSS (budget is 200 ms a screen).
- Log spam: trades.read "you may act only for your own franchise"; "the war room: body not read to the end";
  MFL liveScoring "rate limiter wait failed" falls back to the archived copy.
- Three directory players carry pre-GSIS ids (HAR447512, JON157260, THO581952): they read as no stats identity.
- Schedule read includes a week 14 with no games (dropped from the drawer).
- RailWeek and Pulse each run usePulseLive(''): two scoreboard pollers (mine; fold into one).
- With no franchise chosen RailWeek calls week.read with an empty franchise (mine; guard it).
- Claude-OS /tmp has fixwalk.py and mark.py of unknown origin: left alone.

## 11. Honest status
Rail matches the mock. Pulse: Live works on real data with starter stat lines and three of five drawers (box,
schedule, stat changes); Race shows today's projected seeds with two of three drawers (tiebreakers, bracket); Archive is not
started. Missing: WEEK STORY, NFL BOX SCORE, PRODUCTION. Nothing is pushed, the full suite has not run
on these commits, and Christopher has only seen screenshots.
