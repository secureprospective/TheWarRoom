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
- [x] Lab data check: model scores imported (run 1, 2,277); test drafts drew from them (11-23).
- [x] Both leagues created on the Lab, one TEST draft each, clean (11-23, Tom Test + Tom Test Twin, v57); card "ready for real drafts".
- [x] S8 + iPhone seated via join links and signed in (11-23).
- [ ] One join link for the whole league (ruled 11-23 ~1 PM), built + on the Lab.

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

## 4. In flight (11-23 ~1 PM CT) — Christopher: "keep driving after compaction"
- **Nothing running in the background.** No agents, watchers or releases. adb server stays up for the phones.
- **Lab kit (secrets, never print):** `~/scratch/commish-1123/` (KEEP.md): `lab-commish.json` (tomlab account), `totp.seed`,
  `phone-s8.json` / `phone-iphone.json` (s8lab / iphonelab), `codes-*.txt` (recovery codes), `helper-native.tomtest-*.txt` (twr1
  lines, expire 11-24 ~17:27 UTC), `cfg/` + `cfg-twin/` (twr connections), `bin/twr`, `server.url`, rendered setup in
  `rehearsal-tomtest-*`. Scripts (copies without secrets in `~/fleet/runs/twr-testdrafts-1123/`):
  - `labcommish.py enroll|helper <league>|confirm <twrc1-file>|revoke <league>` (User-Agent `twr-lab-script/1`; default
    Python UA gets Cloudflare 1010).
  - `apply.sh <change.json>...` preview -> confirm via labcommish -> execute (set XDG_CONFIG_HOME=cfg or cfg-twin).
  - `cycle.sh` deletes + recreates both Tom Test leagues, helpers, setup, opens both drafts (10 s clock).
  - `cdp_join.py` (S8: new Chrome tab via devtools `adb forward tcp:9290 localabstract:chrome_devtools_remote`),
    `cdp_app.py` (sign in through the installed app window), `seat_phone.py` (iPhone via phone-chrome.py).
- Report card job on Hermes still has no token (`~/fleet/twr-reportcard/token`); it now can be made from tomlab.

