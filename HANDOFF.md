# Handoff

- Baton: ClaudeBox, 2026-11-18; PR #356 open against `bundle/b5`; CI green at `d5ac85603`.
- State: League uses the shared hosted ledger query; `[league, filter]` scopes the cache so topic rereads preserve the active chip. Notices stay separate. Frame tabs retained; current bundle/b5 head #354 is merged.
- Verification: Go race tests pass for `internal/app/commissioner`, `internal/commissioner`, and `internal/leagueservice`; frontend 130 files/931 tests and TypeScript pass; `make lint` and `make bloat` pass.
- Slop fixed: removed League's duplicate ledger query; topic rereads preserve the filter. No Go or server-contract changes in the League lane.
- Next: ClaudeBox review and merge PR #356 into `bundle/b5`.
- Blocked: none. Lane decisions untouched: snooze scope; deputy job visibility; written-request visibility; division naming.
- Tried/rejected: filter only in a closure; it was absent from query scope and topic refresh did not preserve it.
