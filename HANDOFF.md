# Handoff

- Baton: Bee, lane `lineup12`, 2026-11-19.
- State: Based on `bundle/b9` at `8a6b55d4c`. Import reads current lineups via the validated `livescoring` adapter, merges completed-week starters, previews rows, and records ordinary `LineupCorrect` events in the atomic import batch. Missing lineups remain absent; import request IDs do not use the ongoing-sync prefix. No schema change.
- Evidence: `make test` passed (full Go race suite); `make lint` passed; Vitest passed (154 files/1028 tests); `pnpm tsc --noEmit` passed. Integration test proves present lineup scores `Complete`, absent lineup `Incomplete`. Mutant proof: suppressing imported lineup commands failed that test at the recorded-lineup assertion; mutation restored.
- Next: commit/push `lane/lineup12` (push hook runs `make verify`), open one PR into `bundle/b9`, wait for green CI, then tidy.
- Blocked on: none.
- Tried/rejected: `lineup.sync.*` request IDs misrepresent a one-time import as ongoing sync and affect missed-lineup flags; use `mfl-import.lineup.*`. Batch originally rejected `LineupCorrect`; gather its normal lineup facts so all import events remain atomic.
