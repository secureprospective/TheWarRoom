# Resume — Rehearsal season (from Sun 11-22, Tom on the Beelink)

## 1. What we are doing
Six NFL weeks left are a dress rehearsal: two NATIVE 32-team leagues run entirely by TWR (the 32-Team
Default, and a twin-pool league = the same default with `players.pools = per_side`, AFC 16 / NFC 16). Christopher
owns one team in each; the other 62 are automatic. Plan: `docs/reset/REHEARSAL-SEASON-1122.md` (LIVE, Claudebox
edits it). Environments (Dev / Lab CT 260 / Season server 192.168.1.162): CLAUDE.md "Environments".
Repo `~/work/TheWarRoom`; working branch **`session/rehearsal-auto`** (worktree `.worktrees/rehearsal-auto`).

## 2. Done looks like (Rehearsal A, due Thu 11-26)
- [x] Gaps posted on the tracker before building (inventory `~/fleet/runs/twr-rehearsal-a-1122/`).
- [x] Auto-pick, auto-lineup, O-Line handout built, each checked with a planted break.
- [ ] Argon review of the stack addressed (running, see 4).
- [ ] Lab data check: `model_runs`/`model_scores` rows and weekly stats exist for the rehearsal leagues.
- [ ] Both leagues created on the Lab, one TEST draft each, clean; card to needs_you "ready for real drafts".

## 3. Agents + routing (Christopher, 11-22 ~6 PM — overrides older notes)
- **After the current lanes finish, Tom pushes the project himself.** Haiku subagents (Agent tool, model haiku)
  for small well-scoped jobs; **Sonnet at medium** for slightly larger ones. **Gemini weekly is thin: keep it
  for second opinions** (Argon reviews), not bulk building. Claudebox `claude -p` on a `git archive` copy
  still works for read-only surveys (launch shape in memory `haiku-runs-on-claudebox.md`).
- Briefs `~/fleet/briefs/`, runs `~/fleet/runs/<job>-1122/`. AGY: `~/fleet/bin/agy-run.sh <job> <brief> "<model>"
  <workdir>`; done = `exit` file. Quota 13:20: Gemini week 27.4% left (now less: 4 Neon runs + 1 Argon since).

## 4. In flight at compaction
- **Argon (High) review** of `session/rehearsal-auto`, started 5:47 PM, pid 2066167. Alive: `ps -p 2066167`.
  Output `~/fleet/runs/twr-rehearsal-auto-review-1122/REVIEW.md`; done = `exit` file there. Brief
  `~/fleet/briefs/twr-rehearsal-auto-review-1122.md`. On result: verify each BLOCKER yourself, fix on the branch
  with a test that fails without it, re-run lint + touched tests.
- **Claudebox cold-start re-run** for the environments directive (asked 5:23 PM in
  `~/Downloads/TOM-ASKS-CLAUDEBOX-environments-rerun.md`); answer will be
  `~/Downloads/CLAUDEBOX-ANSWERS-TOM-environments-rerun.md`. Pass = "the Lab, never the Season server" ->
  tracker card "Environments" done, task T690 checked.
- Beelink Vite 557753 (Christopher's wall) is up on purpose; leave it.

## 5. Gates / status
| What | State |
|---|---|
| Stack lint (`make lint`) | 0 issues (5:45 PM) |
| `go test` history, leagueservice, commissioner/..., nativecheck, rookiedraft, cmd/twr-server | all ok (5:45 PM) |
| Full 32-team auto-pick draft test | opt-in `TWR_SLOW_DRAFT=1` (~55 s); passed 18 rounds, all 31 teams legal |
| Stack base | 513b7058a (old main). Main is now 0e33a8182: rebase before the PR |
| PR | none yet; one merge ask after Argon + Lab test drafts |

## 6. Artifacts
- Branch `session/rehearsal-auto` head e0dbb4475 (11 commits on 513b7058a; superset of `session/rehearsal-a`
  98a94bcff's tests, `session/auto-pick` 76bb9fa06+c51403cc9, `session/auto-lineup` 1a4b28343,
  `session/oline-handout` 642878130). The four source branches can be deleted after the stack merges.
- Reports: `~/fleet/runs/twr-{auto-pick,autopick-fill,auto-lineup,lineup-injuries,oline-handout}-1122/REPORT.md`;
  survey files `~/fleet/runs/twr-rehearsal-a-1122/{rehearsal-inventory,native-stats-trace,twinpool-proof,
  twin32-proof,autoplay-facts}.md`; Claudebox `/root/twr-dynasty-hosted-1122.md`,
  `/root/twr-schedule-weeks-proof-1122.md`.
- Season swap script (merged, #463): `deploy/season/season-swap.sh`, tests `deploy/season/test_season_swap.py`.

## 7. Refuted / settled — do not retest
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

## 10. Next actions, in order
1. Read Argon's REVIEW.md; verify and fix BLOCKERs (Sonnet-medium subagent for bigger fixes, Haiku for small).
2. After Sunday Night Football (~10:30 PM): Lab v50 then v51 — steps in `docs/RESUME-gameday-1122.md` on branch
   `session/pwa-sw-shell` (COMPACT POINT 3 section). Command:
   `REF=origin/session/pwa-sw-shell ~/fleet/bin/twr-lab-release-ref.sh 50
   ~/fleet/runs/labrel-v49-1121/release-v49/metadata/targets.json <rundir>`; then v51 from
   `origin/session/live-panel-ball`; phones update themselves; then one merge ask for that stack.
3. Lab data check: newest `model_runs` row and `model_scores` count by position in the Lab reference file;
   weekly stats (`stats_player_week`) held; injury feed answering.
4. Rebase `session/rehearsal-auto` on main, release it to the Lab, create both leagues (Default32 32 teams; twin =
   Default32 + `players.pools="per_side"`; schedule 18 weeks with 1-12 empty; long `draft.clock`), seat
   Christopher on one team each, run one TEST startup draft per league, screenshot; card -> needs_you.
5. Laptop Season server setup (after the releases; not needed before Nov 29): versioned install, OpenRC service,
   firewall, nightly backup to CT105, Hermes watch. Handoff `~/Downloads/CLAUDEBOX-ANSWERS-TOM-laptop-season-server.md`.
6. Report card checker on Hermes (T688) after Rehearsal A.

## 11. Needs Christopher
- Merge OK for the v51 stack (pwa-sw-shell + t677-t679 + live-panel-ball) after the Lab check tonight.
- Later: merge OK for `session/rehearsal-auto`; then real drafts Thu/Fri.

## 12. Environment notes
- Hermes tracker `~/fleet/tracker/data/plan.json`: find cards by title, back up first (last backup suffix
  `bak-1122aq`, next `ar`), log times from Chicago zoneinfo (Hermes is UTC), no apostrophes in the heredoc.
- Claudebox: `ssh claudebox`; prompt right after `claude -p`, add `< /dev/null`.
- New worktrees need `frontend/dist/.placeholder` (and a `frontend/node_modules` symlink for `make lint`).

## 13. Honest status
Code is built and unit-tested, never run on a real server. Unproven: the Lab holds model scores and weekly stats
(else picks fall back to projection and lineups to ID order); draft speed on the 2-core Lab across 43 rounds;
MFL injuries lag on Sunday inactives; how the draft room shows 600+ instant picks (browser change push is a no-op,
T685). ETA for "ready for real drafts": Wed 11-25 if Argon finds no blocker.
