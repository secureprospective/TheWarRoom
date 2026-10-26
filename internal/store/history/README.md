# History database

`migrations.go` owns the history-file `user_version` list. The hosted server uses
this same file for league events, football evidence and account data. The account
store still initializes its tables separately; consolidating that schema remains
unfinished.

Responsibilities currently grouped here:

- `archive.go`, `observe.go`, `features.go`: fetched bodies and measured facts.
- `modelruns.go`, `runs.go`, `inputs.go`: model runs and reproducible inputs.
- `events.go`, `commissioner.go`, `native_*.go`: league authority and replay.
- `transfer*.go`: inactive imports and fenced league transfers.
- `roster_restore.go`: checked, standalone roster/contract capture and migration
  backfill, using the existing league transaction. `internal/db/league_recovery.go`
  owns its SQL persistence; `internal/commissioner/roster_restore.go` owns the
  versioned private data format. These are not transport APIs.

History v9 saves the latest roster/contract snapshot when its contents change,
and appends a per-league plain-English log entry. Event, audit, restore point and
log commit together. Transfers capture recovery data without activating the
import. Existing whole-file encrypted R2 backups include both new tables;
no backup transport or encryption mechanism has been replaced.

**Not implemented:** rebuilding a fresh working league from a point, restoring
rules/authority, a recovery command, canonical player identity, or the broader
history package split. These points are a capture foundation, not a replacement
for the existing complete server recovery procedure in
`deploy/server/RESTORE.md`. They omit draft and historical results.

Evidence: `roster_restore_test.go` checks exact contract cents, edits/reversals,
rollback when either recovery write fails, private-document validation, migration
backfill without changing event bytes, and inactive-transfer fencing.
`migration_prefix_test.go` checks every history migration prefix against the
empty-database final schema. Main and what-if have corresponding prefix tests.
The requested fresh-database round-trip and big-league identity counts are not
yet established.
