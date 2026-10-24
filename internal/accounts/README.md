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
recovery mechanism, league creation or role grant. Hosting, enrollment, stronger
password policy, login throttling/KDF concurrency, TOTP and sessions belong to
later slices. No network authentication is enabled here.

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
