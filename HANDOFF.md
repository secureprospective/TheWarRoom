# Baton
ClaudeBox — 2026-11-14. L7 Part B ready for review; do not merge/deploy with publisher disconnected.

## Where it stands
Shared ESPN/Sleeper collector, bounded durable current-week raw checkpoint, immutable overlay at nativeEvidence; results/standings/odds capture its generation in flights and week-score memo keys. Pin digests stay stable. Provisional/source/as-of/reduced/stale metadata is returned. Restart replays normal validators. No historical chronology rewrite. SPIKE.md is intentionally untracked; its copy is in the lane run output.

## Evidence
Five changed-package race suites: 941 passing test/subtest events, 4 skips. Final overlay/collector acceptance: 24 passing events, including a later raw-final QB/WR lineup-total parity test. Lint: zero issues; nativegen check and frontend build passed. Affected handwritten production Go: 3,632 → 4,272 lines (+640); 133,446 → 153,475 Unicode characters (+20,029). No touched production file exceeds 600 lines.

## Next move / blocked on
Connect L5's PublishScores implementation to the nil startup seam and reconcile C5 using the reply's captured evidence.live, not a fresh LiveEpoch sample. Official finals acquisition = later; ESPN final-whistle packets stay provisional. Verify thin-window updates after connecting the publisher; no MFL comparison or deployment/load certification claimed.

## Tried / rejected
Reusing SetLiveStats would populate unbounded legacy chronology, so the server sink publishes a separate current-week overlay. No package move or copied collector: the existing now poller accepts a narrow evidence port. Legacy chronology and bracket-flight generation omissions are deferred slop; separate SourceIDs queries also need a coherent reference snapshot if identity repair overlaps polling.
