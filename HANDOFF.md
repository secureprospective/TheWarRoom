# Handoff

## Baton
Review — 2026-10-28, `session/pick-value`. No push requested or attempted.

## Where it stands
- `caca82bc`: Pick Value v2, four-class missing-slot denominator and non-increasing curve.
- `9ce6c4e8`: independent pick discount, 0.6 power setting, explicit original-franchise slots,
  native rookie-ledger replay/history reader; startup/skipped/extra/unknown picks excluded.
- `181f6447`, `fe5dbd8f`, `345c28c5`: picks add after the lineup, franchise only; Picks column,
  method/weight words, safe missing totals and held-only settings rebuild. Policy lives in services;
  shared season types replace duplicate wire structs. Source-only wiring reviewed; no new decision allowance.
- `b8becf8c`, `345c28c5`: Control Room > League settings use existing validated params commands/reset.
- Current flow and native integration limits: `docs/architecture/pick-value.md`.

## Evidence
- Approved run #5: 2,278 player Score fields replayed exactly, zero movers. The disposable replay
  request maps mirror ContractYear to PlayerState ExpirationYear; original snapshots stay unchanged.
- Go: 2,886 unique top-level PASS across batches. The full race invocation hit the tool's 15-minute
  limit with composition unfinished; its remaining 170 tests subsequently passed. Stale wire goldens
  corrected; practice-tool tests pass with a canonical temp directory.
- Frontend: 1,578 PASS / 133 files with two workers; build, 148,715-byte eager bundle and CSP checks pass.
- Full `make verify` invoked once: stopped at transport guard, NOT an all-green aggregate run.
  Fixed transport/policy placement; final lint, boundary, vulnerability, bindings, typecheck and build pass.
- Monotone-clamp mutant rejected at selection 4; restored source passes. Evidence under `.tmp/`.

## Next move / blocked on
- Native integration is partial: the loader reads a named native league correctly, including after
  restart, but the live asset/power composition still selects MFL. Do not pool unrelated histories.
- MFL's held future inventory has no explicit original-franchise slot order; it correctly uses round means.
- Review, authorize a fresh aggregate verify, and click League settings on a disposable profile:
  weight 0 removes only the franchise pick contribution; reset restores 0.6; season/player values stay fixed.
- No GUI acceptance or clean aggregate verify claimed. Initial untracked `.tmp-env.sh` preserved.
- Disposable `.tmp/pickcalc` harness remains: recursive cleanup was blocked by the tool's confirmation gate.

## Tried and rejected
- Inferring future slots from standings: not an explicit order.
- Keeping pick policy in composition or duplicating power wire types: moved inward/shared to meet gates.
- Long physical temp socket paths: unchanged socket test passes through a directory FD resolving inside
  this worktree. Practice-tool ownership tests require the canonical temp path instead.
