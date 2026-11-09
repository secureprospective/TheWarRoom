# APP-SCORE-3 — durable live evidence

## Baton
Bee → integration owner, 2026-11-02, `build/SCORE-3`, base `7b684d65`. Local commit only.
Worktree: `.worktrees/score3`; the existing SCORE-2 worktree was left alone.

## Finding / flow / preserved contracts
Live fields and exposed merged revisions disappeared at restart. Validated provider lines → existing
ESPN/Sleeper merge → `SetLiveStats(ctx, ...)` → atomic changed-line capture + active board replacement →
`mergeLiveVersions` → durable exact chronology → existing commissioner selection/scoring.
Startup restores the board, latest held lines, every exposed revision and original final-origin keys.
Non-desktop evidence reads restore lazily too. No raw-field merge, scoring maths or feed ladder changed:
ESPN live, Sleeper gaps/outage fallback, nflverse field authority, commissioner last, no MFL counting.
`DropTotals` also runs on a detached line before capture; caller mutation cannot poison held evidence.

## Schema / effects
History migration **13**, through `historyMigrations()`:
- `live_evidence`: append-only captures, indexed by player/season/week/id; provider `espn|sleeper`,
  season, week, game, TEXT player id, fetched-at, digest, JSON held fields. Mixed ESPN/Sleeper lines
  retain the combined authority and per-field owners inside the JSON; their provider tag is `espn`.
  Digest includes authority and field attribution. Only a change from the latest digest inserts a row;
  returning A → B → A is a genuine third capture, not globally deduplicated away.
- `live_chronology`: append-only exact `LineRevision` JSON, immutable sequence and original final sequence.
  Persists original/live/derived revisions before returning them to score or approval callers.
- `live_board`: replaceable active-player pointers, committed with captures; an empty poll durably clears
  the board without discarding retained evidence. Poll metadata/pointers do not create evidence rows.

`liveMu` serializes capture/read/retention inside the store; SQLite transactions own durable changes.
Failed capture or chronology writes return errors before installing candidate memory. Polling logs failures
and does not publish a changed signal. Restore uses one read transaction and installs state only on success.
The approval worker now overlays live evidence for pinned/custom/archive feeds and live-only seasons;
retained, exposed earlier live-only weeks remain visible after the active board moves on.

## Retention
After the automatic approval batch commits, retention checks the actual counted score against the held
nflverse final. A pin to live evidence prevents compaction for that scope. Covered, superseded captures
never exposed as revisions may be removed. Keep the latest capture, active pointers, every capture whose
fields the final lacks (including event fields), and every capture used by any exposed live/merged revision.
All chronology remains immutable, preserving manual pins, automatic approvals and undone decisions.
Deletion is a transaction-scoped exception to the capture delete trigger; rollback restores rows/guard.
SQLite reuses freed pages; no VACUUM or immediate file shrink is claimed.

## Verification
Listed tests first; `go test -race -p 2 -json -run <filter> ./internal/<package>`; named tests/subtests,
not package PASS events. No full suite, desktop launch, real profile access or git remote operation.

| Package | PASS | SKIP |
| --- | ---: | ---: |
| store/history | 40 | 0 |
| app/now | 22 | 0 |
| fantasyscore/source | 19 | 0 |
| commissioner | 7 | 0 |
| composition | 3 | 0 |
| **Total** | **91** | **0** |

History filter: DurableLive, Live, TheFinal, AnUnchanged, AutomaticStatFeed, PinnedStat, NativeStat,
StatQueue, HeldLegacyWeeks, HeldWeeksScoreUnderTheDefault, SleeperLegacyEvidenceBytesAndDigests,
HistoryMigration. Source/commissioner: ScoreSource, FeedCounted, LaterLive, LineDigest.
Composition: TestStartupCapturesMatchedSetBeforeOpeningPools,
TestStartupCompletionWithoutFreshMirror, TestNativeStartupRefusesUnsafeOriginBeforeWindow.

New checks close/reopen the database and compare complete merged chronology, old merged pins, scores,
automatic-approval retry revision counts, durable board clearing, and live-only earlier-week worker evidence.
They assert no unchanged-poll insert, A/B/A capture, compaction retains referenced/gap/latest rows, the
retention trigger is restored, capture failure preserves memory/database, cancellation installs no partial
restore, no parts totals, detached evidence and failed-poll publication behavior.
A new detachment regression exposed a shared held-line pointer; it was fixed and the final batch rerun.

Changed-package golangci-lint: 0 issues. `make lint` is blocked by inherited tiny-file baseline
drift: 28 vs 24; a source-content count at base `7b684d65` is also 28. No baseline was raised.
Ifaceguard passed; the three file-length reports are unchanged files outside this slice. `make build VERSION=dev COMMIT=SCORE-3 BUILD_DATE=SCORE-3`:
PASS (existing Wails generic-model warnings). Build output was not launched.
Handwritten affected footprint vs base, physical `splitlines`: production 10 files, 1,568 → 1,956;
tests 5 files, 321 → 673. No generated changes; no debloat claim.

## Growth estimate / debt / next move
- Estimate, not a measured game-week workload: 16 games × 60 matched players/game × 30 actual
  line changes/player = 28,800 captures. At 1.5–3 KB per JSON copy and roughly one chronology copy
  per exposed capture: **about 90–180 MB/week**, plus indexes/final-derived rows. About 6–12 MB/game.
  Unchanged polls add zero captures/revisions; more actual changes scale linearly. Books are global,
  not duplicated per league. Conservative retention deliberately keeps more than pins alone require.
- No hard lifetime cap: all exposed chronology stays addressable. More aggressive reference-aware
  chronology retention needs a separately reviewed replay contract. Startup loads latest raw lines
  and all chronology; long-season/multi-season memory and latency have not been benchmarked.
- One active live writer per profile is assumed; independently active store instances do not refresh
  each other's in-memory book. Existing profile ownership is unchanged.
- SCORE-2's exhaustive upstream IDP/kicker/recovery coverage debt remains. No new upstream coverage claim.
- Independent review and Christopher's disposable-profile live check remain: pin a live/merged score,
  advance the poll, allow a final missing a live-only field, restart, and verify the pin and field survive.
  Tom's protected files and the SCORE-2 worktree were not edited.

## Rejected
No digest/sequence renumbering, per-poll evidence rows, full-history snapshots on every poll,
unconditional age pruning, removal of exposed revisions, provider totals inversion or silent storage failure.
