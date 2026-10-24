# Client/server 13: verified league identities

## Intended and implemented flow

Trusted account verification → `governance.NewPrincipals` (immutable, one league) →
`commissioner.WithPrincipals` request context → validated history records → private
identity binding → canonical governance fold, owner/private-read comparisons and retry matching.

The map grants no roles. Conflicting aliases, account-to-account aliases, malformed account IDs
and mappings supplied for another league fail closed. An unmapped identity keeps its own grants,
not those of a similarly named account. Local calls without a mapping retain their prior behavior.
Neither a command nor a request JSON body can supply this dependency.

`history.transactionRecords` binds only after reading/copying the decoded cache; the service's
`Records` adapter binds its independently decoded reads too. `Record` carries only an unexported,
immutable policy dependency. The governance fold canonicalizes private copies of authors and people
arguments. Ledger authors, canonical request bodies, receipts, event actors, hashes and payloads
are not rewritten. Yes/no ballots are not candidate identities.

Private bid/waiver claims, private lineups, franchise ownership and draft retries use the same
identity relation. Trade DOT comparisons also recognize earlier votes under verified aliases.
Ordinary retries still require current credential admission and domain permission, then compare
the complete request using the stored author's principal. A different body or an unverified author
cannot claim the original receipt.

Closed-book checkpoints retain legacy identities, so an alias-bound request deliberately uses
authoritative replay instead. Original checkpoints and the shared decoded cache stay untouched.
This preserves correctness, not a claim of alias-aware checkpoint performance.

## Evidence

- `TestAccountRetriesEveryVerifiedAliasWithoutChangingHistory`: real store writes under two
  historical principals; account retries return the original receipts; changed bodies,
  unverified authors and revoked credentials refuse. Raw SQL actor/payload/hash bytes stay identical.
  Replacing alias-aware retry equality with literal equality made this test fail; restored afterward.
- `TestAliasGrantsOwnershipPrivacyAndRevocation`: grant/seat/private reads agree; wrong franchise,
  unrelated identity and cross-league mappings refuse; account-addressed revocation removes the
  legacy grant. Reads/refusals preserve stored bytes.
- `TestAliasPrivateOwnerReadingsAndDraftRetry`,
  `TestAliasesReplayAcrossClosedBooksWithoutRewritingThem`,
  `TestAliasResolutionDoesNotChangeYesNoBallots`: owner-private predicates, draft retry,
  checkpoint fallback and ballot semantics.
- `TestServiceRecordReadsResolveVerifiedAliases`: independent service decoding retains original
  authors while applying verified grants and rejecting another league's mapping.
- Governance tests exercise canonical UUIDs, conflicting mappings and caller-input isolation.

## Boundaries and limits

This slice supplies the trusted identity contract, not alias-proof collection, persistence,
enrollment or sessions. Those belong to subsequent account/auth slices. No hosted real-account
route is enabled here; no MFL calls are needed. The retry path and the service's separate record
reader needed small changes beyond the plan row's governance/commissioner description.
