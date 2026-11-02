# APP-SCORE-1

## Baton
Bee — build/SCORE-1, 2026-11-02. Local only.

## Map before coding (source-traced)
- `internal/composition/live_stats_app.go` starts one poller after stores open; `internal/app/now/live_stats.go` fetches ESPN scoreboard/summary, resolves ESPN IDs, holds live lines through `history.SetLiveStats`, signals moves. Failed acquisitions retain held evidence. Game scope stays on each player week.
- `internal/ingestion/livestats` converts passing/rushing/receiving/defense only. Other ESPN categories are ignored today. Missing quantities are absent, not zero.
- `internal/ingestion/livegame.Adapter` ESPN → Yahoo fallback is NFL game clocks/display, not fantasy stat sourcing. Do not confuse that provider chain with scoring.
- `internal/fantasyscore/source` parses nflverse weekly/play-by-play; `internal/store/history/football_stats.go`/`football_sleeper.go` construct pinned revisions. Sleeper currently imports 13 edge fields only and overwrites matching nflverse quantities in attach. No live Sleeper fetch exists.
- `internal/store/history/live_stats.go` appends live revisions without renumbering held finals/pins. `internal/commissioner/stat_ledger.go` FeedVersion prefers newest exact nflverse authority; countedVersion honors manual pin and withCorrections applies per-field manual overrides last. `stat_feed.go` auto-records selection; `stat_queue.go` compares revisions and routes review evidence. These excluded files will not be edited.
- `internal/ingestion/livescoring` MFL feed carries already-scored player/franchise totals, not raw stat quantities. It cannot feed a field-level raw-stat chain safely. No new league-origin paths will be introduced.
- Provenance exists internally in HeldWeek.Authority; NativePlayerScore does not serialize that authority; shell NativeLeaguePulse's scores label is currently static “the league ledger”.

| Class | Today | Locked ruling / intended change |
| --- | --- | --- |
| Core offense/IDP live fields | ESPN converted subset only | ESPN first; Sleeper fills absent fields, including explicit ESPN zero preserved |
| Snaps/drops/broken tackles/air yards/ST tackles | Sleeper only in pinned imports | Sleeper live gaps too |
| nflverse posted quantities | Final revision wins wholesale, but pinned Sleeper attach overwrites overlap | nflverse field wins; Sleeper only absent quantities |
| Missing nflverse player | Missing evidence; live revision may count | Keep other sources, never fabricate zero |
| Manual selection/correction | Pin, then corrections last | Preserve existing commissioner engines |
| ESPN unavailable | Retain held live evidence, no raw MFL fallback | Raw MFL fallback unresolved: existing adapter has totals only |

## Intended flow
Validated ESPN lines + validated Sleeper week → ordered per-field source merge (map presence, never nonzero preference) → live book → unchanged revision/commissioner engines. Pinned nflverse + Sleeper uses the same merge rule. Network/identity failures retain ESPN evidence; optional Sleeper failure cannot stop ESPN. Never infer zero for absent player.

## Delivered (partial acceptance)
- One ordered field chain in `source.MergeStatSources`: nflverse → ESPN → MFL (only without an ESPN candidate) → Sleeper. Map presence preserves explicit zero; event/quantity ownership stays together; inputs are not mutated. Commissioner override stays in its existing, tested ledger overlay.
- Live polling now fetches/validates Sleeper once per active regular-season scope, resolves held Sleeper IDs, fills absent ESPN fields, and includes Sleeper-only players with recorded evidence. Unknown players are ignored. Optional acquisition/validation failure keeps ESPN unchanged. No external live fetch was used for verification.
- Pinned Sleeper attachment uses the same field merge: no more replacement of an explicit nflverse edge value. Existing legacy serialization/fingerprint tests pass.
- ESPN conversion additionally handles reported assists, fumbles/lost fumbles, interceptions/return yards, kick/punt return yards and extra points. No guessed fumble-recovery ownership or kick-distance bands.
- Live held authority names actual contributors (`ESPN live`, `ESPN live + Sleeper`, or `Sleeper`). Shell scores label displays the feed **policy**, explicitly not selected-matchup provenance.

## Verification
`go test -list` filters were listed first; targeted `go test -race -p 2` only. Counts include named tests and subtests, not package PASS events:

| Package | PASS | SKIP |
| --- | ---: | ---: |
| internal/fantasyscore/source | 16 | 0 |
| internal/app/now | 15 | 0 |
| internal/store/history | 26 | 1 |
| internal/ingestion/livestats | 12 | 0 |
| internal/commissioner | 2 | 0 |
| Total Go | 71 | 1 |

History skip is opt-in `TestSleeperUnknownIDReport`. Filters: `Test(Live|TheFinalFeed|AnUnchangedLive|PinnedSleeper|Sleeper|ScoreSource)` on the first four packages; Commissioner filter `Test(FeedCountedRevisionFollowsLiveThenNFLVerse|LaterLiveEvidenceGapsDoNotDowngradeNFLVerseFinal)` includes real automatic final selection, per-field manual correction and manual pin assertions. Additional ESPN conversion cases ran with `TestLive`.

- Frontend `pnpm exec vitest run src/app/shell/appLeaguePulse.test.tsx --maxWorkers 2`: 4 PASS.
- Changed Go package golangci-lint: 0 issues.
- `make build VERSION=dev COMMIT=SCORE-1 BUILD_DATE=SCORE-1`: PASS. Existing Wails generic-model warnings remain. No desktop launch or full suite.
- Inverted the source-priority table temporarily: overlap regression test failed as intended; restored exact production bytes and reran source race batch successfully.
- Whole affected handwritten production set: 9 files, 791 → 1,104 physical lines (`splitlines`, tests/docs/config excluded). Growth adds live protocol conversion/acquisition and one shared priority rule; not claimed as debloat.

## Remaining debt / excluded-file coordination
**Do not mark APP-SCORE-1 complete.**
1. MFL fallback is tested only as a supplied raw-stat candidate in the merge function. No MFL raw-stat acquisition is wired. Existing liveScoring exposes fantasy totals, not per-field quantities; converting those into raw stats would invent evidence or violate arbitrary league settings. ESPN outage still retains held evidence. Need a verified raw MFL stat contract before wiring a real fallback.
2. ESPN coverage remains a converted subset: kicking distance bands, some recovery/return touchdown distinctions and play-derived fields still require acquisition/conversion. The new live Sleeper conversion is an explicit field whitelist, not every possible Sleeper stat. Therefore “every stat ESPN carries / any missing stat” is not fully satisfied.
3. Finals still count through the existing whole-revision `FeedVersion` preference. Pinned final imports fill Sleeper edges, but live-only fields are not merged into an already-counted final revision. Full final/live field consolidation must preserve immutable revision digests and manual pins; coordinate any selection change with APP-STALL-2 in excluded `stat_ledger.go` / `stat_feed.go`, rather than silently rewriting held revisions.
4. Actual selected-source provenance is not yet exposed through NativeResults/NativePlayerScore to the aggregate scores label. Current visible text describes policy only; internal live authority is accurate. Add the selected authority to the reading contract and render it before claiming runtime provenance complete.
5. No excluded file was edited. No excluded-file change is necessary for the delivered subset; the full unified selection above needs coordination. No league-origin branches, scoring-rate literals, new scoring model, remote operations or real-profile access.
