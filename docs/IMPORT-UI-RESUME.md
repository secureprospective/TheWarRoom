# TheWarRoom — Import + UI-to-mocks RESUME (compact-safe, 2026-11-09 ~20:20 CT)

## 1. What we are doing
Build the five screens to the locked mocks on the NATIVE league, League Pulse first, with the import read folded into
each screen (trace only the readings that screen is missing). Then the lab server serves the app. Phone layout is out.
- Handoff (read once, digested): Beelink ~/Downloads/TheWarRoom-IMPORT-UI-HANDOFF.md. North Star 2.1 (repo North_Star.md).
- Work happens on Claude-OS (`ssh claudeos`): worktree ~/work/TheWarRoom/.worktrees/ui, branch `session/ui-to-mocks`
  (off main 8899a8bf). Claude-OS has NO git remote. To push: bundle the branch to the Beelink, fetch into a NEW worktree
  under ~/work/TheWarRoom/.worktrees/ (never touch the main checkout, it is session/league-history-pwa), push from there
  (pre-push hook runs the full suite, ~10 min, GOTOOLCHAIN=go1.26.10). Squash merge only, after Christopher's live look.
- Env for every command: export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.10
  GOMEMLIMIT=1500MiB GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false" GOLANGCI_LINT_CACHE=$PWD/.golangci-cache
  VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1. Commit identity: -c user.name=secureprospective
  -c user.email=secureprospective@users.noreply.github.com.

## 2. Agents / harness
I do this work myself (no dispatch). Edits: write a python replace script + files in scratchpad, scp to claudeos, run.
Test loop on Claude-OS:
- `make build` (~40 s) -> build/bin/thewarroom.
- `~/scratch/ui-desk.sh` starts Xvfb :5 (1920x1080) if absent and launches the build on profile ~/scratch/ui-profile
  (a COPY of ~/f2click/config; real league data, stays on Claude-OS, delete when the job ends).
- Wait ~110 s after launch before screens fill. Then: W=$(xdotool search --onlyvisible --name "The War Room"|head -1);
  xdotool windowmove $W 0 0 windowsize $W 1920 1080; click with `xdotool mousemove X Y click 1`; `scrot -o file.png`.
  Clicks: League Pulse (86,286) · prev week (229,88) · Control Room (86,322) · CR League tab (848,79) ·
  MIGRATE AN MFL LEAGUE (471,816) · expand tray (289,~1000) · league-id field (365,903) · Prepare (648,992).
- Stop the app by PID (`kill $(pgrep -x thewarroom)`), never by pattern. Space launches out (MFL refuses rapid fetches).
- Mock render: chromium --headless=new --no-sandbox --disable-gpu --hide-scrollbars --window-size=1920,1080
  --virtual-time-budget=4000 --screenshot=/tmp/x.png file://.../docs/ui/mocks/<mock>.html
- Show Christopher shots by scp to Beelink ~/Downloads and giving the file:// path.

## 3. Gates / status
| Gate | State |
|---|---|
| Shell vitest (src/app/shell) | 263/263 pass on db668c16 |
| go test leaguemigration/..., leagueload | pass on db668c16 |
| make build | pass |
| commit hooks (golangci, gitleaks, ifaceguard) | pass |
| make verify (full) | NOT run; run once before the push |
| native_forks ratchet (19) | not lowered yet; lower as each screen's forks collapse |
| Push / PR / Christopher's live look | not done |

## 4. Artifacts
- Commit db668c16 on session/ui-to-mocks: Pulse slice 1 + import division names (+ this resume in a later commit).
- Files: frontend/src/app/shell/{PulseNow.tsx,pulseModel.ts,NativeLeaguePulse.tsx,PulseInspector.tsx(new),pulse.css},
  internal/leaguemigration/mflsource/{prepare.go,divisions_test.go(new)}.
- Shots on Beelink ~/Downloads: twr-pulse-{built,mock}-1109.png, twr-pulse-slice1-week{8,9}.png.

## 5. Current problem (none blocking)
Pulse week 9 shows no fantasy scores because the test league (native.mfl-14432-2026 in the scratch profile) is an OLD
11-01 seed (commish.league.seed_held) with lineups only for weeks 1-4 (15/15/18/9 teams) and 8 (31). It does not show
what the CURRENT import (internal/leaguemigration + mflsource, which backfills played lineups and has LiveWeekResult
for the week in play) does. Caveat: not yet proven either way; needs the fresh import.

## 6. Already established: do not redo
- The desktop build DOES mount the mock frame for an app league on all five rail items (AppLeagueMount.tsx); the
  "two layouts" Tom saw is MFL-synced vs app league.
