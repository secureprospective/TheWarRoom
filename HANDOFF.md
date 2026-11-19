# Handoff

- Baton: Bee completed lane `rel10`, 2026-11-19.
- State: PR #403 (`release: bundle/b9 carries release number 10`) is open into `bundle/b9`. Release 10/schema 16 appended; roadmap row identifies 0.10.3. No tag or GitHub release.
- Verification: `make test`, Vitest (153 files/1026 tests), `tsc --noEmit`, `make verify`, and push hook passed. PR CI passed all test shards, checks, release-tools, package, install, and verify.
- Mutant: Removing release row 10 caused `TestCurrentContract` to fail with current release 9 vs expected 10; restored row and targeted race test passed.
- Decision: Schema remains 16 because this is a release-number-only bump. No explicit release allowlist exists; tooling reads the append-only contract.
- Next: Christopher review PR #403; do not merge.
- Blocked on: None.
- Tried/rejected: None.

## Bee — pulse-freshlab (2026-11-22)
- Baton: Bee, preparing an offline, reviewable clean-alpha runbook for CT 260; CT 260 must remain untouched.
- State: Pulled `origin/main`; read governing docs, desktop modes and pulse-ready evidence. All 32 seats are occupied by stress-test accounts; preserve commissioner identity.
- Evidence: `backup.now` gives a checked R2 result (`internal/leagueservice/backup.go:41-66`; `cmd/twr-server/backup_hosted.go:94-116`). `commish.seat.remove` only releases a seat (`internal/commissioner/people.go:147-148,184-185,231-232`). Accounts are immutable and DB deletion is refused (`internal/store/accounts/store.go:61-66`); reset-sign-ins is global but preserves account/password/authenticator (`deploy/server/reset-sign-ins.sh:1-14`; `deploy/server/RESTORE.md:178-194`).
- Finding: No supported command removes stress-account rows. Option B is first-load for a new CT and bootstraps a new commissioner; prior import had 3,367 entries vs current 3,399 (`deploy/server/README.md:57-82`; `pulse-lab4/RESULT.md`; `pulse-ready/RESULT.md`). Neither path meets every constraint.
- State: Added dry-run-only `deploy/server/alpha-clean-plan.sh`; its `cmd/twr-server/alpha_clean_plan_test.go` uses `newOneFileServer`, checks the throwaway DB digest stays unchanged, and rejects `--apply`.
- Verified on candidate: `sh -n`; final `make lint` (golangci-lint 0 issues); focused race test `TestAlphaCleanPlanDryRunLeavesThrowawayDatabaseUnchanged` passed. `shellcheck` is unavailable. CT 260 untouched.
- Next: resolve this handoff conflict by retaining bundle/b9's release-10 section, finish rebase, push/open PR to `bundle/b9`, wait for CI, and run tidy check.
- Blocked on: approval if deleting test-account rows is mandatory; no supported command exists. Option A frees seats/revokes sessions but leaves accounts able to sign in; the plan explicitly stops there.
- Tried/rejected: Option B new-server import; it would not preserve exact history/account identity and is not an in-place reset tool. No CT commands or live account/session mutations; task says plan only.
