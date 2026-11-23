# Resume — Rehearsal season (from Sun 11-22, Tom on the Beelink)

## 1. What we are doing
Six NFL weeks left are a dress rehearsal: two NATIVE 32-team leagues run entirely by TWR (the 32-Team
Default, and a twin-pool league = the same default with `players.pools = per_side`, AFC 16 / NFC 16). Christopher
owns one team in each; the other 62 are automatic. Plan: `docs/reset/REHEARSAL-SEASON-1122.md` (LIVE, Claudebox
edits it). Environments (Dev / Lab CT 260 / Season server 192.168.1.162): CLAUDE.md "Environments".
Repo `~/work/TheWarRoom`; working branch **`session/rehearsal-auto`** (worktree `.worktrees/rehearsal-auto`).

## 2. Done looks like (Rehearsal A, ready Thu 12-03; real drafts Tue 12-08)
- [x] Gaps posted on the tracker before building (inventory `~/fleet/runs/twr-rehearsal-a-1122/`).
- [x] Auto-pick and auto-lineup built, each checked with a planted break. O-Line handout REMOVED (bfb91c18d): the auction close already hands leftover cards to every team without one.
- [x] Argon review addressed (5 BLOCKERs verified + fixed, each with a planted break: 106a71ae2 server facts, 563b51d11 clock advance, ebed15dcc lineup lock/wave/per-team, bfb91c18d handout revert). SHOULDs 6/7/9/10 open, see 13.
- [ ] Lab data check: `model_runs`/`model_scores` rows and weekly stats exist for the rehearsal leagues.
- [ ] Both leagues created on the Lab, one TEST draft each, clean; card to needs_you "ready for real drafts".

## 3. Agents + routing
- **11-23 7:35 AM (LATEST, overrides all below): own Haiku subagents only (Agent tool, model haiku). NO Claudebox Haiku /
  `claude -p` (Christopher needs Claudebox for other projects). AGY only when a 2nd opinion is absolutely necessary.**
- (older, superseded where it conflicts) 11-22 ~6 PM:
- **After the current lanes finish, Tom pushes the project himself.** Haiku subagents (Agent tool, model haiku)
  for small well-scoped jobs; **Sonnet at medium** for slightly larger ones. **Gemini weekly is thin: keep it
  for second opinions** (Argon reviews), not bulk building. Claudebox `claude -p` on a `git archive` copy
  still works for read-only surveys (launch shape in memory `haiku-runs-on-claudebox.md`).
- Briefs `~/fleet/briefs/`, runs `~/fleet/runs/<job>-1122/`. AGY: `~/fleet/bin/agy-run.sh <job> <brief> "<model>"
  <workdir>`; done = `exit` file. Quota 13:20: Gemini week 27.4% left (now less: 4 Neon runs + 1 Argon since).
- **11-22 ~9:40 PM Christopher: "just the Haiku on Claudebox the rest of the night".**
  - Shape: ship a `git archive` + `git init` repo to claudebox `/root/<job>-1122/repo`, plus a brief.
  - Launch: `nohup claude -p "$(cat brief.md)" --model haiku --effort high --allowedTools "Bash Read Edit Write Grep Glob"`.
  - The job writes `out/REPORT.md`, a format-patch and a `sentinel`.
  - Tom pulls the patch, applies it on a Beelink worktree, tests, plants a break, then commits and pushes.
  - Claudebox has Go 1.26.4 and 6 GB RAM.
- **Base every build on the branch the Lab runs** (lab-v5x / rehearsal-auto), not main. Main is 37+ commits behind.
  Memory: `base-briefs-on-live-stack.md`.

## 4. In flight (11-23 ~9:00 AM CT) — Christopher: "go ahead, keep driving"
- **Nothing running in the background.** No agents, no pushes, no watchers.
- **PR #470** `session/no-seat-quiet-1123` (worktree `.worktrees/no-seat-1123`, head 3c27c45cc): no-seat 403 on 'enter' →
  plain message (no red box) + this resume's 11-23 rulings. Pushed; CI not watched yet; WAITS for Christopher's merge OK.
  After merge: remove the worktree.
