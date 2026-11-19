# Handoff

- Baton: Bee, lane `pulse-mclaims`, 2026-11-19.
- State: Branch `session/pulse-mclaims`, based on clean `80dac0754` (`bundle/b9`); Q5 approves owner claim reorder/cancel. Added private catalog commands, owner/deputy/token permissions, ledger projection, claim-position reads, and due-window priority. Maintenance is limited to same-principal claims; reorders preserve other principals' queue slots. Cancellation and processing remove claims from pending order. Preview now uses the authenticated actor for private claim projections. `make bindings` regenerated `position` in `frontend/wailsjs/go/models.ts` (Wails printed existing generic-type “Not found” diagnostics; no nonzero exit). `make lint` now passes. The latest affected Go race packages passed before a final validation refactor; rerun them. Another race suite is active; wait on `heavy-ok.sh`. No commit/PR.
- Next: Wait for active suites to finish; rerun affected Go race packages, make lint, Vitest and tsc; review diff, then commit/push and create one PR into `bundle/b9`.
- Blocked on: none.
- Tried and rejected: reusing `waivers.order.set` (commissioner priority, not owner claim list); client-only mutation (not persistent and cannot affect server processing); changing `bid.*` or adding the blind-bid sheet (outside approved scope).
