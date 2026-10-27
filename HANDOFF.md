# Handoff

## Baton
ClaudeBox / Christopher — 2026-10-27, `session/revamp-E-edge-scoring`.
Revamp E part 2 is plan-only. Do not push or rebase this branch.

## Where it stands
- Plan: `docs/build-handoffs/revamp-E-slices.md`, 23 serial slices, 25–40 min each,
  with exact paths, named tests, acceptance commands and SOL/TERRA ownership.
- No application code changed; no test suites ran. `git diff --check` passed.
- `git pull --ff-only` refused (no upstream); no alternate pull attempted.
- Part-1 requested stdout log was empty. Prior handoff/reproduction supplied context.
- This run independently computed five conditional weekly totals: Hunt 9.96,
  McCaffrey 48.27, Hurts 31.05, Keenan Allen 20.87, Dart 31.87.
  Plan embeds the executed research command, arithmetic and input fingerprints.
- Raw Sleeper checks matched all 14 fields for each selected row. p6 conversion-only
  PPG reproduced for all five. No passing Go golden claimed.

## Next move
Start E01. Work serially; every slice must compile/pass before the next.
Before E10/E19 acceptance, confirm the prior-season TFL numerator source.

## Blocked on
Ruling fixes weekly/prior-season/no-floor RTL, but not prior TFL source. Plan recommends
full prior REG nflverse PBP numerator / weekly carries at held league positions;
current-week TFL remains Sleeper. Golden totals explicitly depend on that choice.
Full prior Sleeper is not held (weeks 1–13 only); full-game completeness certification
and raw-nflverse base-event verification remain builder acceptance work.

## Tried and rejected
- Current-season/rolling rates or season settlement: superseded by weekly prior-season ruling.
- Rounded four-place residual multiplied by coefficient: existing exact multiply cannot
  represent arbitrary rational values; price the whole rational RTL item then round once.
- Unchanged p6 whole-score targets: p6 §5 publishes conversion-only PPG, not weekly totals.
- Silent missing-edge zeros, retroactive seed upgrades and mutable historical rate params:
  violate completeness/no-retroactivity; plan requires explicit evidence and writer freeze.
