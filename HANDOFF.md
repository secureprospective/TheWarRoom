# Handoff

## Baton
Review — 2026-11-01, build/INTEGRATE-1101. APP-VALUE partially implemented; local commits only.

## Where it stands
- f26e00e3: app HQ/War Room inspectors read the existing held player age arc; WR age fallback.
- 71a4c3cf: app HQ roster/season/week projections feed Starting N and player inspector. Scope/availability preserved, no mirror-score fallback.
- `.app-value-state.md` records the initial per-region trace, implemented routes and missing regions. No model changes or refits.
- TypeScript PASS; whole Vitest PASS: 158 files / 1720 tests. Targeted Go race PASS: 24 test/subtest outcomes. Plain `make build` PASS. Hooks passed on both region commits.
- 14432 fixture reproduces 8-vs-5: unused dynasty keepers, unused hard-cap deadline, and an older unresolved later-round draft pattern. Actual imported ledger not inspected; draft-age explanation is inferred, not live-confirmed.

## Next move / blocked on
- Task incomplete: expose app-settings/roster model aggregation for Best N and player values; wire age distribution, next-season needs and lenses. Current UI placeholders exist on mirror too. THIS WEEK usage still needs app-scored matchup context.
- Existing cut quote and trade lineup delta remain wired. Actual imported cut-price absence still needs diagnosis; no made-up zero.
- No full Go suite, desktop walk, remote operations or pushes. Logs `.app-value-*`; pre-existing `.cr-*` and `.integ-*` untouched.

## Tried and rejected
- Reusing mirror research/asset boards wholesale leaks mirror scoring/ownership into app readings.
- Importing the WR market reader into HQ data creates command/provider initialization coupling. Shared projection parser now lives under data, with the old export retained.
- Eight-versus-five is not three missing universal rules: current Prepare filters irrelevant settings and learns draft kind; older stored settings do not change themselves.
