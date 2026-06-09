# Legacy NFL Position Group Rubric — Cornerback (CB)
**Version:** 1.0 — June 2026
**Status:** Locked. Third position to apply SL-019 RAS-modulator interactions (after TE and DE), with slightly reduced modulator strengths (0.30 / 0.30 / 0.25) reflecting CB longevity variability within the position. First rubric to elevate NGS coverage metrics as a dedicated analytical anchor. Gemini's CB open questions reconciled into SL-OQ-034 and CAL-027.
**Companion:** Engine_Specification.md Layer 4 is authoritative on mechanics. This rubric specifies position-specific values, the SL-019 mechanic at CB with reduced modulator strengths, and the NGS sub-signal architecture.

---

## 1. Architectural Baseline

- **Layer 4 RAS Tier:** **High** (per SL-004). Athletic profile drives baseline multiplier at year 0; collapses to residual by year 2+ per SL-018.
- **Layer 3 Peak Limit:** 28 years (steeper than offensive skill positions — CB reactive speed decays one year earlier than WR/TE peaks at 29).
- **SL-019 application:** **YES, with reduced modulator strengths.** CB meets the SL-OQ-027 gating rule criteria (High-tier RAS + position-specific athletic-profile-to-longevity predictive relationship — elite-RAS CBs with 4.30 speed maintain recovery into their 30s; mid-RAS CBs lose half-step recovery earlier). Modulator strengths reduced from TE/DE 0.35 / 0.35 / 0.30 to **0.30 / 0.30 / 0.25** reflecting CB longevity variability within the position (boundary vs. slot, press vs. zone, scheme-dependency on safety help).
- **SL-019 gating rule reference:** This is the third position to apply SL-019. Gating rule documented in DE rubric Section 7; CB satisfies both criteria.
- **Layer 2 Base Points drivers** (per Official Rulebook lines 88–117, MFL-sourced per DECISION-009):
  - **True Position split (LB/CB/S):** Tackle 1.5, Assist 1.0
  - **Pass-rush production (rare for CB but possible on blitzes):** Sack 4.5, QB Hit 1, Tackle for Loss 2.5
  - **Coverage (primary CB scoring):** Pass Defensed 2.5, Interception Caught 5, Interception Return Yards 0.025/yd, Interception Return TD 6
  - **Turnover production:** Forced Fumble 4, Opponent Fumble Recovery 3, Opponent Fumble Recovery Yards 0.025/yd
  - **Special teams adjacent:** Blocked Kick 7
  - **Game-state:** Safety 10
- **Data Parity Rule:** Missing/Unknown sub-signal data collapses component deviation to 0.0 via confidence weighting, returning neutral 1.00 fallback.

---

## 2. Film Component Configuration

**Cap (asymptote):** ±5% (standard).

### Sub-signal weights

| Source | Weight | Classification |
|---|---|---|
| PFF Cornerback Coverage Grade (analytical anchor) | 0.35 | Analytical — self-regulates |
| NFL Next Gen Stats Coverage Metrics (analytical anchor) | 0.30 | Analytical — self-regulates |
| The IDP Show (subjective anchor) | 0.10 | Subjective — Madden-regulated |
| The IDP Guru (analytical modifier) | 0.10 | Analytical — self-regulates |
| The Draft Network (pre-draft trait eval) | 0.08 | Subjective — Madden-regulated |
| Dynasty Nerds | 0.07 | Subjective — Madden-regulated |

Sums to 1.00.

**Key architectural choice — NGS as dedicated anchor:** CB is the first position where NFL Next Gen Stats coverage metrics earn their own anchor weight (0.30). Tracking-data measurements at CB — target separation, completion % allowed, ADOT against, average separation at catch — are uniquely valuable at the position and analytically distinct from PFF grades. NGS metrics CALIBRATE pre-snap leverage and post-snap recovery directly; PFF grades MEASURE overall coverage value. Both anchors needed.

**TDN weight intentionally lower (0.08) at CB** vs. offensive positions (typically 0.10-0.15) per handoff guidance — TDN coverage of CB prospects is weaker than at offensive positions where Waldman-style depth analysis is available. RSP intentionally omitted (offense-focused).

### Madden regulation parameters

- **Threshold:** 0.15 (normalized scale)
- **Blend scaling:** Linear gradient over 0.10 delta beyond threshold

### Madden attribute mapping

