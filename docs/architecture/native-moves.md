# Native HQ move reading

`moves.read` / `TargetNativeMoves(league, franchise)` is read-only. It adds no store,
mutation command, engine rule or MFL dependency. The locked HQ mock remains unchanged.

## Flow and guarantees

- The shared league service validates `RawNativeMovesScope`, admits hosted credentials,
  resolves the trusted principal and calls `CommissionerRecords` once.
- `market.NativeMovesReading` authorizes through `NativeTradesReading`,
  `NativeBidsReading` and `NativeWaiversReading`, all over that same record snapshot.
  It rejects mixed-league records and excludes records after the reading instant.
- The new adapter filters trades to this franchise's parties, including during review.
  Bid cards use only `YourBid`; other owners' bids and amounts never enter the queue.
  Pending claim cards use the existing private claim reading. There is no global
  revision/count that changes when an unrelated blind bid is placed.
- Planned is an empty array with reason `native plans not yet stored`. Personal lists
  are never read: they are not league-scoped plans.
- An offered trade needs this owner only while their franchise has not accepted.
  Already-accepted parties, DOT reviews/appeals, live bids and unprocessed claims are
  handed off. No expiry or decline is interpreted as a landing.
- Active committed trade approvals, bid awards, waiver awards and cuts are landed.
  `Replay` and `NewFrame` supply compensation and final award state; no engine policy
  is reimplemented. A landing names its committed ledger entry and recorded instant.
- Each card retains the HQ receipt fields (`correlationId`, `spec`, `state`, `audit`).
  Native targets use `kind: native`, and native evidence uses `event: recorded`.
  The native adapter deliberately does not fabricate MFL targets, checks or matches.
  `frontend/src/app/data/nativeMoves.ts` validates this wire independently; the MFL
  receipt parser remains unchanged. Timestamps on the wire are UTC; deadline wording
  comes from the existing viewer-local readings.
- Errors fail the read rather than returning a misleading empty queue. Neither this
  service nor its parser submits moves, retries commands or fetches external data.

## Settings read through the existing readings

The new adapter reads no rule value directly and has no preset-specific branch.
Observable timing/visibility depends on `bids.standFor`, `bids.ufa` and
`waivers.duration`; the command-flow tests change each and observe the changed read.
Contract terms and dead money are the recorded command outcomes, not invented prices
or repriced history. Offered trade expiry/policy is frozen by the existing command.

The delegated readers also resolve/read these rule fields while building their
existing policies and rule lines (not all become fields in this smaller HQ read):

- `bids.yearMultipliers`, `bids.maximumActive`, `bids.oneYearMaximum`, `bids.tiers`,
  `bids.minimumIncrement`, `bids.snipeAfter`, `bids.snipeIncrement`,
  `bids.multiYearEndWeek`, `bids.playoffOnlyAfterWeek`, `bids.playoffMaximumYears`,
  `bids.playoffCapLifted`, `bids.pointDecimals`.
- `contracts.minimumSalaries`, `cap.quantum`, `playoffs.startWeek`.
- `waivers.trigger`, `waivers.method`.
- `trades.review`, `trades.dotSeats`, `trades.decideAt`, `trades.futureYears`,
  `trades.rounds`, `trades.rosterGate`, `trades.offerExpiry`, `trades.maximumParties`.

No roster/lineup/taxi/IR limits, tiebreak order, rookie scale, cap limit or contract
end-type value is added to this reading. Ledger replay still owns its existing rule
resolution; this adapter does not change those dependencies.

## Evidence and limits

Real local command-flow tests cover empty plans, incoming/offered/reviewed/approved
trades, won bids, processed claims, cuts, compensation and restart. They reject another
franchise/league and test trade/bid/claim isolation. An HTTPS workflow test proves blind
bid existence does not change another owner's complete response. The Go-written wire
fixture is consumed by frontend tests, which also prove the MFL parser rejects native
receipts. Catalog/door/role, admission and generated binding checks cover publication.

The only composition source fingerprint changed is the straight-line `Bind2` entry
for the new method in `command_bindings.go`; no decision-bearing fingerprint changed.

Suspected issue retained: `NativeTradesReading` itself exposes unaccepted offers to
non-parties. This read filters locally as mandated; the broader desk is unchanged.

Debt: league-scoped plans remain deferred. No addressed commissioner-request command
or persisted lifecycle exists in the current commissioner records (notices are public
text with no recipient or completion state), so this read cannot manufacture those
cards. The new read/parser are published for HQ consumption; this slice does not mount
a native moves screen or change the locked mock. Graphical acceptance and independent
review were not performed.
