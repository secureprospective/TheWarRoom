# APP-ROSTER-11 — importer budget repaired; weekly classification evidence incomplete

Base `ab6fbe36a`. Worktree-local changes/copies only; no remote, network request,
owner page, real MFL move, live configuration or source database write.

## Historical refusals: what the held evidence establishes

**Exact historical classification is blocked, not complete.** The archived NFL
schedules contain 275 bodies, all for weeks **7, 8, 9, 13 or 14**. None supplies
week-1–6 kickoffs. The weekly-results body supplies starters, not an observation
of their active/IR/taxi membership at that week's first kickoff. Transaction times
are known, but mapping an arbitrary date to a fantasy-week cutoff would invent a
football fact. No calendar subtraction or lineup-as-proof-of-active inference was
used.

`internal/commissioner/lineup_week.go:144` selects `NativeAssetsAsOf` only when
`firstKickoff` is held. Otherwise the ordinary check uses the current roster.
That explains why importing the complete history did not clear these 74 rows.
It does **not** establish that the historical starters were illegal, nor that
supplying missing kickoffs necessarily clears every lineup.

All **74 / 74** named players are explicitly `starter` in the held MFL body for
that franchise-week. Current ordinary replay agrees with the latest held roster
for **32 / 32** franchises (membership and reserve status). Its named-player
mismatches with the historical starter body divide as follows:

| Current replay class, not historical-week status | Rows |
| --- | ---: |
| Same franchise, IR | 27 |
| Same franchise, taxi | 1 |
| Another franchise | 34 |
| Unrostered anywhere | 12 |
| **Total** | **74** |

- **Fixed historical rows: 0.** No missing/misdated move or wrong reserve transition
  has been demonstrated. Do not insert a fabricated correction to clear a refusal.
- **Proven reserve/absent historical starters needing a ruling among these 74: 0
  established**, not a claim that none exists.
- **Unresolved historical rows: 74.** Need held week-1–6 first-kickoff evidence (or
  an explicit different product ruling), then use the unchanged ordinary check.
- Every row, player ID, current location/status, held weekly `starter` result and
  dated replay transitions is in `APP-ROSTER-11-rows.tsv` (74 data rows).
- The 32 distinct named player/franchise pairs below are grouped only to avoid
  repeating identical transition evidence. The listed weeks account for all 74.
  **Their replay status at the actual week cutoff is unknown for every row.**

The transitions below were exercised through `commissioner.NativeAssetsAsOf` on
records read back from the completed ordinary importer, immediately before and
at each relevant held transaction time. Baseline repairs were not treated as
source transactions. Each pair has an active interval before its later loss or
reserve move; every listed transition matches the held transaction body. This
is stronger than using today's roster alone, but is not a substitute for a week
cutoff. `ROSTER` means active; `absent` means absent from the named franchise.

