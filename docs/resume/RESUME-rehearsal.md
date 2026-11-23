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

## 3. Agents + routing (Christopher, 11-22 ~6 PM — overrides older notes)
- **After the current lanes finish, Tom pushes the project himself.** Haiku subagents (Agent tool, model haiku)
  for small well-scoped jobs; **Sonnet at medium** for slightly larger ones. **Gemini weekly is thin: keep it
  for second opinions** (Argon reviews), not bulk building. Claudebox `claude -p` on a `git archive` copy
  still works for read-only surveys (launch shape in memory `haiku-runs-on-claudebox.md`).
- Briefs `~/fleet/briefs/`, runs `~/fleet/runs/<job>-1122/`. AGY: `~/fleet/bin/agy-run.sh <job> <brief> "<model>"
  <workdir>`; done = `exit` file. Quota 13:20: Gemini week 27.4% left (now less: 4 Neon runs + 1 Argon since).

## 4. In flight (11-22 ~7:45 PM CT)
- No subagents running. T695 plan DONE -> `~/Downloads/TOM-REPORT-sleeper-projections-plan.md` (8 slices; top risk: Sleeper
  projections URL unknown in code and Sleeper not in Approved_Sources; needs one supervised look + Claudebox/Christopher OK).
- Waiting on Claudebox: `~/Downloads/CLAUDEBOX-ANSWERS-TOM-environments-rerun.md` (pass -> Environments card done, `cc task check T690`);
  replies to `TOM-ASKS-CLAUDEBOX-rehearsal-timeline.md`, `TOM-ASKS-CLAUDEBOX-icp-tasks-ack.md`, `TOM-REPORT-redraft-check.md`.
- Scheduled wakeup ~8:19 PM re-checks the above and starts the Lab night when SNF is final.

## 5. Gates / status
| What | State |
|---|---|
| Argon BLOCKERs 1-5 | all fixed, each with a planted break caught |
| `session/rehearsal-auto` head 8e48bb48a (pushed) | touched pkgs ok; lint 0 issues |
| `session/lab-v52-1122` head f4e2ac276 (pushed) = origin/session/live-panel-ball + the 21 code commits of the stack (docs commits left out: v51 branch predates docs/resume/) | history, leagueservice, commissioner/..., nativecheck, rookiedraft, cmd/twr-server, serverhttp ok; lint 0 issues |
| Full draft speed, 2 cores (Beelink GOMAXPROCS=2) | 42 rounds x 32 = 1,344 picks in 3 min 6 s; late burst of 31 computer picks ~10 s |
| Lab data (CT 260 reference.db) | model_runs 0, model_scores 0 -> load via import-model-scores; weekly stats are nflverse files via fetch_log (not checked per season); MFL injuries feed fetching every 4 h (ok) |
| PR | none yet; main is now past #465 (rebase before the merge ask) |

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

## 9. Ledger state
All work above is committed on branches; nothing uncommitted. Main has #462 (docs/reset) and #463 (environments).

## 10. Next actions, in order (Christopher 11-22: push through the night)
1. After SNF final (~10:30 PM CT): Lab v50 then v51 per `docs/RESUME-gameday-1122.md` on `session/pwa-sw-shell` (COMPACT POINT 3):
   `REF=origin/session/pwa-sw-shell ~/fleet/bin/twr-lab-release-ref.sh 50 ~/fleet/runs/labrel-v49-1121/release-v49/metadata/targets.json <rundir>`,
   then 51 from `origin/session/live-panel-ball`; check phones self-update.
2. Lab v52 from `origin/session/lab-v52-1122` (same script, targets from the v51 run).
3. On the Lab CT (ssh root@192.168.1.182 -i ~/.ssh/id_ed25519 -o IdentitiesOnly=yes, then `pct` into 260): copy model-run-10.json in,
   stop server, `twr-server import-model-scores -db /var/lib/twr/server.db -file model-run-10.json`, start.
4. Create TEST leagues native.rehtest-default and native.rehtest-twin with `setup/setup-root.sh` + `setup-commish.sh`; seat Christopher's
   account on franchise.01 each (join-link); short draft clock for the test; run both startup drafts; screenshot draft room + league switcher
   on Claude-OS; note timings. Fix anything UNPROVEN that fails (runbook lists them).
5. Delete the test leagues (`twr-server delete-league`), tracker card Rehearsal A -> needs_you "ready for real drafts Dec 8".
6. Then: T694 redraft S pieces only after Claudebox confirms one-season; T705 native power-rank input;
   Argon SHOULDs 9/10; laptop Season server setup (not before Dec 13).

## 11. Needs Christopher
- Merge OK for the v51 stack (pwa-sw-shell + t677-t679 + live-panel-ball) after tonight's Lab check.
- Later: merge OK for `session/rehearsal-auto` (rebased on main) after the Lab test drafts; real drafts Tue Dec 8.

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
Code is unit-tested and combined; NOTHING of the rehearsal stack has run on a real server yet. Unproven until tonight's Lab run:
create-league on the real server, import-model-scores on the real reference.db, refresh-nfl-schedule with empty weeks 1-13,
helper-token rights for pools/draft.clock/draft.startup.open, draft.timer side args, O-Line close on 32 teams, draft speed on the
Lab CPU, the PWA league switcher. MFL injuries lag on Sunday inactives; draft room showing 600+ instant picks (T685).
ETA "ready for real drafts": Thu Dec 3 (real drafts Tue Dec 8).
