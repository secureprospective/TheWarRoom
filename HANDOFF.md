# Pulse L6

- Baton: head-brain review, 2026-11-14; branch session/pulse-l6.
- State: extended frontend/src/app/data/query.ts and session.ts with hosted session-owned tagged reads;
  clientCache.ts is the public-only IndexedDB adapter. Local overloads remain compatible.
- Next: L8 wires one hosted session, identity, C2 policies/dependencies, known/tags reader adapters,
  active hooks and applyTopics-before-notify. Reconcile versions before reconnect cache hits.
- Blocked on: no implementation blocker; desktop/Pulse integration is outside this lane.
- Evidence: targeted query/session/hosted/readability tests, TypeScript and production frontend build pass.
- Rejected: second cache owner; private persistence; clock-derived current-week keys.