| Subjective Expert Claim | Madden Sub-Attribute / Composite | Formula |
|---|---|---|
| "Elite recovery speed / vertical match" | Speed (SPD) + Acceleration (ACC) | Average(SPD, ACC) |
| "Physical press / disruptive jam" | Press (PRS) + Strength (STR) | (0.8 × PRS) + (0.2 × STR) |
| "Sticky man coverage / fluid hips" | Man Coverage (MCV) + Agility (AGI) | Average(MCV, AGI) |
| "Zone instincts / spatial awareness" | Zone Coverage (ZCV) + Play Recognition (PRC) | Average(ZCV, PRC) |
| "Elite ball skills / catch point dominance" | Catching (CTH) + Jumping (JMP) | Average(CTH, JMP) |

Five rows. Press archetype uses asymmetric 0.8 × PRS + 0.2 × STR weighting — PRS attribute is more directly predictive of press effectiveness than raw STR (a strong-but-slow CB can't execute press; a smaller-but-technical press corner like Jaire Alexander can).

### Signal mechanics

- `film_position_weight`: 1.00
- `film_inflection`: 0.50
- `film_steepness`: 12.0

### EMA blend rates (dynamic sub-signals)

| Sub-signal | α | Classification |
|---|---|---|
| PFF | 0.18 | Dynamic — weekly grades with slow-moderate blend |
| NGS Coverage Metrics | 0.20 | Dynamic — weekly tracking data with moderate blend |
| IDP Show | 0.30 | Dynamic — weekly subjective content, moderate blend |
| IDP Guru | 0.20 | Dynamic — weekly analytical content |
| TDN | N/A | **STATIC** — locked at rookie evaluation; no re-publication for veterans |
| Dynasty Nerds | 0.50 | Dynamic — annual-publication-rate content |
| Madden | 0.20 | Dynamic — multiple mid-season updates with moderate blend |

### Season transition behavior

CONTINUATION across all dynamic sub-signals.

### Sub-signal normalization (0.0–1.0 mapping)

- **PFF:** PFF CB Coverage Grade / 100 (grades are already 0–100)
- **NGS Coverage Metrics:** Composite z-score of (target separation, inverse completion % allowed, inverse ADOT allowed) mapped to position percentile, then 0–1 scaled. CAL-026 flags specific metric bundle definition for empirical calibration.
- **IDP Show:** Tier ranking inverted to percentile within The IDP Show's CB pool
- **IDP Guru:** Weekly CB ranking inverted to percentile within IDP Guru's CB pool
- **TDN:** TDN composite grade mapped to percentile within draft class CB
- **Dynasty Nerds:** Dynasty CB ranking inverted to percentile within Dynasty Nerds' CB pool

---

## 3. RAS Component Configuration

**Cap (asymptote):** ±8% (High-tier per SL-004).

### Parameters

- `RAS_position_weight`: **1.00 at year 0 baseline**, modulated by SL-018:

| NFL career stage | RAS_position_weight |
|---|---|
| Rookie / pre-NFL data | 1.00 |
| After 1 NFL season | 0.50 |
| Year 2+ | 0.10 |

- `RAS_inflection`: **0.50** (reset from Gemini's 0.55 for cross-position consistency — "universe of athletes" baseline preserved across all rubrics)
- `RAS_steepness`: 11.0 (Gemini's value retained — CB athletic profiles are bimodal, sharper transition defensible)

### Normalization

`RAS_normalized = raw_RAS / 10.0`

Missing RAS → Layer 1 position-group mean imputation. Confidence flag set to Unknown.

### Late-career interaction (SL-018 + SL-019 reduced buffer)

Once player age > 28 (CB peak limit), Layer 3 age_pull is RAS-buffered. **SL-019 amplifies the buffer for CB, but at reduced strength (0.25× vs. TE/DE 0.30×):**

```
buffer_pct        = 0.25 × RAS_normalized   ← CB-specific (reduced from TE/DE)
buffered_age_pull = 1.0 + (raw_age_pull − 1.0) × (1 − buffer_pct)
```

A CB with RAS = 9.5 gets a ~24% buffer against age decay. A CB with RAS = 5.0 gets a 12% buffer. Elite-athletic CBs (Peterson, Sherman late-career, Asomugha prime years) play meaningfully longer than mid-RAS peers, but the differential is less aggressive than at TE/DE because scheme-fit variance at CB introduces noise into the RAS-longevity signal.

**SL-018 scope:** SL-019 modulator interactions are INDEPENDENT of SL-018 decay (same architecture as TE and DE).

---

## 4. Breakout Component Configuration

**Cap (asymptote):** ±5%.

### Sub-signal weights

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.20 |
| School Tier | 0.25 |
| College Production Share (PD + INT market share) | 0.40 |
| Age Trajectory | 0.15 |

Sums to 1.00.

**School Tier elevated to 0.25** (vs. typical 0.20) — NFL-caliber CBs come from P4 programs more reliably than at other positions. **College Production Share at 0.40** with PD + INT market share definition — CB college production is the cleanest college-to-NFL predictor (more reliable than scouting at this position). **Breakout Age reduced to 0.20** reflecting that CB breakout timing is less predictive than at offensive positions (many CBs ascend through coverage repetition rather than year-zero dominance).

### Parameters

- `breakout_position_weight`: 1.00
- `breakout_inflection`: 0.50
- `breakout_steepness`: 10.0 (Gemini's value — slightly less aggressive than 11.0 used at DE/LB, defensible given CB college production data noise)

### Normalization functions

**Breakout Age** — base curve:

| Breakout Age | Normalized |
|---|---|
| ≤19.5 | 1.00 |
| 20.5 | 0.75 |
| 21.5 | 0.45 |
| ≥22.5 | 0.15 |

Linear interpolation between defined points.

**SL-019 modulation applies (strength 0.30):**
```
breakout_age_modulated = base + (1.0 − base) × 0.30 × RAS_normalized
```

Worked examples:
- Base 0.15, RAS 9.99 → 0.15 + 0.85 × 0.30 × 0.999 = **0.405**
- Base 0.15, RAS 5.00 → 0.15 + 0.85 × 0.30 × 0.50 = **0.278**
- Base 1.00, any RAS → **1.00** (already maxed)

**School Tier** (template defaults):

| Tier | Normalized |
|---|---|
| Power Four | 1.00 |
| Group of Five | 0.70 |
| FCS | 0.40 |
| Non-FCS | 0.10 |

**College Production Share** (final-year PD + INT market share):

| Market Share | Normalized |
|---|---|
| ≥24% | 1.00 |
| 16% | 0.55 |
| ≤8% | 0.15 |

Linear interpolation between defined points.

**Age Trajectory** (current age relative to CB peak limit of 28):

| Age | Normalized |
|---|---|
| ≤24 | 1.00 |
| 25 | 0.85 |
| 26 | 0.70 |
| 27 | 0.55 |
| 28 (peak) | 0.50 |
| 29 | 0.35 |
| 30 | 0.20 |
| 31 | 0.10 |
| ≥32 | 0.00 |

Steeper than WR/TE curves (peak 29) — pulls one year earlier across the board, reflecting CB reactive-speed decay.

**SL-019 modulation applies (strength 0.30):**
```
age_trajectory_modulated = base + (1.0 − base) × 0.30 × RAS_normalized
```

Worked examples:
- Base 0.50 (age 28), RAS 9.50 → 0.50 + 0.50 × 0.30 × 0.95 = **0.6425**
- Base 0.20 (age 30), RAS 7.00 → 0.20 + 0.80 × 0.30 × 0.70 = **0.368**
- Base 0.00 (age 32+), RAS 9.30 → 0.00 + 1.00 × 0.30 × 0.93 = **0.279**

### Three-zone classification

- **Elite zone:** composite_input ≥ 0.80
- **Average zone:** 0.40 < composite_input < 0.80
- **Late zone:** composite_input ≤ 0.40

---

## 5. Verification Cases

**S-curve formula:**
```
output = 1 + cap × (2 × σ(steepness × (input − inflection)) − 1)
where σ(x) = 1 / (1 + e^(−x))
```

**Component combination:**
```
component_effective = 1.0 + (component_raw − 1.0) × confidence × position_weight
Layer_4_Output      = film_effective × RAS_effective × breakout_effective
```

---

### Case 1 — Push: Patrick Surtain II

**Profile:**
- Age 26 (born April 2000) — 2 years before peak limit
- College: Alabama (Power Four — SEC)
- R1 #9 overall 2021
- Breakout age: 18 (true freshman impact, multiple starts as freshman)
- Junior-year college production share: ~22% PD + INT market share
- RAS: ~9.5 *(estimated, pending ras.football verification — 4.42 40 at 203lbs with elite agility and explosion)*
- PFF: elite recent grades, ~91 (multiple All-Pro honors)
- Year 5 NFL veteran → SL-018 Year 2+ tier (RAS_position_weight = 0.10 at High)

**Film component:**

Sub-signal normalizations:
- PFF CB Coverage Grade ~91 → 0.91
- NGS coverage metrics (top-tier completion % allowed, target separation) → 0.93
- IDP Show top-tier CB consensus → 0.92
- IDP Guru top-3 CB → 0.92
- TDN (static, 2021 R1 #9 elite grade) → 0.95
- Dynasty Nerds top dynasty rank → 0.95

Composite:
```
(0.35 × 0.91) + (0.30 × 0.93) + (0.10 × 0.92) + (0.10 × 0.92) + (0.08 × 0.95) + (0.07 × 0.95)
= 0.3185 + 0.279 + 0.092 + 0.092 + 0.076 + 0.0665
= 0.924
```

S-curve(0.924, 0.50, 12.0, 0.05):
- arg = 12.0 × 0.424 = 5.088
- σ(5.088) ≈ 0.9939
- output factor = 2 × 0.9939 − 1 = 0.9877
- film_raw = 1 + 0.05 × 0.9877 = **1.049**
- film_effective = **1.049**

**RAS component (SL-018 Year 2+):**

- RAS_normalized = 9.5 / 10 = 0.95
- S-curve(0.95, 0.50, 11.0, 0.08):
  - arg = 11.0 × 0.45 = 4.95
  - σ(4.95) ≈ 0.9930
  - output factor = 0.9859
  - RAS_raw = 1 + 0.08 × 0.9859 = **1.079**
- Year 2+ → RAS_position_weight = 0.10
- RAS_effective = 1.0 + (1.079 − 1.0) × 1.0 × 0.10 = **1.008**

**Breakout component (SL-019 modulators at 0.30 / 0.30):**

- Breakout Age 18 → base 1.00 (already maxed)
- School Tier P4 → 1.00
- College Production Share 22% → linear interp between 16% (0.55) and 24% (1.00): 0.55 + 6 × 0.05625 = **0.888**
- Age Trajectory 26 → base 0.70, modulated = 0.70 + 0.30 × 0.30 × 0.95 = **0.786**

Composite:
```
(0.20 × 1.00) + (0.25 × 1.00) + (0.40 × 0.888) + (0.15 × 0.786)
= 0.200 + 0.250 + 0.355 + 0.118
= 0.923
```

Composite is in the **Elite zone** (≥ 0.80).

S-curve(0.923, 0.50, 10.0, 0.05):
- arg = 10.0 × 0.423 = 4.23
- σ(4.23) ≈ 0.9857
- output factor = 0.9714
- breakout_raw = 1 + 0.05 × 0.9714 = **1.049**
- breakout_effective = **1.049**

**Layer 4 combined:**

```
Layer_4_Output = 1.049 × 1.008 × 1.049 = 1.109
```

**Multiplier: ~1.11** — clear push.

**Full Layer 3 × Layer 4 chain for Surtain:**

Layer 3 age_pull at age 26 (pre-peak) = no decay yet, age_pull = 1.000.

```
Layer 3 × Layer 4 (Surtain) = 1.000 × 1.109 = 1.109
```

---

### Case 2 — Pull-attempt: Patrick Peterson

**Profile:**
- Age 36 (born July 1990) — 8 years past peak limit
- College: LSU (Power Four — SEC)
- R1 #5 overall 2011
- Breakout age: 19 (true freshman impact, dominant sophomore season as LSU's #1 CB)
- Senior-year college production share: ~28% PD + INT market share (LSU's primary CB through three seasons)
- RAS: ~9.3 *(estimated, pending ras.football verification — 4.34 40 at 219lbs with elite explosion)*
- PFF: declining, ~65 recent grade (was 85+ in prime 2011–2018)
- Year 15 NFL veteran → SL-018 Year 2+ tier (RAS_position_weight = 0.10 at High)

**Film component:**

Sub-signal normalizations:
- PFF CB Coverage Grade ~65 → 0.65
- NGS coverage metrics declining (high completion % allowed in recent years) → 0.40
- IDP Show declining vet tier → 0.40
- IDP Guru mid-tier vet → 0.45
- TDN (static, 2011 R1 #5 elite grade) → 0.95
- Dynasty Nerds low dynasty rank → 0.20

Composite:
```
(0.35 × 0.65) + (0.30 × 0.40) + (0.10 × 0.40) + (0.10 × 0.45) + (0.08 × 0.95) + (0.07 × 0.20)
= 0.2275 + 0.120 + 0.040 + 0.045 + 0.076 + 0.014
= 0.5225
```

S-curve(0.5225, 0.50, 12.0, 0.05):
- arg = 12.0 × 0.0225 = 0.270
- σ(0.270) ≈ 0.5671
- output factor = 2 × 0.5671 − 1 = 0.1342
- film_raw = 1 + 0.05 × 0.1342 = **1.007**
- film_effective = **1.007**

Note: Film raw barely pushes despite genuine decline (PFF dropped 20+ points, NGS dropped 50+ percentile, Dynasty Nerds rank collapsed) — the static TDN at 0.95 (locked at 2011 elite grade) holds the composite up. Early Lockett-pattern signal at film level.

**RAS component (SL-018 Year 2+):**

- RAS_normalized = 9.3 / 10 = 0.93
- S-curve(0.93, 0.50, 11.0, 0.08):
  - arg = 11.0 × 0.43 = 4.73
  - σ(4.73) ≈ 0.9913
  - output factor = 0.9826
  - RAS_raw = 1 + 0.08 × 0.9826 = **1.079**
- Year 2+ → RAS_position_weight = 0.10
- RAS_effective = 1.0 + (1.079 − 1.0) × 1.0 × 0.10 = **1.008**

**Breakout component (SL-019 modulators applied):**

- Breakout Age 19 → base 1.00 (already maxed)
- School Tier P4 → 1.00
- College Production Share 28% → 1.00 (above 24% threshold)
- Age Trajectory 36 → base 0.00 (≥32), modulated = 0.00 + 1.00 × 0.30 × 0.93 = **0.279**

Composite:
```
(0.20 × 1.00) + (0.25 × 1.00) + (0.40 × 1.00) + (0.15 × 0.279)
= 0.200 + 0.250 + 0.400 + 0.0419
= 0.892
```

Composite is in the **Elite zone** (≥ 0.80) — despite age 36. Three of four sub-signals MAXED, and SL-019 lifts the age trajectory base from 0.00 to 0.279 because his elite RAS (0.93 normalized) earns him longevity credit.

S-curve(0.892, 0.50, 10.0, 0.05):
- arg = 10.0 × 0.392 = 3.92
- σ(3.92) ≈ 0.9805
- output factor = 0.9611
- breakout_raw = 1 + 0.05 × 0.9611 = **1.048**
- breakout_effective = **1.048**

**Layer 4 combined:**

```
Layer_4_Output = 1.007 × 1.008 × 1.048 = 1.064
```

**Multiplier: ~1.06** — **PUSH despite age 36.**

**Structural finding (Lockett pattern at CB, with SL-019 amplification):** Peterson's static breakout signals (breakout age 19, P4 LSU, 28% college market share) are all maxed; only age trajectory updates with current age, and at 0.15 weight + SL-019 modulator lifting base from 0.00 to 0.279, it cannot move the composite below the Elite zone threshold. Layer 4 lands at 1.06 — Peterson, at 36, still PUSHES Layer 4.

This is the seventh confirmed Lockett-pattern instance (WR, RB-with-Herbert-exception, TE, QB, DE, LB, CB). SL-019 amplifies the pattern at CB by lifting the age trajectory sub-signal — an elite-RAS aging CB gets MORE Layer 4 protection than a mid-RAS aging CB, which is the architectural intent.

**Full Layer 3 × Layer 4 chain for Peterson:**

Layer 3 age_pull at age 36 = 0.97^8 ≈ 0.784. SL-019 reduced buffer (0.25 × RAS_normalized) = 0.25 × 0.93 = 0.2325 (23.25%). Buffered age_pull = 1.0 + (0.784 − 1.0) × (1 − 0.2325) = 1.0 + (−0.216)(0.7675) = **0.834**.

```
Layer 3 × Layer 4 (Peterson) = 0.834 × 1.064 = 0.887
Layer 3 × Layer 4 (Surtain)  = 1.000 × 1.109 = 1.109
```

A 22% spread between an in-prime elite CB and an eight-years-past-peak veteran. SL-019 buffer pulled Peterson's age_pull from 0.784 (without buffer) up to 0.834 — about 5 percentage points of protection from age decay, appropriate for his elite RAS profile. The buffer is meaningful but not dominant; Layer 3 still does most of the aging work.

---

## 6. Open Questions Surfaced

Prior sessions surfaced SL-OQ-013 through SL-OQ-032 and CAL-015 through CAL-025. CB adds:

- **SL-OQ-033:** Subjective scouting anchor strength at CB. Pre-draft scouting depth is weaker at CB than at offensive positions — RSP (offense-focused) is intentionally omitted from the source list, and TDN coverage of CB prospects is thinner than for WR/RB/TE. The IDP Show + IDP Guru combined fill the subjective-anchor role at 0.20 weight, but this may be insufficient. Question: should a CB-specific scouting source be added to the approved source library (e.g., All-22 film analysts who specialize in CB technique)? Or is the current PFF + NGS analytical dominance (0.65 combined weight) sufficient to make subjective input lower-priority at CB specifically?

- **SL-OQ-034 (from Gemini, renumbered from local SL-OQ-019):** Targets-starved elite CB NGS interpretation — the "Revis Island" problem. An elite shadow corner may be systematically avoided by opposing QBs to the point that his target volume drops below the threshold where NGS metrics (completion % allowed, target separation, ADOT against) remain statistically reliable. The current NGS sub-signal definition (CAL-026) does not account for this — a corner with 25 targets all season looks identical in the metric to a corner with 95 targets if their per-target rates match, but the small sample materially undermines confidence in the signal. Refinement candidates: target-volume confidence weighting on the NGS sub-signal, or a minimum-targets gate below which NGS confidence collapses to a floor and PFF/subjective signals carry more weight. Links to CAL-026 — both resolve together once the NGS bundle is empirically calibrated.

**Calibration Backlog additions from CB build:**

- **CAL-026:** NGS Coverage Metrics bundle definition. The 0.30-weight NGS sub-signal is currently defined as "composite z-score of (target separation, inverse completion % allowed, inverse ADOT allowed) mapped to position percentile." This is a placeholder definition; empirical calibration needed to determine which NGS metrics best predict CB future production. Additional candidates: average separation at catch, slot vs. boundary alignment %, yards after catch allowed, blitz rate when on the field. Methodology question — does the bundle weight metrics equally, or rank-weight by predictive power?

- **CAL-027 (from Gemini, renumbered and reframed from local CAL-021):** RAS inflection appropriateness across the CB archetype spectrum. Gemini's original framing referenced the 0.55 inflection point, but Christopher's call reset that parameter to 0.50 for cross-position consistency; the underlying empirical concern is preserved against current 0.50 implementation. Question: does standard inflection 0.50 appropriately calibrate the RAS-to-outcome curve across the full CB archetype spectrum (boundary press, boundary zone, slot, hybrid), or does CB warrant a position-specific inflection? Specific tensions Gemini identified: protecting against low-RAS boundary liabilities (who lack the recovery speed to survive at boundary) without filtering out long-armed instinctual slot anchors (lower RAS, technique/length/IQ compensates). If empirical data shows the standard inflection systematically misranks one archetype, position-specific calibration may be warranted. Defer to live-data calibration; pairs with the Section 7 note on RAS inflection consistency.

---

## 7. Position-Specific Notes

- **SL-019 reduced strength at CB (0.30 / 0.30 / 0.25):** First position to apply SL-019 with strengths reduced below the TE/DE 0.35 / 0.35 / 0.30 baseline. Justification: CB longevity is more variable within the position than at TE/DE — a boundary press corner ages differently from a slot zone corner, scheme-dependency on safety help introduces noise into the RAS-longevity signal. The reduction is calibration-pending; if CAL-022 (DE SL-019 strengths) and a future CAL-019-equivalent for CB both converge on similar strengths, the reduced CB values may merge back to a single SL-019 default. Defer to live data.

- **NGS as dedicated anchor:** CB is the first position to elevate Next Gen Stats coverage metrics to dedicated 0.30-weight anchor status. The pattern may extend to S (where deep zone coverage tracking is similarly valuable) but does NOT apply to LB, DE, or interior defensive positions where NGS coverage metrics are not meaningful. Decision documented here; S rubric will need to evaluate independently.

- **RAS inflection reset to 0.50:** Gemini's draft shifted the inflection to 0.55 to reflect that NFL-drafted CBs skew above the universal athletic mean. The intuition is empirically defensible but breaks convention with all other rubrics. The "universe of athletes" baseline (inflection 0.50) is the architectural standard, preserved here for cross-position consistency. Position-specific inflection adjustments deferred to live-data calibration if needed.

- **Peak limit 28 — one year earlier than WR/TE:** CB peak is set at 28 reflecting reactive-speed decay onset earlier than at offensive skill positions. Age trajectory curve scales accordingly — pulls begin at age 25 (vs. WR/TE pulling at age 26).

- **EDGE classification reminder:** Per OQ-004 resolution, CB rubric covers cornerbacks only. Hybrid safety-corner roles (where a player is consensus-classified as safety with corner snaps) route through the S rubric.

- All RAS values in verification cases are estimates pending ras.football verification.

---

## 8. Cross-Pollination Source

This rubric synthesized from:
- Universal Rubric Template v1.1 (structural skeleton)
- Engine Specification v2.1 (Layer 4 mechanics)
- Gemini's CB rubric draft (sub-signal weight allocations including the NGS anchor innovation, archetype framework with asymmetric press weighting, base curves for breakout age and college production share, S-curve parameters, breakout sub-signal weight allocation with elevated School Tier)
- Gemini's CB open questions (reconciled — SL-OQ-034 targets-starved-elite-CB NGS interpretation accepted as a novel "Revis Island" concern linked to CAL-026 NGS bundle definition; CAL-027 reframed from Gemini's CAL-021 because Christopher's call reset the RAS inflection from 0.55 to 0.50, removing the specific parameter Gemini was tracking — empirical concern preserved against current 0.50 implementation, deferred to live-data calibration)
- Christopher's calls (SL-019 reduced modulator strengths 0.30 / 0.30 / 0.25; RAS inflection reset to 0.50 for cross-position consistency; Patrick Surtain II push, Patrick Peterson pull-attempt)
- SL-018 applied (Gemini's draft predated this mechanic)
- SL-019 applied with reduced strengths (Gemini's draft predated this mechanic — CB is third instance after TE and DE)
- Layer 2 drivers expanded to include QB Hits (1), Forced Fumble (4), Opponent Fumble Recovery (3) + return yards, INT Return Yards (0.025/yd), Blocked Kick (7), Safety (10) — Gemini's list was incomplete
- EMA blend rates fixed: IDP Show split from IDP Guru with α=0.30 / 0.20, TDN → STATIC at reduced weight 0.08, Dynasty Nerds split from TDN at α=0.50
- RAS inflection reset from 0.55 to 0.50
- Age trajectory curve tightened from Gemini's "≥28=0.10" plateau to gradual decline through 0.00 at age 32
- `[cite: N]` markers stripped throughout
- Verification cases replaced from generic placeholders with named real-player cases and full S-curve math

---

*Built by: Christopher Campbell + Claude (Anthropic)*

| Version | Date | Changes |
|---|---|---|
| 0.9 | June 2026 | Draft from cross-pollinated Gemini baseline + audit refinements. Third SL-019 instance after TE and DE, with reduced modulator strengths (0.30 / 0.30 / 0.25). First rubric to elevate NGS coverage metrics as dedicated 0.30-weight analytical anchor. RAS inflection reset to 0.50 from Gemini's 0.55 for cross-position consistency. SL-018 + SL-019 applied. Layer 2 drivers expanded with missing rulebook lines. Both verification cases worked end-to-end with full math (Surtain ~1.11 push at age 26 pre-peak; Peterson ~1.06 Layer 4 push despite age 36 — Lockett pattern amplified by SL-019 age-trajectory modulator, ~0.89 on full Layer 3 × Layer 4 chain with SL-019 buffer pulling age_pull from 0.784 to 0.834). Pending Gemini's CB open questions for reconciliation pass before v1.0 lock. |
| 1.0 | June 2026 | Locked. Gemini's CB open questions reconciled. SL-OQ-034 (targets-starved-elite-CB NGS interpretation — the "Revis Island" problem) renumbered from Gemini's local SL-OQ-019, accepted as novel concern linked to CAL-026 NGS bundle definition; both resolve together once NGS bundle is empirically calibrated with minimum-targets confidence-weighting. CAL-027 reframed from Gemini's local CAL-021 because Christopher's call reset RAS inflection from 0.55 to 0.50; empirical concern about RAS calibration across CB archetype spectrum preserved against current 0.50 implementation. |
