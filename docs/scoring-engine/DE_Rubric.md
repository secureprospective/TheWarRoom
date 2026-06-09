# Legacy NFL Position Group Rubric — Defensive End / Edge (DE)
**Version:** 1.0 — June 2026
**Status:** Locked. Second position to apply SL-019 RAS-modulator interactions (after TE). Per OQ-004 resolution, DE rubric encompasses all EDGE classifications — pass-rush-primary players tagged as DE, EDGE, or LB by consensus run through this rubric. Gemini's DE open questions reconciled into SL-OQ-030 and CAL-023.
**Companion:** Engine_Specification.md Layer 4 is authoritative on mechanics. This rubric specifies position-specific values and the SL-019 mechanic at DE.

---

## 1. Architectural Baseline

- **Layer 4 RAS Tier:** **High** (per SL-004). RAS directly shapes baseline multiplier at year 0; collapses to residual by year 2+ per SL-018. High-tier consistent with TE and offensive skill positions — athletic profile is dominant at EDGE.
- **Layer 3 Peak Limit:** 30 years.
- **SL-019 application:** **YES.** DE applies RAS-modulator interactions to (a) breakout-age sub-signal normalization, (b) age-trajectory sub-signal normalization, (c) Layer 3 late-career age decay buffer (amplified to 0.30 × RAS_normalized, same as TE). Default modulation strengths held at TE values (0.35/0.35/0.30) pending empirical calibration.
- **SL-019 gating rule resolved:** SL-OQ-027 closes at this rubric — see Section 7.
- **EDGE mapping rule (per OQ-004 resolution):** Pass-rush-primary defenders route through this rubric regardless of MFL tag (DE / EDGE / 3-4 OLB). Coverage- and run-stop-primary off-ball linebackers route through the LB rubric.
- **Layer 2 Base Points drivers** (per Official Rulebook lines 88–117, MFL-sourced per DECISION-009):
  - **True Position split (DT/DE only):** Tackle 2.5, Assist 1.5
  - **Pass-rush production:** Sack 4.5, QB Hit 1, Tackle for Loss 2.5 (direct stat — no proxy)
  - **Coverage:** Pass Defensed 2.5, Interception Caught 5, Interception Return Yards 0.025/yd, Interception Return TD 6
  - **Turnover production:** Forced Fumble 4, Opponent Fumble Recovery 3, Opponent Fumble Recovery Yards 0.025/yd
  - **Game-state:** Safety 10
- **Data Parity Rule:** Missing/Unknown sub-signal data collapses component deviation to 0.0 via confidence weighting, returning neutral 1.00 fallback.

---

## 2. Film Component Configuration

**Cap (asymptote):** ±5% (standard SL-002 upper bound).

### Sub-signal weights

| Source | Weight | Classification |
|---|---|---|
| PFF Edge Defense Grade (analytical anchor) | 0.40 | Analytical — self-regulates |
| The IDP Show (subjective anchor) | 0.30 | Subjective — Madden-regulated |
| The Draft Network (pre-draft trait eval) | 0.15 | Subjective — Madden-regulated |
| Dynasty Nerds / IDP Guru combined | 0.15 | Subjective — Madden-regulated |

Sums to 1.00.

PFF anchor reflects DE's IDP data richness — pass-rush production grades and pressure rates are among the most reliable analytical signals in football. RSP intentionally not included as a DE sub-signal — Waldman's coverage is offense-focused. The IDP Show fills the subjective-anchor role RSP plays at offensive positions.

### Madden regulation parameters

- **Threshold:** 0.15 (normalized scale)
- **Blend scaling:** Linear gradient over 0.10 delta beyond threshold

### Madden attribute mapping

| Subjective Expert Claim | Madden Sub-Attribute / Composite | Formula |
|---|---|---|
| "Elite speed rush / first-step explosion" | Speed (SPD) + Acceleration (ACC) | Average(SPD, ACC) |
| "Power rusher / heavy hands / bull rush" | Power Moves (PMV) + Strength (STR) | Average(PMV, STR) |
| "Technical rusher / counter-flex / hand fighter" | Finesse Moves (FMV) + Agility (AGI) | Average(FMV, AGI) |
| "Elite edge setter / run squeezer" | Block Shedding (BSH) + Tackle (TAK) | Average(BSH, TAK) |
| "High motor / relentless pursuit" | Pursuit (PUR) + Play Recognition (PRC) | Average(PUR, PRC) |
| "Complete edge / power-finesse hybrid" | Power Moves (PMV) + Finesse Moves (FMV) + Block Shedding (BSH) | Average(PMV, FMV, BSH) |

