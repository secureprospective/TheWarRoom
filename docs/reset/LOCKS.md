# Stage 6 Guard Ledger (`docs/reset/LOCKS.md`)

Initial ledger for BUILD-PLAN Stage 6 ("Lock it in"). Lists automatic guards verified in this repository and marks planned or unverified items as **OPEN**.

## Verified Guards

### Commit time (`pre-commit` in `.pre-commit-config.yaml`)

| Guard | Trigger | What it checks | Source |
|---|---|---|---|
| `golangci-lint-full` | Go changes | Full `golangci-lint` suite (`v2.12.2`, pinned SHA) | `.pre-commit-config.yaml` |
| `gitleaks` | Staged files | Secret scan on staged changes (`v8.30.1`, pinned SHA) | `.pre-commit-config.yaml` |
| `ifaceguard` | Go changes | `make ifaceguard` — blocks `interface{}` / `any` escapes in exported Go signatures | `tools/ifaceguard`, `Makefile` |
| `frontend-controls` | Applicable `frontend/` changes (`src/`, `tsconfig*.json`, `package.json`, `pnpm-lock.yaml`) | `pnpm --dir frontend test -- src/app/commands/controls.test.ts` — raw control ban (`<button>`, `<a `, `onClick=`, `role="button"`, unallowlisted `onKeyDown=`), `Act` tooltip/semantics, and TS/TSX max line length (`<= 110` chars under `frontend/src/app`) | `frontend/src/app/commands/controls.test.ts` |
| `frontend-tsc` | Applicable `frontend/` changes (`src/`, `tsconfig*.json`, `package.json`, `pnpm-lock.yaml`) | `pnpm --dir frontend exec tsc --noEmit` — TypeScript typecheck | `frontend/tsconfig.json` |

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
| 11-20 phone test failure / quarantine | **OPEN** | `docs/reset/BUILD-PLAN.md` Stage 6 #3 notes a phone test that failed once on 11-20 during a push and passed on retry (referenced in Claudebox `/root/TheWarRoom-RESUME-desktop-v3.md`). Exact failing test and failure mode are not yet identified in this worktree; phone tests remain unaltered until identified. |
| Additional Stage 6 build guards | **OPEN** | `docs/reset/BUILD-PLAN.md` Stage 6 #2: add one guard per class of bug found (modeled on the `sw.js` asset manifest check in `frontend/vite.config.ts`). |
| Daily cleanup PR job (T665) | **OPEN** | `docs/reset/BUILD-PLAN.md` Stage 6 #4 (Hydrogen from Hermes, after reset settles). |
| Weekly habit check | **OPEN** | `docs/reset/BUILD-PLAN.md` Stage 6 #5 (stale cards, done lines, cold-start test). |
