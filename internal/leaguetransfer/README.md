# League transfer

Exports one league for moving to another computer. An export alone does **not**
seal the source, import anything, grant a role, or activate a server. Keep using only the source until
the confirmed cutover workflow below says otherwise. This is not an off-site server
backup: passwords, sessions, recovery codes, server configuration and other leagues
are deliberately excluded.

## Commissioner commands

Run locally, with access to the existing league database. No network token is used.
Keep the export key separate from the exported file; both contain private league data
(including hidden bids). Never paste the key into command arguments, logs or a PR.

```sh
umask 077
openssl rand -hex 32 > move.key
# Generate ONE UUID for this source generation and keep it throughout this move.
# A later reverse transfer needs the new source generation, not this old UUID.
twr league-export save --db /path/to/history.db --league native.example \
  --source-generation YOUR-SOURCE-GENERATION-UUID \
  --football CURRENT-PINNED-FOOTBALL-SHA256 \
  --key-file move.key --output league.twr
# Add --past-football SHA256,SHA256 for earlier seasons in the ledger.
twr league-export check --file league.twr --key-file move.key
```

The source must already have self-contained pinned football imports for its seasons.
These include provenance, player identity mappings, NFL schedule, weekly statistics,
play-by-play and stat revisions, not links to archives that might disappear. Missing,
changed, invalid or wrong-season evidence refuses export. Existing destination files
are never overwritten. A failed write removes the incomplete destination. Files are
0600; the key file must be private too. The CLI refuses missing databases and does not
initialize or migrate them. It holds the single writer reservation (the same ordering
principle as matched-set backups) across reading immutable account aliases and the
history/evidence snapshot. Release that reservation before accepting normal writes.

## Boundary and format contract, version 1

| Responsibility | Input → output; effects and failure |
| --- | --- |
| `cmd/twr/transfer.go` | Local console + private key file → exclusive 0600 encrypted file. Reserves source writer; no league/auth/schema writes. Reads verified aliases through their owning account store; unrelated aliases are omitted. |
| `history.Store.CaptureTransfer` | One read transaction → verified source audit head, original SQL event text and selected evidence bodies. Uses existing `db.VerifyAudit` and bounded `archive.Gunzip`; failed verification yields no export. Does not copy database files or SQLite metadata. |
| `leaguetransfer.Capture` / `RawExport.Validate` | Versioned events + pinned bodies → content identity and six projection hashes. Validates raw records/receipts/order/aliases/body hashes; replays using existing commissioner and history scoring code. No command execution or target storage. |
| `Encrypt` / `Decrypt` | Validated document ↔ authenticated envelope. Whole-file authentication precedes decoding; unknown fields/versions, truncation, tampering and oversize refuse. |

