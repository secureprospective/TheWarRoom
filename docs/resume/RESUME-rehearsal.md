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

## 4. In flight (11-23 ~4:05 PM CT) — Christopher: "merge 476 when the load test passes, then keep pushing"
- **Load test v59 (running, started ~3:46 PM CT, ~26 min):** `gameday.py` 31 teams x 2 devices x 15 min against
  native.tomload ("Tom Load", 32 teams, tomlab on franchise.01, drafted). Alive: `pgrep -af gameday.py`. Output:
  `~/fleet/runs/twr-stage3-load-1123/v59/{run.log,latency.jsonl,rss.log}`; last line `PASS: 0 x 502, 0 x 5xx, p95 … ms`.
  Compare with v58 run (p95 851 ms; results.read p95 1,863; RSS 408 MB). PASS + p95 better -> merge #476
  (`gh pr merge 476 --squash`, his OK given 11-23 ~4 PM). Break latency down by op AND by 5-s bucket (bursts every 60 s).
- RSS sampler loop (~28 min) writes v59/rss.log; ends by itself.
- **Lab kit (secrets, never print):** `~/scratch/commish-1123/` (KEEP.md). New: `reader-<league>.txt` (bare read
  bearers), `join-load.txt` (Tom Test link, league full now), `join-tomload.txt` (Tom Load link), `cfg-load/`,
  `helper-native.tomload.txt`, `make-load-league.sh` (setup after create-league + take; refresh is last).
  Scripts in repo: `scripts/lab/` (TWR_LAB_KIT=<kit dir>, LAB_SSH_CONF for cycle.sh).
- Report card on Hermes: `~/fleet/twr-reportcard/token-native.tomtest-{default,twin}` (bare bearer, read-only, expire
  12-23 ~19:17 UTC: reissue with `labcommish.py reader <league>` then scp). run.sh: per-league token, LEAGUES env
  `REPORTCARD_LEAGUES`, default tomtest. Dry run 12-15: both GREEN.

## 5. Gates / status (11-23 ~4:05 PM)
| What | State |
|---|---|
| **main = 234f99a08** | merged 11-23 with his OK: #474 one join link (1ef46749e), #472 Lab kit + twr-check helper line (f1f5031fa), #475 load script (18bd5f815), #473 phone wording (234f99a08). Post-merge CI was running |
| **PR #476** `session/results-read-speed` b527e7133 | results/standings memo + one computation per burst. CI: check. Merge when v59 load test passes |
| **Lab = v59** dc72f2a81 (`session/lab-v58` = one-link + wording + Today LIVE + #476) | 8/8 OK after restart |
| v58 load test (Tom Test, 29x2, 15 min) | PASS 0x502/5xx, 22,238 req, RSS 408 MB, p95 851 ms; slow only in 60-s bursts; 409s were the script's wrong week (fixed) |
| One link proven live | 29 claims on one link (201); then 410 "Every franchise in this league already has an owner…" |
| Tom Test | full: 01 tomlab, 02 S8, 03 iPhone, 04-32 load accounts (creds gone). Football pin refreshed weeks 15-18 (both tomtest) |
| Phones | iPhone Today showed "LIVE NOW" with 0 playing on v58 -> fixed in #473 (Today) / v59; not yet re-checked on the phone |

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

## 9. Ledger state (11-23 ~4:05 PM)
main 234f99a08. Open: PR #476. This resume update: `session/resume-1123c` (worktree `.worktrees/resume-1123c`).
Worktrees to remove after #476: one-join-link, phone-wording, lab-v58, results-read-speed, stage3-load, cl-commish-1123.
Remote branch `session/resume-1123b` is stale (content merged via #472). Task list: T662 T1819 checked; T631 T640 updated.

## 10. Next actions, in order (11-23 ~4:05 PM)
1. Read the v59 load test; if PASS (0 x 502/5xx) and p95 not worse, merge #476 (his OK). Post p95/RSS on tracker + PR.
2. Re-check the iPhone Today screen on v59 (no "LIVE" with 0 playing): `~/fleet/bin/phone-chrome.py iphone eval '…innerText'`.
3. If p95 still > 200 ms: break down by op; next suspects native.directory.read (p50 452 ms, service.go:159 Location) and
   standings; consider GOMAXPROCS on the Lab CT (2) vs the Season laptop.
4. Open a PR for this resume branch (docs only).
5. Clean up merged worktrees (above) with `git worktree remove` (no --force next to push), `golangci-lint cache clean`.
6. Later: Season server setup (Saturday, his OK); T709 line-budget into CI; T711; T694.

## 11. Needs Christopher
- Nothing blocking. #476 merge pre-approved if the load test passes. Season-server changes still need his OK each time.

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
11-23 ~4:05 PM: Rehearsal A proven on the Lab; one join link built, proven live and merged; phone wording merged; report card
works (dry run). Unproven: the speed fix's effect on Lab p95 (load test running), iPhone Today wording on v59, anything on
the Season server, week-14 scoring (Thu Dec 10). Real drafts Tue Dec 8.
