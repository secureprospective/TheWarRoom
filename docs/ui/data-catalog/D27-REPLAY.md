# D27 replay — team needs / surplus (2026-10-31)

**Verdict: the method FAILS as a predictor of owner behaviour. Ship needs and surplus as
descriptive facts only.** Show "your Best 21 has 0 WR, the league typical is 2" and "3 players sit
outside your Best 21". Never word it as "you need a WR" or "critical", and never claim it as
advice.

## What was run
- `go run ./tools/d27replay -db <COPY of history.db> -dir ~/fleet/runs/d27-replay` takes about 4 minutes.
- Data:
  - League 14432's own MFL exports for 2023–2025: weekly rosters, trades and the detailed player listing.
  - The player record from `~/fleet/runs/sweep-1028/history.db`, read from a copy.
- Values: the shipped model, `modelrun.Runner` over a held book with `params.DefaultSet()`, using DynastyPPG. Inputs are as of each trade:
  - An offseason trade uses last season's finished record.
  - An in-season trade at week W uses this season through week W−1. Later weeks and this season's season-level rows are withheld.
- Rosters before each trade: take the week's export and undo that trade and every later trade in the window, latest first.
  - 64 of 457 trades failed the check that each side held what it gave, and were skipped.
- Needs and surplus: `internal/teamneeds.Compute` over Best 21 (the league's starter count).
  - The template is the lower median of the Best-21 counts per position.
  - A need is template − mine, and 2 or more short is "critical".
  - Surplus is every valued player outside the Best 21 with a value above 0.
- Bars were fixed before the run (MODEL_METHODS_DRAFT §3).

## Results
Sample: 393 trades where exactly one side paid picks; 96 team-seasons for needs.

| Bar | Pass at | Result |
|---|---|---|
| The pick receiver had more surplus at the positions it sold than the payer had at its own | ≥ 70% | **65%** (256/393, p<.001): FAIL |
| Critical needs at week 8 addressed by week 13 with a Best-21-grade player | ≥ 50% | **12%** (8/67): FAIL. The base rate at positions with no need is 9% |
| Pick-equivalence labels within one fifth-of-a-round band | ≥ 60% | **52%** raw (191/366), but 169 of those have both sides above the 1.01 pick, where every label matches; on the 197 informative trades **11%** (22/197): FAIL |

Diagnostics:
- Critical needs met by ANY added player at the position (no value bar): 21/67 = 31% (2023 17%, 2024 42%, 2025 33%).
- Week-8 template for 2024: CB 3, DE 3, DT 3, K 1, LB 2, QB 2, RB 2, S 3, TE 1, WR 2. It is stable across seasons within ±1.
- Rerun after the lint refactor gave identical numbers.

## Reading
- **Owners sell starters for picks.**
  - 67% of the players sold for picks were inside the seller's Best 21.
  - By contrast, 60% of the players the pick payer sent were outside its own Best 21.
  - Selling for picks is a rebuild decision, not shedding surplus. That is posture, and posture is the owner's pick (D28 ruling, 10-31). Surplus explains some of it: 65% is well above chance but under the bar.
- **Needs do not predict buying.**
  - A team short two at a position fills it with a Best-21-grade player in the next five weeks only 12% of the time, barely above the 9% base rate.
  - So a "you need X" prompt would be wrong almost nine times in ten.

## What ships
- Descriptive needs and surplus from `teamneeds.Compute`:
  - the counts against the league template
  - the players outside the Best 21
  - unvalued players, named
- No grade words. Comparing a team with the league's own shape is a fact.
- D32 TRIM/ASK/SWEETEN may still *prefer* surplus assets. The 65% result supports that as a tiebreak, not as a claim.
- **Pick equivalence does NOT ship.** Bar 3 (run 10-31) fits Pick Value v2 as of each trade (that date's model
  values, only league drafts held then, default pickvalue.discount) against the same DynastyPPG player values.
  - In 70% of one-sided pick trades (255/366) the model rates what the pick receiver sent above the picks (plus
    players) it got back; the gap is 5 or more fifths of a round in 119 trades. Picks-only trades match 28%.
  - The league prices picks far above the model's player-equivalent, so "worth about a late 2nd" would mislead.
  - Cells use the method's own fallback: picks and players are not compared; each shows in its own units.
  - Re-test if the pick model or the value model changes. Harness: tools/d27replay bar3.go + pickequiv.go.
