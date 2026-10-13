# Ring 4 build RESUME — 2026-10-13 17:08 CT (main 1f17d24 after #36)

History of this session's earlier updates: RESUME.prev-1708.md (same dir).

## 1. What we are doing
Ring 4 = TheWarRoom commissioner core (spec rev 4 §16; census groups A–E + G minus CM-055, plus M-093/M-095/
M-024/M-156/M-157/M-137). Christopher 15:35: "lean on Sol 6.1 medium on Bee for heavy lifting, run sessions in
parallel when it makes sense; don't sit idle: prep, cleanup, testing on Claude-OS". He asked for chunks 2 and 3
in parallel with 1. Repo Beelink ~/work/TheWarRoom (192.168.1.191, user chris). Main checkout =
session/league-history-pwa = Christopher's Lab: NEVER touch/commit there. RUN = ~/fleet/runs/warroom-ring4-2026-10-13.

## 2. Agents / harness
- Sol = pi gpt-6.1-sol medium. Dispatch: `cd RUN; cat par/common.md par/<x>.md > par/<x>.full.md;
  WT=<worktree> nohup setsid ./dispatch-wt.sh <phase> par/<x>.full.md > <phase>.launch.log 2>&1 < /dev/null &`
  Done = `<phase>/sentinel`; alive = mtime of `<phase>/sessions/*.jsonl`; report `<phase>/REPORT.md`.
  Sol never runs git. Copy `frontend/node_modules` into each new worktree (cp -a from another ring-4 worktree).
- Claude reviews every report: read core code, plant ≥1 own mutant, commit on the phase branch with -F msg file
  (Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>), never --no-verify.
- Gate env: `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOMEMLIMIT=1500MiB GOMAXPROCS=8
  GOFLAGS="-mod=readonly -buildvcs=false" VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1`;
  `timeout 900 nice -n 10 make verify VERSION=dev COMMIT=ring4 BUILD_DATE=ring4` and `make build …` (run_in_background).
- Push runs pre-push verify (~8 min). PR body ends with "🤖 Generated with [Claude Code](https://claude.com/claude-code)".
  Merge only on Christopher's go: `~/.local/bin/gh pr merge <n> --squash`.

## 3. In flight (check first)
| Phase | Started | Worktree / branch | Brief | What to do with result |
|---|---|---|---|---|
| c-int (Sol) | 17:03 | .worktrees/ring-4-integ, session/ring-4-integ (278534a) | par/c-int.md | review; then dispatch c-int2 (§5) on same worktree |
| c7a-scorer (Sol) | 16:58 | .worktrees/ring-4-c7a-scorer, from session/ring-4-c2-rules 86b3365 | par/c7a-scorer.md | review; Part 1 may stop if no held stat source; golden vs MFL points match rate |
Check: `cat RUN/{c-int,c7a-scorer}/sentinel`; `ls -l --time-style=+%H:%M RUN/*/sessions/*.jsonl`.
Recovery if killed: transcripts in `RUN/<phase>/sessions/*.jsonl`. Background watchers in the Claude session
may not survive compaction; just poll.

