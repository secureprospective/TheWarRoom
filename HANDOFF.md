# Handoff

- Baton: Bee, 2026-11-18.
- State: Fixed thin `ThinHome` mounts after admission; alpha exposes only the admitted league. Cards open hosted HQ, shortcuts and actionable Needs-you items pass their hosted entry, and Control Room opens hosted rules. Local mode remains untouched.
- Reads: Home derives people scope from admission, not local bindings. Franchise members reuse hosted `readHomeBrief` without local research; unassigned commissioners read league Pulse and jobs only (calendar requires a franchise). No `commish.leagues.read` or cross-league probes.
- Checks: targeted Vitest 51/51; `pnpm exec tsc --noEmit`, `make lint`, and `make bloat` pass. First pre-push attempt found the full frontend suite's old route assumptions and line-length issues; fixed, targeted tests now pass.
- Next: commit the route/read follow-up, push, open one PR into `bundle/b3`, and check CI.
- Blocked on: no project blocker. `tidy-check.sh` reported the unrelated live `pulse-b5league` launcher (PID 4122593); left it running and outside this worktree untouched.
- Tried/rejected: no local `TargetApp`/`AppLeagueMount` port; no server-wide league enumeration or cross-league read probing (privacy ruling).
