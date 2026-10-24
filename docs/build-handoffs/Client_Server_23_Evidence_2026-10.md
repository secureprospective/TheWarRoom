# Client/server 23 — recipient concurrency

## Boundary and finding

A blind bid was omitted from its competitor's rows but still advanced the global
revision in bid readings, other hosted readings and conflict messages. Request-key
collisions across owners also revealed otherwise hidden activity through retry errors.

Trusted hosted identity → request-local recipient context → authorized reading over
one history snapshot → opaque equality token. Preview and append recompute that
recipient's token against their transaction snapshot, after credential/domain checks.
Invisible changes do not conflict; visible changes require a reread. Append still
uses the actual contiguous ledger revision, not the token, for SQLite ordering.
Canonical requests retain the submitted Expected; stored receipts retain actual
revisions. Replies and retries project an original-prefix recipient token instead.

## Implementation and preserved contracts

- `internal/commissioner/view_revision.go` hashes visible entry identities, league
  and resolved principal. Global positions and canonical payloads never enter it.
  The numeric token fits JavaScript's exact integer range; it must not be ordered,
  incremented or interpreted as an event count.
- Readings share a recipient-visible concurrency scope, rather than independent
  screen tokens, so an existing reading can still supply a command's Expected.
  This scope includes the recipient's own records/compensations and open bids;
  a commissioner still cannot observe another owner's undecided blind bids.
- `internal/leagueservice/commissioner_api.go` attaches the trusted context at the
  reading/command boundary. No shared startup/root wiring or generated bindings changed.
- History read adapters replace only the outgoing revision, using the same records
  as the body. Nothing recipient-specific is retained in the shared projection cache.
- `internal/store/history/view_revision.go` projects receipt tokens and namespaces
  hosted request keys per canonical principal. Other actors' request keys cannot
  collide with hosted retries or inserts; durable command bodies are not rewritten.
- Existing trusted local revision callers remain compatible. Local bid readings now
  return safe tokens; local bid appends also accept the older ledger-revision input.
  Hosted appends never accept that compatibility fallback.

## Evidence

`internal/leagueservice/view_revision_test.go` exercises:

- `TestHiddenBidPreservesRecipientBodiesTokensAndErrors`: full JSON equality across
  17 owner readings and the commissioner ledger after a blind bid; equality after
  hidden raise/withdraw, stale errors and previews, and zero refused appends.
  An unchanged valid token still executes after hidden activity; retries append once.
- `TestOpenBidChangesRecipientTokenAndConflictsWithoutAppend`: visible bid changes
  the token, stale execution appends nothing, and the fresh token succeeds.
- `TestHiddenBidRequestKeyCannotAffectAnotherRecipient`: identical request strings
  from two actors produce independent entries and exact own retry receipts.
- `TestRecipientTokensSurviveColdReplayWithoutChangingHistory`: recipient scope,
  cold/warm response equality and unchanged serialized durable events.

Disabling only read-adapter token normalization made the hidden-bid test fail at
its unauthorized response/token assertion; restoring it passed the race test batch.
Targeted commissioner/history/service/HTTPS/binary race tests passed during build.
The full Go run found two stale bid goldens; regenerating those tests changed only
those two revision values. The final `go test ./...` passed. The final targeted
race batch passed for commissioner, history, service, HTTPS, binary and both
composition wire tests. No TypeScript binding edits were needed.

Claude reviewed supplied core source/caller excerpts at `a8b29c77`: no blockers
within those excerpts, not a full-worktree review. Its initial file-based review
could not run because the child had no tools. Parent source trace confirmed the
hosted StoredHost identity is immutable, read/write clocks use server time, dry-run
receipts bypass receipt projection, and own-record compensations intentionally
invalidate the affected owner's broader view even if a filtered ledger omits them.

Affected handwritten production Go: 2,904 → 3,065 physical lines against `63797ac3`;
counted with splitlines, excluding tests/goldens/docs. This is a safety addition,
not a debloat claim. No new package or placement/map entry was needed.

## Limits and plan correction

No deployment, timing-side-channel proof, three-OS UI drill, new 100k-event benchmark
or SSE proof is claimed. SSE remains slice 24. Full adversarial role/family coverage
remains slice 50. Legacy non-hosted local views are not the new hosted boundary.
The plan's bid-ledger-count example was not exhaustive: other hosted revision fields,
conflict messages and cross-owner request-key collisions needed the same protection.
No binding decision changed.
