# RESUME — UI edges brief, GATE 2 shape leagues, chat (2026-11-25)

**Standing instruction (Christopher, 11-25):** "lean heavily into gpt for the heavy lifting from here on out to protect
your context". Tom's weekly Claude quota is thin. Luna high = small, well-scoped jobs. Sol medium = larger jobs
(`~/fleet/bin/launch-pulse-lane.sh <luna|sol> <lane> <brief> <high|medium> [nowt]`, `BASE=main`). Tom only writes
briefs, reviews diffs, plants a break, does Lab ops that need the kit secrets, and makes decisions.
**After compaction Christopher brings the Astra 6.2 findings** (the build plan). Review them against the rulings
(brief §1, §9) and turn the plan into lane briefs.

## 1. What we are doing
- Repo `~/work/TheWarRoom`; main = `5068968af` (#522). Lab (CT 260) on v93, "Everything is working".
- Lab ssh: `ssh -F /home/chris/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab`, then `pct exec 260`.
  The Lab kit (secrets) is `TWR_LAB_KIT=~/scratch/commish-1123` (KEEP.md); `LAB_SSH_CONF` is the same ssh config.
- Run dir: `~/fleet/runs/gate2-chat-1125/` (run-watch jobs; `setup.log`, `setup-try1-32teams.log`).

## 2. Done lines
| Done line | State |
|---|---|
| Build map HTML `~/fleet/runs/buildmap-1125/build-map-1122-1125.html` | MET (rendered on Claude-OS) |
| Astra brief `~/fleet/briefs/astra-ui-edges-1125.md` (repo copy `docs/reset/UI-EDGES-1125.md`, PR #523) | MET; handed to Christopher |
| Gaps flagged: Hermes T1836–T1843 + repo docs | MET |
| Chat rulings recorded `docs/reset/RULINGS-1125.md` (PR #523) | MET |
| GATE 2: native.rehtest-10/-12/-16 drafted by the computer, all teams computer-run, week-13 lineups set | **IN FLIGHT** |
| `league.chatLink` setting + "League chat" button (phone League tab, browser League Pulse) | **IN FLIGHT** (Sol lane) |

## 3. In flight (check with `~/fleet/bin/run-watch.sh status ~/fleet/runs/gate2-chat-1125`; re-arm `wait … 50` in background)
1. **shapes-setup**: `scripts/lab/shapes.sh setup` (branch session/pulse-gate2shapes, PR #524) is running on the Lab.
   - Done = `~/fleet/runs/gate2-chat-1125/setup.log.exit` exists (0 = ok). Drafts were being opened at compaction.
   - **The autorelease hold is SET** (`~/fleet/runs/twr-lab-autorelease/hold`). Remove it after setup and release.
   - Next:
     1. Wait for the drafts to settle. Check `grep -c "draft autopick: league=native.rehtest-10" /var/lib/twr/server.log`
        on CT 260 (and -12, -16) for the same count over 3 polls.
     2. Run `TWR_LAB_KIT=~/scratch/commish-1123 LAB_SSH_CONF=<conf> bash <worktree>/scripts/lab/shapes.sh release`
        (revokes the helpers, frees franchise.01).
     3. Verify each league: roster counts, and a week-13 lineup for every team via `twr read` (cfg dirs `~/scratch/commish-1123/cfg-shape{10,12,16}`).
     4. `rm` the hold, and log it.
   - Unknown: the one-season preset's playoff weeks against a schedule of weeks 13–18. Watch the bracket later.
2. **chatlink** (Sol medium, `~/fleet/runs/pulse-chatlink/`, worktree `.worktrees/pulse-chatlink`, brief
   `~/fleet/briefs/sol-chatlink-1125.md`).
   - Done = `completion.txt`. Liveness: the mtime of `sessions/*.jsonl`.
   - On done: read `tail -c 2500 stdout.log`, review the diff, plant a break (e.g. accept `http://`), commit, open a PR,
     and wait for CI.
   - Merges need Christopher's OK (the overnight "any green PR" authority has ended).

## 4. Open PRs (all need Christopher's merge OK)
- #523 docs: UI edges brief + 11-25 chat rulings.
- #524 lab: GATE 2 shape leagues (shapes.sh, schedule.py, tests). Planted breaks caught: first week 12, no rotation, odd count.

## 5. Refuted / lessons — do not repeat
- `twr-server create-league` ignores the preset's team count. **`-teams` defaults to 32.** Always pass `-teams N`.
  The first run made 32-team leagues; it was stopped before any draft opened.
- `join-link -release` needs the commissioner sign-in on stdin (`< lab-commish.json`). `-take … -no-login` does not.
- **Never `pgrep -f`/`kill` with a pattern that appears in your own command line.** It killed my own shell twice (exit 144).
- Luna's tests can be hollow (it tested its own list, not the script's output). Always plant a break that changes the output.
- Chat "retired 11-04" means a planned row was removed. **No chat code ever existed.**

## 6. Decisions (11-25)
- Chat for the rehearsal: link out (`league.chatLink`).
- Native chat: one store with rooms (league first, then draft room, trade-offer threads, notice replies), designed
  with the post-Dec 6 phone study (T1843).
- Alerts: mentions only (designed with T1826).
- Ask button: stays canned now; the owner's own AI helper later.
- Earlier rulings stand: `docs/reset/RULINGS-1124.md`, `BUILD-PLAN.md` §1.

## 7. Next actions
1. Check run-watch status. Finish GATE 2 (§3.1). Review chatlink (§3.2).
2. When Christopher brings the Astra plan: check it against brief §1/§9 and the calendar (Sun 11-29 practice drafts,
   **Tue Dec 1 real drafts on the Lab: nothing risky merges Mon–Tue**). Split it into Luna/Sol briefs and launch them under run-watch.
3. Add to the edge list: create-league `-teams` default 32 ignores the preset (an E1.2 detail).

## 8. Needs Christopher
- Merge OK: #523, #524 (and chatlink once green).
- Earlier, still open: Lab 4 CPUs, canary token T708, PR-F ruling, 3 specs (T1837).
