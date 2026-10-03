# RESUME — TheWarRoom Stage 0 cleanup, in progress (2026-10-03)

## 1. What we are doing
- **Goal:** Stage 0 ("one timeline") of `docs/build-handoffs/Core_Build_Plan_2026-10.md`.
  The reasoning behind the plan is in `Core_Build_Reasoning_2026-10.md`.
- **Repo:** `~/work/TheWarRoom` on the Beelink, branch `session/m1b-bash`, pushed to origin at
  `bda156b`. Never work on main.
- **Go:** `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB`
- **Builder:** Claude builds; Bee reviews when GPT budget allows (ruling R4). GPT budget ran out
  10-03, so both Bee and Astra are stopped.
- **Christopher's standing direction (10-03):**
  - Claude is the engineering and architecture authority (memory
    `claude-is-the-engineering-authority`). Decide and explain; ask him only product questions.
  - Always cut AI slop (memory `seek-the-better-architecture-and-cut-ai-slop`).

## 2. Stage 0 status
| Item | Status | Commit |
|---|---|---|
| 0.1 Merge origin/main | DONE | `12d30a5` |
| 0.1 Fix the pre-push `ifaceguard` failure (task T070) | DONE: `REPOS/` held foreign clones with bad Go fixtures; moved to `~/scratch/warroom-REPOS-2026-08/`, logged in MOVED.md, MANIFEST and CT105 `context.md` | — |
| 0.1 Delete stale branches | PENDING Christopher's confirmation (§6) | — |
| 0.2 Docs up to date (CLAUDE.md, SYSTEM_MAP, Hard Constraints for R6/R7, Commissioner plan copy) | NOT STARTED: do last, so the docs describe the cleaned code | — |
| 0.3 Every launch logs; init has a 2-minute timeout (the July TWR-1 fix); `-probe` = real startup run headless; the duplicate `probe.go` is deleted | DONE, verified on a dev-DB copy and with networking cut | `36d66da` |
| 0.3 Startup failures show a banner through AppInfo; Ping retired | DONE. Banner not visually verified: add to Christopher's live gate | `4d68e9c` |
| 0.4 Five unwired fetchers deleted (2,343 lines); `GetFranchiseState`/`GetFreeAgents` deleted | DONE | `0975fd4` |
| 0.4 DT cushion reads its admin params (test proven with a mutation) | DONE | `8bed8bf` |
| 0.4 `CORRECT` op | KEEP the backend; its UI is deferred (Commissioner work) | — |
| 0.5 Standards rules rewritten upstream | DONE: christopher-coding-standards **PR #33**, open for Christopher to merge | `f148127` there |
| 0.5 Codex re-adopted; `scripts/bloat.sh` ratchet in `make lint`; filelen only reports | DONE | `faec16a` |
| 0.5 Comment cleanup | IN PROGRESS: root, state and transactions done (`ad18183`, `de301c0`, `bda156b`) | — |

**Bloat now:** comment share 25% (was 31%), review history 39 lines (was 68), sub-40-line files 32
(was 33), dupl 44. The `.bloat-baseline` file is ratcheted down to these numbers.

## 3. Cleanup method (keep using it)
- **Tools.** Saved in `~/fleet/runs/warroom-dataflow-2026-10-03/tools/`; `/tmp/codesame` may be
  gone. Rebuild with `cd <tools> && go build -o codesame .`
  - `codesame <repo> HEAD <pkgdirs…>`: a comment- and whitespace-blind declaration diff. It
    passes only if the code is identical. Proven both ways.
  - `blocks.py files…`: lists comment blocks with line numbers.
  - `apply.py edits.py`: a list of `(path, start, end, newtext)`; empty text deletes the block.
    - The edit file is a Python literal.
    - The tool asserts every line it replaces is a comment.
    - Indentation is reused.
    - **Watch:** a `//nolint:x // reason` line followed by comment lines is a trap. Its reason may
      continue onto the next line, and replacing that line truncates the reason.
  - `merge.py target src…`: appends the source files' declarations to the target and unions the
    imports. Run `goimports -w target` afterwards.
- **Per package:**
  1. Run `blocks.py`.
  2. Write the edits.
  3. Run `apply.py`, plus Python string replaces for trailing comments.
  4. Run `gofmt`, then `codesame`.
  5. Run `golangci-lint run ./pkg/` and `go test -race ./pkg/`.
  6. Commit.
  7. At the end, run `scripts/bloat.sh --update`.
