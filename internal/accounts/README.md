# Local accounts

`RawCredentials.Validate` → `Service.LocalBootstrap` → random UUID and salted
Argon2id hash → `Repository.Insert(onlyIfEmpty=true)` → the supplied SQLite
pool's single writer. The conditional insert admits exactly one first account.
`Authenticate` reads a credential, validates its stored shape, and compares the
fixed-budget derived hash in constant time. Neither path calls league execution.

`internal/store/accounts` adds its tables to the supplied server-owned database;
it does not open another database or alter the shared migration version. Identity
updates and deletion are refused by triggers. Password hashes can change without
changing account IDs. Future account schema changes need integration with server
migrations before deployment.

Usernames are 3–64 ASCII letters/digits/dots/underscores/hyphens, normalized to
lowercase without trimming. Passwords are nonempty UTF-8, at most 1024 bytes,
without NUL; they are never normalized. These are local input/storage bounds,
not a claim that hosted enrollment policy is complete. Hashes accept only
Argon2id v19, 65536 KiB, t3, p1, salt16/hash32; malformed or foreign-cost hashes
fail before allocating KDF memory. Credentials redact normal Go formatting and
omit secrets from JSON; callers must never log individual secret fields.

Bootstrap is a local Go service operation, not an HTTP route, CLI flag, account
recovery mechanism, league creation or role grant. Hosting, stronger password
policy, login throttling/KDF concurrency and sessions remain later slices.
No network login is enabled here.

Evidence: `TestLocalBootstrapAuthenticationAndLeagueIsolation` exercises the real
service and stores alongside an existing native league: valid/invalid logins,
unchanged ledger, redacted serialization, and a bootstrap account that remains a
Visitor with no seat or privileged capabilities. Store tests cover uniqueness,
concurrent first-account admission, identity tampering, malformed persisted hashes
and reopen. Password tests independently derive the ruled KDF and reject malformed
encodings, including excessive resource parameters.

## Personal enrollment

`EnrollmentService.Issue` asks a trusted `EnrollmentAuthority` for the current
league policy and issuer (never request-supplied identity), requires a canonical
account with the commissioner role, and generates 32 random bytes. The local
caller hands `EnrollmentCode.Code` personally to the intended owner. There is no
email or HTTP issuance door. Issuance binds a canonical username, not a franchise
or role; it refuses an already registered username. Normal formatting and JSON
serialization redact codes and incoming passwords.

`RawEnrollment.Validate` → `EnrollmentService.Redeem` → salted password hash and
new UUID → `Store.RedeemEnrollment`. The store acquires the single writer before
sampling its server clock, checks the SHA-256 code digest, intended username,
unused state and `[issued_at, issued_at+48h)` validity, then inserts the account
and marks the code consumed in one transaction. Any refusal, collision or failed
commit preserves both. Consumed hashes remain consumed after restart. The code
itself is never persisted; expiry is stored at nanosecond precision.

`serverhttp.EnrollmentHandler` supplies an opt-in HTTPS-only
`POST /v1/auth/enroll` route with protocol 1, bounded strict JSON, no browser
Origin/query arguments, and a no-store account response. It accepts only
`{"protocol":1,"enrollment":{"code":"...","username":"...","password":"..."}}`.
It returns no session or league authority. It is **not installed in twr-server's
fixture listener**: trusted identity/policy adapters and authenticated server
composition belong to slice 20; TOTP, recovery and sessions precede that wiring.
Hash concurrency/throttling and deployment exposure remain later gates.

Account enrollment never reads or writes a seat invitation, role, owner or league
event. After enrollment, a commissioner must separately grant a seat to the new
`acct.<UUID>` through the existing authorized league command path. An account code
cannot redeem a seat invite or replace an occupied seat. A stolen handoff code
plus its intended username can enroll that identity: personal delivery is the
identity-verification boundary, not a claim of remote identity proof.

