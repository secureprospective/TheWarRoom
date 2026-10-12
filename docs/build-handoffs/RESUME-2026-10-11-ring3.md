## >>> RESUME HERE (written 08:40 CDT 2026-10-12, compaction) <<<
session/ring-3 HEAD 1787c9a (p3g1 COMPLETE: s2 2fdb63c, s3 d4e8a50, s4 83545aa + 943644f). Unpushed.
IN FLIGHT, Sol (3 lanes, started 07:31 / 08:11; liveness = mtime RUN/<p>/sessions/*.jsonl, done = RUN/<p>/sentinel,
report RUN/<p>/REPORT.md; recover reasoning from the session jsonl):
- p3s shell/Home/HQ in worktree ring-3-b (brief RUN/par/p3s.md)
- p3r Research in worktree ring-3-c (brief RUN/par/p3r.md)
- p3g2-s1 Lab compiler + proof 1 in NEW worktree ring-3-d, branch session/ring-3-d (brief RUN/par/p3g2-s1.md).
  p3g2-s2 (proof 2 injection seam, today's Lab, Control Room/source lines) still needs a brief after s1.
On each sentinel: review REPORT, plant one mutant, verify, commit IN THAT WORKTREE (never with Sol's edits half-done),
merge into ring-3 (expect registry CSV / registry.test / bindings / SYSTEM_MAP conflicts: union + make bindings +
gen-endpoints), deploy, look.
IN FLIGHT, Claude: p3n Control Room, UNCOMMITTED in worktree ring-3 (all compile; Go tests pass):
- Go done+tested: internal/ingestion/sitenews (What's New, M-141), internal/ingestion/league/allrules.go (MFL
  scoring-event names), control_room_reading.go + control_room_app.go (TargetLeagueSettings M-091/M-097,
  TargetSiteNews, TargetBylaws M-185 from embedded docs/league-rules/Official_Rulebook.md, TargetAccounting
  M-098/M-197 + TargetOpenMFLAccounting -> https://<host>.myfantasyleague.com/<season>/accounting_report?L=14432,
  evidence claudeos ~/mflmap-runs/bee-mfl-ui-map-2026-10-04/nav.log), app.go field `control controlFeeds`,
  control_room_test.go. Bindings regenerated (make bindings).
- Frontend WRITTEN, NOT YET WIRED/TESTED: frontend/src/app/control/{state.ts,LeagueRoom.tsx,WhatsNew.tsx,control.css}.
  TODO: (a) fix imports: `dispatch` is not in commands/dispatch (find how Comms.tsx gets dispatch); (b) add
  'control.read' command in commands/registry.ts (actions league|news|search|open-accounting -> controlState.choose);
  (c) TargetApp.tsx: lazy LeagueRoom for control/league (replace shell-unwired branch), WhatsNew inside AppSettings;
  (d) S-05: presets set per-node workspaces + an Admin switch (off by default, localStorage like shell/franchise.ts)
  that hides C-09 engine parameters in Data and sources; (e) S-10: persist market filters per workspace in localStorage
  + "Reset filters"; (f) vitest tests (bylaw search, parseBylaws, presets round-trip, filter persistence);
  (g) registry: M-091/M-097/M-185/M-098/M-197/M-141/S-05/S-10 -> built; M-137 (no cache page address in docs or
  captured pages), M-108, M-093, M-095 -> ring=4 + ring_note; regenerate endpoints.gen.ts; registry.test built list;
  (h) verify, build, deploy, look at Control Room App + League; commit.
BACKLOG: live Checked-facts with real results needs Christopher's OK to copy league-archive/raw to Claude-OS (hook
blocks rsync of the work tree; do not route around it).
NEXT after p3n: p3l live game; p3h after p3g2; spec rev 3; registry gate; Christopher click-through; squash-merge PR.

## (superseded 08:12 block)
session/ring-3 HEAD 943644f (all local, unpushed). Since the 07:35 block:
- 83545aa p3g1-s4 (1/2): nine checks as compile assertions; internal/lab/testdata/check-exceptions.csv pins 635 identities
  (generated from the reference exceptions.json; Go finds exactly the same 635); full archive equals checks.json.
- 943644f p3g1-s4 (2/2): manifest.WithCurated (note_readings.csv pinned by sha), facts_app.go compile after import + at
  launch, snapshot_checks table, cache by manifest+compiler+rules, failure keeps last good stale, L-34 Checked facts
  report with real results. Deployed `ring3-p3g1-s4b`; pre-compile screen checked. p3g1 is COMPLETE.
- Live view WITH results needs the league archive on Claude-OS: a hook blocks rsync of the work tree. Ask Christopher
  (BACKLOG). Do NOT route around the hook.
- Sol lanes running since 07:31: p3s (shell/Home/HQ, worktree ring-3-b, RUN/p3s/) and p3r (Research, worktree
  ring-3-c, RUN/p3r/). Background watcher in Claude session; liveness = mtime of RUN/<p>/sessions/*.jsonl; done =
  RUN/<p>/sentinel. On done: review REPORT, plant mutant, verify, commit IN THAT WORKTREE, merge into ring-3.
NEXT ACTIONS IN ORDER:
1. Review + merge p3s and p3r when their sentinels land.
2. p3g2 Lab compiler (brief ~/fleet/briefs/warroom-ring3-p3g2-lab-compiler.md), then p3h archive/pools/events; p3n
   Control Room and p3l live game (Sol lanes when free; worktree briefs go in RUN/par/).
3. Spec rev 3, registry gate, Christopher click-through, PR squash-merge. Report the Pools decision to Christopher.
REVIEW ROUTINE: report -> one planted mutant -> make verify/build -> deploy -> look myself (shot.sh) -> fix -> commit ->
update this RESUME + scp CT105. Never `git add -A` without checking for build outputs.

## (previous block, 22:45 10-11, superseded)
HARD STOP: Claude account logs out 04:00 CDT 10-12; timer task wakes Claude at 03:15 to save. Sol budget OUT until ~00:15
(Christopher 21:45); all chain.sh drivers killed. Claude codes the rest itself (Tom only if asked; shares Claude quota).
BRANCHES (all local, unpushed):
- session/ring-3 (worktree .worktrees/ring-3) HEAD 81e2804: p3f 9248f1a, p3g1 groundwork 1d21564, p3f-fix 13c7de5,
  p3g1-s1 b546720 (internal/facts, 12 relations zero-diff parity), merge of ring-3-b 5fee4be (p3m Comms e540d59, p3k photos
  6b5fc37), p3k Data-and-sources photo row+switch 4d97fd9, untrack root binary 81e2804.
- session/ring-3-c (worktree .worktrees/ring-3-c) HEAD 5cff493: p3t Acts 48f6ea2 (reviewed: mutant on accepting-side
  near-match SURVIVED -> test added; trade_link_guard_test.go added; UI fixes; verify 0) + untrack binary. NOT YET MERGED.
- session/ring-3-b: fully merged into ring-3.
NEXT ACTIONS IN ORDER:
1. cd .worktrees/ring-3 && git merge --no-ff session/ring-3-c. Expected conflicts: SYSTEM_MAP.md (keep both sections),
   docs/ui/endpoint-registry.csv (keep both rows' changes), registry.test.ts built list (union), wailsjs App.d.ts/App.js
   (resolve then `make bindings`), endpoints.gen.ts (regenerate via frontend/scripts/gen-endpoints.mjs). TheWarRoom binary is
   now untracked on both sides (should resolve as deleted). Then make verify + build, deploy, look at Trade Floor › Trade desk
   (builder: pick franchise, select assets, Check both sides: teams stacked, Reset Act shown if stale, roster before/after).
2. Code p3g1-s2 (games/lineups/rosters), -s3 (contracts/cap/picks/txns), -s4 (checks, snapshot, L-34) yourself from briefs
   ~/fleet/briefs/warroom-ring3-p3g1-s[234].md, following internal/facts style; reduced-golden parity in verify; full opt-in
   TWR_ARCHIVE_PARITY=1 TWR_FACTS_ORACLE=RUN/p3g1/oracle-final/facts.jsonl.gz TWR_FACTS_ARCHIVE=<archive raw>.
3. Then p3n Control Room, p3s shell/Home/HQ (+ BACKLOG: app-wide sideways scrollbar strip at bottom of every view), p3r
   Research, p3l live game, p3g2 Lab compiler, p3h archive/pools/events. When Sol is back (~00:15) dispatch with
   ./dispatch-wt.sh (one per worktree; WT= env selects worktree); briefs in ~/fleet/briefs/ (worktree-path copies in RUN/par/).
4. Spec rev 3, registry gate, Christopher click-through, PR squash-merge. Report the Pools decision to Christopher.
REVIEW ROUTINE: report -> one planted mutant -> make verify/build -> deploy -> look myself (screenshots via shot.sh) -> fix ->
commit -> update this RESUME + scp CT105. Never `git add -A` without checking for build outputs (the 36 MB root binary slip).
Luna click tests: ./luna-click.sh (stops early; short checklists only).

# RESUME: TheWarRoom ring 3 (Tom = head brain AND builder via tom-dispatch; Sol is Claudebox's) — compact-safe 2026-10-11 ~15:50 CDT

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
| Step 2b market feeds + shared report shell (p3c, p3c2) | done d7f0456 + f7b8cf2 (shared bar axis, AAV desc, team units); click-verified |
| Step 3 League Pulse Race (p3d) | done 35b9847 + 0185bec; mutant (all-play ties) caught; click-verified |
| Now (p3e) | done 3a94003; mutant (average rounding) caught; fixes: NFL week held by season refresh, my side first, mine glyph |
| p3f pipeline s0/s1/s4 + Data and sources | Tom dispatched 15:41 (watch p3f/sentinel; stale = 30 min) |
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
`act act-text` · F order as listed · CLAUDE.md approved · he clicks through only at the end. 12:50 CDT: C (Q15)
post format kept as is · D (Q16) app rolls over when MFL rolls over · ProBoards = https://legacynfl.boards.net/
(logged in on Claude-OS; app only OPENS it, never posts) · 15:15 M-192 RETIRE (no options in rulebook/MFL data) · 15:55 SOL AVAILABLE AGAIN (Christopher: "you now can use Sol at medium thinking to code with if needed"): dispatch-wt.sh for Sol, tom-dispatch-wt.sh for Tom; one builder in the worktree at a time.
Tom: registry has no `deferred` status — deferred = ring names the ring + ring_note reason; generator emits
only fields the app reads; shared Report shell with Report.Link; pool Market column; Pools = one report, Acts
→ ring 4; Archive from compiled facts (FS1), reference = Lab Python read via `git -C ~/work/TheWarRoom show
session/league-history-pwa:tools/league-history/...`; M-075 Victory Points retire (Q6).

## 8. LEDGER STATE
Uncommitted in worktree: market_feeds_reading.go, market_feeds_test.go, frontend/src/app/{pool,reports}/
wire.testdata.json (the ADP/AAV format fix). Commit after verify passes:
`git add -A && git commit -F <msg>` (msg: "Market reports: whole-pick ADP ranges, one-decimal AAV shares").

## 9. NEXT ACTIONS, IN ORDER
HARD STOP: this Claude account logs out at 04:00 CDT 2026-10-12 (Christopher). Save everything by 03:15 (timer set),
or earlier if the 5-hour usage runs low. Per-stage routine now ends with: update this RESUME, copy to CT105, and commit
the repo copy (docs/build-handoffs/RESUME-2026-10-11-ring3.md) in ring-3 BEFORE touching <stage>/APPROVED.
Sol chains keep running after logout; each waits at its APPROVED gate until Claude returns.
NOW (22:18): SOL BUDGET OUT (Christopher 21:45: Sol at 11%, p3g1-s1 and p3t were its last runs for ~2.5 h).
All chains STOPPED (killed chain.sh drivers; no further Sol stages launch). Claude codes the rest itself.
session/ring-3 HEAD 4d97fd9: p3f-fix 13c7de5, p3g1-s1 b546720 (facts skeleton, 12 relations, zero-diff parity),
merge of ring-3-b 5fee4be (p3m Comms e540d59 + p3k photos 6b5fc37), p3k Data and sources row+switch 4d97fd9.
p3t done on ring-3-c (session/ring-3-c, unreviewed at 22:18): review, mutant, commit there, merge into ring-3.
NEXT (Claude codes): p3g1-s2..s4 (briefs ~/fleet/briefs/warroom-ring3-p3g1-s[234].md), p3n, p3s, p3r, p3l, p3g2, p3h.
Deployed on Claude-OS: ring3-merge-b2. EARLIER (21:02) PARALLEL (Christopher: pick up the pace): worktrees ring-3-b (branch session/ring-3-b: chain p3m → p3k)
and ring-3-c (session/ring-3-c: chain p3t → p3r), both from 1d21564, briefs in RUN/par/. Claude merges each approved
phase into session/ring-3 (merge, regenerate registry/wire on conflict). Main ring-3 chain: p3f-fix → p3g1-s1..s4.
Remaining after: p3g2, p3h (after facts), p3n, p3s, p3l. EARLIER (20:50): p3g1 groundwork COMMITTED 1d21564 (Python oracle via tools/labparity, five gaps approved by file+sha;
the 8.7 MB facts.jsonl.gz is deliberately UNTRACKED; full oracle lives at RUN/p3g1/oracle-final/). Sol did not build
the compiler, so the rest is split into small stages run by ./chain.sh (log chain.log): p3f-fix (FIXES from Luna's
click test, p3f/FIXES.md) → p3g1-s1 (golden restructure + league/player facts) → s2 (games/lineups/rosters) → s3
(contracts/cap/picks/txns) → s4 (checks, snapshot, L-34). After each stage's sentinel the chain WAITS for
RUN/<stage>/APPROVED: review REPORT, verify, commit, then touch APPROVED. Send back: write <stage>/FEEDBACK.md and
touch <stage>/REDO. Then deploy+Luna click (short checklists; see memory luna-briefs...). Then p3g2 onward.
EARLIER (19:50): p3f COMMITTED 9248f1a (Sol finished it; 2 mutants caught; verify+build 0; deployed ring3-p3f-a). Luna click-testing Data and sources (p3f/click/); its defects get fixed AFTER p3g1. p3g1 running on Sol since 19:46 (first try blocked: Sol may not run git, so the Lab reference is exported to RUN/lab-reference/, briefs p3g1/p3g2/p3h updated). p3g1 brief carries the five missing 175 B error pins decision. EARLIER (19:10): p3f Tom hit the 5-hour usage limit at 15:58 (its transcript is in p3f-tom/); Sol resumed it from the unstaged edits with brief warroom-ring3-p3f2-continue.md (alive = mtime of p3f/sessions/*.jsonl). Luna click tests: ./luna-click.sh <phase> <click-brief> (report <phase>/click/CLICK-REPORT.md, done <phase>/click/click.done). Earlier: p3f running (Tom builder, started 15:41; alive = mtime of p3f/transcript.jsonl; done = p3f/sentinel; recover reasoning from that transcript). Next phases may go to Sol (dispatch-wt.sh) or Tom. On its sentinel: review REPORT, plant one mutant, verify+build, deploy,
click Control Room › Data and sources, fix, commit; then dispatch p3g1 (tom-dispatch-wt.sh p3g1 <brief>) and repeat
down the PHASE ORDER. The repo copy of this RESUME is stale (024be27); commit an update when no builder is running.
0. (14:35) PHASE ORDER: p3e (running) → p3f → p3g1 → p3g2 → p3h → p3k → p3m Comms → p3n Control Room →
   p3r Research → p3s shell/Home/HQ/lists/what-if/backlog → p3l live game → p3t Acts (all trade hand-offs = O=05,
   no form capture needed; M-192 retired there). ALL briefs written: ~/fleet/briefs/warroom-ring3-*.md. p3n no
   longer has Data and sources (p3f builds it). Samples: RUN/mfl-samples
   (rss, siteNews, playerProfile, depth_charts_2026_latest.csv, mfl-api-info.html), RUN/live-samples (ESPN
   scoreboard+summaries, Yahoo, Fox). FINDING (corrected 19:40 by p3f's import): 2013-2015 messageBoard/polls
   archive files are MFL auth errors (175 B); 2016-2026 are genuinely empty (54 B). Pools archive files are real (decision holds).
   Items 1-3 below are DONE.
1. RERUN make verify (the background run at compaction overlapped a pre-commit stash, so its result is untrustworthy); then commit the ADP fix.
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
