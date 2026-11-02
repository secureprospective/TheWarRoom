# APP-ROSTER-6 — groundwork delivered; acceptance not met

No complete task step is claimed. The historical replay and refusal cleanup are
still unfinished. The pending default is **not** a blocker: the brief explicitly
authorizes admitting an unknown contract without inventing salary. That admission
path was not implemented here.

## Intended slice and preserved guarantees

Checked transaction feed → `mflsource.TransactionCommands` (recorded franchise
mapping, ordinary command bodies, transaction-time effective instants, stable
requests) → `leaguemigration.SyncTransactions` → ordinary ledger authorization,
revision checks and atomic correction batches. Planning failures append nothing;
a failed transaction rolls back its entire group and stops replay. Earlier
successful groups remain durable. No source flag supplies authority.

`contracts.unknown_acquisition` → ordinary catalogue/decode/rule-change/resolve/
stored-value flow. It carries explicit years and fixed, lowest-minimum or
experience-minimum salary rules. Fixed salary must be explicitly supplied;
minimum rules cannot also supply fixed salary. Every preset leaves it undecided.
**Admission, pricing from this setting and commissioner review intake remain
unimplemented.** These setting tests do not establish those behaviors.

## Step status

1. **Partial:** all 895 checked archive rows translate: 116 LOAD_ROSTERS rows are
   observation evidence, and 779 other rows produce 1,600 ordinary commands.
   Free-agent and player-trade moves use roster corrections; pick trades use pick
   corrections; IR/taxi moves use reserve commands. Current draft slots now have
   an ordinary correction identity as well as future original-franchise picks.
   Selected/skipped picks still refuse ownership correction. Pick corrections
   support effective times under the same commissioner-only backdating rule.
   Successful correction transactions are atomic and repeat-zero in a real-store
   test; unauthorized and failed groups leave state unchanged.

   **Not done:** automatic import/sync wiring, migration actor integration and an
   independent transaction-follow setting. `schedule.follow` and
   `syncPlayedLineups` do not exist at the requested starting ref. The protected
   composition files were not changed. Reserve command groups still refuse the
   batch path, which deliberately admits only operations needing no gathered
   football facts/market preparation. It was extended only for roster/pick
   corrections, not by allowing unprepared reserve decisions.
2. **Not done:** `NativeAssetsAsOf` still rejects pre-seed reads. The first real
   transaction is the February 23 cut of player 15239 from source franchise 0018;
   he is absent from the November observation. Ordinary admission rejects it:
   **"This player is not on that franchise's roster"**. No transaction appends.
   The first transaction fails atomically on both attempts and both profile copies.
3. **Setting done; behavior not done:** the undecided setting is typed, catalogued,
   validated, stored, and resolved through the ordinary rule path. No default
   salary/years was chosen. Unknown-contract representation/admission, the exact
   "contract from import, unverified" queue item, and consumption of a later
   configured rule are absent. The existing no-invented-contract guard remains.
4. **Copied-profile rerun done; acceptance failed:** measurements below. All exact
   remaining refusal messages are in `APP-ROSTER-6-refusals.txt` beside this file.

## Backup-API proof

Both requested scratch databases were copied using SQLite's backup API into two
worktree-local disposable profiles. Each copied league began with two records and
observation `2026-11-01T03:09:10.095212087Z`. Bodies were selected from the copied
archive and parsed/validated by the existing ingestion code. No network request,
live-config access, clock substitution or direct ledger SQL write occurred.

Only the archived week-7/8 kickoffs were supplied in the kickoff-aware environment;
no dates for other historical weeks were invented. The reference comparison is
membership/status, **not salary/contract/pick equality**.

| Profile | Prior APP-ROSTER-5 refused | This run refused | First lineup import | Repeat |
| --- | ---: | ---: | --- | --- |
| Schedule-free | 76 | 76 | 148 appended, 40.679903912 s | 0 appended, 76 refused, 10.345994229 s |
| Held week-7/8 kickoffs | 107 | 107 | 117 appended, 36.377286255 s | 0 appended, 107 refused, 10.823849692 s |

Prior first/repeat direct lineup times were 40.639965552/10.446388703 s
(schedule-free) and 32.164532778/9.955029799 s (kickoff-aware). There is **no
measured refusal or runtime improvement**. These are complete direct importer
calls on a shared machine, not app-launch or integration-worker measurements.

Transaction replay, including planning all rows, stopped on its first group:

| Profile | First | Repeat |
| --- | --- | --- |
| Schedule-free | 0 appended, 56.291051 ms | 0 appended, 39.357197 ms |
| Kickoff-aware | 0 appended, 60.485238 ms | 0 appended, 42.023160 ms |

