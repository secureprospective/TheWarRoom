# Ornith Phase 0 — Grading Record (training artifact)
**Graded:** 2026-07-11 by Claude (CT105). Task: chunk-map proposal for TheWarRoom review.
**Inputs:** 62-line dir-level inventory digest + chunking doctrine. Output: 7-chunk JSON.
**Runtime note:** attempt 1 failed — 3500-token cap consumed entirely by hidden reasoning
(Ornith is a reasoning model; `content` came back empty). Fix that worked: dir-level digest
instead of file-level (1.3k vs 4.3k prompt tokens), "keep hidden reasoning brief" instruction,
max_tokens 8000. Attempt 2: finish=stop, 2208 completion tokens, valid JSON first try.

## Score: structure B+ / completeness F → shipped map = Claude reference

### What Ornith got RIGHT (reinforce)
- Isolated transactions as its own high-priority chunk with every subdir enumerated.
- Treated ingestion as one external-data trust-boundary chunk (19 subdirs, correctly grouped).
- Tests as a shallow last chunk; risk-ordered sequence; est_tokens = LOC×12 applied correctly;
  all chunks under the 160k cap. JSON shape exactly as requested, no prose.

### Errors (each is a lesson)
1. **Coverage hole (critical):** ~10 packages never assigned to any chunk — internal/db,
   store/params, store/rulebook, schema, mfl, normalize, playerid, rankings, scouting, output,
   tools/. The doctrine said "every inventory file lands in exactly one chunk"; Ornith built
   7 plausible chunks and stopped without a coverage check. LESSON: after grouping, diff the
   union of chunk paths against the inventory and assign the remainder — completeness is a
   mechanical check, do it mechanically.
2. **Misclassified root Go files:** main.go/app.go/*_app.go/probe.go put in the CONFIG chunk
   ("root-level = config" heuristic). Those files are the Wails binding surface — the doctrine
   even named them. LESSON: classify by role stated in doctrine, not by directory depth.
3. **Binding-surface chunk under-filled:** internal/harness alone; the actual bound structs
   (App, in root files) excluded — consequence of error 2.
4. **Persistence mixed into engine:** internal/store/state (4.5k LOC incl tests) placed in
   business-logic-engine; store belongs with db (different risk profile: data integrity vs
   logic correctness).
5. **No db/persistence chunk at all** despite doctrine hint that DB corruption history matters.
6. **frontend/src root files (App.tsx, main.tsx) missed** by listing only subdirectories.

### Disposition
Shipped chunk_map.json = Claude's reference (chunk_map_reference.json). Nothing in the
proposal improved on the reference. Boundary-rules proposal graded below when complete.

## Boundary-rules proposal — grade: B+ (shipped with 2 corrections)
Attempt: single, clean (1731 in / 1020 out, finish=stop, valid JSON).

### Right (reinforce)
- Persistence-as-leaf, business-logic-never-imports-transport, ingestion-must-not-touch-engines,
  frontend-only-via-wailsjs, tests exempt — all 8 rules coherent, correctly reasoned prose.
- Correctly gave the binding surface broad backend imports (it IS the hub).

### Corrections (lessons)
1. **Missed shared-type granularity:** chunk-level rules false-flag ~38 legitimate imports of
   internal/domain + internal/schema (pure types used by every layer, but living inside the
   engine-and-domain chunk). LESSON: when a leaf TYPE package is nested inside a coarser chunk,
   boundary rules need a package-level carve-out, not just chunk-level edges.
2. **Too permissive on engine→persistence:** allowed engine-and-domain → db-and-store; real
   edges exist (3) and are exactly the should-use-interfaces smell. Tightened to must-not,
   confidence low, adjudicate.

### First live classification batch (validation)
Ornith correctly flagged all 3 store/rulebook→ingestion edges as HIGH layer_leakage with the
right rule citation and clean JSONL — matching Claude's independent adjudication list. The
judgment layer works.

## Pipeline lessons (for the arch-map skill + future Ornith drivers)
- Ornith = reasoning model: `content` empty + finish=length means thinking ate the budget.
  Always: dir-level digests not file lists, "keep hidden reasoning brief" line, max_tokens ≥2×
  expected answer, read reasoning_content for diagnostics (saved as *.reasoning).
- Observed throughput ~3 tok/s generation on the 680M — budget 10-20 min per 2k-token answer.
