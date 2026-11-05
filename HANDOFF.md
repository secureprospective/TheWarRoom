# Handoff

## Baton
Bee → next worker, 2026-11-05, `session/trade-floor-cut` (local only).

## Where it stands
Extracted the existing offer card into `frontend/src/app/shell/OfferCard.tsx`; the inspector and old desk now share it. TypeScript and the 11 trade desk tests pass. The preliminary trace remains incomplete; no rail, registry, draft controls or HQ routing have been cut or moved. OfferCard's reject/withdraw actions still call the old builder and need a destination before its removal.

## Next move
Move the native draft actions and owner pick into War Room Draft using the existing tray/IPC, with league and commissioner-role isolation verified. Repoint HQ's player action and all old draft navigation. Complete the trace and row dispositions before cutting the old node. Run the brief's gates, copied-profile checks and soak update.

## Blocked on
Configured bundle remote is unavailable; pull and push cannot run. Existing app-league trade route displays Not wired before the old draft room can render; do not assume retained controls are accessible.

## Tried and rejected
Deleting the node immediately would strand dependencies and lacks a replacement for draft actions. No live actions were executed to establish their current behavior.
