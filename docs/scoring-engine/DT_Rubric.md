# Legacy NFL Position Group Rubric — Defensive Tackle (DT)
**Version:** 1.0 — June 2026
**Status:** Locked. **First position with hybrid tier architecture** — Medium-tier classification with elevated High-tier RAS treatment and two DT-unique mechanics (dynamic Year 1 / Year 2+ PFF EMA, Late-Career Cushion Guard). New locked decision **SL-021** packages the full DT tier resolution per Christopher's call on the SL-004 vs. SL-005 contradiction. Gemini's DT open questions reconciled into SL-OQ-039 and CAL-031.
**Companion:** Engine_Specification.md Layer 4 is authoritative on baseline mechanics. This rubric specifies position-specific values, the hybrid tier configuration, and the two DT-unique mechanics.

---

## 1. Architectural Baseline

- **Layer 4 RAS Tier (resolved):** **Medium classification with elevated High-tier RAS treatment.** Resolves the SL-004 vs. SL-005 contradiction noted in the Deliverable 2 handoff (SL-004 places DT at Low tier; SL-005 says "RAS is the primary Layer 4 signal" for DT). Christopher's resolution: amend SL-004 to make DT a hybrid case — Medium-tier scouting weight treatment (SL-005 film compression applied), High-tier RAS schedule and cap (RAS is primary at year 0 baseline, recedes per High-tier schedule as NFL data accumulates), and two DT-unique mechanics described below. **Locked as SL-021.**
- **Layer 3 Peak Limit:** 30 years.
- **DT-unique mechanic 1 — Dynamic Year 1 / Year 2+ PFF EMA:** Standard fixed-α model replaced for PFF specifically. PFF α = 0.50 in Year 1 of NFL data accumulation (aggressive blend, forces new NFL grades into the model quickly to displace rookie-era RAS dominance), then α = 0.10 in Year 2+ (slow blend, stable vet signal). Carried by SL-021 as part of the DT resolution package. The mechanic addresses the cold-start problem at DT where rookie-era valuation is RAS-anchored and real NFL grades need to overwrite that anchor quickly when they arrive.
- **DT-unique mechanic 2 — Late-Career Cushion Guard:** If Raw RAS ≥ 8.00, late-career penalty velocity reduced by 10%. Applied at two points: (a) Layer 3 age_pull beyond peak, (b) breakout component Age Trajectory sub-signal beyond peak. **Decline-rate interpretation:** cushioned value = peak − (peak − base) × 0.90 (NOT deviation-from-neutral interpretation). Carried by SL-021. The mechanic is DT's equivalent of SL-019's late-career buffer but with binary threshold and conservative 10% flat reduction; reflects that DT athletic profile is less cleanly predictive of longevity than coverage positions.
- **SL-019 application:** **NOT applied.** Cushion Guard is DT's late-career mechanism; running SL-019 in parallel would double-protect elite-RAS DT veterans.
- **Layer 2 Base Points drivers** (per Official Rulebook lines 88–117, MFL-sourced per DECISION-009):
  - **True Position split (DT/DE only):** Tackle 2.5, Assist 1.5
  - **Pass-rush production:** Sack 4.5, QB Hit 1, Tackle for Loss 2.5 (direct stat — no proxy)
  - **Coverage (rare for DT):** Pass Defensed 2.5, Interception Caught 5, Interception Return Yards 0.025/yd, Interception Return TD 6
  - **Turnover production:** Forced Fumble 4, Opponent Fumble Recovery 3, Opponent Fumble Recovery Yards 0.025/yd
  - **Special teams adjacent:** Blocked Kick 7
  - **Game-state:** Safety 10
- **Data Parity Rule:** Missing/Unknown sub-signal data collapses component deviation to 0.0 via confidence weighting, returning neutral 1.00 fallback.

---

## 2. Film Component Configuration

**Cap (asymptote):** **±3%** (SL-005 compression — vs. standard ±5%). Thinnest film coverage of any position with engine value. Compression acknowledged in Section 7.

### Sub-signal weights

