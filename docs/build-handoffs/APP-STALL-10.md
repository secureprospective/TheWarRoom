# APP-STALL-10

Branch `build/STALL-10`, base `build/INTEGRATE-1102` (1b4e8187). **Desktop gate: PASS.** With Control Room held
open for 60 s, CPU falls to 0–9% after the first 10 s (it was 300–340%). Every screen shows data. RSS peaks at 1904 MB.

## Cause
Each Control Room reading priced every queue item from scratch: 9,047 open items on the real profile. Each item:
- built its own full copy of the 16.8k-record ledger (`scoreTrial`), and the GC paid for it;
- did an O(n) `sameLedger` compare to find the held fold;
- re-ran `withProposal` and the touched-franchise flip.

On top of that, `seasonStatFeed.Name` rebuilt the whole player-facts table **once per item** (11% of the remaining
profile). The stat feed appends approvals while the app runs, so every new revision paid all of this again.

## Changes
1. **Cross-reading price cache** (`commissioner/stat_price_cache.go`, `StatQueueReadingWith`; the store holds one
   per `Store`). An item's price (points and flip) is keyed by a SHA-256 over:
   - the item ID and its full held evidence;
   - the player's counted statistics (baseline, changes, corrections) and his before-score;
   - the item's week results before the decision (lineups and every starter's score, revision zeroed);
   - the reading's shared inputs: league, season, rules, directory, published schedule and every
     non-statistics ledger entry.

   A statistics decision for one player therefore re-prices only the items in that week. Words and names are
   rebuilt every reading; only the price is reused. Memory is bounded to the prices that the last 4 readings
   used or made. The old `StatQueueReading` (cache nil) behaves as before.
2. **One shared trial ledger per reading** (`trialLedger`). The held ledger is copied once with room for one
   proposal, and each item overwrites that last slot. Address-keyed memos now also check the proposal:
   - `RulesMemo.fold` compares the trial's last entry (`sameProposal`);
   - `heldFold` remembers the last held prefix by address and length, replacing the per-item O(n) compare.
3. **Names come from player facts read once per reading** (`held.Name`).
4. `flipBase.beforeWeek` is split out of `checkTrial` so the cache can digest a week. A cancelled read of a week
   is no longer kept in the base.

Equivalence: `TestStatPriceCacheMatchesFullPricing` compares the cached + shared-trial path with the reference
(no cache, a fresh ledger copy per item). The output must be JSON-identical across four readings:
- the first reading;
- the same revision again (the second reading must hit the cache);
- after an approval in the same week, which turns over another item's game;
- after a non-statistics entry.

Two mutations make the test fail:
- leaving the week results out of the key (`skipWeek`, built into the test);
- dropping `sameProposal` from `RulesMemo.fold` (checked by hand).

## Desktop gate (final tree, DISPLAY=:5, 1920x1080, one profile copy; CPU % over 5 s, 20 s after arrival)
| screen | lap1 | lap2 | lap3 |
|---|---|---|---|
| hq | 376 | 17 | 13 |
| pulse-now | 259 | 0 | 0 |
| pulse-race | 348 | 0 | 0 |
| home | 9 | 9 | 27 |
| warroom | 0 | 11 | 0 |
| tradefloor | 0 | 2 | 0 |
| control | 311 | 0 | 0 |

Control Room held open after Home had settled at 11%, sampled every 10 s: 140, 0, 0, 9, 0, 9. Home 20 s after
leaving: 11. RSS ranged 1709–1904 MB. StatQueueReading stacks at exit: 0.

A measuring build with a temporary log (not committed) showed the cache working: of four readings at one
revision, the first missed all 9,047 items, the next two hit all of them, and the fourth re-priced 1,086. That
fourth reading is the live week moving, which the week key picks up.

## Tests (`go test -race -p 2`; `-list` first, 62 top-level tests matched; counts include subtests)
- commissioner: 34 PASS
- store/history: 31 PASS

The run used `-run 'StatQueue|StatPrice|Rescored|TrialFold|TrialReplay|ScoreIndex|StatPreview|ResultFlip|RulesMemo|PlayerScore|StatFeed|Stat'`.
`make build` OK. No full suite. No binding added, so the door fixture is unchanged.

## Debt
- **Cold first pricing still costs about 20 s at around 300%.** Two cases pay it: the first Control Room open
  after launch, and any revision that moves a non-statistics entry or the rules (the "control lap1 311" above).
  Next steps: price lazily (visible rows first), or keep the cache warm from the stat feed's write path.
- **HQ/Pulse first-visit spikes (task 3, profiled, not fixed).** One profile sample (10 s) taken during a
  Control Room reading also shows `appvalue.scoreSeasons → addWeek → fantasyscore.countLine` at 21% of samples.
  That is the season valuation behind Pulse/HQ. It was not changed here, because a byte-identical fix needs
  its own equivalence work.
- The cache key trusts that a trial's `now`-dependence shows up in the week's before-results, the before-score
  and the rules. That holds for statistics approvals, the only proposal the queue prices.
