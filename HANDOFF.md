# Handoff

- Baton: Bee, 2026-11-17 (Control Room League tab complete; PR pending).
- State: `HostedControlRoomLeague.tsx` reads hosted notices and the full ledger; official notices and league notes use the existing preview → review → execute path. Newsletter, engine/feed controls, league creation/migration, and house-rule edits stay explicitly unavailable.
- Verified: frontend 128 files/927 tests; `pnpm exec tsc --noEmit`; `make lint`; `make bloat`; lane test `HostedControlRoomLeague.test.tsx` (6 tests).
- Slop fixed: none inherited in this lane; all new reads use the hosted session/query path and existing server parsers, and writes reuse `HostedLeagueRoom`/`HostedReview`.
- Next: commit, push, and open PR against `bundle/b5`; include CI and filelen notes in the report.
- Blocked: none. No Go files changed; no new server read/command or rule key added.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
- Tried/rejected: none.
