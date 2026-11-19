# Handoff

## Part 2 — bundle/b9 #398 integration (2026-11-19)
- Baton: ClaudeBox for review; PR #400 into `bundle/b9`.
- State: merge commit `03f2a8d54` integrates `origin/bundle/b9` (`eb27c6e2a`, including #398 synced Home boards and Ask). ThinHome retains B9 phone boards and B7's role-derived commissioner read behavior; both handoff histories remain. PR body records this choice.
- Verified: eight focused frontend files/53 tests and tsc pass; pre-push passed; PR CI green on pushed commit `5ee0469b6` (all four test shards, checks, release-tools, package, install).
- Next: ClaudeBox review/merge PR #400; do not merge from this lane.
- Blocked on: none.
- Tried and rejected: none.


## lane/b9take7 progress (2026-11-18)
- Baton: ClaudeBox for review; PR #400 into `bundle/b9`; merge commit `a7d6b6ed1` merges `origin/bundle/b7` into B9.
- State: conflicts resolved with B9 phone routes/layout and new reads alongside B7 browser ingress/security and franchise schedule/read contracts. Session security remains same-origin, `SameSite=Strict`, and rejects Authorization with a cookie. `league.activity.read` and commissioner filtered-ledger reads remain distinct.
- Verified: Go build/vet, lint, no-Origin valid-cookie session regression, `tsc`, hosted Pulse tests; pre-push passes. PR CI is green: all test shards, checks, release-tools, package and install.
- Next: ClaudeBox review/merge PR #400; do not merge from this lane.
- Blocked on: none.

- Baton: ClaudeBox for review; Bee completed lane `pulse-b7door`, 2026-11-18, branch `session/pulse-b7door`, base `bundle/b7` (`b597b6122`).
- State: commit `84f79ddab` pushed to ready PR #372; CI green. Browser door implemented without schema migration or league-recency state. `POST /app/session` returns 409 with the desktop-app message for accounts admitted to multiple leagues. `make verify` passed, including race tests, frontend build, lint and staged embedded-bundle test; staging cleaned.
- Flow: exact configured HTTPS Origin and Host → `/app/session` creates/revokes the existing owner session in `__Host-twr_session`; GET/POST return only the one admitted league. Native command routes accept that cookie only from the configured origin and retain current authorization/TOTP checks. `/app/` embeds the built web app with locked headers.
- Security: browser never receives owner bearer or recent-auth proof; reject Authorization+cookie and agent tokens with Origin or as cookie; changes feed, enrollment and account-administration routes stay outside the cookie exception. No league list or `commish.leagues.read`.
- Deployment: `-public-host` is sourced from `/etc/twr/public-host`; Cloudflare routes `/app`, and upgrade recovers the hostname from existing tunnel config. Static staging is cleaned after verify/release build.
- Next: ClaudeBox review/merge PR #372. Alpine deployment integration was not run; changed shell scripts passed syntax checks.
- Tidy: `tidy-check.sh` found unrelated active alpha010 lanes `fixcr`, `fixhome`, and `fixsearch` launched from the main checkout. Left their processes/files untouched; their owners need to file KEEP records. Flag this to ClaudeBox.
- Blocked on: none for this lane. The unrelated active lanes still need their owners to file KEEP records.
- Tried and rejected: persistent last-used league selection (requires a migration and contradicts Part 3); no picker or league-list endpoint.

## pulse-b9boards lane (Bee, 2026-11-18)
- Baton: ClaudeBox review; implementation ready on `session/pulse-b9boards`, base `bundle/b9`, PR #388.
- State: server preference store/table implemented; schema migration is sign-in file v4. Layout writes are bounded/validated and revision-checked. The account-only workflow uses `/v1/account/preferences` so the existing sign-in backup admission guards writes without pausing league writes. Thin Wails forwards the account command there; browser cookie calls require exact same-origin. Both agent token scope and access refuse it. Sign-in conversion copies/drops the new table when present and safely skips it for older one-file sources.
- Flow: typed Raw read/set → catalog validation → session-authenticated account ID → account-store CAS. Conflicts return the current server copy; no league/franchise argument and no private reply caching. Hosted HomeBrief reads server-first, caches only as an offline fallback, and keeps local mode on localStorage.
- Evidence: focused race tests pass in all changed Go packages, including existing/absent layout sign-in conversion; `make lint`, nativegen freshness, frontend TypeScript and layout/transport tests pass. Frontend fresh-store test proves a second device reads the server copy. CI passed on `007cb5dd1`: all four race shards, lint/checks, release-tools, package and install. The first shard attempt stopped without a reported test failure; the retry passed.
- Next: ClaudeBox review/merge PR #388.
- Blocked on: none. `tidy-check.sh` reports unrelated active lanes/hosted test server; left untouched for their owners.
- Tried and rejected: reusing a per-franchise league store (wrong owner boundary and not cross-league); no settings/preferences table exists in the hosted account store.

## Bee — pulse-b9cr1
- Baton: Bee implementing B9-5 Control Room phone layout, 2026-11-18, branch `session/pulse-b9cr1`, target `bundle/b9`.
- State: Commit `3e8edb5aa` rebased onto current `bundle/b9`. Added <700px hosted Control Room layout rules and focused phone CSS test; reads, commissioner checks and authorization unchanged. Focused tests 14/14; full Vitest 995/995; tsc, frontend build, bundle budget and CSP passed. `make verify` exceeded command time limit in Go tests. Push pre-check failed on one unrelated `nativeAccount.test.tsx` flake; that test passed alone on retry.
- Next: Retry pre-push/push, open PR into `bundle/b9`, verify CI.
- Blocked on: Heavy guard refuses new suites (disk 91%, other full suite activity); do not clear shared cache or touch other lane files. Claude-OS/Brave phone browser not installed; 390px CSS contract test passed, interactive emulation unverified.
- Tried and rejected: no JavaScript viewport mode; retain existing hosted screen and use the approved CSS breakpoint.

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

- Baton: Bee, 2026-11-18; B9-2 lane on `session/b9-2-league-activity`, based on `origin/bundle/b9`.
- State: phone League view now has Table, Schedule, Activity and Live tabs. Activity uses only the validated member-safe public feed; player status remains visibly unavailable. Blank/invalid NFL kickoffs read “dates not published yet.”
- Evidence: 147 Vitest files/1005 tests and frontend tsc pass; member-safe activity unit and hosted-read tests pass under `-race`. The phone CSS regression covers 390px and 44px targets.
- Next: commit, push through pre-push, open PR to `bundle/b9`, wait for CI.
- Blocked on: pre-push and PR CI.
- Tried and rejected: none.

- Baton: Bee, 2026-11-18; B9-2 implementation complete, PR #397 open into `bundle/b9`.
- State: commit `b3477a5e3` is pushed; pre-push verify and every PR check passed. No merge requested.
- Evidence: full Vitest 147/1005, tsc, member-safe activity race tests, 390px CSS test, PR checks green.
- Next: leave PR for review.
- Blocked on: review/merge decision.
- Tried and rejected: none.

## Bee — pulse-sec9 security review (2026-11-19)
- Baton: ClaudeBox review; PR #405 into `bundle/b9`, branch `session/pulse-sec9-private-pref`, head `2988d6f2e`.
- State: #387, #388, #389 and #400 pass. #5 found hosted account-layout localStorage/offline caching in B9; PR #405 removes it, retaining local-mode preference storage. Regression reproduced before the fix.
- Evidence: targeted Go race tests pass for activity, account preferences, roster.add and browser origin policy; 35 frontend tests and `pnpm exec tsc --noEmit` pass. Pre-push `verify` passes. PR checks: release-tools passed; four test shards and `checks` pending at close.
- Report: `~/fleet/runs/pulse-sec9/RESULT.md` (9 lines).
- Next: ClaudeBox review/merge PR #405; do not merge from this lane.
- Blocked on: PR CI and review.
- Tried and rejected: broad unfiltered Go race package run exceeded the 600s tool limit; scoped security tests passed. First pre-push exposed a raw test button/long line; replaced it with `HQAct` and reran the focused 35 frontend tests successfully.
