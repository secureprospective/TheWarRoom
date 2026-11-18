# Handoff

- Baton: ClaudeBox for review; Bee completed lane `pulse-b7door`, 2026-11-18, branch `session/pulse-b7door`, base `bundle/b7` (`b597b6122`).
- State: commit `84f79ddab` pushed to ready PR #372; CI green. Browser door implemented without schema migration or league-recency state. `POST /app/session` returns 409 with the desktop-app message for accounts admitted to multiple leagues. `make verify` passed, including race tests, frontend build, lint and staged embedded-bundle test; staging cleaned.
- Flow: exact configured HTTPS Origin and Host → `/app/session` creates/revokes the existing owner session in `__Host-twr_session`; GET/POST return only the one admitted league. Native command routes accept that cookie only from the configured origin and retain current authorization/TOTP checks. `/app/` embeds the built web app with locked headers.
- Security: browser never receives owner bearer or recent-auth proof; reject Authorization+cookie and agent tokens with Origin or as cookie; changes feed, enrollment and account-administration routes stay outside the cookie exception. No league list or `commish.leagues.read`.
- Deployment: `-public-host` is sourced from `/etc/twr/public-host`; Cloudflare routes `/app`, and upgrade recovers the hostname from existing tunnel config. Static staging is cleaned after verify/release build.
- Next: ClaudeBox review/merge PR #372. Alpine deployment integration was not run; changed shell scripts passed syntax checks.
- Tidy: `tidy-check.sh` found unrelated active alpha010 lanes `fixcr`, `fixhome`, and `fixsearch` launched from the main checkout. Left their processes/files untouched; their owners need to file KEEP records. Flag this to ClaudeBox.
- Blocked on: none for this lane. The unrelated active lanes still need their owners to file KEEP records.
- Tried and rejected: persistent last-used league selection (requires a migration and contradicts Part 3); no picker or league-list endpoint.

## pulse-b9boards lane (Bee, 2026-11-18)
- Baton: Bee; implementation ready on `session/pulse-b9boards`, base `bundle/b9`.
- State: server preference store/table implemented; schema migration is sign-in file v4. Layout writes are bounded/validated and revision-checked. The account-only workflow uses `/v1/account/preferences` so the existing sign-in backup admission guards writes without pausing league writes. Thin Wails forwards the account command there; browser cookie calls require exact same-origin. Both agent token scope and access refuse it. Sign-in conversion copies/drops the new table when present and safely skips it for older one-file sources.
- Flow: typed Raw read/set → catalog validation → session-authenticated account ID → account-store CAS. Conflicts return the current server copy; no league/franchise argument and no private reply caching. Hosted HomeBrief reads server-first, caches only as an offline fallback, and keeps local mode on localStorage.
- Evidence: focused race tests pass in all changed Go packages, including existing/absent layout sign-in conversion; `make lint`, generated-catalog freshness, frontend TypeScript and layout/transport tests pass. Frontend fresh-store test proves a second device reads the server copy. First PR CI exposed a stale generated workflow-argument door; regenerated `frontend/src/app/commands/arguments.gen.ts`, then `go test -race ./frontend/scripts/nativegen`, doors `-check`, and frontend TypeScript passed.
- Next: commit and push the generated-door refresh; pre-push `make verify`, then wait for PR CI and report PR/migration.
- Blocked on: none. `tidy-check.sh` still reports unrelated active lanes/hosted test server; left untouched.
- Tried and rejected: reusing a per-franchise league store (wrong owner boundary and not cross-league); no settings/preferences table exists in the hosted account store.
