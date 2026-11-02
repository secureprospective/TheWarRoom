# Handoff

## Baton
Bee → ClaudeBox / APP-MERGE-9 — 2026-11-02, build/INTEGRATE-1102. Local commits only.

## Where it stands
- Report: docs/build-handoffs/APP-MERGE-9.md. Prior APP-MERGE-8 report remains unchanged.
- f9af9632 numeric ESPN week verified failing-before; copied desktop logs 774 live players.
- 6cc8ab15 fixes foreground refresh lock scope and repeated automatic-stat ledger folds.
  1,200-player/two-generation race regression now takes 7.46/8.52 s within 15 s each.
- 1239ba41 fixes painted accessible Home selection labels; abee0930 wraps the source line.
- Name resolver from a8a31c434 remains intact. Copied Home eventually shows named TOP 5
  players, live scores, week 8 and record/seed; final War Room shelf-name check remains open.
- Targeted race: 69 PASS tests/subtests, zero failures. src/app: 180 files / 1,833 tests PASS.
  tsc, boundarycheck (170 frozen functions/sources), plain make build and commit hooks PASS.
- Evidence: .task-evidence/app-merge-9/. Temporary diagnostic source removed. No task desktop
  remains running. Real profile untouched. No full Go suite or hook bypass.

## Next move / blocked on
- Five-minute acceptance NOT passed: copied desktop exits 137 around 2m15s; parent RSS reached
  4,252,340 KiB even on a second launch with experimental GOMEMLIMIT=1500MiB. Suspected
  memory pressure, not confirmed OOM (kernel records inaccessible). No panic/deadline logged.
- Profile peak memory during a fresh copied-profile launch, distinguishing live/read sources;
  then rerun uncapped five-minute acceptance and verify War Room shelf names.
- Held standings diagnostic exceeded 20 s in score-evidence/directory reads; CPU profile points
  at absence-depth reconstruction/scoring columns/GC, not yet a causal diagnosis.
- Inherited keyring/calendar, backup-entry and Wails model-alias warnings remain.
- origin actually points to a local bundle, not a receiving repository; no push attempted.
  .golangci-cache/ and .task-evidence/ remain untracked.

## Tried and rejected
- Initial failure precedes five-minute history worker; did not shorten its delay or bypass auth.
- Kept per-event credential admission, stat evidence, retries, audit and atomic rollback.
- Did not rewrite intact name resolvers or count early stand-in labels as a proven dropped source.
- A memory cap did not prevent termination; it is diagnostic only, not a shipped workaround.
