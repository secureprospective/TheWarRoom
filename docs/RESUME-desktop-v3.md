# RESUME — Desktop v3 finish (and phone v1) — started 2026-11-20

## 0. Christopher's rulings for this workstream (2026-11-20, read first)
- **Finish the product to the mock. No slop, no "ship less" options, the date never cuts quality.** Sunday 11-22 is a checkpoint, not the finish. (memory: feedback_twr_finish_product_not_ship_date)
- **Claude (head brain) does the UI work itself.** No Luna. Haiku agents on Tom only for easy, mechanical jobs (inventories, greps, wiring checks, CI/merge chains).
- **Desktop target = `~/Downloads/TheWarRoom-mock-v3-FINAL.html` (Beelink).** Phone target = `~/Downloads/TheWarRoom-mobile-mock-v1.html`, never a blend.
- **All UI testing on Claude-OS at 1920x1080** (`ssh claudeos`; shared desktop — check idle first, leave it running). Done for a screen = side-by-side screenshot with the mock at 1920x1080, taken by Claude, before Christopher sees it.
- His verdict on the current desktop (thin mode PWA, screenshot `/home/chris/content/feedback/2026-11-20_08-51.png`): "the UI is dogshit, not close". **The wiring (server reads/commands) is likely in place; the UI is the job.**
- Merges for this work are approved (squash, after green CI). Hard limits in CLAUDE.md still apply (no email, no Proxmox host changes, never print secrets/join codes/passwords, never --no-verify).

## 1. What we are doing
Bring the thin-mode desktop app (the product, `docs/architecture/desktop-modes.md`) up to mock v3 region by region, then the phone to mobile v1. Repo: Beelink `~/work/TheWarRoom`, worktrees in `.worktrees/`, branch per piece `session/<desc>`, never work on main.
Key fact: most v3-looking pieces exist in the **legacy local shell** `TargetApp.tsx` (Home, War Room, HQ, Pulse, Control Room, command bar). Thin mode (`NativeStartup.tsx` → `NativeAccount` → `ThinHome` → `HostedLeague`/`HostedWorkflows`) got a bare rail + Pulse. Plan: bring the v3 shell into thin mode, feed it from hosted reads, then polish each panel to the mock. Legacy mode must not grow (desktop-modes.md).

## 2. Done looks like (all NOT met yet)
1. Every region of mock v3 exists in the thin app with real data: top bar (league picker, search/command, Needs you, Assistant, Feeds, avatar), NOW ticker, tab strip (Home, Today, Game day, All leagues live, Commissioner desk, Front office, Board, Add a piece, Edit), left rail (App nav, My shortcuts, league picker list), Home panels (Docket, This week across my leagues, My players today, NFL news, Deadlines, Shortcuts), status bar. Side-by-side at 1920x1080 on Claude-OS.
2. Phone: every tab matches mobile v1, side by side.
3. Data behind the panels is right on the lab (current week, lineups, projections, values) — empty panels from missing data are defects.
4. Tracker `~/Downloads/twr-usable-week-plan.html` lists each region matched/not with screenshots.

