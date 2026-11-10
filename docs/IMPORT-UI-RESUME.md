# TheWarRoom — Import + UI-to-mocks RESUME (compact-safe #2, 2026-11-09 ~20:45 CT)

## 0. FIRST THING AFTER COMPACT (Christopher's ask)
"After compact, show me Pulse weeks 8 and 9 when lineups land." Do IN-FLIGHT below, then NEXT ACTIONS 1.

## IN-FLIGHT (check first)
- The app is RUNNING on Claude-OS, display :5, profile ~/scratch/ui-profile2 (the FRESH import), relaunched ~20:37 CT
  to trigger the lineup follow-up sync. Alive check: `ssh claudeos 'pgrep -x thewarroom'`.
- A background watcher (task bznla9ziq, up to 18 min) waits for lineup events. It may die with the compaction; it is
  only a watcher. Re-check by hand:
  `ssh claudeos 'sqlite3 "file:$HOME/scratch/ui-profile2/config/TheWarRoom/history.db?mode=ro" "select kind,count(*) from league_event where league like \"native.mfl-%\" group by kind order by 2 desc"'`
  and `grep -i "lineup\|recorded results" ~/scratch/ui-profile2/desktop.log | grep -v trades.read | tail`.
- How lineups arrive (read in code, internal/app/mflref/migration/schedule_sync.go): ScheduleSync.Sync runs on a
  refresh; current-week starters at once (syncCurrentWeekLineups, MFL live scoring), played weeks after a 5-minute
  launch delay with a 10-minute bound (syncPlayedLineups -> BackfillLineups), ONLY if the league "follows" MFL
  (mflsource.FollowingLeague). UNPROVEN: that the freshly imported league counts as following. If no lineup events
  by ~20:55 CT, read the log lines above and FollowingLeague / rules.Transactions.Follow before guessing.

## 1. What we are doing
Build the five screens to the locked mocks on the NATIVE league, League Pulse first, import read folded into each
screen; then the lab server serves the app. Phone layout is out. Christopher 11-09 night: "most unique fantasy
football experience ever; act with purpose and conviction"; "moving in the right direction".
- Handoff digested: Beelink ~/Downloads/TheWarRoom-IMPORT-UI-HANDOFF.md. North Star 2.1 (repo North_Star.md).
- Work on Claude-OS (`ssh claudeos`, always `< /dev/null`): worktree ~/work/TheWarRoom/.worktrees/ui, branch
  `session/ui-to-mocks` off main 8899a8bf. Claude-OS has NO git remote. To push: bundle the branch to the Beelink,
  fetch into a NEW worktree under ~/work/TheWarRoom/.worktrees/, push from there (pre-push runs the full suite,
  ~10 min, GOTOOLCHAIN=go1.26.10). Squash merge only, after Christopher's live look.
- Env: export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.10 GOMEMLIMIT=1500MiB
  GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false" GOLANGCI_LINT_CACHE=$PWD/.golangci-cache
  VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1. Commit: -c user.name=secureprospective
  -c user.email=secureprospective@users.noreply.github.com, message from a file (-F), trailer
  Co-Authored-By: Claude Fable 5.2 <noreply@anthropic.com> + Claude-Session line.

## 2. Harness
I do the work myself. Edits: write files/python replace scripts in the scratchpad, scp to claudeos:/tmp, run; remove
them BY NAME afterwards (a wildcard rm in /tmp is refused by the permission classifier).
- `make build` (~20-40 s). `UIP=ui-profile2 sh ~/scratch/ui-desk.sh` launches on a profile (default ui-profile);
  starts Xvfb :5 if absent. Wait ~125 s. Then W=$(xdotool search --onlyvisible --name "The War Room"|head -1);
  xdotool windowmove $W 0 0 windowsize $W 1920 1080; `scrot -o`. Stop: kill $(pgrep -x thewarroom).
- Profiles: ui-profile = OLD 11-01 seed league (+2 schedule events from my re-sync). ui-profile2 = clean
  pre-import history (~/f2click/pre-import/history.db) + FRESH import of MFL 14432 done 20:27 CT. Both are real
  league data: stay on Claude-OS, delete when the job ends.
- NEW RAIL click points (app league, 1920x1080): node rows are ~34 px: Home (86,168) War Room (86,201)
  Franchise HQ (86,235) League Pulse (86,268) Control Room (86,302) when no node above is open; an open node pushes
  later rows down by its leaves (~27 px each). Pulse prev-week arrow (260,72). Always screenshot before trusting.
