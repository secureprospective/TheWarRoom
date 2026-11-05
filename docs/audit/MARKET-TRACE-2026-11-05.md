# Market trace — 2026-11-05

Baseline: `23eb29e7`, branch `session/market-rework`. Source trace only; no application changes.

## Disposition: stop before mocks or implementation

STRING is partly available, not wholly absent: depth rank, NFL injury/IR designation and NFL free-agent team code exist. **No real NFL practice-squad status reader or held field was found.** A missing depth slot does not establish practice-squad status. The brief requires real status and says stop/report if in doubt; do not substitute fantasy taxi/IR or invent a collector. Owner/head-brain decision needed: identify an existing authoritative practice-squad input, or authorize an explicitly unknown STRING for uncovered players. PTS exists; its active-league/offseason selection still needs wiring and verification, not another collector.

## Existing STRING inputs

| Fact | Existing owner and route |
| --- | --- |
| NFL depth position/rank | `internal/fantasyscore/source/depth.go:12-35`: validated RawDepth has snapshot, team, formation, position, slot and rank, but no practice-squad/roster-status field. `internal/ingestion/researchfeeds/depth.go:19-59` retains the newest snapshot. |
| Depth identity join | `internal/app/research/context_reading.go:48-58`: `DepthID` resolves GSIS first, ESPN second through held links. `internal/app/research/player_reading.go:85-97` already renders all matching depth slots and says “not held” otherwise. Multiple formations can produce multiple slots; absence is not a status. |
| NFL injury / real IR | `internal/leaguefeed/types.go:89-99`: Injury preserves MFL's open vocabulary, including IR, IR-R, IR-PUP and IR-NFI. `internal/app/research/player_reading.go:66-83` uses held NFL injuries, distinguishes unavailable from not on report, and retains designation/details/return. Not every injury designation is IR. |
| Real NFL free agent | `internal/ingestion/players/fetcher.go:23-34`: Team can be FA; Status is a rookie flag, not NFL roster status. `internal/normalize/lookup.go:59-65` preserves Team and derives IsRookie from Status. |
| NOT real NFL status | `internal/domain/constants.go:35-41`: ROSTER/TAXI_SQUAD/IR are fantasy-franchise buckets. `frontend/src/app/control/state.ts:68` labels fantasy taxi “practice squad.” `internal/app/research/reading.go:167-185` labels “Free agent” by fantasy ownership, not NFL employment. None may be used to infer real NFL PS/IR/FA. |

Search: case-insensitive depth chart, injury, practice squad, roster status and IR across `internal/`; practice squad across `frontend/src/app/`. No authoritative NFL practice-squad field/reader found. This is a source-search finding, not proof about every possible external payload or profile's contents.

Existing frontend research route: `frontend/src/app/research/api.ts:15-17` → `TargetResearchPlayer`; `internal/app/research/api.go:56-70` hydrates held inputs, prepares depth, snapshots and renders the player. War Room's non-native inspector already consumes it at `frontend/src/app/shell/warroomInspectorReads.ts:52-56`. Reuse that owner, not a second depth/injury reader. Pool-wide per-player IPC fan-out has not been approved or measured.

## Existing PTS inputs

- `internal/app/research/finished_scores.go:18-41`: `FinishedScores` sums held league `playerScores` for finished weeks only. Missing weeks remain missing; zero-valued held scores count.
- `internal/app/research/player_reading.go:99-131`: existing “Season fantasy points,” last finished week and PPG presentation. Season sum and PPG are distinct; Market must not relabel PPG as PTS.
- `internal/app/scoring/points.go:31-65,68-90`: existing historical season-total loader and `outcome.fantasy_points` measure. Current and last season are refreshed; last season uses today's MFL league scoring year. This route is not by itself proof that an arbitrary native league's settings match MFL scoring.
- `frontend/src/app/pool/contract.ts:20-41` and `frontend/src/app/shell/warroomMarketReads.ts:12-13,77-82`: pool/Market currently transport engine/PPG values, age and minimum, but neither season-total PTS nor STRING.
- Existing research season presentation does not select previous season in the offseason (`playerPoints` uses `v.Finished`); cannot simply rename its current output and claim the requested offseason behavior works. Active league/rules and season scope need confirmation before connection.

