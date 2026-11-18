# Handoff

- Baton: Bee, 2026-11-18; lane `pulse-b7web`, branch `session/pulse-b7web`, base `b597b6122` (`bundle/b7`).
- State: PR #373 is open into `bundle/b7`; head `7b4406340`; all GitHub checks pass. Browser transport uses `/app/session` and `/v1/native/workflow` with same-origin credentials, no Authorization or JS bearer. PWA shell/build/offline handling are implemented.
- Contract checked against b7door draft: login `{protocol, username, password, code?}`; response `{protocol, admission}`; device assigned server-side. Browser-cookie exception covers native workflow JSON. Enrollment/recovery/TOTP setup/device revoke remain hidden; sign-in accepts optional TOTP code.
- Known gap: b7door keeps invalidation SSE bearer-header-only. Browser does not open it, so external changes do not push live; reads refresh on app actions/reads. No server changes in this lane.
- Verified: `pnpm run build:web`; `make verify` passed (full Go race suite, 139 frontend files/986 tests, TypeScript, desktop build, bundle budget and CSP checks). PR CI is green.
- Next: ClaudeBox review/merge after b7door deployment integration; decide separately whether browser invalidation push is required.
- Blocked on: b7door integration is still in flight. `tidy-check` found active b7door launcher PID 802854; left it running and did not edit its run folder.
- Tried/rejected: account-enrollment/recovery/TOTP-admin/device-revoke routes (outside b7door's cookie-Origin exception); EventSource to the bearer-only invalidation handler.
