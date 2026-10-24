# Client/server 08 — fixture HTTPS mutation

## Scope and flow

`cmd/twr-server -fixture -fixture-changes` creates a fresh disposable SQLite fixture;
no existing database path or caller identity is accepted. Default fixture mode remains
read-only. Both modes require literal loopback HTTPS. The new route is
`POST /v1/fixture/change`: protocol 1, one validated `commish.whatsnew.post` command
for `native.https-fixture`, with explicit expected revision and request ID.

Go client → bounded/strict Raw JSON → shared hosted Service.Execute → fixture-only
credential admission → existing domain authorization → writer transaction → current
credential/expected revision checks → shared engine → credential recheck → append/commit
→ receipt. Readback uses the shared authorized ledger reading. Denial records nothing;
canonical retry returns its original receipt only after current admission.

Finding: append checks sampled several instants and market preparation rounded to
microseconds; event columns also dropped nanoseconds. The transaction now samples one
UTC instant after obtaining the writer and passes it through append checks and market
preparation. Event timestamps retain nanoseconds; old microsecond rows remain readable.
Other history timestamps and existing rows are not rewritten. Preview uses the same
unrounded instant for its shared admission checks and trial. Event reads order by revision,
not timestamp text. No worker, accounts, public exposure, or real MFL access is added.

## Exercised evidence

- Frontend frozen install and build: pass.
- `go test -race ./cmd/twr-server ./internal/serverhttp ./internal/serverclient ./internal/leagueservice`: pass.
- `TestRealBinaryFixtureHTTPS`: compiled headless binary, trusted CA, Go-client note write/read;
  malformed/version/identity/league/trailing/oversize requests rejected without another event.
- `TestHTTPSFixtureWriteReadReopenAndConcurrentRevision`: HTTPS write, idempotent retry,
  concurrent expected revisions with exactly one winner, fully closed server/pools and cold
  disk reopen. Sub-microsecond receipt survives disk decoding.
- `TestNativeBidNanosecondAdmissionWithStoppedWorker`: place/raise/withdraw at deadline−1ns,
  at deadline and deadline+1ns; direct and deterministically queued behind the writer.
  Only −1ns succeeds; receipt instant and event count checked, then physical reopen.
  The due worker is never started. Targeted composition/history/service race batch passes.
- `go test ./...`: pass.
- `make lint`: pass with a worktree-local lint cache. The shared cache initially returned
  diagnostics from deleted sibling worktrees; no unrelated source was changed.
- Claude excerpt review: no blockers; no repository tools were available to that reviewer.
  Parent traced callers and ran tests. Residual fixture limitations include coarse HTTP error
  mapping, non-draining shutdown, and the existing HTTP/service timeout mismatch (retry is
  idempotent). RFC3339Nano text is not a sortable timestamp key; event queries use revisions.

## Limits and plan corrections

Fixture authority only. Due processing/restart catch-up and untrusted-clock suspension
remain slice 09; account authentication and public deployment remain later slices.
The existing reading API owns a bounded host context rather than the HTTP request context.
No plan ruling changed. CLI/client/fixture-host wiring was necessary to make the new
HTTP mutation reachable through the real binary. No new package or binding surface.

Affected handwritten production set (eight files, including new client/HTTP files):
1,271 → 1,548 physical lines; 48,238 → 58,754 Unicode characters including LF.
Tests/docs excluded. The growth supplies protocol, fixture composition and precision preservation;
it is not a debloat claim.
