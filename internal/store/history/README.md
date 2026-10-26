# Shared history file

`migrations.go` owns the history-file `user_version` list (v10). Hosted `server.db`
uses the same list. Main and what-if are independently v2. Account schema adoption
remains separate and is explicit debt, not completed consolidation.

## Jobs and owners

- History retains fetched evidence/observations, model runs/inputs, snapshots,
  native ledger replay and fenced transfers. Original evidence is append-only.
- `internal/store/playeridentity` independently owns canonical IDs, provider links
  and side-qualified ownership. V10 migrates the old player map into these tables;
  the `player_ids` view is an explicit MFL compatibility adapter, not another map.
- `internal/leaguerecovery` owns offline request/destination handling.
  `internal/commissioner/roster_restore.go` owns the checked document;
  `internal/db/league_recovery.go` owns recovery tables/log SQL.
- `roster_restore.go` captures points inside native transactions; `stand_up.go`
  rebuilds and checks native rosters/contracts, then fences the result read-only.

All jobs stay in the physical file for shared atomicity and existing encrypted R2
coverage. This does not claim that evidence, models and ledger became separate
packages. Stores do not import sibling stores.

Every roster/contract change, its audit entry, restore point, permanent ownership
and recovery log commit together. Unchanged points add no log noise. Inactive
imports also retain points without gaining authority. New transfer documents
preserve canonical identities; conflicts fail rather than silently merge people.

## Recovery

Use the offline `twr-server stand-up` command described in `cmd/twr-server/README.md`.
Its source can be a decrypted R2 server.db. It creates a new private folder, never
overwrites one, verifies exact roster/contract equality and logs the result.
The recovered copy is **read-only**. It does not reconstruct rules, accounts,
credentials or authority; use `deploy/server/RESTORE.md` for full server recovery.

Proof and the two justified deferrals are recorded in
`docs/architecture/storage-recovery-identity.md`. Tests include every migration
prefix, atomic-capture rollback, big-league migration counts/unchanged bytes,
roster round trips through encrypted offsite backups, immutable identities,
provider-link collision rollback, and side-qualified ownership uniqueness.
