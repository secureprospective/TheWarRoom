# Handoff

- Baton: Bee on `lane/lineup12`; PR #408 is open into `bundle/b9`, not merged.
- State: Import reads current MFL lineups through the validated `livescoring` adapter, merges completed-week starters, previews rows, and records ordinary `LineupCorrect` events atomically. Missing lineups remain absent; import IDs do not use the ongoing-sync prefix. No schema change.
- Verification: Full `make test`, `make lint`, Vitest (154 files/1028 tests), `pnpm tsc --noEmit`, push hook `verify`, and all PR #408 CI checks passed. Mutant proof: suppressing imported lineup commands failed the integration test at the recorded-lineup assertion; mutation restored. `tidy-check.sh` clean.
- Next: Christopher reviews PR #408; do not merge.
- Blocked on: none.
- Tried/rejected: `lineup.sync.*` IDs misrepresent a one-time import as ongoing sync and affect missed-lineup flags; use `mfl-import.lineup.*`. Batch initially rejected `LineupCorrect`; gather its normal lineup facts so all import events remain atomic.
