# Handoff

- Baton: Bee, 2026-11-17.
- Previous: current main includes PRs #330 and #331 hosted player-usage facts.
- State: PR #333 https://github.com/secureprospective/TheWarRoom/pull/333 is open. Its first CI run exposed server-only queue commands leaking into NativeDoor parity/matrices; the catalog/generator now keep them on a separate server workflow surface. Full affected Go package tests, all 121 frontend files / 904 tests, typecheck, and `make lint` pass locally.
- Next: commit/push the CI fix, rerun PR checks, and close #323 with the #333 link.
- Blocked: CI dependency audit reports CVE-2026-112366 in picomatch 4.0.7/4.0.4 via Vite; npm reports no patched release and upstream PR #194 remains open. Do not bypass audit or pin an unreviewed patch; needs operator/security decision if still blocking after code fix.
- Tried/rejected: no schema change, shared league cache, commissioner-wide view, or local queue route.
