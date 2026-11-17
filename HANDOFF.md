# Handoff

- Baton: Bee, 2026-11-17.
- State: Hosted HQ reads wired; player value/contract/cap/taxi/decision fields now render from existing hosted payloads. Inspector local reads are mode-gated; name joins are indexed; unsupported taxi columns removed. `pnpm test` (881), `pnpm build`, and `make lint` pass. No `frontend/dist` changes staged.
- Next: commit and push branch, open non-draft PR; do not merge.
- Blocked on: `tidy-check.sh` reports six dangling Chromium symlinks under `~/scratch/bee-resume-v14-{browser,render2}` outside this worktree; left untouched due scope.
- Follow-ups: existing HQ what-if drawer duplication and `hqReads.ts` subscriptions remain unchanged.
