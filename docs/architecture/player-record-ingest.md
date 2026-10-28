# Player record ingestion

The validated MFL listing → `composition.directory` publication → record store `Write` resolves
identities and writes all player records in one history-file transaction. A cached listing read
neither fetches nor writes. Signals loads (background and explicit refresh) → `sources.LoadSignals`
→ record write → signals publication → season event. The record write sees the newly ingested facts.

The record store owns identity resolution and atomic persistence. A failed transaction retains the
previous records and is logged; the next ingest retries. Missing history at startup skips the write.
Valuation reads the stored Book only; an empty record store refuses rather than resolving records.
Composition supplies callbacks only. No scoring formula, population or precedence changed.

Tests: `TestPlayerRecordIsWrittenWhereItsFeedsLand`, `TestValuationReadsTheRecordItDoesNotWriteIt`,
`TestRecordIngestKeepsFailuresVisible`, and the record store's held/stored Book equality test.
GUI acceptance and the held 10-28 value-board replay are separate checks.

## Supplied career facts

The already-held nflverse `draft_picks.csv` supplies `seasons_started`, `allpro` and `probowls`.
Registry mappings ingest them as player-grain career counts, not draft-season statistics. The record
stores them separately from the model's inputs/population, with nflverse provenance; a known zero is
not a missing count. Mislinked identities do not inherit these facts. History migration 12 adds a
non-destructive `career` JSON column (existing records default to unknown); server release 5 names it.

`seasons_started` means seasons as a primary starter, **not games started**. None of the held registered
feeds supplies player wins or game starts: those are not manufactured. In particular `w_av` is weighted
approximate value, not wins. Career facts do not enter the approved value formula.
