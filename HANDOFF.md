# Handoff

## Baton
Bee — 2026-10-27, `session/revamp-P6-draft`. Local only; do not push.

## Where it stands
- Step 0 is `1d0c42d6`. A mutant retaining the replaced pick player fails `TestDraftBodies`; restored it passes.
- `636f8a59` moves the calendar/job helpers and their shared close-record lookup into `draft_core.go`; commissioner and history race tests and the 71 selected composition tests passed.
- Draft package move is not committed. The experimental move was reverted; the working tree is otherwise unchanged.

## Next move
- Resolve the move's core/public-surface conflict: `admission.go` holds `*DraftPickInput`, but the design puts `PickInput` in draft, which cannot be imported by core. Decide how to preserve the intended type and generated bindings.
- Resolve the seam: `NativeStandings` has private `level`, `overrideLevel` and `regularWeeks`, and draft preview uses private `PreviewCatalogue` internals. The attempted move exposed roughly two dozen other core-private references. Decide a bounded core interface before resuming the compiler-led move.

## Blocked on
- A seam that satisfies the specified public surface, one-way imports and short shared-helper list without changing wire data.

## Tried and rejected
- Moving the files directly compiled neither draft nor history. Core references to `draftSetupValues`, `countWords`, `controlOfCommand`, and `DraftPickInput` remain load-bearing; moving only draft-named files is insufficient.
