# Handoff

- Baton: Bee, 2026-11-17.
- State: hosted Negotiations drawer filters to in-motion trades and shows server status/expiry. Accept/decline/own-withdraw use the shared `HostedWorkflows` preview-confirm path and reread. Live offers and saved drafts load into the existing desk; no decline note or new state.
- Checks: frontend 914/914; `pnpm exec tsc --noEmit`; `make lint`; `make bloat`; tidy-check clean. No Go files changed, so no touched Go package test.
- Slop fixed: consolidated duplicate hosted asset/detail parsing; removed misleading grab cursor/touch lock from non-draggable proposal cards.
- Next: commit/push and open PR against `bundle/b1`; wait for CI green.
- Blocked: none.