## Why Legacy shows unavailable while MFL fills

Inspected the supplied 1920×1080 screenshot (not a new runtime capture). It identifies Legacy NFL as “32 teams · app”; targets, pool and placed bids show the generic unavailable wording.

Source-traced differences:

1. `frontend/src/app/shell/warroomMarketReads.ts:39-74` takes an early-return native/app path. It does **not** call `poolProvider.pool`, `personalWire.read` or `poolProvider.plans`.
2. Native pool reads `native.assets.read`, maps only `parseHostedRoster(...).pool`, then adds waiver-window players (`warroomMarketReads.ts:43-57,65-72`). `frontend/src/app/commissioner/hostedReadings.ts:103-118,134-137` parses that pool from `snapshot.pool`, defaulting it to empty.
3. That pool is explicitly free-agent lots (undrafted rookies / league-start O-Line), not the full NFL directory minus league rosters (`internal/commissioner/native_seed.go:40-41`). `internal/commissioner/native_reading.go:118-124` clones seeded lots; native assets reads reach the history owner via `internal/app/core/host_commissioner.go:161-164`. Waiver reading iterates recorded windows only (`internal/commissioner/market/waiver_reading.go:86-109`). Neither route constitutes the full MFL free-agent pool.
4. Private targets have no native read, and mutations explicitly refuse native context (`warroomMarketReads.ts:96-101`). Watch controls are disabled for native context (`frontend/src/app/shell/WarRoomMarket.tsx:108-111`).
5. Native bids read recorded own placed bids (`warroomMarketReads.ts:59-63,18-33`), not local READY plans. The card's draft/revise actions still use the local pool provider (`frontend/src/app/shell/WarRoomMarket.tsx:50-75`), an existing ownership mismatch that must not be extended blindly.
6. Non-native/MFL path instead calls existing `poolProvider.pool`, personal targets and local plan board in parallel (`warroomMarketReads.ts:76-93`). `internal/app/offers/pool_api.go:104-137` reads held MFL freeAgents and directory facts; never-fetched feeds produce a failed reading. `poolRows` also filters NFL-teamless players unless requested (`pool_api.go:149-156`).
7. Generic wording conceals the difference between empty, failed, pending and unsupported: `frontend/src/app/shell/WarRoomHeader.tsx:9-11` does not display the supplied reason, and even its title is the generic unavailable text. Native targets and empty pool invoke it (`WarRoomMarket.tsx:98-99,118`). Placed cards also invoke it for unavailable/empty bids (`WarRoomMarket.tsx:32-33`).

**Established:** wrong/incomplete pool scope for a full native free-agent market, unwired native targets, different placed-vs-plan bid sources, and loss of diagnostic wording. **Not established:** whether the pictured pool had zero lots, a failed read, or an unfinished read; the screenshot cannot distinguish these. The supplied soak app log contains lineup authorization failures but no matching assets/bids/waivers failure proving this specific pool failure. No database inspection or fresh click reproduction was performed. Not fixed.

## Next move and preservation

Resolve the authoritative PS/unknown-status decision and active-league PTS scope with the head brain/operator. Then connect existing owners, preserving identity joins, missing-data semantics, private franchise authorization, exact money and native ledger ownership. Do not add a parallel collector, route native plans into an unrelated MFL league, or infer NFL status from fantasy buckets. Resume the brief's mock-first sequence only after this blocker is resolved.

## Evidence and limits

- Disk preflight: 36 GB available, above the 15 GB stop threshold.
- `git pull --ff-only` failed: configured `/home/claude/twr.bundle` remote is unavailable. Requested branch was created from the exact supplied baseline.
- No protected profile/path/PID touched; no GUI launched, no profile copied, no MFL interaction, no screenshots created.
- No build/typecheck/Vitest run: documentation-only early stop. Existing supplied screenshot inspected at `~/scratch/soak-1105/run-1105-080433/shots/L1-war-market.png`.
