# Handoff

- Baton: Bee, 2026-11-17 (Season tab lane; PR pending).
- State: Hosted Control Room Season tab now reads the league calendar, published/draft schedule counts, and playoff state. Schedule/game and playoff/title writes use shared commissioner act forms. Divisions remain an honest not-on-server state. Rollover is unavailable because `backup.now` is not hosted; do not bypass the required backup-first order.
- Verified: frontend 128 files/920 tests; `pnpm exec tsc --noEmit`; `make lint`; `make bloat`; `git diff --check`.
- Slop fixed: none found in scope. Existing shared workflow retained; no second write path or division feed added.
- Next: commit, push a lane branch and open PR against `bundle/b5`; head brain reviews and merges.
- Blocked: hosted backup command is missing, so rollover cannot be enabled without violating backup-first. Other Bundle 5 lanes are still running.
- Open decisions untouched: division naming, snooze ownership, deputy job visibility, written-request visibility.
