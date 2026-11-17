# Handoff

- Baton: Bee, 2026-11-17.
- State: Reviewed the approved private-entry task and PRs #323/#313. Hosted league schema is 15; `history.leagueMigrations()` is versioned and `serverversion.releases()` is append-only. Adding the requested migration makes schema 16 and requires the next contract row, Release 9. Draft PR #325 is open; no product code or tests changed. Push preflight passed; `gh pr checks 325` reports no checks.
- Next: Wait for authorization to add server Release 9 / schema 16; do not implement the migration before that decision.
- Blocked on: The task's explicit fallback is triggered: `TestCurrentContract` requires the current schema to match a new sequential release row, and shipped release rows must not be edited. `tidy-check` also reports six dangling Chromium symlinks in unrelated `/home/chris/scratch/bee-resume-v14-*` folders; left untouched under the write restriction.
- Tried/rejected: Reusing Release 8 would change an immutable historical row and violate the contract test; no migration or local desktop schema was touched.
