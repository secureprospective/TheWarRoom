# Handoff

- Baton: Bee, 2026-11-18; lane `pulse-b7web`, branch `session/pulse-b7web`, base `b597b6122` (`bundle/b7`).
- State: Wails/web transport seam is in place. Browser login/session/logout uses `/app/session`; hosted workflows post the Go `RawNativeRequest` envelope to `/v1/native/workflow`. Same-origin credentials only; no JS bearer. PWA build, manifest/icons, static-shell service worker and offline screen are implemented.
- Contract checked against active b7door draft: login is `{protocol, username, password, code?}` (device is server-assigned); session returns `{protocol, admission}`. GET/POST/DELETE `/app/session` supported. Browser-cookie exception covers native workflow JSON; enrollment/recovery/TOTP setup/device revoke are not admitted. Hidden in browser; optional TOTP code remains on sign-in.
- Known gap: b7door draft leaves invalidation SSE outside the browser-cookie exception. Web transport intentionally does not open the bearer-only stream. Hosted reads refresh on app actions/reads, but external changes do not push in real time. No server changes in this lane.
- Verified: `pnpm run build:web`; `make verify` passed (full Go race suite, full frontend suite 139 files/986 tests, tsc, desktop build, bundle budget, CSP checks). Final web artifact checked at `frontend/dist-web` with `/app/` base and static-only hashed asset cache.
- Next: inspect final diff, run `tidy-check`, commit/push, open one PR into `bundle/b7`. PR body must note browser-hidden account/local features and the no-push invalidation limitation, and how to try it once b7door lands.
- Blocked on: b7door deployment/integration is uncommitted; external-change push needs follow-up if required. `tidy-check` found active b7door launcher PID 802854; left it running and did not edit its run folder.
- Tried/rejected: using enrollment/recovery/TOTP-admin/device-revoke APIs from browser (origin routes are outside b7door's cookie exception); EventSource against the bearer-only invalidation handler.
