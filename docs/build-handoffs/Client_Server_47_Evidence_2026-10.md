# Hosted market and season workflows

## Flow and preserved guarantees

`HostedLeague` → `HostedWorkflows` named forms → `hostedRoom` in-memory check/hold/confirm →
Go desktop session → `serverhttp.RawWorkflow.Validate` bounded publication → request-scoped
`leagueservice` admission → canonical commissioner/history preview or append transaction →
validated response parsers → named readings. No local authoritative store, credential in
React, offline queue, new engine, MFL submission or permission derived from a role label.
The role label only chooses which controls to show; every server request is authorized.

State/ownership: the history transaction owns revisions, request deduplication, effects,
privacy and replay. The client owns unsent inputs/review only. Command-specific commissioner
choices use `TargetCommissionerMarket` and the recipient version. Failed/ambiguous delivery
clears the review and requires refresh, never automatic execute retries. Parallel readings
with different recipient versions or football digests fail closed. A missing new-season
lineup is explicitly unavailable, not an authoritative empty lineup; unrelated successfully
read workflows can still operate. Server-selected evidence fences lineups and bids by season.

## Findings and dispositions

- New hosted market controls must not invent forms or routes around existing permission
  and preview rules. Reused `actTable`, `ActFields`, `actChange`, hold confirmation and the
  existing shared league service. Extended only the workflow publication whitelist.
- Commissioner market choices revealed the existence and closing time of a running blind
  auction. `Market.readBids` now omits it until due, including when its closing rule is
  missing; open auctions retain their existing visibility. The real HTTPS privacy test
  failed before this fix and passes afterward, comparing entire bodies and revision tokens.
- Bids needed the pinned football schedule, not the desktop's environment. Startup now
  installs a single selected evidence bundle for both lineups and bids. After rollover,
  old-season evidence refuses bids without an append. Missing playoff facts remain unknown;
  this slice does not invent playoff eligibility.

## Evidence and remaining acceptance

- `TestHostedBidsWaiversTradeCorrectionAndRollover`: real HTTPS owner/commissioner mutation
  route; repeated execute returns identical receipt; pick offer/accept/nonparty review;
  fresh-store computed/overridden standings agree; rollover assets agree with canonical
  replay; old-season bid refusal preserves revision.
- `TestHostedBlindBidIsInvisibleEvenToCommissioner`: ordinary owner can see their own bid;
  commissioner cannot read that seat or infer a running bid through commissioner choices,
  assets or standings. Shared service privacy tests retain the broader read matrix.
- `TestPinnedBidEnvironmentUsesScheduleAndFailsClosedAfterSeason`: week zero/first kickoff
  boundary and unknown completed schedule, not a wall-calendar guess.
- `hostedRoom.test.tsx`, `hostedWorkflows.test.tsx`: real Go wire fixtures, five task screens,
  named choices, no visible raw identifiers, stale/reconnect and duplicate-confirm refusal.
- Launcher lifecycle test: real disposable server/password/TOTP/desktop transport lifecycle,
  including prebuilt operation without Go; desktop GUI is not launched by this test.

Manual graphical acceptance is not claimed. Follow `cmd/twr-practice/README.md` using the
final desktop build. Practice results intentionally explain missing final NFL statistics;
this is not a fabricated completed NFL week. Rollover requires explicitly selected new-season
football evidence before new-season lineup/bid changes. Those safety limits are not bypasses.
