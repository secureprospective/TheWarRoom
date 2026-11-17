# RESUME — TheWarRoom alpha 0.10 + Franchise HQ + OVERNIGHT RUN (Claudebox head brain), Mon 2026-11-16 night
Updated 8:50 PM CT (sixth compaction today). On "we are back": do his instruction first, else section 10 item 1.
**OVERNIGHT RUN ACTIVE: `~/fleet/runs/overnight-twr-1116/` (Beelink) — PLAN.md, briefs/, REPORT.md (keep REPORT current; "Needs Christopher" first).**
**MERGE AUTHORITY TONIGHT: he said "review, test if needed, merge - yolo mode" → I merge tonight's lane PRs myself after my review + green CI.**

## 0. STATE AT 7:35 AM CST TUE 11-17 (supersedes sections 3, 10, 11 below)
**Rulings today:** Luna ONLY (high/xhigh/max by difficulty; restart same brief before rescoping). Full send: no lane cap,
launcher `room()` gates on >=4 GB RAM + load <12. Windows 3 h (max 4 h). Check lanes every 50 min
(`/root/.claude/scripts/twr-lane-check.sh 50` background, re-arm each time). DAY MERGE AUTHORITY: yes, all day, after my
review + green CI (decisions #313, #320, F51 still his). Phone/audio = paragraphs. Never edit a running bash script in place.
**Merged today/overnight:** #312 #314 #315 #316 #317 #318 #319 #321. HQ steps 1,2,4,5 + slop cleanup on main.
**In flight (Beelink lanes, `~/fleet/runs/pulse-<lane>/`):**
- smem part 2 (Luna xhigh, 7:22 AM): PR #322 memory -21.9% but removed shared-ledger clone + its test; must restore
  `TestSharedRecordsCallerCannotChangeAnother`, audit callers, race test. Merge only if those hold.
- hqreserve (Luna xhigh, ~7:30 AM): HQ step 6 IR injury/return + taxi experience. Review, merge.
- wrplan (Luna high, ~7:30 AM): War Room read-only plan -> `~/fleet/runs/pulse-wrplan/REPORT.md`. Then launch its jobs.
- Watchers: done `twr-lane-wait.sh smem hqreserve wrplan`; 50-min checker. If lost after compaction, re-arm both.
**Needs Christopher:** lab1 league import CT 260 (`~/fleet/runs/pulse-lab1/RESULT.md`); notes storage #313; stress line-14
measurement #320; F51 #304/#307 (memory limit now set); rotate CT 257 keys; install 0.10.2.
**Overnight skill v2** written (Claudebox, Tom ~/.claude, pi ~/.pi/agent/skills, Hermes ~/.hermes/skills); Beelink copies of
the watcher scripts in `~/fleet/bin/`. Launcher backup `~/fleet/bin/launch-pulse-lane.sh.bak-1117`.
**Next:** handle pings; after wrplan, launch War Room jobs (Luna); keep Beelink full; tracker + report current.

## 1. WHAT WE ARE DOING
(a) Alpha 0.10: prove the 15 lines of `docs/roadmap/RELEASE-0.10.md` (lab jobs tonight after the game).
(b) NEXT SCREEN = FRANCHISE HQ on real server data (his "hard focus"). Plan on the tracker, 6 steps.
(c) Standing rule: slop from old models gets flagged/rewritten as we go (memory `feedback_twr_active_slop_cleanup`; tracker `slop` list, S04 + RULE).
(d) Fable (his separate session) builds 3 live-animation HTML versions from `~/Downloads/fable-handoff-live-animations.md`.
Repo: Beelink `chris@192.168.1.191:~/work/TheWarRoom` (main checkout is HIS; worktrees only). Lab host 192.168.1.182 (CT 257 live test server, CT 258 lab, CT 110 gateway). From Beelink: `ssh -F ~/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab`. Never touch CT 110, 256. CT 257 read-only for lanes. No lab load until the game is final.

## 2. DONE LOOKS LIKE (overnight PLAN.md)
1. 0.10 lab proof lines 3-5, 13, 14 MET/NOT MET with quoted evidence — pending (after game).
2. Week 10 final vs MFL for 32 teams — pending.
3. HQ step 1 inventory DONE + reviewed (`~/fleet/runs/pulse-hqmap/REPORT.md`); steps 2-4 as PRs merged (yolo) — lanes running.
4. lword + cpu7 PRs reviewed, green, merged — lanes running.
5. REPORT.md + tracker current.
Tonight's stream: MET (live ticks every 10 s, 0 errors; #309 #310 #311 merged).

## 3. IN FLIGHT
| Job | Started | Check | On result |
|---|---|---|---|
| Lane lword (Luna): refusal reason shown in app (client half of #309) | 8:13 PM | `~/fleet/runs/pulse-lword/completion.txt`, stdout.log | review PR, CI green → merge |
| Lane cpu7 (Luna): contracts cache, `rememberNativeContracts` | 8:13 PM | `pulse-cpu7/` | review equality test + before/after table → merge |
| Lane hqreads (Luna): hosted `player.usage.read` + `player.agearc.read` + shared adapters | 8:44 PM | `pulse-hqreads/` | review → merge; then write step-4 brief |
| Lane hqnotes (Luna): hosted private notes read/change, PRIV privacy tests | 8:44 PM | `pulse-hqnotes/` | review privacy tests carefully → merge |
| Watcher b9evudbwl: `twr-lane-wait.sh lword hqreads hqnotes` (cpu7 FINISHED 8:50 PM: review its PR first!) | 8:46 PM | prints first lane done/dead, exits (seen list `/tmp/claude-0/twr-lanes-seen`) | handle, re-arm with remaining lanes |
| Timer b60mlntfd: 10:15 PM | — | — | confirm game final, then section 10 item 2 |
Lane transcripts: `~/fleet/runs/pulse-<lane>/sessions/`, session id in `session-id.txt`. Launch: `ssh -f ... 'nohup setsid ~/fleet/bin/launch-pulse-lane.sh luna <lane> ~/fleet/briefs/alpha010/<lane>.md high >/dev/null 2>&1 &'`.

## 4. RESULTS THIS WINDOW
- Merged: #309 (after HANDOFF conflict fix), #310, #311. Main has #303 #305 #306 #309 #310 #311.
- ESPN study (Sol): `~/fleet/runs/pulse-espn-soak/REPORT.md` (46 shots), recordings `video/go1-1.mp4 go2-1.mp4 test-b.mp4`, frames `video/frames/`. ESPN updates in chunks every 45-65 s w/ full reload; ours flows every 10 s. Fable handoff written.
- Walked hosted app on Claude-OS :5 (thewarroom-main vs CT 257): HQ roster/cap/lineup real; player panel "Not available yet" because it calls desktop Wails bindings (`market/api.ts readAgeArc`, `offers/api.ts readPersonalNotes`, research usage).
- hqmap: 19 HQ rows; only 3 new hosted reads needed (usage, age arc, notes); rest = wire existing `value.read`, `native.cap.check`, `contract.quote`, `calendar.read` into HQ (step 4).
- CT 258 lab keys moved to `~/fleet/lab-release-ct258/` (symlink at old path, logged in ~/MOVED.md).
- Tracker cleaned (72 old lanes removed), HQ plan + slop rule added.

## 5. ARTIFACTS
Briefs `~/fleet/briefs/alpha010/`: lword, cpu7, hqmap, hqreads, hqnotes, lab1, lab2, lab3, f51t, espn. Copies in overnight run dir.
Fable handoff `~/Downloads/fable-handoff-live-animations.md` (8,305 B); mock screenshot `~/scratch/fable-handoff-1116/` (KEEP.md).
Stream app: `~/scratch/twr-mnf-1116/thewarroom-main` (sha c9fa86d8…).

## 6. OPEN BUGS / GAPS
- f51t brief points at `/tmp/f51/rehearsal257.tgz` which is GONE (Beelink /tmp cleared). Fix brief to `~/scratch/f51-1117/rehearsal257.tgz` after making a fresh copy post-game.
- lab1 container restart: guard blocks agents; lane writes command under "Needs Christopher".

## 7. REFUTED — DO NOT RETEST
CT 258 for Pulse · lineup refusals as timing race · preview ignoring kickoff roster · release 9 for a patch · Terra for multi-step jobs · `adb pull -q` (lost a clip: no -q flag) · `pct reboot` text inside a Bash heredoc trips Claudebox hook (use Write tool for such briefs).

## 8. DECISIONS
F51 option A. Release number = contract (#308 closed; older-bundle proof via LAB-ONLY release-7 bundle). Luna high default. Overnight merges: yolo (his words). Lab1 restart: skip, morning. CT 257 data copy for F51: yes after final, delete after. HQ is the next screen; War Room → Home → Control Room after. Fable: 3 versions, buttons Run/Pass/Penalty/FG/TD/Random drive; ESPN = inspiration only; keep our 10 s feed.

## 9. LEDGER
This RESUME: repo copy `docs/RESUME-alpha010-1116.md` on branch `session/a010-docs` (worktree `.worktrees/pulse-a010`), plus `/root/TheWarRoom-RESUME.md`.

## 10. NEXT ACTIONS
1. On watcher ping: review that lane's PR (read the diff myself), check CI, merge (`gh pr merge N --squash --delete-branch` via Beelink `bash -lc`); resolve HANDOFF.md conflicts with `--theirs`(main). Update REPORT.md + tracker. Re-arm watcher.
2. 10:15 PM: confirm game final (CT 257 log / nfl.games). Then: copy CT 257 data → `~/scratch/f51-1117/rehearsal257.tgz` (+KEEP.md), fix f51t brief path, launch lab2, lab1, f51t; lab3 (waits inside for lab2 stress). Arm watcher on them.
3. Week 10 final vs MFL (read-only), into REPORT.md.
4. After hqreads+hqnotes merge: write + launch step-4 brief (one shared hosted player-panel read path; wire value/cap/quote/calendar into HQ; remove desktop calls in hosted branch; slop items from hqmap report).
5. Morning: finish REPORT.md (Needs Christopher first), tracker, task list on Hermes, one-line result to him.

## 11. NEEDS CHRISTOPHER
Lab1 container restart (morning, command from lane). Rotate CT 257 keys. Tue: install 0.10.2 (lines 6-7).

## 12. ENVIRONMENT
gh on Beelink needs `bash -lc` (PATH ~/.local/bin). Remote heredocs inside ssh '…' break on apostrophes: write locally, scp. Tracker edits: python with `twr-tracker.py` module + flock (see scratch tu*.py pattern: load/stamp/save). adb S25 serial 192.168.1.171:35907 (paired).

## 13. HONEST STATUS
0.10: lines 1, 8, 12 met; 3-5, 13, 14 depend on tonight's lab jobs (unproven; stress previously FAILED on 502s). HQ: only inventory done; reads/notes lanes just started. Morning ETA ~6 AM.
