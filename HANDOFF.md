# Handoff

- Baton: Bee, lane `pulse-b7fix`, 2026-11-18, branch `session/pulse-b7fix`, base `bundle/b7`.
- State: commit `c35dfdb45` pushed; PR #378 targets `bundle/b7`. `ownBrowserOrigin` permits Origin-absent GET/HEAD unless Sec-Fetch-Site is cross-site; other requests still require exact Origin. Focused race tests and `make verify` passed. All PR CI checks, including all four race shards, passed.
- Tidy: `tidy-check.sh` found unrelated active runs `b7upd` and `fixrace` running from the main checkout; left their processes and scratch untouched. Owners must file KEEP records.
- Next: ClaudeBox review/merge PR #378. `POST /app/session` returns 409 with the desktop-app message for accounts admitted to multiple leagues. `make verify` passed, including race tests, frontend build, lint and staged embedded-bundle test; staging cleaned.
- Flow: exact configured HTTPS Origin and Host → `/app/session` creates/revokes the existing owner session in `__Host-twr_session`; GET/POST return only the one admitted league. Native command routes accept that cookie only from the configured origin and retain current authorization/TOTP checks. `/app/` embeds the built web app with locked headers.
- Security: browser never receives owner bearer or recent-auth proof; reject Authorization+cookie and agent tokens with Origin or as cookie; changes feed, enrollment and account-administration routes stay outside the cookie exception. No league list or `commish.leagues.read`.
- Deployment: `-public-host` is sourced from `/etc/twr/public-host`; Cloudflare routes `/app`, and upgrade recovers the hostname from existing tunnel config. Static staging is cleaned after verify/release build.
- Next: ClaudeBox review/merge PR #372. Alpine deployment integration was not run; changed shell scripts passed syntax checks.
- Tidy: `tidy-check.sh` found unrelated active alpha010 lanes `fixcr`, `fixhome`, and `fixsearch` launched from the main checkout. Left their processes/files untouched; their owners need to file KEEP records. Flag this to ClaudeBox.
- Blocked on: none for this lane. The unrelated active lanes still need their owners to file KEEP records.
- Tried and rejected: persistent last-used league selection (requires a migration and contradicts Part 3); no picker or league-list endpoint.
