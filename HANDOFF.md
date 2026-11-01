# Handoff

## Baton
Bee / APP-GATES-3 — 2026-11-01, build/GATES-1. Local only; do not contact remote.

## Where it stands
- `709695a8` raises only the all-rule-form test timeout after the isolated test took 2.89s (88% in the test). Full Vitest passed twice: 161 files, 1751 tests each. The timeout cascade is consistent with this result, not independently proven as the only possible leak.
- Placement still fails: `internal/leaguemigration/mflsource/schedule.go:114` checks origin, and `frontend/src/app/shell/WarRoom.tsx:100` added the 20th native fork in `edc9a259`. No ratchet changed. Existing `internal/leaguerules/competition.go:3` schedule rules have weeks and pattern, not a reference-feed following setting; `internal/commissioner/migration.go:25` tracks cutover, but origin still decides who follows. Need a real setting and backward-compatible migration, not an accessor that hides an origin check. The calendar sources need a shared status path, not a syntax-only ratchet dodge.
- Boundary inventory has 15 changed/new fingerprints; not refreshed. Includes nontrivial later changes `31c71c7e` (value preparation) and `d0479ccb`/`3318ec6c` (league picker), in addition to migration/schedule/lineup wiring. Review each source and decision before accepting; `make boundary` stops at startValuePreparation.
- `make lint` stops at placement; ifaceguard and bloat pass. Plain `make build` passed (Wails reported missing model types but compiled and packaged). No Go files touched this session, so no targeted Go race test. Full Go race deliberately skipped. Preset archive remains not reproducible per prior handoff.
- `.golangci-cache/` was already untracked. No remote operations.

## Next move / blocked on
- Design and test explicit reference-feed following setting and historical league compatibility, including cutover and native leagues; unify calendar status from the existing clock/calendar readers. Then `make placement`.
- Review all boundary fingerprints against the branch changes; move policy inward where needed, refresh only reviewed wiring. Run `make boundary`, `make lint`, targeted race tests for changed Go packages, and plain `make build`.

## Tried and rejected
- Did not replace origin check with a differently named origin accessor or alter fork spelling; either would hide the violations rather than fix them.
- Did not blindly refresh boundary hashes; value-preparation and league-picker changes require review alongside intended migration wiring.
