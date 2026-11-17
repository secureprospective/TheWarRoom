# RESUME — TheWarRoom alpha 0.10 + MNF live stream (Claudebox head brain), Mon 2026-11-16
Updated 7:15 PM CT (fifth compaction today). On "we are back": do his instruction first, else section 10 item 1.
**NOW: Christopher goes LIVE on YouTube at 7:30 PM CT** (kickoff LAC at BAL 7:15). Stream from his Beelink app on CT 257.

## 1. WHAT WE ARE DOING
(a) Tonight: Live Pulse on dev server CT 257 (`https://twr-lab-257.secureprospective.com`, league `native.mfl-14432-2026`).
(b) Alpha 0.10: prove the 15 lines of `docs/roadmap/RELEASE-0.10.md`. GPT lanes on the Beelink build; I review/merge.
Repo: Beelink `chris@192.168.1.191:~/work/TheWarRoom` (main checkout is HIS; worktrees only). Lab host 192.168.1.182
(VM 503, holds CT 257, CT 258, CT 110 gateway). From Beelink: `ssh -F ~/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab`.
Never touch CT 110, 256. CT 257 = tonight's stream: NO lab load on host .182 until the game ends.

## 2. DONE LOOKS LIKE
Tonight: T1 week 10 MET · T2 all 32 lineups MET (3:54 PM; 24 starters dropped: not active at Thu kickoff) · T3 box scores MET ·
T4 app on Beelink MET (`~/scratch/twr-mnf-1116/thewarroom-main`, CI build of main a7b7adb, sha c9fa86d8…, walked on Claude-OS :5;
fallback `thewarroom-257` = d6b2 build) · T5 GO given · T6 live ticks during MNF: CHECK at 7:22 PM.
0.10 lines (tracker "the 15 done lines" mirrors this): 1 MET · 2 server MET, desktop fixed by #305 (proof at next tag) ·
3-5 lab job 1 tonight · 6-7 him Tue with 0.10.2 · 8-11 proven on CT 257, redo on 0.10 · 12 MET · 13 tampered MET, older bundle =
lab job 3 · 14 NOT PASSED (502s) rerun tonight with #310 · 15 11 pass / 0 fail / 3 not proven.

## 3. IN FLIGHT
| Job | Check | On result |
|---|---|---|
| Merge watcher: #311 then #309 (CI watch, then `gh pr merge --squash --delete-branch`) — started ~7:10 PM | `cat /tmp/claude-0/-root/28007245-980e-4ed8-8097-60c542958a5b/merge2.out`; or `gh pr view 311/309 --json state` | Both approved by him. If CI fails: read log, send fix lane. Update tracker F56/F53 → merged |
| Timer 7:22 PM: live tick check | `pct exec 257 -- grep "server live" /var/lib/twr/server.log \| tail` via lab | Tell him ticks flow or switch to fallback app |
| Timer 10:15 PM: lab lanes | — | Confirm game final, then launch 4 Luna lanes (section 10) |
No GPT lanes running (checked 7:15 PM).

## 4. RESULTS TODAY (this window)
- Lineup fill: real cause = after first kickoff a save needs every starter ACTIVE at Thu kickoff (`CheckLineupAppend`,
  lineup_week.go). Preview DID list it under `Blockers`; my w10.py ignored blockers (my error). w10.py now filters via
  `lineup.read` at `at=2026-11-13T01:20:00Z`.
- Merged (main): #303 odds read once (not every 10 s) · #305 desktop version from tag (dev-<sha> otherwise) · #306 LAB.md roots ·
  #310 overload → 503 + Retry-After 2 s, client retries reads once. Closed: #302 (no CPU gain, superseded by #311), #308 (ruling).
- Open: #311 odds seed+cache from real inputs (playOut 22% → <0.5% CPU; p95 0.95–1.6 s vs 1.7–3.2 s) · #309 lineup refusal reason
  for permitted principals, generic for outsiders (all CI green before merge-conflict fix; hosted.go conflict resolved keeping
  both `writeFailure(w, r, …)` and the refusal branch) · drafts #304 (archive weekly files) + #307 (MFL players list daily) for F51.
- Lane launcher now supports luna (`gpt-6-luna`). Tracker fully rebuilt 6:10 PM (goal, 15 lines, tonight, review, Tue).

