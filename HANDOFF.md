# Handoff

- Baton: ClaudeBox review, 2026-11-18; Bee finished this rerun.
- State: CT 260 is unchanged (release 8 / `c4d38df6c`, server stopped, data empty). Current main `cfb7d5d34` includes #346; a matching LAB-only bundle was built but not transferred.
- Job 1 NOT MET: no documented way found to install current main on this existing CT while keeping the server stopped. Existing `twr-upgrade` starts it; fresh `TWR_FIRST_LOAD=1` install refuses existing `/etc/twr` and CT 260. No credential staged or import attempted.
- Jobs 2–3 NOT MET: league absent, so no post-import upgrade/persistence proof or stress traffic.
- Job 4 NOT MET: Claude-OS had an active `seat0` session and `thewarroom-main` PID 182698; left untouched.
- Job 5 MET: report and evidence are in `../../../../fleet/runs/pulse-lab4/RESULT.md` and `../../../../fleet/runs/pulse-lab4/evidence/`.
- Tried/rejected: bundle from stale lane checkout `554b03f2f` omitted #346; not used. Correct bundle from `cfb7d5d34` built with the matching CT root, but not deployed.
- Next: approve a supported no-start update path for CT 260; release Claude-OS for testing; then resume jobs 1–4.
- Cleanup `rm -rf .scratch deploy/lab/__pycache__` was blocked (`Command recursively deletes; blocked (no UI for confirmation)`); no retry. `.scratch/KEEP.md` records our retained artifacts.
- `tidy-check.sh` found foreign active lanes PID 3148274 (`b1whatif`), 3148435 (`b5frame`), 3148600 (`b5jobs`), 3148836 (`b5perm`). Their scratch/KEEP files belong to those owners; not stopped or edited. See `../../../../fleet/runs/pulse-lab4/evidence/cleanup-blocker.txt`.
- No remote state/service changes, stress traffic, secrets, repo source changes, PR, or container power action.
