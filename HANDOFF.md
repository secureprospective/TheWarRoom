# Handoff

## Baton
Bee / APP-MERGE-8 — 2026-11-02, build/INTEGRATE-1102. Local only; no remote.

## Where it stands
- Items 1, 4–10 fixed; item 3 report-only origin findings documented, no source edit.
- Full report: docs/build-handoffs/APP-MERGE-8.md, including per-item commits and fixture explanations.
- Live stats retain game season/week, clear on empty success, preserve evidence on failure.
- Calendar drag uses league civil dates; Tokyo and DST cases tested.
- Rollover award was hidden by inconsistent test clocks, not removed; hosted reads now use
  the ledger clock. Award asserted present; only explained follow-setting fixtures regenerated.
- Final explicit nine-package race batch: 2,022 PASS tests/subtests, four existing opt-in skips;
  composition 846 PASS. Separate commissioner contract-alert checks: nine PASS.
- src/app: 180 files / 1,833 tests PASS. tsc, boundarycheck and plain make build exit 0.
- Build-generated game-scope bindings committed. Raw evidence remains in .task-evidence/.

## Next move / blocked on
- Operator full suite; independent review/mutant check and disposable-profile live acceptance.
- Item 3 origin-based startup/cutover rules remain debt requiring approved setting semantics.
- Wails still emits inherited missing-model-alias diagnostics despite a successful build.
- Existing .golangci-cache/ and .task-evidence/ remain untracked.

## Tried and rejected
- Did not accept empty rollover awards or weaken the as-of projector: aligned clocks instead.
- Did not regenerate fixtures until complete normalized diffs were explained.
- No engine/model changes, boundary exemptions, live-profile launch, full suite or remote contact.
