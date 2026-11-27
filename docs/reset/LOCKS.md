# Stage 6 Guard Ledger (`docs/reset/LOCKS.md`)

Initial ledger for BUILD-PLAN Stage 6 ("Lock it in"). Lists automatic guards verified in this repository and marks planned or unverified items as **OPEN**.

## Verified Guards

### Commit time (`pre-commit` in `.pre-commit-config.yaml`)

| Guard | Trigger | What it checks | Source |
|---|---|---|---|
| `golangci-lint-full` | Go changes | Full `golangci-lint` suite (`v2.12.2`, pinned SHA) | `.pre-commit-config.yaml` |
| `gitleaks` | Staged files | Secret scan on staged changes (`v8.30.1`, pinned SHA) | `.pre-commit-config.yaml` |
| `ifaceguard` | Go changes | `make ifaceguard` — blocks `interface{}` / `any` escapes in exported Go signatures | `tools/ifaceguard`, `Makefile` |
| `frontend-controls` | Applicable `frontend/` changes (`src/`, `tsconfig*.json`, `vite.config.ts`, `package.json`, `pnpm-lock.yaml`) | `pnpm --dir frontend test -- src/app/commands/controls.test.ts` — raw control ban (`<button>`, `<a `, `onClick=`, `role="button"`, unallowlisted `onKeyDown=`), `Act` tooltip/semantics, and TS/TSX max line length (`<= 110` chars under `frontend/src/app`) | `frontend/src/app/commands/controls.test.ts` |
| `frontend-tsc` | Applicable `frontend/` changes (`src/`, `tsconfig*.json`, `vite.config.ts`, `package.json`, `pnpm-lock.yaml`) | `pnpm --dir frontend exec tsc --noEmit` — TypeScript typecheck | `frontend/tsconfig.json` |

### Push time (`pre-push` via `scripts/prepush.sh`)

| Guard | Trigger | What it checks | Source |
|---|---|---|---|
| `make lint` | Go, gate-script, or unknown-base push | `ifaceguard`, `filelen` (>600 lines report), `bloat` (`scripts/bloat.sh` vs `.bloat-baseline`), `placement` (`check-placement.sh`, `check-root-ratchet.sh`, `check-line-budget.sh`, `check-source-branches.sh`), `frontend-transport` (`check-frontend-transport.sh`), `boundary` (`tools/boundarycheck`), `golangci-lint run ./...` | `Makefile`, `scripts/` |
| `go build ./...` + changed package tests | Go changes | Compiles all Go packages; runs `go test -shuffle=on -timeout 10m` on directly changed packages | `scripts/prepush.sh` |
| Frontend push suite | `frontend/*` changes | `pnpm test` (full Vitest suite), `pnpm run build` (`scripts/check-path-casing.mjs && tsc && vite build`, including `sw.js` asset manifest check in `frontend/vite.config.ts`), `node scripts/bundle-budget.mjs` (`tsc --noEmit` is not duplicated here because `frontend-tsc` runs at `pre-commit` and `pnpm run build` also runs `tsc`) | `scripts/prepush.sh`, `frontend/package.json`, `frontend/vite.config.ts` |

### CI & Full Verification (`.github/workflows/verify.yml` and `make verify`)

| Guard | What it checks | Source |
|---|---|---|
| Secret scan (full history) | `gitleaks git --redact --no-banner .` | `.github/workflows/verify.yml` |
| Dependency audits | `pnpm audit --prod` and `pnpm audit` (blocking) | `.github/workflows/verify.yml` |
| Go lint & vuln scan | `make lint` and `govulncheck ./...` | `.github/workflows/verify.yml`, `Makefile` |
| Full frontend checks | `pnpm tsc --noEmit`, `pnpm test`, `pnpm run build`, `node scripts/bundle-budget.mjs`, `node scripts/csp-check.mjs`, `node scripts/csp-browser-check.mjs` | `.github/workflows/verify.yml`, `Makefile` |
| Go race suite | `go test -race` across 4 shards (`scripts/ci-shard.sh`) | `.github/workflows/verify.yml` |
| Release publisher tests | `scripts/release/test.sh` | `.github/workflows/verify.yml` |

*(Note: `frontend/scripts/card-type-gate.mjs` also exists in the repo as a standalone type-gate script.)*

## Open / Planned Guards

