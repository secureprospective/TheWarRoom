# APP-ROSTER-7 — useful implementation; replay acceptance still blocked

Starting ref `88398d4d4`. Local commits only. No remote, MFL request/move,
live-config access, clock substitution, direct ledger SQL write or protected
composition/Pulse edit. **This is not completion of APP-ROSTER-7.**

## Steps

1. **Ordinary reconstruction implemented/tested, real-feed replay blocked.**
   Checked transaction evidence is inverted into an earlier roster, then expressed
   as ordinary commissioner corrections. Baseline repairs and all pre-observation
   rows form **one atomic batch**; a failed reconstruction or append cannot expose
   a half-reconstructed present roster. Post-observation groups remain independent.
   Effective times are the transaction times; baseline corrections explicitly
   establish the state one nanosecond before the first supplied observation/move.
   Reads before the earliest recorded roster state still refuse. Compensation uses
   the same effective/receipt-time view as projection; pick-only entries do not
   establish roster history. The seed's body is never rewritten.

   LOAD_ROSTERS cannot simply be discarded. For player 16674, the checked archive
   records an October 29 acquisition on 0017, a LOAD_ROSTERS removal from 0017,
   then acquisition on 0018. Ignoring that membership observation made reverse
   replay fail. It now supplies membership evidence without inventing acquisition
   provenance or contract terms.

   Historical reserve **observations** use an ordinary commissioner roster-status
   correction; they are not owner reserve decisions priced against today's injury
   or experience facts. Owners gain no correction authority. Actual reserve moves
   retain ordinary preparation and eligibility checks (step 3).

   **Real blocker:** `internal/leaguemigration/mflsource/reconstruction.go:128`:
   `reconstruct TRADE at 2026-03-07T13:08:31Z: traded pick 2026 round 3 slot 0 from franchise.04 has no observation`.
   The transaction names `FP_0004_2026_3`; the seed has current-year draft slots,
   not this original-franchise identity. `mflsource/prepare.go:288` records selected
   current picks with slot/owner and no original-franchise link. The needed mapping
   cannot be inferred safely from franchise order or free-text draft comments.
   Selected/skipped pick safeguards remain intact. No source row is silently
   skipped, split out of its trade, or relabelled successful to bypass this.
   Both copied-profile attempts therefore append **zero transactions**.

2. **Blank admission, current-roster jobs and no-default policy done.**
   Ordinary roster admission with no recorded contract creates explicit missing
   state: salary and expiration are JSON null, distinct from a recorded zero salary.
   The job is **“Contract from import — fill it in”**, opening CM-034. Ordinary
   contract edit supplies salary/years, clears the job, and compensation restores
   it. Imported wholly blank contracts are marked missing at the adapter boundary.
   Cap/runway and monetary admission never treat these contracts as free players.
   Frontend parsers, card/contract displays, sorting and reserve totals preserve
   absence; generated Wails declarations are nullable through Go tags, not hand edits.
   The unused `contracts.unknown_acquisition` setting and its default-policy types
   were removed. No salary, years or default rule was invented.

   **Remaining limit:** jobs currently derive from rostered missing-contract
   players (`commissioner/missing_contract.go:53`). A historical player subsequently
   removed is not a durable off-roster contract-review item, and ordinary contract
   edit still requires a roster player (`native_projector.go:153`). That part of a
   complete historical review intake is not delivered.

3. **Prepared reserve batching done; automatic wiring/follow still missing.**
   `ExecuteCommissionerBatchWithFacts` uses the same authorization, credential
   admission, revision/retry, preparation, calendar, eligibility and cap path as
   single-command execution. Caller-supplied outcomes remain refused. Every step
   sees the preceding transaction state; failure rolls back the whole batch.
   `MigrationLedger` exposes held-fact/zone callbacks. A direct batch retry retains
   the original internally assigned expected revisions and writes nothing.

   **Not implemented:** import/sync invocation as a migration actor and independent
   transaction-follow setting. `leaguemigration/service.go:150` still appends only
   creation, seed, answers and schedule. At this base, `composition/migration_app.go:79`
   owns the existing refresh, but there is **no `syncPlayedLineups` body** and no
   `schedule.follow` setting. Editing that existing refresh would cross the brief's
   protected boundary. No callback was smuggled into an unrelated read to simulate
   wiring, and no unused follow setting was added without a working consumer.

4. **Backup-copy measurements done; acceptance failed.** Details below. The exact
   remaining 76/107 historical and 32 explicit week-8 refusal texts are in
   `APP-ROSTER-7-refusals.txt`. Before and after texts match byte-for-byte.

## Backup-API proof

Four isolated profiles (before/after, schedule-free/kickoff-aware) were copied via
SQLite's backup API from the authorized scratch profile, including all three DBs.
Only copies were opened. Each followed league began with two entries and roster
observation `2026-11-01T03:09:10.095212087Z`. Archived bodies were parsed/validated;
all **895** transaction rows were checked. Week-7/8 kickoffs came from the held
2026 NFL schedule; dates for other weeks were not invented.

