# Transactional audit chains

The history store owns the `league_event` chain; the account store owns the
`account_security_audit` chain (recovery and agent-token audit events). Both may
share one SQLite file. No credentials or ledger bodies are copied into the chain.

## Flow and failures

- `history.insertEvent` inserts the original event, then its chain link and head,
  inside the existing caller-owned transaction. Commissioner, automatic due-work
  and generic event appends all use this function.
- `accounts.recoveryAudit` inserts the original security event, then its link and
  head, inside the recovery/token transaction. An audit failure also rolls back
  the password/generation, recovery codes or token change.
- Link or head persistence errors, including an ignored write, abort the mutation.
  Publication remains after the owner's transaction commits. Retries append no
  additional links and keep their original receipts.
- Startup verifies every original row and chain link in a single snapshot before
  admitting the store. `Store.VerifyAudit` repeats that check and returns the
  verified checkpoint that `twr-server checkpoint` saves away from the server. It is not an HTTP
  endpoint and must not be exposed to owners: global sizes reveal hidden activity.
- The append path checks the local tip/head before extending it. Full source-row
  verification runs at startup or when `VerifyAudit` is called, not on every read.

## Adoption

History migration 5 automatically adopts the existing ledger in one transaction.
Account initialization similarly adopts existing security audit rows transactionally.
Their existing contents are the trusted baseline: this cannot establish whether
someone changed them **before** adoption. No command is re-executed, payload
upcast persisted, identity rewritten or receipt replaced. `baseline` records the
number of adopted rows; later startup verifies rather than rebuilding the chain.
Migration failure rolls back adoption. No manual hashing or setup is required.

## Version 1 format

Each source has independent contiguous chain positions starting at 1. The source
names are `league_event` and `account_security_audit`. Keys are the original event
ID or decimal security sequence. Payloads are SQLite `json_array` encodings of all
original columns, in the fixed order in the store's `*AuditColumns` query, before
any application decoding or upcasting. Ledger ordering is insertion (`rowid`);
security ordering is its sequence.

`digest = lowercase_hex(SHA256(UTF8(payload)))`.
`hash = lowercase_hex(SHA256(encoding/json.Marshal([]string{
"twr-audit-v1", source, decimal_position, key, digest, previous_hash})))`.
The first previous hash is empty. The empty chain has size 0 and an empty hash.
The golden vector in `db.TestAuditHashV1` locks this framing and domain separation.

The verified JSON checkpoint has `version:1`, `source`, `size`, `hash`, and
`baseline`. `twr-server checkpoint` writes both heads with a capture time;
see Saved checkpoints below.

## Limits and evidence

Local source edits/deletions and missing chain links or tails fail verification,
including simultaneous source/chain tail deletion while the head remains. Ordinary
SQL updates/deletes of audit rows and chain links are blocked by triggers.
A file administrator can rewrite the data **and** all matching local integrity
metadata, or restore a whole older database. Local hashes cannot detect that;
the saved checkpoints below detect that, back to the newest saved checkpoint; changes
after it are not covered. Hashes cannot recover lost events or authenticate pre-adoption
history. There is no claim against a running/root compromise.

`TestLedgerAuditDetectsTampering`, `TestSecurityAuditDetectsTampering`, the two
`*FailureRollsBack*` tests, and the trusted-baseline tests exercise these guarantees.
Existing retry/reopen, concurrency, recovery and token tests cover the unchanged
owners of the transactions. Deployed-host drills remain
outside this slice.

## Saved checkpoints (slice 33)

Ruling 2026-10-25: TheWarRoom is a recreational league, secured only enough not to be the
weak point. No extra server, transparency log or witness service. Checkpoints are copies of
both chain heads kept away from the server, where a server administrator cannot reach them:
with each off-site backup (slice 36) and on the commissioner's desktop (transfer slices 39–41).

- `twr-server checkpoint -db <server.db>` prints `{version, captured, ledger, security}` after
  fully verifying both chains. It only reads: no migration, initialization or listener, and it
  refuses a missing database instead of creating one.
- `twr-server checkpoint -db <server.db> -check <file>` re-verifies both chains and checks that
  each saved head is still a prefix of the current chain (`db.CheckAuditAnchor`): same source and
  baseline, size not past the current tip, and the same hash at that position.
- A coherent rewrite (data plus every later link and the head) or a rollback below a saved
  checkpoint passes local verification but fails this check
  (`TestSavedCheckpointDetectsCoherentRewriteAndRollback`).

Limits: anything changed after the newest saved checkpoint is not covered, and a checkpoint
copy proves nothing if the same person can rewrite it too. Sol's off-host Tessera/OmniWitness
design is kept unmerged on `session/cs33-tessera-shelved` for reference.
