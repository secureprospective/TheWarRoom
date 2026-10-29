# Handoff

## Baton
Review — 2026-10-29, `session/p9-s4`. No push authorized.

## Where it stands
- MFL envelope predicates/checks, lineup policy, bid-plan checks, move SQL, trade-plan
  SQL and held-seed translation now have explicit reference owners. Stored names,
  IDs, JSON keys and migration sequence stay unchanged. State owns shared migrations;
  composition injects its initializer into the independent trade-plan store.
- Exact command/package allowlists reject new commands/packages/imports. Native
  production closure still cannot reach fenced reference code through debt or composition.
- Direct exceptions 47 → 44, each explained. Five MFL-only fixture helpers are fenced.
  Mixed hosts/source readers remain explicit debt, not a claim of complete isolation.
- Eleven source fingerprints refreshed: import paths/selectors only, except the
  explicitly approved trade-plan initializer injection. No function fingerprints changed.

## Evidence
- Race: 579 top-level PASS / 1,247 including subtests: 455 package/tool, 112 touched
  composition + whatif/moves, 12 migration/history. Includes literal old-data reopen,
  migration preparation/roundtrip fixtures and synthetic fence rejection.
- Lint, boundary (159 fingerprints), placement, bindings freshness and frontend build
  PASS. Existing large-chunk warning; no generated binding differences.
- Full composition/history attempt exceeded the outer 600-second budget; replaced
  with completed scoped batches, not counted as a full-suite pass. No GUI/live-source claim.
- Logs/report and exact fingerprint list: `_x/REPORT-s4.md`, `_x/s4-refreshed.txt`.
  Move ledger: `_x/MOVED-s4.md` (worktree-only because writes outside are forbidden).

## Next move / named debt
Review shared source normalization/transport, rulebook mirror storage, standings
services and mixed control/core/offers/draft/market/now/race/scoring/season adapters.
Their existing ports are preserved; do not relabel all as composition exemptions.
Shared state session records/migrations and engine bid math remain shared on purpose.
Reviewer GUI: old what-if/plan survives restart, stale mirror requires Reset, owner
handoff remains local and watching never submits to MFL.

## Tried and rejected
Store-to-store initialization and duplicated migrations remain rejected. Inject the
single migration owner instead. No wildcard fence exemptions or stored-data rewrites.
