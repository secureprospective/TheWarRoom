# Handoff

- Baton: Bee, 2026-11-18.
- State: PR #349 is open to `bundle/b2`, mergeable, CI green; commit `249ca70df`. No Home UI. This lane fixes #349's source-failure behavior: working source items remain visible alongside unavailable items for failed sources. People/whole-brief failure still yields one unavailable item.
- Tests: focused Vitest (8 passed); frontend `pnpm exec tsc --noEmit` passed; `make lint` passed; `tidy-check.sh` clean. One regression proves bids failure preserves working trades plus the source error item.
- Next: commit and push this fix to PR #349, then report; not merged.
- Blocked on: `tidy-check.sh` previously saw unrelated active b4/b5 lane launchers in `/home/chris/work/TheWarRoom`. Leave them untouched; run folders/agent files are outside this task's write scope, so no `KEEP.md` was added.
- Tried/rejected: no server read or Home alert-band reuse; unavailable per-league reads become one item.
