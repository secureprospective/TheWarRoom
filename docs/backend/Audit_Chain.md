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
  verified checkpoint for the next slice's anchor exporter. It is not an HTTP
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
`baseline`. An off-host writer can append this checkpoint as a Tessera leaf and
have OmniWitness witness it without changing the chain format. The off-host log
must also bind the deployment identity; source names alone do not identify a host.
No Tessera/OmniWitness deployment or network operation is included here.

## Limits and evidence

Local source edits/deletions and missing chain links or tails fail verification,
including simultaneous source/chain tail deletion while the head remains. Ordinary
SQL updates/deletes of audit rows and chain links are blocked by triggers.
A file administrator can rewrite the data **and** all matching local integrity
metadata, or restore a whole older database. Local hashes cannot detect that;
slice 33's independently held anchors must detect rollback/rewriting and document
the unanchored tail. Hashes cannot recover lost events or authenticate pre-adoption
history. There is no claim against a running/root compromise.

`TestLedgerAuditDetectsTampering`, `TestSecurityAuditDetectsTampering`, the two
`*FailureRollsBack*` tests, and the trusted-baseline tests exercise these guarantees.
Existing retry/reopen, concurrency, recovery and token tests cover the unchanged
owners of the transactions. Off-host anchoring and deployed-host drills remain
outside this slice.
