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
