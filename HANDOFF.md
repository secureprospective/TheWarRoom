# Handoff

## Baton
Review — 2026-10-29, `session/p9-s3`. Coherent what-if/watcher slice; no push authorized.

## Where it stands
- Frozen contracts and trade plans moved under `internal/app/mflref`; local watcher
  polling, landing, roster-plan checks and binding implementation belong to planwatch.
  Platform promotes those methods for desktop compatibility (one explicit debt pair).
- Session model/lifecycle belongs to `internal/mflref/whatif`; raw SQL, mirror types
  and the unchanged shared whatif.db migration sequence remain in state. No stores
  import another store. Old literal sessions/plans survive file close/reopen.
- Fence debt 50 → 47: trade source pair and three archived commstest source pairs
  become reference-owned; platform compatibility adds one pair. Every pair has a reason.
- Native domain and pure engine/model roots get transitive production-import checks; legacy debt and
  composition cannot launder a native path. Mixed adapters remain named direct debt.
- Exactly 13 authorized composition fingerprints refreshed; imports/selectors only.
  No commands retired, schemas renamed, engine/native policy changed or MFL writes.

## Evidence
- Scoped race batches: 325 top-level PASS (220 packages/tool, 98 touched composition,
  7 migration); 11 preparation fixtures + 11 stored roundtrips PASS.
- Restart/stale mirror, raw old-data reopen, native-rule/MFL separation, guarded
  source observation and no-MFL-write checks PASS. No GUI or full-suite claim.
- Lint, boundary (159 fingerprints), placement, generated doors and frontend build PASS.
  Wails regenerated without output changes; bindings-fresh PASS. Commit uses hooks.
- Worktree-only details, logs and move ledger: `_x/REPORT.md`, `_x/MOVED-s3.md`.

## Next move / debt
- Review this coherent first half. Remaining mixed ownership: envelope rule predicates,
  offers/bids plan policy, state trade plans/schema, moves persistence; shared lineup
  rule policy; held native-seed source types and other direct source-reading adapters.
- Do not claim all mixed application adapters have native transitive isolation.
  Native roots are explicit in `docs/architecture/boundary-mfl.tsv`; expand only after
  removing their held-source debt. Platform is a compatibility facade, not native core.
- Reviewer GUI check: existing MFL what-if survives restart; a newer mirror says stale
  until Reset; handed-off owner plan lands after source refresh, with no MFL submission.

## Tried and rejected
- A second session store importing state: violates store ownership. Keep raw record SQL
  next to its shared migration; move the reference model/lifecycle instead.
- Standalone tiny binding file: bloat ratchet rejected it. Bindings live with plan actions.
