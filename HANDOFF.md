# Handoff

- Baton: Bee, lane `pulse-b7upd`, 2026-11-18; base `bundle/b7` (`98121d5e0`, PR #378).
- State: server and Tunnel start hooks idempotently add `/app`; server start recovers a missing public hostname from Tunnel config without logging it. Recovery is root:root 0644; failure is nonfatal and `twr-status` explains the disabled browser door. Reused `twr-status`, already deployed by the old updater's fixed allowlist. Fresh install restarts the server after Tunnel provisioning.
- Verification: `make verify` passed after rebase. Pinned-Alpine signed install/upgrade/refusal/first-load integration, release tests, ShellCheck, shell syntax, and `git diff --check` passed; Alpine run preceded rebase, which only added the unrelated #378 browser-Origin change.
- Next: push branch, open one PR into `bundle/b7`, wait for CI, then report PR number and status.
- Tidy-check flagged unrelated active lanes `pulse-fixrace`, `b8shell`, `b9feed`, and `b9boards`; left their processes and files untouched. Never touch CT 260.
- Tried and rejected: a new helper would be absent during the first upgrade because the old updater installs only a fixed allowlist; reused the already-deployed `twr-status` instead.
