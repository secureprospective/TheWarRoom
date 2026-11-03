# APP-SET-2 — imported leagues keep the source's recorded results before cutover

Ruling (Christopher 11-02): pre-import weeks = source-recorded scores as-is; the app scores from cutover on.
Started by Sol (uncommitted); finished and reviewed by Tom.

## Storage
`RawRecordedResults{Season, CutoverWeek, Teams[franchise, week, points, W/L/T]}` rides on the ordinary
`ScheduleGenerate` ledger command (`RawScheduleArgs.Recorded`), published in the existing schedule
draft+publish batch — the same imported-ledger path as the schedule itself. No schema migration.
Validated at append (season, unique franchise/week below cutover, known franchises, boundary immutable).

## Boundary
Data, not origin: `CutoverWeek` in the held evidence. New import = first unfinished source week.
Weeks < cutover read recorded results; weeks >= cutover are app-scored. No `if MFL`. Native leagues hold
no evidence, so nothing changes for them.

## Readers
One resolver: `scoreFranchiseWeek` → `recordedWeekScore`. Every results consumer (NativeResults, standings,
odds, draft order, final standings, bracket, result flip) reaches it via `franchiseWeekScore`. The source's
recorded W/L/T verdict is kept (`applyRecordedVerdict`, `rankingGame`). Season score cache key includes
the boundary.

## Backfill (live MFL league)
`ScheduleSync.Sync` (existing reference refresh) fetches weeklyResults and calls
`mflsource.BackfillRecordedResults` for every imported league, following or not. Idempotent, revision-fenced,
failure keeps the ledger. Tom's fix over Sol: for a league imported by an older build the boundary is
`min(first unfinished week, week in play at the import receipt's RecordedAt)` (via held NFL schedule,
`App.importWeek`) — Sol's version would have used the refresh-time week and flipped app-scored weeks to
recorded. Unknown import week → no capture, retried next refresh.

## Verify
Targeted `-race -p 2`: history 181 PASS (2 FAIL pre-existing on base 8978f859:
TestNativeLineupReadingNamesLocksInTheOwnersTime, TestNativeLineupWire — reproduced on a clean archive of base);
migration+leaguemigration 88 PASS; commissioner 67 PASS. golangci-lint 0 issues. `make build` OK.
No new binding (models.ts gains `recordedResult` field only).

## Debt
- No test compares derived standings with an MFL standings fixture (no such fixture in repo); standings W/L/PF are asserted against the recorded fixture instead.
- `ScheduleCommands`/`SyncSchedule` take recorded evidence as a variadic; a plain param would be clearer.
