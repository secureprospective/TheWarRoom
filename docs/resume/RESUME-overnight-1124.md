# Resume — overnight run 11-23/24 (Claude Opus head brain on the Beelink)
Updated 2026-11-24 03:35 CT. Run folder: `~/fleet/runs/overnight-twr-1123/` (REPORT.md = full log, PLAN.md, briefs/).

## 1. What we are doing
We are working the pickup note's five steps (docs/from-tom/2026-11-23-twr-pickup.md on session/resume-1123e).
Christopher's rulings for the run: Gemini when there is quota, otherwise Sonnet medium or Haiku high; keep pushing; standard of excellence.
- Repo: `~/work/TheWarRoom`, one worktree per job under `.worktrees/`.
- Lab: `ssh -F ~/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab`, then `pct exec 260 -- …`.

## 2. Done looks like
1. **MET.** #487 merged on green and carried by the Lab.
   - Merged 21:25; the Lab runs 9e749162f.
2. **NOT MET.** The MFL helper is built, works end to end on a real hosted server, and has an open PR.
   - The build is done and PR #489 is open, now marked DRAFT.
   - The live Lab check FAILED, so fix pass F is running.
3. **MET.** standings.read is faster, with a PR.
   - 163 -> 70 us/op. PR #488 is open with green CI.
4. **MET.** The Lab load test was rerun on the #487 release.
   - Results in `~/fleet/runs/overnight-twr-1123/load/RESULT.md`.
5. **MET.** The Sunday 11-29 practice drafts are prepared.
   - PR #490 is open with green CI. It was tested twice on the Lab.

## 3. Agents and harnesses
- **Claude headless lanes:** `~/fleet/runs/overnight-twr-1123/claude-lane.sh <job> <workdir> <model> <effort> <brief>`.
  - Output goes to `<run>/<job>/{out.log,REPORT.md,exit}`.
- **Gemini lanes:** `~/fleet/bin/agy-run.sh <job> <brief> "<model>" <workdir>`.
  - Output goes to `~/fleet/runs/<job>-MMDD/{report.md|RESULT.md,exit,pid}`.
- **Quota:** Argon is LOCKED until Fri 11/27, so Claude Opus reviews every lane and plants one break per lane.
- **Watcher:** `~/fleet/bin/run-watch.sh {add,wait,status} ~/fleet/runs/overnight-twr-1123`.
  - Stale rows standings-ship/standings-ship2 show "run OVERDUE": ignore them, their work landed.
- **Push scripts:** pre-push needs `frontend/node_modules`. Symlink main's copy, or lab-dev's for vitest; also `frontend/dist/.keep`.
  - Pattern: `<run>/{heldid,touch,mflpr}/ship.sh`.

## 4. Gates / status
| Item | State |
|---|---|
| #488 standings 2.3x (session/standings-read-speed) | CI green, open, waits OK |
| #489 My MFL moves (session/mfl-helper-build) | DRAFT: live Lab check failed; fixF running |
| #490 Sunday practice-draft scripts (session/practice-drafts-1129) | CI green, open, waits OK |
| #491 held identity key (session/held-identity-key) | CI green, open, waits OK. heavy reads 342 ms -> 0.78 ms at 110k ids |
| #492 session touch 25 ms bound (session/session-touch) | CI green, open, waits OK |
| Lab CT 260 | release 9e749162f, twr-status OK at 03:21 CT |
| Season server 192.168.1.162 | answers ping since ~03:20 CT (was down earlier) |

## 5. Artifacts
- **Lab release binary:** sha256 09ed15c15a0f9489f09c74938a9cd21896433f3c66af140463f4f9585c9a9f8d; a copy is at /root/twr-server.release on CT 260.
- **Profiles:** `~/fleet/runs/profile-1123/` (RESULT.md: where the ~90 ms per heavy read went).
- **Live MFL check:** `~/fleet/runs/mfllive-1124/RESULT.md`, with every finding and its file:line.
- **Lab kit (secrets, never print):** `~/scratch/commish-1123/` (has KEEP.md).
  - It also holds `bin/twr-mfl`, the branch CLI build from the live check.

