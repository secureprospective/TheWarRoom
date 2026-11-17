# Handoff

- Baton: Bee, 2026-11-17.
- State: PR #330 open: https://github.com/secureprospective/TheWarRoom/pull/330. Hosted Inspector now uses existing mode-aware age arc/usage adapters; Usage drawer added. Explicit server missing-input errors are treated as unavailable facts. Full frontend suite/typecheck/build pass locally.
- Blocked: First CI run found the test accessed `window.go` outside the adapter (frontend-transport gate). Changed test to assert adapter-owned Wails spies instead; pushed fix, awaiting CI rerun.
- Next: Confirm CI green, tidy-check, report.
