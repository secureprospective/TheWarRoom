# Handoff

## Baton
Sol → reviewer, APP-FIX-1, 2026-11-02, `build/FIX-1`. Local only.

## Where it stands
Fixed the audit test's fake v4 schema (it retained v13 live tables). Production
migration remains strict and atomic. Added adoption/rollback/retry/reopen evidence;
refreshed only intended provenance fields and three batch-stat schema hashes.
Real-history copy migrated v12 → v13, reopened cleanly, then was deleted.
Full evidence and verification: `docs/build-handoffs/APP-FIX-1.md`.

## Next move
Review the fixture correction and golden additions; integrate locally if accepted.

## Blocked on
No desktop launch attempted; Tom owns the display. No remote operations permitted.

## Tried and rejected
`IF NOT EXISTS` on v13: would hide an artificially mismatched version/schema;
the real v12 copy and atomic rollback/retry work with the existing migration.