## 6. Current bug (MFL helper, #489)
On a real hosted server every mflmoves.* command answers 409.
- **Error text:** the server log says "MFL moves storage is not available on this host" or "MFL checks are not available on this host".
- **Execute commands:** they additionally fail with "admit private owner data: credential admission: current league permission: this league action is not available".
- **Cause, confirmed by reading the code:**
  - footballHost and dueHost forward PrivateEntries() but not MFLMirror/MFLFetcher/LatestModelScores.
  - mflmoves.* is missing from privateOwnerPersonalOperation (serverhttp/hosted.go ~:775).
- **More findings in RESULT.md:**
  - The capability check runs before auth.
  - serverhttp swallows the plain words, so the panel can deadlock before the first refresh.
  - mfl.league is owned by CM-038 (the IR/taxi question).
  - Sorting by minimum compares strings.
  - The football refusal blocks mflmoves; the league export is fetched twice.
  - CLI/tokens can't run the commands.
- **Caveat:** not re-tested live after the fix yet.

## 7. Refuted. Do not retest
- **"#487 made phones faster":** not end to end. results.read p50 is 128.6 ms vs 131.8 ms on v59.
  - The real cost was heldSeasonKey re-hashing 125k ids per request (fixed in #491).
- **"Unit tests prove the MFL commands work hosted":** they don't. They pass StoredHost directly and skip the wrappers and admission.
- **"draft.timer 14-day set is needed":** it isn't. It races the autopicker, and the 13-day draft.clock already covers a seated owner's turn.
- **"cycle.sh can open drafts right away":** it can't. The autopicker drafts every open seat in minutes, so seats are claimed first.

## 8. Decisions
- **mfl.league:** a plain league setting.
- **Owner's MFL team:** a private choice per seat.
- **Values:** from the model run, labelled as such.
- **Roster limits:** from the MFL league's own rules; no invented minimum.
- **Practice league ids:** native.rehtest-*. The real native.rehearsal-* are never touched.
- **mflmoves executes:** delegable like player.note.set (in fixF).
- **#489:** stays draft until fixF passes and a live Lab recheck passes.

## 9. Ledger
- **Committed and pushed:** everything listed in §4.
- **Uncommitted:** fixF, which is in progress in `.worktrees/mfl-helper-build` (Neon is writing there; do not edit).

## 10. Next actions, in order
1. **When fixF exits** (`test -f ~/fleet/runs/fixF-1124/exit`):
   - Read report.md and review the diff.
   - Confirm the serverhttp end-to-end test fails on df29473de and passes after.
   - Plant one break, then commit and push (see `<run>/mflpr/ship.sh`).
2. **Rerun the live Lab check:** reuse `briefs/mfl-live.md` on Neon High, and say the branch head is the new commit.
   - It must pass every step, then restore the release.
3. **If it passes:** mark #489 ready (`gh pr ready 489`) and update its body.
4. **Command-line access (lane G):** let `twr` run workflow commands with raw JSON (RESULT.md finding 6), as its own PR.
5. **After #491 merges:** rerun the Lab load test (`<run>/load/load-run.sh`). p95 < 200 ms is the target.
6. **Close the run:** finish REPORT.md and run tidy-check. Clean up the worktrees from the pickup note's list.

## 11. Needs Christopher
- **Merge OKs:** #488, #490, #491, #492, and #489 once its recheck passes.
- **Sunday 11-29:** `cycle.sh` -> `seat-link.sh native.rehtest-default` (claim the seat) -> `open-drafts.sh` (scripts/lab/README.md).

## 12. Environment notes
- **"Force-push" hook false positive:** the hook refused a command that only contained `rm -f` after a push. Split the command into steps.
- **Self-kill:** `pkill -f <pattern>` inside a command matches its own shell. Use script files.

## 13. Honest status
- **#491 and #492:** green and reviewed, but not yet proven on the Lab under load.
- **MFL helper:** not proven on a real server until the recheck passes.
