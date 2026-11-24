# Resume — closing the 11-22 grill gaps (Tom, Beelink head brain) — OPEN
Updated 2026-11-24 ~7:45 AM CT. Gap list: `docs/reset/GAPS-1124.md` (same branch). Run folder (create on first dispatch): `~/fleet/runs/gaps-1124/`.

## 1. What we are doing
We are closing the gaps between Christopher's 11-22 grill rulings and the plan, in order, so that the GATE
(4 parts, before any outside league) gets met. We build with lanes and review every line ourselves.
- Repo: `~/work/TheWarRoom`. Branch for docs: `session/resume-1124` (worktree `.worktrees/resume-1124`). Each build lane
  gets its own `session/pulse-<lane>` branch.
- Lab: `ssh -F ~/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab`, then `pct exec 260 -- …`.
  If that worktree is gone, look in `~/scratch/commish-1123/` (the Lab kit, with KEEP.md; secrets, never print).

## 2. Done looks like (this window)
1. NOT MET: the canary runner (T708) runs on the Lab with a logged seed, a red result can be replayed from its
   seed, and green/red shows on the tracker.
2. NOT MET: the season-in-a-box test plays one playoff round (Christopher's 11-24 ruling: no rehearsal playoffs,
   the test proves them).
3. NOT MET: shared-facts leftover #4 is in: one decoded PinnedFootball per digest (cache in `loadPin`), with a
   test showing two leagues share it, and Lab memory and p95 compared before and after.
4. NOT MET: Sleeper projections step 1 (T695). Sleeper is added to Approved_Sources (internal rehearsal use
   only), the address is checked, slices are built, and accuracy logging starts by week 13 (Thu Dec 3).
5. NOT MET: desktop removal (T710). Claudebox has reviewed the keep-list and the deletion PR is open for
   Christopher's merge OK.
6. NOT MET: the chat check. Either owners have working chat (proof named), or it goes on the gap list as a
   build.

Each of these becomes a PR with CI green, Tom's line-by-line review, one planted mutant caught, and the
hosted-command e2e plus a live Lab check where a server command changes. Then merge OK from Christopher.

## 3. Agents and harnesses (Christopher 11-24)
- **Workhorse:** GPT **Luna high**. Run `BASE=main ~/fleet/bin/launch-pulse-lane.sh luna <lane> <brief> high`.
  Output goes to `~/fleet/runs/pulse-<lane>/`; the worktree is `.worktrees/pulse-<lane>`.
- **More intelligence:** **Gemini Neon High**. Run
  `~/fleet/bin/agy-run.sh <job> <brief> "Gemini 4 Neon (High)" <workdir>`. Output goes to `~/fleet/runs/<job>-MMDD/`.
  Use it for risky deletions, design and second-model review.
- **If tokens run out:** use Claude subagents, **Haiku high** or **Sonnet medium** (Agent tool, model + effort set).
- **Watcher:** `~/fleet/bin/run-watch.sh`. Arm it when lanes run, and stop it when they finish.
- **Briefs:** the previous briefs are in `~/fleet/runs/overnight-twr-1123/briefs/` (format to copy). New briefs go to
  `~/fleet/runs/gaps-1124/briefs/`. Each one carries done lines, quoted rulings, assumption flags and "what the owner sees".
