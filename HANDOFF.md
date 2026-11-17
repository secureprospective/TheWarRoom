# Handoff

- Baton: Bee, 2026-11-17.
- State: Hosted bid place/raise/withdraw use private preview → hold-to-confirm → receipt, then reread owner bids and cap. Authority/football evidence is checked; own bids stay SEALED. Local draft/revise remain on `poolProvider`. Frontend: 116 test files / 891 tests, production build and bundle budget pass; targeted adapter tests (6) and `tsc` pass.
- Next: Commit/push, open PR, verify CI green.
- Blocked on: `tidy-check.sh` flags six dangling Chromium symlinks in `~/scratch/bee-resume-v14-{browser,render2}`. They are outside the authorized write paths and untouched.
- Tried/rejected: No new persistence, no local-provider route in hosted mode, no rival bid values/counts in War Room view.
