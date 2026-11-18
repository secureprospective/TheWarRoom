# Handoff

- Baton: Bee, 2026-11-17; PR #351 open against `bundle/b5`.
- State: Server tab reads `commish.ledger.read`, renders every visible row, and identifies restore/feed/upgrade as operator-only. `backup.now` catalog entry is desktop-only, so hosted backup is missing; no new read/command invented.
- Verified: `HostedControlRoom.test.tsx` and `pnpm exec tsc --noEmit` pass; `make bloat` passes. Full frontend: 918/920 passed; isolated rerun of the two failures passed (40 tests). `make lint` blocked by another `golangci-lint` run.
- Next: retry `make lint` when the other process exits; watch PR checks.
- Slop fixed: none.
- Blocked: hosted `backup.now` command path missing.
- Open decisions untouched: snooze scope; deputy job visibility; written-request visibility; division naming.
