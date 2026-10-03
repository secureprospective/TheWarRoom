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
| 0.5 Comment cleanup | DONE: every package outside the skip list, one commit each (`ad18183` … `bd8dc3e`). Along the way: deleted dead code (`db.Health/JournalMode`, 11 unread `scouting.Profile` fields, the template `internal/schema` package), folded 3 tiny engine files into `pipeline.go`, and fixed 4 comments that stated false facts | — |

**Bloat now:** comment share 20% (was 31%), review history 8 lines (was 68, all in skipped
packages), sub-40-line files 28 (was 33), dupl 44. The `.bloat-baseline` file is ratcheted down
to these numbers.

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
1. ~~Comment cleanup~~ DONE. Skipped, because Stages 4-7 rewrite or retire them:
   `engine/l4/*`, `scouting/assembly`, `harness`, and the nflverse/CFBD ingestion packages.
2. **Code refactors found during the cleanup.** These are real code changes, each with tests:
   - ~~live-or-cache helpers~~ DONE (generic `liveOrCache`). The two tables are left for Stage 4,
     where `raw_archive` replaces them; a migration now would be thrown away.
   - ~~franchise names~~ DONE (`domain.FranchiseLabel`).
   - The signing-window and trade-deadline directive read/write → one generic "phase directive by
     key".
   - Coordinator Execute/Preview × Tag/Extension/Sign → resolve the request, then the shared
     Execute/Preview.
   - Feed kind classified twice (SQL `CASE` + Go mirror for a test) → classify once, in Go.
   - TxWriter interface groupings exist only for the `interfacebloat` limit (`LogTradeNote` and
     `AppendCorrection` sit in `CapLedgerWriter`). Restructure, or raise the limit for this
     interface.
   - Every MFL fetcher repeats DiscoverHost → Do → status → CheckAPIError → decode envelope →
     flatten. One generic `ingestion.FetchMFL[Env]` would own that, leaving each fetcher its
     envelope type and Validate. Check that rosters/players/playerscores all call
     CheckAPIError today (salaryadjustments and leaguestandings do).
   - `normalize.lookupEntry` duplicates `PlayerFacts` field for field; embed it.
   - `.pre-commit-config.yaml` uses deprecated stage names (`pre-commit migrate-config`).
   - Params seed only into an empty `param_defaults` table, so a parameter added in code never
     reaches an existing database (GetGlobal then errors). Today's live DB has all 5, but
     Stage 5 adds many. Seed with `INSERT OR IGNORE` on every start; overrides are a separate
     table and stay untouched. Test: an existing DB gains a new key on restart.
   - The "one per franchise per season" allowance check is copied in Tag, Extend and
     Restructure (`contracts.go`). Fold it into the coordinator rework above.
   - `contracts.contract_years` column: never populated (1,374/1,375 rows are 0). Drop it with
     a migration.
3. **0.2 docs** (CLAUDE.md, SYSTEM_MAP, Hard Constraints per R6/R7, Commissioner plan copied from
   CT105 marked DEFERRED, Build_Tracker pointer already done). Also `docs/scoring-engine/Scouting_Schema.md`:
   it still lists the 11 deleted Profile fields. Then the Stage 0 gate:
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
- Stage 0 is about 75% done. Left: the refactors (item 2), the docs (item 3), Christopher's
  three items (item 4).
- Nothing has been verified in the GUI.
- Bee and Astra are stopped: no GPT budget.
