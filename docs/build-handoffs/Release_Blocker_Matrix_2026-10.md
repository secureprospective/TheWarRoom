# Release-blocker matrix evidence

Slice 50 covers the currently implemented roles and helper-token modes. RB2/RB3/RB4/RB8
are defined in `Client_Server_Build_Plan_2026-10.md`. Support/operator grants (R9/R10)
are not implemented and are outside this matrix, per the reviewer's scope ruling.
This is automated evidence, not a complete product release sign-off.

## Scope and flow

`internal/app/commissioner.NativeCommands` → adapter → shared league service → current
credential/domain policy → history transaction → receipt or refusal. Denial preserves
both leagues. Recipient projections own bid privacy; the transaction clock, not worker
progress, owns deadline admission. No production policy or dependency changes.

## Catalog and authorization matrix

The catalog has 136 commands: 35 reads and 101 mutations. The mutation fixture is
`internal/app/commissioner/testdata/release-mutations.json`. Each mutation has validated
Raw arguments, placement and explicit expected admitted roles. The fixture loader walks
the real catalog and fails on a missing or obsolete row. The new-command guard was
mutation-tested with `commish.release_probe.action`: it failed with “has no release
matrix fixture”; the probe was removed.

| Axis | Executable evidence |
|---|---|
| Every mutation; commissioner, two distinct co-commissioners, DOT, owner, outsider; two separately governed seeded leagues | `internal/app/commissioner/release_roles_matrix_test.go`, `TestReleaseMatrixCatalogRolesAndDesktopDoors`: explicit role expectations against real `AuthorizeRecords`, local/hosted service and Wails previews, refused execution through store, both services, legacy Wails and applicable named market doors |
| HTTP paths | The same table calls `release_http_matrix_test.go`: legacy preview/execute, shared dispatch, dedicated lineup/pick, and every workflow mutation accepted by the actual workflow validator. Real enrolled accounts and genuine recent-auth proofs prevent authenticator refusal from masking league permission checks |
| Specialized desktop ports | `internal/composition/native_gate_test.go`, `TestReleaseMatrixSpecializedDesktopDoors`: the actual desktop runtime's held-data seed, lineup and draft-pick ports, six identities, two leagues, vacant/foreign seat and repeat-seed refusals |
| Helper suggest/record modes and caretaker seat | `release_helper_matrix_test.go`: every mutation, ordinary-owner and caretaker token scopes, both leagues, wrong mode, foreign league, denied delegation and revocation, through hosted service, HTTP legacy doors and dispatch. An undelegable command is also attempted with a valid differently scoped token, rather than omitted |
| Real CLI and MCP | `cmd/twr/release_matrix_test.go`, `TestReleaseMatrixCLIAndMCPTokenModes`: same mutation fixture, both token modes and leagues, actual Cobra commands and SDK MCP calls against a real TLS hosted server. Includes wrong-command substitution for undelegable commands and revoked-token retries; only OS keychain storage is substituted |
| Denial and disclosure | Per-cell zero-event tail checks plus full before/after record equality for both leagues. Service permission failures must name the expected refusal, not an unrelated storage failure. Refused previews expose no league fields; HTTP permission/change refusal transcripts are fixed and non-cacheable, with no bearer/proof disclosure |
| Current scope and indirect references | `release_tokens_test.go`: every mutation's real issued ceiling, persisted scope, league/franchise/command/principal/mode/Raw mismatch and revoke. `internal/commissioner/token_scope_test.go`, `TestTokenArgumentScopeIncludesIndirectReferences`: actual player, pick, bid/claim and compensation references and every reference-bearing argument shape |
| Issuance and seat replacement | `release_token_roles_test.go`: owner and two genuine deputies cannot mint authority; caretaker token fails immediately when an owner arrives and never resurrects after owner removal |

