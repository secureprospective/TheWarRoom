# T487 close-books handoff — incomplete

## Baton
ClaudeBox / reviewer — 2026-10-23. Bee investigated `session/t487-close-books`; no performance fix delivered.

## Where it stands
- Production code unchanged. No schema or ledger-format change.
- Evidence and final D9/D1 five-run medians: `docs/build-handoffs/T487_Close_Books_2026-10.md`.
- Existing rollover plans already carry rules/assets, but not every save/read fold.
- Frozen frontend install/build passed. Final `go test -race ./...` failed at the default 10-minute `internal/composition` package timeout; history/commissioner/governance passed.
- No differential test for closed seasons, no GUI/live gate, no merge. Performance budgets remain unmet.

## Next move
Define a summary-aware replay contract retaining authoritative history; add identical several-closed-season fixtures for summary/full differential tests and performance comparisons. Review with ClaudeBox before wider storage decisions.

## Blocked on
Growth fixtures have no closed seasons, so a closed-season summary alone cannot improve those measured paths. No safe general summary implementation was completed. Any schema change needs Christopher's sign-off before merge.

## Tried and rejected
Mutating private governance replay state avoided per-event copies and passed targeted race tests, but did not meet budgets or implement the approved summary. Reverted completely. Do not treat the unchanged-code rerun as before/after evidence of an improvement.
