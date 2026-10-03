# Sector 4 — starting values for the blend (Claude, 2026-10-03)

Astra hit its GPT usage limit after writing Sector 3 (E0–E10, `REPORT-3-evidence.md`). Its
`SECTOR-3.DONE` and Sector 4 were never written. I wrote this from Astra's Sectors 1–3.
Every cell points to an E-section in Astra's Sector 3 report. **These are starting priors for
calibration, not final values.**

## The three facts that most change the engine

1. **k does not come from the literature; it comes from our own free history.**
   - No published NFL position-specific k exists (E0).
   - Under the random-effects reading, `k = σ²/τ²`: per-snap noise variance over the
     between-player variance (E0).
   - Both are estimable from nflverse 2021–2025 by split-half and season-pair reliability, per
     position and per stat, using data Astra confirmed exists back to 2021 (Sector 2, G5).
   - So "hand-set now, learned later" becomes **"fitted from five free seasons now, refined from
     our own clock later"**. Only the scouting-prior weight has to wait for the clock, because
     no history of our own priors exists (Bee 4-F3).
2. **Opportunity is stable; efficiency and coverage outcomes are mostly noise.**
   - WR targets per route run, season to season: R² 0.41. Yards per target: R² 0.08 (E3).
   - Pass-rush pressure rate, year to year: r 0.72. Sack rate: r 0.51 (E2).
   - Player-level coverage EPA, year to year: −0.015 and 0.03, essentially zero (E4).
   - Tackle totals, first half of the season vs second: LB 0.64, DB 0.46 (E5).
   - So on-field-now must be built from **opportunity and usage shares**, and CB/S production
     must be heavily shrunk.
3. **Only draft capital has solid evidence as a scouting prior.**
   - Draft capital: player-level R² about 0.2–0.3 for career value (E9a).
   - Combine and athletic testing: weak and inconsistent. RB speed is the exception, at about 9%
     unique variance (E9b).
   - Breakout age: a WR hypothesis with no leakage-audited weight (E9d).
   - College production: some evidence at TE (E9c).
   - **Madden: no evidence at all** (E9e), yet today's code makes Madden the film "backbone".
   - So the prior is draft capital, conditioned on position, with small add-ons. Madden is
     demoted or dropped.

## Method fixed for every position

- **On-field-now production signal:** per-snap or per-opportunity rates, exposure-weighted
  across seasons. Injured weeks add no trials and are not zeros (E0).
- **Evidence `e`:** recency-weighted snaps. Use the effective-count form `(Σw)²/Σw²` when
  weights discount (E7). Rescaling the weights requires rescaling k (E7).
- **Recency baseline:** Marcel `1 / 0.8 / 0.6` as the comparison baseline only (E8a).
  - Recent-heavier per-position weights are a lead to test, not adopt (E8c: the Box Score blog,
    for example RB 68/20/10/3).
  - Fit the weights by temporal holdout on 2021–2025.
- **Scale:** percentile within position, so priors and production share one scale (E0.3).
- **Age:** **no flat 3%** (E10). Estimate per-position curves from 2021–2025 history, correcting
  for survivorship by regression with imputation (E10a, Schuckers/Lopez/Macdonald). Published
  peaks serve only as sanity checks.

## Per-position table

`k`: "fit" means estimated by the E0 method from nflverse 2021–2025. There are no literature
values. The "first-run placeholder" is the Marcel-style regression of 1 season-equivalent to the
mean, used only until the fit runs. **No evidence — placeholder.**

| Pos | On-field-now production (free, Astra G1–G3) | Evidence unit | k | Scouting prior, strength | Age sanity check |
|---|---|---|---|---|---|
| QB | dropbacks, EPA/dropback, ANY/A, rush share | dropbacks | fit (E1 is a cautionary lead only) | draft capital: moderate (E9a); combine: none (E9b); Madden: none (E9e) | ANY/A peak about 27 (E10b, blog) |
| RB | snap share, carry share, target share, red-zone share; efficiency shrunk hard | snaps | fit | draft capital: moderate; 10-yard split: small (E9b); college share: unvalidated | volume peak 26, survivorship (E10c) |
| WR | target share, targets per route (StatRankings routes need approval; otherwise target share per snap), air-yards share | snaps or routes | fit; opportunity k ≪ efficiency k (E3) | draft capital: moderate; breakout age: hypothesis (E9d); vertical: small (E9b) | 25–29 window, selected sample (E10d) |
| TE | snap share, target share, routes where available | snaps | fit | draft capital; college receptions and YPR: some (E9c) | about 27, selected sample (E10e) |
| K | FG% by distance band, shrunk to league by distance (E6); XP | attempts | shrinkage weight fitted (E6: 670 is the paper's, not ours) | none established | no reliable curve (E10f) |
| DT | defensive snap share; QB hits, sacks, TFL per snap (no pressures free: G3) | defensive snaps | fit; sack-type rates noisier than pressure (E2) | draft capital, pooled (E9a); RAS: unvalidated | linemen decline later, qualitative (E10a) |
| DE | as DT | defensive snaps | fit | as DT | as DT |
| LB | snap share; solo and assisted tackles per snap; TFL | defensive snaps | fit; tackles moderately stable (E5, LB 0.64–0.73) | draft capital, pooled | no IDP numbers (E10g) |
| CB | snap share; PD, INT, tackles per snap. Coverage outcomes are about noise (E4) | defensive snaps | fit; expect large k (E4) | draft capital, pooled | no IDP numbers (E10g) |
| S | snap share; tackles, PD, INT per snap; alignment matters (E5 context) | defensive snaps | fit; expect large k (E4) | draft capital, pooled | no IDP numbers (E10g) |

**Dynasty value** uses the same production blend, then the position age curve over a horizon,
with a heavier prior weight. Contract and cap stay separate (Bee 4-A).

## Where the evidence was too thin to rely on

- Every position's k. None is published; all are fitted from our data (E0, E1).
- How many seasons before scouting stops adding information: **could not establish** for any
  position (E9f).
- IDP age curves: none numeric (E10g). K aging: none (E10f).
- Madden, consensus boards and the app's RAS substitute: no validation (E9b, E9e).
- Recency weights: only the Marcel baseball baseline and an unreproducible blog (E8).
- Defender pressures, routes, run stops and coverage snaps: no free source (Sector 2, G3), so the
  IDP signals above are box-score proxies.
- Unfinished in Astra's run: the CB/slot numbers in E4 charts were not extracted; E2's full
  baseline years were not stated.
