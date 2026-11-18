# Handoff

- Baton: Bee, 2026-11-18.
- State: Pulse implementation and head-brain review fixes are committed on `session/pulse-b1pulse`, PR #345 against `bundle/b1`.
- Changes: My Road uses `schedule.franchise.read`; team W/L form shares `recentForm` across Pulse and Home. HQ now charts each player's `results.read` starter points with a separate player-bars renderer. Removed 22-week magic cap; final games without a winner count as ties; removed stray PulseRace blank line and dead research-form calculation.
- Verification: frontend 125 files / 919 tests; `pnpm exec tsc --noEmit`, `make lint`, `make bloat`, `tidy-check.sh` pass. No Go files changed.
- Next: CI on PR #345 was still running when this handoff was written; monitor checks and request head-brain review/merge once green.
- Slop fixed: removed duplicate Home research-form read; replaced shared-franchise marks on every HQ player row with per-player result points; removed stale research-based form calculation.
