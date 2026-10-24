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