| Run | Refused before → after | First lineup import before → after | Repeat after |
|---|---:|---|---|
| Schedule-free | **76 → 76** | 148 in 40.540 s → 148 in 40.323 s | 0 appended / 76 refused, 10.541 s |
| Kickoff-aware | **107 → 107** | 117 in 33.141 s → 117 in 33.407 s | 0 appended / 107 refused, 10.335 s |

These are whole direct lineup-import calls, not GUI launch/worker measurements.
There is no established refusal or runtime improvement. Fresh-copy first timings
were measured on `6bdc4de69`; final `c8991085e` repeated both modes and confirmed
zero new lineup entries, identical refusal texts, rosters and Bills outcomes.
The subsequent `cd66522c1` changes only generated nullable declarations/Go tags;
its runtime missing-contract JSON was rechecked with targeted race tests.

Transaction calls fail during reconstruction, before any append:

| Run | First / repeat elapsed | Appended |
|---|---|---:|
| Schedule-free | 49.886 / 47.916 ms | 0 / 0 |
| Kickoff-aware | 67.122 / 64.706 ms | 0 / 0 |

**These are early-refusal timings, not full successful importer times.** Repeat
zero after refusal is not historical replay idempotence. Successful replay/repeat
zero is separately exercised by the real-store synthetic reconstruction and
post-observation trade tests. No successful full transaction-import time exists.

**Roster equality:** membership/status remains **28/32**, not monetary/pick
 equality. The same three checked post-seed transactions explain every difference:
- 1793505951: adds 10976 to 0003.
- 1793506094: adds 15263 to 0018.
- 1793506821: sends 14467 from 0025 to 0015 for two future picks.
They were not reached. 0003/0015/0018 lack the additions; 0025 retains 14467.

**Bills:** weeks 1–7 recorded schedule-free; weeks 1–6 recorded kickoff-aware,
week 7 still refused. Explicit archived week-8 starter selections (not final
results): schedule-free 32 appended / 0 refused in 9.691 s, repeat zero; kickoff-aware
0 appended / 32 pre-observation refusals in 1.340 s, repeat identical. This is not
Pulse UI or a live in-play acceptance check.

## Verification and commits

- Final targeted batches: **193 distinct Go test/subtest PASS**, overlap removed;
  **0 FAIL/0 test SKIP in those final batches**. Package test discovery used
  `go test -list`; no project-wide suite or `make verify` ran. Phase batch counts:
  123, 78, 25, 55, 19, 14 (overlapping, not additive).
- Frontend: **66 PASS** in six targeted files. Type check passed; final plain
  `make build` passed, including frontend compilation. Packaged binary not launched.
  Existing generic Wails discovery warnings remain. All commit hooks passed.
- An inherited startup digest test failed on the initial selected run. A temporary
  source snapshot at exactly `88398d4d4` reproduced its old mismatch (`7c491213a4d198a6`).
  Restoring the removed default-contract setup job to the new result reproduces
  that same prior digest. The updated pin reflects the explicitly removed setup
  rule; draft/order/assets were not changed to appease it.
- Normal gate feedback was fixed: complexity/function limits, temporary staging
  separation and batch retry identity. No suppression or bypass. Heavy checks were
  run. Their process grep can count the invoking shell's literal test command;
  one early shell composition erroneously continued after such a STOP. Later
  checks ran as separate invocations and their STOPs were respected.
- Handwritten affected production, whole file set, physical lines against
  `88398d4d4`: **6,008 → 6,386** across 32 files. Tests: **1,292 → 1,809** across 16;
  generated models: **15,286 → 15,201**. No new package or dependency. No reduction
  claim: blank-state safety and reconstruction add behavior. The recursive contract
  prefix replay was replaced by a per-projection trail, collected only when needed.

Commits:
- `9aecbebc6` — blank contracts, ordinary jobs, frontend absence handling, default removal.
- `5c212545a` — ordinary reconstruction/status observations and guarded feed planning.
- `a2b74949c` — prepared atomic reserve batches and retry identity.
- `6bdc4de69` — retain removed contract history without rescanning unrelated entries.
- `c8991085e` — pre-observation compensation timing.
- `cd66522c1` — generated nullable contract declarations.

## Debt / exact next move

Obtain checked evidence linking the consumed 2026 original picks to observed draft
slots (starting with 0004/round 3), or an explicit commissioner ruling for how to
record that historical asset without pretending it is a current slot. Do not guess
or skip a traded asset. Then resolve the allowed sync integration base, wire import
and transaction-follow refresh as the migration actor, finish off-roster contract
review, and rerun successful full replay/repeat, both kickoff modes, 32 teams and
Bills 1–7. Independent review and GUI acceptance remain unperformed.
