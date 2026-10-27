# Handoff

## Baton
Christopher — 2026-10-27, `session/revamp-P6-market`. Local only; do not push.

## Where it stands
- Step 3b: ten frame-taking operations split from existing entry points, with staged construction, authorization, correction timing, early returns and trade governance-before-rules retained. Commit d662ac6a. No market package move yet.
- Targeted commissioner, history and leagueservice tests, composition market filter, hosted/worker filter, build, placement and bloat passed. Testdata diff from 78b2c4ad empty. No mutant or full suite run.

## Next move
Resolve the step-4 Frame construction seam, then move the judgment layer; preserve ledger visibility predicates in commissioner and run the sealed-bid mutant proof.

## Blocked on
The proposed eager `NewFrame` cannot represent the existing correction-time and error-order contracts: it always folds without the correction time and eagerly loads rules and governance, while owner readings authorize before folding and several paths return before those loads. Frame construction and staged loading must be exposed or orchestrated without changing those contracts before callers can pass a Frame across packages.

## Tried and rejected
- Unconditionally calling eager `NewFrame` changes authorization/error precedence and correction timing; retained staged loading instead.

## Staged Frame contract
- Callers authorize and filter records before NewFrame; command guards remain before folding.
- NewFrame folds once, loading neither rules nor governance.
- Optional correctionAt preserves requested correction time; omission uses the final record time.
- Operations call ReadRules/ReadGovernance at their existing stages, including early returns.
- Trades load governance before rules; Gate and Trial retain their existing timing.
- Records are transaction-held inputs; Frame does not write them or register callbacks.
- The fold, wire bodies, visibility predicates and core-called book helpers stay in commissioner.