Six rows. Sixth added to cover the Garrett/Bosa archetype — the "complete edge" that wins with both power AND finesse and is the dynasty difference-maker at the position. Gemini's five rows collapse this archetype to either pure-speed or pure-power readings.

### Signal mechanics

- `film_position_weight`: 1.00
- `film_inflection`: 0.50
- `film_steepness`: 12.0

### EMA blend rates (dynamic sub-signals)

| Sub-signal | α | Classification |
|---|---|---|
| PFF | 0.15 | Dynamic — weekly grades with slow blend |
| IDP Show | 0.30 | Dynamic — weekly podcast/content, moderate blend |
| TDN | N/A | **STATIC** — locked at rookie evaluation; no re-publication for veterans |
| Dynasty Nerds / IDP Guru | 0.50 | Dynamic — annual-publication-rate weekly content, half-weight blend |
| Madden | 0.20 | Dynamic — multiple mid-season updates with moderate blend |

### Season transition behavior

CONTINUATION across all dynamic sub-signals.

### Sub-signal normalization (0.0–1.0 mapping)

- **PFF:** PFF Edge Defense Grade / 100 (grades are already 0–100)
- **IDP Show:** Tier ranking inverted to percentile within The IDP Show's EDGE pool
- **TDN:** TDN composite grade mapped to percentile within draft class EDGE
- **Dynasty Nerds / IDP Guru:** Combined IDP dynasty rank inverted to percentile within combined EDGE pool

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

- `RAS_inflection`: 0.50 (equivalent to raw RAS = 5.00)
- `RAS_steepness`: 10.0

### Normalization

`RAS_normalized = raw_RAS / 10.0`

Missing RAS → Layer 1 position-group mean imputation. Confidence flag set to Unknown.

### Late-career interaction (SL-018 + SL-019 amplified buffer)

Once player age > 30 (DE peak limit), Layer 3 age_pull is RAS-buffered. **SL-019 amplifies the buffer for DE (3× the standard 0.10 rate):**

```
buffer_pct        = 0.30 × RAS_normalized   ← DE-specific (matches TE)
buffered_age_pull = 1.0 + (raw_age_pull − 1.0) × (1 − buffer_pct)
```