- **Report card job on Hermes** (installed 11-23): `~/fleet/twr-reportcard/{run.sh,twr-check,server.url}`, user timer
  `twr-reportcard.timer` Tue 9 AM CT (first fire Tue 11-24: exits quietly, before season). Missing: `~/fleet/twr-reportcard/token`
  (read-only connection token, chmod 600). Leagues native.rehtest-default/-twin; target lab. Test env vars REPORTCARD_DATE /
  REPORTCARD_PLAN. Build copy + KEEP.md: `~/scratch/reportcard-1123/`.
- Phones: S8 on wifi adb + Chrome inspector works (`~/fleet/bin/phone-chrome.py s8 eval|shot|goto`); iPhone inspector works on
  a real page. Keep the adb server (pid in tidy-check) running for the phones.
- Scratch `~/scratch/merge-1123/` (KEEP.md): PR bodies, s8-v54.png / s8-v55.png evidence.

## 5. Gates / status (11-23 ~9 AM)
| What | State |
|---|---|
| **main = 9fcc4a771** (#469) | after #468 2806fb444. Season box, report card checker, twr read, no-team sign-in, line-budget draft. CI all green |
| **Lab = v55** `session/no-seat-quiet-1123` 259e3a0d | main + no-seat polish. 8/8 OK at 8:49 AM. S8 shows only the plain no-team message + sign-in form (s8-v55.png) |
| Lab v54 = main 9fcc4a77 | 8/8 OK; S8 showed the message but with the raw red error above (fixed in v55) |
| Report card on Hermes | tested 3 paths on a copy of plan.json: before season quiet; no token → 1 red line; bad token → reaches Lab, plain refusal |
| **Lab = v53** `session/lab-v53-1122` a211f8f9e | v52 ce7417190 + the B7 owner-pick-door fix. Status 8/8 OK at 6:14 AM (sign-in backup was paused for the first ~15 min after restart, then OK) |
| Lab v50 b94fe7d1 / v51 087c96b9 / v52 ce741719 | Each shipped 8/8 OK. S8 self-updated e45a22cef5c1 -> 6b38e277e7ef -> 195e3b46bacf (T668 proven on Android). iPhone not checked |
| Lab data | model scores imported (run 1, 2026, 2,277). Test leagues native.rehtest-default + native.rehtest-twin created (default_32, 2026); commissioner account 564f67ef holds franchise.01 + commissioner seat in both |
| S8 account 3da8c020 | no league seat (403 on /app/session since 11-21): v52 screens unchecked on the phone |
| `session/b7-pick-door-1122` d626c91f6 (main base) | B7 fix + season box drafting via PickPath; TestHostedOwnerRookiePickAdmitted |
| `session/season-box-1122` 23742d295, PR #467 | season box: draft (store), lineups, lock, scores move, official final, stat correction, standings with 1 decided game, results. ~3 s |
| `session/twr-read-1122` cded5ccef (on t688 base) | `twr read` + MCP `league_read` |
| `session/line-budget-1122` 847154bce | draft line-budget guard; not in CI |
| `session/autopick-1122` 95d2b13e3 | **SUPERSEDED, do not merge.** Duplicates auto-pick that already exists on the Rehearsal A stack |
| `session/rehearsal-auto` c6fb1a8ca | stack + resume doc |
| `session/t688-replay-1122` 2461ed596 | twr-check comparator |
| Frontend installs | main checkout lacks vitest; worktrees symlink `.worktrees/t705/frontend/node_modules` |

## 6. Artifacts
- Stack `session/rehearsal-auto` (base 513b7058a): twin/twin32/midseason proof tests; auto-pick (starters first) + fixes 106a71ae2
  (server player facts) 563b51d11 (clock advance); auto-lineup + injuries + bye reserve + ebed15dcc (locked bench, wave window,
  per-team errors) + 8e48bb48a (side minimum reserve); O-Line handout REVERTED bfb91c18d; schedule test weeks 1-13 empty 53a8c2b4b;
  dd7e5ecd6+90da91b4b `twr-server export-model-run` / `import-model-scores`; a5f5a0d0a+17b80349a `twr-server create-league`.
- Model run export (Beelink history.db run 10, 2026-10-29): `~/fleet/runs/twr-lab-model-run-1122/model-run-10.json` 1,007,117 B, 2,277 scores.
- League setup: runbook `~/fleet/runs/twr-rehearsal-leagues-runbook-1122/RUNBOOK.md` (+ corrections), `DOOR.md`, scripts
  `.../setup/` (setup-root.sh, setup-commish.sh, gen-schedule.py, render-commands.py, commands/, check.sh passes offline, README.md).
- Reports to Claudebox (in ~/Downloads): TOM-REPORT-shared-facts-audit.md (T693 checked), TOM-REPORT-redraft-check.md,
  TOM-ASKS-CLAUDEBOX-rehearsal-timeline.md, TOM-ASKS-CLAUDEBOX-icp-tasks-ack.md. Claudebox brief: CLAUDEBOX-BRIEF-TOM-icp-tasks-1122.md.
- Earlier reports: `~/fleet/runs/twr-{auto-pick,autopick-fill,auto-lineup,lineup-injuries,oline-handout,rehearsal-auto-review}-1122/`.

## 7. Refuted / settled — do not retest
- "Browser door admits one league per hostname": REFUTED (browser_session.go browserAdmissions lists every league; PWA switches per request).
- Model scores and the draft pool use the same MFL ids (PinnedFootball.Players keyed by canonical MFL id): no id mapping needed.
- AFC = franchise.01-16, NFC = 17-32 in Default32 (default32Alignment). Schedule validation allows cross-side games.
- The hosted server has NO way to run a model run (desktop only) and had NO create-league: both now built as console commands.
- `commish.league.seed_held` is desktop-only on the door; create-league seeds through the normal create path.
- Native redraft does not exist (no rollover reset, contracts always on); one-season redraft needs only S pieces (see redraft report).
- Twin pools DO hold once-per-side (test + 2 planted breaks). Two leagues stay apart (two ledgers; one shared
  store not tested).
- Scores fill in by themselves for native leagues: live running totals (ESPN/Sleeper, 5 s), finals from nflverse
  >= 6 h after last kickoff, re-read every 3 h. No weekly settle job is needed.
- Draft room already runs a startup draft, one per side in a twin league. Build nothing new in the UI.
- Owner draft queue cannot carry a ranking (private per seated owner).
- appvalue (live dynasty engine) does NOT run on the hosted server; ranking uses stored `model_scores.dynasty_ppg`
  of the newest model run, then projection, then ID.
- Schedule cannot start at week 13 ("number schedule weeks consecutively from one"); use 18 weeks with weeks
  1-12 empty (works today, test committed).
- Draft clock cannot be off; "no clock on Christopher" = a very long league-wide `draft.clock` (<= 14 days via
  `draft.timer set`). `draft.onExpire=autopick` stays unbuilt (would change replay rules).
- Planted-break traps hit today: `false && A || B` still runs B; sed on a two-line signature breaks the build and
  a grep for `--- FAIL|^ok` hides "[build failed]" (always grep `^FAIL` and `build failed`).

- REFUTED 11-23 ~1 AM: "draft auto-pick is not built". It IS built on the Rehearsal A stack: e5b24bf99, b730dfd1a,
  b48d146cd (pinned facts on server), ec9050c3d (model scores). It is only missing on main. session/autopick-1122 duplicates it.
- REFUTED: runbook step 2 "seat Christopher needs a join link". `twr-server join-link -no-login -take franchise.01` (as user
  twr, server stopped) seats the league's commissioner account directly. A join link for ANOTHER account needs the
  commissioner's username+password on stdin.
- Lab status command: `pct exec 260 -- /usr/local/libexec/twr-status`. Plain `twr-server status -status-socket` is wrong (no such flag).
- S8 "league server could not be reached" = /app/session 403 "league access unavailable": its account has no seat. Not a server fault.
  The wrong message is FIXED on merge-rest (cf659e6f5): web 403 at startup -> sign-in view "This account has no team in a
  league on this server yet...". Test `nativeStartupBrowser.test.tsx` (fails without the fix).
- REFUTED 11-23: "Tom can seat a phone without Christopher". Every console door that makes a join link (join-link
  -rotate, reset-password) needs the commissioner's username+password; `-no-login` only seats the commissioner. By design;
  do not add a bypass. Phones get used once Christopher signs in on them himself (cookie persists).
- iPhone inspector: works when Chrome shows a real web page; on Chrome's own new-tab page `runtime_enable` hangs forever.
  pymobiledevice3 lives in pipx (`~/.local/share/pipx/venvs/pymobiledevice3/bin/python`), not system python.
- Dates: the REHEARSAL-SEASON doc (Claudebox's) still says drafts Nov 26–29; the LATER ruling is drafts Tue Dec 8, first
  scoring week NFL 14 (Thu Dec 10), first report card Tue Dec 15. Claudebox asked to fix its doc
  (`/root/TOM-ASKS-CLAUDEBOX-rehearsal-dates-1123.md`).
- VM 503 = the twr-lab VM (192.168.1.182) holding CT 260. After the 11-23 Proxmox reboot it came back fine: v53, 8/8 OK.
- line-budget `--write` once needed two runs (baseline file grew by a line when a new area appeared): fixed to loop until stable.
- Sign-in backup "paused" right after a Lab restart is normal for ~15 min (first backup). Not a fault.

## 8. Decisions (Christopher, 11-22)
0. TIMELINE MOVED (6:20 PM): real drafts Tue Dec 8. Nov 29 is NFL week 12 (starts Wed 11-25); a week's lineups use the roster at its first kickoff, so Thu/Fri drafts would score nothing. First scoring week = NFL week 14 (Thu Dec 10 7:15 PM CT); first rehearsal Sunday Dec 13; first report card Tue Dec 15; schedule 18 weeks with 1-13 empty (test 53a8c2b4b). Claudebox told: ~/Downloads/TOM-ASKS-CLAUDEBOX-rehearsal-timeline.md.
6. Routing: Haiku high + Sonnet medium subagents on Tom's account; Gemini only for second opinions when clarity is thin or a bug is hard.
1. Computer draft picks: TWR dynasty value, FILL STARTERS FIRST, then best available.
2. Unsold O-Line cards -> computer teams without one, at the minimum price, after the auction closes.
3. Report card runs TUESDAY ~9 AM CT (Monday-night finals land Tuesday morning). First: Tue Dec 1.
4. Lab releases need no per-release OK; the Season server only Saturday night with his OK each time.
5. Tom's engineering calls (veto-able): automatic = no seated owner (caretaker or not); healthy = not Out /
   Doubtful / IR / PUP / NFI / Suspended (Questionable starts); lineup set at lock - 60 min per kickoff wave,
   locked starters kept, a human lineup never touched; dynasty order ignores each league's own scoring this once;
   a benched player starts only to meet a minimum nobody playing can fill (O-Line card on bye).

## 9. Ledger state (11-23 ~9 AM)
main 9fcc4a771 holds everything from "merge all" (#468, #469; #467 closed as included). Remote `session/autopick-1122` deleted
(content already on main). Only open branch of this workstream: `session/no-seat-quiet-1123` (PR #470). Nothing uncommitted.

## 10. Next actions, in order (11-23 ~9 AM) — all but 1 wait on Christopher's commissioner sign-in
1. Watch CI on #470 (`gh pr checks 470`); merge only with Christopher's OK (`gh pr merge 470 --squash`), then remove
   `.worktrees/no-seat-1123` and Lab-release main as v56 if the code differs from v55.
2. With Christopher signed in as commissioner (account 564f67ef, franchise.01 in both test leagues): make join links for the
   S8 account 3da8c020 and the iPhone (never print links), seat both phones, then check v52/v53 screens at phone width.
3. When he pastes the twr1 helper line: `setup-commish.sh` per league (`~/fleet/runs/twr-rehearsal-leagues-runbook-1122/setup/`,
   twin needs `twin`); short `draft.clock`, `draft.onExpire = autopick`, open startup draft, watch auto-picks fill 31 teams
   (server.log "draft autopick"), screenshots, then delete/reset the test leagues. Rehearsal A card -> "ready for real drafts".
4. Put a read-only token at Hermes `~/fleet/twr-reportcard/token` (chmod 600); dry-run with `REPORTCARD_DATE=2026-12-15
   REPORTCARD_PLAN=/tmp/copy.json ./run.sh`.
5. T715 Lab test confirmer: NO server change. Lab-only commissioner account whose authenticator secret lives only in the Lab test
   script; script signs in, steps up, POSTs /v1/native/confirm (serverhttp/confirmation.go:76-98). Needs the account created.
6. Later: T694; laptop Season server (not before Dec 13); line-budget into CI (T709); season box real binary + fake clock (T711).

## 11. Needs Christopher
- ONE commissioner sign-in on the Lab unlocks: phone seats, the helper line for test drafts, the report-card token, T715.
- Merge OK for PR #470 (`gh pr merge 470 --squash`).
- RULED 11-23 ~8:15 AM (Christopher): (a) the MFL solo-owner tools (mflref, transactions, offers, what-if) STAY for now;
  the desktop is not retired before the rehearsal season and these tools are decided later (no delete).
  (b) YES to a test-only "confirmer" that types the G2/G3 confirmation code, on Lab test leagues ONLY; never the
  Season server or a real league.
- "merge all" (11-23) is used up: #468 + #469 merged. New PRs need a new OK.

## 12. Environment notes
- Guard hook: any command holding both `push` and a `-f`/`--force` (even `rm -f`, `worktree remove --force`) is blocked as a
  force-push; split them. Commit from inside a worktree (hook refuses when the shell cwd is the main checkout on main).
- `golangci-lint cache clean` after deleting a worktree (stale cache made the pre-commit lint fail).
- Lab CT 260 has no sqlite3/python3: `pct pull 260 /var/lib/twr/<db> <tmp>` on VM 503 and query the copy; delete copies after.
- Hermes tracker `~/fleet/tracker/data/plan.json`: find cards by title, back up first (last backup suffix
  `bak-1122aq`, next `ar`), log times from Chicago zoneinfo (Hermes is UTC), no apostrophes in the heredoc.
- Claudebox: `ssh claudebox`; prompt right after `claude -p`, add `< /dev/null`.
- New worktrees need `frontend/dist/.placeholder` (and a `frontend/node_modules` symlink for `make lint`).

## 13. Honest status
11-23: the merged code runs on the Lab (v54/v55) and the S8 screen is proven; test drafts, seats and the report card against
real league data are still unproven because they wait on the commissioner sign-in. Older note:
Code is unit-tested and combined; NOTHING of the rehearsal stack has run on a real server yet. Unproven until tonight's Lab run:
create-league on the real server, import-model-scores on the real reference.db, refresh-nfl-schedule with empty weeks 1-13,
helper-token rights for pools/draft.clock/draft.startup.open, draft.timer side args, O-Line close on 32 teams, draft speed on the
Lab CPU, the PWA league switcher. MFL injuries lag on Sunday inactives; draft room showing 600+ instant picks (T685).
ETA "ready for real drafts": Thu Dec 3 (real drafts Tue Dec 8).
