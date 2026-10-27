# Handoff

## Baton
Bee — 2026-10-27, `session/revamp-E-edge-scoring`. E2a stopped at investigation under the brief's data-loss rule. Do not push.

## Where it stands
E1/E3/E4 remain at 17bfb283. No production code or evidence format changed. E2a steps 2–4 deliberately not attempted: real scoring loss exists, not merely inactive/camp rows.

### Join investigation (weeks 1–13, regular season)
Read p5 `db_playerids.csv` and `stats_player_week_{2023,2024,2025}.csv.gz`, p6 `p6cache/sleeper_{year}_{week}.json`, and traced `p6_extract.py:bridges/sleeper`, `legacy_scorer.py:events`, `p6.py:load` in the supplied study runs. Numeric Sleeper IDs identify player rows; all nonnumeric IDs are classified as team/defense aggregates, as in the study. Counts are rows, not distinct players.

| Class | 2023 | 2024 | 2025 |
|---|---:|---:|---:|
| All dictionary-valued Sleeper rows | 28,101 | 28,936 | 28,982 |
| Team/defense aggregates (excluded) | 772 | 780 | 776 |
| Player rows | 27,329 | 28,156 | 28,206 |
| Study ID bridge succeeds | 21,307 | 21,775 | 21,018 |
| Study ID bridge fails | 6,022 | 6,381 | 7,188 |
| ↳ No crosswalk row, nonzero scoped edge stat | 3,233 | 3,347 | 3,496 |
| ↳ No crosswalk row, all scoped edge stats zero | 2,687 | 3,014 | 3,680 |
| ↳ Crosswalk has MFL + Sleeper, missing GSIS, nonzero edge | 22 | 6 | 12 |
| ↳ Crosswalk has MFL + Sleeper, missing GSIS, zero edge | 80 | 14 | 0 |
| Missing MFL / other bridge-failure cause | 0 | 0 | 0 |
| Bridge succeeds AND nflverse player-week exists | 11,940 | 12,195 | 12,482 |
| Bridge succeeds but no nflverse player-week | 9,367 | 9,580 | 8,536 |
| ↳ Nonzero scoped edge stat | 2,782 | 2,657 | 2,120 |
| ↳ All scoped edge stats zero | 6,585 | 6,923 | 6,416 |

The reported 25.5% = 7,188 / 28,206 (2025); it measures an ID bridge failure, NOT a player-week join failure. Distinct unbridged IDs: 602 / 641 / 713. A second loss occurs when `p6.py:load` iterates only the preexisting nflverse-based event keys.

Scoped edge fields checked: `rush_yac`, `rec_yar`, `rec_air_yd`, `pass_air_yd`, `rush_btkl`, `rec_drop`, `penalty`, `penalty_yd`, `idp_sack_yd`, `st_tkl_solo`, `off_snp`, `def_snp`, `st_snp`. Conversions/red-zone/RTL excluded because play-by-play owns them.

**Activity is not a bye diagnosis.** Among bridge failures, Sleeper `gp > 0`: 3,254 / 3,359 / 3,520; active without gp: 2,767 / 3,019 / 3,666; neither flag: 1 / 3 / 2. Among bridged rows missing an nflverse week, `gp > 0`: 2,783 / 2,679 / 2,130; no gp: 6,584 / 6,901 / 6,406. True bye versus inactive cannot be separated from these tables alone; no schedule or roster-status claim made. Thus the unjoined rows are demonstrably not almost all inactive/practice-squad players.

**No fix needed:** team aggregates must stay excluded (double-count risk); rows with all 13 scoped edge fields zero lose no Sleeper Package A points. Unknown-ID nonzero rows need identity/position classification before deciding scoring eligibility; many have offensive snaps, but cannot honestly be labelled offensive linemen or irrelevant from this crosswalk.

**Confirmed points lost:**
- 2025 Dorian Mausi, LB, Sleeper `12944`, MFL `17367`, GSIS `NA`: 12 played rows, 219 ST snaps, 4 ST solo tackles, 1 penalty. All are discarded by the unnecessary GSIS gate despite a direct Sleeper→MFL mapping. At existing Default32 rates the aggregate omitted edge contribution is +1.407 before engine rounding (219×0.003 + 4×0.25 − 0.25).
- 2023 known-MFL missing-GSIS nonzero rows: Qwuantrezz Knight (5), Shyheim Carter (2), Rodney Williams (8), Mekhi Garner (2), Joshua Pryor (3), Benny Sapp (2). Combined: 256 ST snaps, 54 defensive snaps, 42 offensive snaps, 5 ST solo tackles, 1 drop, 1 penalty / 8 penalty yards. Apply position eligibility, not the raw sum. 2024: Rodney Williams (6), 5 offensive snaps plus ST fields; his TE offensive snaps are scoreable.
- Even successfully bridged players lose snap-only weeks if attach uses an inner join: 2025 DJ Johnson, MFL `16277`, week 1 has 16 defensive and 13 ST snaps, but no nflverse player-week. At Default32 rates that is +0.167 before engine rounding. Bridged missing-week nonzero rows include 9,899 defensive / 7,105 offensive / 21,067 ST snaps in 2025; eligibility still applies per player position.

## Next move
Owner must resolve direct Sleeper→MFL identity and sparse-week policy before E2a format/E2b attach. Recommendation: preserve valid direct mappings without requiring GSIS; attach by union of validated scoring-eligible player-weeks rather than silently dropping absent nflverse weeks. First step: approve those two policies with Mausi and DJ Johnson as acceptance cases, then resume E2a. Do not build around the evidence gap.

## Blocked on
Owner decision above. `git pull --ff-only` refused because branch has no upstream; no alternate merge attempted. Tests/build not run: investigation-only stop, no Go changes. Existing E3 RTL-rate discrepancy remains unresolved (0.09368635 versus study 0.09386).

## Tried and rejected
Rejected the study HTML's unsupported “almost all practice-squad and camp players with no fantasy relevance” explanation: positive played flags and identified, scoreable rows contradict it. Did not infer bye status from absent weekly rows. No new evidence scheme, decoder, E2b row rewrite, registry, or speculative provider fields added.

Debt left: E2a format/digest-compatibility/typed-shape/round-trip tests and E2b implementation brief; identity coverage for unknown Sleeper IDs; sparse player-week handling; E5–E6 and E3 rate reconciliation.
