# Handoff

- Baton: Bee, `lane/flaky2`, 2026-11-19.
- State: Based on current `origin/bundle/b9` (`4c15e137a`, after #400 and the latest main merge); latest fast-forward only added a workflow file. NativeAccount flake at `nativeAccount.test.tsx:124-130` was an assertion before lazy HostedLeague mount; now waits for visible content and room effect inside `act` (20/20 focused runs). Serverclient race is already fixed in this base by #379: its fixture serves the invalidation stream and snapshots under `d.mu` (50/50 race runs). Hosted roster-add passed 50/50; cause not reproduced.
- Verification: `make test` passes on same application tree; full frontend `pnpm tsc --noEmit` and Vitest pass (153 files/1026 tests). No production changes.
- Next: Commit/push, open one PR into `bundle/b9`, and verify pre-push and CI. PR body will note #3 is already in base and #1's cause was not reproduced.
- Blocked on: none.
- Tried and rejected: no sleeps, retries, or production behavior changes.