## 5. Gates / status (11-23 ~1 PM)
| What | State |
|---|---|
| **main = 1d3a368b3** (#471) | create-league -commissioner + autopick batches of 32 per worker pass. #470 merged d2bfb7ab. CI all green |
| **Lab = v57** 341ad1189 (= main code) | 8/8 OK after restart (sign-in backup pause ~1 min). Test leagues native.tomtest-default ("Tom Test") + native.tomtest-twin, commissioner tomlab (acct 13d4c223) at franchise.01 |
| Test drafts on v57 | both 43-round startup drafts opened 17:31 UTC, ran together without stalls; twin AFC+NFC closed 11:49 AM CT, Tom Test closed ~12:05 PM CT |
| Phones | S8 = s8lab, Franchise 2 in Tom Test; iPhone = iphonelab, Franchise 3. Both signed in, Today shows week 14 (screens in ~/fleet/runs/twr-testdrafts-1123/) |
| Tracker on Hermes | lanes/order/flags refreshed 11-23; collector timer fixed (OnStartupSec, tracker repo 5718ba3) |
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
- REFUTED 11-23: "a Lab sign-in reset makes Tom commissioner". Leagues name their commissioner by account ID; an empty
  accounts file makes nobody commissioner. The fix was create-league -commissioner (#471).
- REFUTED 11-23: "draft.onExpire=autopick". The engine refuses it ("not built yet; choose skip"); empty seats auto-pick
  regardless; a seated owner's expired clock is skipped (replay, no ledger record). Keep onExpire = skip.
- 11-23 FOUND+FIXED: one league's long autopick run held the single due worker (Tom Test froze 7 min at round 13 while the
  twin's all-empty NFC side drafted). Fixed by batches of 32 per pass (#471). Expected 409 "league version ... reread" on a
  draft.timer change while picks land: not a bug.
- 11-23: S8 page loads time out when the phone screen is off (dozing): wake it (`input keyevent KEYCODE_WAKEUP`, swipe,
  `svc power stayon true`, turn off after). phone-chrome `goto` on S8 attaches to installed-app windows; use a new tab.
- 11-23: join links are SINGLE USE today (claim consumes the code); `join-link -rotate` needs the server stopped and
  commissioner creds on stdin. iPhone: a goto that only changes the #hash does not reload: call location.reload().

## 8. Decisions (Christopher, 11-22)
- 11-23 ~11 AM: Lab guards loosened: the Lab no longer needs commissioner/password guards; the Season server (laptop) keeps every guard. Tom may own Lab commissioner accounts. (memory `lab-guards-loosened.md`)
- 11-23 ~1 PM: ONE join link for the whole league (not one per owner). Any owner can use it until seats run out; commissioner can still revoke/replace it.
- 11-23: merged #470 (OK "merge it") and #471 (OK "merge 471 when CI is green").
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

## 9. Ledger state (11-23 ~1 PM)
main 1d3a368b3 (#471). Branches of this workstream merged and deleted locally; this resume update is on
`session/resume-1123b` (worktree `.worktrees/cl-commish-1123`). Tracker repo on Hermes 5718ba3 (timer). Task list 14aa12a.

## 10. Next actions, in order (11-23 ~1 PM)
1. Build ONE join link per league (ruling 11-23): the link stays valid for every open seat until the commissioner revokes or
   replaces it; each claim still takes one open franchise; a full league says so plainly. Find the single-use check
   (`joinLinkClaimed`, internal/leagueservice/join_seats.go ~111; accounts/join.go claim ~262-300), test first (two claims on
   one link both succeed; revoke still kills it), PR, Lab release, prove with S8+iPhone or the API. Merge needs his OK.
2. Small phone wording fixes (same or separate PR): join done page "The Franchise 2 are yours" (use "Franchise 2 is yours" /
   team name), "Open The War Room app on your computer" -> phone-neutral; sign-in screen double "Sign in" + "HttpOnly cookie"
   jargon; "LIVE" badge with 0 playing.
3. Report-card token: issue a read-only token from tomlab (labcommish-style, modes ["read"]) for the Tom Test leagues, point
   `~/fleet/twr-reportcard/run.sh` LEAGUES at native.tomtest-*, put token at Hermes `~/fleet/twr-reportcard/token` (600),
   dry-run with REPORTCARD_DATE=2026-12-15 on a plan.json copy.
4. File the Lab kit properly (T715 home: fleet runs dir or repo scripts/lab/), keep secrets in scratch.
5. Stage 3: 32-team load test (scripts/load/gameday.py on branch session/stage3-load).
6. Later: Season server setup (Saturday, his OK); T709 line-budget into CI; T711; T694.

## 11. Needs Christopher
- Nothing blocking. Merges to main and Season-server changes still need his OK each time.
- Earlier rulings: 11-23 ~8:15 AM (a) MFL solo-owner tools STAY for now; (b) YES Lab-only test confirmer (done, T715 checked).

## 12. Environment notes
- Guard hook: any command holding both `push` and a `-f`/`--force` (even `rm -f`, `worktree remove --force`) is blocked as a
  force-push; split them. Commit from inside a worktree (hook refuses when the shell cwd is the main checkout on main).
- `golangci-lint cache clean` after deleting a worktree (stale cache made the pre-commit lint fail).
- Lab CT 260 has no sqlite3/python3: `pct pull 260 /var/lib/twr/<db> <tmp>` on VM 503 and query the copy; delete copies after.
- Hermes tracker `~/fleet/tracker/data/plan.json`: find cards by title, back up first (last backup suffix
  `bak-1123d`, next `1123e`), log times from Chicago zoneinfo (Hermes is UTC), no apostrophes in the heredoc.
- Claudebox: `ssh claudebox`; prompt right after `claude -p`, add `< /dev/null`.
- Hook blocks commit messages containing "reboot" (power-state guard): reword.
- Hermes git commits: run ssh from `cd /tmp` (repo hook refuses when cwd is the main checkout).
- New worktrees need `frontend/dist/.placeholder` (and a `frontend/node_modules` symlink for `make lint`).

## 13. Honest status
11-23 ~1 PM: Rehearsal A is proven on the Lab: two leagues, full startup drafts, autopick, scripted commissioner confirms,
two phones seated and signed in. Not yet proven: one-link-per-league (not built), the report card against real data (no
token yet), anything on the Season server (not set up), lineups/scoring for week 14 (starts Thu Dec 10). Real drafts Tue Dec 8.

