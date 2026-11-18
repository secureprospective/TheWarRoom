# Handoff

- Baton: Bee, 2026-11-18. Lane `pulse-fixsearch`, branch `session/pulse-fixsearch`.
- State: regression test added and observed failing before the fix; player search now matches case-insensitive name tokens in either order, including `Last, First` against `First Last`. Targeted test, TypeScript, full Vitest (138 files/978 tests), and `make verify` pass. `make verify` needed retries for concurrent `golangci-lint`/full-race jobs; final run passed including Go race tests, frontend build, budget, CSP and browser CSP checks. `tidy-check.sh` is clean.
- Next: commit, push, open one PR into `main`, then report CI/PR status.
- Blocked on: none.

- Baton: Bee, 2026-11-18. Lane `pulse-b5sync`, branch `session/pulse-b5sync`; merging `origin/main` (347388a19) into Bundle 5.
- State: merge commit `b418b1b3` has parents `4a764caa2` (Bundle 5) and `347388a19` (main); preserved on pushed branch `session/pulse-b5sync`. PR #370 is open into `bundle/b5`; all local gates and GitHub checks pass. Bundle 4 rail/Home routing stays; its Control Room rail route opens Bundle 5's hosted Control Room, with Rules in the Rules tab. Workflow output union retains BackupResult, WhatIfStack, and ContractDecisions.
- Verified: `pnpm exec tsc --noEmit`; full Vitest (138 files/977 tests); `pnpm run build`; targeted commissioner Control Room/rules tests (30); `make verify` passes, including lint, Wails freshness, full Go race tests, bundle budget and CSP. Contract-decision has member-action/owner-seat classification; bid.withdraw remains unpublished; backup stale-write behavior is unchanged.
- Next: ClaudeBox may fast-forward `bundle/b5` after review. Do not merge PR #370 in this lane.
- Blocked on: none.

- Prior Baton: ClaudeBox review of PR #355; Bee implementation complete, 2026-11-18. Base `bundle/b5`.
- State: D65, security fixes and CI follow-up are pushed on `session/pulse-b5perm`; PR #355 open against `bundle/b5`, current CI green.
- D65: six permission groups; whole-mutation-catalog classification; preview/execute permission command; replay/revocation; co-commissioners expose all six; vote close/tiebreak stay commissioner-only.
- Security: commissioner/co-commissioner shortcuts removed; permitted deputy trade rulings use reviewer eligibility and bar the deputy's own franchise; lineup.set retains seat/ability checks.
- CI follow-up: added `commish.deputy.permissions.set` release fixture, aligned succession close/tiebreak expectations, and preserved strict bid/waiver/O-Line/draft clock authorizers for automatic system writes.
- Verified: `go test -race ./internal/commissioner/... ./internal/governance/... ./internal/store/...`; `go test -race ./internal/app/commissioner/... ./internal/leagueservice/... ./internal/store/history/...`; frontend 127 files/921 tests + TypeScript; `make lint`; `make bloat`. Code head `5c2fd670d` had PR CI all green; this handoff-only commit will trigger CI again, with no source changes since.
- PR body contains the command mapping, security fixes and Slop fixed list. No new table, migration, UI or release change.
- Filelen: `internal/store/history/commissioner.go` is 774 lines (base 727); transaction-time authorization stays beside the serialized append check so writes share one gate.
- Final `tidy-check` found unrelated active runs b5backup, b4sync, b3home and b5league; left them untouched. This lane created no scratch files.
- Open decisions untouched: snooze scope, deputy job visibility, written-request visibility, division names.

## b5backup lane

- Baton: Bee, 2026-11-18; backup feature is based on current `bundle/b5` (#355).
- State: hosted `backup.now` uses the existing per-league R2 manager, admits only commissioner/co-commissioner, refuses agents and overlapping backups, and bypasses only the stale-backup write guard. Server UI previews/confirms; rollover backs up first.
- Verification: `make lint`/`make bloat` pass; focused Go race tests for deputy classification, backup authorization, catalog coverage and hosted execution pass. Full frontend suite (132 files/939 tests), TypeScript, nativegen, and Wails binding freshness pass. First pre-push run passed Go and exposed five long frontend lines; fixed and reran the full frontend suite.
- Done: PR #367 is open against `bundle/b5`; all GitHub checks are green after regenerating stale `arguments.gen.ts` and `commandbar.gen.ts`. The existing per-league R2 manager is callable; no fallback needed.
- D65 rule/test in `internal/commissioner/deputy_permissions.go` and `deputy_permissions_test.go`: backup is commissioner-only and an all-ticks deputy is refused.
- Verification: nativegen freshness; focused Go race tests for commandcatalog, commissioner, leagueservice, offsite, serverhttp; frontend 132 files/939 tests and TypeScript; `make lint`; `make bloat`; pre-push verify; CI green. `tidy-check` clean.
- No blockers. Earlier tidy-check found another lane's active b4sync2 launcher (pid 429861; run output at `~/fleet/runs/pulse-b4sync2`); left it untouched.
