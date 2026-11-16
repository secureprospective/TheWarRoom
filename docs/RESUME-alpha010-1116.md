# RESUME — TheWarRoom alpha 0.10 (Claudebox head-brain session, Mon 2026-11-16)
Written about 8:20 AM CT. Read this, then resume at section 9. Do not re-derive or re-ask settled rulings.

## 1. WHAT WE ARE DOING
Giving TheWarRoom a finish line: release **alpha 0.10** = a real server + real desktop client Christopher tests
live, while work continues on a dev pair. Claudebox is head-brain only: GPT lanes (Sol, Terra, Luna) on the Beelink do
the heavy lifting; I brief, review, and hand Christopher merge commands.
- Repo: Beelink `chris@192.168.1.191:~/work/TheWarRoom`. Its main checkout is HIS branch: worktrees only.
- Tag `v0.10.0` = commit `44a663173` = origin/main (PR #295 merged; Christopher pushed the tag).
- Test-server VM ("lab"): 192.168.1.182, from the Beelink `ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 root@192.168.1.182`.
  CT 257 = dev server (live test tonight), CT 110 = gateway, CT 256 running (not mine): never touch these.

## 2. AGENTS + HARNESSES
- Builder lanes: `~/fleet/bin/launch-pulse-lane.sh <sol|terra> <lane> <brief.md> <thinking>` → worktree
  `.worktrees/pulse-<lane>`, branch `session/pulse-<lane>`, run dir `~/fleet/runs/pulse-<lane>/` (task.md, stdout.log =
  report, completion.txt = done marker, sessions/ = liveness by mtime).
- Read-only Luna survey lanes: `~/fleet/briefs/alpha010/launch.sh <letter>` (brief `<LETTER>.md`, reads
  `.worktrees/alpha010-read`, report `~/fleet/runs/a010-<letter>/report.md`). Lanes a–g are done.
- All briefs: `~/fleet/briefs/alpha010/`. Launch with `ssh -f … 'nohup setsid … &'`, arm a watcher in the same step.
- The Bash hook blocks command TEXT that mentions pushing to main: write briefs with the Write tool, then scp.
- CLAUDE.md v6.0 (new today): merges and pushes to main need Christopher's sign-off each time — hand him the command.
  Sol medium, Terra high, Luna high.

## 3. STATUS AGAINST THE 0.10 FINISH LINE (`docs/roadmap/RELEASE-0.10.md` on main)
| Line | State |
|---|---|
| 1 tag + CI green + Linux package | Tag pushed. CI on the tag was still running at 8:18 AM CT (verify in progress, desktop packages queued). CHECK. |
| 2 both report 0.10.0 | Server stamp merged (#295); proven only by test + GoReleaser snapshot. Not yet seen on an installed server. |
| 3–5 real server installed, reboot-safe, league + backups | Not started. Waits on stress + security. |
| 6–7 real client, he signs in | Not started. |
| 8–11 game day | Proven on CT 257 at v13 (older build); not yet on 0.10.0. |
| 12–13 update path (0.10.1 via twr-upgrade; tampered refused) | Not started. |
| 14 stress test | NOT STARTED. Sol lane building the missing check tool first (section 5). |
| 15 security sweep | Written, not in repo yet, not run. |

## 4. ARTIFACTS THAT EXIST
- On main: `docs/roadmap/RELEASE-0.10.md` (14 lines), `RELEASE-0.10-STRESS.md` (8 gates, says 64 devices), `RELEASE-0.11.md` (DRAFT).
- NOT yet in the repo, durable copies in Beelink `~/fleet/briefs/alpha010/pending-docs/`:
  - `RELEASE-0.10.md` with new line 15 (security sweep) and the Update-button wording.
  - `RELEASE-0.10-STRESS.md` with 80 devices and storm gates 9–11.
  - `RELEASE-0.10-SECURITY.md` (14 checks, new file).
- Lane reports: `~/fleet/runs/a010-{a..g}/report.md` (a install/version, b client, c CI gates, d game day,
  e stress tooling, f update button, g security survey).
- Tom's F51 (player values) handoff: branch `session/server-values`, `docs/from-tom/2026-11-16-server-values.md`;
  resume `~/fleet/briefs/twr-pulse-parallel-1114/RESUME-TOM.md`. F51 is NOT in 0.10.

## 5. THE CURRENT BLOCKER
The stress schedule needs `stresscheck` (integrity check + ledger listing inside the container). Tom wrote it
11-08 outside the repo (raw SQL broke the lint rule) and deleted it. Sol lane **a010s round 2** is rebuilding it at
`tools/stresscheck/` on branch `session/pulse-a010s`, opening a PR (not merging), then building the 0.10.0 bundle from
the tag, standing up **CT 258** (1 GiB / 2 cores / no swap), seeding, onboarding 80 devices, and starting
`stress-schedule.sh 258 STATE /root/stress-0.10.0` detached (about 3 hours). It stops and reports if a documented
step fails twice. Caveat: the server now splits data into per-league files; Sol must work out where the ledger
lives, so the tool's call line may differ from Tom's.

## 6. ALREADY SETTLED — DO NOT REDO
- 11-08 stress on an older build: durability passed (15 cycles, 72/72 actions once); speed failed (about 1 s screens at
  8 devices, 502s at 32+). Not re-measured on 0.10. Speed is a baseline, not a gate.
- The macOS desktop-package failure on #295 was GitHub runner capacity, not code; a re-run passed.
- `upgrade.sh` parses only the first line of `twr-server version`; the added `Server build …` line is safe.
- `stress.py onboard` is hard-wired to 80 devices; we keep it (heavier), not edit it.

## 7. CHRISTOPHER'S RULINGS TODAY
1. First release = alpha 0.10. 0.10.1 = one-PR patch (maybe hourly); 0.11 = recommended update; 0.20 = full field.
   Alpha below 1; beta runs until done. (memory `project_warroom_release_numbering`)
2. Real pair (he tests live; moves only by numbered release) and dev pair. Testers: him only until he opens it;
   league mates will look in, so it must stay up.
3. The 14-line done list approved. Real server goes on the test-server VM for now; CT 257 becomes dev.
4. Stress test required before the server is "permanent". Load/stability is the gate; speed is recorded as baseline.
   Simulate devices as best we can; a request storm is my call → flood from outside, ramp until refusal (gates 9–11).
5. Security sweep required before permanent (line 15).
6. Formal Update button required, in the commissioner console, only the commissioner may use it → headline of 0.11.
   Its privileged helper is a security design decision → expert panel before building.
7. "Have Sol make the necessary changes and builds."

## 8. LEDGER STATE
- Committed on main: #295 (version stamp + three roadmap docs).
- Committed on branch `session/a010-docs` (Beelink worktree `.worktrees/pulse-a010`): this RESUME at `docs/RESUME-alpha010-1116.md`.
- Not committed: the three pending docs (section 4). Plan: add them to Sol's `session/pulse-a010s` branch after its
  lane reports (overwrite Sol's STRESS edit with the pending copy), then hand Christopher the merge command.
- Claudebox backbone: memory index rewrite committed (`1967346`, `c5150df`). Unrelated to TWR; finished.

## 9. NEXT ACTIONS, IN ORDER
1. Check Sol: `ssh -n chris@192.168.1.191 'cat ~/fleet/runs/pulse-a010s/completion.txt; cat ~/fleet/runs/pulse-a010s/stdout.log'`.
   If no completion.txt, check liveness: `ls -lt ~/fleet/runs/pulse-a010s/sessions | head -3` (mtime within minutes = alive).
   If dead with no report, read the newest session file for where it stopped, then relaunch with `stress2.md`.
2. Review the `stresscheck` PR diff (read-only tool, no writes, SQL only via internal/store or internal/db, test present).
3. Copy the three pending docs into `.worktrees/pulse-a010s/docs/roadmap/`, commit, publish the branch, wait for CI,
   hand Christopher the merge command.
4. Confirm the stress run is alive on the lab: `cat /root/stress-0.10.0/phase`; arm a watcher for phase `stop`.
5. When it finishes: launch a Terra lane to score gates 1–8 with `stress-report.py` + `stress.py reconcile` and write
   `docs/roadmap/RELEASE-0.10-STRESS-RESULT.md`.
6. Sol lane: add a storm mode (lab tool), run gates 9–11 on CT 258; then run the 14-line security sweep on CT 258
   → `RELEASE-0.10-SECURITY-RESULT.md`.
7. Check CI on tag `v0.10.0`: `gh run list --branch v0.10.0`; get the Linux desktop package artifact.
8. Tonight after the 7:15 PM CT game (Chargers at Ravens): compare week 10 final points for all 32 teams with MFL
   on CT 257 (`claudeos:~/twr-d2/tools/w9cmp.py` pattern; the MFL fetch must run on Claude-OS).
9. If stress + security pass: stand up the real server (new CT on the lab, fresh keys), install the real client on
   his screen (:0 on Claude-OS is HIS screen), then lines 12–13 with a 0.10.1 bundle.
10. Later: rotate CT 257 keys (Hermes task T621); F51 fixes on dev; 0.11 panel on the update helper.
11. Reap when the lanes are done: worktrees `alpha010-read`, `pulse-a010`, `pulse-a010s`; CT 258 after results are saved.

## 10. ENVIRONMENT NOTES
- Times to Christopher in Central (machine clocks are UTC, CT = UTC−6).
- Never print `/etc/twr/*` contents. Commits over ssh need `GOTOOLCHAIN=go1.26.11` and a per-worktree
  `GOLANGCI_LINT_CACHE`; a fresh worktree needs `pnpm install --frozen-lockfile && pnpm build` in `frontend/` first.
- Background watchers time out at 10 minutes; re-arm them.

## 11. HONEST STATUS
0.10 is tagged but not proven: nothing has run on an installed 0.10.0 server yet. The stress run has not started and
may expose real failures (the last one saturated at 32+ devices). Earliest the stress result can exist is about
3.5 hours after Sol starts the schedule. The security sweep's protections are documented but never proven on a real install.
