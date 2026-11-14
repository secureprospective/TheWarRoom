# Storage, league recovery and player identity

The league is its rosters and contracts. Recovery brings those back first, checks
exact cents, years and statuses, and records what happened. A roster document
cannot prove who may operate a league: the recovered copy stays read-only. Full
server recovery remains the path to recover rules, credentials and authority.

## Files and version ownership

`db.Pools.Migrate` transactionally applies one ordered `user_version` list per
SQLite file. Main (`thewarroom.db`) remains v2. What-if (`whatif.db`) is v3: v3
deletes the trade-projection scratch generations older builds left behind. History
(`history.db`, or hosted `server.db`) is v10: v9 introduced roster recovery; v10
separates canonical identity and side-qualified ownership. Version zero is the
oldest supported unversioned installation. Empty and every intermediate migration
prefix converge; a newer file is refused before opening a writer. Old state
owner markers remain adoption evidence, not another upgrade path.

On the desktop, accounts share `history.db` and carry their own ordered list
(`accounts.SchemaVersion`, v1) applied by `db.Pools.MigrateSchema`, versioned in the
`schema_version` row named `accounts` because that file's `user_version` belongs to
history. The hosted server keeps sign-ins in their own file, `accounts.db` beside
`server.db` (`accounts.NewServerFile`, `cmd/twr-server/signin_file.go`); that file's
version is its own `user_version` (`accounts.ServerVersion`, v2) and it is refused
when newer. It holds every `account_*` table, `league_join_links`, the security audit
chain and `league_directory` (league, state, created, football). The directory is a
listing: it is written after a league's first commit and refreshed from the league file
on every start; the league file is the truth for whether a league exists and its
state, and nothing reads the directory to make a league appear. Its one reader is the
football pin: `football` (v2) is the league's pinned football evidence digest, empty
for the server's `-football` pin. Each distinct pin is served by its own
`history.Store` over the same league file (`cmd/twr-server/football_pins.go`), so
re-pinning one league (`twr-server pin-football`, read at the next start) never moves
another's.

The hosted server keeps football reference data in a third file, `reference.db`
(`cmd/twr-server/reference_file.go`, `history.ReferenceTables`): the raw archive,
fetch log, loads and observations, scoring and model runs, compiled inputs, live
lines, `canonical_player`, `player_external_id`, `player_record` and the measure
mirror. It is built with the history migration list, so its version is history's
`user_version` and a newer one is refused the same way; its league tables stay empty.
The league file is opened with it attached on every connection as schema `reference`
(`db.OpenWithReference`): a table the league file no longer holds resolves there by
name, so the stores' queries are unchanged, and no code looks for a reference table in
the league file. SQLite commits each file on its own under WAL, so no transaction
writes both: player ids are minted and committed first (slice 2's pre-step), then the
league write runs.

What crosses between league and reference, and in what form: raw archive bodies by
SHA-256 (transfer receipts, football evidence digests), measures by name, and player
identity by permanent canonical id. Identity is the one exception to "by content":
the ids are random, so the league file keeps its own `league_player` rows (history
v14) for the ids it owns, and `player_ownership` refers to them inside the league
file. Outside links and player facts stay on the reference side. A restore therefore
refuses a reference set lacking any `league_player` id.

Writes between the two files are ordered, never atomic: an account commits in the
sign-in file before the league records its seat (join, first sign-in, league-load),
so a crash between leaves an account with no seat. Every sign-in write that withdraws
a credential first reserves the league file's writer, so it cannot commit between a
league write's credential check and its commit.

A one-file server converts on first start. Each step and why a crash is safe:
(1) pre-change backup of `server.db`; (2) build `accounts.db.converting`, copy the
sign-in tables and the security chain rows as stored, prove them (row count and
ordered-row digest per table; same chain head; chain verifies), close and fsync it;
a crash in (1)-(2) leaves `server.db` untouched and the half-built file is deleted on
the next start. (3) One transaction in `server.db` removes the sign-in tables (with
`secure_delete`) and writes the `signin_move` record: the only point of no return.
(4) Rename the proven file to `accounts.db` and fsync the folder; a crash between (3)
and (4) is finished on the next start, because the record says the move happened and
the proven file is waiting. The security chain's own rows stay in `server.db` as
frozen evidence: the audit tables are append-only, and they hold hashes, not secrets.

The reference move runs next, through the same steps (`cmd/twr-server/file_move.go`):
(1) pre-change backup of `server.db`; (2) build `reference.db.converting` from the
history migrations, copy every reference table as stored (identity first, so the
evidence identity triggers mint nothing), prove each table (row count and ordered-row
digest) and the raw archive (key, size and SHA-256 of the stored bytes, in key order),
close and fsync; a crash here leaves `server.db` untouched and the half-built file is
deleted on the next start. (3) One transaction in `server.db` drops the reference
views and tables and writes the `reference_move` record: the point of no return.
(4) Rename to `reference.db`, fsync the folder, then `VACUUM` `server.db` (atomic; a
crash during it leaves the file as before it). A crash between (3) and (4) is finished
on the next start. Contradictory states (record and tables both present, record with
no file, a file with no record) refuse to start with a plain message.

v1 adopts the tables
older builds created at runtime (idempotent DDL, rows kept); a newer accounts version
is refused. The security audit chain is still verified on every start. Existing migration backups and encrypted R2 matched sets remain unchanged.

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

- Future history migrations run on both the league file and the reference file. A
  migration that changes a reference table must be written to skip the league file
  (where the table is gone), and one that changes a league table runs harmlessly on the
  reference file's empty copy. Splitting the migration list per file is the clean fix.
- The reference set pauses nothing when it falls behind; a league change that minted
  a player id during that time cannot be restored until a reference set after it is
  verified.

- A join that stops between its two commits leaves an account with no seat; that
  name stays taken, so the owner picks another or the commissioner releases the
  account by hand. No automatic retry of the seat exists.

- Rules/authority recovery from the standalone point: not small. Authority includes
  account principals, grants, transfer lineage and cutover settings; manufacturing
  it from rosters would be unsafe. Use complete server recovery; roster-only
  recovery deliberately stays fenced.
