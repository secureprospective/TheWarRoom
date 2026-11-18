# Handoff

- Baton: Bee, 2026-11-18.
- State: b0load built per head-brain ruling 11-18. `league-load` creates an empty `server.db` when `-db` is absent and its directory is empty (`prepareFirstLoad`); otherwise existing checks. `install.sh`/`proxmox.sh` take opt-in `TWR_FIRST_LOAD=1`: server installed, stopped, unscheduled, empty `/var/lib/twr`; next commands printed; proxmox readiness skips twr-server status in that mode. Default install unchanged.
- Source trace: updater is `deploy/server/upgrade.sh`; league load is `runLeagueLoad` → `prepareFirstLoad` → `openStateFiles`/file creation. Lab evidence: `~/fleet/runs/pulse-lab4/RESULT.md`.
- Implementation: updater opt-in requires empty state + stopped/unscheduled service, verifies and installs the signed bundle, skips DB preflight/snapshot/server start, and prints the next load/start steps. `league-load` individually chowns only paths newly added in the fixed state layout to the state directory uid/gid; symlinks are not followed. Installer/proxmox/server/upgrade docs no longer prescribe partial chown.
- Evidence: `go test -race ./cmd/twr-server/...` passed (349.043s); `make lint`, `make bloat`, `git diff --check`, `sh -n`/`bash -n` passed. Root-only UID/GID test skips on Bee (uid 1000); helper test runs.
- Not run: `test-alpine.sh`/ShellCheck; its required `.scratch/server.tar.gz` and `.scratch/` are absent, and the lane must not create test scratch outside the shared scratch workbench. The shell test was added to the Alpine harness; host shell syntax passes. CI remains pending.
- Tidy: `tidy-check.sh` found foreign active lane PID 3945562 (`luna b3home`); left untouched (not started by this lane / do not edit its files).
- Next: final review, commit/push this branch, open PR to `main`, wait for CI. Do not merge.
