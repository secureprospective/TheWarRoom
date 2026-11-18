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

## Bee — B8-2 Today / Needs you (2026-11-18)
- Baton: Bee, branch `session/pulse-b8today`, base `bundle/b8` with PR #380.
- State: added <=699px Today layout styling in `frontend/src/app/shell/thinHome.css`; increased phone action targets and stacked content. Added 390px-viewport regression to `ThinHome.test.tsx`; existing hosted reads and unavailable states retained.
- Checks: focused Vitest 22/22; full Vitest 990/990; `pnpm tsc --noEmit`; `git diff --check`; `tidy-check.sh` clean. One initial full-suite run had a transient unrelated `hostedPulse.test.tsx` failure and a test line-length finding; split the import and reran, all tests passed.
- Next: commit/push (pre-push verification), then PR into `bundle/b8`.
- Blocked on: none. Browser 390px/1280px visual check remains for PR/Claude-OS; current automated 390px regression validates rendered content but does not emulate layout.

## Bee — B8-1 phone shell (2026-11-18)
- Baton: ClaudeBox review; PR #380 from `session/pulse-b8shell` into `bundle/b8`, commit `8f2fab5ee`.
- State: 390px phone shell has Today/Home, Live/Pulse, Team/HQ, Moves/War Room and League/Pulse standings; commissioners get Control Room from League. At 1280px the existing rail remains and phone tabs are absent. App bar is fixed to the admitted league name.
- Checks: focused frontend tests 58/58, full Vitest 989/989, `pnpm tsc --noEmit`, pre-push verification passed; PR CI green on `bb5df4b68` (all checks passed). No server, auth, session or service-worker changes.
- Tidy: `tidy-check.sh` found unrelated active `b9feed` and `b9boards` lanes in the main checkout. Left their processes/files untouched; their owners need to file KEEP records.
- Next: ClaudeBox review/merge. Blocked on review.
