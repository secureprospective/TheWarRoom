# README refresh: phone mock v1, desktop mock v3, live touchdown GIF
**Date:** 2026-11-17
**Branch:** docs/readme-mock-v1 (PR #339)
**Status:** ready for review. Merging to main needs Christopher's sign-off.

## What changed
- README rewritten top to bottom:
  - a new hero with download buttons for the phone mock v1 and the desktop mock v3 (zips in `docs/ui/showcase/v1/`);
  - a recorded GIF of the phone field showing a big-play touchdown;
  - fresh phone and desktop screenshots;
  - a new "Smarter Than Projections" section;
  - an updated readout and road map.
- Removed as stale: the old preview zip section, the "target" concept images, and the internal Trim section.
- The existing product sections were kept as they were on main (AI/CLI, import, model, server, native league, 32-team default, security, backups, architecture, licence).

## Why
Christopher asked for a full README refresh that shows the finished phone and desktop mocks, with downloadable copies and an animated field.

## Verification
- All 9 CI checks pass on PR #339, and it can merge cleanly.
- On the rendered branch page, every contents anchor resolves.
- All 7 images and both zips return HTTP 200.
- The downloaded phone zip matches the published mock (sha256 dd5740ce636d2ad7…).
- The mocks themselves pass their own gates: 9 mobile suites at 390 and 430 wide, theme, reach 56/0, and the 113-click sweep. The source and gates live outside this repo; see `~/archive/2026-11-twr-mock-v1/HANDOFF.md` on the Beelink.

## Open items / what Claude or Christopher should check
- The desktop mock v3 still shows projections in places. The README says so; a desktop catch-up to the phone's data rules is pending.
- The README claims carried over from main (137 commands, speed numbers, model holdout) were not re-measured in this pass.
- The old `docs/ui/showcase/*.png` and the preview zip are no longer referenced. They were left in place in case other docs link to them; they can be pruned in a follow-up.
