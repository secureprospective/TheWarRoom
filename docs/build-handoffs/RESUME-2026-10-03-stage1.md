# RESUME — TheWarRoom Stage 1 starts (2026-10-03)

Stage 0 is closed and merged. Its full record is in the plan under Stage 0 and in
`RESUME-2026-10-03-stage0.md`. This file is the starting point for Stage 1.

## 1. What we are doing
- **Goal:** Stage 1 of `docs/build-handoffs/Core_Build_Plan_2026-10.md`: the measure-dictionary
  storage (raw archive → observations → features, plus the registries and scoring runs). Design
  reasoning: `Core_Build_Reasoning_2026-10.md` §5a (also §5, §5b, §6).
- **Repo:** `~/work/TheWarRoom` on the Beelink.
  - Branch `session/stage1-measure-store` was cut from `main` at `98b915f`. It has no
    upstream; the first push uses `git push -u origin session/stage1-measure-store`.
  - Never work on main.
- **Go:** `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB`
- **Gates:**
  - `make lint` runs ifaceguard, the filelen report, the bloat ratchet and golangci-lint.
  - `make verify` adds race tests and the frontend build; the pre-push hook runs it.
  - The bloat baseline is comment 20, provenance 7, tiny files 28, dupl 38. Run
    `bash scripts/bloat.sh` before pushing.
- **Roles:**
  - Claude builds and owns architecture (R4).
  - Claude runs live gates on Claude-OS (R12; the procedure is in CLAUDE.md "Workflow").
  - Bee reviews when GPT budget allows (Bee and Astra are stopped).
  - Christopher decides product and priority, and gives the go-ahead for merges to main.

## 2. State at handoff
- `main` = `98b915f` (PR #3, rebase-merged: 66 linear commits). Its tree is identical to the
  pre-merge tip, which is preserved as tag `archive/session/m1b-bash` (`5421be9`). Commit hashes
  cited in Stage 0 docs resolve through that tag.
- Only `main` and `session/stage1-measure-store` exist. 20 `archive/*` tags are on origin.
- christopher-coding-standards: PRs #33 (standards) and #34 (CVE fix) are merged; its main is
  `93ee82a`.
- Nothing is running. Claude-OS is shut off.

## 3. Facts about the live data (found in Stage 0, needed in Stages 1–2)
- Real DB: `~/.config/TheWarRoom/thewarroom.db`, still on migration v2.
  - It migrates to v3, with a backup, the first time a production build opens it.
  - It holds 831 contracts.
  - Its only rulebook version (v1, 2026-07-05) has no franchise list, so screens read
    "Franchise 0014". Stage 2 refreshes the rulebook.
- The live gate's snapshot of it, already migrated, is on Claude-OS at
  `~/.config/TheWarRoom/thewarroom.db`. Take a fresh snapshot for each new gate (SQLite backup
  API).
- MFL's players endpoint allows one call a day.

## 4. Next actions, in order
1. `git status`; confirm the branch is `session/stage1-measure-store` and `main` is `98b915f`.
2. Read plan §Stage 1 and reasoning §5a. Then write the Stage 1 design: tables, the separate
   `history.db`, the registry files and their loader, the features views, `scoring_runs` and
   the AD-04 append-only rule, and how it sits beside the existing stores (depguard rules in
   SYSTEM_MAP). Record the storage-shape decisions in the plan; they are Claude's calls (R4).
3. Build it to the Stage 1 gate list: idempotent fetch, as-of corrections, the source-loss and
   source-gain drills, the generated Measure Dictionary, and two scoring runs that are both
   readable.
4. Update SYSTEM_MAP in the same commit as any new package.

## 5. Follow-ups logged, not done
- `mfl.Client` has no test seam; add an injectable transport when Stage 2/4 touches fetching.
- `DiscoverHost` makes a full `league` request on every fetch; cache discovery per client.
- The frontend still has `name || franchiseID` fallbacks the backend makes redundant.
- A failed startup repeats its error below the banner and leaves SCORE LEAGUE enabled
  (cosmetic).
- The Wails CLI is v2.12.0 while the module is v2.13.0 (v2.16.0 is out). Align them on purpose.
- README "How It Gets Built": rewritten after the last stage (Christopher, 2026-10-03; plan
  open items).
- On the Hermes task list, TheWarRoom tasks sit in PARKED although the project is active. That
  is Christopher's call. Stage 1 is T368.

## 6. Decisions (do not relitigate)
- R1–R12 in the plan. History is a measure dictionary (R10); a lost source keeps running,
  flagged, and Christopher approves rebalances (R11).
- Packages Stages 4–7 rewrite are not polished; the ratchet holds them flat.
- Merges to main are rebase-merges (main requires linear history), with the branch tip tagged
  `archive/<branch>` first.

## 7. Lessons (do not relearn)
- `git commit` after `git add <paths>` also commits anything already staged. Check
  `git diff --cached --stat` before committing.
- pre-commit stashes unstaged edits but not untracked files: commit in dependency order, and
  check the commit's exit code.
- `git describe` takes any tag: the Makefile matches `v[0-9]*` only.
- `wails generate module` from the v2.12 CLI rewrites `frontend/wailsjs/runtime` file modes;
  keep only `models.ts`.
- trivy runs as `docker run --rm -v <dir>:/src:ro mirror.gcr.io/aquasec/trivy:latest fs
  --scanners vuln --severity HIGH,CRITICAL --ignore-unfixed /src`.
- On Claude-OS, launch with `setsid -f` or the ssh session hangs.
- Codesame tool: `~/fleet/runs/warroom-dataflow-2026-10-03/tools/` (`codesame <repo> HEAD
  <pkgdirs>`) proves a comment-only change left the code identical.