| Player / ID | Franchise | Weeks (count) | Current replay | Held replay transition(s), UTC |
| --- | --- | --- | --- | --- |
| Dean, Nakobe / 15821 | 03 | 1, 2, 3 (3) | franchise.08:ROSTER | 2026-10-01T05:40:41Z ROSTER->absent |
| All, Erick / 16645 | 04 | 1, 2, 3, 4, 5 (5) | franchise.04:IR | 2026-10-30T12:06:02Z ROSTER->IR |
| Pierce, Alec / 15768 | 13 | 1, 2 (2) | franchise.18:IR | 2026-10-03T03:50:11Z ROSTER->absent |
| Williams, Jameson / 15756 | 15 | 1, 2, 3, 4 (4) | franchise.14:ROSTER | 2026-10-11T02:22:30Z ROSTER->absent |
| Jones, Brandon / 14910 | 14 | 1, 2, 3, 4, 5, 6 (6) | franchise.32:ROSTER | 2026-07-27T01:59:17Z absent->ROSTER; 2026-10-22T15:51:21Z ROSTER->absent |
| Hunter, Danielle / 12221 | 09 | 1, 2, 3, 4, 5 (5) | franchise.10:ROSTER | 2026-10-15T22:36:04Z ROSTER->absent |
| Edmunds, Tremaine / 13697 | 10 | 1, 2, 3, 4, 5, 6 (6) | franchise.10:IR | 2026-05-20T23:16:04Z absent->ROSTER; 2026-10-25T14:37:52Z ROSTER->IR |
| Watson, Deshaun / 13113 | 06 | 1, 2 (2) | franchise.22:ROSTER | 2026-09-24T06:51:17Z ROSTER->absent |
| Horn, Jaycee / 15361 | 08 | 1, 2, 3 (3) | franchise.08:IR | 2026-09-30T16:02:41Z ROSTER->IR |
| Jones, Chris / 12741 | 32 | 1, 3, 4 (3) | franchise.09:ROSTER | 2026-04-09T00:32:06Z absent->ROSTER; 2026-10-10T06:41:51Z ROSTER->absent |
| Davis, Jalen / 14037 | 24 | 1, 2, 3 (3) | franchise.24:IR | 2026-02-24T01:05:45Z ROSTER->absent; 2026-06-19T16:18:27Z absent->ROSTER; 2026-09-30T20:11:01Z ROSTER->IR |
| Payne, Da'Ron / 13723 | 22 | 1, 2 (2) | franchise.06:ROSTER | 2026-09-24T06:51:17Z ROSTER->absent |
| Ojulari, Azeez / 15385 | 19 | 1, 2, 3 (3) | absent | 2026-10-03T03:50:18Z ROSTER->absent |
| Dart, Jaxson / 17032 | 17 | 1, 2 (2) | franchise.17:IR | 2026-09-25T00:18:56Z ROSTER->IR |
| Maddox, Avonte / 13767 | 20 | 1, 2 (2) | absent | 2026-05-24T05:05:28Z absent->ROSTER; 2026-09-24T07:41:25Z ROSTER->IR; 2026-10-01T06:45:17Z IR->absent |
| Miller, Von / 10267 | 18 | 1 (1) | franchise.20:ROSTER | 2026-05-31T22:40:51Z absent->ROSTER; 2026-10-14T11:16:22Z ROSTER->absent |
| White, Devin / 14168 | 25 | 1 (1) | absent | 2026-10-18T02:16:55Z ROSTER->absent |
| Rush, Cooper / 13124 | 18 | 2 (1) | absent | 2026-09-13T04:46:40Z absent->ROSTER; 2026-10-09T13:11:31Z ROSTER->absent |
| Brooks, Chris / 16387 | 25 | 2 (1) | franchise.25:IR | 2026-10-11T15:26:39Z ROSTER->IR |
| Downs, Josh / 16187 | 32 | 2, 5, 6 (3) | franchise.14:ROSTER | 2026-05-26T04:29:32Z absent->ROSTER; 2026-10-22T15:51:21Z ROSTER->absent |
| Keenum, Case / 10948 | 18 | 3 (1) | absent | 2026-09-27T04:32:28Z absent->ROSTER; 2026-10-15T21:51:10Z ROSTER->absent |
| Hodge, KhaDarel / 13963 | 13 | 3 (1) | absent | 2026-09-24T07:09:04Z absent->ROSTER; 2026-10-28T04:15:49Z ROSTER->absent |
| Odeyingbo, Dayo / 15386 | 30 | 3, 4, 5 (3) | franchise.30:IR | 2026-10-22T09:43:50Z ROSTER->IR |
| Chenal, Leo / 15831 | 25 | 3 (1) | franchise.25:IR | 2026-09-29T23:42:09Z ROSTER->IR |
| Williams, Tyleik / 17146 | 03 | 4 (1) | franchise.25:ROSTER | 2026-10-10T06:43:55Z ROSTER->absent |
| Robinson, Tavius / 16298 | 24 | 4, 5, 6 (3) | franchise.24:IR | 2026-10-27T03:27:25Z ROSTER->IR |
| Barnett, Derek / 13201 | 18 | 4 (1) | absent | 2026-10-21T19:23:06Z ROSTER->absent |
| Woodaz, Wade / 17621 | 15 | 5 (1) | franchise.15:TAXI_SQUAD | 2026-10-29T22:02:35Z ROSTER->TAXI_SQUAD |
| Irving, Bucky / 16598 | 18 | 5 (1) | franchise.25:ROSTER | 2026-10-18T02:16:55Z ROSTER->absent |
| Evans, Akayleb / 15880 | 08 | 6 (1) | absent | 2026-10-01T22:12:43Z absent->ROSTER; 2026-10-29T23:11:25Z ROSTER->absent |
| Williams, Kyle / 17134 | 18 | 6 (1) | franchise.32:ROSTER | 2026-10-22T19:04:49Z ROSTER->absent |
| Jones, Naquan / 15618 | 20 | 6 (1) | absent | 2026-10-22T17:27:20Z ROSTER->absent |