Evidence: `TestEnrollmentConcurrentRedemptionAndReopen` proves one winner and
persistent consumption; expiry/rollback and queued-writer tests prove exact 48h
admission and atomic refusal. `TestAccountEnrollmentDoesNotGrantOrReplaceSeat`
observes unchanged real league history, seats and roles and a Visitor account.
`TestEnrollmentHTTPSRoute` exercises actual TLS requests, strict input rejection,
wrong identity, success, replay and login using the new password.

## Privileged authenticators

The account service owns password verification, TOTP verification and encryption;
the account store owns replay counters, setup expiry and recent-auth admission.
The seed is 20 random bytes, provisioned as unpadded base32, using the RFC 6238
HMAC-SHA1 / six-digit / 30-second profile with one adjacent step of clock leeway.
HMAC-SHA1 is for authenticator compatibility, not password hashing or signatures.

| Flow | Contract and effects |
|---|---|
| `TOTPService.Begin` → `Store.BeginTOTP` | Verify this account's password; generate seed and nonce; AES-256-GCM encrypt with account ID as associated data. Persist only ciphertext with a ten-minute setup lifetime. Return the seed once for the account's setup UI. A pending setup can be replaced, never a confirmed factor. |
| `RawTOTP.Validate` → `Confirm` → `Store.ConsumeTOTP` | Verify password, decrypt the account-bound seed, compare the six-digit code in constant time. Acquire the writer, resample server time, check pending expiry/current counter/unchanged seed, and atomically confirm and consume the counter. Setup itself grants no recent-auth proof. |
| `StepUp` → `Store.ConsumeTOTP` | Require password and a fresh code for a confirmed factor. Atomically consume the counter and store a random proof's SHA-256 digest and server authentication time. Only return the plaintext proof after commit. A new step-up supersedes this account's prior proof. |
| `CheckPrivileged` → `Store.CheckRecentAuth` | With current trusted policy and canonical principal, require a confirmed factor and matching account-bound proof for the commissioner and every co-commissioner. Accept only `[authenticated_at, authenticated_at+5m)`; reads never extend it. Ordinary accounts get no extra TOTP requirement, and this check grants no domain rights. |

The consume path requires a strictly increasing counter: confirmation, concurrent
requests, adjacent-window codes and restarts cannot reuse a consumed counter.
A queued verification rechecks its code window after writer admission. Any
refusal or failed persistence preserves factor/counter/proof state. Confirmation
uses its code, so the user needs the next unused authenticator code for step-up.
Seeds, codes, ciphertext state and proofs redact normal Go formatting and omit
secrets from JSON. Callers must never log individual secret fields.

`NewTOTP` takes an external 32-byte encryption key, never saved in SQLite. Losing
it loses factor access; rotation, offline recovery and deployed key delivery are
not implemented here. There is no factor-reset or commissioner-reset door. The
repository is a trusted service port, not an API for request-supplied counters,
account IDs or policy. No TOTP HTTP route or deployable admission adapter is
installed: slice 20 must call this guard on every hosted path, including the
inside-transaction recheck with current policy; slices 18–19 supply recovery and
sessions. Device-bound step-up integration, request throttling and operational
clock/key handling still need their planned slices. This is not evidence that
the fixture listener or the legacy local desktop now enforces TOTP.

Exercised evidence: `TestPrivilegedTOTPEnrollmentReplayAndRecentAuth` covers
unconfigured/confirmed commissioners, replay, forged proofs, exact five-minute
boundaries and redaction; `TestTOTPDeputiesAndAccountBinding` covers two deputies
and cross-account proof denial. Concurrent-code/reopen tests observe one winner,
persistent replay protection and fresh-code success after restart. RFC vectors,
wrong passwords/codes, wrong keys, ciphertext tampering/account swapping,
expired/replaced pending setup, injected persistence failure and queued-code
expiry have separate regression coverage. Changing the replay comparison to
allow equality makes the proving test fail on confirmation-code replay.

## Lost credentials and printed codes

