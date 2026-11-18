# Handoff

- Baton: ClaudeBox, 2026-11-18 (PR #350 open against `bundle/b5`; CI status to verify).
- State: Hosted Rules board is server-backed; rule rows route by explicit section-label table, unknown sections go to Other. Owners see the commissioner ruling guidance. Preset comparison, house rules and written requests remain unavailable without hosted reads.
- Verified: targeted frontend tests (21/21), frontend typecheck, `make lint`, `make bloat`, tidy-check.
- Slop fixed: removed name-substring rule-board classification, which could duplicate/misclassify rows; added explicit fallback board and coverage. Restored the required owner ruling text.
- Next: push these fixes to PR #350 and check CI.
- Blocked: preset comparison read is available in desktop bindings but not hosted room reads; no side-door read added. House rules/written requests have no hosted reads.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
- Tried/rejected: none.
