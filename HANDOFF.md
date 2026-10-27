# Handoff

## Baton
Claude (CT105) — 2026-10-27, main after the revamp E (edge scoring) and Part 6 market merges.

## Where it stands
- E1–E6 done: edge scoring, sources, play definitions and RTL are documented in
  `docs/architecture/default32-scoring.md`; code remains authoritative.
- Part 6 market steps 1–5 done. Sealed-bid visibility is pinned by `TestLedgerSealedBidVisibility`.
  Raise/withdraw and projection look up a specific bid ID, so they keep the indexed check rather
  than `ActiveFor`.

## Debt left
- Unknown Sleeper IDs need identity and roster coverage; do not assume they are irrelevant.
- Held play bodies without a `sack` column can count red-zone sacks as attempts. Repin those
  bodies before requiring the column; requiring it now rejects pinned evidence.
- The red-zone boundary includes the 20-yard line; Christopher has not confirmed it.
- Part 6 session 3: competition judgment (tie-breakers, playoff fields and brackets, schedules) is in `internal/engine/competition`; commissioner keeps aliases. Its tests still live in commissioner; move the pure ones into the engine. Wails now emits those seven wire types under the `competition` TS namespace (JSON unchanged).
- Market step 6 (retire frame wrappers) not started; the quote path stays staged.
- `contract_actions.go` writes rules fields and remains in core for the rules session.

## Tried and rejected
- Do not eagerly construct market frames: owner and commissioner readings authorize first, trade
  replay handles non-trade reversals separately, and quote loads governance → directory → fold → rules.