- Old plain layout (no app league selected): Control Room (86,322), League tab (399,22).
- Import form by keyboard: number field click+type; select = Tab then Down/Return; datetime = click, ONE Escape
  (closes the calendar popup), then type each segment followed by Right (03 Right 01 Right 2027 Right 12 Right 00
  Right P). In the Control Room tray a second Escape closes the whole tray.
- Show Christopher: scp to Beelink ~/Downloads, give file:// path.

## 3. Gates
| Gate | State |
|---|---|
| Shell vitest (src/app/shell) | 263/263 on a75661a5 |
| tsc --noEmit | clean on a75661a5 |
| go test leaguemigration/..., leagueload | pass on db668c16 (no Go change since) |
| make build | pass |
| make verify (full) | NOT run; once before the push |
| native_forks ratchet (19) | not lowered |
| Push / PR / Christopher's live look | not done (he has seen screenshots only) |

## 4. Artifacts
- Commits on session/ui-to-mocks: db668c16 Pulse slice 1 + division names; 7879948c resume; a75661a5 ONE RAIL
  (ShellRail.tsx, rail.css, TargetApp.tsx, AppLeagueMount.tsx RailWeek, pulseReads.ts readRailNow/RailNow).
- Beelink ~/Downloads: twr-rail-{pulse,hq,control}-1109.png, twr-pulse-{built,mock}-1109.png,
  twr-pulse-slice1-week{8,9}.png.

## 5. Open question (not a bug yet)
Do starters land on a freshly imported league? See IN-FLIGHT. Fresh import itself: 3,380 events (create, seed_held,
5 rules.set, schedule.generate/publish, 2,884 correct.roster, 487 correct.pick); 32 franchises in 8 NAMED divisions
(fix proven); 1,462 players; 480 picks; 311 adjustments; recorded team results weeks 1-8 (256 rows), cutoverWeek 9;
NO lineup events at import time.

## 6. Established: do not redo
- Importing an MFL league that already exists in the profile is a RE-SYNC (schedule, + transactions if follow is on),
  not a fresh import (service.go ImportWithOptions). League id is fixed: native.mfl-<id>-<season>.
- The ledger refuses deletes (league_event_no_delete trigger, audit chain). Never remove a league by hand; use a
  clean profile instead.
- Desktop mounts the mock frames for an app league via AppLeagueMount; no browser transport yet (Part C); twr CLI
  needs a server; CT 256 is the phone-test league, not for UI work.
- For an app league the old rail's Calendar/Reports/Status opened empty panels, and there is no command bar; the new
  rail drops those buttons and shows "/ command" as text like the mocks.
- Mock CSS for Pulse is already ported in pulse.css; HQShell.tsx and WarRoom.tsx still carry their own copies of the
  rail for the non-embedded (MFL-synced) layout: collapse into ShellRail later, lower native_forks then.
- Prepare works with no API key. 5 waiting settings: Trade review seats (number), Shared date boundary (Start/End of
  day), Season rollover instant (datetime), Trade deadline (datetime), Draft order tie breaker (text).

## 7. Decisions (Christopher)
- 11-09: plan confirmed, Pulse first; clear pathway then execute; precise outputs; stay true to the mocks.
- 11-09 night: rail then fresh import (done); after compact show Pulse weeks 8 and 9 when lineups land.
- My calls (flag if he objects): >16 starters keep the single list with tight rows; War Room and Home have no rail
  leaves yet (War Room tabs stay in the tab row until its screen step).

## 8. Ledger state
Committed through a75661a5 (+ this resume). Uncommitted: nothing. Not pushed. Beelink repo untouched.

## 9. Next actions, in order
1. Check lineups landed on ui-profile2 (IN-FLIGHT). Then open League Pulse on the fresh league: the app starts on the
   MFL-synced league, so first Switch league to the app league (old rail "Switch league" (96,72) -> pick "Legacy NFL"
   app entry), click League Pulse, shoot week 9, press prev-week, shoot week 8. scp both to Beelink ~/Downloads as
   twr-pulse-fresh-week{8,9}-1109.png and give Christopher the file:// paths with a plain read of what is and is not
   connected.
2. If starters never land: read FollowingLeague + the log, report the evidenced cause; fix only if it is the import
   path (leaguemigration), with a test.
