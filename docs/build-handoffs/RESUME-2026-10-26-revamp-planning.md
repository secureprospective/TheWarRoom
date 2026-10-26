# TheWarRoom — RESUME 2026-10-26 (revamp planning)

## 1. What we are doing
We are planning the inward revamp of TheWarRoom now that client/server is done (50/52 slices). This session mapped the whole system, locked 16 parts with Christopher one at a time, and shipped a README makeover. **No revamp code has been written. Session 0 starts only when he says go.**
- Repo: Beelink `~/work/TheWarRoom` (main checkout sits on old branch `session/league-history-pwa`; use worktrees from origin/main). origin/main = `37e25a21` (README #180).
- ssh: `chris@192.168.1.191` (key works from CT105).

## 2. Agents / harnesses
- Sol 6.1 medium on Bee: `~/fleet/bin/launch-sol61-medium.sh <brief.md>...`. Runs go to `~/fleet/runs/sol61-<brief>/` (stdout.log, sessions/, completion.txt). Launch with `ssh -f`.
- Mapping runs (done, exit 0): `~/fleet/runs/sol61-revamp-map-{A-engine-data,B-league-server,C-frontend-surface}/stdout.log` (74/91/88 KB). Briefs: `~/fleet/runs/revamp-map-2026-10-26/briefs/`. The read-only src snapshot was reaped; recreate with `git archive 3110ab39`.

## 3. Status
| Item | State |
|---|---|
| System map (3 Bees) | DONE, read in full, key claims spot-checked |
| 16 parts walked with Christopher | DONE, all locked |
| HTML revamp map | DONE: Beelink `~/logs/TheWarRoom/warroom-revamp-map.html` (+ `~/Downloads`) |
| Plan file | CT105 `/root/.claude/plans/in-this-session-we-logical-puzzle.md` (the 16 parts + session order) |
| README makeover | MERGED, PR #180 → main `37e25a21`, CI green |
| Revamp session 0 | NOT STARTED, waits for him (task T494) |

## 4. Verified facts (spot-checked in code @3110ab39)
- `internal/labcompile/state.go:11`: `lastYear = 2026`. `model.go:68-70` refuses valuations outside → **asset values break 2027-01-01**.
- `frontend/src/app/commands/registry.ts:757`: dispatch throws unless roles include `gm`; `data.parameter` (line 286-289) is `['admin']` → the button can never run.
- `commissioner/surfaceActions.ts:13`: `handlers.get(screen)?.(args)` silently drops actions.
- `commissioner/hold.ts:2`: `holdMs = 700` (ruling ≤600).
- `internal/commissioner`: 21,597 production lines. Go total 122k + 106k test; TS 38k + 18k test.
- `nativeGrammar`/`createNativeClient` and `ReadPreviewCommands` have 0 production callers.
- Command lists: 62 UI registry / 136 native catalog / 9 MFL / 136 Wails methods; server workflow allowlist = 42.

## 5. Current bug
None in flight.

## 6. Refuted — do not retest
- "Six-layer engine is live": FALSE. Only decay remains; l4 knobs are inert (removed-code.md:82-109).
- "lab/labcompile is a spent harness": FALSE. It's live at runtime (~12 package families import it). The ruling is to separate it anyway.
- GUIDES/WORK_LOGS/archive/league-archive dirs don't exist on main.

## 7. Decisions (Christopher, 10-26; all saved to memory)
- One part per session. Order follows what the map shows pays off most. MFL import keeps working.
- Every button is a command, under ONE standard: an executable registry that generates every door.
- **The league = rosters + contracts.** Recovery restores those first, with a per-league recovery log and a tested "stand back up" command.
- Our own canonical player ID. Ownership keyed by side + player.
- **Default mode 1:** mirrored 32-team AFC/NFC. Two full player pools, each player owned once per side, no cross-side moves, normal cross-side schedule, side champions meet in a final. **Mode 2:** a modified Legacy league, to be discussed later (T495).
- Lean native; the MFL harness gets less attention. The Lab leaves the app and becomes an R&D bench for data fields. A small 2027 stopgap is approved.
- Widest stat suite possible (QB hits, first downs, IDP, play-level, snaps, live, corrections). MFL is never the football-fact source. Compare providers; he picks.
- Non-techie first, experts never limited.
- Screens get a scorecard (tools rate against the philosophy) before tinkering.

## 8. Ledger state
- Committed: README (#180). Memory files written in `/root/.claude/projects/-root/memory/`: league_is_rosters_contracts, default_mirrored_conference_league, lab_separate_native_first, full_stat_suite; revamp_after_cs updated.
- Uncommitted: nothing in the repo.

## 9. Next actions, in order
1. Wait for Christopher's go. Then run session 0: Lab 2027 stopgap, GM/admin dispatch fix, hold 600 ms, CLAUDE.md truth pass (Go 1.26.10, no RESUME ref, UI spec rev 4, pre-push wording, SYSTEM_MAP ratchet target). Branch `session/revamp-0-quick-fixes`, worktree from origin/main.
2. Then session 1: the command spine (see the plan file, Part 1).
3. Before building default modes: scope mode 2 with him (T495).

## 10. Environment notes
- CI on main takes ~18 min. The `verify` check is required; auto-merge is disabled. Merge with `gh pr merge N --squash --delete-branch` after green.
- Commits over ssh: `GOTOOLCHAIN=go1.26.10`; PATH needs `~/.local/bin:~/vendor/gh/bin:/usr/local/go/bin`.
- Task list: Hermes `cc` (T494, T495 added).

## 11. Honest status
The plan and map are agreed. Nothing in the revamp is built or proven. The 2027 cliff is real and dated, so session 0 should land before Jan 1.
