# Handoff

- Baton: Bee, 2026-11-17.
- State: Server tab in `HostedControlRoom.tsx` now reads `commish.ledger.read`, renders every visible row, and clearly identifies restore/feed/upgrade as operator-only. Backup-now is left unavailable: its catalog entry is desktop-only, not a hosted server command.
- Tests added: hosted ledger at 32 rows and operator-only seam.
- Next: run hosted Control Room frontend tests and TypeScript, then required lint/bloat; commit and open PR against `bundle/b5`.
- Blocked: `heavy-ok.sh` currently refuses because another full `-race` suite is running. Retry before tests/heavy checks.
- Slop fixed: none identified outside this tab.
- Open decisions deliberately untouched: snooze scope; deputy job visibility; written-request visibility; division naming.
