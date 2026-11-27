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
| `make lint` | Go, gate-script, or unknown-base push | `ifaceguard`, `filelen` (>600 lines report), `bloat` (`scripts/bloat.sh` vs `.bloat-baseline`), `placement` (`check-placement.sh`, `check-root-ratchet.sh`, `check-line-budget.sh`, `check-source-branches.sh`, `check-stub-plan-paths.sh`), `frontend-transport` (`check-frontend-transport.sh`), `boundary` (`tools/boundarycheck`), `golangci-lint run ./...` | `Makefile`, `scripts/` |
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
| Nov 20 commissioner admission timing | **VERIFIED** | The recorded failure (`/home/chris/fleet/runs/bundle-alpha-1120/vitest.log` lines 1525–1549) was in `hostedRules.test.tsx`, which asserted on `Change Lineup lock` before Rules rendered. Commit `877683d50` waits for the button (`vi.waitFor`). Decision: preserve this enabled test (`it.each(['Commissioner', 'Co-commissioner'])`). On Nov 26 (`11a0d8350`), `hostedRules.test.tsx` (16), `LiveTab.test.tsx` (18), and `controls.test.ts` (15) all passed (49/49). |
| Phone Live request-fanout regression (`#543`) | **VERIFIED** | Commit `11a0d8350` narrows `LiveTab` default brief reads to pulse, results, and matchup and adds the focused regression in `frontend/src/app/phone/LiveTab.test.tsx` (`default Live path skips Home-only reads and shows scores, lineups and errors`), asserting zero Home-only reads (e.g. `native.assets.read`, `trades.read`, `bids.read`, `draft.read`, `schedule.read`) on initial load, invalidation, and `results.read` failure. Old-reader deliberate-break rejection verified in `/home/chris/fleet/runs/twr-stage5-phone-live-1126/old-reader-deliberate-break.log` (`native.assets.read` rejected as expected). |
| Additional Stage 6 build guards | **OPEN** | `docs/reset/BUILD-PLAN.md` Stage 6 #2: add one guard per class of bug found (modeled on the `sw.js` asset manifest check in `frontend/vite.config.ts`). |
| Daily cleanup PR job (T665) | **OPEN** | `docs/reset/BUILD-PLAN.md` Stage 6 #4 (Hydrogen from Hermes, after reset settles). |
| Weekly habit check | **OPEN** | `docs/reset/BUILD-PLAN.md` Stage 6 #5 (stale cards, done lines, cold-start test). |

## Planned guards by Nov 26 bug class

| Bug class | Evidence | Planned check |
|---|---|---|
| Admission assertions run before Rules renders | The Nov 20 failure in `hostedRules.test.tsx` was fixed by waiting for the button in `877683d50`. Both roles passed in the Nov 26 run. | Add a test with a delayed Rules response that checks loading first and confirms the control becomes usable afterward. Run it in the existing Vitest CI suite. |
| Phone Live waits on unrelated Home requests | PR #543 added a regression that rejects the old reader on `native.assets.read` during initial load, refresh or a results-read failure. | Keep the existing transport assertions in `LiveTab.test.tsx` and run them in Vitest CI whenever the Live reader changes. |
| Phone Live retains the previous seat state | Argon found that switching or losing a seat within the same league can retain the previous score, lineup or error. Tracked as T1847. | Test same-league switches and seat loss while the old result is delayed. Verify the previous seat clears and its late response cannot restore its state. Run in `LiveTab.test.tsx` after the fix. |
| Desktop stub plan refers to retired source files | PR #541 replaced 59 stale references after a Git tree audit. | **EXISTS** (`scripts/check-stub-plan-paths.sh`, wired into `make placement` / `make lint`; fixture test in `scripts/check-stub-plan-paths.test.sh`). Checks source paths (backticked or plain) in `docs/ui/v3/V3-STUB-PLAN.md` against `git ls-files`. |
| Parallel PRs conflict on the generated line budget | PRs #542 and #543 conflicted on the shared baseline. The #543 refresh verified that only the generated file changed. | Preflight the merge tree for queued branches. When this file conflicts, regenerate on the new base and verify the remaining feature diff is identical. |
| Local lint findings leak between worktrees | An isolated lint cache cleared stale diagnostics pointing at another checkout on #542 and #543. | Run the same lint gate from two worktrees with distinct `GOLANGCI_LINT_CACHE` directories. Verify each reports only paths from its own checkout. |
| A later release invalidates an earlier watch result | After v104, the previous version watchers appeared overdue because they used the mutable latest-release pointer. Their archived health records were still valid. | Test two successive release fixtures. Verify the older check still passes using its archived status and expected commit and backup evidence. |
