# Handoff

- Baton: Bee, 2026-11-16.
- State: Stable odds seed and bounded simulation cache implemented. Commissioner and history race tests pass. Real-data pulse run verified identical seed and all 32 odds rows before/after lineup writes and on repeated reads. Three baseline/after load runs plus one parity run are in `.scratch/`.
- Next: Project hooks block commit/PR. Restore the required frontend toolchain/build output through the project owner, then rerun hooks, commit, push and open a PR against main. Do not merge.
- Blocked on: `make lint` and commit hooks fail because `frontend/dist` is absent (`main.go:12:12: pattern all:frontend/dist: no matching files found`). `pnpm --dir frontend build` also fails: global tsc rejects existing `esModuleInterop=false` and `moduleResolution=node10`; local `node_modules` is absent. Do not edit `frontend/dist` or install dependencies without authorization. Tidy reports known dangling scratch symlinks and another agent's scratch folder.
- Tried/rejected: The records-length seed changed odds after lineup-only writes. Replaced it with a hash of simulation inputs; no contracts-cache change.
