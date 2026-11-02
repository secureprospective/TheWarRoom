# Handoff

## Baton
Bee → Christopher / ClaudeBox — 2026-11-02, `build/ROSTER-1`.
APP-ROSTER-12 complete locally; no remote git.

## Where it stands
- `a67aaece1`: W1–6 public NFL schedules held through the existing
  nflschedule.Fetch → MFL api → archive.Transport → history.Store path on a copy.
  Public response bytes / URL / fetch time / hashes committed with the evidence.
- `65dde3ab5`: original 74 at held first kickoffs: **73 active / 1 absent**.
  All 34 other-franchise rows active then; raw timestamps match their removals.
  **1,937/1,937** persisted source-command dates match the held transaction body.
  No transaction repair demonstrated or fabricated. Case Keenum W3 (18) acquired
  September 27, after the September 25 cutoff: Christopher ruling, rule unchanged.
- Complete held starters expose **26 conflicts / 20 franchise-weeks** (22 absent,
  2 taxi, 2 IR), including newly visible rows. Full named/datestamped list:
  `docs/build-handoffs/APP-ROSTER-12-cutoff-conflicts.tsv`.
- `b49ed4b78`: membership directly scans the projected snapshot, no unused
  per-player exception replays. Membership O(players + starters), same decisions.
  Fresh paired W7/8-only calls **11m19.887s → 2m58.877s**, 147 appended / 77 refused;
  complete-date calls **12m7.919s → 3m53.982s**, 204 appended / 20 refused.
  Both modes' entire refusal text and appended RawCommand JSON byte-identical.
  Schedule-free **75 → 75**, 2m48.019s after (before time inherited 10m22.422s).
  Product kickoff-aware refusals **77 → 20** by supplying dates, not changing rules.
- Fresh importer **3,338 / 8m36.294s**, repeat **0 / 552.128ms**; no importer change.
  Complete-date lineup repeat **0 / 15.437s**, same 20 refusals. Bills W1–7 recorded.
  Latest held reference membership/status equality **32/32** after final replay.
- Final focused race tests **46 test/subtest PASS** total (11 schedules + 35 roster /
  lineup / reconstruction), zero FAIL/SKIP. Franchise-filter mutant rejected.
  Plain make build and production hooks pass; protected files unchanged.
- Evidence: `docs/build-handoffs/APP-ROSTER-12-evidence.md`, rows/transfer-times TSVs.
  Ignored logs/profiles/comparison artifacts: `.tmp/roster12/`. Own disposable DBs,
  binaries and one-job harnesses deleted after process/open-file checks.

## Next move / blocked on
- Christopher: rule on the 26 first-kickoff conflicts. These establish status at
  the app's league-wide cutoff, not illegality at each player's own NFL kickoff.
- Existing fillScheduleDates can fetch these dates; committed evidence does not
  write them to the real profile. No live configuration or real MFL move occurred.
- Combined transaction+lineup worker deadline, actual startup/GUI and independent
  review remain unproved. Other projection/governance/recovery costs unchanged.
- Prior debt: off-roster missing-contract jobs; older leagues' transaction-follow
  setting. No full suite, remote, live config or protected-file edit.

## Tried and rejected
No invented dates, transfer-time repairs without evidence, rule weakening,
timeout increase, direct ledger writes or new provider. Removed unused work at
membership, not exceptions from the checks that actually enforce them.
