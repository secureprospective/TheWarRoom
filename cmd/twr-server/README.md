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
