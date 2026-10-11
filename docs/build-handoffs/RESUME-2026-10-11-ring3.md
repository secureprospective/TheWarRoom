# RESUME: TheWarRoom ring 3 (Tom = head brain, Sol builds) — compact-safe 2026-10-11 ~12:20 CDT

## 1. WHAT WE ARE DOING
Complete ring 3 of TheWarRoom (Christopher's /goal): build, review, click-through every phase on Claude-OS
myself; Christopher clicks through only once at the very end, then PR squash-merge. Excellence is the standard.
Worktree `~/work/TheWarRoom/.worktrees/ring-3`, branch `session/ring-3` (from main b4b3418; NOT pushed yet).
Main checkout `~/work/TheWarRoom` is on Christopher's `session/league-history-pwa`: never touch it.

## 2. AGENTS + HARNESSES
- Sol (gpt-6.1-sol medium via pi): `~/fleet/runs/warroom-ring3-2026-10-11/dispatch-wt.sh <phase> <brief>`.
  Launch: `cd RUN && nohup setsid ./dispatch-wt.sh <phase> <brief> > <phase>.launch.log 2>&1 < /dev/null &`;
  then background loop waiting for `<phase>/sentinel` (stale = no `<phase>/sessions/*.jsonl` write in 15 min).
  Report = `<phase>/REPORT.md`. **Sol was DOWN until ~12:45 CDT 2026-10-11** (Christopher).
- Sol down → code myself: inline for small fixes, or `tom-dispatch-wt.sh <phase> <brief>` (headless Claude,
  liveness = `<phase>/transcript.jsonl`).
- Briefs: `~/fleet/briefs/warroom-ring3-*.md`. Helpers in RUN: `shot.sh <name>` (Claude-OS screenshot →
  RUN/shots/<name>.png), `click.sh x y`. BACKLOG.md in RUN.

## 3. GATES / STATUS
| Item | State |
|---|---|
| CLAUDE.md ring 3 | done 56071e3 |
| Step 1 registry check (p3a) | done 16e2aa2 |
| Step 2a draft reports (p3b, p3b2) | done, click-verified vs MFL O=17 |
| Step 2b market feeds + shared report shell (p3c, p3c2) | committed 20c693c; ADP format fix UNCOMMITTED (see §8); report-reveal on pool NOT yet click-verified |
| Step 3 League Pulse Race (p3d), Now (p3e) | briefs ready, not run |
| Step 5 pipeline s0/s1/s4 (p3f), s2a (p3g1), s2b+s3 (p3g2) | briefs ready, not run |
| Archive + Pools + league_event (p3h) | brief ready, not run |
| Step 4 headshots (p3k) | brief ready, not run |
| Comms, Control Room League/App, live game view, remaining Acts, Research tab, what-if buttons, BACKLOG | briefs TO WRITE |
| Spec rev 3, registry gate, Christopher click-through, PR | end |
Gate commands (env): `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOMEMLIMIT=1500MiB
GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false" VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1`;
`timeout 600 nice -n 10 make verify VERSION=dev COMMIT=ring3 BUILD_DATE=ring3` and `make build …` exit 0.
Bundle cap 321,818 B; now 288,499 B. Fresh worktree: `make build` before `make verify`.

## 4. ARTIFACTS
- Commits: 56071e3 · 16e2aa2 · 6a1874b · 0062375 · 7987d23 · 20c693c (session/ring-3).
- MFL samples fetched once: RUN/mfl-samples/{projectedScores,playerRanks,adp,aav}.json (copied into
  internal/ingestion/marketfeeds/testdata by p3c).
- Claude-OS: app running label ring3-p3c2-a binary (20c693c-era build); previous binaries kept as
  `~/warroom-ring1/thewarroom.<label>` (ring2-final = ring 2 merge).

## 5. CURRENT STATE OF WORK
ADP/AAV text was "37.71 pick · 2.00–514.00"; fixed in `market_feeds_reading.go` to "Pick 37.7 · picks 2–514"
and "30.1% of auction budget · 3.4%–113.6%", pinned in `market_feeds_test.go` (after `adp.Rows[0]` check),
wire fixtures regenerated. `go test . -run 'Market|Pool'` passes; full `make verify` was started in background
at compaction (log /tmp/r3-verify.log). Report-reveal fix: market reports moved outside the anchored
`<section>` in `frontend/src/app/pool/Pool.tsx`. Leading hypothesis (strong, from code): Pool's
`useWorkAnchor` restored the clicked button's position and undid `revealEndpoint`. Caveat: not yet seen
working on Claude-OS.

## 6. REFUTED — DO NOT RETEST
- "No model run is held" on Steals and busts was not a code bug: the dev build reads `history-dev.db`, which
  had no model run. Fixed by copying run 3 into it (done; persists on Claude-OS).
- p3a verify failure was the hard-coded built-list in `registry/registry.test.ts`, not the CSV.
- First fresh-worktree verify failure = missing `frontend/dist` (embed), not code.
- playerRanks/adp/aav on www47 → "must go to api.myfantasyleague.com"; ADP default period empty in season →
  PERIOD=ALL; IS_KEEPER=D/K invalid or empty.
- Legacy runs no pools (only 2015 one franchise; survivor never set up) — don't build nine pool screens.

## 7. DECISIONS
Christopher: A Research tab = build in ring 3 · B M-192 in ring 3 · E what-if Reset/Undo/Make-this-a-plan →
`act act-text` · F order as listed · CLAUDE.md approved · he clicks through only at the end. C (ProBoards post
format) and D (season rollover) OPEN: ask before step 8.
Tom: registry has no `deferred` status — deferred = ring names the ring + ring_note reason; generator emits
only fields the app reads; shared Report shell with Report.Link; pool Market column; Pools = one report, Acts
→ ring 4; Archive from compiled facts (FS1), reference = Lab Python read via `git -C ~/work/TheWarRoom show
session/league-history-pwa:tools/league-history/...`; M-075 Victory Points retire (Q6).

## 8. LEDGER STATE
Uncommitted in worktree: market_feeds_reading.go, market_feeds_test.go, frontend/src/app/{pool,reports}/
wire.testdata.json (the ADP/AAV format fix). Commit after verify passes:
`git add -A && git commit -F <msg>` (msg: "Market reports: whole-pick ADP ranges, one-decimal AAV shares").

## 9. NEXT ACTIONS, IN ORDER
1. Check /tmp/r3-verify.log (or rerun verify); commit the ADP fix.
2. `make build`, deploy (`gzip -1 -c build/bin/thewarroom > /tmp/twr.gz && scp -q /tmp/twr.gz
   claudeos:warroom-ring1/thewarroom.new.gz && ssh claudeos "cd ~/warroom-ring1 && gunzip -f thewarroom.new.gz &&
   chmod +x thewarroom.new && bash deploy.sh ring3-p3c2-a"`), open War Room (click 76,240) › Valuations and pool
   (435,86), wait 20 s, scroll down, click "Average draft position": the report must scroll into view; read ADP/AAV text.
3. Dispatch p3d Race (Sol if up, else tom-dispatch-wt.sh); while it runs, write the remaining briefs.
4. Then p3e → p3f → p3g1 → p3g2 → p3h → p3k → Comms → Control Room League/App → live game view → ask C/D →
   remaining Acts → Research tab, what-if buttons, BACKLOG, shell verb drift, alerts IR-unavailable.
5. Spec rev 3 + registry gate → Christopher's one click-through → PR squash (`~/.local/bin/gh pr merge <n> --squash`).
Review pattern per phase: REPORT + key diffs, one planted mutant, verify + build, deploy, click every
report/button on Claude-OS, fix or brief a fix round, commit with `git commit -F`.

## 10. ENVIRONMENT NOTES
Claude-OS: `ssh claudeos`, DISPLAY=:0, xdotool/scrot; 1280×800. Dev build DBs in `~/.config/TheWarRoom/*-dev.db`.
Wheel scroll over a report/pool table scrolls the table, not the page: put the mouse over headers (y≈230-260)
to scroll the page. Close any MFL tab opened (`xdotool windowactivate <brave id> key ctrl+w`).
Lines >110 chars fail a frontend test. Wire fixtures: `TWR_WRITE_WIRE=1 go test -run <TestName> .`.

## 11. HONEST STATUS
Roughly a quarter of ring 3 is done (steps 1–2). The heavy parts remain: pipeline port with parity (biggest
risk: Python reference ~180 KB incl. a hand-curated note_readings.csv), League Pulse breadth, live game view
(unlicensed feeds, personal use), and the remaining Acts whose MFL forms were never captured.
