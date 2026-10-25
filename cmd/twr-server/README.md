# Loopback native fixture

Development only: one fixed, public native ledger reading computed by the existing
commissioner engine. No database, writes, real league selection, MFL calls, accounts,
catalog publication or Wails/display. The explicit `-fixture` flag is mandatory.
Do not deploy this as an owner-facing server.

From the repository root, create disposable local TLS material (OpenSSL required):

```sh
mkdir -p .scratch/cs01-tls
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
  -keyout .scratch/cs01-tls/key.pem -out .scratch/cs01-tls/cert.pem -days 1 \
  -subj '/CN=Local fixture' -addext 'subjectAltName=IP:127.0.0.1'
chmod 600 .scratch/cs01-tls/*.pem
go build -o .scratch/twr-server ./cmd/twr-server
.scratch/twr-server -fixture -listen 127.0.0.1:0 \
  -tls-cert .scratch/cs01-tls/cert.pem -tls-key .scratch/cs01-tls/key.pem
```

The first output line is the HTTPS origin, including the assigned port. In another terminal:

```sh
.scratch/twr-server -fixture -read https://127.0.0.1:PRINTED_PORT \
  -ca .scratch/cs01-tls/cert.pem
```

The Go client prints the ledger JSON. Trust is explicit; no insecure-skip-verification
option exists. Ctrl-C stops the server. TLS keys above are disposable and ignored by git.

Protocol: GET `/v1/fixture/ledger`, no query or body. Response is
`{"protocol":1,"ledger":{...}}`; the client invocation prints the inner ledger.
Unsupported routes/major versions return 426; non-GET requests return 405;
arguments return 400. The route accepts no identity, franchise or command arguments.

Proving test (generates a test CA/leaf, builds and starts the real binary):

```sh
go test -race ./cmd/twr-server -run TestRealBinaryFixtureHTTPS -v
```

It compares HTTPS and CLI JSON against a direct `commissioner.Reading`, refuses an
untrusted certificate and unsupported major, rejects writes/spoofed arguments, then
rereads unchanged fixture state. Targeted service/client tests cover fresh fixture
ownership, cancellation, malformed/oversize/new-major replies and redirect rejection.
Public deployment, storage, authorization, accounts, mutations, version negotiation
beyond rejecting unknown majors, and desktop integration remain later slices.

## Offline account recovery (local console only)

Stop the hosted service before using its database. Mount the sealed offline key
only for this operation. It is exactly 32 random binary bytes in an owner-only
regular file, independent of the authenticator encryption key. Its SHA-256 digest
must already be pinned in trusted operator configuration; do not obtain the digest
from the reset requester. Initial offline-key custody/delivery is an operations
prerequisite, not an automatic key bootstrap in this command.

```
twr-server recover -accounts-db /path/to/server.db \
  -offline-key-file /mounted-offline/key \
  -offline-key-sha256 <configured-64-hex-digest>
```

Supply one JSON object on stdin (4096-byte maximum): `username`, new `password`,
and a nonempty printable `reason` (256 characters maximum). Use a console secret
input tool or an owner-only temporary input file; do not put the password in
shell arguments/history. The command emits only a generic result, opens no
listener, and cannot be invoked through league commands. Remove the mounted key
and any temporary input immediately afterward. Do not log raw stdin or secrets.

Success atomically resets the password, removes the authenticator and recent-auth
proof, invalidates printed codes, advances the account credential generation and
records the account/time/reason audit. It changes no account ID, username, role,
seat or league event. Set up the authenticator again before privileged use.
Device-session/token admission will consume the generation fence in later slices;
there are no live sessions or delegations to revoke yet. Audit chaining and
host-replacement recovery are separate later gates.

`TestOfflineConsoleRecovery` exercises stdin validation, insecure-key permissions,
success, preserved identity, new-password login and the persisted audit record.
This is not a deployed recovery or key-custody drill.

## Account-authenticated mode

```
twr-server hosted -db /path/to/server.db \
  -tls-cert /path/to/server.pem -tls-key /path/to/server.key \
  -totp-key-file /path/to/authenticator.key \
  -preview-registry docs/ui/endpoint-registry.csv
```

This is a separate mode, still literal-loopback-only pending the deployment work.
It opens one existing server-owned database for history **and** accounts, sharing
one SQLite writer. It never seeds a fixture, registers `/v1/fixture/*`, reads an
installation principal or supplies a domain authorization override. `-fixture`
and `-fixture-changes` are not accepted in this mode. No real-owner exposure is
approved by this wiring.

Provision the first account and initial league through the existing trusted local
account/history APIs; bootstrap itself grants no league role. Enrollment codes
must already have been issued locally by a current canonical commissioner and
handed personally to the intended username. The listener only redeems codes;
there is no network issuance, account-reset or role-override route. Legacy league
migration and persisted verified-alias configuration are not supplied here.

