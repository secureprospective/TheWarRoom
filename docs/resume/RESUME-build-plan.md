# Build Plan Resume (from Sun 11-22)

**Plan (live):** `docs/reset/BUILD-PLAN.md` (written by Claudebox; Tom builds).
**Rehearsal (live):** `docs/reset/REHEARSAL-SEASON-1122.md` (goes ahead of Stages 3–7).
**Rulings:** `docs/reset/RULINGS-1122.md` (Christopher's 11-22 decisions). Index: `docs/reset/README.md`.
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

- **Stage 1:** done (this PR).
- **Stage 2:** T668 v50 lab release waits until after Sunday Night Football (~10:30 PM CT); T677+T679 on
  `session/t677-t679` are in review fixes.
