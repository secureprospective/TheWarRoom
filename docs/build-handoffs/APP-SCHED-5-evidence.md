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
worker authorization. Historical automatic approvals describe held revision history; recorded time is
ledger processing time, not the old source arrival time (held evidence retains arrival).

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
