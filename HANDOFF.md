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

## Bee — pulse-b9feed
- Baton: ClaudeBox for review; Bee completed lane 2026-11-18, branch `session/pulse-b9feed`, target `bundle/b9`.
- State: Commit `732bf2bfa` pushed; PR #387 open and CI green. Implemented the member-safe read, strict filters and hosted workflow/catalog route; generated native client metadata. Allow-list: notices; approved trade transitions; completed roster drops, waiver awards and free-agent additions, with bid details omitted. Unknown commands, notes, private events and reasons are hidden.
- Evidence: race tests passed for commissioner, leagueservice and commandcatalog; hosted member/commissioner and agent-scope integration passed. Catalog generation checks, `make lint`, full `make verify`, pre-push verify and PR CI passed. One hostedPulse UI test flaked once, then passed alone and in the successful full rerun.
- Next: ClaudeBox review/merge PR #387.
- Blocked on: none for this lane. Final `tidy-check.sh` reported unrelated active alpha010 lanes `b7lab2`, `b9fa`, and `perf`; left them untouched due worktree-only scope; owners need to file KEEP records. No task-owned scratch or staged webapp remains.
- Tried and rejected: Do not reuse `commish.ledger.filtered.read`; it carries reasons and broader commissioner-visible entries. Reuse existing append-only records and an independent allow-list projection.
