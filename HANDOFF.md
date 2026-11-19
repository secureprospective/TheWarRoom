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
