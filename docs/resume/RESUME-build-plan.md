# Build Plan Resume (from Sun 11-22)

**Plan:** `/home/chris/Downloads/TWR-BUILD-PLAN-1122.md` (written by Claudebox; Tom builds).
**Rulings:** `/home/chris/Downloads/TheWarRoom-DIRECTION-GRILL-1122.md` (beside the plan).
**Progress tracker:** Hermes `~/fleet/tracker/data/plan.json` (`ssh hermes`, UI at
`http://192.168.1.222:8790/`).

## The 7 stages

1. **Stage 1 — Clean the workspace:** inventory, backups to CT105, prune worktrees/branches, one front door,
   archive old docs.
2. **Stage 2 — Finish alpha loose ends:** T668 v50 phone auto-update release after 10 PM, T677+T679 v51
   release, one merge ask.
3. **Stage 3 — Stability & safety carry-overs:** T640/T631 32-team load test, T662 lineup.read, T663 auto lab
   release.
4. **Stage 4 — Find old assumptions:** read-only map into `docs/reset/ASSUMPTIONS.md` + task list triage;
   Christopher rules.
5. **Stage 5 — De-slop by area:** rank by agent friction/bugs; rebuild worst few; phone live scoring first.
6. **Stage 6 — Lock it in:** T664 commit-time fast checks, build guards, T665 daily cleanup, weekly habit
   check.
7. **Stage 7 — Product backlog after reset:** phone live "wow", test leagues + Stand-ins (T681), market study
   (T682), commissioner UX, MFL import (T456).

## State (11-22 ~1 PM CT)

- **Stage 1:** done apart from this cold-start docs PR.
- **Stage 2:** T668 v50 lab release waits until after Sunday Night Football (~10:30 PM CT); T677+T679 on
  `session/t677-t679` are in review fixes.
- **Live game-day notes:** `docs/RESUME-gameday-1122.md` on branch `session/pwa-sw-shell`.

## Remaining workspaces (11-22, Stage 1 step 4 ask)

61 workspaces. None is both unmerged and idle more than 7 days (oldest unmerged: 6.9 days), so
nothing meets the removal bar yet. The unmerged ones idle 4+ days, with a recommendation each (no removals
until Christopher or Claudebox says so). Full list: `~/fleet/runs/twr-wt-review-1122/INVENTORY.md`.

| Workspace | Branch | Idle (days) | PR | Recommendation |
|:--|:--|--:|:--|:--|
| uiux-fold1 | `session/uiux-fold1-record` | 6.9 | - | no PR: read its last commit, then bundle and remove (not read line by line yet) |
| values | `session/server-values` | 6.3 | - | no PR: read its last commit, then bundle and remove (not read line by line yet) |
| pulse-a010o | `session/pulse-a010o` | 5.9 | #302:CLOSED | back up (bundle), then remove: PR closed unmerged |
| pulse-f51a | `session/pulse-f51a` | 5.9 | #304:CLOSED | keep until Stage 3: F51 memory work (T640) the load test needs |
| pulse-f51b | `session/pulse-f51b` | 5.8 | #307:OPEN | keep until Stage 3: F51 memory work (T640) the load test needs |
| pulse-rel9 | `session/pulse-rel9` | 5.8 | #308:CLOSED | back up (bundle), then remove: PR closed unmerged |
| pulse-hqnotes | `session/pulse-hqnotes` | 5.7 | #313:CLOSED | back up (bundle), then remove: PR closed unmerged |
| pulse-f51t | `session/pulse-f51t` | 5.6 | - | keep until Stage 3: F51 memory work (T640) the load test needs |
| pulse-f51mem | `f51mem` | 5.3 | - | keep until Stage 3: F51 memory work (T640) the load test needs |
| pulse-s502 | `session/pulse-s502` | 5.3 | #320:OPEN | keep: open PR |
| pulse-wr4 | `session/pulse-wr4` | 5.2 | #323:CLOSED | back up (bundle), then remove: PR closed unmerged |
| pulse-wr5 | `session/pulse-wr5` | 5.2 | #328:CLOSED | back up (bundle), then remove: PR closed unmerged |
| pulse-adp | `session/pulse-adp` | 4.9 | - | no PR: read its last commit, then bundle and remove (not read line by line yet) |
| readme-v1 | `docs/readme-mock-v1` | 4.9 | #339:MERGED | remove after bundle: its PR merged (later commits differ) |
| pulse-b1hqdec | `session/pulse-b1hqdec` | 4.7 | - | no PR: read its last commit, then bundle and remove (not read line by line yet) |
| pulse-lab4 | `session/pulse-lab4` | 4.6 | - | keep: has unsaved edits; back up, then ask |
| pulse-lab5 | `session/pulse-lab5` | 4.2 | - | no PR: read its last commit, then bundle and remove (not read line by line yet) |
| pulse-b7lab | `session/pulse-b7lab` | 4.1 | - | no PR: read its last commit, then bundle and remove (not read line by line yet) |
| pulse-b8plan | `session/pulse-b8plan` | 4.0 | - | no PR: read its last commit, then bundle and remove (not read line by line yet) |