- The frontend has no browser transport yet (only Wails bindings + /desktop/native, /desktop/workflow); a browser/PWA
  view is Part C work, not a quick way to look at screens.
- `twr` CLI needs a server (twr connect); it cannot read the desktop profile.
- A local twr-server stand-up needs TLS + totp key + league-load; twr-practice was cut in the debloat. Not worth it now.
- Lab CT 256 is the phone-test league with claimed franchises: do not use it for UI work.
- The mock's inspector/starters CSS is already ported in pulse.css (.insp .ihead .nums .idr .kv .mgrp .chipc .ball2 ...).
  Wire to it; do not re-port.
- Both sides' lineup.read succeeded for the commissioner session (clubs showed for both franchises).
- Desktop "Move my MFL league" Prepare works with no API key: 32 teams, 1462 players, 480 picks; preset "Legacy NFL";
  5 settings wait: Trade review seats, Shared date boundary, Season rollover instant, Trade deadline, Draft order tie breaker.

## 7. Decisions (Christopher, 11-09)
- Plan confirmed: Pulse first; import read scoped per screen; then rail = one shell; then War Room, HQ, Home, Control Room;
  then server serves the app.
- "In all steps have a clear pathway in the plan then execute; use tokens wisely on precise outputs."
- Stay true to the mocks (proven with players); add what is missing; the UI is what the project is waiting on.
- After compact: do the rail, then the fresh import.
- My call (flag if he objects): leagues starting more than 16 keep the mock's single starters list with the detail
  line beside the name ("tight" rows) so nothing scrolls.

## 8. Ledger state
Committed: db668c16 (+ resume commit). Uncommitted: nothing. Not pushed. Beelink repo untouched.

## 9. Next actions, in order
1. RAIL: read docs/ui/mocks/rail-mock-v1.html and glm-rail-funnel-2026-10-30.md; diff against TargetApp.tsx rail
   (lines ~137-190) for an app league; rebuild to the mock (THE WAR ROOM head with league + week/live line, MY SHORTCUTS,
   numbered nodes 1-5 with open leaves, Pulse live leaf "Leading/Trailing x–y", CUSTOMIZE + / command foot). One shell:
   no plain layout for a native league. Keep "View a franchise" + Switch league + Calendar/Reports/Status reachable.
   Run shell vitest, build, shoot at 1920x1080 next to the mock, commit.
2. FRESH IMPORT on the scratch profile via Control Room > League > Migrate (see clicks). Answer the 5 waiting settings
   with plain test values, reason "UI test import", import. If it refuses because native.mfl-14432-2026 exists, read the
   refusal and report; do not hand-edit the ledger. Then check with sqlite (history.db, table league_event, columns
   league/kind/payload): lineups per week, division names, and look at Pulse weeks 8 and 9.
3. Pulse remainder: Archive tab; bottom drawers (matchup box, week story, schedule, stat changes, NFL box score);
   followed-game chips row (mock shows up to 4); slate words; "MY MATCHUP ‹" back button on other matchups.
4. Readings the league does not serve yet (backend, through the command catalogue, replay proof on lab before merge):
   starter stat line; club + game state inside results.read; points against, all-play, streak, points by week, last 5,
   head to head, history; score source label.
5. Batch point: make verify, bundle to Beelink, push, PR, show Christopher the real Pulse screen.
6. Then War Room trade desk (warroom-mock-v6), Franchise HQ (v10), Home (v5), Control Room (v1); then Part C.

## 10. Findings log (evidence seen; report, do not hunt)
- Stat feed writes ledger events on every launch: 16,7xx commish.stat.queue.approve events; revision climbs each launch.
- Launch: ~110 s at ~360% CPU, 2.6 GB RSS before screens fill (budget is 200 ms a screen).
- Log spam every few seconds: trades.read "you may act only for your own franchise" (signed in franchise.29, viewing Dallas).
- Franchise panel/standings: record 0-0 and "0 games" beside 480.1 points for; tray says "through week 13: 11 games".
- Old seed stored divisions as MFL ids ("00"); fixed in the import for new imports (names, ids kept if names collide).
- Imported lineups are stored sorted by player id, so screens must order by position themselves (Pulse now does).
- Control Room League tab clips its panels when the import tray opens.
- Home on the old league sat on "Reading…" for the first ~40 s.

## 11. Honest status
Pulse is roughly a third of the way to the mock: centre column and franchise panel match in form; rail, archive,
drawers and several readings are missing. Win chance on week 9 is wired but unseen (no odds for that week on the old
seed). Nothing is pushed or seen by Christopher on a real screen beyond the screenshots.
