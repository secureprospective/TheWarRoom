# Handoff

- Baton: Bee, 2026-11-17.
- State: frontend-only slop task. Trade partner reads already use parser-checked standings rows; Plans already had one honest unavailable state/history; hosted bids already use hosted commands while local retains `poolProvider`.
- Changes: named the inverted filter helper `filterUnavailable`; used the requested Plans copy; added partner-read coverage for checked standings and separate cap scope. No Go, schema, or tdrafts-owned files changed.
- Evidence: frontend tests 122 files / 906 tests pass; `pnpm exec tsc --noEmit`, `pnpm build`, `make lint`, `tidy-check.sh` pass. Build emitted existing chunk warnings; lint reports existing oversized Go files, untouched.
- Next: commit, open PR, check final-head CI, record PR and CI state.
- Blocked: none. PR: none yet.
