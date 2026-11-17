# Handoff

- Baton: Bee, 2026-11-17.
- State: Hosted bid place/raise/withdraw use private preview → hold-to-confirm → receipt, then reread owner bids and cap. Authority/football evidence is checked; own bids stay SEALED. Local draft/revise remain on `poolProvider`. PR #329 is open; all CI checks pass. Commit: `276e4fac5`.
- Next: Christopher review and merge decision; do not merge without approval.
- Blocked on: `tidy-check.sh` flags six dangling Chromium symlinks in `~/scratch/bee-resume-v14-{browser,render2}`. They are outside the authorized write paths and untouched. No live league click-through was performed.
- Tried/rejected: No new persistence, no local-provider route in hosted mode, no rival bid values/counts in War Room view.
