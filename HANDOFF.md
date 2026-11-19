# Handoff

- Baton: Bee completed; PR #410 awaits Christopher review, 2026-11-19.
- State: Wired existing `commish.people.read` `actors[].effectivePermissions` into hosted People and Control Room. The local team inspector uses the already-loaded `TargetSignInFacts` for authenticator status. Custom permission-set editing and remaining account/server placeholders have no supporting read routes; no new server reads or league mutations.
- Verification: `go test -race ./internal/serverhttp/...`; targeted Vitest (3 files, 22 tests); `pnpm exec tsc --noEmit`; `pnpm run build:web`; `make lint`; pre-push verify; PR CI all passed. `tidy-check.sh` clean. Web build retains existing dynamic-import/chunk-size warnings.
- Next: Christopher review PR #410; do not merge.
- Blocked on: none.
- Tried and rejected: Assigned worktree pointed at `origin/main`; preserved it and based this lane on `origin/bundle/b9`. Initial heavy-work guard waited for another full race suite to finish; guard passed before tests.
