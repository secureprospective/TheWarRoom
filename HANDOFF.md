# Handoff

## Baton
Review — 2026-10-31, build/HQ-wire. Local only; no pull/push authorized.

## Where it stands
- Locked v10 CSS/regions ported; existing editor, commands, reports and plan reads reused.
- Native HQ uses existing hosted reads, including moves, runway, reserves and selected cut quotes.
- No Go, generated binding, dependency or locked-mock changes.
- Full frontend: 1,638 PASS / 144 files; typecheck, transport, make build, bundle self-test and
  static/browser CSP PASS. Entry 148,925 / 321,818 bytes. No live Wails gate claimed.
- Coverage, exact gaps and suspected upstream issues: docs/build-handoffs/HQ-WIRE-report.md.
  State/read ownership and settings: docs/architecture/hq-wire.md.

## Next move / blocked on
- Review and disposable-profile mirror/native click-through. Exact slot-target/cascade/autofill
  parity, native plan persistence and the explicitly unavailable regions remain open.
- Do not treat passing compilation/tests as full locked-mock acceptance.

## Tried and rejected
- No mock sample numbers or invented missing projections/percentages.
- No calling signed adjustments dead money; no pretending all-history moves are a season ledger.
- No local MFL private/model data substituted for a native hosted seat.
- Bundle filename assumption replaced by graph verification, retaining eager-leak rejection.
