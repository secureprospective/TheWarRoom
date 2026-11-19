# Handoff

- Baton: Bee, lane `pulse-b7lab2`, 2026-11-18.
- State: Bundle 7 merge `dd6f5c615` is pushed to `origin/bundle/b7`; `make verify` and all 11 release tests passed. Signed LAB-only release upgraded CT 260; post-upgrade status is clean, history is 3,399, and backups verify in R2. Owner/commissioner desktop and PWA walks are complete. The 390×844 pass is verified; mobile panes overflow and findings are in the lane report.
- Findings: commissioner People returns `League people.actors: unexpected field`; commissioner search misses a Market-listed player; Pulse has no current-week read for either role; owner Home League Read returns the generic action error. Commissioner Home no longer shows the lab5 seed error. No source changes or PRs.
- Next: Christopher review the reported findings and choose any follow-up fix lane. This test lane is otherwise complete.
- Blocked on: #381 recovery was not independently demonstrated because the host and `/app/` route were already correct before upgrade. The run-scoped browser profile remains after the command guard blocked recursive deletion; its browser process is closed and the final session was revoked. Earlier test sessions may remain until idle expiry; I did not revoke all devices because that could interrupt another test client. The pre-existing isolated browser profile was left untouched because it was live and held unrelated pages. Final tidy check also found foreign active lane `b9fa` (Luna); left it running and did not edit its scratch. Its owner needs the required `KEEP.md`.
- Tried and rejected: release metadata v3 was refused by CT's cached trust state; rebuilt with metadata v5 and the established LAB-only signing path. No manual host/tunnel repair was attempted.

## Bee — pulse-b7fix2 (2026-11-18)
- Baton: Bee, working the three b7lab2 browser read findings and #381 recovery proof.
- State: Pulled `bundle/b7`; no initial worktree changes. Heavy-work guard passed. Source evidence and governing contracts are being reviewed before changes.
- Next: Verify each finding against the catalog, implementation and lab evidence; fix only proven code defects, run targeted tests, then do the lab-only recovery check and prepare PR/report.
- Blocked on: none yet.
- Tried and rejected: none.

### Bee progress — evidence triage
- People: browser `commish.people.read` is HTTP 200; the typed server record includes `actors` and `yourPermissions`, which the strict frontend parser omits. The client contract is stale; preserve strict unknown-field rejection and parse the catalog's full reading.
- Owner Home: DevTools shows `schedule.franchise.read` HTTP 400 because its arguments are absent from `RawWorkflowScope.commandScope`; the app asks for a league/franchise/season/zone already permitted by the catalog. Separately, Home attempts commissioner-only `commish.jobs.read` (HTTP 409); gate that read on commissioner authority rather than widening permission.
- Pulse: both roles' `week.read` is HTTP 200 with 13 published league schedule rows, no current/next week, and `The NFL week dates are not held yet.` Nine scheduled rows lack NFL kickoff dates (weeks 1–5, 10–12; five have dates). Commissioner Pulse currently hides that server message behind the generic wait state; correct the display and point to the server operator, not league schedule edits.
- Lab: CT 260 is running, host `twr-lab-260.secureprospective.com`; pinned LAB root hash matches the CT 258 signing directory. `public-host` and tunnel config are currently live, so recovery drill will back up both before deliberately removing the required state.
- Correction to the preceding Pulse bullet: eight rows lack NFL kickoff dates (weeks 1–5 and 10–12); five of the 13 rows have dates.