## 3. Next actions, in order
1. **Gap list (Claude, by hand):** open mock v3 at 1920x1080 and the thin app (commissioner sign-in) at 1920x1080; for each region record: mock look, existing code (thin or legacy file), the hosted read that feeds it, grade match/partial/missing. Write to `docs/desktop-v3-gaps.md` on a branch.
2. Build the shell first (top bar, tab strip, rail, status bar) in thin mode, then Home panels, then the other tabs. One PR per piece, screenshot-checked against the mock before merge.
3. Lab week fix (see §4) — must be done before any screenshot judging "week" panels.
4. Small open item: sign-in screen label. Worktree `.worktrees/signin-label`, branch `session/signin-code-optional`, **no changes made yet**. Change `NativeAccount.tsx:307`: on the login screen label "Authenticator code (only if you set one up)"; keep "Six-digit authenticator code" on the authenticator screen. Add a test in `frontend/src/app/control/nativeAccount.test.tsx`. (Rehearsal also saw a button click not submit on the first try; unproven, don't chase without evidence.)
5. Lab housekeeping before any outside join: free the rehearsal seat (franchise.01 BUF, creds `~/fleet/runs/pulse-lab4/private/rehearsal-owner.json`; use `~/fleet/runs/hk-seats/free_seats.py` pattern — it skips only the commissioner seat), then rotate the join link (summary of CLI in §6).
6. Hermes weekly job (Tue 03:00 CT) to run refresh-nfl-schedule on the lab (stop server, refresh, start) so the week keeps advancing until the server refreshes on its own (post-fix design item).

## 4. In flight at handoff
- **PR #450** (refresh-nfl-schedule re-fetches past not-final weeks) MERGED 7968877. Claudebox background chain `btgcqjvoo` then runs lab release **v22**: log `~/fleet/runs/overnight-b9walk-1120/labrel-v22/out.log` (Beelink). Check: `tail -3` that file for `LAB RELEASE DONE`.
- **After v22:** run on the lab (server must be stopped): copy of `~/fleet/runs/overnight-b9walk-1120/lab-nfl.sh` (it backs up, stops, runs `twr-server refresh-nfl-schedule -db /var/lib/twr/server.db -league native.mfl-14432-2026 -season 2026` as user twr, starts, prints status) via `bash ~/fleet/runs/overnight-b9walk-1120/labrun.sh <script>`. Expect "Re-fetched results for N past week(s) that were not final: 9, 11." (11 only if its games are over). Then `week.read` should give current week 11 (Sunday 11-22 is week 11). If the clock still says 9: the archived re-fetch is not what the schedule cache reads — next suspect is `internal/app/commissioner/native/schedule_cache.go` (reads `History.ReferenceBodies`, newest copy first; cache keyed on archive mark) vs where the transport sink writes. Medium confidence on that hypothesis.

## 5. State / gates
| Item | State |
|---|---|
| Lab CT 260 | release v21 (1c8be58d) live; v22 pending. 1 commissioner account + rehearsal owner on BUF. 31 open seats. |
| PR #448 Live follows selected league | merged, lab v20 |
| PR #449 phone Team skips value.read (banner gone) | merged, lab v21 |
| PR #450 stale-week re-fetch | merged, v22 in flight |
| CI on main | green at #450 merge |
| Backup | `/var/lib/twr-backup-sat-1120/twr-full.tgz` 203,087,044 B, sha256 73da36…; restore verified (16 dbs integrity ok) |
| Rate limits (labcfg lane) | fine: per-IP via CF-Connecting-IP, 10 auth/min/IP, 64 in-flight global, 6 auth in-flight; sessions 7d idle/30d |
| Server drift since 11-16 | no migrations, no TODO/panic; draft.queue seat check concern REFUTED (seat comes from caller's own principal, `draft_queue.go:88-111`) |

## 6. Refuted / known — do not retest
- Team-tab "Some reads are unavailable" = `value.read` 409 for everyone: lab has no values feed ("player records and fitted parameters are not available on this host"). Fixed by #449 for phone; desktop HQ still reads values — the values feed itself is a data item.
- Week stuck on 9 = held nflSchedule copy for week 9 fetched before games ended (`gameSecondsRemaining` != 0); week 11 copy fetched Thu 11-19 16:02Z. `leagueweek.LineupWeek` needs all games Final.
- Empty starters / PROJ 0.0 on lab = no lineup set + no NFL stats held (projections.read available:false). Data, not code.
- Live "updated 9:12 AM · ESPN · provisional" = board line, as designed; "Updated n min ago" only inside a followed game.
- Commissioner sign-in does NOT need an authenticator (ruling 11-14, `accounts/totp.go:221`). Leave the code box empty. Browser sign-in verified 200 with stored creds.
- Blank screen after sign-in on the installed PWA = stale page across the v21 restart; fresh browser renders Home fine.
- golangci-lint-full "153 issues naming a deleted worktree" = stale cache: `golangci-lint cache clean`, then commit normally.

## 7. Tools and access
- Lab shell: `bash ~/fleet/runs/overnight-b9walk-1120/labrun.sh <script-file>` (busybox; no python/sqlite/journalctl; log `/var/lib/twr/server.log`). Lab URL `https://twr-lab-260.secureprospective.com/app/`, league `native.mfl-14432-2026`.
- API helpers: `~/fleet/runs/hk-seats/free_seats.py` (login/post; workflow reads: season/week are integers). Creds in `~/fleet/runs/pulse-lab4/private/` — never print.
- Lab release: `~/fleet/bin/twr-lab-release.sh <V> <prev>/release-v<prev>/metadata/<prev>.targets.json <run-dir>` (builds origin/main).
- Haiku launcher: `~/fleet/runs/overnight-b9walk-1120/hkx.sh <lane>` with brief `briefs/hk-<lane>.md` → worktree `.worktrees/hk-<lane>`, output `~/fleet/runs/hk-<lane>/{stdout.log,completion.txt}`. Launch: `setsid nohup bash .../hkx.sh <lane> >/dev/null 2>&1 </dev/null &`.
- Headless screenshot without playwright: node 24 + brave `--headless=new --remote-debugging-port=…` over CDP (built-in WebSocket); fill React inputs via the native value setter + input event, `form.requestSubmit()`. Redact the password from output. (For the real check use Claude-OS 1920x1080.)
- Christopher installed the PWA on the Beelink in Brave (shows "Uninstall" in title bar).

## 8. Honest status
Wiring and server: solid, CI green, lab healthy. Desktop UI: far from mock v3 — essentially unstarted against the mock. Phone: close to mobile v1 per the earlier walks, not yet side-by-side verified at full fidelity. ETA for desktop v3 match: several days of focused work; not Sunday.
