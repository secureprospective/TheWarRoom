# APP-ROSTER-12

## Step 1 — held kickoffs

Used the existing provider, MFL's **public, league-independent NFL schedule**:
`nflschedule.Fetch(ctx, client, "2026", week)` → `mfl.Client.Do` (api host,
no L/APIKEY) → `archive.Transport` → `history.Store.Record` on a SQLite
backup-API copy of the authorized scratch profile. This is the same path used by
`fetchLineupWeek` / `fillScheduleDates`; ScheduleCache only reads the held bodies.
No new provider, owner page, remote git, live configuration or source DB write.
All six requests succeeded November 2, 2026. Each parsed and validated, returned
the requested week, and graded through `nflschedule.ToWeek`.

Public response bytes are retained in `APP-ROSTER-12-schedules/`, with requested
URL, fetch time, SHA256 and size in `index.tsv`; no league/private response is
included. These are evidence, not hardcoded application dates.

| Week | Games | First kickoff UTC |
| --- | ---: | --- |
| 1 | 16 | 2026-09-10T00:20:00Z |
| 2 | 16 | 2026-09-18T00:15:00Z |
| 3 | 16 | 2026-09-25T00:15:00Z |
| 4 | 16 | 2026-10-02T00:15:00Z |
| 5 | 15 | 2026-10-09T00:15:00Z |
| 6 | 14 | 2026-10-16T00:15:00Z |

Verification: test discovery followed by targeted `-race` schedule fetch/cache
checks: **11 test/subtest PASS**, zero FAIL/SKIP; plain `make build` passed.
Launch gate passed before heavy work. Logs in ignored `.tmp/roster12/`.
Historical classification and performance work follow separately.

## Step 2 — exact weekly classification

Read back all 3,340 records from a fresh ordinary transaction import of the
backup copy (2 → 3,340). `NativeAssetsAsOf` was exercised at each held week's
first kickoff, not today's state or a calendar estimate. All 74 per-week rows
are in `APP-ROSTER-12-rows.tsv`; held weekly bodies identify each as a starter.

| Prior current-roster class | Before rows | Active at week cutoff | Acquired later / absent at cutoff |
| --- | ---: | ---: | ---: |
| IR | 27 | 27 | 0 |
| Taxi | 1 | 1 | 0 |
| Another franchise | 34 | 34 | 0 |
| Unrostered | 12 | 11 | 1 |
| **Total** | **74** | **73** | **1** |

**No demonstrated misdated source move.** Re-translated the held 895-row
transaction body through `TransactionCommands` and matched persisted commands
by stable source request key: **1,937/1,937** source commands have the exact
held transaction time as Effective. Reconstruction baselines were excluded, not
mistaken for dated source moves. All 34 "another franchise" rows are active for
the named franchise at their actual week cutoff; their later transfers explain
the current location. Their dated transitions are retained per row. No invented
transaction, reserve-rule change or direct ledger repair was needed.

**73 rows addressed by supplying the missing football facts**, through the same
held schedule read / ordinary lineup admission. **0 transaction repairs.**

One of the 74 needs Christopher's ruling under the unchanged first-kickoff rule:
week 3, franchise.18, **Case Keenum / 10948**. At `2026-09-25T00:15:00Z` he is
absent. The held acquisition is `2026-09-27T04:32:28Z`; MFL lists him as starter.
This is absence at the app's league-wide first kickoff, not proof of absence at
his own NFL game's kickoff. The other 73 are not reserve/absent starters at the
app's cutoff. Prior week-7 Kirk / Terrell / Miller conflicts remain separate.
Full ordinary lineup replay and performance comparison are recorded below when
complete; the 74-row audit is not a claim that only those 74 can refuse.

Fresh importer on this copy, unchanged code: **3,338 appended / 8m36.294054093s**;
repeat **0 / 552.127885ms**, within the unchanged ten-minute deadline. Prior
APP-ROSTER-11 measurement was 7m12.678332031s; no importer change is claimed here.

