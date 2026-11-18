# Handoff

- Baton: Bee, 2026-11-18.
- State: implementation committed as `6d0ddc29a` and pushed to `session/pulse-b1pulse`, based on `bundle/b1`; PR not opened.
- Changes: Pulse Race uses `schedule.franchise.read` for My Road. Shared results-derived `recentForm` feeds HQ, Home Matchup/form, and Pulse. Home's per-player research form read was removed.
- Verification: frontend 125 files / 918 tests; `pnpm exec tsc --noEmit`, `make lint`, `make bloat`, `tidy-check.sh`, and pre-push `make verify` pass. No Go files changed.
- Blocked: HomeBrief is mounted only by `AppLeagueMount`/local `TargetApp`, while the task asks to change Home and Bundle rules freeze local mode. `readNativeHQ` is also shared by both modes. Need Christopher to confirm whether these requested changes are an exception before opening the PR.
- Next: resolve the mode-scope conflict; if authorized, open PR against `bundle/b1` and wait for CI.
