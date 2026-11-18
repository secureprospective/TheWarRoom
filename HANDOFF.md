# Handoff

- Baton: Bee, 2026-11-18; PR #356 against `bundle/b5`; merge of #351 is in progress.
- State: League uses the shared hosted ledger query; `[league, filter]` scopes the cache so league-topic rereads preserve the active chip. Notices remain separate. All frame tabs retained.
- Verification: 13 targeted tests pass; full frontend suite 929/929, `pnpm exec tsc --noEmit`, `make lint`, and `make bloat` pass.
- Slop fixed: removed League's duplicate ledger query; topic reread now retains filter scope. No Go or server contract changes.
- Next: finish merge with latest `origin/bundle/b5` (#354), rerun affected checks, then commit, push, and update #356.
- Blocked: no code blocker. CI still required.
- Open decisions untouched: snooze scope; deputy job visibility; written-request visibility; division naming.
- Tried/rejected: filtered ledger scope captured only in a closure; topic rereads lost the selected filter.
