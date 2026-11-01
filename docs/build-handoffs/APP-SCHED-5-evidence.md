# APP-SCHED-5

## Item 1

Flow: held revisions (authority retained, oldest first) → existing counted-version selector →
newest live, then newest nflverse; explicit commissioner approval pins a revision and entered
statistics overlay it. Ordinary scheduled scoring sync → ProcessStatFeed → ordinary StatApprove
commands, deterministic request IDs, worker-only authorizer; append transaction rechecks held
revision/authority/manual pin. Existing queue, player change provenance and ledger read the same
commands. No score arithmetic/settings changes. Automatic records never become commissioner pins.

New race tests observe live → newer live → nflverse → ignored later live, correction/pin preservation,
ledger who/what/when, repeated processing and reopening without duplicate entries, and refusal outside
worker authorization. Bootstrap records the currently selected revision, not fictitious intermediate counts. Recorded time is
ledger processing time; held evidence retains source arrival.

Changed tests (one line each):
- TestStatItemIDRoundTripsAndRefusesAnythingElse: revision zero is now valid to record initial feed selection or explicitly pin it; negative sequence remains invalid.
- TestNativePlayerScoreRetainsCountedRevisionGaps: explicitly pin the incomplete revision; without a commissioner choice the newest complete revision now counts.
- TestStatQueueMakesAnItemOnlyWhereTheFeedChangedWhatTheLeagueCounted: explicitly pin the original revision to exercise manual-review deltas.
- TestStatQueueRejectedItemStaysOutUntilTheFeedChangesAgain: rejected suggestions concern an explicit commissioner pin, not a frozen default.
- TestStatQueueApprovalMovesTheBaselineAndUndoMovesItBack: undo restores the earlier explicit pin, not the retired revision-zero default.
- TestStatQueueNewestPositionChangeIsAnItemToo: a position change queues against an explicit commissioner pin.
- TestStatQueueNewestFirst: explicitly pinned players retain suggestion ordering coverage.
- TestPinnedStatApprovalKeepsCompleteBaselineAndQueueParity: explicitly pin the original evidence before testing approval/reopen; the ledger contains that extra pin.
- NativeStat fixture tests: revise now explicitly pins its original line before offering a competing revision, preserving manual approval/rejection/preview/permission coverage; corrections and silent-feed tests remain unpinned.

Limits: no new ESPN-to-player-stat acquisition adapter; authority policy accepts live revisions when
provided through existing evidence ports. nflverse weekly/plays and pinned nflverse evidence are tagged
at their existing producers. No GUI or fleet review. Full-suite inherited startup digest failure remains.


Bootstrap trial exposed a scaling defect (14,121 historical approvals; only 142 in 45 seconds).
Fixed by recording only the actual current selection on adoption, gathering held facts once and
checking/appending a batch under one immediate transaction. Checks use its locked starting snapshot:
feed approvals cannot alter permissions, rules, rosters or manual pins. Each append keeps its normal
sequential revision, canonical command, credential admission, event/audit writer and ledger readers.
Stat-only commands do not rebuild the unchanged roster recovery point. The trial's cancelled batch
appended zero events (atomic rollback). Final remaining 6,988 selections committed in **91.710 s**;
142 earlier trial records remain on the disposable copy. Initial bootstrap latency is debt, not claimed fast.
Later incomplete live evidence cannot downgrade a selected complete nflverse revision.
