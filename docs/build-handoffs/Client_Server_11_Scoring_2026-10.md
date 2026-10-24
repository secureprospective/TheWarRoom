# Client/server 11 — pinned native statistics

## Flow and ownership

Commissioner import → `RawFootballEvidence.Validate` → weekly CSV validation +
`ReadPlays`/`AttachPlays` coverage validation → immutable digest-addressed history
body. Invalid JSON, identities, revision order, season or play coverage writes nothing.

`LoadFootball(digest)` → `PinnedFootball` installed at startup by the existing
`leagueservice.NewFootballHost` → history's native player/result/standings/preview
and statistics-queue adapters → the existing commissioner/fantasyscore engine.
All hosted native evidence uses the selected pin, never the mutable crosswalk or
fetch archive. Ordinary desktop archive readings retain their previous behavior.
No new package, database, HTTP route, binding or persisted projection.

## Import contract

The version-1 football envelope accepts optional `stats`, newest first (at most
100 revisions, within the existing 8 MiB envelope limit). Each revision contains:

- `asOf`: nonzero source timestamp, no later than the envelope's `asOf`;
- `weekly`: exact nflverse weekly CSV bytes;
- `plays`: exact projected play-by-play CSV bytes, including all required columns
  and coverage for each weekly row's game/season/type/week.

With statistics present, player `identity` values must be unique GSIS identities.
The envelope's `source` supplies provenance; its digest binds facts, identities,
source times and both CSV bodies. Each held statistic revision's `SHA256` hashes
`decimal byte length of weekly + ':' + weekly bytes + play bytes`, so changing
plays changes the evidence identity even when weekly quantities do not change.
The existing line digest includes position, quantities and exact distance events.

Keep all previous distinct revisions when preparing a replacement import: the
first line remains baseline; later lines require the existing ledger approval.
An import does not select itself or change running authority. Select the same
explicit digest again after restart. A facts-only pin stays explicitly unavailable
for native scores; it does not fall through to mutable archive statistics.

Complete held revisions retain their completeness through `DistinctLines`.
Previously that function recreated the missing-play-by-play warning even when
its caller supplied complete evidence. Missing rows are not invented zero lines;
a later file omitting a kicker preserves the original pinned baseline, just as
for other players. Native position `K` still maps to `PK`.

## Evidence

`go test -race ./internal/fantasyscore/... ./internal/store/history/...`.
The new `TestPinned*` cases exercise:

- published final results and exact QB/kicker totals, equal JSON digests after
  reopen and when replayed against an isolated archive holding only the pin;
- 39/40-yard passing and 19/20/40-yard rushing/receiving bonus boundaries through
  `Store.NativePlayerScore`, including stacked bonuses;
- the real held weekly/play fixtures, complete evidence retained across revisions;
- baseline versus approved revisions, queue/player-score parity and approved-score
  digest parity after reopening;
- a kicker omitted by the newest file retaining his 12-point baseline;
- invalid imports preserving archived row counts and the selected environment.

The isolated-archive proof does not delete append-only history tables or relax
triggers. It deliberately has no duplicate weekly/play fetch bodies to consult.

## Plan mismatch and limits

The older hardening plan's **kicker fallback** originates in audit recommendation
A14: scouting valuation should fall back to last-season performance where the fit
loses. That is `internal/model`, not native statistics or fantasyscore. This slice
proves native kicker scoring and baseline retention; it does **not** implement or
claim the separate valuation fallback. That requires an explicitly scoped follow-up.

No live MFL calls, production import, deployed server, full-season capacity or
source-provider completeness were exercised. Game presence is the existing
`AttachPlays` coverage contract, not independent proof every source play exists.
The envelope remains bounded; commissioner imports must fit it. Automated provider
ingestion, migration export and public owner workflows remain later slices.
