# Handoff

## Baton
Review — 2026-10-31, build/WR-wire. Local only; no remote contact.

## Where it stands
- WR-POLISH2 shelf spacing, compact contracts, dark scrollbars and header wrapping addressed; real-data screenshots in `frontend/.tmp/wrreal/after-*.png` (untracked).
- Fixture salaries now use millions-to-cents mapping; cap read remains unavailable.
- Partner positions remain missing. An attempted native assets enrichment was reverted: replacing snapshot player names caused composition correction and golden-wire tests to fail.

## Next move / blocked on
- Add identity as a separate read-only wire field without changing recorded snapshot semantics; cover mirrored sides, regenerate bindings and update wire expectations. Then recapture and verify partner positions.
- Verify full UI and contract values in the native app; fixture does not exercise cap.

## Tried and rejected
- No remote operations. Directly overwriting the snapshot's recorded names with current facts breaks snapshot preservation.

## Picker follow-up — Bee, 2026-10-31
- Local commits 9b11837d and 90dccf54 remove the rail's static source label and distinguish league choices using mirror provenance or app league ID.
- Still required: native HQ/War Room shell and franchise chooser, per-league persistence and authorization, plus the requested fake-port render test. Build, heavy gate and full suite have not run. No remote contact.
- Existing untracked `.golangci-cache/`, `.picker-state.md`, and `.picker-tmp/` were left untouched.
