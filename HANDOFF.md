# Handoff

## Baton
Review — 2026-11-01, build/INTEGRATE-1101. APP-VALUE-2 incomplete; local commits only.

## Where it stands
- 511bc901: Prepare and Jobs share applicability in `internal/leaguerules/applicability.go`; no stored rules changed.
- 69d0f8a8: calendar errors keep contract quotes refused but no longer erase known cut pricing. Frontend preserves priced zero; inspector uses the recorded owner and rejects ambiguous copies.
- `internal/store/history/migration_port_test.go` additionally proves Jobs=Prepare=5 after explicitly recording the same draft evidence; older ledger is 6, not 8.
- `.app-value-state.md` contains per-item status, source traces, cut root cause, verification and debt. Models unchanged.
- TypeScript PASS; whole Vitest PASS (158 files / 1,722 tests); targeted Go race PASS (12 named outcomes); plain `make build` PASS. Old Jobs count mutant rejected. No desktop walk, full Go suite, remote operation or push.

## Next move / blocked on
- Still unfinished: app-scored pts/game/dynasty read, Best-N over those values, age aggregation, and inspector lineup fit for both contexts. Existing model pass and lineupdelta engine must be exposed, not replaced.
- No implemented next-season-needs or your/their-lens evaluator found; skipped rather than invented.
- Actual live imported ledger not inspected. Fixture proves calendar coupling; missing pricing rules or contract terms still legitimately produce no price.
- No external blocker established: remaining value work simply is not implemented.

## Tried and rejected
- Mirror research/model scores are not a substitute for app-scored production.
- D11 projection medians are not the model's Now/dynasty output and cannot masquerade as Best-N model value.
- A closed or unresolved calendar must not authorize a move, but it need not suppress a known contract price.
