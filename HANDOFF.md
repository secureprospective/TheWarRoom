# Handoff

- Baton: Bee, 2026-11-18. Bundle base includes Control Room PRs #347 and #350.
- State: D65 and Part 2 security fixes are committed on `session/pulse-b5perm`; PR #355 targets `bundle/b5`. CI follow-up is implemented locally, not yet committed.
- D65: six permission groups; whole-mutation-catalog classification; preview/execute permission command; replay/revocation; co-commissioners expose all six; vote close/tiebreak stay commissioner-only.
- Security fix: commissioner/co-commissioner shortcuts removed; permitted deputy trade rulings use reviewer eligibility and bar the deputy's own franchise; lineup.set retains seat/ability checks. Restored pre-shortcut role behavior.
- CI follow-up: added release-matrix coverage for `commish.deputy.permissions.set`; updated succession close/tiebreak expectations; retained strict bid/waiver/O-Line/draft clock authorizers for automatic system writes.
- Verified: `go test -race ./internal/commissioner/... ./internal/governance/... ./internal/store/...`; `go test -race ./internal/app/commissioner/... ./internal/leagueservice/... ./internal/store/history/...`; frontend 127 files/921 tests + TypeScript; `make lint`; `make bloat`.
- PR #355 body updated with security and CI follow-up, command mapping and Slop fixed list. Previous CI failed on the release fixture and system clock authorization; fixed locally, new CI awaits push.
- Filelen: `internal/store/history/commissioner.go` is 774 lines (base 727); transaction-time authorization stays beside the serialized append check so writes share one gate.
- Open decisions untouched: snooze scope, deputy job visibility, written-request visibility, division names.
- Next: commit CI follow-up, push PR #355 (pre-push verify runs the full suite), run tidy-check, confirm CI green.