### Separately evidenced week-7 cutoff conflicts (outside the 74)

These remain Christopher rulings, not importer repairs. The unchanged held first
kickoff is `2026-10-23T00:15:00Z`:

| Week | Franchise | Player | Replay at cutoff | Later held move |
| --- | --- | --- | --- | --- |
| 7 | 15 | Kirk, Christian / 13633 | IR | Activated October 25, 15:43:02Z |
| 7 | 32 | Terrell, A.J. / 14918 | IR | Activated October 25, 02:48:42Z |
| 7 | 20 | Miller, Von / 10267 | Absent | Acquired October 25, 01:56:39Z |

Schedule-free additionally refuses All, Erick / 16645, franchise 04, week 7:
active at the held cutoff, IR only from October 30, 12:06:02Z. Supplying the held
week-7 cutoff already clears that row; no rule change is needed.

## Runtime finding and single change

Finding: the batch re-read, decoded and validated the entire growing event prefix
again after every ordinary command. The existing profile measured **196.97 CPU
seconds / 22.17%** there, plus overlapping GC. Correction projection was larger
(270.18 CPU seconds), but prefix rereading was a separable redundant cost with a
smaller correctness surface. No projection/governance shortcut was stacked.

Proposal/implemented flow: `ExecuteCommissionerBatchWithFacts` retains its private
checked prefix, reads only events after its current revision from the **same SQL
transaction**, validates those through the existing `scanEvents` /
`commissionerRecords` path, then appends that suffix to its private replay input.
Retries produce an empty suffix; subsequent commands still see all prior changes.
The existing full-read helper delegates to the same parameterized query with zero.
No shared mutable cache or assumed reconstruction of durable records was added.

Preserved: every ordinary admission, authorization, preparation and append;
parameterized SQL, ordered revisions, one writer, same batch transaction, rollback,
final recovery capture and receipts. A failed suffix read still aborts the batch.
The timeout and protected composition/Pulse files are unchanged.

### Measured replay

One SQLite backup-API copy of the authorized scratch profile; importer starts with
2 records and validates all 895 held transaction rows. Same input hashes and CPU
profiling as APP-ROSTER-10. The new run enforces **the unchanged ten-minute context
deadline** around the complete `BackfillHeldTransactions` call, including held-draft
lookup, reconstruction, admission and every atomic batch.

| Measure | Before (APP-ROSTER-10, inherited completed run) | After (fresh) |
| --- | ---: | ---: |
| Importer appended | 3,338 | 3,338 |
| Full importer elapsed | 10m43.484829317s | **7m12.678332031s** |
| Margin inside ten minutes | Failed by 43.485s | **167.322s** |
| Repeat appended / elapsed | 0 / 569.330031ms | **0 / 538.781359ms** |
| Final records | 3,340 | 3,340 |
| Membership/status equality | 32/32 | **32/32** |
| Missing-contract / unverified-pick jobs | 2 / 0 | **2 / 0** |

Elapsed reduction **210.806s / 32.76%**. Total sampled CPU **888.27s → 615.12s**;
profile percentages are overlapping cumulative CPU, not wall-time shares.
Remaining measured costs: correction admission 256.94s (41.77%), governance replay
122.77s (19.96%), contract capture 118.76s (19.31%), background GC 142.52s (23.17%).

The prefix-read mechanism itself falls **196.97 CPU seconds → 0.60 seconds**
(22.17% → 0.098%). JSON unmarshal under that path falls to 0.28 CPU seconds.
The large projection/governance work was deliberately retained.

