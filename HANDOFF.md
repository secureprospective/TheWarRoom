# Handoff

## Baton
Bee — 2026-10-28, `session/terra-franchise-scope`. Do not push without direction.

## Where it stands
- Scoped reserve exceptions, player decision exceptions and contract quotes by franchise for per-side pools.
- Legacy player-only exceptions remain valid in single-pool leagues, but do not cross per-side copies.
- Contract quote schema changed; frozen command metadata updated. Fixture rewrites contained unrelated
  timestamps and were discarded.

## Next move
Review commits and run the full race suite on a quiet machine. Verify the reserve preview and contract
quote flows in the UI; the new franchise argument is available at the API boundary.

## Blocked on
No upstream configured, so `git pull --ff-only` could not run. No push attempted.

## Tried and rejected
Fixture regeneration changed volatile timestamps and revisions without an intended wire change.

## Debt left
Full race suite timed out under concurrent workload; targeted race tests passed. UI affordances for selecting
franchise in quotes and exception forms have not been implemented or verified.
