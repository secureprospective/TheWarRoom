# Archive import guard handoff

## Baton
Christopher / reviewer — 2026-10-23. Bee completed `session/archive-guard`; not pushed.

## Where it stands
- Disk import: `internal/app/sources/archive.go` → one `leaguearchive.DiskFS` → discovery and walk → validate everything → Record. The filesystem owns actual-byte, read-count and entry budgets; failures precede recording.
- `CheckArchiveDir` also uses guarded discovery instead of Glob. Import does not run a separate discovery budget first.
- Guard is 108 production lines. Static symlinks/special files and symlinked roots refused before opening; handles checked after opening; depth bounded to three components. No IPC changes; 133 generated bound functions remain.
- `make lint`, `make verify`, `make test`: exit 0. Verify: 125 frontend files / 1,395 tests pass; bindings fresh, vulnerability check, bundle budget and CSP pass. Existing React act/CSS warnings remain.
- `TestGuardedRealArchive` explicitly ran with the real archive selected via WARROOM_TEST_ARCHIVE: temporary history DB, 1,720 recorded, repeat 1,720 held. Archive read-only.
- Symlink, FIFO, depth and exact/over-budget tests pass; refusal assertions require zero Record calls. Mutation: removing pre-open guard made symlink tests accept links and FIFO time out at three seconds. Restored before gates.

## Next move
Review the implementation commit and merge only after operator approval. No push authorized.

## Blocked on
No implementation blocker. Concurrent swap-race protection / platform-specific safe opening intentionally deferred to server mode. Non-Unix FIFO behavior not exercised (Unix-only test).

## Tried and rejected
Raw DirFS / Glob bypass the shared guard. Separate import preflight would add duplicate discovery outside the import budget; import now lets readLeagues perform discovery once within its filesystem instance.
