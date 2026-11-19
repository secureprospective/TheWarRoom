# Handoff

- Baton: Bee, `session/pulse-notyet-b9`, 2026-11-19.
- State: Wired existing `commish.people.read` `actors[].effectivePermissions` into hosted People and Control Room. The local team inspector now also uses the already-loaded `TargetSignInFacts` for authenticator status. Custom permission-set editing and remaining account/server placeholders have no supporting read routes; no new server reads or league mutations.
- Verification: `go test -race ./internal/serverhttp/...` passed; targeted Vitest passed (3 files, 22 tests); `pnpm exec tsc --noEmit`, `pnpm run build:web`, `make lint`, `gofmt -d`, and `git diff --check` passed. First push-hook full Vitest run caught two >110-character lines; both are fixed. Web build retains existing dynamic-import/chunk-size warnings.
- Next: Retry push hook, open one PR into `bundle/b9`, check CI, then run tidy check. PR body must include the placeholder disposition table.
- Blocked on: none.
- Tried and rejected: Assigned worktree pointed at `origin/main`; preserved it and based this lane on `origin/bundle/b9`. Initial heavy-work guard waited for another full race suite to finish; guard passed before tests.
