# Handoff

- Baton: Bee, 2026-11-17.
- State: Current main merged; frontend frozen install/build passed. PR #312 has three sequential 180s origin/main runs and three branch runs from the same local harness. CPU decreased 6.6%; read errors were 0 in both sets. PR body has the table and is ready; no merge.
- Next: push this sync commit, verify CI/PR; do not merge.
- Blocked on: tidy-check flags six dangling Chromium symlinks in external scratch folders; left untouched because they are outside this worktree and authorized output paths.
- Tried/rejected: Earlier single-run evidence was not used for the final table; the six fresh run data sets were used. The untracked `cmd/twr-server/zz_pulse_repro_test.go` was preserved and not staged.