A DE with RAS = 9.99 gets a ~30% buffer against age decay. A DE with RAS = 5.0 gets a 15% buffer. Elite-athleticism EDGE defenders play meaningfully longer at peak (Garrett, JJ Watt's prime years, Suggs late-career) — the architecture rewards that.

**SL-018 scope:** SL-019 modulator interactions are INDEPENDENT of SL-018 decay. SL-018 governs the RAS COMPONENT weight only. RAS as a modulator of other curves remains active across career — an aging DE's athletic profile is a structural attribute that doesn't become residual once NFL data accumulates.

---

## 4. Breakout Component Configuration

**Cap (asymptote):** ±5%.

### Sub-signal weights

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.30 |
| School Tier | 0.20 |
| College Production Share (Sack + TFL market share) | 0.35 |
| Age Trajectory | 0.15 |

Sums to 1.00.

College Production Share elevated to 0.35 (vs. WR's 0.20 College Usage Rate, TE's 0.30) — sack + TFL market share is the cleanest pre-NFL pass-rush production signal and the strongest college-to-NFL predictor at this position. Breakout Age held at 0.30 (slightly below WR/RB) reflecting that EDGE players develop technique into their early 20s more than skill-position players.

### Parameters

- `breakout_position_weight`: 1.00
- `breakout_inflection`: 0.50
- `breakout_steepness`: 11.0

### Normalization functions

**Breakout Age** — base curve (EDGE breakouts at 18–19 are rare but distinguishing):

| Breakout Age | Normalized |
|---|---|
| ≤19.5 | 1.00 |
| 20.0 | 0.80 |
| 20.5 | 0.50 |
| ≥21.0 | 0.15 |

Linear interpolation between defined points.

**SL-019 modulation applies:**
```
breakout_age_modulated = base + (1.0 − base) × 0.35 × RAS_normalized
```

Worked examples:
- Base 0.15, RAS 9.99 → 0.15 + 0.85 × 0.35 × 0.999 = **0.447**
- Base 0.15, RAS 7.50 → 0.15 + 0.85 × 0.35 × 0.75 = **0.373**
- Base 0.15, RAS 4.18 → 0.15 + 0.85 × 0.35 × 0.418 = **0.274**
- Base 1.00, any RAS → **1.00** (already maxed, no modulator effect)

**School Tier** (template defaults):

| Tier | Normalized |
|---|---|
| Power Four | 1.00 |
| Group of Five | 0.70 |
| FCS | 0.40 |
| Non-FCS | 0.10 |

**College Production Share** (final-year sack + TFL market share):

| Market Share | Normalized |
|---|---|
| ≥28% | 1.00 |
| 20% | 0.55 |
| ≤12% | 0.15 |

Linear interpolation between defined points.

**Age Trajectory** (current age relative to DE peak limit of 30):

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

**SL-019 modulation applies:**
```
age_trajectory_modulated = base + (1.0 − base) × 0.35 × RAS_normalized
```

Worked examples:
- Base 0.50 (age 30), RAS 9.99 → 0.50 + 0.50 × 0.35 × 0.999 = **0.675**
- Base 0.20 (age 32), RAS 7.50 → 0.20 + 0.80 × 0.35 × 0.75 = **0.410**
- Base 0.00 (age 34), RAS 4.18 → 0.00 + 1.00 × 0.35 × 0.418 = **0.146**

### Three-zone classification

Thresholds on the composite input (weighted sum before S-curve):

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

### Case 1 — Push: Myles Garrett

**Profile:**
- Age 30 (born Dec 1995, June 2026) — at peak limit
- College: Texas A&M (Power Four — SEC)
- R1 #1 overall 2017
- Breakout age: 18 (true freshman impact, 11.5 sacks)
- Junior-year college production share: ~31% combined sack + TFL market share
- RAS: 9.99 *(estimated, pending ras.football verification — one of highest combine scores ever recorded)*
- PFF: elite recent grades, ~92
- Year 9 NFL veteran → SL-018 Year 2+ tier (RAS_position_weight = 0.10)

**Film component:**

Sub-signal normalizations:
- PFF Edge Defense Grade ~92 → 0.92
- IDP Show top-3 EDGE consensus → 0.95
- TDN (static, 2017 rookie eval — #1 overall grade) → 0.95
- Dynasty Nerds / IDP Guru top-tier dynasty rank → 0.95

Composite:
```
(0.40 × 0.92) + (0.30 × 0.95) + (0.15 × 0.95) + (0.15 × 0.95)
= 0.368 + 0.285 + 0.1425 + 0.1425
= 0.938
```

S-curve(0.938, 0.50, 12.0, 0.05):
- arg = 12.0 × 0.438 = 5.256
- σ(5.256) ≈ 0.9948
- output factor = 2 × 0.9948 − 1 = 0.9896
- film_raw = 1 + 0.05 × 0.9896 = **1.049**
- film_effective = **1.049**

**RAS component (SL-018 Year 2+):**

- RAS_normalized = 9.99 / 10 = 0.999
- S-curve(0.999, 0.50, 10.0, 0.08):
  - arg = 10 × 0.499 = 4.99
  - σ(4.99) ≈ 0.9933
  - output factor = 0.9865
  - RAS_raw = 1 + 0.08 × 0.9865 = **1.079**
- Year 2+ → RAS_position_weight = 0.10
- RAS_effective = 1.0 + (1.079 − 1.0) × 1.0 × 0.10 = **1.008**

**Breakout component (SL-019 modulators applied):**

- Breakout Age 18 → base 1.00 (already maxed, no modulator effect)
- School Tier P4 → 1.00
- College Production Share 31% → 1.00 (above 28% threshold)
- Age Trajectory 30 (peak) → base 0.50, modulated = 0.50 + 0.50 × 0.35 × 0.999 = **0.675**

Composite:
```
(0.30 × 1.00) + (0.20 × 1.00) + (0.35 × 1.00) + (0.15 × 0.675)
= 0.300 + 0.200 + 0.350 + 0.101
= 0.951
```

Composite is in the **Elite zone** (≥ 0.80).

S-curve(0.951, 0.50, 11.0, 0.05):
- arg = 11.0 × 0.451 = 4.961
- σ(4.961) ≈ 0.9931
- output factor = 0.9861
- breakout_raw = 1 + 0.05 × 0.9861 = **1.049**
- breakout_effective = **1.049**

**Layer 4 combined:**

```
Layer_4_Output = 1.049 × 1.008 × 1.049 = 1.109
```

**Multiplier: ~1.11** — clear push case.

**Full Layer 3 × Layer 4 chain for Garrett:**

Layer 3 age_pull at age 30 = 0.97^0 = 1.000 (at peak, no decay yet). No buffer applied.

```
Layer 3 × Layer 4 (Garrett) = 1.000 × 1.109 = 1.109
```

---

### Case 2 — Pull: Demarcus Lawrence

**Profile:**
- Age 34 (born April 1992, June 2026) — 4 years past peak limit
- College: Boise State (Group of Five — Mountain West); JUCO transfer from Butler CC
- R2 #34 overall 2014
- Breakout age: 21 (junior year at Boise — first NCAA D1 lead-rusher season, 9.5 sacks)
- Junior-year college production share: ~25% combined sack + TFL market share at Boise
- RAS: 4.18 *(estimated, pending ras.football verification — technical/effort rusher profile, 4.80 40 at 251lbs with limited explosion testing)*
- PFF: declining, ~65 recent grade (was 78–85 in prime 2017–2019)
- Year 12 NFL veteran → SL-018 Year 2+ tier (RAS_position_weight = 0.10)

**Film component:**

Sub-signal normalizations:
- PFF Edge Defense Grade ~65 → 0.65
- IDP Show clearly past prime tier → 0.35
- TDN (static, 2014 rookie eval — R2 #34 grade) → 0.55
- Dynasty Nerds / IDP Guru low veteran dynasty rank → 0.20

Composite:
```
(0.40 × 0.65) + (0.30 × 0.35) + (0.15 × 0.55) + (0.15 × 0.20)
= 0.260 + 0.105 + 0.0825 + 0.030
= 0.478
```

S-curve(0.478, 0.50, 12.0, 0.05):
- arg = 12.0 × (−0.022) = −0.264
- σ(−0.264) ≈ 0.4344
- output factor = 2 × 0.4344 − 1 = −0.1313
- film_raw = 1 + 0.05 × (−0.1313) = **0.993**
- film_effective = **0.993**

**RAS component (SL-018 Year 2+):**

- RAS_normalized = 4.18 / 10 = 0.418
- S-curve(0.418, 0.50, 10.0, 0.08):
  - arg = 10 × (−0.082) = −0.82
  - σ(−0.82) ≈ 0.3057
  - output factor = −0.3886
  - RAS_raw = 1 + 0.08 × (−0.3886) = **0.969**
- Year 2+ → RAS_position_weight = 0.10
- RAS_effective = 1.0 + (0.969 − 1.0) × 1.0 × 0.10 = **0.997**

**Breakout component (SL-019 modulators applied):**

- Breakout Age 21 → base 0.15, modulated = 0.15 + 0.85 × 0.35 × 0.418 = **0.274**
- School Tier G5 → 0.70
- College Production Share 25% → linear interp between 20% (0.55) and 28% (1.00): 0.55 + 5 × (0.45/8) = **0.831**
- Age Trajectory 34 → base 0.00, modulated = 0.00 + 1.00 × 0.35 × 0.418 = **0.146**

Composite:
```
(0.30 × 0.274) + (0.20 × 0.70) + (0.35 × 0.831) + (0.15 × 0.146)
= 0.0822 + 0.140 + 0.2909 + 0.0219
= 0.535
```

Composite is in the **Average zone** (0.40 < x < 0.80).

S-curve(0.535, 0.50, 11.0, 0.05):
- arg = 11.0 × 0.035 = 0.385
- σ(0.385) ≈ 0.595
- output factor = 0.190
- breakout_raw = 1 + 0.05 × 0.190 = **1.010**
- breakout_effective = **1.010**

**Layer 4 combined:**

```
Layer_4_Output = 0.993 × 0.997 × 1.010 = 1.000
```

**Multiplier: ~1.00** — Layer 4 sits essentially neutral.

**Structural finding (Lockett pattern at DE):** Layer 4 alone does NOT pull a declining veteran below 1.0 even with G5 school + late breakout + low RAS. Three of four breakout sub-signals (breakout age, school tier, college production share) are static and reflect what Lawrence was at draft entry. His Boise junior-year production share was real (25% market share at G5 is meaningful), and SL-019 lifts his late breakout age modestly via his still-positive-but-modest RAS contribution. Combined with mild film pull and near-neutral RAS, the components compound to essentially 1.0.

This is consistent with WR (Lockett 1.01), TE (Henry 1.05, Higbee 0.99 only at G5+age 22 breakout), and QB (Carr 1.01). **Layer 3 does the aging work**, as designed.

**Full Layer 3 × Layer 4 chain for Lawrence:**

Layer 3 age_pull at age 34 = 0.97^4 ≈ 0.885. SL-019 amplified buffer = 0.30 × 0.418 = 0.125 (12.5%). Buffered age_pull = 1.0 + (0.885 − 1.0) × (1 − 0.125) = 1.0 + (−0.115)(0.875) = **0.900**.

```
Layer 3 × Layer 4 (Lawrence) = 0.900 × 1.000 = 0.900
Layer 3 × Layer 4 (Garrett)  = 1.000 × 1.109 = 1.109
```

A 23% spread between an in-peak elite EDGE and a four-years-past-peak veteran — produced by the engine separating "what kind of player is this?" from "how long does he have?"

The Lockett pattern is now formalized at five positions (WR, RB-Herbert-exception, TE, QB, DE). Veteran-era re-weighting of static breakout signals is the proper venue for Deliverable 3 (Veteran Scouting Layer Extension, SL-008).

---

## 6. Open Questions Surfaced

Prior sessions surfaced SL-OQ-013 through SL-OQ-028 and CAL-015 through CAL-021. DE adds:

- **SL-OQ-029:** DE college production share data source. The sack + TFL market share calculation requires team total sack and TFL data from college. Sports Reference College Football has team-level sack data but TFL coverage is inconsistent at G5/FCS levels. PFF College has both but is paywalled and not in the approved source set. Pipeline question — how is this signal sourced at scale for the rookie class each year? Manual research is acceptable for Phase 1 (Christopher's competitive edge), but Phase 2 needs an automation path.

- **SL-OQ-030 (from Gemini, renumbered from local SL-OQ-017):** Mid-season MFL position re-tagging behavior for hybrid-scheme defenders. A player like Haason Reddick or Micah Parsons may have their MFL position tag cycle between LB and DE within a single season as defensive coordinators alter usage. Per OQ-004 resolution, rubric routing is by consensus role classification (pass-rush primary → DE rubric), not by MFL tag — so tag flips should not change rubric routing once a player is classified for the season. Operational question for the ingestion layer: freeze role classification at season start, or monitor MFL re-tagging events and surface them as flags for manual review? Recommended default: classification locked at season start, MFL tag changes logged but ignored for routing. Requires confirmation when the ingestion module is specified.

- **SL-OQ-027 RESOLVED (closure documented in Section 7):** The SL-019 applicability gating rule. Criterion established with DE as second SL-019 instance — High-tier RAS + position-specific predictive relationship between athletic profile and longevity arc. Documented in Section 7. Remaining positions (LB, CB, S, DT, K) evaluated against this rule in their respective rubrics.

**Calibration Backlog additions from DE build:**

- **CAL-022:** SL-019 modulation strengths empirical tune for DE — breakout age modulator (currently 0.35), age trajectory modulator (currently 0.35), Layer 3 buffer multiplier (currently 0.30 × RAS_normalized). Held at TE values for v0.9. Requires longitudinal athletic-EDGE longevity data to determine whether DE wants the same strengths as TE or asymmetric values (DE peak is hard-capped at 30 vs. TE at 29 but with longer practical play windows for elite athletes — Watt, Suggs, Demarcus Ware all played productively past 32).

- **CAL-023 (from Gemini, renumbered from local CAL-019):** College Production Share snap-count normalization. The current 0.35-weighted sack + TFL market share signal does not account for defensive snap participation. A rotational DE playing 45 pass-rush snaps per game has fewer opportunities than a three-down DE playing 65 — raw market share rewards the latter and depresses the former. Refinement: normalize market share by pass-rush snap share (or total defensive snap share as a proxy where pass-rush snap data is unavailable). Requires college snap-count data, which is inconsistently published at the G5/FCS level. **Linked dependency with SL-OQ-029** — both are blocked by the same college-data source question; resolve together once the pipeline data source is settled.

---

## 7. Position-Specific Notes

- **SL-OQ-027 closure — SL-019 applicability gating rule:** SL-019 RAS-modulator interactions apply to a position when BOTH conditions are met:
  1. **High-tier RAS** per SL-004 (athletic profile is a primary Layer 4 signal at this position)
  2. **Position-specific predictive relationship** between athletic profile and longevity arc — elite-RAS players at this position demonstrably play meaningfully longer at productive levels than mid-RAS players (separable in career-length distributions)

  Confirmed applicable: **TE, DE**.
  Confirmed not applicable: **QB** (Low-tier RAS forces SL-020 exclusion), **RB** (Medium-tier — RAS less dominant, and RB attrition is injury-driven rather than athletic-decay-driven).
  Pending determination in subsequent rubrics: **LB** (Medium — likely no), **CB** (High — likely yes, athletic profile critical for coverage longevity), **S** (High — likely yes, same reasoning), **DT** (tier unresolved — pending Section 8 in DT rubric), **K** (excluded per Low-tier per SL-020).

- DE is the first defensive position to apply SL-019. Mechanic transfers cleanly from TE — same modulator strengths (0.35/0.35/0.30) and same independence from SL-018 decay.

- Per OQ-004 resolution, this rubric encompasses all pass-rush-primary EDGE classifications regardless of MFL position tag. The disambiguation between this rubric and the LB rubric is consensus role classification (pass-rush primary vs. coverage/run-stop primary), not MFL string matching.

- RSP intentionally omitted from sub-signal list. Waldman's coverage is offense-focused and his DE/EDGE coverage is too thin to weight. The IDP Show fills the subjective-anchor role.

- Verification cases follow the Lockett pattern at five positions now (WR, RB-with-Herbert-exception, TE, QB, DE). Layer 4 = "what kind of player is this?" remains a development-era statement; Layer 3 = "how long does he have?" carries the aging work. Veteran Scouting Layer Extension (Deliverable 3, SL-008) is the venue for re-weighting static breakout signals across NFL career stages.

- All RAS values in verification cases are estimates pending ras.football verification. Garrett's 9.99 and Lawrence's 4.18 are widely-reported figures but should be locked from the source before final calibration runs.

---

## 8. Cross-Pollination Source

This rubric synthesized from:
- Universal Rubric Template v1.1 (structural skeleton)
- Engine Specification v2.1 (Layer 4 mechanics)
- Gemini's DE rubric draft (sub-signal weight allocations, Madden archetype framework, base curves for breakout age and college production share, S-curve parameters)
- Gemini's DE open questions (reconciled — SL-OQ-030 mid-season MFL re-tagging accepted with recommended default of season-start lock; CAL-023 college production share snap-count normalization accepted with linked-dependency note to SL-OQ-029 since both blocked by the same college-data source question)
- Christopher's calls (hold SL-019 modulators at TE defaults; IDP Show α=0.30; Demarcus Lawrence as pull case; SL-OQ-027 closure documented in Section 7)
- SL-018 applied (Gemini's draft predated this mechanic)
- SL-019 applied (Gemini's draft predated this mechanic — DE is second instance after TE)
- Layer 2 drivers expanded to include Forced Fumble (4), Opponent Fumble Recovery (3) + return yards, INT Return TD (6) + return yards — Gemini's list was incomplete
- EMA blend rates fixed: TDN → STATIC, Dynasty Nerds/IDP Guru split from TDN at α=0.50, IDP Show α=0.30
- Sixth Madden mapping row added (Complete edge / power-finesse hybrid — Garrett/Bosa archetype)
- Age trajectory curve tightened from Gemini's "30+=0.20" plateau to gradual decline through 0.00 at age 34
- `[cite: N]` markers stripped throughout
- Verification cases replaced from generic placeholders with named real-player cases and full S-curve math

---

*Built by: Christopher Campbell + Claude (Anthropic)*

| Version | Date | Changes |
|---|---|---|
| 0.9 | June 2026 | Draft from cross-pollinated Gemini baseline + audit refinements. Second SL-019 instance after TE. SL-018 + SL-019 applied. Layer 2 drivers expanded with missing rulebook lines. OQ-004 EDGE mapping rule encoded. SL-OQ-027 closure documented (gating rule for SL-019 applicability). Both verification cases worked end-to-end with full math (Garrett ~1.11 push; Lawrence ~1.00 Layer 4 with Lockett-pattern finding, ~0.90 on full Layer 3 × Layer 4 chain). Pending Gemini's DE open questions for reconciliation pass before v1.0 lock. |
| 1.0 | June 2026 | Locked. Gemini's DE open questions reconciled. SL-OQ-030 (mid-season MFL re-tagging) renumbered from Gemini's local SL-OQ-017, accepted with recommended default of season-start classification lock per OQ-004 routing rule. CAL-023 (college production share snap-count normalization) renumbered from Gemini's local CAL-019, accepted with linked-dependency note to SL-OQ-029 — both are blocked by the same college-data source question. |
