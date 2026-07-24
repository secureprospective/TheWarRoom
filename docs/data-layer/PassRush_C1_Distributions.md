# pfrpassrush C-1 Distributions — evidence for the SL-021 EMA `new_observation`

**Created:** 2026-07-24 · Branch `session/pfrpassrush-idp-calibration`
**Purpose:** The C-1 evidence sheet (FILM_Calibration §4) for wiring `pfrpassrush` as the SL-021
dynamic-α EMA `new_observation` (the PFF-grade replacement, handoff 50). **Sets no weight, locks
no decision** — it is the input the composite-recipe + EMA-weight decision + expert-panel gate rest on.
**Routing (Christopher, 2026-07-24):** feed the SL-021 EMA ONLY; the LOCKED IDPFilm Madden budget is
untouched. No new film-budget seat (option B rejected).

**Source:** live pull on CT105 (nflverse `advstats_season_def.csv.gz` 2023 + EA m24). Sampler =
`internal/ingestion/pfrpassrush/c1sample_test.go` (throwaway, opt-in `TWR_C1_SAMPLE=1`, delete at
wire-time). Population: 577 pass-rush records / 1404 Madden records.

---

## The headline finding — the pressure signal is LARGELY REDUNDANT with Madden exactly where 3G lives

Correlation of each candidate per-game composite with the LOCKED Madden IDP composite (K1), paired
on players present in both feeds. C-1 rule: **high |r| → double-count (little new info to smooth);
low |r| → a genuine independent grade.** (The K5 pre-ship guard used r>0.5 as the redundancy line.)

| Bucket | n (paired) | pressures/g r | components/g r | sacks/g r | Read |
|---|---|---|---|---|---|
| **DT** | 88 (76) | **+0.749** | +0.754 | +0.654 | HIGH redundancy |
| **DE** | 72 (56) | **+0.822** | +0.822 | +0.726 | VERY HIGH redundancy |
| **LB** | 187 (158) | +0.468 | +0.468 | +0.456 | moderate — most independent |

**The irony that must reach the panel:** SL-021 dynamic α is a **DT** mechanic (DE = fixed-0.15
control). DT and DE are exactly where the pressure composite is *most* redundant with the film anchor
already in production (r ≈ 0.75 / 0.82). The one bucket where the signal is genuinely additive (LB,
r ≈ 0.47) is the position SL-021's dynamic α **does not apply to**. So the mechanic 3G tests and the
place the signal adds information do not overlap.

All candidates are strongly positive at DT/DE (r > 0.65). `sacks/g` is the *least* redundant single
component (DT 0.65, DE 0.73) but still well above the 0.5 line. There is no low-redundancy pressure
recipe available at DT/DE.

---

## Distributions (raw per-game rates — NOT yet [0,1]-normalized; normalization is a downstream decision)

```
=== DT (n=88, paired=76) ===
  pressures/g   p50=0.467 p75=0.824 p90=1.375 p95=1.750 max=2.438  r=+0.749
  sacks/g       p50=0.118 p75=0.265 p90=0.458 p95=0.529 max=0.765  r=+0.654
  knockdowns/g  p50=0.143 p75=0.333 p90=0.688 p95=0.824 max=1.062  r=+0.678
  hurries/g     p50=0.125 p75=0.235 p90=0.400 p95=0.562 max=0.867  r=+0.554
  components/g  p50=0.455 p75=0.812 p90=1.344 p95=1.625 max=2.344  r=+0.754
=== DE (n=72, paired=56) ===
  pressures/g   p50=0.750 p75=1.312 p90=2.059 p95=2.188 max=2.941  r=+0.822
  sacks/g       p50=0.214 p75=0.438 p90=0.636 p95=0.833 max=1.029  r=+0.726
  knockdowns/g  p50=0.235 p75=0.417 p90=0.765 p95=0.882 max=1.000  r=+0.634
  hurries/g     p50=0.235 p75=0.412 p90=0.562 p95=0.706 max=1.176  r=+0.676
  components/g  p50=0.706 p75=1.312 p90=1.912 p95=2.147 max=2.912  r=+0.822
=== LB (n=187, paired=158) ===
  pressures/g   p50=0.438 p75=0.857 p90=1.529 p95=2.059 max=2.941  r=+0.468
  sacks/g       p50=0.133 p75=0.294 p90=0.533 p95=0.676 max=1.118  r=+0.456
  knockdowns/g  p50=0.133 p75=0.294 p90=0.588 p95=0.667 max=1.250  r=+0.470
  hurries/g     p50=0.118 p75=0.294 p90=0.529 p95=0.647 max=1.059  r=+0.319
  components/g  p50=0.412 p75=0.824 p90=1.471 p95=1.938 max=2.882  r=+0.468
```

---

## What this means for the wire-up (LEADS for the panel — not decisions)

1. **A large DT/DE pressure weight would mostly re-smooth Madden.** At r ≈ 0.75–0.82 the EMA's
   `new_observation` and the film anchor move together; a heavy α blend risks double-weighting the
   pass-rush dimension (identity-risk-adjacent) for little independent lift.
2. **`sacks/g` is the most independent single component at DT/DE** but still r > 0.65 — no clean
   low-redundancy recipe exists there.
3. **The independent signal (LB, r ≈ 0.47) is off the SL-021 mechanic.** If pressure is to earn a
   real seat on evidence, LB is where it does — but that is a different mechanic than 3G.
4. **Caveat — single season, m24 stale.** One year (2023) paired against the only live EA slug
   (m24). Madden is 2 seasons stale; correlations are directional, not a locked coefficient.

**Recommended framing for the gate:** given the redundancy, the defensible options are (a) wire 3G
as the **α-schedule property assertion only** (synthetic `previous`/`observation`, no production
weight — proves the SL-021 mechanic without asserting a redundant signal into scoring), and treat a
**live pressure weight as a small/zero-at-DT-DE knob** pending the panel; or (b) if pressure is
wanted as a live grade, site it where it is additive (LB) under its own calibration. **Do NOT ship a
heavy DT/DE pressure weight on this evidence.** Expert-panel gate required before any weight locks.
