# TheWarRoom — Import + UI-to-mocks RESUME (compact-safe #3, 2026-11-09 ~22:30 CT)

## 0. FIRST THING AFTER COMPACT
Christopher asked for a plan for the rest of this session. It is section 9. Start at 9.1. Nothing is running.

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
| vitest src/app/shell + src/app/data | 406/406 on a8b4ab8a |
| tsc --noEmit | clean on a8b4ab8a |
| Targeted go tests (commissioner lineup/recorded, store/history NativeEvidence/Recorded, fantasyscore) | pass |
| make build | pass |
| make verify (full) | NOT run; once before the push |
| native_forks ratchet (19) | not lowered |
| Push / PR / Christopher's live look | not done (screenshots only) |

## 4. Artifacts
- Commits on session/ui-to-mocks: db668c16 Pulse slice 1; a75661a5 one rail; a9e364dc unusable stats id + recorded
  weeks parse; 05110385 pre-cutover lineups + recorded starters + slate/back button/matchup box; a8b4ab8a schedule
  drawer; then resume commits.
- Beelink ~/Downloads: twr-pulse-fresh-week{8,9}-1109.png, twr-pulse-fresh-week8-box-1109.png,
  twr-pulse-schedule-{league,myteam}-1109.png, twr-rail-{pulse,hq,control}-1109.png.

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
- 11-09: plan confirmed, Pulse first; clear pathway then execute; precise outputs; stay true to the mocks.
- 11-09 night: direction is good; excellence, lean and clean code; fix the lineup backfill then keep going on Pulse.
- My calls (flag if he objects): >16 starters keep the single list with tight rows; recorded weeks show app-scored
  starter points beside the untouched recorded total; schedule grid shows last week through the next ones.

## 8. Ledger state
Everything committed (+ this resume). Not pushed. Beelink repo untouched. Untracked: only .golangci-cache/.

## 9. PLAN for the rest of this session (in order)
1. Starter stat line. Read how NativePlayerScore is built (commissioner PlayerScore source, fantasyscore StatLine
   Quantities); add a short stat line per starter to StarterScore in results.read (one field, built from the
   quantities already scored, no new fetch); parse it in nativePulse.ts; show it as the mock's `.ln` under each
   starter and as LINE columns in the matchup box. Go test + vitest case. Mock lines 1236-1240 and 1323-1330.
2. Race drawers from reads already served: BRACKET (bracket.read is parsed already) and TIEBREAKERS (check what
   standings.read says about ties first). Pulls on the Race tab. Mock DRAWERS.race, lines 1345-1360.
3. STAT CHANGES drawer: score changes are already in results.read (`changes`, currently dropped by the parser);
   line revisions exist in fantasyscore/source. Show first held vs latest. Mock line 1340.
4. One screenshot pass of Pulse (Live wk 9, wk 8, Race, each drawer) against the mock; fix what differs.
5. Batch point: make verify; bundle to the Beelink; push; PR; launch on :0 for Christopher's real-screen look
   (his display; never claim what he sees). Merge only after he confirms.
6. Deferred, needs his word or new backend: WEEK STORY (generated text), NFL BOX SCORE, PRODUCTION, ARCHIVE tab
   (history/eras/trophies). List what each needs and ask him to order them.
7. After Pulse: War Room trade desk (v6), Franchise HQ (v10), Home (v5), Control Room (v1); then Part C.
Stop rule: if a step needs a new ledger event, a schema change or a permission change, stop and tell him first.

## 10. Findings log (evidence seen; report, do not hunt)
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
Rail matches the mock. Pulse is roughly half built: Live works on real data for live and past weeks with two of
five drawers; Race has no drawers; Archive is not started; starter stat lines are missing. Nothing is pushed, the
full suite has not run on these commits, and Christopher has only seen screenshots.