### Bee progress — fixes in review
- Added the strict `People.actors` / `yourPermissions` parser contract and tests; unknown top-level and nested fields still fail.
- Added `schedule.franchise.read` to the hosted workflow read map and validated its existing catalog scope; owner read succeeds in tests, outsiders remain denied. The separate Home jobs read now requires Commissioner/Co-commissioner role (not a deputy's change-rules flag).
- Pulse wait-state keys now follow the server's note; the server gives the commissioner an operator-directed action when held NFL dates are missing, without implying league schedule edits can fix it.
- `make lint` passes. Targeted Go and frontend tests pass; full `internal/serverhttp` race suite exceeded its 10-minute test timeout during the first broad run, so endpoint tests were rerun narrowly. TypeScript check remains to rerun after the latest role-gating edit.
- Next: finish targeted verification, commit/push, open the PR, build a LAB-only bundle, then perform CT 260 recovery drill and browser retest.

### Bee progress — CI and lab preflight
- PR #394 is open. CI exposed a stale generated browser-door map for the newly cataloged schedule read; regenerated `frontend/src/app/commands/arguments.gen.ts`, and its freshness test plus frontend checks now pass. Re-run CI after pushing the generated change.
- CT preflight: Proxmox host `twr-lab`, CT 260 running as `twr-260`; both OpenRC services are started, `/app` ingress is present, and `/etc/twr/public-host` is mode 0640 root:twr. Root hash is `8c91770c…`, matching the lab-260 signing root.
- Next: prepare a scoped copy of the matching signing material in this run's private evidence workspace, build the LAB-ONLY branch bundle, back up/remove only the requested browser state, and run the installed normal upgrader.

### Bee progress — CT recovery and release retest (2026-11-19)
- Commits `2467dd9a8` and `f97d348e2` are pushed on `session/pulse-b7fix2`; PR #394 is open. `verify`, `checks`, release-tools, and all four test shards pass; the package job was still pending at last check.
- LAB-only bundle `r9` with TUF targets v6 was built from `f97d348e2`. On CT 260, backed up the tunnel config and `/etc/twr/public-host`, removed only `/app` and the hostname file, then ran `/usr/local/libexec/twr-upgrade`. The upgrader itself reported restoring both; no manual repair ran. Evidence is under `~/fleet/runs/pulse-b7fix2/evidence/381-*.txt` and `381-upgrade.log`.
- Recovery checks: tunnel validation passes, both services started, `twr-status` reports healthy, `/app` and `/app/` return HTTP 200, and restored hostname bytes match backup. The normal upgrade performed its standard automatic off-site backup; no league backup action was submitted.
- Browser smoke uncovered the bundle script built `frontend/dist` instead of the hosted `dist-web`: initial HTML loaded but lazy chunks requested `/assets` and returned 404. The existing `build:web` script is the correct hosted build. Updating `deploy/server/bundle.sh` to use it; need a new TUF targets version (v7) for changed bytes, normal upgrade, then complete screen retests.
- Next: verify and commit the bundle-mode correction, push and wait for green CI, rebuild/sign v7, upgrade CT 260 normally, retest People/Owner Home/Pulse in the isolated profile, write `RESULT.md`, and run tidy checks.
- Blocked on: package CI check and completion of the hosted-mode rebuild/retest.
- Tried and rejected: plain `pnpm build` produced root-relative lazy chunk imports and 404s under `/app`; do not reuse that bundle for browser verification.

### Bee closeout — pulse-b7fix2 (2026-11-19)
- State: task done. PR #394 is open into `bundle/b7` at `2cc5cc0ff`; all checks were green. Final report: `~/fleet/runs/pulse-b7fix2/RESULT.md`.
- People and owner Home passed in isolated Brave against the final hosted build; Pulse was retested for both roles and correctly identifies missing held NFL dates. Screenshots and sanitized network evidence are in the run folder.
- #381 recovery passed on CT 260: backed up the tunnel config and public-host, removed only `/app` and the hostname file, and let the installed normal upgrader restore them. No hand repair. Final CT build is `2cc5cc0ff`; services, tunnel validation, public routes and status are healthy.
- No league mutations or manual backup action. Both normal upgrades made their standard automatic R2 recovery backups; status verified them. Temporary CT/PVE staging and the drill-only file copies were removed.
- The plain `pnpm build` bundle had lazy `/assets` 404s; `deploy/server/bundle.sh` now uses existing `build:web`. The final LAB v7 package served all sampled browser assets under `/app/assets`.
- Tidy: the task-specific Brave profile was signed out and closed. `tidy-check.sh` reports ADB PID 2504044 (PPID 1, cwd `/home/chris/work/TheWarRoom`); left it running because it is outside this worktree and has no associated scratch folder. The reusable pinned lab venv remains with `.scratch/KEEP.md`.
- Next: Christopher review/merge PR #394. Blocked: none in this task.
- Tried and rejected: plain `pnpm build` for hosted packaging; it emitted root-relative lazy asset paths. Use `pnpm run build:web`.

## Bee — pulse-mstub-team (2026-11-19)
- Baton: Bee completed the mobile Team-tab inventory lane.
- State: Read-only audit against `origin/bundle/b9`, the v3 plan and the supplied mock; 40 rows delivered in the required run artifact. No application source changes.
- Next: Christopher can use the inventory for the mobile Team implementation; no PR or lab action is part of this lane.
- Blocked on: none.
- Tried and rejected: creating another worktree at the assigned path; it was already registered and live, so used read-only `git show origin/bundle/b9` evidence instead.
