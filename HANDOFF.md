# Handoff

- Baton: Bee, 2026-11-18.
- State: Complete — PR #360 (`bundle/b4`) is open and CI is green. Thin Home-first rail and held-read player search are in place; HQ roster, War Room, and Pulse player names open existing inspectors. Pulse Race franchise names use the existing franchise inspector. Local mode remains frozen.
- Source trace: thin entry is `NativeStartup` → `NativeAccount` → hosted league views. HQ/War Room already have their inspectors; Pulse is thin. Existing `ShellRail` dispatches local commands, so a shared presentational rail extraction must preserve local behavior.
- Next: review and merge PR #360. League-name search is skipped for alpha.
- Waiting: no feature blocker. `tidy-check.sh` found detached fleet lanes `pulse-b5fix`, `pulse-b5league`, `pulse-b5people`, `pulse-b5picks`, `pulse-b5server`, and `pulse-b2docket`; they are unrelated/in flight and were left running.
- Checks: `pnpm exec tsc --noEmit`, full `npx vitest run` (124 files/915 tests), and `make lint` pass. First push gate caught long-line limits and an async HQ assertion; fixed. Pre-push verify and all PR CI checks pass. Tidy only named unrelated live lanes above.
- Previous state: b0load built per head-brain ruling 11-18. `league-load` creates an empty `server.db` when `-db` is absent and its directory is empty (`prepareFirstLoad`); otherwise existing checks. `install.sh`/`proxmox.sh` take opt-in `TWR_FIRST_LOAD=1`: server installed, stopped, unscheduled, empty `/var/lib/twr`; next commands printed; proxmox readiness skips twr-server status in that mode. Default install unchanged.
- Refactor: `loadLeague` split into mirror wiring + `loadHeld(heldLeague)` so the cmd test loads the Legacy test export without a mirror file.
- Tests: `cmd/twr-server/league_first_load_test.go` (fresh dir → prepare → load 32 franchises; second load refused "already on this server"; non-empty dir refused, nothing created).
- Not executed: installer first-load path (needs an Alpine CT; `test-alpine.sh` not run — no containers in this lane). `sh -n`/`bash -n` pass; shellcheck not installed on Bee.
- Docs: deploy/server/README.md (new sequence, "does not yet provide" removed), cmd/twr-server/README.md.
