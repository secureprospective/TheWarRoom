# Handoff

## Baton
Bee — 2026-10-27, `session/revamp-E-edge-scoring`. E2a resumed with the owner's direct-MFL and union-of-weeks rulings. Do not push.

## Where it stands
E2a format/typed-subset implementation is committed at 77b2c7ea on top of E1/E3/E4. Sleeper attachment is deliberately E2b, not claimed complete. The owner's direct-MFL and union-of-weeks policies resolve the previous investigation stop.

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

## Selected-evidence flow
Owner approved direct Sleeper→MFL without GSIS and the union of known players' source weeks; nflverse remains final for its own stats. Existing investigation above is retained, not repeated. The previous stop is resolved, not bypassed.

`RawFootballPlayer.Sleeper` pins the Sleeper→canonical MFL bridge in the existing player directory; GSIS may be empty for such a player. Optional `Sleeper []source.RawSleeperWeek` lives in both inline statistics and week parts: each entry holds season, week, exact JSON body. `RawSleeperStats.Validate/Convert` covers 13 edge quantities, including fractional sack yards and signed air/receiving YAC; conversions/red-zone/RTL remain PBP-owned. Revision fingerprints cover the new bodies/scopes and bridge only when Sleeper is present; legacy serialization/fingerprints stay byte-identical. ImportFootball/ImportFootballWeek → archive digest → LoadFootball/revisionBodies remains the sole storage path. E2b owns attachment/canonical-ID reads; no fabricated GSIS.

`TestSleeperUnknownIDReport` recomputed unknown-ID nonzero rows: 3,233 / 3,347 / 3,496, from 371 / 381 / 399 distinct unknown IDs respectively. Their rostered-player share is **unknown**: the supplied caches contain statistics/crosswalks, not fantasy rosters or an independent unknown-Sleeper-ID roster bridge. A nonzero snap or gp does not establish fantasy-roster membership; do not report zero. No new identity resolution will be built. The per-season counts are test-visible; team/zero rows remain excluded.

## Next move
Execute the E2b row in `docs/build-handoffs/revamp-E-slices.md`. It names files, canonical-read route, four acceptance tests and done criteria. Known IDs: Mausi Sleeper 12944 → MFL 17367 (no GSIS); DJ Johnson Sleeper 10938 → MFL 16277, GSIS 00-0038535. `TestSleeperNamedLossContributions` proves their retained edge quantities price at +1.407 / +0.167 before rounding; it does **not** prove attachment. E2b must prove both through import → reopen → canonical read → engine.

## Blocked on
No format blocker. Unknown-ID rostered share needs independent roster evidence; no new identity resolution authorized. `git pull --ff-only` still refuses because branch has no upstream; no alternate merge attempted. Existing E3 RTL-rate discrepancy remains unresolved (0.09368635 versus study 0.09386).

## Tried and rejected
Rejected the study HTML's unsupported “almost all practice-squad and camp players with no fantasy relevance” explanation. Did not infer bye status or fantasy-roster membership from absent weekly rows. No new pinning scheme, identity registry, fake GSIS, or speculative provider fields. Initial targeted tests exposed E1's incomplete shared line fixture; added the three missing edge columns and compared line behavior with equal edge capability, retaining every prior value/assertion.

## Verification
`GOTOOLCHAIN=go1.26.10 go test -race ./internal/store/history/ ./internal/fantasyscore/...` passed: history 180.723s, fantasyscore cached, source 5.996s. `go build ./...`: PASS (exit 0). Opt-in `TestSleeperUnknownIDReport` passed: history 4.974s, using p5/p6 caches via `TWR_STUDY_CACHE` / `TWR_SLEEPER_STUDY_CACHE`. Targeted golangci-lint: `0 issues.` Commit hooks (full lint, secrets, ifaceguard) passed for 77b2c7ea. One prior targeted run timed out at the tool's 240s limit under machine load; a subsequent complete run passed. No full test suite or make verify run; no push.

Legacy selected-evidence SHA is frozen, as are legacy inline bytes/SHA and week-part bytes; split SHA retains the old ordered-part formula. Save/load table covers inline and split evidence across database reopen. Rejected invalid imports leave no archived body. Source trace: new scope/body validation runs before statistics parsing, both inline and split flow through sleeperRevision, and SourceID/ReadNativeEvidence still use the old GSIS read path until E2b. Affected handwritten production files: 566 → 769 physical lines (+203); tests/docs measured separately, not claimed as a reduction.

Debt left: E2b attach/canonical read and end-to-end loss tests; unknown Sleeper identity/roster coverage; E5–E6 and E3 rate reconciliation.
