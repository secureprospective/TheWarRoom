# Handoff

- Baton: Bee, 2026-11-18.
- State: Fixed thin `ThinHome` is mounted first after sign-in; the card/shortcut opens existing `HostedLeague`, with a Home return. It takes `LeagueChoice[]`; alpha builds one choice from the admitted league only. No `commish.leagues.read`, cross-league probes, or local people/research reads.
- Source trace: thin entry is `NativeStartup` → `NativeAccount`; `ThinHome` reuses `gatherNeedsYou`, `readHomeBrief`, and existing Home helpers. `thinHomeRoute` exports `homeRoute`. Local `HomeBrief`/`AppLeagueMount` remain untouched.
- Checks: targeted Vitest 34/34 passed; `pnpm exec tsc --noEmit`, `make lint`, and `make bloat` passed.
- Next: commit, push, open PR into `bundle/b3`, and check CI.
- Blocked on: none. `tidy-check.sh` reported unrelated live `pulse-b5jobs` launcher PID 4038372; left it running. Its scratch/KEEP is outside the authorized write paths.
- Tried/rejected: no local `TargetApp`/`AppLeagueMount` port; no server-wide league enumeration or cross-league read probing (privacy ruling).
