# Handoff

## Baton
Christopher — 2026-10-27, `session/revamp-P6-market`. Local only; do not push.

## Where it stands
- Part 6 slice 3: staged market Frame fold, rules, gate, trial and exceptions; ten entry-point preambles use the staged fold. External signatures unchanged.
- Behavior requires staged loading: waiver/market/deadline use the requested correction time; bid/trade do not. Some paths return before rules; trade reads governance before rules; market reads governance only for trades. Bid/trade/waiver trials preserve their different principals.
- NewFrame eagerly loads rules and governance; existing entry points use staged marketFrame to preserve those semantics. Frame.Book is still the private waiverState. Entry points are not yet thin wrappers around frame-taking operations; the intended shared-frame seam and line reduction are incomplete.
- Market-body goldens unchanged; targeted packages, composition market subset, commissioner race tests, build and bloat passed. No full verify.

## Next move
Review the staged Frame design and complete reusable frame-taking operations without changing authorization, correction time or error order. Then reassess line count and plan the market package move.

## Blocked on
No push authorized. Shared-frame API and ~70-line reduction are not achieved.

## Tried and rejected
- Unconditionally calling eager NewFrame from every entry point would change early-return/error precedence and correction timing; retained staged loading instead.
