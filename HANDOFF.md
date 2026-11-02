# Handoff

## Baton
Bee → Christopher / ClaudeBox — 2026-11-02, `build/ROSTER-1`.
APP-ROSTER-9 implementation advanced; full acceptance blocked. Local only, no remote.

## Where it stands
- `d9845a841`: ordinary historical pick ownership, checked selection links or
  explicit unverified references + commissioner jobs; selected live assets unchanged.
- `91e32e292`: import and permitted played-worker transaction wiring; independent
  `transactions.follow` ordinary rule; original recorded seed on repeat import.
  Uses recorded commissioner authority, no privileged source-kind exemption.
- `4bfa36d1a`: ordinary batch recovery captured once before atomic commit, not for
  every uncommitted intermediate command. Rollback/final-state race test passes.
- March 7 original Miami 2026 R3 has held evidence: slot 1, player 17529, used by
  0027. Complete 895-row plan succeeds: 3,338 commands, first atomic batch 3,332;
  85 historical references checked, zero unverified in the plan.
- First full replay before the recovery optimization exceeded ten minutes;
  terminated and rollback verified (followed ledger still two records). No complete
  importer time or profile repeat-zero. Optimized replay remains unmeasured.
- Latest focused results: 47 distinct test/subtest PASS, hooks pass. No full suite,
  make verify, final plain make build, generated bindings, GUI or independent review.
- Evidence: `docs/build-handoffs/APP-ROSTER-9-evidence.md`; logs `.tmp/roster9/`.
  Own disposable DBs/harnesses removed after open-file checks; others' files kept.

## Next move / blocked on
- External heavy gate stops at disk 85% (`heavy-ok.sh:12`). Global Go cache about
  95 GB is outside the authorized worktree; no global cleanup or gate bypass.
  Operator must free disk, then heavy-ok, plain make build and final targeted tests.
- One new full backup-API replay; verify optimized batch fits the protected
  ten-minute worker lifetime (`internal/composition/migration_app.go:161`).
- Complete repeat-zero, both kickoff modes / each reason, 32-team equality,
  Bills 1–7 including locked 7, explicit week 8, full timings and actual job counts.
- Prior acceptance only: refused 76 free / 107 locked; roster 28/32, differing
  0003/0015/0018/0025; Bills free 1–7, locked 1–6; week 8 free 0 / locked 32 refused.
  These were NOT freshly re-measured. Exact inherited reasons remain in
  `docs/build-handoffs/APP-ROSTER-7-refusals.txt`.
- Older imported leagues need `transactions.follow` explicitly enabled through
  ordinary rules (new imports set true). Durable off-roster missing-contract jobs
  remain prior debt; current contract jobs enumerate rostered players only.

## Tried and rejected
No guessed draft origins, invented contracts, skipped assets, selected-live-pick
safeguard bypass, raw ledger writes, real MFL requests/moves or protected-file
workaround. A disposable in-memory archive reduction hit append-only protection
and was abandoned without disabling it. No speed/acceptance claim from planning,
rollback, a stopped replay or synthetic-only idempotence.
