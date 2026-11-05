# Handoff

## Baton
Bee → next worker, 2026-11-05, `session/trade-floor-cut` (local only).

## Where it stands
Preliminary trace in `docs/audit/TRADE-FLOOR-TRACE-2026-11-05.md`; explicitly incomplete. No code cut, registry change, soak change, build or live check. Trade node still exists.

## Next move
Complete transitive trace and row-by-row mock coverage before cutting. Rewire HQ ADD TO TRADE and inspector/preset/phase routes, preserving live shared code and CLI IPC; examine old commissioner draft controls before retiring them. Then carry out the brief's gates and profile-copy checks.

## Blocked on
Configured bundle remote is unavailable; pull and push cannot run.

## Tried and rejected
Deleting the node immediately: old desk's OfferCard is used by the inspector, HQ routes a player into the old builder, and old draft room contains commissioner controls not established in the new room.
