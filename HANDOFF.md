# Handoff

- Baton: Christopher, 2026-11-18; PR #363 is open into `bundle/b3`.
- State: Fixed thin `ThinHome` mounts after admission; alpha exposes only the admitted league. Cards open hosted HQ, shortcuts and actionable Needs-you items pass their hosted entry, and Control Room opens hosted rules. Local mode remains untouched.
- Reads: Home derives people scope from admission, not local bindings. Franchise members reuse hosted `readHomeBrief` without local research; unassigned commissioners read league Pulse and jobs only (calendar requires a franchise). No `commish.leagues.read` or cross-league probes.
- Checks: targeted Vitest 51/51; `pnpm exec tsc --noEmit`, `make lint`, and `make bloat` pass. Pre-push verify passed; PR checks are green at `b7a157029`.
- Next: review PR #363 and manually inspect thin Home plus its HQ/War Room/Pulse/Control Room and Needs-you entry routes before merge.
- Blocked on: `tidy-check.sh` reported unrelated live `pulse-b5backup` (PID 90442), `pulse-b4sync` (PID 128631), and `pulse-b5league` (PID 4122593) launchers. Left them running; their workdirs/KEEP paths are outside this authorized worktree.
- Tried/rejected: no local `TargetApp`/`AppLeagueMount` port; no server-wide league enumeration or cross-league read probing (privacy ruling).