These times measure **early refusal, not a completed full transaction import**.
The copied-profile zero counts do not prove successful historical replay
idempotency. A successful two-command post-observation trade and repeat-zero are
exercised separately by `TestMigrationTransactionSync` against the real store.

**Roster equality:** 28/32 on both copies. The unchanged mismatches are source
franchises 0003, 0015, 0018, 0025. The same three post-seed transactions documented
in APP-ROSTER-5 still explain the differences; they were not reached by replay.

**Bills:** weeks 1–7 recorded schedule-free; weeks 1–6 recorded kickoff-aware and
week 7 still refused by the pre-observation guard.

**Explicit archived week 8:** historical backfill skips unscored results. A
separate harness selection of its starter sets, not a claim of final results,
produced:

- Schedule-free: first 32 appended / 0 refused in 9.788778395 s; repeat 0 / 0 in
  74.721735 ms.
- Kickoff-aware: first 0 / 32 in 1.621436291 s; repeat 0 / 32 in 1.649255914 s.
  Every refusal is the pre-observation-history guard, including franchise.18 and
  the Panthers/Dolphins/Saints. This is not a live week-8 or Pulse UI check.

## Verification and limits

- Targeted race batch: **144 test/subtest PASS, 0 FAIL, 0 SKIP** over migration,
  commissioner, league rules and history. The transaction grouping/rollback test
  initially failed because corrections were not batchable; it passes after the
  narrow ordinary batch extension. The rule-package batch also found missing
  catalogue choices/citations/golden additions; those were fixed, and the final
  batch checked the golden without its write flag.
- Final focused race rerun after extracting reserve translation: **12
  test/subtest PASS, 0 FAIL, 0 SKIP**, covering transaction translation, feed
  reorder identities, and real-store authorization/atomicity/repeat-zero.
  These overlap the earlier 144, not 156 distinct cases.
- Plain `make build`: PASS; its frontend build includes `tsc`. The packaged binary was never launched. Existing generic Wails
  discovery warnings remain. `frontend/wailsjs/go/models.ts` was regenerated by
  that build, not hand-edited. No frontend feature/Pulse source was changed.
  `git diff --check 750d1b2a4` flags 11 whitespace-only lines emitted by the Wails
  generator in that models file; generated bindings were not manually reformatted.
- Heavy checks that printed STOP were respected; another agent's race jobs and
  linter periodically occupied the machine. Commit attempts encountered ordinary
  gate failures (complexity, temporary staged/untracked schema mismatches and the
  shared linter lock); no gate was bypassed. No full suite or remote contact.
- Disposable database copies and the one-job proof harness were checked with
  `lsof` and process inspection, then deleted using explicit nonrecursive paths.
  Recursive deletion was tool-blocked; text evidence remains under `.tmp/roster6/`.
- Independent review, GUI acceptance, completed replay, pre-seed reconstruction,
  unknown-contract intake and 32-team equality are **not established**.

## Commits and source anchors

- `c2fabb074`: undecided ordinary contract rule and current-slot corrections;
  regenerated Wails models. `internal/leaguerules/contracts.go:65` registers the
  setting; `internal/commissioner/correction_command.go:211` checks pick identity.
- `3c616b695`: transaction translation and atomic ordinary correction replay.
  `internal/leaguemigration/mflsource/transactions.go:21` plans the feed;
  `internal/leaguemigration/transactions.go:15` owns replay.
- Remaining refusal/implementation anchors:
  `internal/commissioner/native_reading.go:40` (pre-observation refusal),
  `internal/commissioner/correction_projector.go:46` (no unknown-contract intake),
  `internal/store/history/migration_import.go:94` (reserve operations need a
  prepared path, not a setup-only batch allowlist entry).
- Handwritten affected Go production: **1,535 → 1,833 physical lines** against
  `750d1b2a4`; affected tests **48 → 369**. Generated models: **+91** lines;
  golden: **+21** lines. Counts include added files, exclude docs, and make no
  claim of debloat. No new package or dependency.

## Debt / next move

Implement the ordinary unknown-contract/review state and inverse-consistent
pre-observation projection, preserving monetary uncertainty rather than treating
unknown as zero. Extend ordinary prepared reserve batching (or choose an ordinary
atomic correction representation) without bypassing eligibility/preparation.
Then wire the transaction-follow setting and import/sync paths on the correct
integration base. Rerun successful replay/repeat, both kickoff modes, all 32
rosters and Bills 1–7. No pending policy choice excuses the missing implementation.