`RecoveryService.PrintCodes` verifies the account password; the store rechecks a
confirmed, account-bound recent TOTP proof **after writer acquisition**. It replaces
any prior set with SHA-256 digests of eight independent 256-bit codes and audits
issuance in the same transaction. `RecoveryCodes.Codes` is returned once for the
caller to print and keep offline; formatting and JSON omit the codes. No printer
or UI is installed here. Losing the printout requires authenticated regeneration
or the offline-key console. Regeneration invalidates the old set.

`RawRecovery.Validate` → `Recover` → hash the new password → `ResetAccount`:
under the single writer, match username plus code digest, replace the password,
remove TOTP/setup/recent-auth state, invalidate **all** old recovery codes, advance
`account_auth_generation`, and insert a security audit entry before commit.
A code alone is a full recovery credential; password knowledge is not required.
Two concurrent uses have one winner. Invalid codes, wrong accounts and audit
failure preserve credentials and consumption. Identity and league state never
change. The account must re-enroll its authenticator before privileged admission.
Recovery does not grant recent authentication, a session, or league rights.

`ConsoleRecover` additionally requires a 32-byte offline key whose SHA-256 digest
matches trusted operator configuration in constant time, plus a bounded printable
reason. It performs the same transactional reset and records `offline_key_reset`,
account ID, server time and reason. Only successful operations are recorded here;
failed-attempt throttling/security telemetry and hash chaining remain later work.
Never put secrets in the operator reason. The offline key is independent of the
TOTP encryption key; neither is saved in this database. A host administrator can
change local configuration/database state: this does not protect against root.
Commissioners can reassign seats, not reset accounts; no league policy or actor
field is accepted by either recovery operation.

`AuthGeneration` is the durable revocation fence: a credential/delegation's issued
generation must equal the current account generation, checked at admission and
again inside append. Device sessions and agent delegations do **not** yet exist;
slices 19/20/25 must implement that comparison on their real paths. This slice
proves generation advancement/persistence and recent-proof revocation, not live
token rejection. The repository is trusted, not a request-facing reset API.

Tests: `TestRecoveryConcurrentUseRevokesCredentials`,
`TestRecoveryAuditFailureRollsBackAndConsoleRequiresOfflineKey`,
`TestRecoveryPrintingRequiresFreshAccountBoundProof`,
`TestRecoveryResetPersistsAcrossReopen`, and
`TestCommissionerSeatReassignmentCannotResetAccount` exercise the real stores and
league seat path. Console usage and its exercised boundary are documented in
`cmd/twr-server/README.md`. Account schema deployment/migrations, hosted recovery
routes, off-host audit anchors and restoring a fresh auth epoch remain unproven.

## Hosted integration

The earlier slice-local statements above describing network wiring as future work
are superseded by `cmd/twr-server/README.md` → Account-authenticated mode.
`serverhttp.Hosted` installs enrollment redemption, device login and authenticator
setup/step-up, then constructs a request-scoped shared league service from the
session account. Its admission guard rechecks sessions, current league policy and
privileged proofs inside preview/append, including retry lookup. Credentials and
history must share the same database pools/writer; separate account databases are
not a supported hosted composition. Device sessions now enforce recovery generation
and epoch fences. Agent delegations and deployed recovery remain separate work.

## Hosted password resource budget

The deployed composition shares one `PasswordWork` between `NewWithPasswordWork`
and `NewEnrollmentWithPasswordWork`; TOTP and recovery use the same account service.
All account-service hashing and verification now admit at most two active Argon2
calls and four waiters, rejecting overflow with `ErrPasswordBusy`. Defaults create
private budgets for independent local services. Raw `HashPassword`/`VerifyPassword`
remain fixed-cost primitives, not network admission doors. Canceled queued work
never enters Argon2; active work retains its slot until the library returns.
Every hosted password door answers a full budget, or password work that ran out of
request time while queued, with HTTP 503 and `Retry-After`, worded "The server is busy;
try again in a minute" (`TestPasswordWorkTimeoutIsBusyNotRefused`); the desktop shows
that wording rather than password or phone-clock advice.
Hosted throttles and performance evidence supersede the earlier future-work
statements about KDF concurrency above. See
`docs/build-handoffs/Client_Server_30_Evidence_2026-10.md`; target-CT performance and
proxy log redaction still require staging evidence.

