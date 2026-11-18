# Handoff

- Baton: ClaudeBox, 2026-11-17 (PR #350 open against `bundle/b5`; CI pending).
- State: Rules tab now presents server-backed settings in six boards, displays recorded source and pending effective timing, and exposes edits only for supported rule fields through the existing hosted preview/review/execute path.
- Verified: full frontend suite (920/920), frontend typecheck, targeted Rules/Control Room tests (20), `make bloat` pass. `make lint` retries hit another lane's concurrent golangci-lint; no bypass. PR CI pending.
- Slop fixed: none identified.
- Next: ClaudeBox reviews PR #350 and resolves CI/lint result. Other Bundle 5 lane launchers were active at tidy-check; left them untouched.
- Blocked: preset comparison read is available in desktop bindings but not exposed by the hosted room read path; kept out rather than adding a side-door read. House rules/written requests likewise have no hosted server reads here.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
- Tried/rejected: none.
