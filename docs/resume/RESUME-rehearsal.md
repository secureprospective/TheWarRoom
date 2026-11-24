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

## 4. In flight (11-23 ~6:20 PM CT) — Christopher: "merge 479 480 481 482 when green"; after compaction: brief him on ETAs for everything on the plan
- **Merge chain (background shell, survives compaction):** #480 MERGED, #481 MERGED, #482 waiting on CI then auto-merge
  (`gh pr view 482 -R secureprospective/TheWarRoom --json state`). If not merged, `gh pr checks 482` and merge when green.
- **#479 line limit (his OK given) — do LAST, by hand:** its baseline predates #480/#481/#482, so main would fail the
  budget. In `.worktrees/line-budget-ci`: `git merge origin/main` (no rebase: force-push is blocked), run
  `scripts/check-line-budget.sh --write`, commit, push, wait CI green, `gh pr merge 479 --squash`. Check main CI after.
- **Gemini lanes (agy-run.sh, started ~6:05 PM; done = `<run>/exit`):**
  - `~/fleet/runs/twr-nativegen-off-desktop-1123/` (Neon High, worktree `.worktrees/nativegen-off-desktop`): command-table
    generator off `internal/composition` (first step of desktop removal). Review diff + `go run ./frontend/scripts/nativegen -check`.
  - `~/fleet/runs/twr-season-box-live-1123/` (Neon Medium, worktree `.worktrees/season-box-live`, based on old
    session/season-box-binary = #480): live-frame replay seam in the binary season box. Its branch must merge origin/main.
  - Alive: `pgrep -af agy`; output `out.log`, `report.md`/`REPORT.md`. Lanes push branches; Tom opens PRs (merge = his OK).
- **MNF phone check (T685):** background timer fires 7:35 PM CT. Then: S8/iPhone on the Lab during the Monday game, plays
  should land within 1-2 s (`~/fleet/bin/phone-chrome.py <s8|iphone> eval ...`). Wake the S8 first.
- **Lab auto-release (T663, permanent, his task):** Hermes `twr-lab-autorelease.timer` every 10 min ->
  `~/fleet/bin/twr-lab-autorelease.sh` on the Beelink; state `~/fleet/runs/twr-lab-autorelease/state`; log
  `~/logs/TheWarRoom/lab-autorelease.log`. After the merges it should release v61 by itself: CHECK the log + twr-status
  (first fully automatic run; unproven). Stop: `ssh hermes systemctl --user disable --now twr-lab-autorelease.timer`.
- Lab kit (secrets, never print): `~/scratch/commish-1123/` (KEEP.md). Report-card tokens on Hermes expire 12-23.

## 5. Gates / status (11-23 ~6:20 PM)
| Item | State |
|---|---|
| Stage 3 load (zero 502 at 32-team load) | MET twice: v58 22,238 req p95 851 ms; v59 24,430 req, all 200, p95 393 ms, results 536, RSS 535 MB |
| p95 < 200 ms | NOT met on Lab 2 CPUs; sign-in directory read likely measurement (connect in timer), #482 fixes the script |
| Stage 4 assumptions | DONE, #478 merged with his rulings (docs/reset/ASSUMPTIONS.md) |
| T663 Lab auto-release | Built; v60 = bbe4f298f by it; fully automatic path unproven until next merge |
| Merged today PM | #476 speed, #477 resume, #478 Stage 4, #480 season box binary, #481 one-season leagues |
| Open | #482 (auto-merge on green), #479 (re-baseline then merge) |

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

- 11-23 PM: T662 (lineup.read 409 "pinned football evidence does not hold NFL week 15") = load-script bug: it used
  week.read's nextWeek; week.read answers {current:{week}, nextWeek}. App reads current.week. Settled, fixed in #475.
- 11-23 PM: Lab slowness is NOT per-read CPU (results.read ~3 ms on the fixture): it is the 60-s burst where every device
  re-reads at once (300-400 req in 5-10 s on 2 CPUs). Fix = one computation per burst (#476).
- 11-23 PM: report-card 403 = token lacked native.directory.read; twr-check now accepts the twr1 line too.

- 11-23 PM: v59 directory read 1.35 s p50 = first request per device; timer included TCP+TLS connect (store cost 0.52 ms,
  Location check 0.064 ms). Not a server bug (to confirm with #482 on next run).
- 11-23 PM: #476 first CI fail = TestResultsWarmReadsPinnedLeaguesAndStops expected a starter-cache hit the memo pre-empts;
  test relaxed to "no new starter miss" (not a product bug).

## 8. Decisions (Christopher, 11-22)
- 11-23 evening (Stage 4, #478): remove the desktop program; rebuild MFL helper tools small as a reference against mistakes;
  one-season leagues as settings ("sandbox, no new app structure, open the possibilities"); many templates, each only
  fills in settings; projections = projected STATS scored by each league's own rules; desktop-era ideas retired;
  T537/T654/T659 closed. Lean into Gemini (Neon/Hydrogen); Argon sparingly (heavy token eater).
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

## 9. Ledger state (11-23 ~6:20 PM)
main = #481 merged (plus earlier #476-#478, #480). Resume update: `session/resume-1123d` (worktree `.worktrees/resume-1123d`).
Worktrees still in use: line-budget-ci (#479), loadtest-warm (#482), nativegen-off-desktop, season-box-live (Gemini), resume-1123d.
Remove after merge, then `golangci-lint cache clean`.

## 10. Next actions, in order (11-23 ~6:20 PM)
1. FIRST after compaction: brief Christopher on ETAs for everything on the plan (his ask; plain words; sections 13 + BUILD-PLAN).
2. Finish the merge chain (#482, then #479 re-baseline). Post merges on the tracker (Hermes plan.json, back up first,
   next suffix `bak-1123n`).
3. Check the Lab auto-released the new main (v61) and Everything is working; post on tracker.
4. 7:35 PM: T685 phone check during MNF.
5. Read the Gemini lane reports; review each diff yourself (plant one break); open PRs; ask his merge OK.
6. Lab click-through: create a 12-Team One Season league on the Lab, check the template values on the phone.
7. Next builds (Gemini first, Argon only for risky reviews): desktop removal (T710; Claudebox keep-list review first, Argon
   review of the deletion), MFL helper tools rebuilt small (plain-word spec first), season box draft/recap gaps (B7, B2).
8. Saturday 11-28 night with his OK: Season server setup (laptop) — hand him exact commands.

## 11. Needs Christopher
- Nothing blocking right now; #479-#482 merge OK given. Future PRs: fresh OK each.
- Saturday: Season server (laptop) setup, his OK each command.
- Open product question, ask only when projections are next: where projected stat lines come from.

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

- Pre-push needs `frontend/node_modules` in each worktree: `cd frontend && pnpm install --frozen-lockfile` (else
  "Cannot find module ./frontend/node_modules/typescript"). Go builds need `frontend/dist/.keep`.
- twr-server console commands that exit non-zero under `set -e` leave the Lab server STOPPED: always start it again.
- Read tokens need `native.directory.read` too (every hosted read checks the directory first; else 403).

## 13. Honest status
11-23 ~6:20 PM. Proven: Stage 3 (two load passes, zero errors), Stage 4 rulings recorded, Lab self-release (one hand-finished
run), one-season templates (tests + planted break; not clicked on the Lab yet), whole-season test on the real binary
(no draft/live/recap yet). Unproven: fully automatic Lab release, phone live push latency (T685), p95 under 200 ms,
anything on the Season server. Timeline: Season server Sat 11-28 (his OK) -> Dec 6 wall/recording session -> real drafts
Tue Dec 8 -> week 14 Thu Dec 10 -> Rehearsal Sunday 1 Dec 13 -> first report card Tue Dec 15.
Christopher was concerned 11-23 evening by off-lane questions (resume chatbot etc. from a raw task-list sweep): ask only
current TheWarRoom product choices, plain words, click-through (memory questions-stay-in-lane).

