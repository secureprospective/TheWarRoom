# Handoff

- Baton: Bee, 2026-11-18; PR #356 against `bundle/b5`; merged current `origin/bundle/b5` (#354) into this branch.
- State: League uses the shared hosted ledger query; `[league, filter]` scopes the cache so topic rereads preserve the active chip. Notices remain separate. All frame tabs retained; #354 D60 producers are present from upstream.
- Verification: 13 focused tests; merged-tree Go race tests pass for `internal/app/commissioner`, `internal/commissioner`, and `internal/leagueservice`. Frontend 130 files/931 tests, TypeScript, `make lint`, and `make bloat` pass.
- Slop fixed: removed League's duplicate ledger query; topic reread now retains filter scope. No Go or server contract changes in this lane.
- Next: conclude merge commit, update PR #356 body, push, and wait for CI.
- Blocked: no code blocker. CI still required.
- Open decisions untouched: snooze scope; deputy job visibility; written-request visibility; division naming.
- Tried/rejected: filter kept only in a closure; it was absent from query scope and topic refresh did not preserve it.
