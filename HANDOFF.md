# Handoff

- Baton: Bee, 2026-11-17.
- State: hqslop implementation is in `frontend/src/app/shell`: shared what-if drawer/ports and one season+move subscription utility. Caller inventory complete. `pnpm test` (881), `pnpm build`, and `make lint` pass. Full affected production source count is 289 → 294 lines (+5); existing files shrink 289 → 283, while the new shared shell adds 11 lines. This misses the brief's total-line-count-decrease criterion; disclose in PR body.
- Next: commit, push, and open the requested non-draft PR; do not merge.
- Blocked on: no technical blocker. `make lint` needed a 120-second retry because another golangci-lint run was active. Verify `tidy-check.sh` at close.
