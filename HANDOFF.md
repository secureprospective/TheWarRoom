# Handoff

- Baton: Bee, 2026-11-18.
- State: PR #349 is open to `bundle/b2`, mergeable, CI green; commit `249ca70df`. No Home UI. The collector reads authenticated per-league scope, calls `readHomeBrief` once per native app league in parallel, isolates failures and sorts deadlines. Reference-only mirror is skipped because it has no native HomeBrief port.
- Tests: focused Vitest (7 passed); frontend `tsc --noEmit` passed; `make lint` passed; pre-push changed-package verify passed; PR CI all checks passed.
- Next: await owner review/merge decision; not merged.
- Blocked on: `tidy-check.sh` sees unrelated active b4/b5 lane launchers in `/home/chris/work/TheWarRoom`. Left them untouched; run folders/agent files are outside this task's write scope, so no `KEEP.md` was added.
- Tried/rejected: no server read or Home alert-band reuse; unavailable per-league reads become one item.