`TWR-LEAGUE-XCHACHA20POLY1305-v1\n` + random 24-byte nonce + AEAD ciphertext/tag.
The exact header is associated data. The plaintext is one strict JSON `RawExport`,
limited to 64 MiB (100,000 events / 100 season bundles, each subject to existing 8 MiB
football-import validation). Encryption uses the **existing**, maintained
`golang.org/x/crypto/chacha20poly1305.NewX`, not a new cipher/framing scheme with
independently authenticated chunks. Its documented random-nonce recommendation is
why XChaCha20-Poly1305 was selected over a short-nonce construction. The key is 32
random bytes, hex-encoded only in the private key file. This is a bounded league
artifact, not a streaming backup protocol. Length is not hidden. Key custody is the
operator's responsibility; shared-key authentication does not prove who exported it.
Research reference: the module's `go doc .../chacha20poly1305.NewX` and package docs
([maintained upstream](https://pkg.go.dev/golang.org/x/crypto/chacha20poly1305)).

JSON fields:
- `version`, `league`, `sourceEpoch` (opaque canonical UUID), `frozenAt` (UTC).
  The initial generation is explicit operator input; later moves use the durably
  recorded generation. It is **not** a schema version, login epoch or activation proof.
- `events`: all league streams, ordered by stream/revision. `RawTransferEvent` keeps
  original payload and timestamp **strings**, IDs, actors, request correlation IDs,
  input hashes and nullable supersedes intact. No durable upcast, canonical rewrite,
  renumbering, command re-execution or receipt replacement. Unknown newer commands refuse.
- `football`: current-season bundle digest; `blobs`: SHA-256 → exact uncompressed
  evidence bytes (JSON base64 encoding). One bundle per included season. Evidence for
  every season from league creation through the frozen current season is required.
- `aliases`: only relevant, verified league-scoped legacy → account mappings. They
  confer no target login or role by themselves; no account credential data travels.
- `checkpoint`: verified **source-wide ledger** `db.AuditHead`, including its trusted
  legacy baseline. Keep it on the commissioner desktop to compare against that source
  with `VerifyAuditAnchor`; it is not a target chain head. As in saved-checkpoint
  recovery, changes after the saved head are not covered by this checkpoint.
- `projection`: roster, cap, lineup, bid, result and standings SHA-256 hashes of
  existing engine readings at `frozenAt`, UTC. All owners' bids and all franchises'
  lineups for held schedule weeks are included. All 22 weeks of season results, plus standings, use the
  corresponding self-contained bundle. Aliases resolve only in a disposable policy
  view. Evidence bundles are ordered by digest before hashing. These hashes are
  comparison evidence, **never** persisted snapshots or target authority.
- `content`: SHA-256 of Go's compact JSON encoding of the entire typed document with
  `content` set to the empty string (sorted map keys). Includes the epoch, checkpoint,
  evidence and projection digests. Preserve this v1 encoding for retry identity.

## Inactive import

```sh
twr league-import --db /path/to/target/history.db --file league.twr --key-file move.key
```

Use an existing destination database prepared by the current application. The command
never creates or migrates a database. It opens the encrypted input once, checks that
the opened descriptor is a regular file, authenticates the bounded bytes and validates
all six replay hashes before acquiring the target writer. No temporary plaintext file,
path-based check followed by a second archive open, network call or account grant occurs.
Filesystem access to the destination is required; this is not a remote upload endpoint.

| Responsibility | Input → output; effects and failure |
| --- | --- |
| `cmd/twr/transfer_import.go` | Local file + private key + existing database → plain inactive receipt. One archive descriptor; missing destinations refuse without creation. |
| `leaguetransfer.Import` | Authenticated `RawExport.Validate` → immutable comparison document and validated history-store input. Authentication/domain/replay failures perform no destination writes. |
| `history.Store.StageTransfer` | Immediate writer transaction → original event text, pinned archive bodies, target audit entries and one append-only `league_import` receipt/document. No command execution, authority policy or account enrollment. Any insert failure rolls everything back. |
| `stageArchive` | Validated body → insert-or-reuse, then decompress and compare the winning row under that same transaction. Concurrent archive writers cannot substitute unchecked bytes under an existing digest. Damaged/mismatched held bodies refuse the entire import. |
| Database import guards / `NativeLeagues` | Committed import → excluded from normal league lists and due-worker discovery; SQL triggers refuse every further event append or authority-policy write, including independent stores and direct SQL. Historical reads remain available. |

`(league, sourceEpoch, content)` identifies a retry. Repeating it returns the original
receipt, including its original timestamp, across restart and independent writers.
Any different content or generation for that league conflicts. Existing destination
history or authority settings also conflict; unrelated leagues and valid shared archive
bodies remain unchanged. SQLite's immediate writer reservation owns the collision check,
archive guard, event/audit inserts and receipt commit together, not a process-local mutex.
No observer can see a partial imported league. A crash before commit leaves no import;
a crash after commit is recovered by the same idempotent retry.

The saved comparison document retains aliases, source checkpoint, stream heads (in
original events) and six replay hashes for cutover. The target audit is built separately
from the exact imported SQL bytes; the source-wide checkpoint is never installed as its
head. The receipt/document is immutable. Target-owned activation/fence state evolves separately;
no activation permission is accepted from the export. See confirmed cutover below.

## Evidence

- `TestEncryptedExportRoundtripFrozenReplayParity`: encrypted roundtrip preserves raw
  records, mappings and replay; compares roster/cap/lineups/private bids/results/standings
  to direct store readings at a fixed clock, including a real active bid and final game.
- `TestExportRejectsMissingBlobBadDigestAndReplayMismatch`: missing/changed evidence,
  stream gaps/duplicates, wrong league, future command/format, changed receipts, bad
  aliases and each of six altered projection digests refuse, even after content rehash.
- `TestEncryptedExportRejectsWrongKeyTamperingTruncationAndVersion`: authentication and
  framing failures do not yield a readable league.
- `TestExportCarriesNoSchemaOrWritableAuthority`: source authority/schema unchanged;
  schema/activation injection refused.
- `TestCaptureRejectsMissingSelectedEvidenceAndDamagedAudit`: source integrity fails closed.
- CLI tests save/check, private file permissions, overwrite refusal and missing DB refusal.

- `TestInactiveImportPreservesHistoryAndReplayAcrossRestart`: exact original event and
  evidence bytes, target audit, six imported replay digests, immutable comparison document,
  write fences and original retry receipt after restart; unrelated active league preserved.
- `TestImportConcurrentRetryAndConflictingContent`: independent stores return one receipt;
  altered content or generation refuses without replacing the original artifact.
- `TestImportRollbackOnEventFailureAndArchiveCollision`: injected event/receipt failure,
  poisoned archives, occupied destination and wrong key leave no partial import.
- `TestArchiveGuardChecksWinnerUnderIndependentWriter`: an independent transaction's
  poisoned archive winner is checked after acquiring the writer, and import refuses.
- `TestInactiveImportCrashRecovery`: subprocess exits without rollback after event/audit
  insertion and after receipt insertion, with dirty WAL pages; reopen recovers no partial
  events, archives, receipt or audit. Uses the same transaction insert helpers as import.
- `TestLeagueImportCLIInactiveRetryAndMissingDestination`: terminal save/import/retry,
  stable inactive receipt, normal league list exclusion and missing DB refusal.

These are automated local proofs, not a live cutover or Christopher's click acceptance.
Review: the missing destination boundary is now an authenticated replay followed by one
history-owned writer transaction. Existing source authority, target accounts, unrelated
leagues, durable event bytes and target audit ownership are preserved. Source-traced callers
are the local CLI and transfer service; normal app/due-worker discovery uses `NativeLeagues`.
Removing its inactive filter deliberately failed the CLI integration test; restoring it passed.
Across the complete affected handwritten production set (CLI root/import, transfer service,
history migrations/commissioner/import), physical lines increased from 1,255 to 1,508 against
the slice's starting commit, excluding tests/docs/generated files. This adds a required boundary;
no code-size reduction is claimed. Live hosted cutover remains unverified; the confirmed local cutover below is automated-tested.

## Commissioner-confirmed cutover and reverse

`league-move` runs locally with filesystem custody of **both existing databases**.
It is not a network activation endpoint. Select the two paths explicitly; never
copy a live database to make a destination (that also copies its identity).
Prepare/migrate both databases through the application first. Import the encrypted
file inactive, then use that same file for each confirmed action:

```sh
twr league-move seal --source-db SOURCE --target-db TARGET --file league.twr \
  --key-file move.key --commissioner CURRENT-COMMISSIONER --confirm
twr league-move activate --source-db SOURCE --target-db TARGET --file league.twr \
  --key-file move.key --commissioner CURRENT-COMMISSIONER --confirm
# Instead of activation, cancel and resume the source:
twr league-move abort --source-db SOURCE --target-db TARGET --file league.twr \
  --key-file move.key --commissioner CURRENT-COMMISSIONER --confirm
```

Current source governance must recognize the commissioner; co-commissioners and
owners cannot move the league. The principal flag is a local operator assertion,
not remote authentication. No role/account is granted on the destination. Target
account linking remains a separate, target-owned approval. Never expose this local
operation directly over HTTP with an untrusted principal argument.

| Owner / boundary | Contract and effects |
| --- | --- |
| `leaguetransfer.Move` | Validate/replay artifact, require confirmation and source commissioner; bind to distinct destination database identity. Select seal, activate or abort explicitly. |
| `SealTransfer` | Immediate source writer transaction compares **all exact event rows**, generation and content before committing the fence. A stale export refuses. Concurrent app/worker/direct SQL appends either precede the comparison (and invalidate it) or encounter the committed fence. |
| `DecideTransfer` | Hold source writer while committing destination decision, then append source outcome. Serializes abort and activation across stores/processes; does not rely on a process mutex. Source remains fenced if destination commits and source crashes. |
| `DecideDestination` | Match database identity and inactive receipt, reject cancelled moves; activation atomically receives evidence/history and records new writable generation. Abort first records durable target cancellation, then source resumes. |
| Reverse | Export from the **current** authority using its `TransferFence.Epoch`, seal that authority, activate the former source with the same confirmed operation. Former source must be `moved` from that generation, with byte-identical held history. Only missing original events are appended with target-owned audit entries, in one transaction. No old snapshot restoration or server-event loss. |
| Discovery/status | SQL event and policy guards enforce the current fence. `NativeLeagues` excludes non-writable copies; its cache includes authority transitions, not only event additions. `TransferStatus` provides plain owner wording and an explicit writable bit for service/desktop consumers; errors are not permission. |

Ordering is `source sealed → target active → source moved`, or
`source sealed → target cancelled → source active`. A crash can leave **zero**
writable copies, never two. Repeat the same action with the original file/paths.
If activation committed but its reply was lost, abort refuses; activation retry
finishes source retirement. A cancelled move cannot later activate. A fresh seal
gets a new move identity, even if the same comparison file is reused.

The generation on activation is the move UUID, not the import's old source epoch.
For reverse, omit `--source-generation` on `league-export save`: the CLI selects the
stored current generation. Embedding callers use `TransferFence.Epoch` with Capture. The immutable original import receipt
and comparison document are retained; authority transitions are separate append-only
history. Owner labels deliberately contain neither these IDs nor database paths.
No remote two-host transport, new owner screen, or live hosted cutover is claimed;
those callers must preserve the source-locked decision contract, not issue an
independent portable activation token. Loss of the source during a pending decision
fails closed and needs recovery, not unilateral target activation.

Automated evidence: `TestMoveCrashBoundaries` exits subprocesses before/after seal,
before target commit, after target activation/cancellation but before source commit,
and after source retirement/resumption; reopens both databases and retries recovery.
`TestMoveSealAbortAndActivation` exercises confirmation/roles, SQL write rejection,
abort, retry and post-activation abort refusal. `TestMoveActivationRacesAbort` proves
one winning decision. `TestReverseMoveIncludesEveryServerEvent` adds a real server
command, reverses and compares every original SQL event byte and discovery status.
`TestMoveRejectsStaleExportAndWrongDestination` rejects stale heads and a second target.
These are local automated checks, not a physical host-loss or desktop click drill.

Research/ordering: this extends the repository's existing SQLite immediate-writer
transactions (`db.Pools`, commissioner execution, inactive imports), not a new lease
or home-grown distributed consensus scheme. SQLite atomic commit and single-writer
ordering are the underlying guarantees ([transactions](https://sqlite.org/lang_transaction.html),
[atomic commit](https://sqlite.org/atomiccommit.html)). Source checkpoints are checked
with existing `db.VerifyAudit` / `db.CheckAuditAnchor` under the seal writer; unrelated
source events after the checkpoint are permitted, but any changed league row refuses.

Slice review: source-traced CLI → Move → source writer → destination writer → source
outcome, including both crash recovery callbacks. No independently usable activation
proof or network principal endpoint was introduced. Placement stays in the existing
transfer/history/service packages and existing commissioner Host adapter. Across the
complete affected handwritten production set, physical lines rose from 1,793 to 2,435
(excluding tests/docs/generated files); the added responsibilities are durable source
fencing, destination decision/reverse receive and commissioner orchestration. No size
reduction is claimed. Removing the source event guard deliberately failed
`TestMoveSealAbortAndActivation` on a changed persisted head; the guard was restored.
Reverse abort/reseal and injected reverse insertion failure are also covered; failed
receive preserves both fences and the original destination history before retry.

Import retries report the **current** location: repeating the original import after
activation returns the immutable receipt without incorrectly calling the active league
inactive. The CLI cutover/reverse test checks this alongside the stored-generation export.

## Permanent player identity

New exports include authenticated canonical player identities and outside-ID links
for current and historical league players. These are captured in the same source
read transaction as events and evidence. Old exports omit the optional field and
remain readable. Import and reverse transfer preserve matching IDs; conflicting
permanent identities at a destination fail without reassignment. Use a fresh
destination rather than guessing that two different permanent IDs are one person.
