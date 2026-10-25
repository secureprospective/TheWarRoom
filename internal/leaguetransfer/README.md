# League transfer

Exports one league for moving to another computer. It does **not** seal the source,
import anything, grant a role, or activate a server. Keep using only the source until
the later cutover workflow explicitly says otherwise. This is not an off-site server
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
  The generation is explicit operator input until the seal/activate slice owns its
  durable lifecycle. It is **not** a schema version, login epoch or activation proof.
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
head. The receipt/document is immutable and currently **always inactive**. Slice 41 must
add target-owned activation/fence state and evolve the SQL guards and league discovery
in one migration; it must not remove or rewrite this receipt. No activation operation
exists yet. No `user_version`, authority policy, auth epoch, seal or activation flag is
accepted from the document. Trusted account links still require the target's identity
approval, not automatic enrollment.

Seal/activation and fenced reverse transfer must compare source generation, content,
stream heads and replay hashes, and ensure the exported head still equals the sealed
head. This unsealed export alone is not permission to write on either target. A source
can acquire more events immediately after export. Source sealing, abort/activation and
reverse fencing belong to the cutover slice, not this import operation.

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
no code-size reduction is claimed. Live source sealing and activation remain unverified and
unimplemented here by design.
