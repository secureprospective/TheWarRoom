# Handoff

- Baton: ClaudeBox for review; Bee completed lane `pulse-b7upd`, 2026-11-18, branch `session/pulse-b7upd`, PR #381 to `bundle/b7`.
- State: server and Tunnel start hooks idempotently add `/app`; server start recovers a missing public hostname from Tunnel config without logging it. Recovery is root:root 0644; failure is nonfatal and `twr-status` explains the disabled browser door. Fresh install restarts the server after Tunnel provisioning.
- Verification: `make verify` passed after rebase; pinned-Alpine signed install/upgrade/refusal/first-load integration, release tests, ShellCheck, shell syntax, and isolated reconciliation tests passed. Alpine run preceded the unrelated #378 browser-Origin base update; deployment scripts were unchanged by rebase.
- Next: ClaudeBox reviews PR #381; merge after all PR checks pass.
- Blocked on: PR CI is pending.
- Tidy-check flagged unrelated active lanes `pulse-fixrace`, `b8shell`, `b9feed`, and `b9boards`; left their processes and files untouched. Never touch CT 260.
- Tried and rejected: a new helper would be absent during the first upgrade because the old updater installs only a fixed allowlist; reused the already-deployed `twr-status` instead.
