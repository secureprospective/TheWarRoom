# Handoff

## Baton
Claude (CT105) — 2026-10-27, `session/revamp-P6-draft`, after merging main (E + market).

## Where it stands
- Part 6 market (steps 1–5) and revamp E (edge scoring) are on main.
- Draft: job A (rules owners, `RuleValueWords`, `TestDraftBodies` with 15 goldens, the
  draft_wire/apply/restore split), the `draft.change` bite fix `1d0c42d6`, and `draft_core.go`
  `636f8a59`. The draft package move was stopped on purpose at this green cut.

## Next move
- The draft package move joins the naming pass after Part 6 session 3. It needs a bounded core
  interface first: `admission.go` holds `*DraftPickInput`; `NativeStandings` has private `level`,
  `overrideLevel` and `regularWeeks`; draft preview uses private `PreviewCatalogue` internals; about two
  dozen other core-private references cross.

## Debt left
- Unknown Sleeper IDs need identity and roster coverage; do not assume they are irrelevant.
- Held play bodies without a `sack` column can count red-zone sacks as attempts. Repin those
  bodies before requiring the column; requiring it now rejects pinned evidence.
- The red-zone boundary includes the 20-yard line; Christopher has not confirmed it.
- Market step 6 (retire frame wrappers) not started; the quote path stays staged.
- `contract_actions.go` writes rules fields and remains in core for the rules session.

## Tried and rejected
- Moving only draft-named files compiles neither draft nor history: core references to
  `draftSetupValues`, `countWords`, `controlOfCommand` and `DraftPickInput` are load-bearing.
- Do not eagerly construct market frames: owner and commissioner readings authorize first, trade
  replay handles non-trade reversals separately, and quote loads governance → directory → fold → rules.