- **Rules:**
  - Keep a short doc comment on exported names.
  - Keep the "why" and rulebook § references.
  - Drop review/agent names, B-/Session/Ship/AD-/SL- labels, restated code, and file headers
    that only explain a size split.
  - Merge files that were split only for size when they share one job.

## 4. Next actions, in order
1. **Finish the comment cleanup**, one commit per package or group. Comment lines/total, measured
   10-03:
   - `transactions/contracts` 134/366
   - `harness` 292/965
   - `store/rulebook` 131/698
   - `store/params` 110/505
   - `output` 140/512
   - `composition` 138/494
   - `rankings` 145/405
   - `m2service` 75/343
   - `powerrankings` 67/204
   - `normalize` 93/385
   - `domain` 108/325
   - `db` 48/150
   - `mfl` 39/281
   - `playerid` 48/112
   - `engine` (non-l4) 174/434
   - `ingestion` root and the MFL-side packages: league, players, rosters, playerscores,
     leaguestandings, leagueschedule, salaryadjustments, crosswalk
   - `scouting` 141/209 (types only)

   **Skip, because Stages 4-5 rewrite them:** `engine/l4/*`, `scouting/assembly`, and the
   nflverse/CFBD ingestion packages (agetrajectory, collegedefense, collegeshare, madden,
   pfrcoverage, ras, schooltier, veteranfilm).
2. **Code refactors found during the cleanup.** These are real code changes, each with tests:
   - `standingsOrCache` / `leagueScheduleOrCache` (app) → one generic live-or-cache helper. The
     two cache tables become one `mfl_cache(kind, …)`; that needs a migration.
   - `franchiseDisplayName` (app), `resolveFranchiseNames` (feed) and the m2service copy → one.
   - The signing-window and trade-deadline directive read/write → one generic "phase directive by
     key".
   - Coordinator Execute/Preview × Tag/Extension/Sign → resolve the request, then the shared
     Execute/Preview.
   - Feed kind classified twice (SQL `CASE` + Go mirror for a test) → classify once, in Go.
   - TxWriter interface groupings exist only for the `interfacebloat` limit (`LogTradeNote` and
     `AppendCorrection` sit in `CapLedgerWriter`). Restructure, or raise the limit for this
     interface.
   - `contracts.contract_years` column: never populated (1,374/1,375 rows are 0). Drop it with
     a migration.
3. **0.2 docs** (CLAUDE.md, SYSTEM_MAP, Hard Constraints per R6/R7, Commissioner plan copied from
   CT105 marked DEFERRED, Build_Tracker pointer already done). Then the Stage 0 gate:
   - one branch line
   - verify green, plus `make bloat`
   - docs match the code
4. **Ask Christopher:**
   - Confirm stale-branch deletion: tag `archive/<name>` first, then delete. Every branch except
     m1b-bash and main is either merged or superseded.
     - Proven merged into main: `session-0`, `session-2`, `alpha-*`, `claude-md-archive-cleanup`.
     - Content already on main: `migration/pnpm-thewarroom-frontend`, `warden-pr2`,
       `glm-audit-fixes`.
     - Superseded older drafts: the rest.
   - Merge standards PR #33.
   - The live gate: launch the app (log is written, banner on failure).
5. **Stage 1** (measure-dictionary storage): see the plan.

## 5. Decisions (do not relitigate)
- **R1–R11** are in the plan.
- **Christopher, 10-03:**
  - The history store is a measure dictionary.
  - A lost source keeps running and is flagged; he approves the rebalance.
  - Claude owns the architecture.
  - The standards change goes upstream too.

## 6. Hypotheses refuted (do not retest)
- The push failure is not Go code: it was the foreign `REPOS/` fixtures, now moved.
- Unmerged remote branches carry no lost work, with one exception: review-harvest's TWR-1
  startup-timeout fix, now re-applied in `36d66da`. Its docs (harvest outcome, July fullmap
  review) are superseded by the 10-03 audit.
- `codesame` false positives came from git root-path handling and per-file package entries.
  Both are fixed.

## 7. Honest status
- Stage 0 is about 60% done. The biggest remaining parts are the comment pass on about 15
  packages, then the refactors.
- Nothing has been verified in the GUI.
- Bee and Astra are stopped: no GPT budget.
