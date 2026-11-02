# APP-ROSTER-8 — blocked on the same two boundaries

Starting and tested runtime ref: `cd66522c1`. No production changes. No remote,
MFL requests, live configuration, protected files or source database writes.
The prior uncommitted HANDOFF and APP-ROSTER-7 evidence are retained.

## Step 3: blocked, not delivered

Prepared reserve batching already exists and passes the focused real-store test.
Automatic import/sync invocation as the migration actor and a consumed ordinary
transaction-follow setting are still absent.

- `internal/leaguemigration/service.go:150`: import builds create/seed/answers and
  schedule only. Wiring import alone would not deliver the requested sync path.
- `internal/composition/migration_app.go:121–126`: the actual refresh calls the
  lineup importer here, outside the permitted edit boundary. This base contains
  no `syncPlayedLineups` body. Source search also finds no `schedule.follow` or
  `transactions.follow` setting. No side effects were hidden inside a read or
  lineup adapter to manufacture a sync route; no unused setting was added.

Required next input: authorize editing the actual refresh body, or supply the
integration base containing the permitted hook. Then wire both entrypoints and
consume the independent setting through the ordinary rule-change flow.

## Step 4: fresh replay attempted; full acceptance blocked

SQLite backup API copied all three databases from the authorized scratch profile
into one disposable worktree-local profile. Only the copied history store was
opened. The followed league started with two records and observation
`2026-11-01T03:09:10.095212087Z`. All **895** transaction rows validate.

Archive SHA-256:
- Players: `4c060f471e6ef4308c28ddec63fb6428d91cbed4f79f49104f3657b674c6ee0a`.
- Transactions: `9375de87e8537c9a68cc01f602e279997047052100047a2374b3ff37f0768592`.

Both calls to `mflsource.BackfillTransactions` refuse before writes:

> reconstruct TRADE at 2026-03-07T13:08:31Z: traded pick 2026 round 3 slot 0 from franchise.04 has no observation

First: **0 appended, 55.083133 ms**. Repeat: **0 appended, 57.481096 ms**.
Full record equality before/after both failures was checked. These are early
refusal times, **not completed importer times or successful replay idempotence**.
The harness initially needed its schema-version symbol and followed-league
selection corrected; the final calls above completed as reported.

Blocker: `internal/leaguemigration/mflsource/reconstruction.go:128`. The trade's
consumed original pick has no checked link to a seed slot;
`internal/leaguemigration/mflsource/prepare.go:288` records slot/owner without an
original-franchise identity. Need checked evidence or a commissioner ruling for
historical representation. No traded asset was skipped or identity guessed.

No new lineup runs were performed after this unchanged-code blocker. The last
measured results remain **inherited APP-ROSTER-7 evidence**, not fresh APP-ROSTER-8
acceptance:

| Measure | Last measured result |
| --- | --- |
| Schedule-free refusals | 76 → 76 |
| Kickoff-aware refusals | 107 → 107 |
| Membership/status equality | 28/32, not the required 32/32 |
| Bills schedule-free | Weeks 1–7 recorded |
| Bills kickoff-aware | Weeks 1–6 recorded; week 7 refused |
| Explicit archived week 8 | 0 refused schedule-free; 32 kickoff-aware |

The 0003/0018 additions and 0025→0015 player trade are post-seed transactions not
reached by replay. Exact prior refusal texts and importer/lineup timings remain
in `APP-ROSTER-7-refusals.txt` and `APP-ROSTER-7-evidence.md`.

## Verification and debt

`go test -list` discovered the four selected tests. Fresh targeted `-race -count=1`
batch: **16 test/subtest PASS, 0 FAIL, 0 SKIP**, over:
- `TestReconstructionRefusesUnprovedHistory`
- `TestMigrationPreObservationReconstruction`
- `TestMigrationTransactionSync`
- `TestPreparedReserveBatch`

Heavy checks passed before the jobs. No full suite, make verify, GUI or new build
was run; no production changes require a new build. Local logs:
`.tmp/roster8/test-list.txt`, `.tmp/roster8/race.jsonl`, `.tmp/roster8/replay.txt`.
Disposable DBs and harness were removed after process/open-file checks.

Debt: the two blockers above; automatic migration actor + follow wiring;
successful full replay/repeat-zero; fresh full proof in both kickoff modes;
32/32 equality and kickoff-aware Bills 7; durable off-roster missing-contract
review (already documented by APP-ROSTER-7). No new functional completion claimed.