| Item | Status | Notes |
|---|---|---|
| Nov 20 commissioner admission timing | **VERIFIED** | The recorded failure was in `hostedRules.test.tsx`. The test asserted on the Change Lineup lock button before Rules rendered. Commit `877683d50` waits for the button. Both commissioner roles still run in Vitest. The focused admission and controls suites passed 31 checks on Nov 26. |
| Additional Stage 6 build guards | **OPEN** | `docs/reset/BUILD-PLAN.md` Stage 6 #2: add one guard per class of bug found (modeled on the `sw.js` asset manifest check in `frontend/vite.config.ts`). |
| Daily cleanup PR job (T665) | **OPEN** | `docs/reset/BUILD-PLAN.md` Stage 6 #4 (Hydrogen from Hermes, after reset settles). |
| Weekly habit check | **OPEN** | `docs/reset/BUILD-PLAN.md` Stage 6 #5 (stale cards, done lines, cold-start test). |

## Planned Guards by Nov 26 Bug Class (Stage 6)

Appended as a separate trailing section so PR `#544`'s updates to earlier rows in `Verified Guards` and `Open / Planned Guards` remain disjoint. When integrating both branches, retain `#544`'s earlier rows above, keep this section at the end, and regenerate `docs/architecture/line-budget-baseline.txt` with `scripts/check-line-budget.sh --write`.

| Bug Class | Current Evidence (Existing vs. Planned) | Intended Check / CI Entry Point |
|---|---|---|
| Observed async admission assertions | **Existing:** `frontend/src/app/commissioner/hostedRules.test.tsx` (lines 300, 323) uses `vi.waitFor` (from `877683d50`), fixing the Nov 20 synchronous assertion failure (`bundle-alpha-1120/vitest.log`). **Planned:** Automated lint/AST check across `frontend/src/**/*.test.*` for synchronous DOM assertions on lazy views without `waitFor`/`findBy*` (not yet implemented). | `pnpm --dir frontend test` (`scripts/prepush.sh`, `.github/workflows/verify.yml` `checks` job) |
| Unused phone reads | **Existing:** `frontend/src/app/phone/LiveTab.test.tsx` (`default Live path skips Home-only reads...`) and `frontend/src/app/shell/homeReads.test.ts` fail on any call to the 10 Home-only commands; `homeReads.test.ts` `rejectingPort()` does not reach `calendar.read`/`schedule.read` when `season === undefined`. **Planned:** Extend port-allowlist / fanout coverage to all Phone tabs and `season !== undefined` paths (not yet implemented). | `pnpm --dir frontend test` (`src/app/phone/LiveTab.test.tsx`, `src/app/shell/homeReads.test.ts` in `scripts/prepush.sh`, `.github/workflows/verify.yml` `checks` job) |
| Separate stale seat selection (`T1847`) | **Existing:** Unguarded. `twr-stage5-phone-review-1126/report.md` found `LiveBody` (`LiveTab.tsx:39`) is keyed only by `league.id`, so `mine` becoming `''` (`LiveTab.tsx:68`) or switching franchises (`LiveTab.tsx:55-61`) leaves prior `failure`/`brief` state, and `readLiveBrief` lacks `AbortSignal`. Queued as `T1847`. **Planned:** Seat-transition & `AbortSignal` regression test in `LiveTab.test.tsx` (not yet implemented). | `pnpm --dir frontend test -- src/app/phone/LiveTab.test.tsx` (`scripts/prepush.sh`, `.github/workflows/verify.yml` `checks` job) |
| Stale plan paths | **Existing:** Unguarded. Nov 26 audit (`#541`, `87634a45e`) required manual audit/fix of 59 flagged rows (43 unique missing-file references) in v3 stub plan docs. **Planned:** Script verifying repo-relative paths cited in `docs/reset/` and `docs/plans/` exist on disk (not yet implemented). | `scripts/check-doc-paths.sh` in `make lint` (`placement` target, `.github/workflows/verify.yml` `checks` job) |
| Generated budget collisions | **Existing:** `scripts/check-line-budget.sh` enforces `docs/architecture/line-budget-baseline.txt` in `make lint` (`placement`), `scripts/prepush.sh`, and CI `checks`; parallel PRs (`#542`, `#543`) still collide on the shared generated baseline and require manual rebase + `--write`. **Planned:** Merge-friendly baseline check or post-merge baseline regeneration (not yet implemented). | `scripts/check-line-budget.sh` in `make lint` (`placement` target, `.github/workflows/verify.yml` `checks` job) |
| Lint cache cross-worktree contamination | **Existing:** Unguarded. `#542` first push hit stale `golangci-lint` findings from a shared cache across worktrees; isolated cache retry passed. **Planned:** Per-worktree `GOLANGCI_LINT_CACHE` / `--cache-dir` isolation in local lint runners (not yet implemented). | `Makefile` (`lint` target) and `.pre-commit-config.yaml` (`golangci-lint-full` hook) / `scripts/prepush.sh` |
