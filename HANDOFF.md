# Handoff

- Baton: Bee, 2026-11-18 (Bundle 5 lane b5fix; PR #358 open against `bundle/b5`).
- State: Hosted Fix wires the nine actTable forms, reversible ledger entries and correction/calendar exceptions. Reversal uses `commish.correction.revert`; no new server read/command. 32-team and preview → review → execute behavior is tested.
- Scope: thin hosted Fix tab only; local mode unchanged.
- Verified: frontend typecheck; frontend suite (127 files, 923 tests); `make lint` (0 golangci-lint issues); `make bloat`; targeted Fix tests; pre-push verify. PR checks on `e4f3556` passed (package/install, verify, all four test shards).
- Next: hand off PR #358 for review into `bundle/b5`.
- Blocked: `tidy-check.sh` reports launchers for other active lanes (b5league, b5people, b5picks, b5perm, b0load2 and b3home). They were not started by this lane and remain running; their scratch and KEEP files are untouched under the requested write boundary.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
- Slop fixed: revert choices filter on server-provided undo eligibility/replacement state; calendar exceptions use the existing calendar read.
