# Revamp E: edge-stat scoring in the 32-Team Default, as six slices

Status: plan. This supersedes the 23-slice draft at 02a1f551, which is kept in git history; its per-file notes are still a useful reference for E1 and E2.

## Rulings (the owner, 2026-10-27)
- **Package A values:** from `~/fleet/tom/out/config-study-p6.out` §2.
- **Special-teams solo tackles:** a top-up of +0.25.
- **Snaps:** defensive 0.008, offensive 0.002.
- **Third/fourth-down conversions and red-zone stats** come from nflverse play-by-play.
- **Rushes tackled for a loss (RTL)** are scored weekly, at half strength, against last season's league rate at the position:
  `−0.31242 × (TFL − r_pos × carries)`. There is no 40-carry floor, and past scores never change.

## The design choice that keeps this small
RTL is linear, so it needs no new arithmetic, no season settlement and no freeze machinery. It becomes two ordinary rules:
1. A `rushing_tfl` stat at **−0.3124** each, for the offensive positions.
2. A **position extra** on `rushing_attempt` (the existing `Extras` mechanism, `fantasyscore/kinds.go` extraItems) at **+0.3124 × r_pos** for each of QB, RB, WR and TE.

`r_pos` is a constant in the Default32 preset, computed once from the pinned 2025 nflverse play-by-play by a checked-in script. Its value is cited like every other preset value. Changing it in a later season is an ordinary commissioner rules change, already governed by the rules engine. Nothing new is built for that.

The source of `r_pos` is nflverse play-by-play, the final truth on the score truth ladder.

## Slices (built in order; each one compiles and passes its own targeted tests)
| ID | Goal | Builder |
|---|---|---|
| E1 | In the nflverse adapter, derive per-player counts from the already pinned play-by-play: 3rd- and 4th-down conversions (rusher/receiver credit and passer credit, as p6 credits them), the red-zone stats in Package A, and rushes tackled for a loss. The digests of existing evidence must stay unchanged; test that. | Terra |
| E2 | A Sleeper edge-stat adapter for the Package A codes Sleeper supplies, typed decode, attached to player-weeks, saved and fingerprinted through the EXISTING evidence path (no new pinning scheme). | Terra (high) |
| E3 | `scripts/` or `cmd/` tool: compute r_pos per position from the 2025 play-by-play, then add the `rushing_tfl` row and the `rushing_attempt` position extras to Default32. Table test: an RB with 20 carries and 3 TFL scores −0.3124×3 + 20×(0.3124×r_RB). | Terra |
| E4 | Every remaining Package A value, the special-teams top-up and the snap values as Default32 rows. Table tests per code. | Terra |
| E5 | Golden: 5 real 2025 players (Hunt, McCaffrey, Hurts, Allen, Barkley), weeks 1–13, from adapter to engine. The expected totals are computed independently by the p6 study scripts, adjusted for the ruled RTL. Tolerance 0.01 per player-season. | Terra (high) |
| E6 | Scoring docs and the Default32 spec updated to match. | Terra |

Each slice's brief names its files, its tests and a "done =" line. I review each slice before the next one starts.
