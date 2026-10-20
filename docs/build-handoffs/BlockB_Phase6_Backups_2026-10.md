# Block B phase 6: local backups and checks

## Shipped: routine checks

`internal/db/routine_checks.go` checks each pool after launch and at seven-day
intervals. The app's hourly maintenance wake checks eligibility; the existing
worker lifetime cancels and joins it before pool closure. Checks use readers,
not the single writer. Data Sources adds `databaseChecks` with the existing
report renderer, including unchecked, failed-execution and violation results.

The last attempted result is process-local and owns its slices. Restarting runs
checks again. Persisting results would require a separate storage decision; no
new file format or schema was introduced. A two-minute budget covers all three
files per maintenance pass. Check failures remain visible until the next check.

Proof: db race suite passes. Removing the weekly guard made
`TestRoutineChecksAreWeeklyAndRetainFindings` fail with an early FK finding.
Removing reading wiring made `TestDataSourcesIncludesLocalDatabaseChecks` fail.
The frontend sources suite initially rejected `databaseChecks` as unexpected;
it passes after parsing/rendering the report. TypeScript and targeted Go lint pass.

The single full root race run failed `TestTradeBuilderIRTaxiAndRemovedWhatIfAssets`
after its existing seed-ledger context deadline expired. Its isolated race rerun
passed in 24.5 seconds. No second full root run; no full gate or GUI check.

## Blocked: matched backup/restore format

The brief requires stopping an item that needs a durable file format or recovery
shape decision. Existing per-file `.pre-vN.bak` snapshots remain unchanged; they
are not matched sets. No daily backup, retention pruning, manifest validation,
whole-set restore, or set-wide pre-migration barrier is claimed.

Options for reviewer/operator decision:

- Versioned JSON manifest inside each immutable dated set, with exactly three
  allowlisted filenames, sizes, SHA-256s, user_versions, app build and capture
  time; or a SQLite catalog with the same fields. Recommend JSON for portable
  inspection and independent restore tooling. Decide manifest version and
  rejection policy before implementation.
- Restore offline into the current fixed filenames using a durable operation
  journal and rollback directory; or adopt generation directories and atomically
  switch a current-generation pointer. Recommend offline staging/journal to avoid
  changing every database path. Three renames alone are not crash-atomic.
- A matched capture must freeze all three writers through one coordinator-owned
  boundary before any migration or snapshot starts. Sequential independent
  VACUUMs while writers remain active do not satisfy the brief. Decide the
  quiescence boundary with the pre-migration startup integration.
- Clarify weekly bucket/weekday selection and whether pre-migration or
  pre-restore safety sets are exempt from the 7-daily/4-weekly pruning policy.

## Restore drill finding

`TestBackupSnapshotDifferentHomeCommissionerRightsFinding` creates a native
league under the real local principal, snapshots all three fixture files using
`VACUUM INTO`, and opens them in another temporary home. No concurrent writers
exist in this drill; it is not the missing production restore API.

Native league discovery and rules reproduce exactly. The original commissioner
becomes **Visitor**, with `MayChangeRules=false`. Looking up the old principal in
the restored store still grants commissioner rights: data is intact, but
`localCommissionerSession.Principal` hashes dataDir and uid. The first test run
asserted retained rights and failed with precisely that Visitor reading. The
committed test records this expected finding, not a passing rights guarantee.
Identity was not redesigned. Populated native results/rosters and hostile-set
restore rejection still need the production matched-set implementation.
