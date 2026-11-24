# T710 — Desktop Removal Plan and Keep-List (Rescanned on `60ebd21fe`)

**What changes for an owner: nothing.** Hosted leagues, the web/phone app served by `cmd/twr-server`, the CLI and MCP server (`cmd/twr`), MFL league import, and the hosted "My MFL moves" helper (PR #489) all keep working without change. Only the legacy Wails desktop shell and desktop-only code leave the main build, in small reversible PRs.

## Plain-words summary

- On `origin/main` (`60ebd21fe`), the module has **151 Go packages**.
  - `cmd/twr-server` and `cmd/twr` link **100** of them at runtime (**107** with transitive test dependencies).
  - Adding the kept dev/release tools (`cmd/twr-check`, `tools/{boundarycheck,fit,hostedfixture,stresscheck,twr-practice}`, `scripts/release`) brings runtime deps to **108** (**114** with transitive test deps, plus `internal/seams` and `frontend/scripts/nativegen` = **116** kept packages).
  - Before PR-A, `go list -deps ./cmd/twr-server ./cmd/twr ./frontend/scripts/nativegen` links **130** packages because `frontend/scripts/nativegen` imports `internal/composition` (dragging in 30 desktop packages), and `nativegen` tests also import `internal/app/commissioner`.
- **35 Go packages can be removed** (**26,880** prod + **16,895** test = **43,775 lines**), plus second-wave desktop files inside kept packages and the frontend Wails glue.
- **10 packages moved from Tom's earlier delete list (`c7b6f2337`) onto KEEP** (**4,984** prod + **7,022** test = **12,006 lines**): PR #489 ("My MFL moves on the hosted app") wired them directly into `cmd/twr-server`. Deleting them would break the server build.
- **Solo-owner rule ("trim, never delete"):** PR #489 now carries the hosted MFL moves path on top of those 10 kept packages. Removing the remaining desktop MFL/offer packages and updating `CLAUDE.md`, `North_Star.md`, and `README.md` still waits for Christopher's sign-off in PR-F.

## Scan method

Rescanned on `60ebd21fe` with:

```sh
go list -e ./...
go list -deps ./cmd/twr-server ./cmd/twr
go list -deps ./cmd/twr-server ./cmd/twr ./frontend/scripts/nativegen
```

plus transitive `TestImports` / `XTestImports` closure of the server, CLI, generator, and kept dev/release tools. Line counts are real `wc -l` counts over `*.go` files on `60ebd21fe` (production and `*_test.go` separated).

## Must-keep packages (moved off Tom's earlier delete list)

Every row below is linked by `cmd/twr-server` on `60ebd21fe` via PR #489 (and engine tests). All 10 are **KEEP**.

| Package | Prod | Test | Why it stays / Importer on `60ebd21fe` |
|---|---:|---:|---|
| `internal/mflref/envelope` | 1,947 | 2,080 | Hosted MFL checks and plans (`internal/mflmirror/mflread/{checks,plans}.go` via `cmd/twr-server`) |
| `internal/mflref/lineup` | 166 | 117 | Hosted MFL lineup checks (`mflread/{checks,plans}.go`, `envelope`) |
| `internal/store/mflref/moves` | 522 | 774 | Private hosted MFL plans store (`mflread/plans.go`) |
| `internal/transactions` | 1,365 | 3,490 | Imported by `internal/mflref/envelope`; coordinates the four subpackages |
| `internal/transactions/acquisitions` | 38 | 0 | Imported by `internal/transactions` |
| `internal/transactions/contracts` | 286 | 130 | Imported by `internal/transactions` |
| `internal/transactions/deadcap` | 299 | 164 | Imported by `internal/transactions` and `mflread/checks.go` |
| `internal/transactions/freeagency` | 58 | 45 | Imported by `internal/transactions` |
| `internal/engine/contractacts` | 199 | 196 | Core engine (`internal/engine/*`); imported by `envelope`, `store/mflref/moves`, `transactions` |
| `internal/ingestion/freeagents` | 104 | 26 | Hosted MFL fetch (`mflread/fetch.go`) and engine test (`internal/modelrun/value_board_test.go:17`) |
| **Total must-keep (10 packages)** | **4,984** | **7,022** | **12,006 lines kept** |

## Packages outside server, CLI, and kept tools (43 packages)

Verdict: **STAY** = kept (test dependency of kept packages, repo guard, or generator); **GO** = removable in PR-D, PR-E, or PR-F.

| Package | Prod | Test | Verdict |
|---|---:|---:|---|
| `.` (root `main.go`, Wails entry) | 25 | 0 | GO (PR-D) |
| `internal/composition` | 4,989 | 1,618 | GO (PR-D, after PR-A and PR-B) |
| `tools/fixtures` | 441 | 292 | GO (PR-D) |
| `internal/app/commissioner` | 903 | 2,360 | GO (PR-E, after PR-A and PR-D) |
| `internal/app/commissioner/commissionertest` | 80 | 0 | GO (PR-E) |
| `internal/app/commissioner/native` | 543 | 279 | GO (PR-E) |
| `internal/app/control` | 2,202 | 1,717 | GO (PR-E) |
| `internal/app/core` | 964 | 660 | GO (PR-E) |
| `internal/app/core/coretest` | 121 | 0 | GO (PR-E) |
| `internal/app/draft` | 607 | 431 | GO (PR-E) |
| `internal/app/draft/drafttest` | 74 | 0 | GO (PR-E) |
| `internal/app/market` | 1,722 | 865 | GO (PR-E) |
| `internal/app/market/markettest` | 51 | 0 | GO (PR-E) |
| `internal/app/mflref/bids` | 230 | 0 | GO (PR-E) |
| `internal/app/mflref/contracts` | 3,330 | 1,289 | GO (PR-E) |
| `internal/app/mflref/contracts/contractstest` | 134 | 0 | GO (PR-E) |
| `internal/app/mflref/migration` | 213 | 244 | GO (PR-E) |
| `internal/app/mflref/planwatch` | 766 | 423 | GO (PR-E) |
| `internal/app/mflref/seed` | 122 | 118 | GO (PR-E) |
| `internal/app/mflref/trades` | 1,398 | 828 | GO (PR-E) |
| `internal/app/offers` | 1,151 | 784 | GO (PR-E) |
| `internal/app/offers/offerstest` | 47 | 0 | GO (PR-E) |
| `internal/app/offers/personal` | 74 | 35 | GO (PR-E) |
| `internal/app/offers/pool` | 217 | 78 | GO (PR-E) |
| `internal/app/platform` | 1,130 | 989 | GO (PR-E) |
| `internal/app/race` | 297 | 719 | GO (PR-E) |
| `internal/app/research` | 2,666 | 1,784 | GO (PR-E) |
| `internal/app/research/researchtest` | 117 | 0 | GO (PR-E) |
| `internal/app/scoring` | 545 | 402 | GO (PR-E, after PR-B parity check) |
| `internal/mflref/whatif` | 71 | 66 | GO (PR-F, needs Christopher's ruling) |
| `internal/store/mflref/tradeplans` | 72 | 50 | GO (PR-F, needs Christopher's ruling) |
| `internal/store/rulebook` | 787 | 323 | GO (PR-F, needs Christopher's ruling) |
| `internal/store/playerrecord` | 207 | 244 | GO (PR-F, needs Christopher's ruling) |
| `internal/ingestion/researchfeeds` | 469 | 246 | GO (PR-F, needs Christopher's ruling) |
| `internal/ingestion/sitenews` | 115 | 51 | GO (PR-F, needs Christopher's ruling) |
| `frontend/scripts/nativegen` | 572 | 218 | STAY (retargeted off desktop in PR-A) |
| `internal/app/now/nowtest` | 52 | 0 | STAY (test dep of `internal/app/now`) |
| `internal/app/season/seasontest` | 98 | 0 | STAY (test dep of `internal/app/now`/`season`) |
| `internal/ingestion/pendingtrades` | 195 | 70 | STAY (test dep of `internal/ingestion/seasonfeed_test.go`) |
| `internal/ingestion/playerscores` | 148 | 173 | STAY (test dep of `internal/measures/measures_test.go`) |
| `internal/ingestion/playoffs` | 197 | 34 | STAY (test dep via `seasontest`) |
| `internal/leaguemigration/migrationtest` | 425 | 0 | STAY (MFL-import test fixtures used by server and 6 kept pkgs) |
| `internal/seams` | 0 | 67 | STAY (`FUTURE-SEAM(` doc-parity guard) |
| **Total (all 43 packages)** | **28,567** | **17,457** | **46,024 lines** |
| **Less 8 STAY packages** | **-1,687** | **-562** | **-2,249 lines** |
| **Removable (35 GO packages)** | **26,880** | **16,895** | **43,775 lines** |

Kept packages NOT in the table above (already linked by server, CLI, or stayed tools): `internal/app/{now,report,season,sources}`, `internal/snapshot`, `scripts/release` (56/41), `tools/boundarycheck` (521/293 — keep MFL fence, drop composition/core half in PR-D), `tools/fit`, `tools/hostedfixture`, `tools/stresscheck`, `tools/twr-practice`, `cmd/twr-check`.

Second-wave desktop code inside kept packages (PR-G and PR-H):
- `internal/serverclient/desktop*.go`: 1,265 prod (5 files), 940 test (5 files)
- `internal/commandcatalog/desktop.go`: 305 prod; `migration_desktop_test.go`: 218 test; `WailsDoor` in `registry.go`
- `internal/serverhttp/desktop*_test.go`: 342 test (2 files)
- `frontend/wailsjs/`: 28 generated files, 18,130 lines; 43 references to `wailsjs` or `window.go` across 24 files in `frontend/src/`

## PR order (A..I)

Each PR must pass `go build ./... && go vet ./...` and tests of touched packages on its own. Every step is a clean `git revert`.

| Step / PR | Scope | Gate / Proof |
|---|---|---|
| **Step 0** | Rescan keep-list on `60ebd21fe` and record this plan (`docs/architecture/desktop-removal-plan.md`). No code deletion. | Real `go list -deps` and `wc -l` counts on `60ebd21fe`. |
| **PR-A** | **Generator:** snapshot all generated outputs; re-point `frontend/scripts/nativegen` (`doors.go`, `registry.go`, `doors_test.go`, `reads_test.go`, `main_test.go`) onto `internal/commandcatalog` and drop its `internal/composition` and `internal/app/commissioner` imports. | `nativegen -check` clean across all surfaces; `diff -r` against snapshots = 0 bytes; `go list -deps ./frontend/scripts/nativegen` has no `composition` or `app/commissioner`. |
| **PR-B** | **Parity tests only (no deletes):** verify `internal/app/scoring` parity (`leagueservice`/`modelrun`) and port any missing restore / MFL-separation tests from `internal/composition` (`backup_restore_drill_test.go`, `restore_cli_test.go`, `mfl_separation_test.go`, `selfprotect_test.go`) into `cmd/twr-server`. | New parity tests pass on `cmd/twr-server` / engine packages. |
| **PR-C** | **Desktop CI:** remove `.github/workflows/desktop-packages.yml`, `scripts/desktop/`, and the Wails lines in `.github/workflows/verify.yml` (lines 44, 64, 66) and `scripts/prepush.sh` (line 36). Must land before deleting `wails.json` so CI stays green. | Verify workflow and prepush stay green without `make bindings-fresh` / `check-wails-version.sh`. |
| **PR-D** | **Root and composition (one commit; requires PR-A):** remove root `main.go`, `wails.json`, `build/`, `internal/composition`, `tools/fixtures`, composition half of `tools/boundarycheck` (+ `boundary-adapters.tsv`), composition shard row in `scripts/ci-shard*`, obsolete `.golangci.yml` depguard rules, and matching `placement.tsv` / `root-go-baseline.txt` entries. | `go build ./... && go vet ./...` and `make lint` pass. |
| **PR-E** | **Desktop `internal/app/*` leaves:** remove `internal/app/{core,platform,control,commissioner,draft,market,race,research,offers,mflref}` and their `*test` helpers (`internal/app/scoring` only after PR-B). Keep `app/{now,report,season,sources}` and `nowtest`/`seasontest`. | `go build ./... && go test ./...` pass. |
| **PR-F** | **Remaining solo-owner desktop leaves + doc updates (requires Christopher's ruling):** remove `internal/mflref/whatif`, `internal/store/mflref/tradeplans`, `internal/store/rulebook`, `internal/store/playerrecord`, `internal/ingestion/{researchfeeds,sitenews}`; update `CLAUDE.md`, `North_Star.md`, `README.md`, and `desktop-modes.md` in the same PR. **Keep** all 10 must-keep packages (`envelope`, `lineup`, `moves`, `transactions/*`, `contractacts`, `freeagents`). | Christopher's explicit ruling; `go build ./cmd/twr-server ./cmd/twr` and tests pass. |
| **PR-G** | **Desktop code in kept Go packages:** remove `commandcatalog/desktop.go`, `WailsDoor`, `serverclient/desktop*.go` + tests, `serverhttp/desktop*_test.go`, and desktop-only surfaces in `nativegen`; regenerate CLI/PWA tables. | Repeat PR-A byte-diff on kept CLI and browser outputs; `nativegen -check` clean. |
| **PR-H** | **Frontend Wails removal and `go.mod`:** remove `frontend/wailsjs/` (28 files, 18,130 lines), update the 43 references across 24 files in `frontend/src/`, re-point `frontend/vite.config.ts:10` off `wails.json`, drop Wails deps from `go.mod` + `go mod tidy`. | `cd frontend && pnpm tsc --noEmit && pnpm test && pnpm run build` and bundle budget pass. |
| **PR-I** | **Makefile, scripts, and docs cleanup:** move `Makefile:18-19` `LDFLAGS` to `internal/serverversion`; re-point `make build` (a `CLAUDE.md` gate) to the server build rather than deleting it; drop Wails targets/scripts (`check-bindings-fresh.sh`, `check-wails-version.sh`, Wails half of `check-frontend-transport.sh`, stale `bench-league` line); update `SYSTEM_MAP.md` and `system-narrative.md`. | Full `make lint` and verification gate pass once. |
