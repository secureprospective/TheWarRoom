# Handoff

- Baton: Bee, 2026-11-17 (Rules tab lane; PR not opened).
- State: Rules tab now presents server-backed settings in six boards, displays recorded source and pending effective timing, and exposes edits only for supported rule fields through the existing hosted preview/review/execute path.
- Verified: full frontend suite (920/920), frontend typecheck, targeted Rules/Control Room tests (20), and `make bloat` pass. `make lint` is retrying after a concurrent-lint lock collision.
- Slop fixed: none identified.
- Next: finish lint, inspect diff, commit, and open PR against `bundle/b5`. Other Bundle 5 lane launchers were active at tidy-check; left them untouched.
- Blocked: preset comparison read is available in desktop bindings but not exposed by the hosted room read path; kept out rather than adding a side-door read. House rules/written requests likewise have no hosted server reads here.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
- Tried/rejected: none.
