# Hosted resource boundary

## Flow and ownership

`Hosted.Handler` creates one bounded peer throttle and request admission pool for
its listener. Socket peer → reject forwarding headers → six-auth/64-total request
admission → bounded peer throttle → request deadline/body cap → existing validated
Raw route. No request logging is installed; refusals contain fixed messages, not
request bodies, credentials or internal errors. Existing redacted credential
formatters remain in force; successful login/TOTP wire replies necessarily carry
secrets and must never be logged as JSON.

`cmd/twr-server.hostedServices` shares one `accounts.PasswordWork` across account
passwords, enrollment and TOTP (which uses the account service). Recovery also
uses its account service's work budget. Admission permits two active fixed-cost
Argon2 invocations and four additional waiters; overflow is refused immediately.
Canceled waiters release capacity; an executing Argon2 call retains its slot until
it returns because the library cannot cancel it. No process-global semaphore,
second writer, new persistence, dependency, or authentication-policy change.

The HTTP auth pool includes body decoding and non-KDF work, so six slow auth
requests cannot occupy all ordinary-command slots. All auth routes share ten
attempts per socket IP per minute, regardless of source port, username or route.
The peer table caps at 4096 entries; overflow fails closed and expired entries
are reclaimed. HTTP overflow/password saturation returns generic 429 with
Retry-After; total request saturation returns 503. The budget is listener/service
instance owned: deploy exactly one hosted handler and share its password budget.

## Limits and proxy contract

- JSON: 256 KiB; enrollment retains its tighter 8 KiB cap. No hosted seed-upload
  route exists yet; its planned 2 MiB exception must be added with that route.
- Headers: Go MaxHeaderBytes 16 KiB (net/http permits its normal parsing slack).
- Header/body/write/idle network deadlines: 5/10/5/30 seconds, preserved in the
  shared server constructor. Request contexts: auth 10 seconds, commands 5 seconds.
  Context cancellation is cooperative, not preemption of pure engine CPU work;
  blocked socket body reads are ended by the network read deadline.
- TLS 1.3 and literal-loopback-only exposure remain unchanged.
- **No proxy is trusted.** Forwarded, X-Forwarded-*, X-Real-IP,
  CF-Connecting-IP and True-Client-IP are rejected even from loopback. Throttles
  use only RemoteAddr, never a purported originating IP. A local proxy therefore
  shares one IP allowance; do not claim per-owner throttling through it.
- For the later proxy/Tunnel deployment, explicitly strip those headers before
  this origin, never forward client-supplied values. If real peer identity is
  needed, first implement and test an authenticated trusted-hop contract in a
  separate reviewed change. There is no trusted-proxy flag here.
- SSE-specific queue/device/write limits belong to its separate route and are
  not established by these non-streaming handler limits.

## Exercised evidence

- `TestPasswordWorkHundredRequests`: hold both active slots, send 98 more
  concurrent admissions, observe exactly four waiting and 94 refused, then drain.
- `TestPasswordWorkCanceledQueueReleasesCapacity`: deadline/cancellation rejects
  queued hashing and does not leak a slot.
- `TestPasswordWorkRealKDFLoad`: 100 simultaneous real verification attempts;
  records admitted/refused counts and actual hash p95 excluding queue time.
- `TestHostedLoginFloodKeepsCommandsAvailable`: real TLS, persistent accounts,
  100 password failures alongside 20 authorized command previews. Asserts generic
  refusal, throttling, no secret in replies/server error log, and unchanged ledger;
  reports command p95 and local heap observations.
- Proxy, per-peer expiry/table cap, 64-request limit, six-auth isolation,
  oversized headers/JSON and real TLS slow-header/body timeout regressions are
  in `internal/serverhttp/*limits*test.go` and `resource_integration_test.go`.
  Network deadline tests verify production values, then shorten durations only
  for the real-socket test.

Local candidate checks (Go 1.26.10, Beelink): targeted accounts/serverhttp/server
race tests passed; `go test ./...` passed (the first invocation was interrupted
by the command runner's 120s timeout, then completed with a longer budget).
A three-active-slot mutant failed `TestPasswordWorkHundredRequests`; restored
production code passed. The real KDF load admitted six/refused 94, hash p95
181.3ms. HTTPS flood returned five 401/95 429; authorized preview p95 38.1ms,
with unchanged history. HTTP credential formatting also rejects sentinel leaks.
Existing wire-only JSON serializers were deliberately preserved because desktop
Go needs those secrets; they are not suitable log payloads.

Source review traced each service hash/verify caller through its shared budget,
all hosted routes through request admission, and transaction/read cancellation
through the existing request context. The network limits already existed and
were preserved/exercised, not represented as newly invented defaults.

These are local engineering tests, **not target-CT acceptance**. Heap snapshots
are not peak RSS and do not establish no-OOM behavior on another machine.

## Christopher: target-CT gate (not run)

Use the actual disposable staging CT, its intended CPU count/memory quota, no real
accounts, no Tunnel/public exposure and no production files. Compile Linux test
binaries in this worktree at the reviewed candidate (same architecture as CT):

```sh
export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOTOOLCHAIN=go1.26.10
mkdir -p build/cs30-evidence
go test -c -o build/cs30-evidence/accounts-load.test ./internal/accounts
go test -c -o build/cs30-evidence/http-load.test ./internal/serverhttp
git rev-parse HEAD
sha256sum build/cs30-evidence/*.test
```

Transfer only these disposable test binaries to the staging CT. Record candidate
SHA, binary digests, CT CPU/RAM/swap limits and kernel, then run under its actual
memory limit; set these variables to the intended deployment values, not guesses:

```sh
: "${CT_CPUS:?intended CPU count}" "${CT_MEMORY_BYTES:?intended byte quota}"
export GOMAXPROCS="$CT_CPUS"
/usr/bin/time -v ./accounts-load.test -test.v \
  -test.run '^TestPasswordWork(HundredRequests|CanceledQueueReleasesCapacity|RealKDFLoad)$' \
  -test.count=1 -test.timeout=60s
/usr/bin/time -v ./http-load.test -test.v \
  -test.run '^Test(HostedLoginFloodKeepsCommandsAvailable|HostedOversizeAndForwardingPreserveState|ProxySpoofingAndAuthThrottle|HTTPAuthFloodReservesCommandSlots|BoundedServerSlowBodyAndHeaders|HostedServerOversizeHeaders)$' \
  -test.count=1 -test.timeout=60s
```

Before running, verify that the CT's effective cgroup `memory.max` equals
`CT_MEMORY_BYTES` and the CPU allocation matches `CT_CPUS`; stop on mismatch.
Repeat each command five times without increasing the CT limit. Capture maximum
resident set size from time, cgroup `memory.peak`, `memory.events` and
`memory.max` before/after, plus all printed percentiles. Require both tests to
exit zero, no increase in oom/oom_kill counters, peak below the recorded quota,
command p95 <1s, real hash p95 <=750ms, and the deterministic admission proof
(two active/four queued) passing. Do not interpret a test PASS as a latency gate:
the measurement tests report percentiles rather than enforce host-speed assertions.
If any gate fails, preserve evidence and adjust the deployment resources or
investigate implementation; do not weaken the ruled Argon2 parameters.

A staging proxy access-log check is still required with sentinel password, code,
seed, bearer and recent-auth values: logs must contain none of them. No real
proxy/CT inputs were supplied; no external deployment or target performance is
claimed here. Public flood/slow-body, SSE integration and end-to-end Tunnel
resource acceptance remain the release-blocker staging matrix.
