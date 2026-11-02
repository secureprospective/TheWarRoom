# Handoff

## Baton
Bee → reviewer / APP-STALL-6 — 2026-11-02, `build/STALL-6`. Local only.

## Where it stands
- Revision-keyed season scores, concurrent sharing, live invalidation, caller isolation, facts reuse and held-key narrowing implemented.
- Targeted race: history 25 PASS; commissioner 16 PASS including subtests. Build and targeted lint pass; mutation test rejects removal of copy isolation.
- Desktop FAIL: HQ remains empty; Pulse data arrives late; idle CPU 746–769%. RSS peak 2,112 MiB; final countSeason stacks 0.
- Full evidence/source trace: `docs/build-handoffs/APP-STALL-6.md`.

## Next move / blocked on
- Final dump has 20 NativeLineupReading stacks: playersOf → ScopedDecisionExceptions → Replay repeatedly clones/hashes replay state. Fold shared legality inputs once per ledger/reading, preserve results, then repeat the desktop gate.
- Review before integration; no remote git operations performed. Desktop stopped; tool blocked disposable-profile cleanup, so operator cleanup remains.

## Tried and rejected
- Palette-only TargetWorkflow restriction: false diagnosis. The endpoint is the shared Home/HQ/Pulse transport; the palette does not prefetch workflows. Disabling it would remove screen data. No palette cancellation fix claimed.
- No scoring/valuation changes, screen-specific hacks, weakened gates or full-suite run. Inherited bloat: identical 29 tiny files versus baseline 24.
