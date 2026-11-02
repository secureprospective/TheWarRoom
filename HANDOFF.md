# Handoff

## Baton
Bee → ClaudeBox / APP-MERGE-10 — 2026-11-02, build/INTEGRATE-1102. Local commits only.

## Where it stands
- Report: docs/build-handoffs/APP-MERGE-10.md. **Partial acceptance; RSS ceiling fails.**
- 001825e9 streams depth parquet in 256-row batches, keeps only Research's newest
  snapshot, and joins overlapping per-player depth refreshes. Historical evidence keeps all dates.
- Baseline heap: ReadDepthHistory/parquet.Read accounts for 2,816.93 MiB of 3,476.27 MiB;
  21 concurrent TargetResearchPlayer calls. Trigger is cbad237c3's Home FORM Promise.all;
  whole-file parser dates to 5a1c3299, relocated in 152274c8. No binary bisect performed.
- Targeted race: 253 PASS tests/subtests, 4 fixture-gated SKIP, zero failures. Allocation
  regression: 152.6 MB before, 68.3 MB after; 80 MiB ceiling. Build, commit hooks and boundary PASS.
- Copied desktop survives >10 minutes, 774 live players, zero deadline log occurrences.
  War Room MY SHELF visibly shows Flowers, Metellus, Latu, Warren and Phillips by name.
- Real profile untouched; no memory cap or diagnostic code shipped; no task desktop remains.

## Next move / blocked on
- RSS at 30-second samples ranges up to 1.894 GB; startup one-second peak 2.655 GB.
  Full table in report. **Do not mark <1.5 GB acceptance complete.**
- Profile the repaired build's first 20 seconds and steady native reads. Baseline cumulative
  allocators also show heavy governance/native replay and evidence reconstruction churn;
  not yet a causal diagnosis of the remaining fixed-build memory pressure.
- make lint fails inherited bloat: ignored concurrent .worktrees/odds tree contributes;
  tracked-only old/new metrics remain provenance 0, tiny 28 (baseline 24), comments 5%.
  No other agent files or ratchet edited. Existing backup/model-alias warnings remain.
- origin is a local bundle. No pull/push attempted. Evidence stays untracked under
  .task-evidence/app-merge-10/; raw profiles/screenshots are disposable scratch.

## Tried and rejected
- The five-minute history worker cannot explain the reproduced two-minute spike.
- Streaming alone avoids the fatal spike but still exceeds RSS budget; coalescing helps
  without satisfying the strict ceiling. Neither is reported as complete acceptance.
- No GOMEMLIMIT workaround, authorization bypass, lost historical membership, skipped
  older-row validation, real-profile mutation or speculative player-name rewrite.
