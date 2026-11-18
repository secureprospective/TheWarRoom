# Handoff

- Baton: ClaudeBox, 2026-11-18 (PR #350 open against `bundle/b5`; latest CI pending).
- State: Hosted Rules board is server-backed; rule rows route by explicit section-label table, unknown sections go to Other. Owners see the commissioner ruling guidance. Preset comparison, house rules and written requests remain unavailable without hosted reads.
- Verified: full frontend suite (921/921), frontend typecheck, targeted Rules/Control Room tests (21), `make lint`, `make bloat`, tidy-check; changed-package pre-push checks passed.
- Slop fixed: removed name-substring rule-board classification, which could duplicate/misclassify rows; added explicit fallback and test. Restored required owner guidance. PR body lists both fixes.
- Next: ClaudeBox reviews PR #350 and checks CI.
- Blocked: preset comparison read is available in desktop bindings but not hosted room reads; no side-door read added. House rules/written requests have no hosted reads.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
- Tried/rejected: none.