## 5. ARTIFACTS
- Apps on Beelink: `~/scratch/twr-mnf-1116/thewarroom-main` (sha256 c9fa86d82381fef6…), `thewarroom-257` (b9fe3682…). CI artifact run 43979023178.
- Claude-OS: `~/twr-d2/thewarroom-main` running on :5 via `TWR_BIN=thewarroom-main ~/twr-d2/run-a.sh` (Eagles test acct).
- Briefs `~/fleet/briefs/alpha010/`: f51a/a2/a3, f51b, lsave, storm47, vstamp, rel9, cpu6, busy, labmd. Runs `~/fleet/runs/pulse-<lane>/`.
- F51 dry-run kit `/tmp/f51/` (rehearsal257.tgz holds accounts.db: delete when done).
- CT 258 lab keys: `.worktrees/pulse-a010s/.scratch/lab-release/` (root 8c91770c…) — move to `~/fleet/lab-release-ct258/` (follow-up).

## 6. OPEN BUGS
- #309 gap: after a refused execute the desktop/frontend still shows generic wording (`internal/serverclient/desktop.go:363`,
  `frontend/src/app/commissioner/hostedRoom.ts:183`). Follow-up lane.
- 502 cause at 80 devices: LIKELY 5 s deadlines (a010d), not proven; stress rerun is the proof.

## 7. REFUTED — DO NOT RETEST
- CT 258 for Pulse (no NFL week). Lineup refusals as timing race. Preview "does not check kickoff roster" (it does, in Blockers).
- Release 9 for a patch (contract rule; #308 closed). Terra for multi-step jobs (quits early 3x) — use Luna high.
- Earlier list in repo copy history (commit 9ceaea402).

## 8. DECISIONS
- F51 = option A: server reads MFL league-scoped players list once a day (24 h archive reuse, 1 h failure floor).
- Release number = compatibility contract; older-bundle proof via LAB-ONLY bundle signed as release 7.
- GPT: Luna high default, Terra short scoped only, Sol when needed; many Luna lanes OK (RAM-bound, keep ≥3 GB free).
- Merges: when he says "merge N", I run `gh pr merge` myself after CI; batch the asks.
- Odds seed from real inputs (my engineering call). Lab jobs after the game (his yes). App updated to main build before stream (his pick).

## 9. LEDGER
Main includes #303 #305 #306 #310. This RESUME: branch `session/a010-docs` (worktree `.worktrees/pulse-a010`) `docs/RESUME-alpha010-1116.md`.

## 10. NEXT ACTIONS
1. 7:22 PM: check CT 257 live ticks; tell him (switch to `thewarroom-257` if the main app misbehaves).
2. Confirm #311 and #309 merged; tracker F56/F53 → merged.
3. After the final (~10:15 PM): launch Luna lanes on the lab (host .182; never CT 110/256/257 changes beyond reads):
   (1) fresh install new CT 2xx via `twr-up` with a fresh lab bundle (lines 3-5: address printed, reboot → twr-status 0, league + backups);
   (2) stress rerun on CT 258 after upgrading it to main (line 14), brief base `stress7.md`;
   (3) LAB-ONLY bundle signed as release 7 refused by CT 258 before services stop (line 13), keys in pulse-a010s/.scratch;
   (4) F51 trial: #304+#307 on `/tmp/f51/rehearsal257.tgz` copy, 1 GB/2 CPU, non-zero values, peak < 800 MB.
4. Week 10 final vs MFL for all 32 teams.
5. After the game: update CT 257 to main + rebuild his app; walk both first.
6. Follow-ups: #309 client wording; move CT 258 keys; next CPU `rememberNativeContracts` 13–16%.

## 11. NEEDS CHRISTOPHER
Rotate CT 257 TLS + authenticator keys (after stream). Tue: install 0.10.2 package on his screen (lines 6-7).

## 12. ENVIRONMENT
Times Central. Never print /etc/twr/* or passwords. Lanes: `ssh -f chris@192.168.1.191 'nohup setsid ~/fleet/bin/launch-pulse-lane.sh luna <lane> <brief> high >/dev/null 2>&1 &'`.
Lane hooks need `frontend/dist`: `cd frontend && pnpm install --frozen-lockfile && pnpm build`. -race needs CGO_ENABLED=1.
Remote heredocs inside ssh '…' break on apostrophes: write briefs locally and scp. Tracker: `~/fleet/bin/twr-tracker.py` (never `<cmd> --help`: it logs it).
Merge conflicts are usually HANDOFF.md: `git checkout --ours HANDOFF.md`.

## 13. HONEST STATUS
Stream ready (app walked, lineups full). Not yet proven: main-build app during a live tick (7:22 check). 0.10: 2 of 15 fully met,
lines 3-5, 13, 14 depend on tonight's lab jobs; 6-7 on him Tuesday.
