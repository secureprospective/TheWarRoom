# Handoff

## Baton
Review — 2026-10-29, `session/p9-s1`. Slice 1 complete; no push authorized.

## Where it stands
- MFL `lineup.set` → `mfl.lineup.set`; MFL `trade.accept` → `mfl.trade.accept`.
  Native names/authority unchanged. Frontend callers, receipts, supersession and
  watcher policies use qualified names; existing Wails signatures stay unchanged.
- Registry dispatches both new names into real local plans. All eleven MFL-context
  definitions audited; registry rejects shared native/MFL entries and unqualified
  MFL names. Negative tests exercise both violations.
- Legacy append-only move rows normalize only in `envelope.Restore`; both old names
  read/save without rewriting stored JSON. No old MFL command aliases.
- Endpoint CSV input updated; nativegen MFL/all-door outputs and endpoint projection
  regenerated. Wire fixtures regenerated through `TWR_WRITE_WIRE`.
- Flow, command inventory and remaining boundary scope:
  `docs/architecture/command-spine.md` (MFL command namespace).

## Evidence
- Filters listed before Go tests. Final affected-package `-race` results: 300
  top-level PASS (195 catalog/app/envelope/generator/import; 18 moves; 86 composition;
  1 history migration roundtrip). Includes offline import preparation and the
  existing fake-source import/review workflow. No live MFL requests.
- Vitest: 294 PASS across 34 files. Frozen install + frontend build PASS.
  `make lint` and `make bindings-fresh` PASS; boundarycheck unchanged (159 frozen
  functions/sources). No full-suite or GUI claim. Logs: `.tmp/p9-s1/`.
- Affected handwritten production: 18 files, 3,069 → 3,132 physical lines against
  starting HEAD; tests/docs/generated output excluded. No file moves/deletions.

## Next move / blocked on
- Independent review and disposable-profile UI acceptance: draft a lineup and accept
  plan, restart, see both plans and hand off without submitting anything to MFL.
- Later slices fence MFL packages; browser controls/unnamed legacy desktop bindings
  are not renamed here. What-if and watched plans remain frozen and visible.
- Untracked `_p9/` is the pre-existing planning scratchpad; `.lintcache/` is local
  gate output. Neither is part of this commit. No blocker.

## Tried and rejected
- Mixing all MFL definitions into offers' desktop view: lineup/trade belong to their
  existing feature bindings; filter by feature without changing Wails signatures.
- Rewriting stored plan names or retaining ambiguous command aliases: normalize on
  read only, preserving append-only JSON and native command meaning.
