# twr-check: the Tuesday report card

Reads one league week over HTTPS with a read-only token and says, in plain words, whether the
published numbers add up. It never previews or executes anything and never edits a league.

    twr-check -server https://host -token-file F -league native.x -season 2026 -week 14 [-out detail.md]

One line per check: GREEN, AMBER, RED or UNKNOWN. Exit 0 = nothing red, 1 = something red,
2 = could not connect or the token was refused.

## What it checks
1. **Scores add up.** Each team's score = its starters' points + adjustments; games match team scores (within 0.005).
2. **Official stats final.** Every NFL game of the week is final and the evidence is not provisional, reduced or stale. If not: AMBER "official stats not final yet", never red.
3. **Standings match games.** W-L-T and points-for recomputed from every played week's results equal the standings.
4. **Lineups locked at kickoff.** The lineup read at each starter's kickoff holds the same starters that were scored. If the token cannot read a team's lineup, that team is UNKNOWN (the run does not fail).

## Read-only token
A commissioner makes it in The War Room: AI helpers, Connect a helper, choose "look and suggest"
(read mode only, no change rights). It needs native.directory.read (every league read checks the
league's directory first; without it each read is refused with 403), results.read, standings.read,
nfl.games.read and lineup.read. A token is for one league, so each league needs its own. Save the
helper line (`twr1.…`) or the bare token to a file (`chmod 600`) and pass it with `-token-file`.
Read-only tokens last 30 days.

## Hermes cron (not installed; Tuesday 9:00 and 11:00 AM Central)
    CRON_TZ=America/Chicago
    0 9,11 * * 2  twr-check -server https://HOST -token-file ~/.twr/check.token -league native.X -season 2026 -week WEEK -out ~/fleet/runs/reportcard/detail.md

Official stats need about 6 hours after Monday night's game plus a 3 hour re-read, so the 11:00 run is the retry.