## Scoped agent tokens

`RawTokenScope.Validate` → human device session → `TokenService.Issue` → shared
single-writer transaction → recheck session and trusted `TokenAuthority` → hash-only
agent credential plus security audit → return bearer once, after commit. Issuance
fixes one league/franchise, explicit command allowlist and explicit modes. Any
execute mode caps the entire token at 24 hours; read/preview-only tokens last 30
days. Expiry is server-owned, nanosecond precise, exclusive, and never refreshed.
The persisted scope and issuance fields cannot be updated. Policy cannot extend
the issuance ceiling when an issuer gains new rights.

`TokenService.Admission` → validate/hash bearer → check expiry, revocation, account
recovery generation and auth epoch → intersect requested operation with issuance
scope → recheck current issuer rights for that operation → check all argument
references. Role loss can remove execution while retaining an independently held
read right. `Authenticate` establishes credential identity only; callers must use
`Admission` on reads/previews/execute/retries, including inside ledger append.

`TokenAuthority` is a mandatory trusted policy port. Its scope check must authorize
every requested command/mode at issuance, and the single actual command/mode on
use. Its access check must resolve player, bid, trade, pick and other indirect
references against the scoped franchise and current policy. Both checks are
read-only, must use the current state (including `Access.Records` at append), and
must not acquire the writer. Accounts and history must share the same pools so
revocation/recovery/role writes serialize with append. Missing policy fails closed.
Transport composition and production reference resolution belong to slice 26;
this slice does not enable agent HTTP/CLI access or claim all-family scope proof.

Tokens cannot issue/revoke credentials or administer seats, roles, succession or
permissions. Generic correction/revert is excluded because it could undo a role
change. These exclusions are enforced independently of the supplied authority.
Agent and device hashes live in separate tables; neither credential is accepted
as the other. A current human device session can revoke only its account's token
ID. Recovery generation/restore epoch changes invalidate tokens automatically.
Device-only revocation does not itself revoke separately issued agent tokens.
IDs (not token digests or bearer values) are recorded in transactional issuance
and revocation audits. Bearers are omitted from JSON and normal Go formatting;
never log an individual bearer field. Audit chaining remains separate work.

Tests `TestAgentTokenLifetimesScopeAndCurrentIssuerRights`,
`TestAgentTokenHashOnlyCeilingAndCredentialIsolation`,
`TestAgentTokenRevokedQueuedAppendAndRetry`,
`TestAgentTokenValidationAndFailClosedPolicy`, and
`TestAgentTokenPersistenceRecoveryAndAtomicIssuance` exercise real account/history
stores, explicit trusted test policy, exact expiry, role loss/gain, cross-scope and
reference denials, hash-only storage/redaction, queued revocation before retry,
zero denied ledger growth, immutable scope, reopen, recovery/epoch fences and
rollback on audit failure. No deployed policy adapter or live owner is exercised.

## Desktop sign-in and restored copies

The desktop's local principal hashes its data folder and OS user, so a backup restored
under another folder or user is a new identity. `LocalSeat.SignIn` creates the first
local account (or verifies the existing one) and appends `account_aliases(alias →
account)` in history.db, beside the ledger, so a matched backup carries it. The history
store resolves every linked alias to its account through the existing verified-alias map
(`governance.Principals`) whenever a caller supplies none; recorded authors never change.
A restored copy links its new identity only after the original username and password
verify, and bootstrap stays closed once any account exists, so a copy without the
password cannot link or create its way into a role. Links are append-only.

Evidence: `TestRestoreOnAnotherComputerKeepsCommissionerWithSignIn` (restored under a
different local identity: Visitor, wrong password and a second account refused, then
Commissioner after the original sign-in, and a ruled change executes) and
`TestStrangersRestoredCopyWithoutSignInStaysVisitor`. Removing the history alias hook
makes the first fail with a Visitor reading.
