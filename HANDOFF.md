# Handoff

- Baton: Christopher, 2026-11-17.
- State: PR #318 is open against `main`, not draft, not merged. Hosted HQ reads wired; player value/contract/cap/taxi/decision fields render from existing hosted payloads. Inspector local reads are mode-gated; player-name joins are indexed; unsupported taxi columns removed. `pnpm test` (881), `pnpm build`, `make lint`, and pre-push verification pass. CI/review pending.
- Next: review PR #318/CI; do not merge.
- Blocked on: `tidy-check.sh` reports six dangling Chromium symlinks under `~/scratch/bee-resume-v14-{browser,render2}` outside this worktree; left untouched due scope.
- Follow-ups: existing HQ what-if drawer duplication and `hqReads.ts` subscriptions remain unchanged.
