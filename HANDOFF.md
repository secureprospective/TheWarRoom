# Handoff

## Baton
Bee / APP-MERGE-6 — 2026-11-02, build/INTEGRATE-1102. Local only; no remote.

## Where it stands
- Reference schedule/current/played-lineup decisions now live in app/mflref/migration;
  composition supplies resources and worker lifetime. Timing, transactions and cutover preserved.
- Control Room uses shared Act controls; 53-method commissioner transport includes SignInFacts.
- Commits: 68d39d77f (extraction), a9ad04e29 (fixture clock), 1f00dd7db (frontend).
- Final targeted race: service 2 PASS, composition migration cases 4 PASS. Earlier extracted
  composition candidate: 846 PASS; not a final-tree full-package rerun.
- src/app: 1829 PASS / 2 inherited shell guard failures. tsc and plain make build pass.
- Full evidence and scoped review: docs/build-handoffs/APP-MERGE-6.md.

## Next move / blocked on
- Boundarycheck now exposes existing native history → ingestion → MFL transitive coupling.
- Strict controls guards still find HomeBrief/homeTiles raw controls and 86 long shell lines.
- Follow-up separation/control conversion, independent review and disposable-profile UI testing.
- Prior unresolved operator rulings/transaction-follow activation: APP-MERGE-4 debt remains.
- APP-MERGE-5.md was absent from this checkout.

## Tried and rejected
No fence exemptions, keyboard allowlist expansion, engine/model changes, remote contact or
full Go repository suite. Did not turn the named room repair into an incidental Home rewrite.
