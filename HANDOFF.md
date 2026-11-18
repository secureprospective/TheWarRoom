# Handoff

- Baton: Bee, 2026-11-17.
- State: Hosted Control Room frame, Jobs and Situations implemented in `frontend/src/app/commissioner/`. Jobs and League Health share the `commish.jobs.read` count; commissioner forms stay on `hostedRoom`.
- Scope: thin hosted mode only; local mode unchanged. Snooze/settlement remain out of scope. Open decisions remain unresolved.
- Verified on this tree: frontend 127 files/919 tests; `pnpm exec tsc --noEmit`; path-casing check; `make lint`; `make bloat`. No Go files changed.
- Slop fixed: renamed the hosted helper module to avoid a case-insensitive path collision with `HostedControlRoom.tsx`.
- Next: commit, push `session/pulse-b5frame`, open PR against `bundle/b5`, confirm CI, then leave review baton to ClaudeBox.
- Blocked: none. Deputy visibility is passed through from the server read; permission checks stay on the hosted command path.
- Open decisions deliberately left untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
- Tried/rejected: none.
