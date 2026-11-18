# Handoff

- Baton: ClaudeBox, 2026-11-18 (PR #357; based on bundle/b5 including PR #350).
- State: Hosted People reads `commish.people.read` and renders all seats with search/vacancy filter; seat tools, roles, deputy order and succession use the shared hosted act form. Quiet/Staff and D65 permission sets stay explicitly unavailable.
- Findings: b5perm is paused on a D65 authority conflict; no permission-set read/commands are merged. `commish.people.read` lacks last-seen and staff fields (D69 partial). No server read, command, table or migration added.
- Changed: People tab owner/query with identity reset; one frame registration; role grant/revoke actTable rows; 32-seat, preview→review→execute, and server-denial tests.
- Verified: `HostedControlRoomPeople.test.tsx` and full frontend suite (925/925); `pnpm exec tsc --noEmit`; `make lint`; `make bloat`; production build and bundle budget pass. PR #357 CI is green (checks, verify, test shards, package, install, release-tools). No Go files changed; Go race test not applicable. Desktop-mode checks pass.
- Tidy: `tidy-check.sh` flagged detached b5picks, b5perm, b0load2 and b3home launchers (PPID 1, other lanes); left them running and did not touch their files or scratch.
- Next: ClaudeBox reviews and merges PR #357; D65 authority remains the only functional boundary, outside this PR.
- Blocked: D65 authority decision remains with b5perm/Christopher; no code depends on it.
- Open decisions untouched: snooze scope; deputy job visibility; written-request visibility; division naming.