Initial candidate measured 7m02.424914126s. Its first commit attempt hit the existing
40-statement function limit and a test append-style diagnostic. The suffix append
was extracted into `extendBatchRecords` and the test reused its batch slice, with
no gate suppression. Both gates then passed. The table above is the **fresh replay
on committed `b92ba6947`**, not the pre-extraction measurement in its commit message.
Final repeat still appends zero and remains inside the unchanged deadline.

## Verification

- `go test -list` discovery preceded focused race tests. Final production slice:
  **34 test/subtest PASS, zero FAIL/SKIP**, across 10 selected history tests.
  Covers prepared reserve outcomes, mixed retry/new-command batches, suffix/head
  reads, unauthorized/missing-fact/bad-second-command rejection and unchanged state,
  injected write rollback, final recovery, pre-seed reconstruction, historical
  picks, transaction idempotence, lineup idempotence/season isolation, per-side
  recovery and recovery-write failure.
- Mutant ignoring the suffix revision is rejected: the second ordinary command
  refuses instead of seeing the correct replay prefix. Restored candidate passes.
- Plain `make build` passed again after the gate-driven extraction. Existing Wails
  generic-discovery warnings remain; application compilation and packaging pass.
  No generated binding changes and no production binary launch.
- Production commit hooks (lint, secrets, ifaceguard) pass without bypass.
- Whole affected handwritten production files: **241 → 253 physical lines**;
  tests: **80 → 113**. No dependency, new package, or reduction claim.

- Additional focused closed-books batch: **8 test/subtest PASS**, covering persisted
  summaries, six invalid-summary fallbacks and atomic rollover failure. **42 total
  final-candidate test/subtest PASS**, no failures/skips. No full suite was run.
- Re-reading the final committed import's records reproduces the exact same dated
  transition audit. Seed payload unchanged; final roster equality rechecked 32/32.

## Fresh lineup replay on the committed candidate

Two independent backup-API snapshots of the completed import; one full historical
call per mode, strictly one replay at a time. Same held players, weekly results
and week-7/8 schedules as APP-ROSTER-10. No week-1–6 dates were supplied.

| Measure | Schedule-free | Kickoff-aware |
| --- | ---: | ---: |
| Refused before → after | **75 → 75** | **77 → 77** |
| First historical appended | 149 | 147 |
| Prior first historical elapsed | 10m25.269960987s | 10m31.033218601s |
| Fresh first historical elapsed | **10m22.422436433s** | **10m56.621151130s** |
| Every refusal byte-identical to APP-ROSTER-10 | Yes | Yes |
| Bills W1–7, including locked W7 | All recorded | All recorded |

**No lineup-speed improvement claimed.** These calls remain about ten to eleven
minutes on the expanded history. Other fleet race work was observed during the
lineup measurement period; launch gates were checked and pauses between jobs
honoured. These are observed completed times, not isolated comparative benchmarks.
The final importer above completed before those lineup calls and used its own
unchanged ten-minute deadline. Full automatic worker/GUI acceptance is not proved.
No full-profile lineup repeat or explicit week-8 replay was repeated here;
idempotence is covered by the focused real-store test, and APP-ROSTER-10 retains
its completed repeat/week-8 evidence. Importer repeat-zero is fresh.

## Artifacts and debt

Logs, test discovery/results, CPU profiles and transition text are retained ignored
under `.tmp/roster11/`; own copied DBs, raw bodies, one-job harnesses and binaries
are removed after process/open-file checks. Other agents' files and the pre-existing
lint cache are untouched. Local commits only; no push.

**Still blocked:** `internal/commissioner/lineup_week.go:144` cannot select an exact
historical roster without the missing held first-kickoff facts for weeks 1–6.
Supply those held bodies, then rerun the ordinary check before classifying a source
starter as IR/taxi/absent/acquired later or changing a recorded transaction. All
74 per-week classifications remain explicitly unresolved, not silently assigned
from current roster status. Three separately evidenced week-7 conflicts remain
Christopher's ruling. No locked lineup rule, timeout or roster state was changed
to force acceptance.

Remaining performance work: ordinary correction projection, governance replay and
GC; full historical lineup calls still exceed ten minutes. Actual background-worker
combined progress/retry and GUI/Pulse acceptance remain unverified. Prior debt:
durable off-roster blank-contract review and enabling transaction follow through
the ordinary rule flow on older imported leagues. Independent review remains open.
