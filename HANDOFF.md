# Handoff

## Baton
Review — 2026-11-01, build/INTEGRATE-1101. APP-VALUE-4 implemented for app leagues; local commits only.

## Where it stands
- Existing model pass/count/held-week reads exposed; math and mirror run unchanged.
- League-scored adapter feeds `value.read` / `TargetNativeValues`, app Best N + rank, dynasty value by age and HQ/War Room card values.
- Inspector lineup fit uses the shared D11/lineup read in app and local-reference contexts. Hosted sessions cannot select the local reference.
- Proof: imported 14432 settings/paid contracts with controlled 25-WR production; max Now/dynasty differences 0. Mirror golden 832 JSON bytes unchanged.
- TypeScript PASS; Vitest 159 files / 1,725 tests PASS; targeted race Go 23 tests / 44 named outcomes PASS; plain `make build` PASS. No full Go suite, desktop walk or remote operation.
- Full findings and verification: `.app-value3-state.md`.

## Next move / blocked on
- Review and visual gate against disposable data; no desktop launch was attempted.
- Full real-history parity, server-only player-record/params plumbing, copy-scoped valuations and free-agent lineup fit remain.
- Mirror age header, needs and valuation lenses remain unwired.
- Inherited gates: tiny-file count 26 > 25; boundarycheck flags unchanged `migration_app.go: migrationInputs`. Baselines not weakened.

## Tried and rejected
- Per-player pass alone misses paid-roster contract tilt: reuse the existing batch adjustment.
- D11 medians are not the model's Now/dynasty values.
- No origin branches, refit, guessed zero for missing evidence, or silent overwrite of duplicate player copies.
