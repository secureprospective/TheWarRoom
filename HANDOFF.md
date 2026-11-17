# Handoff

- Baton: Bee, 2026-11-17.
- State: Extended existing `native.reserve.check` with optional taxi experience/years remaining from the same validated rule calculation. Hosted HQ shows `NFL YR` / `TAXI LEFT` only when held values exist; desktop columns are unchanged. IR stays honestly unavailable because no current injury/return feed is held by the hosted server. `go test -race ./internal/nativecheck`, all frontend tests (115 files / 886 tests), `pnpm tsc --noEmit`, and `make lint` pass. Wails bindings regenerated for the two optional response fields.
- Next: Final diff review, commit, push through normal hooks, create a draft PR with source-trace table, verify it is not merged.
- Blocked on: `tidy-check` names six dangling Chromium symlinks in external `~/scratch/bee-resume-v14-{browser,render2}` folders; left untouched because writes are outside this task's authorization. Heavy guard currently clear.
- Tried/rejected: No separate read or server-side MFL poll. No guessed injury/back data; exact missing input is MFL `TYPE=injuries` (`status`, `details`, `exp_return`) plus existing NFL schedule return-week resolution, currently local-only. Any hosted acquisition would need to reconcile with the plan's no-ongoing-MFL-sync ruling.
