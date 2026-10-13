# Ring 3 landing RESUME — 2026-10-12 20:15 CT (updated) (C-03 merged #19, live-game fix merged #20, 6 phases in flight)

## What we are doing
Landing rings 0–3 live: walk every built endpoint row on Claude-OS, fix what is wrong, mark `landed`, merge each
fix when tested (Christopher pre-authorised: "When something is fixed and tested we merge it").
Main = 106bca6 (PR #19 C-03 + 70 rows landed; PR #20 live-game parser faults + clock test pin).

## Christopher's rulings (do not relitigate)
- League dates: rulebook base dates, commissioner-editable at will (built, #19).
- Keep moving as a normal build; merge when fixed and tested.
- Engine points are the accurate value: never cut or replace with Lab; only improve accuracy (memory
  warroom-engine-points-outrank-lab-points).
- Christopher 10-12 ~19:00: "work through the Endpoint Index systematically; ask me if you need me".

## In flight (worktrees under ~/work/TheWarRoom/.worktrees/, run dir RUN=~/fleet/runs/warroom-ring3-2026-10-11)
| phase | worktree / branch | state | next |
|---|---|---|---|
| land-home | ring-3-land / session/ring-3-land-home | Sol running (findings 1,2,3,4,7,8,11) | review, mutant, drop Sol's week_app_test patch if any, merge main, gate, live, PR |
| land-hq | ring-3-land-hq | gate green; live-checked 20:05 (all 8 pass); review fix 955490a (Escape, inferred wording); push running (/tmp/hq-push.log) | PR, squash-merge (after pool) |
| land-pool | ring-3-land-pool | gate green; live-checked 20:00 (pass); push running (/tmp/pool-push.log) | PR, squash-merge; after both merge, build main, deploy, sort pool by PPG to confirm scored FAs show |
| land-engine | ring-3-land-engine | Sol dispatched 19:50 (par/land-engine.md: engine reruns itself + values free agents, scoring unchanged) | review: rostered points byte-identical proof is mandatory |
| land-pulse | ring-3-land-pulse | Sol dispatched 19:50 (findings 27-30,32,34,39,40,41,43,45) | review, live |
| land-shell | ring-3-land-shell | Sol dispatched 19:50 (findings 35,36,38,42,44) | review, live |
Liveness: mtime of RUN/<phase>/sessions/*.jsonl; done = RUN/<phase>/sentinel; report RUN/<phase>/REPORT.md.
Sol's phases all hit the wall-clock clock test; main's fix (App.clockNow) wins: `git checkout HEAD -- week_app_test.go`
before committing Sol's work, then merge origin/main.

## Sol quota (20:07)
Sol hit "Codex error: The usage limit has been reached" at 20:07; engine/pulse/shell exited report=NO (transcripts intact);
land-home was mid-gate and will stop at its next model call. resume-at-reset.sh (nohup, log resume-at-reset.log) waits
until 20:53 and runs resume-wt.sh <phase> (same --session-id, "continue where you stopped") for each phase whose
sentinel says report=NO. Check RUN/<phase>.resume.log and the sentinel afterwards.

## Background gates running at compaction (19:58) — DONE, both green
- land-pool verify+build: log /tmp/pool-verify.log, /tmp/pool-build.log; result line "VERIFY=n / BUILD=n" in the
  task output (task baomqn7u2). If the process is gone with no BUILD= line, rerun the gate in ring-3-land-pool.
- land-hq verify+build: /tmp/hq-verify.log, /tmp/hq-build.log (task by43s9atx). Same rule.
Do not edit those two worktrees until their gates finish.

## Landed so far
Registry on main: 70 landed. RUN/land-landed.tsv has more since (Control Room League/Data/App, Now reports, archive,
counterparties, comms, command bar, status strip, M-001/X-01/M-016/M-072) — apply in the next registry commit
(same script as #19: replace placeholder ring_note else append).
Still to walk: X-02/X-03 (need a play by Coleman/Shakir/Kincaid; none by 47 plays), L-28/L-29/L-36 (team panel in
Inspector; open L-27 then scroll Inspector), L-30..32 offers, Draft lab L-15..L-22 and M-065/M-087/M-176 (after
land-pool), HQ rows (after land-hq), Home alert tray/digests (after land-home), S-10 filter persistence, S-06 (finding 37),
pools (product call), M-114 (finding 43).

## Findings
RUN/land-findings.md, 1–45. 22 → ring 4 (Offers history reads the frozen Lab doc). Finding 33 corrected: the league
never ran an MFL pool (every season's pool file empty; survivor "League Does Not Use Survivor Pool").

## Questions for Christopher (product calls; ask as a matrix)
1. Finding 16: next-week bye shortfalls (K wk6, QB wk8: 0 available) in the Alert tray, or only on HQ?
2. Finding 33: retire the six pool rows (M-111/112/113/174/200/201) like Victory Points, or keep dormant?
3. Finding 37: should density (N/T/M) change tables, or say it applies to cards only?

## Process note
19:04 I ran `pkill -f` with a pattern in my own command line (killed only my own shell). Use TaskStop for my
background tasks; never pkill -f.

## Environment
Gate env: export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOMEMLIMIT=1500MiB GOMAXPROCS=8
GOFLAGS="-mod=readonly -buildvcs=false" VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1; make … VERSION=dev COMMIT=ring3
BUILD_DATE=ring3. Deploy: gzip build/bin/thewarroom → scp claudeos:warroom-ring1/thewarroom.new.gz (retry on
"Connection reset") → gunzip, chmod → bash deploy.sh <tag>; wait for "value model fitted" after the last "starting dev".
Claude-OS is 1920x1080 since 10-12 20:03 (Christopher; Calendar link now (38,979)). Helpers: RUN/shot.sh, click.sh, open-row.sh <ID> [wait]. Scroll the page with the mouse over x=905 (the
pane scrollbar), not over tables (nested scroll). Density buttons N/T/M at (839/861/883, 86) with the Inspector at
rest; the Inspector expand/collapse toggle at ~(1130–1141, 86).
