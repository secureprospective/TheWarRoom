# Coverage soak — findings log (2026-11-04)

Evidence only. Every line comes from a run on Claude-OS. Raw files are in `claudeos:~/scratch/soak/`.
Build: HQ-1 `c196a523` in a detached worktree `.worktrees/soak`, `GOFLAGS="-cover -coverpkg=<module>/..." make build`
(36 s, `go version -m` → `-cover=true`). The HQ live-try binary in `.worktrees/hq` was not touched.
Scripts: `docs/audit/soak/coverage-soak.sh`, `docs/audit/soak/soak-report.py`.

## Setup facts
- Claude-OS had just rebooted (up 4 min at 08:34 CT); `:5` was not running. Started `Xvfb :5 1920x1080x24` by hand.
  Nothing starts `:5` at boot.
- Profile copy = `~/f2click/config/TheWarRoom` without `backups/` (about 0.6 G) + `~/f2click/data`. Deleted at end of every run.
  `~/f2click/config` is 8.0 G in total because of three `TheWarRoom.pass-A-*` copies from 10-28/29 (4.9 G). Disk is at 89%; reaping those is his call.
- SIGTERM writes the coverage counters (probe: covmeta + covcounters written; 124 packages report).

## F1 — HQ has its own shell and nav rail (double-written shell)
Home, War Room, Trade Floor, Pulse and Control Room share a 190 px rail (6 items, y=175…359).
Franchise HQ swaps in a different 222 px rail ("THE WAR ROOM", numbered 1–6, My Shortcuts, nested HQ items, its own
franchise/league pickers at the bottom). Same job, two implementations. Shots: `probe-start.png` and `probe-control.png`.

## F2 — Trade Floor is a stub that repeats War Room
The Trade Floor body reads "Trade desk · Not wired · Ring 1". Its tabs (Trade desk, Draft room) repeat War Room's
Trade Desk and Draft Room tabs. It isn't one of the 5 locked mocks. Shot: `probe-trade.png`.

## F3 — Home first paint shows wrong league state for minutes
Same data copy, three launches:
- probe 08:36: "OFFSEASON · Reading the league…", record 2-0 seed 1, then 7-1 week 9 a few minutes later.
- run-0843: still 2-0 · offseason at 08:45 (about 2 min in).
- run-0859: settled on "Week 8 · you lead New England Patriots 239.5–181.8", record 2-0. HQ on the same data shows 7-1 (probe-control.png).
The real state is 7-1, week 9. **Two screens disagree, and Home disagrees with itself across launches.** This is a correctness issue,
not a speed issue.

## F4 — Startup: about 3.5 min before the nav responds; clicks queue for minutes after that
- run-0859: `history ready` at 53 s; `launch refresh` at **218 s**. Nav clicks before that light up the rail but the screen doesn't change.
- After `launch refresh`, clicks still landed late. The NFL Stats tab click at 09:03:36 only showed on screen around 09:06.
  In that loop, 33 steps produced only 9 distinct frames.
- Load average was 5.9 on the 8 GB VM while the walk ran.

## F5 — Memory: 2.33 GB RSS after one loop
`rss.tsv` run-0859, loop 1: 2,332,072 kB. Feeds `docs/debloat-log.md`.

## F6 — Every launch re-fetches whole nflverse files from GitHub
Each launch pulls `stats_player_week_<year>.csv` (from 2021 on) and `depth_charts_2026.parquet`. On 09:06 the connection was reset
→ "offline" state, and the depth charts went offline. The "signals" load ran 4+ minutes and was still going at shutdown
(`player record not written … context canceled`), so **values were never ready during a 1-loop walk** ("Player values are still being prepared").
Cached data should make the launch fetch unnecessary. Review against the score truth ladder (nflverse = authority, not a per-launch download).

## F7 — Control Room League and Server tabs gave the same frame
cr-league and cr-server hashed the same in run-0859. This could be the lag from F4. Recheck in a later loop of the long soak.

## Coverage so far (1-loop prewalks, not the soak result)
run-0843: 58,068 statements, 18,488 ran (31.8%). run-0859: 17,104 ran. Too short and too laggy to judge. The full soak decides.