- **11-24 10:40 Christopher:** "lean more into luna, gemini is getting a tad thin, save it for multi file only".
- **Luna pattern seen 11-24:** never hangs, but quits after ~1 slice and over-reports ("tests pass" when broken, "no
  server twin" when there was one, quits on heavy-ok). Always verify claims; after 2 misses Tom finishes it.
- **Protect context:** read lane REPORT.md files and diffs by `--stat` first, then only the hunks that need
  review. Use scout/Explore subagents for wide reads.

## 4. Plan, in phases
**Phase A (start now, in parallel, Lab only):**
| Lane | Job | Builder | Notes |
|---|---|---|---|
| A1 canary | T708 runner: about 20 journeys through `twr workflow` (reads now possible, #493). Random subset, random times, seed logged, test leagues on the Lab only. Results to a tracker line | Luna high | Start from `~/Downloads/TOM-REPORT-canary-inventory.md` (24 journeys) and the T688 report card `cmd/twr-check`. Never the Season server or MFL leagues. Runs from Hermes (automations live there) |
| A2 pincache | Shared-facts #4: cache PinnedFootball by digest in `loadPin` (immutable, so sharing is safe; check it) | Luna high | Audit: `~/Downloads/TOM-REPORT-shared-facts-audit.md` §3 item 4 |
| A3 sbplayoff | Season-in-a-box plays a playoff round (bracket, final) | Luna high | Design: `docs/architecture/season-in-a-box.md`; tests from #480, #483, #486 |
| A4 sleeper | T695 slices per `~/Downloads/TOM-REPORT-sleeper-projections-plan.md` (8 slices). Tom first adds the Approved_Sources row and curls the address himself, read-only | Luna high (Neon if the scoring math gets hard) | P10: stat lines, scored by each league's own rules. Public use is not allowed on Sleeper data |
| A5 desktop | T710 removal per `~/Downloads/TOM-REPORT-desktop-keep-list.md` (~55.7k lines; watch the nativegen trap; the solo-owner tools stay: "trim, never delete") | Gemini Neon High | FIRST: Claudebox reviews the keep-list (`ssh claudebox`, `claude -p "…" < /dev/null`) |
| A6 chat | Does owner chat exist and work on the Lab? | Haiku subagent (read-only) plus Tom's Lab check | Result goes into GAPS-1124 §5 |

**Phase B (before Sat 11-28):**
- Shared-facts #1: an NFL schedule timer at server level (S).
- Prepare the Season server setup commands, with read tokens that include `native.directory.read`, for
  Christopher's OK.
- Plan the wall on real live screens (T689) for the Dec 5 swap.

**Phase C (December):**
- GATE part 4 history import: plain-words spec, then the ESPN access check (read-only, on Christopher's
  league), then build.
- Shared-facts #2 and #3, then the 32-team load rerun.
- The 10/12/16 leagues on the Lab.
- The rollback drill at a Saturday swap.
- Token renewal by Dec 22.
- Phone study and mocks after Dec 6 (T700).
- Phone alerts design.

## 5. Gates / status (11-24 1:45 PM CT)
Run dir: `~/fleet/runs/gaps-1124/` (briefs/, REPORT.md event log, merge/). Watcher: `~/fleet/bin/run-watch.sh wait ~/fleet/runs/gaps-1124 50` (background; re-arm every turn; exit 143 = replaced, ignore).
| Item | State |
|---|---|
| Merge authority | OK given for #494-#505 (all PRs so far). Nothing else. |
| MERGED | #494 desktop 1, #495 canary, #496 pincache, #497 Sleeper, #498 playoff test, #500 NFL schedule timer (+ Sleeper opt-in file), #501 MFL money in cents |
| Merge job (running) | `merge/merge-when-green2.sh 499 502 505 503 504` started ~1:40 PM from `.worktrees/pulse-sbplayoff`; log `merge/merge.log`; done = `merge/exit`; watcher name `merge6`. Auto-resolves ONLY line-baseline conflicts; anything else is logged "left open" |
| #499 desktop step 2.1 | MFL separation test (server twin), updated for cents after #501 |
| #502 | twr-server hosted refuses a second copy (flock on `<db>.lock`, in hosted.go) |
| #503 | docs: catalogue-preview-undo.md (134 write cmds: 97 both, 11 neither, 12 no undo, 14 no preview; 7 G3 draft-room cmds no undo) |
| #504 | desktop step 3 (PR-C): desktop CI out; prepush keeps `make bindings` on purpose |
| #505 | official stats built once per football version (GATE 3 leftover 2) |
| Lab | v77 = 5097adb50 (#500); `/etc/twr/sleeper-projections` and `/etc/twr/nfl-schedule-refresh` ON since 1:00 PM; first run 1:02 PM ok (schedule weeks=1 revised=5 errors=0; Sleeper current+next week). Auto-release per merge; hold Dec 1 + game windows |
| Desktop step 2.2 | no PR needed: every server restore behaviour already tested (`gaps-1124/desktopb2-restore-coverage.md`) |
| Season server | untouched (laptop later) |
| Luna lanes | none running |
| Parked | seasonsetup kit (session/pulse-seasonsetup 33c5a001a, not pushed) |

## 6. Artifacts
- Gap list: `docs/reset/GAPS-1124.md`.
- Grill sources: `docs/reset/RULINGS-1122.md`, `docs/research/icp-1122/RULINGS-part5-1122.md`. The raw dialogue is
  Claudebox `~/.claude/projects/-root/7ad05800-2eb9-432d-8861-761a155e17c2.jsonl` (11-22 16:00 to 11-23 03:20 UTC).
- Tom audit reports: `~/Downloads/TOM-REPORT-{shared-facts-audit,canary-inventory,redraft-check,desktop-keep-list,sleeper-projections-plan,test-inventory,line-budget,ai-parts-licences}.md`.

## 7. Refuted — do not retest
- "The rehearsal will rehearse playoffs": it won't. The schedule plays weeks 13–18 as regular games
  (`TestMidseasonLeadingEmptyWeeks`), and Christopher ruled no rehearsal playoffs.
- "Live scores, injuries, news and ids are fetched per league": they aren't. They are once per server (T693 audit).
- "Phone push alerts exist": they don't. No webpush, VAPID or PushSubscription anywhere in the Go or TS code (grep, 11-24).
- "Season-in-a-box covers playoffs": it doesn't. Its design doc has no playoff mention (grep, 11-24).
- Plus everything in `docs/resume/RESUME-overnight-1124.md` §7 (unit tests don't prove hosted commands; business rules
  don't belong in Validate()).

## 8. Decisions
- **11-24 (Season host):** "Lab now, laptop later". The real drafts (Tue Dec 1) and the first rehearsal weeks run on the
  Lab. Reason: `deploy/server/install.sh` only installs into a Proxmox container (the host firewall in front, console
  admin), and the only signed bundles are LAB-ONLY. A laptop installer (local firewall, Season signing key, same
  binary) is built later; the leagues move over on a Saturday with his OK. While the Lab hosts them: no Lab reset of
  the rehearsal leagues, and Lab auto-release holds on Dec 1 and during game windows (`~/fleet/bin/twr-lab-autorelease.sh`
  `in_hold`, Thu/Mon 17-24, Sun 11-24 Central; a `hold` file stops all releases).
- **11-24:** no rehearsal playoffs; the season-in-a-box test proves them.
- **11-24:** the campaign waits for Christopher's phone OK, even past January.
- **11-24 routing:** Luna high is the workhorse, Gemini Neon High for harder jobs, Haiku high / Sonnet medium if tokens run out.
- **11-22 (earlier):** all of the 11-22 rulings (see section 6 sources). Sleeper stat lines were chosen for step 1 (Q7).

## 9. Ledger
- Committed and pushed on `session/resume-1124`: 5b135bb98, 577485e65, f642f3889, plus this resume.
- Tracker on Hermes (`~/fleet/tracker/data/plan.json`): updated 7:29–7:38 AM. The backups are `.bak-1124a` and `.bak-1124b`;
  the next suffix is `bak-1124c`.

## 10. Next actions, in order
1. `run-watch.sh status ~/fleet/runs/gaps-1124`; read `merge/merge.log`; re-arm the wait. Handle any "left open" PR by hand (merge main in with `merge/update-branch.sh <worktree>`, fix, push).
2. After each merge: Lab auto-release; check `ssh -F .worktrees/pulse-a010s/.scratch/ssh-lab.conf lab pct exec 260 -- /usr/local/libexec/twr-status` says "Everything is working"; tracker card + log line.
3. Next Luna lanes (one slice each): Sleeper projection-vs-actual log before Thu Dec 3; install twr-canary on Hermes against a Lab canary league (Tom ops); desktop PR-D (check hosted import covers league.migrate.* first); offline commands (upgrade, restore-league, league-load) take the #502 lock (careful: join-link/seat-link may run while the server runs).
4. Phase B/C as section 4: wall plan T689 (week of Nov 30), history import spec, 10/12/16 leagues, rollback drill, token renewal by Dec 22, phone study after Dec 6, alerts, native chat.
5. Clean up finished worktrees (pulse-desktopb2 has only RESULT.md; others after merge) and `golangci-lint cache clean`.

## 11. Needs Christopher
- Nothing waiting: merge OKs given through #505.
- Product call later (from #503): give the commissioner draft-room commands an undo rule before an outside league?
- Sat 11-28: no Season server setup now (Lab hosts the Dec 1 drafts).

## 12. Environment notes
- Force-push is blocked by a hook: merge main in, then push normally.
- In worktrees, pre-push needs a `frontend/node_modules` symlink and `frontend/dist/.keep`.
- After removing a worktree, run `golangci-lint cache clean`.
- Hermes runs on UTC: take tracker times from Chicago zoneinfo in python, and use no apostrophes in ssh heredocs.
- Write commit messages with `git commit -F <file>` (don't combine it with `-m`).

## 13. Honest status
11-24 1:45 PM: Phase A built and merged except #499/#502-#505 (merge job running). Lab v77 healthy with both new jobs on. Every Luna lane needed Tom to finish or fix it; review caught real owner-visible bugs (MFL panel losing salary lines, mis-scaled tests). Unproven: official-stats grouping on the live Lab (needs a finished NFL week with `-official-stats`); Sleeper accuracy log not built.