The suspected transfers also have a separate row-by-row raw timestamp audit:
`APP-ROSTER-12-transfer-times.tsv`. **34/34** match the held transaction body's
numeric epoch and the persisted ordinary removal command's Effective, and are
active at the actual week cutoff. This checks the raw timestamp independently
of the per-week roster fold, rather than inferring that a later owner implies a
bad import date.

### Additional cutoff conflicts surfaced by complete dates

Checking *all* held starters, rather than only the first player named by each
refusal, identifies **26 player-week conflicts across 20 franchise-weeks**:
**22 absent, 2 taxi, 2 IR**. The complete dated/named list is
`APP-ROSTER-12-cutoff-conflicts.tsv`; its transitions include only exact matched
source commands, not reconstructed baseline removals. This includes Case Keenum,
the prior three W7 players, and 22 further rows previously hidden by another
player's refusal or by a current-roster check. W7 also includes Kris
Abrams-Draine / 16735, franchise.15, acquired October 25, 01:52:26Z.

These are ruling rows under the **existing league-wide first-kickoff check**.
They do not establish that MFL let someone start before acquisition at his own
NFL game's kickoff. No locked rule was changed to admit them. The completed
unoptimized ordinary lineup call with W1–8 held dates appends **204**, refuses
**20** (previous W7/8-only **77**); the original schedule-free **75** is a
separate mode, not a claim that holding dates changes a date-free check.

Step-2 verification: the focused race batch includes historical reconstruction
and week-lock boundary tests (**15 test/subtest PASS** for those two tests),
with 74 exact week classifications and 34 direct raw transfer timestamp checks.
Plain `make build` passed on the working candidate; no full suite or remote.

## Step 3 — linear active-membership check

Finding: `checkLineupRoster` called `NativeBase.playersOf`, which groups every
player and computes five scoped decision exceptions for each, replaying/scanning
the ledger repeatedly. This caller then discarded every exception and used only
ID/status. Its measured W7/8-only profile spends **511.09 cumulative CPU seconds**
in membership, **510.79s** in scoped exceptions. Tom's earlier 1,430s profile is
not substituted for this run's measured baseline.

Smallest change: read the already-projected snapshot directly, filter the named
franchise, construct the same active-ID map, then check starters in their original
order. The membership stage is **O(players + starters)** with no ledger replay.
No general exception implementation, shared cache, bypass flag, timeout, provider,
transaction importer or protected composition/Pulse file was changed.

Preserved flow: ordinary importer → ordinary ledger Execute / authorization /
preparation → CheckLineupAppend → already-projected current/as-of snapshot →
active membership → unchanged lock/lineup judgment → ordinary atomic append and
recovery. Error names and first-refused-starter order remain unchanged. Player
copies on another franchise cannot satisfy membership; IR/taxi exceptions still
cannot make a reserve player an active starter. General roster/cap/reserve
legality retains its exception computation. `checkLineupRoster` is called only
after the existing seeded-snapshot check; no new nil/unseeded contract is assumed.

### Paired full-profile proof

One backup-API snapshot after the ordinary 3,338-command import was used to make
independent before/after copies. Fixed store clock, same held players, scored
W1–7 weekly results and ledger prefix; one replay at a time. All calls use
`BackfillLineups` and the existing MigrationLedger/store. Football facts are read
by ScheduleCache from held bodies. "Old" hides W1–6 dates to reproduce the prior
W7/8-only environment; "all" supplies W1–8. No source DB was modified.

| Mode | Before elapsed | After elapsed | Appended before/after | Refused before/after |
| --- | --- | --- | --- | --- |
| W7/8-only (paired fresh) | 11m19.887266453s | **2m58.877046769s** | 147 / 147 | **77 / 77** |
| Complete held W1–8 (paired fresh) | 12m7.918708927s | **3m53.981768607s** | 204 / 204 | **20 / 20** |
| Schedule-free | 10m22.422436433s (inherited APP-ROSTER-11) | **2m48.018668836s** | 149 / 149 | **75 / 75** |

Paired reductions: **73.69% / 67.86%**. Complete dates give the requested product
refusal change **77 → 20**; optimizing alone changes no decision. Schedule-free
remains **75** by definition. All post-change calls are inside the unchanged
ten-minute worker budget; these isolated completed calls do not prove the GUI or
combined transaction+lineup worker's cumulative deadline behavior.

