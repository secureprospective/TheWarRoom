# Handoff

## Baton
Bee → Christopher / ClaudeBox — 2026-11-02, `build/ROSTER-1`.
APP-ROSTER-11 runtime delivered; exact weekly classification blocked. Local only.

## Where it stands
- `b92ba6947`: atomic batches retain their checked prefix and read only new events.
  Final full importer **3,338 appended / 7m12.678s**, down from 10m43.485s, under
  the unchanged ten-minute context deadline. Repeat **0 / 538.781ms**.
  Prefix reading **196.97 → 0.60 CPU seconds**; ordinary gates/recovery retained.
- **32/32** current membership/status equality; jobs **2 contracts / 0 picks**.
- All 74 W1–6 named players are MFL starters. Current replay classes: **IR 27,
  taxi 1, another franchise 34, unrostered 12**. Not historical-week classifications:
  no held W1–6 kickoffs. Dated replay transitions match held moves for all 32 pairs.
- Fresh refusals **75 → 75 / 77 → 77**, exact prior refusal text unchanged.
  Full lineup calls **10m22.422s / 10m56.621s**; Bills W1–7 recorded in both.
  No lineup speed improvement, fresh full lineup repeat or W8 proof claimed.
- Final focused race tests **42 test/subtest PASS**, zero fail/skip; mutant rejected;
  plain final make build and production hooks pass. No full suite or remote.
- Evidence: `docs/build-handoffs/APP-ROSTER-11-evidence.md`; all 74 rows:
  `docs/build-handoffs/APP-ROSTER-11-rows.tsv`. Logs/profiles: `.tmp/roster11/`.

## Next move / blocked on
- `internal/commissioner/lineup_week.go:144`: supply held W1–6 first-kickoff facts;
  then rerun the unchanged ordinary check. Historical fixed: 0; unresolved: 74.
  Do not treat today's IR/taxi/absence as proof of an illegal historical starter.
- Ruling-only W7: Kirk (15), Terrell (32) IR at first kickoff; Miller (20) absent.
  Their October 25 activations/acquisition are already correctly recorded.
- Next measured costs: correction projection, governance and GC; lineup calls still
  exceed ten minutes. Combined actual worker progress/retry, GUI and review unproved.
- Prior debt: off-roster missing-contract jobs; older leagues' transaction-follow setting.

## Tried and rejected
No invented cutoff, reserve-rule weakening, timeout increase, direct ledger write,
protected-file edit or source DB write. First perf commit hit statement/style gates;
extracted suffix append, retested/rebuilt/replayed final commit, passed without bypass.
