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

## Consumers: inactive import and cutover

The import slice can use `(league, sourceEpoch, content)` as its idempotency identity,
store the original event fields without execution, validate in staging, and record
an inactive receipt atomically. A different content identity for an already staged
league/generation must conflict, not append a second history. Preserve the saved
source checkpoint separately from the target's own audit chain. Rebuild target schema
with target migrations and its authority/fence state with target-owned operations.
No `user_version`, authority policy, schema marker, auth epoch, seal or activation flag
exists in this document; strict decoding rejects injected ones. Trusted account links
still require the target's identity approval, not automatic enrollment.

Seal/activation and fenced reverse transfer must compare source generation, content,
stream heads and replay hashes, and ensure the exported head still equals the sealed
head. This unsealed export alone is not permission to write on either target. A source
can acquire more events immediately after export. Crash-safe staging, race-safe import
file handling, seal/abort/activate and reverse fencing belong to slices 40–41, not here.

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

These are automated local proofs, not a live cutover or Christopher's click acceptance.
The change adds a new transfer boundary; no code-size reduction is claimed.
