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

The earlier routine-check-only full root race run failed `TestTradeBuilderIRTaxiAndRemovedWhatIfAssets`
after its existing seed-ledger context deadline expired. Its isolated race rerun
passed in 24.5 seconds. No second full root run; no full gate or GUI check.

## Decided 2026-10-20

Engineering authority approved the following; the implementation does not change
identity or database filenames.

- **D1:** immutable dated directories under `backups/`, versioned JSON
  `manifest.json` (`manifest_version: 1`). Exactly three allowlisted DB entries,
  each with name, size, SHA-256 and user_version; app build, UTC RFC 3339 capture
  time and daily/pre-migration/pre-restore kind. Unknown versions, missing/extra
  members, unsafe names, mismatched bytes and newer schemas are refused.
- **D2:** offline startup restore before application pools open, using staged
  verified copies, durable `restore.journal`, a manifest-bearing pre-restore
  safety directory, fixed-name renames and directory fsync. Incomplete swaps roll
  back; durably completed swaps only need journal cleanup. The entry point is
  `internal/db.Backups.Restore`, with no desktop or transport types.
- **D3:** one backup coordinator reserves all three single writer connections in
  main → history → what-if order, then VACUUMs all three while holding every
  reservation; release is reversed. A reservation is the common write lock for
  every store sharing that file. Startup pre-migration capture runs before any
  application pool/migration/writer. Hourly maintenance captures at most one
  daily set per UTC date, including across process restarts.
- **D4:** retain seven newest daily sets, then the newest daily in each of the
  four prior ISO calendar weeks, measured before the seventh daily's ISO week.
  Keep five newest pre-migration and five newest pre-restore sets separately.
  Only verified sets may be pruned; foreign/invalid entries are reported and
  untouched. A selected restore source can be protected; the coordinator mutex
  excludes pruning throughout restore.
- **D5:** no identity redesign.

Restore into the same data dir keeps commissioner rights. A different home or OS user keeps them once the commissioner signs in with the local sign-in made before the backup (`internal/accounts/README.md` → Desktop sign-in and restored copies).

## Shipped: matched sets and offline recovery

`internal/db/backups.go`, `backup_manifest.go`, `backup_restore.go` and
`backup_retention.go` own the storage-neutral API. Production and development
filenames are separately allowlisted by configuration; schema limits come from
existing migration registries. No new schema or dependency was introduced.
`App.openDatabases` recovers and captures before opening any application pool.
The existing maintenance worker captures and prunes after its routine checks.

Restore sources must be direct children of this data directory's `backups/`;
import a set there before requesting restore. The caller must hold the same
instance lock as startup and have no application pools open. Offline checkpoint
consolidates current WAL contents before hashing and evacuating the old files;
any remaining WAL/SHM companions are evacuated as well. Normal offline safety
sets have only the three standalone DBs and manifest. Restore stages are under
the data directory, so all swapping renames stay on its filesystem.

The journal records prepared, moving, installing and complete. Failure injection
covers each state plus every database evacuation/installation. Recovery copies
old files back (rather than consuming them), making rollback restartable. Missing
or mismatched rollback files stop recovery rather than silently accepting a torn
set. Restore sources are reverified after staging; snapshots are probed with
SQLite's immutable read mode so validation cannot create WAL/SHM companions or
change the hashes.

Proof (Codex §M3/§M10):

- `TestMatchedBackupWaitsForEveryWriter`: busy third writer delays all VACUUMs;
  queued main/history writes cannot slip through. Releasing reservations early
  failed with “capture did not acquire main writer.” Same-date recapture skips.
- `TestRestoreRefusesInvalidSets` and `TestBackupManifestValidation`: unknown
  version, absent/extra files/entries, duplicate names, size/SHA mismatch, newer
  actual and declared schemas, traversal, symlink, unknown/trailing JSON. Removed
  guards accepted version/missing/name/newer/extra/size/SHA adversaries and failed
  these tests. Initial immutable-probe omission created stray WAL/SHM files and
  failed the duplicate-daily and safety-set checks; immutable probing fixed it.
- `TestRestoreRecoversEveryJournalStep`: eleven injected interruption/success
  points, new-coordinator recovery, repeat recovery and safety-set verification.
  Treating every interrupted journal as complete failed with missing marker
  tables and old rather than current values.
- `TestBackupRetentionSixtyDays`: 180 valid sets, explicit retained dates, five
  of each safety kind, protected oldest daily, foreign directory/report.
  Six-instead-of-seven retention deleted day six; deleting invalid entries
  deleted the foreign directory. Both deliberate mutations failed.
- `TestPreMigrationMatchedSetBeforeChanges` and
  `TestStartupCapturesMatchedSetBeforeOpeningPools`: all three original schema
  versions captured. Bypassing either capture or startup wiring failed with no
  pre-migration set.
- `TestDatabaseMaintenanceCreatesDailySet`: actual worker launch produces a
  daily set; another same-date wake does not duplicate it. Removing maintenance
  wiring failed with “maintenance did not publish a daily backup.”
- `TestMatchedRestoreSameHomeKeepsCommissionerAndNativeRoster`: actual local
  principal, populated native roster/rules, later league removed by restore,
  retained commissioner permission. Omitting restore left two leagues and failed.
- `history.TestMatchedRestoreKeepsPopulatedNativeResults`: final native game,
  populated roster and commissioner permission reproduce exactly after the
  matched restore. Omitting restore left a changed scoring revision and failed.
  Complete score evidence/lineup ports are fixture-injected before and after;
  this proves database replay, not external feed availability.

Verification for the matched-set build: `go test -race -count=1 ./internal/db`
and the entire `./internal/store/history` suite pass (history: 124.1 s). The
single full root race run passes (427.5 s); the known trade-builder flake did not
occur. Focused root startup/maintenance/same-home/different-home tests pass again
(7.9 s). Full `golangci-lint run ./...` and ifaceguard pass. `make lint` stops at
the existing tiny-file ratchet: HEAD and this tree both have 25 tiny files against
baseline 22; these new production files add none. The baseline was not raised.
No `make verify` or full all-package test run was performed.

Review twice: the barrier uses the already exclusive SQL writer connections,
not individual stores' in-memory publication mutexes. Restore is intentionally an
offline API contract, not an in-process live restore command. No button, live
profile access, GUI exercise or network fetch was performed.

## Restore drill finding (resolved 2026-10-24)

The original drill showed a snapshot opened in another home made the commissioner a
**Visitor**: the ledger was intact, but `localCommissionerSession.Principal` hashes
dataDir and uid. Debt sweep 2 resolved it with the existing account/alias model, not a
new identity scheme: the local sign-in links each computer's identity to one account,
and the ledger resolves linked identities through the verified-alias map.
`TestRestoreOnAnotherComputerKeepsCommissionerWithSignIn` replaces the finding test and
`TestStrangersRestoredCopyWithoutSignInStaysVisitor` keeps a copy without the password
a Visitor. See `internal/accounts/README.md` → Desktop sign-in and restored copies.