**Byte-identical paired results**, not just equal counts: `cmp` passed for the
entire refusal text (including final LF) and JSON serialization of **every appended
RawCommand** (all fields, including request/expected/reason/effective/visibility
and complete lineup payload). Newly generated receipt entry IDs are intentionally
not compared; they differ for independent appends. SHA256:

| Artifact | Same before/after SHA256 |
| --- | --- |
| W7/8-only commands | `fac0533c141e46e5bd2b664df587277136319227144976633bd4ca83f1623fd5` |
| W7/8-only refusals | `08e884f1a153eea48b8a2fefa899371bd4b931f0c62ba640d2abb7c5e9f7d49e` |
| Complete-dates commands | `bb2a41f0d4e18353d10b4a1cc4a3a0572725f7d5bbf717fd8daa8c1ff1d66826` |
| Complete-dates refusals | `845cb6fb21f3dda732f94ac1c2f60094785cc368e0955c167670c4037435e4d4` |

The schedule-free refusal text and the fresh old-mode baseline also match the
APP-ROSTER-11 artifacts; those older artifacts lack the final LF, so only that
legacy comparison normalizes the terminal newline. Paired new comparisons do not.
After membership consumes **0.02 cumulative CPU seconds** in the old-mode profile;
ScopedDecisionExceptions / decisionException disappear from that lineup profile.
Other projection/governance/recovery costs remain; no whole-engine linearity claim.

Full complete-dates repeat: **0 appended, same 20 refused, 15.43699276s**.
Bills W1–7 all recorded. After the final replay, current membership/status still
matches the latest held reference for **32/32** franchises. The old seed observation
alone matches 28/32; franchises 03/15/18/25 differ because of the already-recorded
post-observation source moves, not this optimization. This comparison used held
league/roster bodies via the existing MFL adapter and roster validator/normalizer,
with a local-only transport that refuses unheld requests; no further network read.
The final 74-row and 1,937 source-time audits reproduce unchanged after lineups.

### Verification and footprint

- Test discovery preceded targeted race batches. Final candidate: **35 test/subtest
  PASS**, zero FAIL/SKIP (15 historical reconstruction/week-lock; 20 membership,
  per-side exception, lineup authorization/legality/locks/corrections/idempotence).
  With step 1, **46** final-candidate test/subtest PASS. No full suite.
- New table tests cover active/IR/taxi/absent/other franchise, same ID on two sides,
  five eligibility exceptions, empty starters, exact first-refusal text and no
  membership input mutation. Existing real-store tests cover rejection with no
  append, ordinary success and repeat-zero.
- Franchise-filter mutant rejected by the new test; final source restored and the
  complete focused race batch passed again.
- Membership benchmark, 100 iterations: 100 / 1,000 / 3,000 players plus records:
  **9,172 / 105,223 / 243,548 ns/op**. This is a scaling observation, not a timing
  threshold or a comparison to an unmeasured old microbenchmark.
- Plain `make build` passed on the candidate. No binding/protected-file changes.
- Whole affected handwritten production file: **252 → 254 physical lines**;
  test file: **50 → 125**. No dependency, new package or size-reduction claim.

### Debt / limits

Christopher's first-kickoff ruling for the 26 rows (20 franchise-weeks) remains;
there is no demonstrated source transaction-date defect to fix. Actual background
worker/GUI and combined import+lineup timing are unproved. Existing general
projection/governance/recovery costs and other real consumers of scoped exceptions
are unchanged. Prior debt remains: off-roster missing-contract jobs and transaction
follow activation on older imported leagues. Independent review is not claimed.

Local commits: `a67aaece1` (schedules), `65dde3ab5` (classification),
`b49ed4b78` (linear membership). Production lint/secrets/ifaceguard commit hooks
passed without bypass. Final protected-file diff is empty. Own disposable DBs,
binaries and one-job harnesses were deleted after `ps` / `lsof` showed no users;
logs/profiles, paired comparison payloads/hashes and public schedule evidence
remain. No remote git, source DB write or real MFL move occurred.
