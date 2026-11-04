# Solo MFL path vs native engines: rule proof (2026-11-04)

Question: does the solo MFL path (`internal/mflref`) run its own copy of rules the native engines already own?
Method: scratch differential tests (not committed; text below), both judges fed identical inputs.

## Lineup legality: `mflref/lineup` vs `nativecheck.JudgeLineup`
- Rules: the real league's MFL starters block (`internal/ingestion/league/testdata/league-2026.json`),
  parsed by `mflref/lineup.ParseRules` and, separately, converted to native rules by
  `leaguemigration/mflsource.Settings` (the import path).
- Inputs: every lineup with QB 0-2, RB 0-4, WR 0-6, TE 0-4, PK 0-2, DT/DE/LB/CB/S 0-5 = **12,247,200 lineups**.
- Result: **0 disagreements** on "full lineup" (mflref Full vs native Valid) and **0** on "breaks a maximum"
  (mflref Legal vs native has no over-limit problem). 254 s.
- Caveat: mflref reads iop_starters as a cap, native as exact. They agree here because this league fixes
  the total (21) and the defence (12); a league without a fixed total could differ. Not tested.
- Verdict: **duplicate engine.** Two parsers of the same MFL starters block, same answers.

## Bid rules: `mflref/bid` vs `nativecheck.BidPolicy`
- mflref/bid hardcodes Legacy's numbers ($12M, $24M, 30 active, weeks 12/13). Native reads them from the
  league's RuleSet (`leaguerules.LegacyNFLForTeams(32)`), as the settings-congruence ruling requires.
- Inputs: salary $0-$40M step $50k x years 0-7 x experience 0-6 x active 0/29/30 x week unknown/0-18 x
  playoff unknown/in/out = **8,074,080 bids**; no bid to beat, no cap room (both skip those rules).
- Result: **39,200 disagree, one class only**: week 14+ (playoffs), playoff team, 1-year bid over $12M.
  Native allows it; mflref blocks it (one-year-cap + tier-years).
- Rulebook: `docs/league-rules/Official_Rulebook.md:171` "During playoffs: $12M cap on 1-year bids is
  lifted" (also Math_Rules_Reference.md:51, census CM-013). **mflref is wrong; native is right.**
- mflref/bid also has a cap-room rule with a conditional drop that native `BidPolicy.Check` does not
  (native checks cap elsewhere, not traced yet).
- Verdict: **duplicate engine with a live rule bug, and hardcoded league values.**

## Recommendation
The MFL path calls the native engines on rules from `mflsource.Settings` (the import conversion), and
`mflref/lineup` + `mflref/bid` rule code goes. That fixes the playoff bid bug, removes the hardcoded
Legacy numbers, and leaves one engine per rule. Needs: where the cap-room/conditional-drop check lives
natively, and what to do when Settings leaves a bid value "waiting for a ruling" on an unmigrated league.

## Outcome (2026-11-05, ruled "make it happen")
- Ruling on bids: an MFL league's bids are placed by hand, so the app checks no bid rules on the MFL
  path; bid plans stay open. `internal/mflref/bid` removed (5fb76ca4, 709c87d6, 60bdaf1c).
- Lineups: `internal/mflref/lineup` keeps no rules. Its limits come from `mflsource.LineupRules` (the
  import's translation) and its verdict from `nativecheck.JudgeLineup` (4fdc1783). The MFL screen keeps
  full / legal / problems, derived from the native verdict. Wording follows the native judge.
- Still hardcoded with Legacy values, not rules checks, left for a ruling: `engine/bid` bid points
  (multipliers, max 4 years) shown in the ProBoards post; minimum salary by experience shown in the pool
  table; `transactions/freeagency` what-if signing floor.
- The differential tests are kept out of the tree at `~/scratch/solo-proof/` (slow; one-off proof).
