# Handoff

- Baton: Christopher, 2026-11-18.
- State: PR #320 is open against `main`, not draft, not merged. It records attribution for 2,482 502s during the 15 scripted stops (none outside), strengthens an origin 503/Retry-After integration assertion, and proposes—but does not apply—a line-14 measurement change. `make lint`, race tests on `cmd/twr-server` and `internal/serverhttp`, and pre-push verification pass. No production behavior changed.
- Next: Christopher reviews the gate-measurement proposal. If approved, rerun the public stress check on a build containing #316, recording the actual stop command and known origin-ready response separately. Do not merge without his go-ahead.
- Blocked: a stopped/killed origin cannot return HTTP 503; doing so requires a separately surviving proxy or edge fallback. `tidy-check.sh` also reports six pre-existing dangling Chromium symlinks under scratch outside this worktree; left untouched due scope.
- Tried and rejected: changing the Tunnel dependency cannot make an absent origin answer, and graceful shutdown cannot cover SIGKILL or a stopped CT. No live service was restarted.
