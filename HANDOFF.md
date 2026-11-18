# Handoff

- Baton: Bee, 2026-11-18.
- State: Bundle 2 Needs You collector implemented; no Home UI. Reads authenticated per-league franchise/role, calls `readHomeBrief` once per native app league in parallel, isolates failures, and sorts deadlines. Reference-only mirror is skipped because it has no native HomeBrief port.
- Tests: `cd frontend && pnpm exec vitest run src/app/shell/homeNeedsYou.test.ts src/app/commissioner/hostedMarkets.test.ts` (7 passed); `pnpm exec tsc --noEmit` passed; `make lint` passed.
- Next: commit, push branch, open PR to `bundle/b2`, and report CI state. No merge.
- Blocked on: `tidy-check.sh` found an unrelated active `b4shell` launcher in `/home/chris/work/TheWarRoom`; left it untouched. Its run directory is outside this task's write scope.
- Tried/rejected: no server read or Home alert-band reuse; unavailable per-league reads become one item.
