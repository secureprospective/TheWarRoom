# Storage, league recovery and player identity

The league is its rosters and contracts. Recovery brings those back first, checks
exact cents, years and statuses, and records what happened. A roster document
cannot prove who may operate a league: the recovered copy stays read-only. Full
server recovery remains the path to recover rules, credentials and authority.

## Files and version ownership

`db.Pools.Migrate` transactionally applies one ordered `user_version` list per
SQLite file. Main (`thewarroom.db`) and what-if (`whatif.db`) remain v2. History
(`history.db`, or hosted `server.db`) is v10: v9 introduced roster recovery; v10
separates canonical identity and side-qualified ownership. Version zero is the
oldest supported unversioned installation. Empty and every intermediate migration
prefix converge; a newer file is refused before opening a writer. Old state
owner markers remain adoption evidence, not another upgrade path.

Accounts still initialize their schema independently in the shared hosted file.
Their consolidation is explicitly deferred below, not a second completed version
plan. Existing migration backups and encrypted R2 matched sets remain unchanged.

## History split: old jobs → current pieces

The old history store grouped fetched bodies, observations, provider mappings,
model runs, compiled snapshots, the native ledger, transfer fencing and checks.
The physical file stays shared so its one writer and backup coverage remain intact.

| Job | Current owner / migration |
| --- | --- |
| Fetched evidence, observations, model runs, snapshots, native ledger and fenced transfers | Existing history adapter; original append-only bytes retained |
| Permanent player identity, provider links, ownership keys | Independent `internal/store/playeridentity`; v10 replaces mutable `player_ids` with canonical tables and an explicit read-only MFL compatibility view |
| Offline recovery orchestration and exclusive destination handling | `internal/leaguerecovery`; no listener or hosted command exposure |
| Private recovery format and native replay | `commissioner.RawRosterRestorePoint`; history supplies transaction-consistent capture/rebuild |
| Recovery persistence and per-league log | `internal/db/league_recovery.go`, v9; captures share the league transaction |

This is a responsibility split, not separate database files. Evidence/model/ledger
APIs remain on history; they have not been advertised as three extracted stores.
Stores never import sibling stores. Recovery composes storage from outside them.

## Capture and stand-up flow

Every roster/contract change captures a validated v1 JSON point in the same
transaction as its ledger/audit write. A failed capture rolls back the change.
Unchanged state does not append recovery-log noise. The point contains season,
franchises, divisions, rosters, exact contracts, adjustments, source revision/time
and owned players' permanent IDs/provider links. It omits draft and results
history. Migration backfills existing leagues, including inactive imports.

`local console → checked request → read-only source snapshot → Raw point Validate
→ exclusive 0700 destination → new history.db → native create/seed → exact roster
comparison → read-only import fence + plain-English log → commit/close`

The source is never migrated. Damaged/missing/newer points fail before creating a
destination. Existing destination folders are never overwritten; failed recovery
removes only the new private folder. The point itself grants no accounts, roles,
cutover policy or transfer activation. Do not serve this copy as a writable league.

The source can be desktop history.db or `server.db` decrypted by existing R2
recovery. Encryption, upload, digest verification and decryption are exercised
through the existing offsite implementation and a local S3-compatible endpoint;
no live R2 credentials or network were used. No backup scheme was replaced.

## Canonical identity and ownership

Opaque immutable `player.<128-bit random hex>` IDs are independent of providers.
`player_external_id` uniquely maps `(provider,value)` to a canonical player; MFL,
gsis, sleeper, espn, pfr and future providers are links. Provider bytes, including
leading zeros, are preserved. New identities can exist without an MFL link.

V10 links all legacy fact/score/player-map and ledger player references without
rewriting evidence. New observations/scores maintain their MFL links. History's
existing MFL-shaped APIs use an explicit compatibility adapter; rebuildable main
and what-if caches continue to speak MFL. Ownership is unique on
`(league,side,canonical_player)`; empty side is today's pool. AFC/NFC keys can each
own a player once, but no mirrored-league mode or conference inference was built.

Restore points preserve owned canonical IDs. Transfer documents authenticate
current and historical player identities in the same capture; import/reverse
moves accept matching links but refuse conflicting identities instead of guessing.
Older transfer documents remain readable with the absent optional field.

## Exercised evidence

- Table-driven direct/SQLite-snapshot recovery: Active, IR, Taxi; exact cents,
  years, statuses, directory and adjustments; overwrite refusal and read-only fence.
- Encrypted R2-compatible capture/upload/verify/decrypt → point → fresh database:
  complete roster/contracts and canonical identities equal.
- Kept big-league fixture SHA-256
  `9597656ccd1f0eaf446ce70a31180aa8408cb240f8a9b6a46a18a50a0cf3b62f`:
  7,398 events and 640 legacy provider links unchanged; 2,825 canonical players,
  3,465 outside links, 2,486 ownership rows. All 2,486 roster/contracts and owned
  canonical identities survive a fresh-database round trip. Event bytes unchanged.
- Every migration prefix; outside-link collisions/rollback; independent identities
  without MFL; side-qualified ownership uniqueness; offline destination safety.

## Debt left

- Rules/authority recovery from the standalone point: not small. Authority includes
  account principals, grants, transfer lineage and cutover settings; manufacturing
  it from rosters would be unsafe. Use complete server recovery; roster-only
  recovery deliberately stays fenced.
- Account-schema consolidation: not small. Hosted and standalone local-seat/account
  bootstraps currently share unversioned account initialization; safely adopting
  credentials, auth epochs and audit chains needs its own migration and compatibility
  tests. No credential/session schema was changed in this branch.