## 4. Committed (local branches, not pushed unless stated)
| Branch | Tip | Content | Review |
|---|---|---|---|
| main | 1f17d24 (#36, merged) | spec rev 4 §16, census doc, CM-001..056 registry + governs | signed off by Christopher |
| session/ring-4-c1-spine | f608299 | internal/commissioner, history/commissioner.go (league_event), commissioner_app.go, lazy LeagueLedger; bundle 312,290 B | my mutant (public reversal of private change) caught; Sol 15 |
| session/ring-4-c2-rules | 86b3365 | internal/leaguerules + Legacy preset golden | my fixes: missed FG split at 30 yd (decode OQ-002); Legacy bids UFA+RFA open; mutant (drop Out from IR) caught |
| session/ring-4-c3-people | 080e5e7 (2ab8afc + c3b) | internal/governance + elections | my mutants: floor threshold, unseated voter, both caught |
| session/ring-4-integ | 278534a + this RESUME commit | c1 + merge c2 + merge c3 + spec §16.3 rulings | c-int running on it |
Full make verify not yet run after my c2 edits (package + race + hooks passed): it runs on integ.

## 5. Current gap
- No UI can START a commissioner action (c-int adds reads only). Next phase c-int2 (brief to write): minimal
  lazy G3 forms under Control Room › League: "Start an app league" (name, preset legacy_nfl|empty, typed reason),
  "Change a rule" (field picker from leaguerules registered fields, typed value, effective next_rollover|now,
  reason) and "Give a role" (CC/DOT), via commissioner.execute, for the live walk.
- Stat source for native scoring unknown until c7a Part 1 reports.

## 6. Refuted / do not redo
- docs/ui/registry/test_normalize.py was already failing on main and is in no gate (Sol fixed it in K5).
- Survey (k6) file:line claims spot-checked 5/5 true; don't re-survey.
- "Missed FG boundary unspecified" is wrong: decode doc OQ-002 gives 0-29 = -3, 30+ = -1.
- Inspector dead cap/form already wired in ring 3 (not ring 5 work).
- A watcher "failed exit 1" earlier was only `cat` of a missing sentinel.

## 7. Decisions (Christopher, 2026-10-13; recorded in spec §16)
- K4/K5 signed off and merged (#36). "Sign off, merge, start chunk 1"; "run chunks 2 and 3 in parallel too".
- Authority: "dont extend to much into wiring into mfl for now, as its not the end goal" → native league,
  app-owned ledger, one-time seed allowed, no sync/cut-over (§16.2).
- At most ONE co-commissioner. Election: majority of ALL franchises (17/32), else runoff top two (ties in),
  most votes; runoff tie → CC (if not candidate) → DOT majority → "Needs a ruling"; 7-day periods (Claude default).
- Legacy bid modes: UFA open, RFA open; UDFA follows UFA.
- Delete ring 3 caches: yes (done 15:50).
- Engineering (Claude): CM rows in lazy commissioner.gen.ts; later rows ring 6; ledger = history.db league_event;
  identity = opaque local principal, remote-owner acts refused on desktop; rules in pure internal/leaguerules;
  governance pure; c1 Authorizer port implemented by governance in c-int.

## 8. Ledger / housekeeping
- Logged in ~/MOVED.md, ~/archive/MANIFEST.md (+ CT105 backbone 15:50, 16:20): ring 3 caches (15:50), K4
  worktrees removed (16:20), ring 4 finished-phase caches ~50 GB (17:07).
- Hermes T434 updated (ring 4 opened). Filing gate passes.
- Old unproven worktrees ring-0, ring-2-lab, ring-3-b, ring-3-c: leave.

## 9. Next actions, in order
1. When c-int finishes: read REPORT, review the adapter (governance as Authorizer; census lookup, not caller),
   plant a mutant (e.g. let CC through a C-only control), commit on ring-4-integ.
2. Write par/c-int2.md (§5) and dispatch on ring-4-integ; review + mutant + commit.
3. Full make verify + make build on ring-4-integ; deploy to Claude-OS: gzip build/bin/thewarroom, scp to
   claudeos:~/warroom-ring1/thewarroom.new.gz, gunzip, chmod +x, `bash ~/warroom-ring1/deploy.sh <tag>`, wait
   "value model fitted" in ~/warroom-ring1/run.log.
4. Live walk on Claude-OS (1920×1080; Control Room nav (87,384), League tab (528,86); RUN/shot.sh, click.sh):
   start "Walk test 10-13" legacy_nfl; set roster limits next_rollover (pending shown) and now (reason, in
   force); revert; grant a co-commissioner; read ledger and People. Screenshots RUN/shots/r4-*.png.
5. Show Christopher (target ~19:30), ask for a PR/merge go. Then chunk 4 (shell, jobs strip, trays, hold-to-fire).
6. When c7a finishes: review; decide native stat source; plan chunk 7b.

## 10. Environment
- CT105 `ssh root@192.168.1.105`; backbone `/root/.claude/backbone/context.md` append only. Hermes `ssh hermes`,
  `~/work/command-center/bin/cc task show T434`. Claude-OS `ssh claudeos`; screenshots via scp (no rsync).
- Never: open MFL league 14432 / commissioner pages; ssh -n with heredoc; pkill -f over ssh; rsync between machines.

## 11. Honest status
Chunks 1–3 built, reviewed, committed; integration running. Nothing in ring 4 is seen live yet. ETA to
Christopher's first look ~19:30 (c-int ~17:45, c-int2 ~18:30, gate+deploy+walk ~19:15). Scorer unproven.
