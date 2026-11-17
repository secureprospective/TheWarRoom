# Handoff

- Baton: Bee, 2026-11-17.
- State: On `f51mem` tracking `origin/session/pulse-f51b` (PR #307, stacked on #304). Changed installed `deploy/server/twr-sandbox` GOMEMLIMIT from 768MiB to 700MiB; `deploy/server/test-alpine.sh` asserts the installed sandbox contains the 700MiB setting. `make lint`, shell syntax checks and `git diff --check` pass. No Go packages changed, so no race tests apply. ShellCheck is not installed; Alpine integration test not run.
- Next: commit and push to `session/pulse-f51b`; comment on #307. Do not merge.
- Blocked on: none.
- Decision: set limit in the existing Bubblewrap environment rather than Go code; installation and upgrade both distribute this sandbox wrapper, so the default is applied to the hosted server without changing other twr-server invocations.