The table tests **authority**, not successful completion of every business lifecycle.
Allowed-role cells assert the real governance gate and compare side-effect-free adapter
previews; missing draft, invitation, trade and other reference state can still block a
lawful role's proposed change. Fold-gated succession actions are checked through the
state machine as well as their entry gate. Specialized owner doors are also exercised
with seats they do not hold. Existing feature integration tests own successful moves,
consent, on-clock picks and legal lineups. Offline Wails has no helper-token mode;
CLI/MCP have no human-login mode; hosted workflow does not publish the entire catalog.
Those combinations are not invented as adapters or roles.

Accounts and league history now share one SQLite writer in the token/matrix fixtures,
matching hosted composition. Groups run in parallel with shared immutable league
fixtures. Recipient revisions are obtained once per identity/snapshot and kept distinct
from desktop event counts. No tests use live leagues or live football feeds.

## Privacy, time and credential lifetime

| Guarantee | Executable evidence |
|---|---|
| Every catalog read reaches credential admission | `release_matrix_test.go`, `TestReleaseMatrixCatalogReadAdmission`: all 35 service adapters, two leagues, missing/revoked admission and unchanged ledgers; this is an injected credential-boundary test |
| Commissioner cannot peek | `internal/leagueservice/view_revision_test.go`, `TestHiddenBidPreservesRecipientBodiesTokensAndErrors`: whole 17-adapter transcript equality across hidden place/raise/withdraw, including commissioner and owner refusals. HTTP: `TestHostedBlindBidIsInvisibleEvenToCommissioner`; SSE hidden-bid cursor/visibility tests in `internal/serverhttp/invalidations_test.go` |
| Exact close without worker progress | `internal/leagueservice/release_deadline_test.go`: place/raise/withdraw and waiver claim at deadline−1ns/at/+1ns; late history exactly unchanged |
| Queued writes and catch-up | `TestReleaseMatrixQueuedBidsUseCommitClock`: each bid write waits for the real SQLite writer, then refuses at close with no append. `TestDueWorkerNearestDeadlineCatchupAndRestart`: reopen, catch-up, exactly once. Existing queued untrusted-clock tests in `due_admission_test.go` |
| Enrollment/recovery/session lifetime | `internal/store/accounts/enrollment_test.go`: 48-hour expiry, concurrent redemption and queued expiry. `internal/accounts/sessions_test.go`: 7-day idle/30-day absolute limits, revoke and queued writes. Existing TOTP/recovery and SSE-revocation integration/security tests |

## Verification and incremental cost

Final local race suites passed for app/commissioner and cmd/twr. The new real-desktop
matrix passed separately under `-race`. A combined attempt to run the entire composition
suite exceeded the local command's 600-second budget; it is not reported as a local
full-suite pass. Hosted CI's four composition shards remain the full-suite gate.
`make lint` passed, including placement, root ratchet, transport, bloat and ifaceguard.
Frontend install/build passed before pushing the resolved main merge.

Local warm-build measurement, Go 1.26.10, GOMAXPROCS=4, `GOFLAGS=-p=2`, `-race -count=1`:

| Package | Baseline overlay (ba6a370c tests) | Candidate | Added |
|---|---:|---:|---:|
| app/commissioner | 18.901s | 166.193s | 147.292s |
| cmd/twr | 5.231s | 20.267s | 15.036s |
| New specialized desktop test alone | Not present | 8.822s | 8.822s |

Approximately **171 additional package-seconds** locally. This is a CI-cost proxy, not
an isolated GitHub runner delta or a sum to add to parallel workflow wall time. The
baseline overlay restores the previous token fixture and omits only the new matrix
files; production code is identical. The local host was shared with other work.

## Debt left

R9 support and R10 server-operator grants need their own cells when those capabilities
exist. No support/operator grant, role, elevation or permission bypass was invented.
This slice does not claim transport-by-transport proof of every successful business
transition, nor a complete release sign-off beyond the listed automated guarantees.
