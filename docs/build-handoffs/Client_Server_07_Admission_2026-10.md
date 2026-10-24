# Slice 07 — credential admission

## Boundary and ownership

Trusted request credential → `leagueservice.NewHosted` → validated Raw command or read
scope → credential admission → existing domain checks → transaction/retry/append.
No account/session/token implementation or new network route is introduced here.

| Boundary | Contract and effects |
|---|---|
| `commissioner.CredentialAdmission` | Receives the trusted principal, canonical catalog operation, mode and actual argument scope. Checks current credential validity, issuer rights and references, not just the selected franchise. A credential never grants domain authority. |
| `commissioner.WithCredentialAdmission` | Trusted Go-only context dependency, propagated through timeouts and the Host into history. Marking a request with nil admission denies it; request JSON cannot supply admission. An unmarked context remains the existing local desktop execution model. |
| `leagueservice.NewHosted` | Requires admission on reads, previews and executes, including static control metadata and league enumeration. Read callbacks receive franchise/player/subject-command scope where applicable. Command callbacks receive the complete prepared Raw command. |
| `history.ExecuteCommissionerWithFacts` | After acquiring the writer and beginning its transaction, passes current records and all Raw references to admission **before domain authorization and retry lookup**. Retries require current admission before returning the original receipt. |
| `history.PreviewCommissionerWithFacts` | Checks admission against its read transaction before retry lookup or description. Refusal returns blockers with no private fields; no writer is opened. |
| `history.RecordDraftPick` | Passes the complete pick input and current records to admission before ownership/retry lookup. The calendar permit and draft legality remain mandatory. |
| `history.appendCommissioner` | Rechecks immediately before building the event, using the planned Raw command and current transaction records. Denial rolls back, with no new event. |

Admission implementations are trusted dependencies: they must not mutate/retain supplied records
or arguments, write, or reenter the ledger. They may be called before a transaction without Records;
inside the transaction Records is its authoritative snapshot. Revocation and role changes must be
read afresh, not cached from the first call. The future account/token implementation owns those
credential facts and must coordinate them with transaction admission; fixture callbacks exercise
this contract here, not real session storage.

The existing domain Authorizer and record-based owner/legality checks remain unchanged. In
particular, the old trade/bid/waiver domain special cases cannot skip credential admission.
The desktop's static controls binding retains its non-error signature and offline behavior;
the hosted service's controls method returns an error and requires admission. No Wails binding
signature or generated client is changed.

## Proving tests

- `TestCredentialAdmissionQueuedFamiliesAndPreview`: occupies the sole writer connection,
  observes the pending connection wait, then revokes/removes rights before release. Generic,
  trade, waiver and bid families refuse, previews reveal no fields, ledger count is unchanged.
- `TestCredentialAdmissionRetryAndNilFailClosed`: admitted retry returns the original receipt;
  a queued revoked retry and nil admission refuse without new events.
- `TestCredentialAdmissionGetsAllReferencesAndTransactionRecords`: a plausible selected
  franchise does not hide referenced trade/window/bid IDs from admission.
- `TestCredentialAdmissionCannotReplaceDomainAuthorization`,
  `TestCredentialAdmissionRechecksBeforeAppend`, `TestCredentialAdmissionDraftPickBeforeRetry`:
  domain checks survive; revocation during planning rolls back; the specialized pick path is covered.
- `TestHostedReadsAndPreviewsRequireCredential`: every shared service read, metadata read,
  preview and execute denies nil/refused credentials before accessing host data.
- `TestHostedReadScopeIncludesFranchiseAndReferencedPlayer`: scopes survive the read adapters.
- `TestHostedServiceCarriesAdmissionIntoStoreTransaction`: actual service → SQLite store,
  transaction refusal, preview refusal, successful append and admitted/revoked retries.

Not proved here: real account revocation persistence, token issuance ceilings, cross-league alias
policy, HTTPS mutation admission, SSE closure, or atomic coordination with the future account
store. These belong to their later slices. No live MFL or owner state is used.

## Candidate verification

- Targeted race suite passed for `internal/leagueservice`, `internal/commissioner`,
  `internal/store/history`, and `internal/app/commissioner/...`.
- `make lint` passed; a worktree-local golangci cache avoids stale paths from other worktrees.
- `go test ./...` passed once on the final code candidate.
- Removing the pre-retry transaction admission check made
  `TestCredentialAdmissionRetryAndNilFailClosed` fail with “queued retry returned a receipt”.
  The check was restored and the targeted tests passed.
- Complete changed handwritten production set: 1,973 → 2,192 physical lines across 12 files
  against parent `90d4c4ab`; tests/docs/generated assets excluded. This is a safety addition,
  not a size reduction. No new package, placement change or generated binding edit is needed.
