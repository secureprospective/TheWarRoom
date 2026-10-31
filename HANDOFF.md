# Handoff

## Baton
Review — 2026-10-31, build/HQ-D20. Local commit only; no pull/push authorized.

## Where it stands
- NativeCapRunway adds four seasons of committed/dead/cap/room cents through the existing TargetNativeCapCheck response.
- Existing engines/stores and locked mocks unchanged. Flow and input limitations: docs/architecture/cap-runway.md.
- Affected handwritten production: 139 → 213 physical lines; 5,320 → 8,066 Unicode code points including LF. Tests/generated/docs separate; no reduction claimed.

## Evidence
- Listed filters first. Final targeted race run: 17 top-level PASS, 33 including subtests; includes real cut/claim and physical reopen.
- Frontend: 1,597 PASS. make build VERSION=dev COMMIT=HQ-D20 BUILD_DATE=HQ-D20 PASS.
- Lint, vulnerability check, binding freshness and frontend typecheck PASS inside make verify.
- Full verify did not finish within 600s; a long private TMPDIR also broke one unrelated Unix-socket test. That test passed separately with default TMPDIR. Full commissioner-only race attempt also exceeded its 180s command budget; not claimed passing.
- Three cap goldens verified additive-only, with four reconciled years per 32 franchises.

## Next move / blocked on
- Review this read and its documented existing-data inconsistencies. GUI consumption/live gate and other room callers remain outside this slice.

## Tried and rejected
- No invented multi-year spreading: existing cut pricing charges remaining-year penalties in the cut year.
- No parsing adjustment descriptions as dead money. Use explicit kind or native cut entry identity.
- No new IPC door or store schema; extend the existing admitted cap read.
