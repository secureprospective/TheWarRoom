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
- Baton: ClaudeBox review; implementation ready on `session/pulse-b9boards`, base `bundle/b9`, PR #388.
- State: server preference store/table implemented; schema migration is sign-in file v4. Layout writes are bounded/validated and revision-checked. The account-only workflow uses `/v1/account/preferences` so the existing sign-in backup admission guards writes without pausing league writes. Thin Wails forwards the account command there; browser cookie calls require exact same-origin. Both agent token scope and access refuse it. Sign-in conversion copies/drops the new table when present and safely skips it for older one-file sources.
- Flow: typed Raw read/set → catalog validation → session-authenticated account ID → account-store CAS. Conflicts return the current server copy; no league/franchise argument and no private reply caching. Hosted HomeBrief reads server-first, caches only as an offline fallback, and keeps local mode on localStorage.
- Evidence: focused race tests pass in all changed Go packages, including existing/absent layout sign-in conversion; `make lint`, nativegen freshness, frontend TypeScript and layout/transport tests pass. Frontend fresh-store test proves a second device reads the server copy. CI passed on `007cb5dd1`: all four race shards, lint/checks, release-tools, package and install. The first shard attempt stopped without a reported test failure; the retry passed.
- Next: ClaudeBox review/merge PR #388.
- Blocked on: none. `tidy-check.sh` reports unrelated active lanes/hosted test server; left untouched for their owners.
- Tried and rejected: reusing a per-franchise league store (wrong owner boundary and not cross-league); no settings/preferences table exists in the hosted account store.

## Bee — pulse-b9cr1
- Baton: Bee implementing B9-5 Control Room phone layout, 2026-11-18, branch `session/pulse-b9cr1`, target `bundle/b9`.
- State: Commit `3e8edb5aa` rebased onto current `bundle/b9`. Added <700px hosted Control Room layout rules and focused phone CSS test; reads, commissioner checks and authorization unchanged. Focused tests 14/14; full Vitest 995/995; tsc, frontend build, bundle budget and CSP passed. `make verify` exceeded command time limit in Go tests. Push pre-check failed on one unrelated `nativeAccount.test.tsx` flake; that test passed alone on retry.
- Next: Retry pre-push/push, open PR into `bundle/b9`, verify CI.
- Blocked on: Heavy guard refuses new suites (disk 91%, other full suite activity); do not clear shared cache or touch other lane files. Claude-OS/Brave phone browser not installed; 390px CSS contract test passed, interactive emulation unverified.
- Tried and rejected: no JavaScript viewport mode; retain existing hosted screen and use the approved CSS breakpoint.

- Baton: Bee, 2026-11-18; B9-2 lane on `session/b9-2-league-activity`, based on `origin/bundle/b9`.
- State: phone League view now has Table, Schedule, Activity and Live tabs. Activity uses only the validated member-safe public feed; player status remains visibly unavailable. Blank/invalid NFL kickoffs read “dates not published yet.”
- Evidence: 147 Vitest files/1005 tests and frontend tsc pass; member-safe activity unit and hosted-read tests pass under `-race`. The phone CSS regression covers 390px and 44px targets.
- Next: commit, push through pre-push, open PR to `bundle/b9`, wait for CI.
- Blocked on: pre-push and PR CI.
- Tried and rejected: none.

- Baton: Bee, 2026-11-18; B9-2 implementation complete, PR #397 open into `bundle/b9`.
- State: commit `b3477a5e3` is pushed; pre-push verify and every PR check passed. No merge requested.
- Evidence: full Vitest 147/1005, tsc, member-safe activity race tests, 390px CSS test, PR checks green.
- Next: leave PR for review.
- Blocked on: review/merge decision.
- Tried and rejected: none.