| Route (POST, protocol 1 JSON) | Input / result |
|---|---|
| `/v1/auth/enroll` | Existing enrollment envelope; creates an account, not a seat/session. |
| `/v1/auth/login` | `protocol`, `username`, `password`, `device`; returns `bearer` and redacted session metadata. |
| `/v1/auth/totp/begin` | `protocol`, `username`, `password`; returns setup `seed` once. |
| `/v1/auth/totp/confirm` | Same plus six-digit `code`; consumes confirmation code. |
| `/v1/auth/totp/step-up` | Same plus a fresh `code`; returns `recent_auth`. |
| `/v1/native/ledger` | `protocol`, `league`; authorized ledger. |
| `/v1/native/preview` | `protocol`, `command` (RawCommand); shared-service dry run. |
| `/v1/native/execute` | Same command envelope; shared-service receipt, including canonical retries. |

Native routes require `Authorization: Bearer <device secret>`. Commissioners and
all deputies also need `X-TWR-Recent-Auth: <proof>`. Credentials, setup seeds and
proofs belong in desktop Go memory/keychain, never the webview or logs. TLS,
no browser Origin, no query parameters, strict single-object bounded JSON and
`Cache-Control: no-store` apply. Identity/role/time fields are not accepted;
spoofed identity headers do not select the principal.

Flow: session authentication/touch → canonical account principal → validated Raw
request → request-scoped shared service → current league policy + privileged
recent-auth guard → store transaction → repeat session/policy/TOTP checks before
retry lookup or append → receipt. Ledger reads require current membership;
preview/execute additionally use argument-aware domain authorization. Refusals
return generic errors and append nothing. Revocation and ledger append share the
writer, so a revocation committed ahead of a queued append wins.

**Clock health:** hosted mode reads the kernel's NTP state with a read-only `adjtimex`
call at each timed admission and worker pass: trusted only when the call succeeds, the
clock state is not `TIME_ERROR` and `STA_UNSYNC` is clear. Any Linux host with a running
NTP client (chrony, ntpd, timesyncd; systemd or OpenRC) qualifies; non-Linux builds and
unsynchronized or unreadable status fail closed. Status is never cached or inherited from fixture trust. The due
worker starts with hosted HTTPS and is cancelled/joined before the database closes.

**Recovery:** `POST /v1/auth/recover` accepts protocol 1, username, one unused printed
recovery code, and the new password over HTTPS. No session is needed; the code supplies
account authority, never league authority. It consumes the shared authentication peer
allowance and password-work budget. Successful reset revokes devices, removes old factors,
and consumes the code. The native desktop Account recovery tab uses this route and then
returns to sign-in. Console recovery remains the fallback when no unused code is available.

Public proxy/Tunnel, SSE and deployment/key-custody drills remain separate gates.

Evidence: `TestHostedHTTPSEnrollmentLoginWriteAndAdmission` exercises real TLS
requests through enrollment, password login, TOTP setup/replay/step-up, authorized
read/preview/write/retry and expired/revoked/spoofed/missing credentials with
unchanged history on denial. `TestHostedHTTPSQueuedRevocation` delays the request
body after authentication, queues the append behind a revocation transaction and
proves zero events append; removing the session-admission rejection makes it fail
with HTTP 200. `TestRealBinaryFixtureHTTPS` also launches the real binary in hosted
mode, logs into its persistent account store and verifies that host bootstrap
acquires no fixture/league authority. No MFL calls are involved.

## HTTP and password resource limits

Hosted mode now uses one shared two-active/four-waiting password budget, a
six-request authentication pool, a 64-request total pool and bounded per-socket-IP
authentication throttling. It rejects forwarding headers rather than trusting
client-supplied IP/HTTPS assertions. Behind Cloudflare Tunnel, pass
`-trusted-proxy 127.0.0.1` (the loopback address cloudflared connects from; off by
default). Only requests from that exact loopback peer may carry forwarding headers,
and each must name exactly one client in `CF-Connecting-IP`, which becomes its
throttle bucket. The same headers from any other peer are still refused
(`TestTrustedTunnelHopThrottlesEachClient`). No public exposure
or target-CT performance acceptance is implied. Limits, proving tests and exact
CT measurement steps: `docs/build-handoffs/Client_Server_30_Evidence_2026-10.md`.

## History checkpoints

`twr-server checkpoint -db /path/to/server.db` prints a checkpoint of the league and sign-in
history after verifying both. Keep it away from the server (slice 36 stores one with each
off-site backup). `twr-server checkpoint -db /path/to/server.db -check checkpoint.json` reports
whether the server's history still matches it; a rewrite or rollback since then fails with a
plain message. Both forms only read the database. Details: `docs/backend/Audit_Chain.md`.

## One-command hosted click tests

For a pre-provisioned throwaway hosted league, TLS, enrolled authenticator and isolated desktop
launch command, use [`twr-practice`](../twr-practice/README.md). This is development tooling, not
commissioner installation/bootstrap or a release artifact.

## Hosted off-site protection

Hosted mode now starts with writes paused unless `-backup-config` identifies private
R2 settings and the first matched set passes encrypted upload/read-back. Deployment
configuration/key/registry files must be colocated under their installed names.
`backup-check -config <file>` validates settings locally without printing credentials.
See [CT setup, retention and restore handoff](../../deploy/server/BACKUPS.md).
