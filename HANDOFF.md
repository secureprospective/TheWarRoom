# Handoff

- Baton: Bee, lane `mhold`, 2026-11-19.
- State: Implementation and generated binding surfaces are present on `session/pulse-mhold`, originally based on `bundle/b9`. The Q4 ruling and inventory (`act:trade-hold-24h`, `cr:trade-hold-reason`) authorize one commissioner-only native `trade.hold` command.
- Flow: `RawCommand` → preview/execute transaction → commissioner-only authorization → `PrepareTradeAppend` → append-only trade event → `TradeState` replay → owner-authorized `trades.read`. Standard `commish.correction.revert` remains the undo path.
- Decision: no separate review deadline exists in `TradeState`; reuse its recorded `ExpiresAt` and add exactly 24 h. Preserve the public reason in replayed trade state and league history. No freeze-between-teams or migration.
- Changed: native and CLI catalogs; commissioner/deputy policy; trade execution, projection, preview and eligible-target selection; hosted commissioner action; owner UI displays hold reasons for both teams. Generated `native.gen.ts`, `arguments.gen.ts`, and CLI verbs are updated.
- Verification: serverhttp race suite passed after revising the not-in-review case to use the same trade before acceptance. `make lint` now passes; preview logic was split into focused helpers to stay under the complexity gate. Commissioner race tests and frontend Vitest/TypeScript remain.
- Blocked on: `heavy-ok.sh` last reports one full `-race` suite running. Per lane rules, wait 120 seconds and retry before tests/lint.
- Tried and rejected: a second pending offer was unnecessary and used a nonexistent fixture pick; the not-in-review case now tests the same trade before acceptance. Treating Hold as `trade.appeal` or job snooze is rejected by the ruling.