3. Pulse remainder: Archive tab; bottom drawers; followed-game chips row; slate words; "MY MATCHUP ‹" back button.
4. Readings the league does not serve yet (catalogue reads, replay proof on lab before merge): starter stat line;
   club + game state in results.read; points against, all-play, streak, points by week, last 5, head to head,
   history; score source label.
5. Batch point: make verify, bundle to Beelink, push, PR, show Christopher the real screen on :0 (his display).
6. War Room trade desk (v6), Franchise HQ (v10), Home (v5), Control Room (v1); then Part C.

## 10. Findings log (evidence seen; report, do not hunt)
- Before any import a new user sits in the OLD plain layout (rail, Control Room App/League pages), not the mock shell.
- Import form: datetime fields pop a calendar over the field; Esc in the Control Room tray closes the whole tray.
- Control Room League tab clips its panels when the import tray opens.
- Stat feed writes ledger events every launch (16.7k commish.stat.queue.approve on the old league).
- Launch ~110-150 s at high CPU, 1.1-2.6 GB RSS before screens fill (budget is 200 ms a screen).
- Log spam: trades.read "you may act only for your own franchise".
- Standings/franchise panel: record 0-0 and "0 games" beside 480.1 points for (old league).
- Imported lineups are stored sorted by player id; screens order by position themselves.
- RailWeek and Pulse each run usePulseLive(''): two NFL scoreboard pollers once Pulse is open (mine; fold into one).
- With no franchise chosen RailWeek still calls week.read with an empty franchise (mine; guard it).

## 11. Honest status
Rail matches the mock and is committed. Pulse is about a third of the way: centre column and franchise panel in form;
archive, drawers and several readings missing. Whether a fresh import shows real starters and scores on Pulse weeks
8 and 9 is NOT yet seen. Nothing is pushed; Christopher has only seen screenshots.

## 12. Update 11-09 ~21:05 CT (after compact #2)
- Starters DO land on a fresh import: week 9 at once (32), weeks 1-8 after the 5-minute delay (235 of 256).
- Commit a9e364dc: (1) a pre-GSIS id in the gsis slot (3 players: HAR447512, JON157260, THO581952) stopped
  results/standings/odds for the whole league; now reads as "no statistics identity" for that player (isGSIS in
  held.go, test in native_identity_test.go). (2) nativePulse.ts parser accepts recordedResult on imported weeks
  (nativePulse.test.ts). Running binary predates the lint-only rework of (1); same behaviour.
- Shown: Beelink ~/Downloads/twr-pulse-fresh-week{8,9}-1109.png.
- OPEN (evidenced): 21 played-week lineups refused: "X is not on this franchise's active roster, so cannot
  start" - the backfill checks past starters against TODAY's roster. Week 8 lacks franchise.18, so that matchup
  shows scores but no starters. Fix belongs in the lineup backfill (roster as of that week, or record as held).
- OPEN: week 9 rows read IN PLAY / "0 of 16 with a result" while every starter says FINAL; week 8 header says
  "scores: source not reported"; power/odds blank on week 8.

## 13. Update 11-09 ~22:15 CT
- Commits after a9e364dc: 05110385 (pre-cutover lineup correction skips the roster check -> 288/288 lineups;
  RecordedStarters lists pre-cutover starters with points in results.read only, team total untouched; slate words;
  MY MATCHUP back button; MATCHUP BOX pull + bottom drawer) and the SCHEDULE drawer commit (PulseSchedule.tsx,
  parsePulseSchedule, readPulseSeason, schedule.read line in localWorkflow.ts).
- Section 12 OPEN items are closed, except: week 8 starter points sum to ~135.3 vs 134.8 recorded (app rules vs
  MFL's own scoring; the recorded total rules).
- App-league reads route through shell/localWorkflow.ts (direct bindings) or TargetWorkflow; a read missing from
  both says "this league action is not available here".
- Drawer click points (1920x1080): MATCHUP BOX pull (300,1062), SCHEDULE pull (425,1062); schedule dtab MY TEAM
  (630,436) league view / (630,581) team view.
- Pulse still to build: WEEK STORY, STAT CHANGES, NFL BOX SCORE drawers; Race drawers (TIEBREAKERS, BRACKET,
  PRODUCTION); ARCHIVE tab; starter stat lines; these need readings the league does not serve yet (section 9.4).
- Shown: Beelink ~/Downloads/twr-pulse-fresh-week{8,9}-1109.png, -week8-box-, twr-pulse-schedule-{league,myteam}-1109.png
