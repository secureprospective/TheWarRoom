Baton: AGY — 2026-11-27
State: T710 PR-F extended on PR #551: removed all internal/app/mflref/* (bids, contracts, contractstest, migration, planwatch, seed, trades), internal/mflref/whatif, internal/store/mflref/tradeplans, and internal/store/rulebook. Kept envelope, lineup, moves, mflmirror, live what-if, and rulebook SQLite tables. No schema/migration or frontend shell change.
Next: review PR #551 and its CI. Do not merge without Christopher's sign-off.
Blocked: none. Build, vet, lint, line budget, and tests verified on branch.
