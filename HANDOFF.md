# Handoff

## Baton
Bee — 2026-10-27, `session/mode1-mirror`. Settings F2a implemented; ready for ClaudeBox review. No push.

## Where it stands
- Create records persist `source`; `LeagueSource` alone reads pre-source compatibility.
- Source selects native empty/OL-lot starts versus unchanged MFL held-data admission.
- Restore points retain source and preset; old points recover as MFL import / Legacy.
- Offline recovery uses `CheckNativeRecoveryAppend`, never a caller-controlled flag. Played native rosters recover without relaxing normal start admission; recovered copies stay read-only.
- Explicit valid source survives preparation independently of preset. Pre-source Create retries still match their original canonical request without rewriting history.
- App capture uses source; generated Wails models include the optional field.

## Evidence
- PASS: build all; vet internal; test inventory; requested package tests with and without race; lint (0 issues); bindings generation/freshness; frontend TypeScript.
- Native 12-team, native Legacy-rules 12-team, native Default32 and old/new Legacy 32-team recovery round trips preserve source, preset, exact contracts and the read-only fence.
- Mutation: reinstating unconditional empty-snapshot refusal fails both native-start and store-admission tests; mutation removed and final race batch passes.
- Existing MFL empty-seed and held-capture error text tested; refused append leaves ledger unchanged.

## Next move
ClaudeBox reviews F2a; live-test a 12-team empty-preset start and offline recovery on a throwaway profile. Passing: zero starting players/picks/pool; recovered source/preset/contracts unchanged and copy read-only.

## Blocked on
Nothing for this brief. No live GUI test or fleet coordination performed.

## Tried and rejected
- Recovery request-name bypass: a caller could reuse the name. Dedicated Go-only recovery admission keeps normal commands strict.
- Unconditional empty snapshot refusal: prevents native empty starts; retained only for MFL held capture/admission.

## Debt left
No F2a implementation debt identified. Live verification remains with ClaudeBox. Earlier unrelated project debt remains in the preceding handoff's git history.
