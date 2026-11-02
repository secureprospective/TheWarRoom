# APP-ROSTER-9 — pick history and wiring committed; full proof blocked

Base: `1207d5d89`. Runtime acceptance is **not complete**. Local commits only;
no remote contact, real MFL requests/moves, live configuration or source DB writes.
Protected composition changes are limited to `syncPlayedLineups` and the transaction
import. No changes to `syncFollowedLineups`, live stats or Pulse.

## Delivered slices

1. **Historical picks:** `d9845a841`, strengthened by `91e32e292`.
   Checked draft comments identify original-franchise chains; the link must also
   match the held selection's player and franchise. March 7's original Miami
   (0004) 2026 R3 links to slot 1, used by 0027 for player 17529. The archive held
   93 distinct draft-results bodies; transaction bodies also retain the trade.
   Ordinary `correct.pick` now records historical ownership separately from live
   selectable/tradable assets. It does not change a selected live pick's owner.
   Missing links are explicitly unverified and create “Pick from import — verify
   it” jobs; an ordinary historical pick correction can record the checked link.
   Validation, immutable seed cloning, inverse reconstruction and atomic baseline
   all carry this history. Synthetic checked/missing-link, invalid-link,
   selected-pick preservation, jobs and real-store repeat-zero tests pass.

2. **Import and follow wiring:** `91e32e292`.
   Read/checked transactions travel through the source's migration plan. Initial
   import batches them atomically with creation/seed; subsequent import uses the
   *original recorded seed*, not a fresh observation. `transactions.follow` is an
   independent ordinary rule, editable through rule-change authorization; imports
   set it true, native defaults set it false. Existing older ledgers with no value
   require an explicit ruling/enabling change. The permitted played-week worker
   reads the setting and imports held-draft-backed transactions before lineups.
   It uses the commissioner's recorded principal and ordinary authorization, not
   a new privileged identity or source-kind exemption. Real-store import,
   follow-on/off and repeat-zero tests pass. Production worker invocation itself
   has not been exercised on the profile; it is source-traced, not acceptance.

3. **Batch runtime correction:** `4bfa36d1a`.
   Recovery capture was repeated for every intermediate uncommitted command.
   Ordinary batches now defer it to one final capture before atomic commit;
   standalone execution is unchanged. Each command still receives ordinary
   validation, authorization, preparation and append checks. A fault after the
   final command preserves both prior ledger and prior recovery point; successful
   batch recovery includes the final state. Focused race tests pass. Full-profile
   speed after this correction is **unmeasured**.

## Disposable-copy attempt and blocker

SQLite backup API copied all three authorized scratch DBs into a worktree-local
profile. The followed league began with two records. All 895 transaction rows
validate. The complete planner now succeeds:

- 4 groups / 3,338 commands; first atomic pre-seed batch: 3,332 commands.
- 85 distinct historical pick references, all 85 linked to checked held selections;
  zero unverified references in this plan. These are **planning counts**, not
  appended changes or job-board measurements.
- The first full replay, before `4bfa36d1a`, ran beyond ten minutes. Its uncommitted
  WAL had reached receipt revision 1,648 at 10m29s. It was terminated because the
  actual worker limit is ten minutes (`internal/composition/migration_app.go:161`).
  SQLite rollback was verified: the followed league still had exactly two
  committed records. This is not a successful importer time or repeat-zero proof.

**Current external blocker:** `heavy-ok.sh:12` reports disk **85%**, exits 1.
Global Go cache is about 95 GB, outside the authorized worktree. No global cache
was removed and the gate was not bypassed. After briefly freeing this run's DB
copy, targeted verification/commit ran at 84%; the next gate returned 85% again.
A proposed in-memory archive reduction was refused by append-only protection and
abandoned, without disabling protection or producing a modified fixture. No
archived evidence or business state was deleted from the source.

All this run's disposable databases, raw command plan and one-job Go harnesses
were removed after checking open file descriptors. Text logs remain in
`.tmp/roster9/`; the pre-existing lint cache and other agents' work were left alone.

## Acceptance measurements

| Requirement | Current evidence |
|---|---|
| Schedule-free refused before/after | Inherited 76; fresh after unmeasured |
| Kickoff-aware refused before/after | Inherited 107; fresh after unmeasured |
| Each remaining refusal | No new complete run; inherited exact reasons remain in `APP-ROSTER-7-refusals.txt` |
| Full transaction repeat appends zero | Synthetic passes; profile unproven |
| 32-franchise roster equality | Inherited 28/32; fresh after unmeasured |
| Bills 1–7 / kickoff-aware 7 | Inherited free 1–7, locked 1–6; fresh after unmeasured |
| Explicit week 8 | Inherited free 0 refused, locked 32; fresh after unmeasured |
| Full importer / lineup times | No new completed full importer or lineup run |
| Actual commissioner jobs created | Profile unmeasured; plan predicts 0 unverified-pick jobs; synthetic creates 1 |

The old four roster mismatches (0003/0015/0018/0025) have not been freshly
re-measured. Their three post-seed transactions are now represented in the plan,
not proven applied to the full profile.

## Verification and remaining work

Latest results for **47 distinct test/subtest names: PASS**, no remaining failure
or skip among those names. This is a de-duplicated union of focused batches, not
one full suite. Includes historical pick correction/reconstruction, ordinary
current-pick guards, catalogue tests, import/follow settings, pre-observation
reconstruction, transaction sync, prepared reserves and atomic batch recovery.
Earlier test attempts exposed a missing fixture reader and receipt-time test
errors; the corrected tests passed. All three production commit hooks passed.
`git diff --check` passed. No full suite, make verify, plain make build, Wails
regeneration, GUI or independent review ran on the final build.

Whole changed handwritten production files against base: 23 files, 4,837 → 5,240
physical lines. Tests: 5 files, 273 → 551 lines. No reduction claimed.

Next: operator frees disk outside this worktree; run heavy-ok. Then plain make
build (including generated bindings), final affected tests, and one full replay
on a new backup-API profile. Measure the optimized initial batch against the
10-minute worker budget; do not extend/bypass the protected worker lifetime.
Complete repeat-zero, both kickoff modes and every refusal, 32-team equality,
Bills 1–7 and explicit week 8, completed timings and actual contract/pick job counts.
Durable off-roster missing-contract review remains prior debt: current contract
jobs enumerate rostered players only. No full acceptance is claimed.