| Source | Weight | Classification |
|---|---|---|
| PFF Interior Defensive Line Grade (analytical anchor) | 0.50 | Analytical — self-regulates, DYNAMIC α per SL-021 |
| The IDP Show (subjective anchor) | 0.20 | Subjective — Madden-regulated |
| The IDP Guru (analytical modifier) | 0.15 | Analytical — self-regulates |
| The Draft Network (pre-draft trait eval) | 0.08 | Subjective — Madden-regulated |
| Dynasty Nerds | 0.07 | Subjective — Madden-regulated |

Sums to 1.00.

PFF anchor weight elevated to 0.50 at DT (vs. 0.40 at DE/LB, 0.35 at CB/S) — reflects that PFF is the only film source with meaningful interior DL coverage depth. The IDP Show and IDP Guru cover DT but less deeply than off-ball positions. TDN coverage of interior DL prospects exists but is thinner than for offensive positions or pass-rush EDGE.

### Madden regulation parameters

- **Threshold:** 0.15 (normalized scale)
- **Blend scaling:** Linear gradient over 0.10 delta beyond threshold

### Madden attribute mapping

| Subjective Expert Claim | Madden Sub-Attribute / Composite | Formula |
|---|---|---|
| "Interior push / pocket collapser" | Power Moves (PMV) + Strength (STR) | Average(PMV, STR) |
| "Space eater / double-team anchor" | Block Shedding (BSH) + Strength (STR) | (0.6 × BSH) + (0.4 × STR) |
| "Elite lateral quickness / gap shooter" | Finesse Moves (FMV) + Acceleration (ACC) | Average(FMV, ACC) |
| "Run stop utility / high tackle rate" | Tackle (TAK) + Play Recognition (PRC) | Average(TAK, PRC) |
| "Hybrid pass-rush specialist / multi-move disruptor" | Power Moves (PMV) + Finesse Moves (FMV) + Block Shedding (BSH) | Average(PMV, FMV, BSH) |

Five rows. Fifth added to cover the Aaron Donald / Quinnen Williams archetype — the DT who wins with multiple moves and disrupts at multiple alignments. Asymmetric weighting on the Space Eater row (0.6 BSH + 0.4 STR) preserved from Gemini — Block Shedding is more directly predictive of double-team-anchor ability than raw Strength.

### Signal mechanics

- `film_position_weight`: 1.00 (standard — SL-005 compression expressed via cap + steepness only, not via position_weight)
- `film_inflection`: 0.50
- `film_steepness`: 10.0 (SL-005 compression — vs. standard 12.0)

### EMA blend rates (dynamic sub-signals)

| Sub-signal | α | Classification |
|---|---|---|
| PFF | **0.50 Year 1 → 0.10 Year 2+** | **DT-unique dynamic α per SL-021** |
| IDP Show | 0.30 | Dynamic — weekly subjective content, moderate blend |
| IDP Guru | 0.20 | Dynamic — weekly analytical content |
| TDN | N/A | **STATIC** — locked at rookie evaluation |
| Dynasty Nerds | 0.50 | Dynamic — annual-publication-rate content |
| Madden | 0.20 | Dynamic — multiple mid-season updates with moderate blend |

### Season transition behavior

CONTINUATION across all dynamic sub-signals.

### Sub-signal normalization (0.0–1.0 mapping)

- **PFF:** PFF Interior DL Grade / 100
- **IDP Show:** Tier ranking inverted to percentile within The IDP Show's DT pool
- **IDP Guru:** Weekly DT ranking inverted to percentile within IDP Guru's DT pool
- **TDN:** TDN composite grade mapped to percentile within draft class DT
- **Dynasty Nerds:** Dynasty DT ranking inverted to percentile within Dynasty Nerds' DT pool

---

## 3. RAS Component Configuration

**Cap (asymptote):** **±8%** (High-tier — needed for RAS to be "primary" at year 0 per SL-021).

### Parameters

- `RAS_position_weight`: **High-tier SL-021 schedule** (RAS is primary till NFL data accumulates):

| NFL career stage | RAS_position_weight |
|---|---|
| Rookie / pre-NFL data | 1.00 |
| After 1 NFL season | 0.50 |
| Year 2+ | 0.10 |

This is the High-tier SL-018 schedule, applied at a Medium-tier-classified position per SL-021. The hybrid is intentional.

- `RAS_inflection`: 0.50 (standard "universe of athletes" baseline)
- `RAS_steepness`: 10.0

