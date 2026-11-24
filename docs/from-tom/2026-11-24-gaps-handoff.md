# Handoff: closing the 11-22 grill gaps (11-24, end of day)
**Date:** 2026-11-24 (written ~4:15 PM CT)
**Branch:** `session/resume-1124` (this note); build work on `session/pulse-<lane>` branches
**Status:** ready to resume. Merges #506–#508 are in flight with your OK; #509 waits for your OK.
**Read with:** `docs/resume/RESUME-gaps.md` (same branch: routing, refuted ideas, decisions, environment notes),
`docs/reset/GAPS-1124.md` (the gap list), `docs/reset/BUILD-PLAN.md` (stages), run log `~/fleet/runs/gaps-1124/REPORT.md`.

---

## 1. Where the project is (plain words)

TheWarRoom runs two rehearsal leagues on the **Lab**. The real drafts are **Tue Dec 1**, and the Lab hosts them
("Lab now, laptop later", 11-24). Everything built today serves the **GATE**, the four parts that must pass before any
outside league joins:

| GATE part | What it means | State tonight |
|---|---|---|
| 1. Rehearsal clean | Clean through week 18; playoffs are proven by the automatic season test (ruled 11-24: no rehearsal playoffs) | **Season test now plays a playoff round (#498, merged).** The rest happens as the season runs. |
| 2. Every league shape runs | 10/12/16-team one-season leagues running; redraft season 2 | Settings exist (#481). Not running yet (week of Dec 7). Redraft season 2 is after the season. |
| 3. One copy of every NFL fact + load test | Schedule, official stats, football evidence, decoded copies shared once per server | **3 of 4 done:** #1 schedule timer (#500), #2 official stats built once per version (#505), #4 decoded-copy cache (#496). **#3 (split the football evidence body) is still open (L).** The 32-team load rerun comes after #3. |
| 4. History in and out | MFL + ESPN history import, full export | **Nothing built.** Biggest open build (December). |

### Merged today (all with your OK)
#494 desktop step 1, #495 canary runner, #496 decoded-copy cache, #497 Sleeper step 1, #498 playoff season test,
#499 MFL bid check stays local (server twin), #500 NFL schedule refreshes itself, #501 MFL money in cents,
#502 no second server on one database, #503 preview/undo report, #504 desktop step 3 (CI), #505 official stats
once per football version, #506 Sleeper accuracy report (3:40 PM).

### In flight right now
| What | State | How to check |
|---|---|---|
| Merge job `506 507 508` | #506 merged 3:40 PM. #507 brought up to date 3:45 PM and waiting on checks; #508 after it | `tail ~/fleet/runs/gaps-1124/merge/merge.log`; done = `merge/exit` |
| **#509 desktop step 4 (PR-D)** | Pushed, full pre-push passed. **Needs your merge OK, after #507** | `gh pr view 509` |
| Lab | v80 = 57548f51d, "Everything is working" 2:39 PM; holds the new one-copy lock. Auto-release every 10 min after each merge | `tail ~/logs/TheWarRoom/lab-autorelease.log` |
| Luna lanes | **none running** | `run-watch.sh status ~/fleet/runs/gaps-1124` (merge3–6, merge499 and labrel-184ebf18e rows are stale leftovers; ignore) |

### What each new PR does for you
- **#506 Sleeper accuracy:** `twr-server sleeper-accuracy -db … -league … -season 2026 -week N` shows how close
  Sleeper's forecasts land to what players scored under the league's own rules (mean miss, high or low, 10 biggest misses).
  **Still to do:** run it on the Lab once week 12 is played, before week 13 (Thu Dec 3). This closes T1834.
- **#507 Import shows what differs:** a hosted MFL import now prints the comparison against MFL (summary, league-wide
  differences, each team that does not match). Before, the server ran the comparison and threw it away; the desktop
  assistant was the only place it was shown. Found by the PR-D coverage check.
- **#508 Stop the server first:** the 12 console commands marked "server stopped" refuse while the server runs. They
  check before asking for a secret. Join links, seat links, password resets and status still work alongside the server.
- **#509 Desktop step 4:** removes the desktop root, its icons/installers, `internal/composition` and `tools/fixtures`
  (8,241 lines). `make build` now builds the server and CLI. Nothing an owner sees changes.

---

## 2. Working with Luna (GPT `gpt-6-luna`, high) — what today taught

**Score today: 10 lanes. 2 done right the first time, 1 done right with only the commit failing, 7 needed Tom to
finish.** Luna never hung; it quits early and reports more than it did. GPT's update made the quitting worse: it now
stops after about one slice of work, often within 3 minutes.

### The patterns (each seen today)
1. **Quits after one slice.** It writes the main code and stops before the tests (sleeperacc twice, offlinelock).
   A "Part 2" relaunch helps about half the time; the second relaunch rarely does.
2. **Skips the tests, not the code.** The code it writes is usually close to right. What is missing is the test, the
   planted break and the commit. Plan for Tom writing tests.
3. **Over-reports.** It has said "tests pass" when they were broken and "no twin exists" when one did. Earlier today
   it committed REPORT.md and HANDOFF.md into the repo. Treat every claim as unverified.
4. **Real bugs hide in small details:** a literal `\n` in output, rounding that is wrong for negatives, the
   10 misses never printed, a test that ran the lock with an empty path and left a `.lock` file in the source folder.
   Read the diff, not the report.
5. **Stops on any guard.** heavy-ok.sh, a busy lint ("parallel golangci-lint is running"), a commit hook timeout,
   or its sandbox refusing `git rm -r` (PR-D) all end the lane. Briefs that say "busy lint = wait 60 s and retry,
   not a stop" helped. Its sandbox cannot delete folders, so deletion work is Tom's.
6. **Good at read-only research with a fixed table to fill.** The migration-coverage check (migcover) finished in
   3 minutes with a correct, useful verdict (it found a real gap). Research briefs that name the files and the output
   table work best.
7. **Works on the wrong base if told loosely.** Always name the base branch (`BASE=` on the launcher).

### How to brief Luna so it finishes
- **One outcome, under 40 lines.** Name the exact function, file and line to change, plus the existing test fixture
  to reuse (`grep` the name for it). It does not explore well.
- **Pre-make every decision:** what not to touch, error wording, where the code goes ("no new tiny file": the
  bloat ratchet fails on small files).
- **Put the order of done lines in the brief:** code, then test, then planted break, then lint, then commit. It
  tends to do the first and stop.
- **Say what is not a stop:** busy lint, a hook timeout, single-package tests not needing heavy-ok.sh.
- **Make the code testable on its own.** Ask for a small pure function that the store or command calls; that is
  how Tom finished sleeperacc in minutes.
- **The rule that worked:** after 2 misses, Tom finishes. Finishing usually takes 10–20 minutes: test, planted
  break, lint, commit, push, PR.

### Routing that held today
- Luna high: wiring, single-package changes, read-only research with a table.
- Gemini Neon High: you said "save it for multi file only". Not used today. PR-D was the right size for it, but
  Luna's sandbox block meant Tom did it directly.
- Tom: tests, mutants, multi-file deletions, anything needing a judgment across scripts and config.

---

## 3. Where we are going (order of the remaining build)

Order follows `docs/reset/GAPS-1124.md` §9, adjusted for today. Each item: one PR, CI green, Tom review, one
planted break caught, your merge OK.

### Now → Sat 11-28
1. **Merge #507, #508** (running), then **#509** with your OK.
2. **Desktop step 5 (PR-E):** remove `internal/app/{core,platform,control,commissioner,draft,market,race,research,offers,mflref}`
   and their test helpers. Keep `app/{now,report,season,sources}`. Large; Gemini Neon or Tom, after #509.
3. **Canary on Hermes (T708):** the runner exists (#495). Install it on Hermes against a Lab canary league, seed
   logged, green/red on the tracker. Tom ops (Hermes, not a code lane).
4. **Owner chat check (A6):** does owner chat exist and work on the Lab? Read-only check; if it is missing it goes on
   the gap list as a build. **Not started.**
5. **Product call (from #503):** should the seven commissioner draft-room commands get an undo rule before an
   outside league? Your call; no build until you rule.

### Before Tue Dec 1 (real drafts on the Lab)
6. Nothing else is required (no rehearsal playoffs, 11-24). Lab auto-release holds on Dec 1 and during game windows.
7. **Run `sleeper-accuracy` on the Lab** for week 12 once its games are played, and post the result (T1834).

### Week of Nov 30
8. **Wall on real live screens (T689)** for the Sat Dec 5 swap.

### December
9. **GATE part 4, history in and out:** plain-words spec ("what the commissioner sees"), then the ESPN access check
   (read-only, on one of your leagues), then the build (champions, records, head-to-head, old rosters), and check
   the full export is complete. Two or three lanes.
10. **GATE part 3 #3:** split the football evidence body (shared season part plus a small league part, L), then the
    **32-team load rerun** on that design (Stage 3 T640/T631).
11. **GATE part 2:** 10/12/16-team one-season leagues on the Lab with computer owners (week of Dec 7).
12. **Rollback drill** at a Saturday swap; **token renewal by Dec 22**.
13. **Desktop steps 6–9 (PR-F to PR-I):** PR-F needs your ruling (solo-owner leaves, doc updates; "trim, never
    delete" the solo-owner tools). PR-G removes desktop code in kept packages; PR-H removes `frontend/wailsjs`,
    `wails.json` and the Wails dependency; PR-I cleans up the Makefile, scripts and docs.
14. **Phone study and mocks after Dec 6 (T700); phone alerts design** (no push alerts exist today).
15. Laptop Season server installer (local firewall, Season signing key, same binary); the leagues move on a
    Saturday with your OK.

### January and after the season
16. Jan 2: tiny-model and harness research rerun (T707). The campaign waits for your phone OK.
17. BUILD-PLAN Stages 4–6 (old-assumptions map, de-slop by area, lock-in checks), redraft season 2, CLI/AI-first,
    wizards, then the Stage 7 backlog.

---

## 4. Needs Christopher
- **Merge OK on #509** (desktop step 4), after #507 lands.
- Product call: an undo rule for the seven commissioner draft-room commands (from #503)?
- Desktop PR-F needs your ruling when we get there.

## 5. Verification for today's work
Every merged PR: CI green, plus a planted break that its test caught (each is named in the PR body). Lab v80 status
"Everything is working" at 2:39 PM. #509: `go build ./... && go vet ./...`, `make build`, tools, nativegen and
commandcatalog tests, `nativegen -check`, `make lint` 0 issues, and the full pre-push gate all passed.

## 6. Open items for the next session
- After #507/#508 merge: confirm the Lab auto-release and twr-status, then add a tracker log line.
- Clean up finished worktrees (about 60 `pulse-*` worktrees; most are merged), then `golangci-lint cache clean`.
- The parked seasonsetup kit (`session/pulse-seasonsetup` 33c5a001a, not pushed) stays parked until the laptop work.
- `tools/twr-practice/README.md` still describes desktop click tests in places (PR-I cleans it).
