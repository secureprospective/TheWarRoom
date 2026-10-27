# Revamp E: edge-stat scoring in the 32-Team Default, as six slices

Status: E1–E6 complete. See [the shipped scoring spec](../architecture/default32-scoring.md). This supersedes the 23-slice draft at 02a1f551, which is kept in git history; its per-file notes are still a useful reference for E1 and E2.

## Rulings (the owner, 2026-10-27)
- **Package A values:** from `~/fleet/tom/out/config-study-p6.out` §2.
- **Special-teams solo tackles:** a top-up of +0.25.
- **Snaps:** defensive 0.008, offensive 0.002, special teams 0.003.
- **Third/fourth-down conversions and red-zone stats** come from nflverse play-by-play.
- **Rushes tackled for a loss (RTL)** are scored weekly, at half strength, against last season's league rate at the position:
  `−0.3124 × (TFL − r_pos × carries)`. There is no 40-carry floor, and past scores never change.

## The design choice that keeps this small
RTL is linear, so it needs no new arithmetic, no season settlement and no freeze machinery. It becomes two ordinary rules:
1. A `rushing_tfl` stat at **−0.3124** each, for the offensive positions.
2. A **position extra** on `rushing_attempt` (the existing `Extras` mechanism, `fantasyscore/kinds.go` extraItems) at **+0.3124 × r_pos**, rounded to the stored precision, for each of QB, RB, WR and TE.

`r_pos` is a constant in the Default32 preset, computed once from the pinned 2025 nflverse play-by-play by a checked-in script. Its value is cited like every other preset value. Changing it in a later season is an ordinary commissioner rules change, already governed by the rules engine. Nothing new is built for that.

The source of `r_pos` is nflverse play-by-play, the final truth on the score truth ladder.

## Slices (built in order; each one compiles and passes its own targeted tests)
| ID | Goal | Builder |
|---|---|---|
| E1 | In the nflverse adapter, derive per-player counts from the already pinned play-by-play: 3rd- and 4th-down conversions (rusher/receiver credit and passer credit, as p6 credits them), the red-zone stats in Package A, and rushes tackled for a loss. The digests of existing evidence must stay unchanged; test that. | Terra |
| E2a | Scoped exact Sleeper bodies in `RawFootballStats.Sleeper` / `RawFootballWeek.Sleeper`; `RawFootballPlayer.Sleeper` bridges directly to the player's MFL map key, even without GSIS. `source.RawSleeperStats.Validate/Convert` owns only the 13 Sleeper edge quantities. Existing archive path and legacy fingerprints retained; round-trip and compatibility tests. Implemented; attachment is E2b. | Bee |
| E2b | **45-minute build: attach and read, no new format.** Files: `internal/fantasyscore/source/sleeper.go`, `internal/fantasyscore/source/identity.go`, `internal/store/history/football_sleeper.go`, `internal/store/history/football_stats.go`, `internal/store/history/playerstats.go`; tests beside them. Decode each validated `RawSleeperWeek.Body` into `map[string]source.RawSleeperStats`; use `Convert()`, exclude nonnumeric team IDs and all-zero rows, count unknown-ID nonzero rows per season (no identity inference). Bridge through `raw.Players[mfl].Sleeper`, never require GSIS. In `scoringFile`, finish nflverse `AttachPlays` first, then build a canonical MFL/week lookup over the union of known nflverse weeks and nonzero Sleeper weeks. Merge only Convert's 13 owned quantities; preserve every nflverse/PBP-owned value. Sleeper-only weeks use pinned player position, real season/week, REG, only edge quantities; no invented GSIS or base stats. Store the canonical lookup on `heldFile`. Add an optional canonical-stat reader interface in `source/identity.go` and implement it on `PinnedFootball`; `ReadNativeEvidence` uses that reader before its GSIS-only fallback, retaining held revisions/absence semantics. Do not change ordinary GSIS readers. Compact real-data fixtures: Mausi Sleeper 12944 → MFL 17367, GSIS empty, LB; ST snaps by week 1–13 = [20,20,14,14,22,19,17,19,20,absent,18,16,20], ST tackles w5=1/w6=1/w9=2, penalty w12=1. DJ Johnson Sleeper 10938 → MFL 16277, GSIS 00-0038535, DE. Tests: `TestPinnedSleeperMausiWithoutGSIS` imports those 2025 weeks 1–13, reads MFL 17367 via ReadNativeEvidence after reopen and totals **+1.407 unrounded** (219 ST snaps, 4 ST tackles, 1 penalty); `TestPinnedSleeperDJJohnsonSparseWeek` reads MFL 16277, 2025 week 1 with no nflverse row and scores **+0.167 unrounded** (16 defensive, 13 ST snaps); `TestPinnedSleeperPreservesNFLverse` checks overlapping passing/rushing and PBP conversion/red-zone counts unchanged, team/zero/unknown rows create no weeks; `TestPinnedSleeperRevisionAbsence` proves newest absence is not replaced by an older row. Keep `TestSleeperEvidenceRoundTrip`, `TestSleeperLegacyEvidenceBytesAndDigests`, `TestSleeperFingerprintIncludesBodyScopeAndBridge`, `TestSleeperOwnedQuantities`, `TestSleeperUnknownIDReport`. Command: `GOTOOLCHAIN=go1.26.10 go test -race ./internal/store/history/ ./internal/fantasyscore/...` then `GOTOOLCHAIN=go1.26.10 go build ./...`; opt-in report uses `TWR_STUDY_CACHE` (p5) and `TWR_SLEEPER_STUDY_CACHE` (p6). **done = both named losses restored through pinned import → reopen → canonical read → engine, legacy bytes/digests unchanged, unknown-ID counts visible, targeted tests/build pass; commit, do not push.** | Terra (high) |
| E3 | `scripts/` or `cmd/` tool: compute r_pos per position from the 2025 play-by-play, then add the `rushing_tfl` row and the `rushing_attempt` position extras to Default32. Table test: an RB with 20 carries and 3 TFL scores −0.3124×3 + 20×(0.3124×r_RB). | Terra |
| E4 | Every remaining Package A value, the special-teams top-up and the snap values as Default32 rows. Table tests per code. | Terra |
| E5 | Golden: 5 real 2025 players (Hunt, McCaffrey, Hurts, Allen, Barkley), weeks 1–13, from adapter to engine. The expected totals are computed independently by the p6 study scripts, adjusted for the ruled RTL. Tolerance 0.01 per player-season. | Terra (high) |
| E6 | Scoring docs and the Default32 spec updated to match. | Terra |

Each slice's brief names its files, its tests and a "done =" line. I review each slice before the next one starts.
