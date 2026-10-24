# Slice 24 — recipient-safe invalidations

## Implemented flow

`Hosted.Handler` routes `/v1/native/invalidations` to an authenticated HTTPS GET.
The league travels in `X-TWR-League`; secrets stay in desktop Go headers, never URLs
or the webview. Session and current league/TOTP policy are checked before opening
and on each recipient ledger read. The stream contains only protocol-1 `invalidate`
messages and fixed-time keepalive comments: no body, revision, event ID or cursor.

Each connection compares the existing recipient token every 250ms. The history
reader exposes committed records only. Invisible commits leave that token unchanged;
failed transactions produce no change. This deliberately uses the slice-23 visibility
rule rather than introducing a second visibility filter or mutation callback.
No history/account/service wiring changes are needed; worker and non-HTTP commits
are covered by the same reads. It coalesces changes into a single last-observed token,
with no buffered event queue or replay log. Limits are two streams per device session
and 64 per Handler. Writes have a 500ms deadline (stricter than the plan's 10s ceiling)
to avoid a blocked consumer delaying revocation closure.

`Desktop.WatchInvalidations` snapshots credentials under the desktop lock, opens the
stream outside it, and calls an authorized reread callback for each validated event.
Every connection begins with an invalidation, so reconnect needs no resume cursor.
Transport loss reconnects after one second; permission refusal stops the watcher.
The callback must replace cached views, never replay commands. Stream parsing bounds
lines at 4096 bytes and refuses duplicate data fields/cursor fields. Existing owner
screens are not wired here; slices 46–47 own hosted workflow integration.

## Evidence

- `TestSSEHiddenBidHasNoActivityCursor`: actual blind-bid append on a seeded Legacy
  fixture leaves another recipient's real HTTPS SSE stream silent.
- `TestSSECommittedVisibilityRollbackReconnectAndRevoke`: private role change and
  refused transactional execute emit nothing; visible note emits the fixed frame;
  a fresh connection invalidates; third device stream is denied; revocation closes
  both existing streams within one second and denies a new connection.
- `TestSSEDesktopRereadsAfterDroppedConnection`: real desktop Go watcher rereads via
  the authorized people route initially and after forcibly dropping TLS connections.
- `TestSSERejectsMissingCredentialsAndPrivileges`: missing session and missing
  privileged step-up fail before SSE opens.
- Client parser tests reject cursor fields, unsupported versions/events and oversized
  frames; incomplete frames do not trigger a reread.
- Existing `TestHiddenBidPreservesRecipientBodiesTokensAndErrors` continues to prove
  the underlying recipient-token rule across native readings.

Targeted race tests, targeted golangci-lint, and `go test -race ./...` passed.
The first full-suite invocation hit the runner's 200s timeout; the completed retry
used a longer runner budget. Push-hook results are recorded in the PR. No real MFL calls, roster moves or deployment occurred.

## Limits / plan interpretation

Post-commit delivery is observed by polling committed recipient views rather than
append-triggered fanout. Intermediate changes may coalesce; consumers reread current
state, not an event history. Target-CT load/latency, 64-stream saturation, slow-network
revocation and proxy/Tunnel streaming remain staging acceptance work, not measured
claims. Handler lifetime owns the connection budget; instantiate it once per server.

## Debt sweep 2 (2026-10-24): privileged streams end with their proof

A commissioner's or deputy's stream is admitted by a five-minute recent-auth proof.
The handler now reads the proof's remaining lifetime on the server clock at admission
(`TOTPService.PrivilegedRemaining`) and closes the stream on a one-shot timer at that
moment, instead of leaving it open and silent until the next commit failed a reread.
Owner streams, which need no proof, have no timer.
`TestSSEPrivilegedStreamClosesWhenProofLapses` opens a stream with 300 ms of proof left
and requires it to close between 200 ms and 5 s; removing the timer case fails it.

## Debt sweep 2: idle-stream cost with session checks

`BenchmarkIdleStreamHeads100k` (`-tags leaguebench`, linux) now includes the device
session checks: one op is one 250 ms round for 24 idle streams on a 100k-event ledger,
24 committed-head queries plus, every second round, 24 `SessionService.Check` calls
against account tables sharing the ledger's pools (12 per op on average). Beelink,
Ryzen 9 6900HX, `-count 3`, local process (not the target CT):

| Version | ns/op | CPU ns/op | session checks/op | B/op | allocs/op |
|---|---|---|---|---|---|
| heads only (`main` before this sweep) | 693,944–721,870 | 861,674–894,458 | 0 | 24,078–24,091 | 648 |
| heads + session checks | 1,206,946–1,274,107 | 1,464,053–1,532,602 | 12 | 42,547–42,601 | 1,177 |

So a session check costs about 45 µs of CPU, and 24 idle streams cost about 6 ms of
CPU per second (four rounds), roughly 0.6% of one core.
