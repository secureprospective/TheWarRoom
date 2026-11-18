# Handoff

- Baton: Bee, 2026-11-18 (lane b5people; based on bundle/b5 including PR #350).
- State: Hosted People reads `commish.people.read` and renders all seats with search/vacancy filter; seat tools, roles, deputy order and succession use the shared hosted act form. Quiet/Staff and D65 permission sets stay explicitly unavailable.
- Findings: b5perm is paused on a D65 authority conflict; no permission-set read/commands are merged. `commish.people.read` lacks last-seen and staff fields (D69 partial). No server read, command, table or migration added.
- Changed: People tab owner/query with identity reset; one frame registration; role grant/revoke actTable rows; 32-seat, preview→review→execute, and server-denial tests.
- Verified on current base: `HostedControlRoomPeople.test.tsx` and full frontend suite (925/925); `pnpm exec tsc --noEmit`; `make lint`; `make bloat`; production build and bundle budget pass. No Go files changed; Go race test not applicable. Desktop-mode checks pass.
- Tidy: `tidy-check.sh` flagged detached b5picks, b5server and b2docket launchers (PPID 1, other lanes); left them running.
- Next: publish the branch as a PR against `bundle/b5`; ClaudeBox reviews it and verifies CI before merge.
- Blocked: D65 authority decision remains with b5perm/Christopher; no code depends on it.
- Open decisions untouched: snooze scope; deputy job visibility; written-request visibility; division naming.
