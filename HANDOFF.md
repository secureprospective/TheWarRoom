# Handoff

- Baton: ClaudeBox, 2026-11-17 (PR #347 open; review pending).
- State: Hosted Control Room frame, Jobs and Situations are implemented in `frontend/src/app/commissioner/`. Jobs and League Health share the `commish.jobs.read` count; commissioner forms stay on `hostedRoom`.
- Scope: thin hosted mode only; local mode unchanged. Snooze/settlement remain out of scope.
- Verified: frontend 127 files/919 tests; `pnpm exec tsc --noEmit`; path-casing check; `make lint`; `make bloat`; PR CI green.
- Slop fixed: shared jobs read/count; reused act-table form and hosted command path; renamed helper module to avoid case-insensitive collision with `HostedControlRoom.tsx`.
- Next: ClaudeBox review and merge PR #347 into `bundle/b5`.
- Blocked: none. Deputy visibility comes from the server read; permission checks stay on the hosted command path.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
- Tried/rejected: none.
