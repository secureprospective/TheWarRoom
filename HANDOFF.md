# Handoff

- Baton: Bee, 2026-11-18 (Bundle 5 lane b5fix; `bundle/b5` now includes PRs #347 and #350).
- State: The hosted Control Room frame and Rules board are on `bundle/b5`. This change wires Fix to the nine actTable forms, hosted reversible ledger entries, active corrections and calendar exceptions. Reversal is `commish.correction.revert`; no new read/command.
- Scope: thin hosted Fix tab only; local mode unchanged.
- Verified before rebasing on #350: frontend typecheck; full frontend tests (127 files, 921 tests); `make lint` (0 golangci-lint issues); standalone `make bloat`; targeted Fix tab test. Rerun after rebase.
- Next: finish post-rebase checks, then push and open the lane PR against `bundle/b5`.
- Blocked: none. Rebase test-file changes applied without conflict; HANDOFF overlap resolved here.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
- Slop fixed: revert choices filter on the server-provided undo eligibility and replacement state; calendar exceptions use the existing calendar read.
