# Handoff

- Baton: ClaudeBox review of PR #355; Bee implementation complete, 2026-11-18. Base `bundle/b5`.
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
- Next: validate CI fix and all checks, commit regenerated nativegen doors, push (pre-push full verify), then wait for PR #367 CI green. No fallback needed; existing manager is callable per league.
- CI #367 failed only because `arguments.gen.ts` and `commandbar.gen.ts` were stale after hosted workflow registration. Regenerated with `go run ./frontend/scripts/nativegen -surface doors`; `go test ./frontend/scripts/nativegen -run TestEveryGeneratedDoorIsFreshAndRejectsDrift` passes.
- The D65-aware rule/test is in `internal/commissioner/deputy_permissions.go` and `deputy_permissions_test.go`: backup is commissioner-only and the all-ticks deputy test refuses it.
- Re-verified after regeneration: focused Go race tests for commandcatalog, commissioner, leagueservice, offsite, serverhttp; frontend 132 files/939 tests and TypeScript; `make lint`; `make bloat`. All pass. Next commit generated files + this handoff, push to refresh PR #367 CI.
- Blocked: no code blockers. Previous `tidy-check` found another lane's live b4sync2 launcher (pid 429861; run output at `~/fleet/runs/pulse-b4sync2`); left it untouched. It owns no scratch folder under `~/scratch`.
