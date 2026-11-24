# Resume — overnight run 11-23/24 (Claude Opus head brain on the Beelink) — CLOSED
Updated 2026-11-24 07:25 CT. Run folder: `~/fleet/runs/overnight-twr-1123/` (REPORT.md: closing summary on top, full log below).

## 1. What we did
We worked the pickup note's five steps (docs/from-tom/2026-11-23-twr-pickup.md, branch session/resume-1123e).
Repo: `~/work/TheWarRoom`. Lab: `ssh -F ~/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab`, then `pct exec 260 -- …`.

## 2. Done looks like
1. **MET.** #487 merged and carried by the Lab (now v70, faa12dd1e).
2. **MET.** The MFL helper is PR #489, merged as a99d18c84 and live on the Lab as v71. It was observed
   working on the real Lab server in `~/fleet/runs/mfllive2-1124/`, `mfllive3-1124/` and `mfllive4-1124/`.
3. **MET.** standings.read is 2.3x faster: #488 merged.
4. **MET.** The load test was rerun on v68 and v70 (`<run>/load2/RESULT.md`).
   - The median request dropped from 95 to 39 ms.
   - p95 is 316 ms, so the < 200 ms target is NOT met.
5. **MET.** The Sunday practice drafts are #490, merged.

## 3. Agents and harnesses (as used)
- **Gemini lanes:** `~/fleet/bin/agy-run.sh <job> <brief> "Gemini 4 Neon (High)" <workdir>`, output in `~/fleet/runs/<job>-MMDD/`.
- **Luna lanes:** `BASE=<branch> ~/fleet/bin/launch-pulse-lane.sh luna <lane> <brief> high`.
  - Output goes to `~/fleet/runs/pulse-<lane>/`; the worktree is `.worktrees/pulse-<lane>`.
- **Routing ruling (11-24):** when GPT is up, use Luna high with tight briefs; use Gemini Neon when a job needs more thinking.
- **Watcher:** `~/fleet/bin/run-watch.sh`. It is stopped at close; nothing is armed.

## 4. Gates / status
| Item | State |
|---|---|
| #488, #490, #491, #492 | MERGED 11-24 ~04:00 on Christopher's OK |
| #489 My MFL moves | MERGED a99d18c84 (Lab v71) |
| #493 `twr workflow` | MERGED 60ebd21fe |
| Lab CT 260 | v70 faa12dd1e, twr-status all OK at 12:17 UTC |

## 5. Artifacts
- **Lab release binary:** sha256 9aab720fe596b54206f25efb7d13872f30c47c400674ace0e8b11c0bb7d3dd29 (v70); a copy is at /root/twr-server.release.
- **Lab kit:** `~/scratch/commish-1123/`. It holds secrets: never print them. It has a KEEP.md.

## 6. Current bug
None open on #489. Known limits are listed in the PR body:
- Values come from the house model.
- MFL names no roster minimum.
- A helper token also needs `native.directory.read`.

## 7. Refuted. Do not retest
- **"Unit tests prove a hosted command works":** they don't. They miss the wrappers, admission and decode. Every
  hosted command needs a serverhttp end-to-end test and a live Lab check (memory hosted-commands-need-e2e).
- **"A refusal in Validate reaches the owner":** it doesn't. Hosted decode turns any Validate error into 400 "invalid
  native request", so business rules belong in the command body.
- **"Writes that skip db writeGuard are a blocker":** they aren't. Owner-private stores (`internal/store/personal`) do the same.
- **"The slow load-test tail is one slow read":** it isn't. All read types share it (p50 ~40, p95 250–420 ms).

## 8. Decisions
- **mfl.league:** a plain league setting.
- **MFL team:** chosen per seat.
- **Values:** from the model run.
- **Roster limits:** from MFL's own rules.
- **Bids:** one drop per bid.
- **mflmoves executes:** delegable.
- **Practice leagues:** native.rehtest-* only.

## 9. Ledger
Everything is committed and pushed. Nothing is uncommitted.

## 10. Next actions
1. **Optional:** profile one score-update burst on the Lab (method: `~/fleet/runs/profile-1123/`) to find what the p95 tail is.
2. **Later:** money units in the MFL JSON are mixed (dollars in, cents out). Settle this before more helpers build on it.
3. **Handoff:** `docs/from-tom/2026-11-24-overnight-close.md`.

## 11. Needs Christopher
- **Merge OKs:** none waiting.
- **Sunday 11-29:** `cycle.sh` -> `seat-link.sh native.rehtest-default` -> `open-drafts.sh` (scripts/lab/README.md).

## 12. Environment notes
- **Force-push is blocked by a hook:** to update a branch after main moved, merge main into it and push normally.
- **Pre-push in a worktree:** needs a `frontend/node_modules` symlink and `frontend/dist/.keep`.

## 13. Honest status
- **#491 and #492:** proven under load on the Lab.
- **#489:** proven live, step by step, at 49da38235 (full), b46ae7d9c (18/19), and 541fb783f (the fixed step).
- **#493:** proven by tests only. No live run, because the CLI talks to a server only after merge plus release, or through a branch build.