### Normalization

`RAS_normalized = raw_RAS / 10.0`

Missing RAS → Layer 1 position-group mean imputation. Confidence flag set to Unknown.

### Late-career interaction — Cushion Guard (SL-021)

**SL-019 does NOT apply at DT.** The Late-Career Cushion Guard replaces it as the DT-unique late-career mechanism:

```
if Raw RAS ≥ 8.00:
    cushioned_age_pull = 1.0 − (1.0 − raw_age_pull) × 0.90
else:
    cushioned_age_pull = raw_age_pull   (no protection)
```

Worked example:
- DT age 33 (3 years past peak), raw_age_pull = 0.97^3 = 0.913
- Raw RAS 9.0 → triggers Cushion Guard → cushioned = 1.0 − (1.0 − 0.913) × 0.90 = 1.0 − 0.078 = **0.922**
- Raw RAS 7.0 → no Cushion Guard → 0.913

The Cushion Guard provides ~1pp of late-career protection per year past peak for qualifying DTs. Conservative vs. SL-019 (which provides ~5pp at S for elite-RAS Mathieu), reflecting that DT athletic-profile-to-longevity is less cleanly predictive than at coverage positions.

---

## 4. Breakout Component Configuration

**Cap (asymptote):** ±5% (standard — NOT compressed).

### Sub-signal weights

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.20 |
| School Tier | 0.20 |
| College Production Share (TFL + Sack market share) | 0.45 |
| Age Trajectory | 0.15 |

Sums to 1.00.

College Production Share elevated to 0.45 (highest of any position) — TFL + Sack market share is the cleanest single-metric capture of interior DT college production. Breakout Age slightly reduced (0.20 vs. typical 0.25) reflecting that DT breakout timing is less predictive than at pass-rush EDGE — many DTs ascend through technical development rather than year-zero dominance.

### Parameters

- `breakout_position_weight`: 1.00
- `breakout_inflection`: 0.50
- `breakout_steepness`: 11.0

### Normalization functions

**Breakout Age** — base curve:

| Breakout Age | Normalized |
|---|---|
| ≤20.0 | 1.00 |
| 21.0 | 0.75 |
| 22.0 | 0.45 |
| ≥23.0 | 0.15 |

Linear interpolation. **SL-019 modulation does NOT apply at DT.**

**School Tier** (template defaults):

| Tier | Normalized |
|---|---|
| Power Four | 1.00 |
| Group of Five | 0.70 |
| FCS | 0.40 |
| Non-FCS | 0.10 |

**College Production Share** (final-year TFL + Sack market share):

| Market Share | Normalized |
|---|---|
| ≥22% | 1.00 |
| 15% | 0.55 |
| ≤8% | 0.15 |

Linear interpolation.

**Age Trajectory** (current age relative to DT peak limit of 30):

| Age | Normalized |
|---|---|
| ≤26 | 1.00 |
| 27 | 0.85 |
| 28 | 0.70 |
| 29 | 0.55 |
| 30 (peak) | 0.50 |
| 31 | 0.35 |
| 32 | 0.20 |
| 33 | 0.10 |
| ≥34 | 0.00 |

**Cushion Guard applies to Age Trajectory beyond peak** if Raw RAS ≥ 8.00:
```
if Raw RAS ≥ 8.00 AND age > 30:
    cushioned_age_trajectory = 0.50 − (0.50 − base_value) × 0.90
                              = peak − (peak − base) × 0.90
```

Worked examples:
- Age 32, RAS 9.0 → base 0.20, cushioned = 0.50 − (0.50 − 0.20) × 0.90 = 0.50 − 0.27 = **0.23**
- Age 34+, RAS 9.0 → base 0.00, cushioned = 0.50 − (0.50 − 0.00) × 0.90 = 0.50 − 0.45 = **0.05**
- Age 32, RAS 7.0 → no Cushion Guard → 0.20

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

### Case 1 — Push: Jalen Carter

