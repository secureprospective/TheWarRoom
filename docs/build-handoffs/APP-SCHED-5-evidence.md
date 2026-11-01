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

## Item 2 — positions

`history.nativeEvidence` supplies the existing lineup/player facts' position by the same canonical
player ID already linked to NFL statistics. Scoring uses it only when the stat-feed position is
ambiguous/absent; it does not replace a specific feed position. The existing commissioner position
choice still wins. No name matching, player exceptions, guessed DB/DL split or invented stat rows.
Both ordinary and pinned evidence use this fallback. Tests exercise the store-to-evidence handoff,
specific fallback, ambiguous refusal, preservation of specific feed positions and commissioner choice.

On the copied profile, **300/300** previously position-blocked starter occurrences now have positions
(**93 distinct players**); truly unresolved scoring positions among those occurrences: **none**.
Absent-stat weeks (472 occurrences) and the unmatched identity (0835 Zion Childress, seven occurrences)
are separate gaps, not unresolved positions. Childress's held listed position exists; no verified
alias for that custom ID was invented. The scorer never manufactures inactivity/zero for absent data.

Specific NFL positions that disagree with league player records remain unchanged by this fallback.
That policy now has observable scoring/lineup consequences; see item 3. It is not a scoring-settings fix.

## Item 3 — final recorded-lineup comparison

SQLite backup API from the authorized scratch profile into `.tmp/sched5/profile`, not the installed
profile. Existing archive parsers and ordinary schedule/lineup backfill recorded the same 148 lineups;
76 historical-roster refusals remain excluded. Ordinary scoring acquisition supplied weekly/plays.
ProcessStatFeed committed the selected revisions through the stat ledger. Final native results and
standings were rerun **after** those durable approvals, including on the final position-helper refactor.
No substitutes for native totals, missing-stat zeroes, settings edits or scoring arithmetic changes.

Within .01 excludes exact. Differ requires a complete native score. Durable detail:
`APP-SCHED-5-comparison.tsv` (148 rows).

| Week | Recorded | Exact | Within .01 | Differ | Unavailable |
|---|---:|---:|---:|---:|---:|
| 1 | 15 | 0 | 0 | 3 | 12 |
| 2 | 15 | 1 | 0 | 2 | 12 |
| 3 | 18 | 1 | 0 | 5 | 12 |
| 4 | 22 | 0 | 0 | 6 | 16 |
| 5 | 23 | 1 | 0 | 5 | 17 |
| 6 | 24 | 0 | 0 | 4 | 20 |
| 7 | 31 | 1 | 0 | 4 | 26 |
| **Total** | **148** | **4** | **0** | **33** | **111** |

### Largest complete differences (app minus MFL)

| Week / franchise | MFL | App | Delta | Observed cause |
|---|---:|---:|---:|---|
| 4 Tampa Bay | 210.75 | 163.30 | -47.45 | -43.45 Lamb invalid-lineup penalty; Collins -2, Chaisson -2 |
| 5 San Francisco | 304.81 | 260.26 | -44.55 | -43.05 Warren penalty; Chubb -1.50 |
| 4 Denver | 222.20 | 180.50 | -41.70 | -38.20 Walker penalty; Bonitto -2.50; Schultz -1 not yet isolated |
| 5 LA Rams | 273.00 | 231.50 | -41.50 | -39 Nacua penalty; Van Ginkel -2.50 |
| 3 Detroit | 262.73 | 222.73 | -40.00 | Rousseau counts 32.50 before penalty vs MFL 39; zeroed (-39 net); Nwosu -1 |

The native settings require at least two DT and two DE, and zero the highest-scoring starter for an
invalid lineup. Existing position precedence retains a specific NFL-feed position even when the held
league player record differs: Tampa Bay's Collins/Chaisson score at LB vs listed DE (only one DE);
SF's Briggs scores DE vs listed DT (only one DT), and Chubb LB vs listed DE; Denver's Bonitto LB vs DE;
LA's Donald DE vs DT and Van Ginkel LB vs DE; Detroit's Heyward DE vs DT and Rousseau/Nwosu LB vs DE.
These observable positional-policy differences explain the slot failures and most residual IDP-rate
differences. They were not silently “fixed” to match MFL. Schultz's one-point residual needs stat/rule
item tracing; no fabricated scoring-setting diagnosis. No pure rate-setting mismatch is established.

**Bills:** MFL **6–1 / PF 1912.35** for W1–7. App **unavailable**: zero complete Bills weeks, so no
supported W–L or PF comparison. Standings' displayed 0–0 / PF 0 means no complete weeks counted,
not an observed losing/zero-point season. Zane Gonzalez absent W1–6; additional absent starter rows
W4, W6 and W7. Still 472 absent player-week occurrences and seven unlinked Childress occurrences
across the recorded cohort; no remaining position rulings or counted play-by-play gaps.

## Verification / remaining debt

- Full affected-package race run (commissioner, history, source adapter, sources): **961 named PASS /
  588 top-level PASS**, **9 skips**, **2 inherited/environment failures**. Startup digest remains
  `8a6936145d37309e` vs `90ca21fd230f0aa6` (clean-baseline proof retained in part 4). Catalog-only preset
  guard also scans the **pre-existing** `.tmp/sched4/baseline` archive and reports its old preset names;
  no production file violation. Neither failure was suppressed or weakened.
- Focused final policy/evidence/stat race batch: **51 named / 46 top-level PASS**. After extracting
  the position helper to satisfy cyclomatic-complexity gate: **11/11 final targeted race PASS**.
- **Plain make build PASS**, final production code. Commit hooks pass. Initial hook failures caught
  wrapcheck, function statement count and complexity; fixed, never bypassed. No production binary run.
- Final copied-profile proof PASS: **121.96 s** for seven native result readings plus standings and a
  repeated feed-ledger reconciliation (**2.193 s**, no new approvals). Initial adoption **91.710 s**;
  not a page-latency or fast-bootstrap claim. One-use composition/worker harnesses removed after caller search.
- Source-traced batch safety: one immediate write transaction owns the ledger snapshot and all appends;
  cancellation before commit leaves no partial batch. Manual pin checks use the locked snapshot;
  new feed commands cannot change it. Existing audit append, credential admission and revision sequencing
  stay in force. No independent reviewer or GUI validation in this session.
- Debt: absent weekly evidence/verified alias for 0835; specific feed-vs-league position precedence;
  Schultz residual; slow first ledger adoption and native result/standings readings; ordinary Sleeper/
  advanced-rushing and ESPN player-stat ingestion; first-publication scoring acquisition timing;
  source-health UI and earlier historical-roster/empty-lineup limitations. Full league parity not delivered.

Affected handwritten production Go footprint vs `3833e518`: **2633 → 2884 physical lines** across 15 files; tests/docs excluded. No reduction claimed. `internal/store/history/commissioner.go` remains over 600 lines because it is the existing cohesive atomic append/recovery pipeline; only its stat check and stat-only recovery decision changed.
