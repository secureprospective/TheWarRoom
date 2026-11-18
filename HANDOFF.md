# Handoff

- Baton: Bee, 2026-11-18.
- State: Blocked — the brief targets `ShellRail` in `TargetApp`, which only mounts in frozen local mode. `CLAUDE.md`, `docs/architecture/desktop-modes.md`, and the governing client/server plan say local mode must not grow; thin mode does not mount `ShellRail`. Need Christopher/ClaudeBox to confirm a re-open or redirect this lane to thin mode before implementation. No feature changes made.
- Verified Home already exists as `nodeKeys[0]` and `nav.open` routes it to `home`; `leaguePicker.interaction.test.tsx` already clicks Home from War Room and checks the Home workspace. The player search/card task remains unimplemented.
- Checks: `git pull --ff-only` clean/up to date. Desktop-mode source checks agree with the mode split. `heavy-ok.sh` initially refused while one full `-race` suite ran; after 120 seconds it passed (no full suite running).
- Previous state: b0load built per head-brain ruling 11-18. `league-load` creates an empty `server.db` when `-db` is absent and its directory is empty (`prepareFirstLoad`); otherwise existing checks. `install.sh`/`proxmox.sh` take opt-in `TWR_FIRST_LOAD=1`: server installed, stopped, unscheduled, empty `/var/lib/twr`; next commands printed; proxmox readiness skips twr-server status in that mode. Default install unchanged.
- Refactor: `loadLeague` split into mirror wiring + `loadHeld(heldLeague)` so the cmd test loads the Legacy test export without a mirror file.
- Tests: `cmd/twr-server/league_first_load_test.go` (fresh dir → prepare → load 32 franchises; second load refused "already on this server"; non-empty dir refused, nothing created).
- Not executed: installer first-load path (needs an Alpine CT; `test-alpine.sh` not run — no containers in this lane). `sh -n`/`bash -n` pass; shellcheck not installed on Bee.
- Docs: deploy/server/README.md (new sequence, "does not yet provide" removed), cmd/twr-server/README.md.