**Profile:**
- Age 25 (born May 2001) — 5 years before peak limit
- College: Georgia (Power Four — SEC)
- R1 #9 overall 2023
- Breakout age: 19 (true freshman impact at Georgia's loaded defensive line)
- Junior-year college production share: ~20% TFL + Sack market share (Georgia's championship defense distributed production across multiple R1 picks)
- RAS: ~9.5 *(estimated, pending ras.football verification — elite agility/explosion for size at pre-draft showings; full combine limited)*
- PFF: All-Pro caliber by Year 2, ~89 recent grade
- Year 4 NFL veteran (drafted 2023) → SL-018 Year 2+ tier (RAS_position_weight = 0.10 at High-tier schedule)

**Film component (SL-005 compressed: cap ±3%, steepness 10.0):**

Sub-signal normalizations:
- PFF Interior DL Grade ~89 → 0.89 *(dynamic α = 0.10 since Year 4)*
- IDP Show top-tier DT → 0.92
- IDP Guru top-5 DT → 0.92
- TDN (static, 2023 R1 #9 grade) → 0.92
- Dynasty Nerds top dynasty rank → 0.90

Composite:
```
(0.50 × 0.89) + (0.20 × 0.92) + (0.15 × 0.92) + (0.08 × 0.92) + (0.07 × 0.90)
= 0.445 + 0.184 + 0.138 + 0.0736 + 0.063
= 0.904
```

S-curve(0.904, 0.50, 10.0, 0.03):
- arg = 10.0 × 0.404 = 4.04
- σ(4.04) ≈ 0.9826
- output factor = 0.9652
- film_raw = 1 + 0.03 × 0.9652 = **1.029**
- film_effective = **1.029**

**RAS component (cap ±8%, Year 2+ schedule = 0.10):**

- RAS_normalized = 9.5 / 10 = 0.95
- S-curve(0.95, 0.50, 10.0, 0.08):
  - arg = 10.0 × 0.45 = 4.50
  - σ(4.50) ≈ 0.9890
  - output factor = 0.9781
  - RAS_raw = 1 + 0.08 × 0.9781 = **1.078**
- Year 2+ → RAS_position_weight = 0.10
- RAS_effective = 1.0 + (1.078 − 1.0) × 1.0 × 0.10 = **1.008**

Note: Even with High-tier ±8% cap, RAS contribution is residual at Year 4 (position_weight 0.10 suppresses). The High-tier cap matters most at Year 0 (rookie) where position_weight is 1.00 — for a rookie DT at this profile, RAS_effective would be 1.078, dominating Layer 4. By Year 4 the NFL data signal has fully taken over.

**Breakout component (no SL-019, no Cushion Guard — Carter is pre-peak):**

- Breakout Age 19 → 1.00 (≤20)
- School Tier P4 → 1.00
- College Production Share 20% → linear interp 15% (0.55) to 22% (1.00): 0.55 + 5 × 0.0643 = **0.871**
- Age Trajectory 25 → 1.00 (≤26)

Composite:
```
(0.20 × 1.00) + (0.20 × 1.00) + (0.45 × 0.871) + (0.15 × 1.00)
= 0.200 + 0.200 + 0.392 + 0.150
= 0.942
```

Elite zone (≥ 0.80).

S-curve(0.942, 0.50, 11.0, 0.05):
- arg = 11.0 × 0.442 = 4.862
- σ(4.862) ≈ 0.9923
- output factor = 0.9846
- breakout_raw = 1 + 0.05 × 0.9846 = **1.049**
- breakout_effective = **1.049**

**Layer 4 combined:**

```
Layer_4_Output = 1.029 × 1.008 × 1.049 = 1.088
```

**Multiplier: ~1.09** — push. Slightly lower magnitude than DE/CB/S High-tier elites due to SL-005 film compression.

**Full Layer 3 × Layer 4 chain for Carter:**

Layer 3 age_pull at age 25 (pre-peak) = no decay, age_pull = 1.000. Cushion Guard inactive (not past peak).

```
Layer 3 × Layer 4 (Carter) = 1.000 × 1.088 = 1.088
```

---

### Case 2 — Pull-attempt: Cameron Heyward

**Profile:**
- Age 37 (born May 1989) — 7 years past peak limit
- College: Ohio State (Power Four — Big Ten)
- R1 #31 overall 2011
- Breakout age: 19 (sophomore-year impact at Ohio State)
- Junior-year college production share: ~17% TFL + Sack market share (Ohio State defense distributed production; Heyward was multi-year starter but never a singular dominant producer in college metrics)
- RAS: ~7.5 *(estimated, pending ras.football verification — 5.08 40 at 294lbs combine, solid agility, NOT elite — below the 8.00 Cushion Guard threshold)*
- PFF: still useful, ~78 recent grade (was 85+ in prime 2017–2022; All-Pro at age 34 in 2023 — remarkably durable)
- Year 16 NFL veteran → SL-018 Year 2+ tier

**Film component (SL-005 compressed):**

Sub-signal normalizations:
- PFF Interior DL Grade ~78 → 0.78 *(dynamic α = 0.10 since Year 2+)*
- IDP Show declining-but-respected vet → 0.55
- IDP Guru top-25 DT → 0.55
- TDN (static, 2011 R1 #31 grade) → 0.70
- Dynasty Nerds low dynasty rank → 0.25

Composite:
```
(0.50 × 0.78) + (0.20 × 0.55) + (0.15 × 0.55) + (0.08 × 0.70) + (0.07 × 0.25)
= 0.390 + 0.110 + 0.0825 + 0.056 + 0.0175
= 0.656
```

S-curve(0.656, 0.50, 10.0, 0.03):
- arg = 10.0 × 0.156 = 1.56
- σ(1.56) ≈ 0.826
- output factor = 2 × 0.826 − 1 = 0.651
- film_raw = 1 + 0.03 × 0.651 = **1.020**
- film_effective = **1.020**

Note: Film still pushes because PFF at 78 is still respectable + TDN static at 0.70 (R1 grade) holds the composite up. SL-005 compression caps the push at ~1.02 even with these signals.

**RAS component (cap ±8%, Year 2+):**

- RAS_normalized = 7.5 / 10 = 0.75
- S-curve(0.75, 0.50, 10.0, 0.08):
  - arg = 10.0 × 0.25 = 2.50
  - σ(2.50) ≈ 0.9241
  - output factor = 0.8482
  - RAS_raw = 1 + 0.08 × 0.8482 = **1.068**
- Year 2+ → RAS_position_weight = 0.10
- RAS_effective = 1.0 + (1.068 − 1.0) × 1.0 × 0.10 = **1.007**

**Breakout component (no SL-019, NO Cushion Guard because RAS 7.5 < 8.00):**

- Breakout Age 19 → 1.00 (≤20)
- School Tier P4 → 1.00
- College Production Share 17% → linear interp 15% (0.55) to 22% (1.00): 0.55 + 2 × 0.0643 = **0.679**
- Age Trajectory 37 → ≥34 = 0.00 (NOT cushioned because RAS < 8.00)

Composite:
```
(0.20 × 1.00) + (0.20 × 1.00) + (0.45 × 0.679) + (0.15 × 0.00)
= 0.200 + 0.200 + 0.306 + 0.000
= 0.706
```

Average zone (0.40 < x < 0.80).

S-curve(0.706, 0.50, 11.0, 0.05):
- arg = 11.0 × 0.206 = 2.266
- σ(2.266) ≈ 0.906
- output factor = 0.811
- breakout_raw = 1 + 0.05 × 0.811 = **1.041**
- breakout_effective = **1.041**

**Layer 4 combined:**

```
Layer_4_Output = 1.020 × 1.007 × 1.041 = 1.069
```

**Multiplier: ~1.07** — **PUSH despite age 37.**

**Structural finding (ninth Lockett-pattern confirmation):** Heyward's static breakout signals (breakout age 19, P4 Ohio State, R1 #31 pedigree) hold the breakout component at 1.04 despite Age Trajectory crashing to 0.00 at age 37. Film still pushes at 1.02 because PFF 78 + TDN static at 0.70 + the SL-005 compression cap. Layer 4 lands at 1.07. The pattern is now confirmed at every position with engine value (WR, RB-with-Herbert-exception, TE, QB, DE, LB, CB, S, DT).

**Full Layer 3 × Layer 4 chain for Heyward:**

Layer 3 age_pull at age 37 = 0.97^7 ≈ 0.808. **NO Cushion Guard** because Raw RAS 7.5 < 8.00 threshold. Buffered age_pull = 0.808 (no protection).

```
Layer 3 × Layer 4 (Heyward) = 0.808 × 1.069 = 0.864
Layer 3 × Layer 4 (Carter)  = 1.000 × 1.088 = 1.088
```

A 21% spread on the full chain — comparable to S Mathieu (21%) and CB Peterson (22%). Layer 3 carries the aging entirely (no Cushion Guard for Heyward).

### Cushion Guard sidebar (Heyward hypothetical at RAS 8.5)

If Heyward's RAS were 8.5 instead of 7.5, Cushion Guard would activate:

**Breakout component with Cushion Guard:**
- Age Trajectory base 0.00 → cushioned = 0.50 − (0.50 − 0.00) × 0.90 = **0.05**
- Composite (other signals unchanged): (0.20 × 1.00) + (0.20 × 1.00) + (0.45 × 0.679) + (0.15 × 0.05) = 0.713
- S-curve → breakout_effective ≈ 1.042 (minimal change)

**Layer 3 with Cushion Guard:**
- raw_age_pull = 0.97^7 = 0.808
- cushioned_age_pull = 1.0 − (1.0 − 0.808) × 0.90 = 1.0 − 0.173 = **0.827**

**Full chain (Heyward hypothetical RAS 8.5):** 0.827 × ~1.070 = **0.885**

Vs. actual Heyward at RAS 7.5 (no Cushion Guard): 0.864.

**Cushion Guard provides ~2pp of late-career protection at this profile.** Modest compared to SL-019's ~5pp at S/CB — reflects the conservative 10% flat reduction. Threshold (8.00) deliberately filters to genuinely elite-RAS DTs.

---

## 6. Open Questions Surfaced

Prior sessions surfaced SL-OQ-013 through SL-OQ-036 and CAL-015 through CAL-029. DT adds:

- **SL-OQ-037:** Cushion Guard threshold — binary 8.00 vs. continuous RAS scaling. Current implementation per SL-021 uses Gemini's binary 8.00 threshold (active or inactive). SL-019 at other positions uses continuous RAS_normalized scaling (linear). Question: should Cushion Guard adopt continuous scaling for architectural consistency, or does DT's specific longevity profile justify the binary threshold? The binary approach has the virtue of being interpretable ("either you're an elite-athletic DT or you're not") and avoids spurious protection at mid-RAS values — but breaks the continuous-RAS pattern. Defer to live-data calibration.

- **SL-OQ-038:** Dynamic Year 1 / Year 2+ PFF EMA — should this mechanic propagate to other positions? At DT, dynamic α addresses a real cold-start problem (RAS-anchored rookie valuation needs to integrate NFL data quickly). The same problem exists at other positions, especially where film signal is otherwise weak (LB). Question for Session 3 (Engine Specification updates): should the dynamic α mechanic become a cross-rubric architectural option, or stay DT-unique? If propagated, every position rubric needs review.

- **SL-OQ-039 (from Gemini, renumbered from local SL-OQ-021):** Dynamic α down-shift trigger implementation. The SL-021 dynamic PFF α specification says α = 0.50 in Year 1 of NFL data accumulation, α = 0.10 in Year 2+. The down-shift mechanic requires a precise definition of when Year 1 ends. Options: (a) calendar 12 months from first NFL snap, (b) end of first NFL regular season, (c) end of first NFL league year (March of Year 2), (d) Week 1 of second NFL season. Each has different implications for off-season valuation behavior — a rookie drafted in April 2026 who plays his first snap September 2026: under option (a), down-shift triggers September 2027; under (b), January 2027; under (c), March 2027; under (d), September 2027. Concrete engineering question for the data integration layer. Recommendation: option (b) — end of first NFL regular season — aligns with the natural cadence of fantasy season transitions and matches the CONTINUATION season transition behavior. Confirm when ingestion module is specified.

**Calibration Backlog additions from DT build:**

- **CAL-030:** Cushion Guard threshold (8.00) and reduction strength (10%) empirical calibration. Both values currently per Christopher's call; require longitudinal high-RAS DT longevity data to validate. Specifically: does the 8.00 cutoff correctly identify the population of DTs who play meaningfully longer than peers? Does flat 10% velocity reduction correctly match observed late-career performance preservation for that population? Pairs with SL-OQ-037 — if calibration shows the binary threshold misranks mid-RAS DTs systematically, that data argues for continuous scaling.

- **CAL-031 (from Gemini, renumbered from local CAL-023 — global CAL-023 is already used for DE snap-count normalization):** Cushion Guard behavior across in-career role transitions. A DT who undergoes mass/weight transformation to transition between roles (3-technique gap shooter → 0-technique nose block-eater, or vice versa) effectively changes the physical profile his RAS captured at the combine. His original athletic measurements may no longer predict his current longevity arc — a 270lb 3-tech who bulks to 340lb nose tackle has different aging dynamics than his combine RAS implies. Question: should Cushion Guard re-evaluate against a current-body-state-adjusted RAS, or stay anchored to combine RAS? The combine-anchored approach is simpler and avoids speculation; a current-state approach would require body composition tracking in the data layer. Defer to live-data observation — flag when first DT in the universe undergoes documented role transition and observe whether Cushion Guard accuracy degrades.

---

## 7. Position-Specific Notes

- **Hybrid tier architecture (SL-021):** DT is the first position with explicitly hybrid tier configuration. Film treated as Medium-with-SL-005-compression (cap ±3%, steepness 10.0); RAS treated as High-tier (cap ±8%, schedule 1.00 / 0.50 / 0.10); plus two DT-unique mechanics (dynamic PFF α, Cushion Guard). The hybrid is intentional and resolves the SL-004 vs. SL-005 contradiction — DT's film signal genuinely is thin (Medium treatment justified) AND RAS is genuinely primary at year 0 baseline (High treatment justified). The architecture handles both truths simultaneously rather than forcing one tier classification.

- **Dynamic Year 1 / Year 2+ PFF α at DT — the cold-start mechanism:** At year 0, a rookie DT's valuation is RAS-anchored (RAS_position_weight = 1.00 at High-tier cap ±8% can drive Layer 4 to ~1.08). When real NFL grades arrive in Year 1, those grades need to integrate fast enough to displace the RAS anchor before mis-valuation persists into Year 2. Dynamic α = 0.50 in Year 1 achieves this by aggressively blending new PFF data into the running estimate. By Year 2+, the model has stabilized and standard α = 0.10 preserves vet signal stability. Mechanic is currently DT-unique pending Session 3 cross-rubric review (SL-OQ-038).

- **Cushion Guard at DT — conservative late-career protection:** Cushion Guard provides ~2pp of late-career chain protection at the Heyward profile (RAS 8.5 hypothetical). This is conservative vs. SL-019 (which provides ~5pp at elite-RAS S/CB veterans). The conservatism is appropriate — DT athletic-profile-to-longevity is less cleanly predictive than at coverage positions (interior strength + technique can preserve career length even at modest RAS; Heyward at RAS 7.5 is playing All-Pro football at 34 with no Cushion Guard credit). The 8.00 binary threshold deliberately filters to genuinely elite-athletic DTs.

- **Why SL-019 doesn't apply at DT:** Running SL-019 in parallel with Cushion Guard would double-protect elite-RAS DTs. The two mechanisms address the same architectural concern (late-career protection differentiated by athletic profile) with different math; SL-021 packages Cushion Guard as the DT choice. Future positions adopting SL-019-style late-career mechanics should pick one approach, not both.

- **EDGE classification reminder:** Per OQ-004 resolution, this rubric covers interior DTs only. Players consensus-classified as pass-rush-primary (DE/EDGE/3-4 OLB by role) route through the DE rubric regardless of MFL position tag.

- **Heyward verification as Cushion Guard non-trigger:** Heyward's RAS estimate at 7.5 is intentionally below the 8.00 Cushion Guard threshold to demonstrate the mechanic's filtering function. The sidebar "what if RAS were 8.5" shows the mechanic in action for comparison. A future verification round with Vita Vea (RAS 9.40) would demonstrate Cushion Guard actively triggered.

- All RAS values in verification cases are estimates pending ras.football verification.

---

## 8. Cross-Pollination Source

This rubric synthesized from:
- Universal Rubric Template v1.1 (structural skeleton)
- Engine Specification v2.1 (Layer 4 mechanics)
- Gemini's DT rubric draft (Medium-tier resolution path, sub-signal weight allocations with elevated PFF anchor at 0.50, archetype framework including asymmetric Space Eater weighting, base curves for breakout age and college production share with elevated 0.45 weight, the dynamic Year 1 / Year 2+ PFF α innovation, the Late-Career Cushion Guard innovation, named verification cases — first Gemini draft to do this)
- Gemini's DT open questions (reconciled — SL-OQ-039 dynamic α down-shift trigger implementation accepted as a concrete engineering question, recommendation of end-of-first-NFL-regular-season trigger to align with fantasy season cadence; CAL-031 Cushion Guard behavior across in-career role transitions renumbered from Gemini's local CAL-023 to avoid collision with global CAL-023 used for DE snap-count normalization, accepted as a live-data observation flag with combine-anchored default)
- Christopher's calls (SL-021 single-lock packaging of tier resolution + dynamic α + Cushion Guard; High-tier RAS schedule and ±8% cap; Cushion Guard interpretation A — decline-rate cushioned; Cameron Heyward pull case; both Gemini innovations confirmed against initial audit rejection)
- SL-018 applied with High-tier schedule (per Christopher's "RAS is primary till NFL data accumulates, 50% after 1 year, 10% subsequent")
- SL-019 explicitly excluded — Cushion Guard is the DT-specific late-career mechanism
- Dynamic Year 1 / Year 2+ PFF α applied per SL-021 (initially rejected in audit, re-incorporated after Christopher's call)
- Late-Career Cushion Guard applied per SL-021 (initially rejected in audit, re-incorporated after Christopher's call; decline-rate interpretation per Christopher)
- Layer 2 drivers expanded to include Forced Fumble (4), Opponent Fumble Recovery (3) + return yards, INT Caught (5) + Return Yards / TD, Blocked Kick (7), Safety (10) — Gemini's list was incomplete
- EMA blend rates fixed: IDP Show split from IDP Guru with α=0.30 / 0.20, TDN → STATIC at reduced weight 0.08, Dynasty Nerds added (Gemini omitted) at weight 0.07 / α=0.50
- Fifth Madden mapping row added (Hybrid pass-rush specialist — Aaron Donald / Quinnen Williams archetype)
- Age trajectory curve tightened from Gemini's "31+=0.20" plateau to gradual decline through 0.00 at age 34
- `[cite: N]` markers stripped throughout
- Verification cases retained Gemini's named-player approach but worked end-to-end with full S-curve math (Carter's chain re-derived correctly for Year 4 vet status; Heyward replaces Vea per Christopher's pick)

---

*Built by: Christopher Campbell + Claude (Anthropic)*

| Version | Date | Changes |
|---|---|---|
| 0.9 | June 2026 | Draft from cross-pollinated Gemini baseline + Christopher's architectural call. **SL-021 locked as DT tier resolution package** — Medium-tier classification with High-tier RAS schedule and cap, SL-005 film compression, dynamic Year 1 / Year 2+ PFF α, and Late-Career Cushion Guard (10% velocity reduction when Raw RAS ≥ 8.00, decline-rate interpretation). SL-019 explicitly excluded (Cushion Guard is DT's late-career mechanism). Two Gemini innovations accepted after Christopher's reframe overrode initial audit rejection. Both verification cases worked end-to-end with full math (Carter ~1.09 push at age 25 pre-peak Year 4; Heyward ~1.07 Layer 4 push despite age 37 — ninth confirmed Lockett-pattern instance; Cushion Guard does NOT trigger for Heyward at estimated RAS 7.5, sidebar shows what activation would look like at RAS 8.5 — ~2pp protection, conservative vs. SL-019's ~5pp). Pending Gemini's DT open questions for reconciliation pass before v1.0 lock. |
| 1.0 | June 2026 | Locked. Gemini's DT open questions reconciled. SL-OQ-039 (dynamic α down-shift trigger implementation) renumbered from Gemini's local SL-OQ-021, accepted as a concrete engineering question with end-of-first-NFL-regular-season recommended as the trigger to align with fantasy cadence. CAL-031 (Cushion Guard behavior across in-career role transitions — 3-tech to 0-tech mass/weight transformations) renumbered from Gemini's local CAL-023 to avoid collision with global CAL-023 (DE snap-count normalization); accepted as a live-data observation flag with combine-anchored RAS as the simpler default. |
