# Handoff

- Baton: Christopher / ClaudeBox decision, 2026-11-16.
- State: Hosted private-notes part 2 is blocked on the versioned storage contract. Reviewable change is documentation only; no functional code or release metadata changed.
- Next: Decide whether to authorize the personal table's league-schema migration and matching release row, then implement and verify the owner-only catalog path.
- Blocked on: `docs/plans/HOSTED-PRIVATE-NOTES-STORAGE-QUESTION.md` records the question and twice-reproduced initializer failure (hosted schema 15, desktop notes initializer supports 2).
- Tried/rejected: Existing personal initializer on a disposable hosted file refuses the schema. Unversioned DDL, historical migration edits and shared-evidence storage would bypass the contract; not attempted. Local resumable candidate is in `.scratch/part2-candidate.patch` and `.scratch/part2-personal-notes.go`, unverified.
