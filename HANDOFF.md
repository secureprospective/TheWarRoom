# Handoff

- Baton: Bee, 2026-11-18.
- State: PR #349 is open to `bundle/b2`; follow-up commit `61112089a` is pushed. No Home UI. Working source items remain visible alongside unavailable items for failed sources. People/whole-brief failure still yields one unavailable item. CI is running on the follow-up push.
- Tests: focused Vitest (8 passed); frontend `pnpm exec tsc --noEmit` passed; `make lint` passed; `tidy-check.sh` clean. One regression proves bids failure preserves working trades plus the source error item.
- Next: confirm PR CI completes; not merged.
- Blocked on: `tidy-check.sh` previously saw unrelated active b4/b5 lane launchers in `/home/chris/work/TheWarRoom`. Leave them untouched; run folders/agent files are outside this task's write scope, so no `KEEP.md` was added.
- Tried/rejected: no server read or Home alert-band reuse; unavailable per-league reads become one item.
