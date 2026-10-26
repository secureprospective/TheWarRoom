# Storage, league recovery and player identity

The league is its rosters and contracts, not its historical charts. Recovery must
bring those back before attempting historical evidence. Outside player numbers
are links, never our permanent identity. MFL import remains supported.

## Database versions

SQLite `user_version`, advanced transactionally by `db.Pools.Migrate`, owns each
file. Main (`thewarroom.db`) and what-if (`whatif.db`) currently have two steps;
history (`history.db`, also the hosted server database) has eight. Version zero
is the oldest supported, unversioned installation. Existing state owner markers
are retained as adoption evidence, not a second ongoing upgrade path.

Accounts share the hosted history file: their account, session, alias, token and
audit DDL must become one history-file migration, rather than independent
`CREATE IF NOT EXISTS` initialization on every startup. The account store owns
operations; the database migration owns schema. Empty, version-zero and every
intermediate prefix must converge, and future versions must fail before writes.

## Split history by job

Today history owns fetched evidence, observations, provider links, model runs,
compiled snapshots, the native league ledger, transfer fencing and recovery
checks. These are not one responsibility. Preserve the physical file (and its
backup coverage), but extract player identity into a focused store; native
recovery should likewise own its format and persistence, with the history
adapter supplying a transactionally consistent league reading. Do not physically
move ledger tables while accounts, transfer and audit share its writer.

## Restore points and log

Intended format: versioned checked JSON, league identity, source revision,
capture time, franchise directory, complete roster and contract cents/status/
expiration, and the rule/authority information needed to restart safely. Keep
provider links and canonical IDs with the owned players. History/evidence is
optional and is not needed to decode this document.

Capture in the same transaction as every roster/contract change, not on a timer:
a scheduled copy permits acknowledged league changes to vanish. A failed capture
must roll back the change. Keep the latest point and an append-only per-league
plain-English recovery log. Backfill existing seeded leagues during migration.
The point lives inside the already encrypted-before-upload R2 SQLite backup;
existing offsite capture, retention and restoration remain authoritative. A
separately exported point is private league data and needs owner-only access.

An offline, operator-only “stand the league back up” operation validates the
point before creating a fresh destination, restores roster/contracts first,
verifies exact equality, and records what was restored and what history was not.
Never overwrite an existing league or bypass hosted credential/authority fences.
Keep its executable catalog wiring in a separate final commit, following
`command-spine.md` on the parallel command branch: metadata plus typed handler,
explicit door exposure, no automatic HTTP permission.

## Player identity

Intended tables: immutable opaque canonical player ID; unique `(provider, value)`
links to it; ownership unique on `(league, side, canonical_player)`. Empty side
means today's single pool; AFC/NFC are future independent ownership namespaces,
not franchise conferences inferred from current NFL teams. Do not implement the
mirrored league mode here.

Migration preserves external values byte-for-byte (including MFL leading zeros),
assigns canonical IDs, links every referenced player, and retains original event
bytes. Existing MFL-shaped reads need an explicit compatibility adapter until
all consumers use canonical IDs. No silent rewriting of append-only evidence.
Proof requires before/after counts on the kept big-league fixture, plus exact
roster/contract round-trip into an empty database. Neither design text nor a
passing compile constitutes that proof.

## Evidence and delivery boundary

This is the intended complete flow, not a completion claim. Each implementation
commit must name the exercised slice; unfinished recovery, identity conversion
or command exposure must be reported explicitly rather than advertised as live.
