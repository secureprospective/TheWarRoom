# Handoff

- Baton: Bee, 2026-11-17.
- State: Rebased on current `origin/main` (#330/#331). Hosted `player.note.read`, `player.note.set`, and `player.watch.set` use #325's league-private store; service derives the seated franchise from authenticated identity. HQ and War Room inspector use hosted actions; local desktop path is unchanged; private reads bypass shared reply cache. Agent-token reads/writes are restricted to the issuer's own franchise.
- Verified: `go test -race ./internal/leagueservice/... ./cmd/twr-server/...` passes; focused commandcatalog/serverhttp race tests pass (privacy, no-cache, and issuer-seat cases). Frontend typecheck + 4 Vitest files pass (24 tests); nativegen checks and `make lint` pass. Pre-push runs exposed a local-host nil-context regression and then an app line-length gate; both fixed, focused checks pass, full verify pending retry.
- Next: Retry push with the pre-push `make verify`, open PR, wait for green CI, close #313/#328 with its link.
- Blocked on: No feature blocker. `tidy-check` found headless Brave sessions using profiles in `~/scratch/twr-live-anim-1116` while working from `~/scratch/twr-mobile-1117`; both `KEEP.md` files identify Tom's separate in-flight mobile/live task, so Bee left them untouched.
- Tried/rejected: No private-read cache, commissioner view, schema migration, request-selected franchise, or change to local desktop notes/watch.
