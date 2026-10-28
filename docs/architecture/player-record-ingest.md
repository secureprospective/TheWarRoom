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
