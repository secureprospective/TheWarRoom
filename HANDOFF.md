# Hardening bugs handoff

## Baton
Claude/reviewer — 2026-10-22. Bee's implementation complete on `session/hardening-bugs`; not pushed, per brief.

## Where it stands
- Accounting: `internal/app/control/reading.go` accepts only full numbered MFL hosts; regression in `internal/app/control/room_test.go`. Commit `b98b9b34`.
- Private lineup: principal flows from the existing session through API/core/store and preview reads to `internal/commissioner/lineup_reading.go`. Current seat holder or the ledger's commissioner/co-commissioner privilege may read. Refusal returns zero lineup data. League-visible corrections stay public.
- Existing composition test extended for owner, commissioner, co-commissioner, other owner and unknown principal; ledger revision unchanged. Correction test preserves public access.
- Both regressions failed before fixes. Per-bug format/vet, touched-package race tests plus composition, internal golangci-lint, `make lint` and `make test` passed. Commit hooks passed for accounting; lineup hooks run at commit.

## Next move
Review both commits; run reviewer mutation/full verify and throwaway-profile live checks before merge. No push authorized.

## Blocked on / unknown
No implementation blocker. Full `make verify`, independent review and live GUI checks not run here.

## Tried and rejected
- Bare `www47`: not the producer's host format.
- Host-only lineup guard: pure reading already has records and must also protect preview reads.
- Making public corrections private: contradicts existing league-visible correction contract.
