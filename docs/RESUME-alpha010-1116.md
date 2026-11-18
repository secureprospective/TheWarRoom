# RESUME — TheWarRoom alpha 0.10 + Franchise HQ + OVERNIGHT RUN (Claudebox head brain), Mon 2026-11-16 night
Updated 1:15 PM CT Tue 11-17. On "we are back": do his instruction first, else section 10 item 1.
**OVERNIGHT RUN ACTIVE: `~/fleet/runs/overnight-twr-1116/` (Beelink) — PLAN.md, briefs/, REPORT.md (keep REPORT current; "Needs Christopher" first).**
**MERGE AUTHORITY TONIGHT: he said "review, test if needed, merge - yolo mode" → I merge tonight's lane PRs myself after my review + green CI.**

## 0. STATE AT 7:55 PM CST TUE 11-17 (supersedes every older 0 block and sections 3, 10, 11)

**New rulings since 7:00 PM (memories: feedback_luna_exclusive_scale_thinking, project_twr_alpha_cuts_1118)**
- **Thinking level is routing.** Pick it for wall clock and to keep the session alive on OpenAI's servers, not for output quality. Use high for short lanes, xhigh for long ones, and max only when a long run already got cut off.
- **Alpha is now after Bundles 1 to 5 plus the release checks.** Bundle 6 is removed.
  - Cut from alpha (moved to the tracker's "later" list): usage/age arc, the D58 layout store and arrangeable boards (Home becomes a fixed v3 look), the alert band, the status bar, team/league search, stat changes, the Archive tab, points per play, and board sync.
  - Kept: stacked what-if, the Picks tab, and player search.
- **#320 counts as a pass for release check line 14.**
- **#304/#307 are parked until after alpha.**
- **Head brain owns the testing:** the lab server (CT 260 on lab host 192.168.1.182) and the client on Claude-OS.
- **Tom (Opus low) may be used as GPT fallback.** Launcher: `TOM_WT=<worktree> TOM_BRIEF=../alpha010/<brief> TOM_SCOPE=... launch-tom-build.sh`. Run dir: `~/fleet/runs/tom-<wt>/`. At most 2 at once.
- **Goal:** testers are waiting. Get to alpha fast, then learn from their feedback.

**Head-brain rulings given to lanes**
- **b5perm:** `commish.vote.close` and `commish.vote.tiebreak` stay commissioner-only.
- **b1pulse:** removing a duplicate read in shared local/hosted code is slop and is allowed.
- **b0load:** league-load prepares an empty state dir, plus an installer `TWR_FIRST_LOAD=1` mode.

**Merged:** #344 (offers drawer) into bundle/b1 at 7:53 PM. My fix in it: offer cards name the partner team.

**In flight (all on Beelink; watchers on Claudebox)**
| Job | What | Check | On result |
|---|---|---|---|
| b92bz4ed9 | merge #345 (b1pulse, reviewed twice, green) into bundle/b1 | `gh pr view 345` | update tracker |
| b9b9f6how | merge #346 (b0load fresh-server first load, reviewed) to MAIN | `gh pr view 346` | lab4 relaunches |
| b0oklgyc1 | after #346 MERGED and 02:54Z, relaunch lab4 (Luna max, brief lab4.md with a RE-RUN section) | `~/fleet/runs/pulse-lab4/` | review RESULT.md; file bugs as lanes |
| b8qoa197a | at 02:52Z relaunch b1whatif, b5frame, b5jobs, b5perm on Luna xhigh (stopped on GPT quota at 7:38 PM) | run dirs `~/fleet/runs/pulse-<lane>/` | arm `twr-lane-wait.sh` on them |
| bpvirqlew | Tom on b1hqdec (contract decisions), continuing from its HANDOFF | `~/fleet/runs/tom-pulse-b1hqdec/completion.txt` | review its PR into bundle/b1 |
| b1cleanup | Luna xhigh, still running at 7:50 PM | lane-check | review its PR into bundle/b1 |
| b4q3ixdx4 | 50-min lane check | — | re-arm |

**CT 260 state**
- Server stopped and `/var/lib/twr` empty.
- Empty first-start data archived at `/root/first-start-data.tar.gz`.
- Seed league at `/root/seed/history.db` (sha256 ca2d477a…ff3e, matches the original).
- Lab ssh: `ssh -F ~/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab`.

**Refuted / traps this window**
- The guard hook blocks any command text containing a container power word (it blocked a brief that mentioned one). It also blocks recursive chown/chmod. Use the documented non-recursive `chown twr:twr /var/lib/twr/server.db*`.
- Pushing from a worktree needs the build env (PATH with Go and wails, plus GOTOOLCHAIN). Without it, verify fails with "wails: not found".
- Heredocs over ssh break on apostrophes. Write the file locally and scp it.
- The storm harness only accepts the lab URL on 443, not a LAN IP. The Cloudflare route is accepted.

**Next actions**
1. As each lane finishes, review its diff myself, fix small slop directly, and merge into its bundle branch.
2. When B1 is complete: merge main into bundle/b1, run the full suite, then open and merge the bundle PR.
3. After B5's frame lands, launch the B5 tabs: Rules, Season, Picks and Server on high; Fix, League and People on xhigh.
4. Review lab4's RESULT and turn every bug into a fix lane.
5. Write the tester-ready checklist: installer, an outside-reachable server, and a feedback channel. Then queue Bundles 2 to 4 (reduced) and 8 to 9.

**Needs Christopher:** rotate the CT 257 keys (whenever he's at a keyboard). Also his choice of how testers reach a server (outward-facing).

## 0-prev. (older) STATE AT 7:00 PM CST TUE 11-17 (supersedes 0a/0b and sections 3, 10, 11 below)

**Goal (Christopher 11-17, latest):** build the full v3 desktop app in bundles, push day and night, and get Bundles 8 and 9 (the phone app) done before **Sunday 11-22 kickoff**. Official alpha comes after Bundle 6 (his call). Tracker: `~/Downloads/twr-usable-week-plan.html`. The page has been rebuilt (template `~/fleet/runs/twr-tracker-template.html`, backups `.bak-1117`, `.bak-1117b`). Its data keys: `plan` (bundles B0–B8 + ALPHA, each with items in state done/building/todo/waiting), `lanes` (cards with `day` 11-17, `at`, `pr`, `ord`), `needsYou`, `later`, `slop`, `log`, `today`.

**Bundle method (his order 11-17):** each bundle gets one shared branch, `bundle/bN`. Lanes are launched with `BASE=bundle/bN ~/fleet/bin/launch-pulse-lane.sh luna <lane> <brief> <level>`; BASE support was added today (backup at `launch-pulse-lane.sh.bak-1117`). Each lane opens its PR with `--base bundle/bN`. I review each PR and merge it into the bundle. Then the full suite and CI run once, and the bundle goes to main as one merge. CI runs on every push to any branch. Bundle 1's shared brief rules are in `~/fleet/briefs/alpha010/b1-*.md`, Bundle 5's in `b5-*.md`. Slop gets fixed first, inside each lane.

**Plan order:** B0 (tonight) → B1 Finish HQ + War Room → B5 Control Room on the server app → B2 Home server side → B3 Home screen → B4 Shell → B6 Pulse extras → B7 browser door + PWA → ALPHA → B8–9 phone app built from `~/Downloads/TheWarRoom-mobile-mock-v1.html`. That mock is complete: 55 of 56 desktop jobs reachable in 3 taps or fewer, per `~/fleet/runs/twr-v3-1117/MOBILE-V1-PARITY.md`. It was missed by me earlier; it is not a draft.

**Merged to main today (11-17):** #312 #315–#319 #321 #322 #324–#327 #329–#341 (#339 is the README; #341 is the v3 stub plan, merged 5:45 PM). The last code merge was #340 (recent winning bids) at 6:02 PM. #338 merged after I removed duplicate code: one `ownerPrivateCommand` helper in `internal/commandcatalog/native_reads.go`.

**In flight (Luna, all running at 6:45 PM):**
| Lane | Branch base | Level | Brief |
|---|---|---|---|
| b1hqdec (owner contract decisions: new owner command, approved) | bundle/b1 | max | b1-hqdec2.md |
| b1whatif (stacked what-if server read) | bundle/b1 | xhigh | b1-whatif.md |
| b1offers (War Room offers drawer) | bundle/b1 | xhigh | b1-offers.md |
| b1pulse (My Road + shared form helper) | bundle/b1 | xhigh | b1-pulse.md |
| b1cleanup (remove withdraw on placed bids, S10 fold, v3 doc fix) | bundle/b1 | xhigh | b1-cleanup.md |
| b5frame (thin Control Room frame + Jobs + Situations) | bundle/b5 | xhigh | b5-frame.md |
| b5jobs (missing job producers, server) | bundle/b5 | xhigh | b5-jobs.md |
| b5perm (deputy six-tick gate, ruled) | bundle/b5 | max | b5-perm2.md |

- **Alive check:** `/root/.claude/scripts/twr-lane-check.sh 50` (armed), or the newest file time in `~/fleet/runs/pulse-<lane>/sessions/`.
- **Watcher:** `twr-lane-wait.sh` for all 8 (armed).
- **When a lane is done:** read `stdout.log`; review the diff, especially policy, token and store files; merge its PR into `bundle/bN`; resolve clashes as a union of both sides and regenerate generated files with `go run ./frontend/scripts/nativegen` and `-surface doors` from the repo root; run `make bloat`.
- **GPT quota:** he said to tell stalled lanes to continue. Relaunch the same lane: the worktree and HANDOFF.md resume it.

**Rulings today (don't reopen):**
- Release 9 / schema 16 private table approved.
- Ignore picomatch advisory GHSA-fc5q-39xc-vjpf until patched (T649).
- API tokens may reach private notes, watch, draft queue and trade drafts for the owner's own team only.
- One sign-in = one server; a server holds many leagues; choosing between servers waits for the login screens (memory project_twr_one_signin_one_server).
- Build an owner contract-decision command (ledger event, no table).
- Alpha after Bundle 6.
- Deputies get the v3 six-tick grid: stat fixes, trades, lineups, picks, rules, rollover. Co-commissioner = all six. Roles, succession and deputy grants are never delegable (memory project_twr_deputy_six_ticks).
- The ADP feed is his call; the Plans store and private two-team threads are parked.

**#341 errors found (fix in the docs):**
1. It says the engine allows withdrawing a placed bid. False: `bid_execute.go` always refuses. The app still offers withdraw; b1cleanup removes it.
2. It says `contract.action.set` is the owner decision command. False: it is commissioner-only league policy (CM-009).

Check each #341 plan line against the code before a lane relies on it.

**Open Control Room decisions (ask when their screens come up):**
- Is snooze per person or per league?
- Do deputies see every job?
- Who can see a written request?
- Can divisions be named?

**NEXT, in order:**
1. Review and merge B1/B5 lane PRs into their bundle branches as they finish.
2. Launch the remaining B5 tab lanes once b5frame lands: Rules, Season, Fix, People (uses b5perm), League, Picks, Server read-only.
3. B0: build from main and install on lab CT 260. The lab path and refused import command are in `~/fleet/runs/pulse-lab1/RESULT.md`; the steps are in `docs/lab/STRESS-READY.md`. Then write the one-run import script for him to read and run.
4. When B1 is complete: merge main into bundle/b1, run the full suite, open the bundle PR to main, merge on green.
5. Queue B2 (D58 layout store; needs a migration → release number: ask him) and B3/B4 so Luna never idles.
6. B7 needs his security sign-off: the server refuses browser Origin today, on purpose.

**Needs him:**
- #320 line-14 call
- #304/#307 player values
- rotate CT 257 keys
- run the CT 260 import script (once written)
- B7 security sign-off
- B2 release number

**Honest status:** B8–9 by Sunday means B1–B7 must merge first. That is about 7 bundles in 5 days at today's pace of roughly 23 merges a day. It is possible but tight. B5 is the largest and most likely to slip.

## 0a. (older) STATE AT 1:15 PM CST TUE 11-17 (supersedes the 7:35 AM block and sections 3, 10, 11 below)
**Rulings today:** Luna ONLY (high/xhigh/max). Full send gated by launcher room() (>=4 GB, load <12). Windows 3 h (max 4 h).
50-min checks (`/root/.claude/scripts/twr-lane-check.sh 50`, re-arm). DAY MERGE AUTHORITY: yes, after my review + green CI.
Private-entries table APPROVED; Release 9 / schema 16 APPROVED. API tokens MAY reach notes/watch/draft queue for the
issuing owner's own team only (same rule as bids). Picomatch advisory GHSA-fc5q-39xc-vjpf ignored (#335; Hermes T649 = remove
when patched). Build plan: SKIP 0.10.2; one build from main after #333/#334 land, then a reviewed one-run league-import
script for CT 260 (he runs it; RESULT.md at ~/fleet/runs/pulse-lab1/ is notes, not a command). "Ahead of schedule."
**Merged 11-17:** #322 #324 #325 #326 #327 #329 #330 #331 #332 #335 (+ overnight #312 #314-#319 #321). HQ steps 1-6 done.
War Room hosted: trade Send, Market bids, Market pool, player panel reads, plans-tab empty state.
**In flight:** #334 MERGED 1:10 PM. #333 conflicted in 9 files (duplicate private-table plumbing) -> Luna xhigh lane dqueue Part 3 (1:20 PM, brief ~/fleet/briefs/alpha010/dqueue.md) rebuilds on #334's plumbing. On done: review (token files too), then ~/fleet/bin/twr-merge-pr.sh dqueue 333.
(Claudebox bg task bulbfy2qc). Check: `gh pr view 334/333 --json state`. If not MERGED: rerun the helper (it syncs main,
HANDOFF.md takes main's, waits all checks, squash-merges; exit 2 = real conflict → resolve by hand). No Luna lanes running.
**Merge helper:** `~/fleet/bin/twr-merge-pr.sh <lane|worktree-dir> <pr>`. War Room plan: `~/fleet/runs/pulse-wrplan/REPORT.md`.
**NEXT (he said: continue on the next set of work after compaction):**
1. Confirm #334 + #333 MERGED; close #313, #323, #328 if still open (link #334/#333).
2. Launch Luna xhigh: War Room Market target lists on the watch kind (WR job 3 rest) — brief like wr3.md, reuse player.watch.*.
3. Check if saved trade drafts (WR job 6) fit private_entries (kind 'offer'?) without a migration; if yes launch Luna xhigh, else ask him.
4. Build from main for install (lab build path in ~/fleet/runs/pulse-lab1/RESULT.md line 3 uses LAB-ONLY root fleet/lab-release-ct258 + twr-up); then write the one-run CT 260 league-import script (lines 17-19 + docs/lab/STRESS-READY.md) for him to read and run.
5. Later: WR plans (job 7, may need table), PAIR threads (job 8, privacy ruling), recent wins (job 9), hosted injury feed.
**Needs him:** #320 stress line-14 measurement; F51 #304/#307 GOMEMLIMIT; rotate CT 257 keys; run the import script (after step 4).
**Lessons today:** pgrep -f over ssh matched my own shell and cut it (find exact PID first). Merge helper needed: wait for all checks
(not --watch), treat not-MERGEABLE as needs-sync, accept non-pulse worktree dirs. Luna lanes over-extend scope into token policy;
review internal/accounts + token_policy diffs every time.

## 0b. (older) STATE AT 7:35 AM CST TUE 11-17 (supersedes sections 3, 10, 11 below)
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
