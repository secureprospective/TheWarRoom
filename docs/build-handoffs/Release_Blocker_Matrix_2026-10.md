# Release-blocker matrix evidence

Slice 50 is **partial**, not a release sign-off. RB2/RB3/RB4/RB8 are defined in
`Client_Server_Build_Plan_2026-10.md`. All paths below are repository-relative.

## Scope and flow

Catalog (`internal/app/commissioner.NativeCommands`) → adapter → shared league
service → current credential/domain policy → history transaction → receipt or
refusal. Denial must preserve the ledger. Recipient projections own bid privacy;
the transaction's clock, not worker progress, owns deadline admission.

No production policy changes are made by the matrix. Merging main also required
combining the desktop workflow and update-required routes; the handler now
separates route checks from native account-action handling.

## Added evidence

| Guarantee | Executable evidence | Limit |
|---|---|---|
| New catalog command requires review | `internal/app/commissioner/release_matrix_test.go`, `TestReleaseMatrixCatalogDelegationCeiling`: explicit 136-command inventory and actual token-scope validation | Reviews delegation ceilings, not each command's successful execution |
| Every catalog read reaches credential admission | `TestReleaseMatrixCatalogReadAdmission`: 35 real service adapters, two existing leagues, missing/revoked credential; expected refusal and both ledgers unchanged | Injected refusal; not the real role/token cross-product |
| League roles do not cross league boundaries | `internal/leagueservice/release_roles_test.go`: commissioner, two deputies, owner, support/operator identities without league grants and other commissioner; real policy/history through store and service | League-note mutation only; support/operator names carry no implicit grant; caretaker seat is seeded but not a separate matrix axis |
| Commissioner cannot peek | `internal/leagueservice/view_revision_test.go`, `TestHiddenBidPreservesRecipientBodiesTokensAndErrors`: commissioner now gets the same whole 17-adapter transcript comparison as the other owner, including expected owner-only refusals, across place/raise/withdraw | Not every catalog read; existing HTTP/SSE tests remain separate |
| Exact bid/claim close without worker progress | `internal/leagueservice/release_deadline_test.go`: bid place/raise/withdraw and waiver claim at deadline−1ns/at/+1ns; late history exactly unchanged | Store path; not every transport |
| Queued bids use current transaction time | `TestReleaseMatrixQueuedBidsUseCommitClock`: each bid write waits behind the real SQLite writer; clock reaches close before lock release; no event append | Queued waiver claim not added |

The catalog guard was mutation-tested: adding `commish.release_probe.action` to
`NativeCommands` fails with “catalog changed; review delegation ceilings and
extend the release matrix inventory.” The probe was removed.

## Existing complementary tests exercised

- RB2: `internal/store/accounts/enrollment_test.go` (48-hour expiry, concurrency,
  queued expiry); `internal/accounts/sessions_test.go` (idle/absolute limits,
  revocation and queued writes); TOTP/recovery integration/security tests.
- RB3: `internal/serverhttp/token_dispatch_test.go` (agent family scopes,
  suggest/record behavior, human consent, expiry/revoke); service admission and
  Wails/service parity tests; `internal/commissioner/caretaker_test.go`.
- RB4: `internal/serverhttp/market_workflow_test.go`,
  `TestHostedBlindBidIsInvisibleEvenToCommissioner`; SSE hidden-bid cursor and
  visibility tests in `internal/serverhttp/invalidations_test.go`.
- RB8: `internal/leagueservice/due_worker_test.go`,
  `TestDueWorkerNearestDeadlineCatchupAndRestart` (reopen, catch-up, exactly once);
  queued untrusted-clock tests in `due_admission_test.go`.

## Verification and runtime

Race suites passed for app/commissioner, leagueservice, serverhttp, accounts and
store/accounts. Merge integration race tests passed for serverclient and
cmd/twr-practice. Frontend install/build passed before the main merge; final
commit/push gates and hosted CI provide the post-merge checks.

Local warm-build timing, Go 1.26.10, `-race -count=1 -p=1`, GOMAXPROCS=4, both
changed packages: baseline overlay 67.949 seconds; candidate 111.866 seconds,
**+43.917 seconds**. Baseline substitutes the original privacy test and omits the
new matrix files without changing production code. This is a local loaded-host
observation, not a measured GitHub runner delta. Deadline subcases share fixtures;
independent command/role groups run in parallel.

## Remaining release criterion

The full every-family × two-league × role/token-mode × applicable-adapter
cross-product is **not established**. In particular, valid command fixtures and
unchanged-state assertions are still needed for every mutation family through
HTTP/workflow/Wails/CLI/MCP, including foreign references; real support/operator
credentials and caretaker revocation need their own matrix cells. Existing
individual tests are not a substitute for that inventory. Do not mark plan row
50 or RB3 complete based on these additions.
