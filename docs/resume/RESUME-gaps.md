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

## 5. Gates / status (11-24 ~10:25 AM CT)
Run dir: `~/fleet/runs/gaps-1124/` (PLAN.md, briefs/, REPORT.md event log, merge/). Watcher: `~/fleet/bin/run-watch.sh wait ~/fleet/runs/gaps-1124 50` (background; re-arm every turn).
| Item | State |
|---|---|
| **Merge authority** | Christopher 10:00 AM: "merge four PRs when green" = #494 #495 #496 #497 ONLY |
| #494 desktop step 1 | MERGED 10:15 AM |
| #496 pincache (+ Tom lock fix 75b09e30e) | conflicted on generated line baseline; main merged in locally (6990e4930); v2 merge job pushes + merges when green |
| #497 sleeper step 1 (+ Tom fixes) | in v2 merge queue |
| #495 canary `cmd/twr-canary` | in v2 merge queue |
| v2 merge job | `~/fleet/runs/gaps-1124/merge/merge-when-green2.sh 496 497 495`; log `merge/merge.log`; done = `merge/exit`; auto-resolves ONLY line-baseline conflicts |
| schedtimer (`-nfl-schedule-refresh`) | session/pulse-schedtimer eaa2a155f, Tom fixes (advancePin race, per-league timeout, no panic), mutant caught. Not pushed, NOT in merge OK. Merge main in after #496/#497 (football_pins.go, hosted.go), push, PR, ask Christopher |
| sbplayoff (bracket to champion) | Gemini 7531f8632 on session/pulse-sbplayoff, all done lines met, test/docs only. Tom: own mutant, untrack notes, push, PR, ask merge OK |
| seasonsetup kit | session/pulse-seasonsetup 33c5a001a, parked (laptop later). Not pushed |
| Lab | v72 = 60ebd21fe before merges; auto-release each merge; hold Dec 1 + game windows |
| Season server | untouched (laptop later) |
| Hooks | commands mixing `git commit -F` and `git push` trip the force-push guard; heredocs with `git merge` while the main checkout is on main trip the main guard. Split calls; write scripts with Write |

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
1. `run-watch.sh status ~/fleet/runs/gaps-1124`; read `merge/merge.log`; re-arm the wait.
2. sbplayoff: plant own mutant (wrong expected champion), untrack HANDOFF.md/REPORT.md, push, PR, ask merge OK.
3. schedtimer: after #496/#497 land, merge main in, `go test -race ./cmd/twr-server -run NFLSchedule`, push, PR, ask merge OK.
4. After merges: Lab check (twr-status "Everything is working"); Sleeper flag on for the Lab before Thu Dec 3 (twr-sandbox flag-file line like schedtimer's); install twr-canary on Hermes vs a Lab canary league; tracker green/red.
5. Desktop step 2 (PR-B parity tests) per docs/architecture/desktop-removal-plan.md; check hosted import covers league.migrate.* before PR-D; PR-F needs Christopher.
6. Phase B/C as section 4. Laptop installer task on Hermes Task_list (~line 429).

## 11. Needs Christopher
- Merge OKs as PRs land (none waiting now). The docs PR for `session/resume-1124` is not opened yet.
- Sat 11-28: Season server setup, with his OK on each command.

## 12. Environment notes
- Force-push is blocked by a hook: merge main in, then push normally.
- In worktrees, pre-push needs a `frontend/node_modules` symlink and `frontend/dist/.keep`.
- After removing a worktree, run `golangci-lint cache clean`.
- Hermes runs on UTC: take tracker times from Chicago zoneinfo in python, and use no apostrophes in ssh heredocs.
- Write commit messages with `git commit -F <file>` (don't combine it with `-m`).

## 13. Honest status
11-24 10:25: Phase A all built; four PRs merging. Luna high ~1 slice per run; Gemini Neon High took canary and playoffs after two Luna misses each. Every lane had real review finds.
Planning is done. Nothing in Phase A has started, and no build is running. The ETA for Phase A is about 1–2 days of
lanes plus review. A4 depends on the Sleeper address still answering (unverified since 11-22).
