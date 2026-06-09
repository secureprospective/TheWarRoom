# Legacy NFL Position Group Rubric — Tight End (TE)
**Version:** 1.0 — June 2026
**Status:** Locked. Introduces SL-019 RAS-modulator interactions as a generalized mechanic, applied to TE as the first instance. Validates Universal Rubric Template v1.1 against the position with the longest development curve and most RAS-dependent longevity arc. Gemini's TE open questions reconciled into SL-OQ-025 and CAL-019.
**Companion:** Engine_Specification.md Layer 4 is authoritative on mechanics. This rubric specifies position-specific values and the SL-019 mechanic.

---

## 1. Architectural Baseline

- **Layer 4 RAS Tier:** **High** (per amended SL-004). Originally Medium-tier per SL-004 v1.0, amended to High based on TE-specific finding: athletic profile is the dominant explanatory variable at this position. SL-004 amendment lands in Roadmap_and_Open_Questions.md at session close.
- **Layer 3 Peak Limit:** 29 years (same as WR).
- **Layer 2 Base Points drivers** (per Official Rulebook, MFL-sourced per DECISION-009):
  - Receptions (1.0), Rec Yards (0.1/yd), Rec TDs (6.0)
  - Stackable Long Play threshold bonuses (+1 at 20+ yds, +1 additional at 40+ yds)
  - 2PT conversions (2.0)
- **SL-019 application:** TE applies RAS-modulator interactions to (a) breakout-age sub-signal normalization, (b) age trajectory sub-signal normalization, (c) Layer 3 late-career age decay buffer. See sections 3 and 4 for mechanics.
- **Data Parity Rule:** Missing/Unknown sub-signal data collapses component deviation to 0.0 via confidence weighting, returning neutral 1.00 fallback.

---

## 2. Film Component Configuration

**Cap (asymptote):** ±5% (high end of SL-002 range).

### Sub-signal weights

| Source | Weight | Classification |
|---|---|---|
| PFF Overall TE Grade (analytical anchor) | 0.40 | Analytical — self-regulates |
| Matt Waldman RSP (subjective anchor) | 0.35 | Subjective — Madden-regulated |
| The Draft Network | 0.15 | Subjective — Madden-regulated |
| Sharp Football Analysis | 0.10 | Subjective — Madden-regulated |

Sums to 1.00. PFF/RSP inverted from WR/RB (PFF higher than RSP at TE) — PFF captures both receiving and blocking grades, broader coverage than for receivers, separates good from mediocre TEs more cleanly because TE production has higher variance than WR/RB.

### Madden regulation parameters

- **Threshold:** 0.15 (normalized scale)
- **Blend scaling:** Linear gradient over 0.10 delta beyond threshold

### Madden attribute mapping

| Subjective Expert Claim | Madden Sub-Attribute / Composite | Formula |
|---|---|---|
| "Seam stretcher / elite speed" | Speed (SPD) + Acceleration (ACC) | Average(SPD, ACC) |
| "Inline blocker / point of attack" | Run Block (RBK) + Pass Block (PBK) + Strength (STR) | (0.4 × RBK) + (0.4 × PBK) + (0.2 × STR) |
| "Contested catch / red-zone weapon" | Catching (CTH) + Catch in Traffic (CIT) + Jumping (JMP) | Average(CTH, CIT, JMP) |
| "Route runner / separation" | Short Route Running (SRR) + Medium Route Running (MRR) | Average(SRR, MRR) |
| "YAC threat / elusive" | Break Tackle (BTK) + Agility (AGI) | Average(BTK, AGI) |
| "Move TE / H-back versatility" | Run Block (RBK) + Catching (CTH) + Agility (AGI) | Average(RBK, CTH, AGI) |

Six rows. Sixth added to cover hybrid-role TEs (Kittle, Freiermuth, Kmet types) that combine blocking with receiving — Gemini's five rows miss this archetype, which collapses to either pure-blocker or pure-receiver readings without the hybrid mapping.

### Signal mechanics

- `film_position_weight`: 1.00
- `film_inflection`: 0.50
- `film_steepness`: 12.0

### EMA blend rates (dynamic sub-signals)

| Sub-signal | α | Classification |
|---|---|---|
| RSP | 0.50 | Dynamic — annual publication blend |
| PFF | 0.12 | Dynamic — slightly slower than WR (0.15) reflecting TE production noise |
| TDN | N/A | **STATIC** — locked at rookie evaluation |
| Sharp | 0.50 | Dynamic — annual + occasional updates |
| Madden | 0.20 | Dynamic — multiple mid-season updates |

### Season transition behavior

CONTINUATION across all dynamic sub-signals.

### Sub-signal normalization (0.0–1.0 mapping)

- **PFF:** PFF TE composite (receiving + blocking, weighted by snap distribution) / 100
- **RSP:** Percentile rank within annual RSP TE rankings
- **TDN:** TDN composite grade mapped to percentile within draft class TEs
- **Sharp:** Sharp dynasty ranking inverted to percentile within Sharp's TE pool

---

## 3. RAS Component Configuration

**Cap (asymptote):** ±8% (High-tier per amended SL-004).

### Parameters

- `RAS_position_weight`: **1.00 at year 0 baseline (High-tier)**, modulated by SL-018:

| NFL career stage | RAS_position_weight |
|---|---|
| Rookie / pre-NFL data | 1.00 |
| After 1 NFL season | 0.50 |
| Year 2+ | 0.10 |

- `RAS_inflection`: 0.50
- `RAS_steepness`: **11.0** (steeper than WR's 10.0). RAS at TE is more sharply predictive than at WR — small differences in athletic profile produce larger separations in NFL outcomes. The 8.5-RAS vs 9.5-RAS TE gap is often the difference between a starter and a hall-of-famer.

### Normalization

`RAS_normalized = raw_RAS / 10.0`

Missing RAS → Layer 1 position-group mean imputation. Confidence flag set to Unknown.

### Late-career interaction (SL-018 + SL-019 amplified buffer)

Once player age > 29 (TE peak limit), Layer 3 age_pull is RAS-buffered. **SL-019 amplifies the buffer for TE specifically:**

```
buffer_pct        = 0.30 × RAS_normalized   ← TE-specific (3× WR's 0.10 standard)
buffered_age_pull = 1.0 + (raw_age_pull − 1.0) × (1 − buffer_pct)
```

A TE with RAS = 9.5 gets a 28.5% buffer against age decay (vs. WR's 9.5%). A TE with RAS = 5.0 gets a 15% buffer. Athletic TEs play meaningfully longer at peak — Kittle, Gronk, Kelce-as-outlier — and the architecture rewards that.

**SL-018 scope:** SL-019 modulator interactions are INDEPENDENT of SL-018 decay. SL-018 governs the RAS COMPONENT weight only. RAS as a modulator of other curves remains active across career — an aging TE's athletic profile is a structural attribute that doesn't become residual once NFL data accumulates.

---

## 4. Breakout Component Configuration

**Cap (asymptote):** ±5%.

### Sub-signal weights

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.35 |
| School Tier | 0.20 |
| College Usage Rate | 0.30 |
| Age Trajectory | 0.15 |

Sums to 1.00. Same allocation as RB (vs. WR's 0.40/0.25/0.20/0.15). College usage rate elevated to 0.30 reflects TE's reliance on solving the "did he produce in college" question, since TE breakout age is naturally later than WR.

### Parameters

- `breakout_position_weight`: 1.00
- `breakout_inflection`: 0.50
- `breakout_steepness`: 11.0

### Normalization functions

**Breakout Age** — base curve (later than WR reflecting position reality, TE breakouts at 18-19 are vanishingly rare):

| Breakout Age | Normalized (base) |
|---|---|
| ≤20.0 | 1.00 |
| 21.0 | 0.80 |
| 22.0 | 0.50 |
| ≥23.0 | 0.15 |

**SL-019 modulator applied on top of base curve:**

```
modulated_breakout_age = base + (1.0 − base) × 0.35 × RAS_normalized
```

Modulation strength `0.35` is TE-specific. Worked examples for breakout age 22 (base 0.50):
- RAS 9.5 → modulated = 0.50 + 0.50 × 0.35 × 0.95 = **0.666**
- RAS 7.0 → modulated = 0.50 + 0.50 × 0.35 × 0.70 = **0.623**
- RAS 4.0 → modulated = 0.50 + 0.50 × 0.35 × 0.40 = **0.570**

Late-breakout penalty is softened by athletic profile. A 9.5-RAS senior-year breakout (Kittle archetype) gets ~33% credit on top of base; a 4.0-RAS senior-year breakout gets ~14% credit on top. Architecture rewards "athletic + late developer" without bailing out "non-athletic + late developer."

**School Tier** (template defaults):

| Tier | Normalized |
|---|---|
| Power Four | 1.00 |
| Group of Five | 0.70 |
| FCS | 0.40 |
| Non-FCS | 0.10 |

**College Usage Rate** — target share for TE-specific contexts:

| Target Share | Normalized |
|---|---|
| ≥22% | 1.00 |
| 15% | 0.50 |
| ≤8% | 0.10 |

Linear interpolation between defined points. Lower thresholds than WR (≥35% / 25% / ≤15%) — TE target share is structurally lower because they share targets with WRs, RBs, and other TEs.

**Age Trajectory** — base curve (mirrors WR, scaled to TE peak 29):

| Age | Normalized (base) |
|---|---|
| ≤25 | 1.00 |
| 26 | 0.85 |
| 27 | 0.70 |
| 28 | 0.55 |
| 29 (peak) | 0.50 |
| 30 | 0.35 |
| 31 | 0.20 |
| 32 | 0.10 |
| ≥33 | 0.00 |

**SL-019 modulator applied on top of base curve:**

```
modulated_age_trajectory = base + (1.0 − base) × 0.35 × RAS_normalized
```

Modulation strength `0.35` — same as breakout age. Same modulator strength on both ends of the career arc. Worked example for age 32 (base 0.10):
- RAS 9.5 → modulated = 0.10 + 0.90 × 0.35 × 0.95 = **0.399**
- RAS 5.4 → modulated = 0.10 + 0.90 × 0.35 × 0.54 = **0.270**

### Three-zone classification

Aligned to WR/RB for cross-position consistency:

- **Elite zone:** composite_input ≥ 0.80
- **Average zone:** 0.40 < composite_input < 0.80
- **Late zone:** composite_input ≤ 0.40

---

## 5. Verification Cases

S-curve formula (Shape B / sigmoid family):

```
output = 1 + cap × (2 × σ(steepness × (input − inflection)) − 1)
where σ(x) = 1 / (1 + e^(−x))
```

### Case 1 — Push: Brock Bowers

**Profile:**
- Age 23 (June 2026), Year 3 NFL (drafted 2024 #13 overall by LV)
- College: Georgia (Power Four — SEC)
- Breakout age: 19 (true freshman 2021, 882 yards as freshman TE — historic production)
- Junior-year college usage: ~18% of team passing yards (Georgia spreads the ball widely)
- RAS: ~9.45 *(estimated pending ras.football verification)*
- PFF: top-tier
- NFL touch share: very high — Raiders' alpha receiving option
- RSP / TDN / Sharp: consensus elite at entry

**Film component:**

```
(0.40 × 0.90) + (0.35 × 0.95) + (0.15 × 0.95) + (0.10 × 0.90)
= 0.360 + 0.3325 + 0.1425 + 0.090
= 0.925
```

S-curve(0.925, 0.50, 12.0, 0.05):
- arg = 12.0 × 0.425 = 5.100
- σ(5.100) ≈ 0.994
- output factor = 0.988
- film_raw = **1.049**
- film_effective = **1.049**

**RAS component (SL-018 Year 2+, High-tier):**

- RAS_normalized = 9.45 / 10 = 0.945
- S-curve(0.945, 0.50, 11.0, 0.08):
  - σ(11.0 × 0.445) = σ(4.895) ≈ 0.993
  - output factor = 0.985
  - RAS_raw = 1 + 0.08 × 0.985 = **1.079**
- Year 2+ → RAS_position_weight = 0.10
- RAS_effective = 1.0 + (1.079 − 1.0) × 1.0 × 0.10 = **1.008**

**Breakout component (SL-019 modulators applied where applicable):**

- Breakout age 19 → base 1.00 (already at max, modulator has no effect)
- School tier P4 → 1.00
- College usage 18% → 0.714 (linear interp: 15% = 0.50, 22% = 1.00)
- Age trajectory 23 → base 1.00 (≤25 = 1.00, modulator has no effect)

Composite:
```
(0.35 × 1.00) + (0.20 × 1.00) + (0.30 × 0.714) + (0.15 × 1.00)
= 0.350 + 0.200 + 0.214 + 0.150
= 0.914
```

Composite is in the **Elite zone**.

S-curve(0.914, 0.50, 11.0, 0.05):
- σ(11.0 × 0.414) = σ(4.554) ≈ 0.989
- output factor = 0.979
- breakout_raw = **1.049**
- breakout_effective = **1.049**

**Layer 4 combined:**

```
Layer_4_Output = 1.049 × 1.008 × 1.049 = 1.109
```

**Multiplier: ~1.11** — clear push case.

Bowers registers near-cap film, near-cap RAS that collapses to residual under SL-018, and Elite-zone breakout. SL-019 modulators don't activate meaningfully because his base curves are already at max — but they're available if he extends into late career.

---

### Case 2A — Pull: Tyler Higbee

**Profile:**
- Age 33 (June 2026), Year 11 NFL veteran (drafted 2016 R4 by LAR)
- College: Western Kentucky (Group of Five — Conference USA)
- Breakout age: 22 (senior year 2015 — 68 catches/803 yards on a pass-heavy WKU offense)
- Senior-year college usage: ~16% of team passing yards (WKU threw for 5000+ yards as a team)
- RAS: ~6.5 *(estimated — mid-tier athletic, not the elite profile SL-019 most rewards)*
- PFF: peak 2019-2021, declining recent years, depth role 2025
- NFL touch share: declining
- RSP / TDN / Sharp: Day 3 draft profile, never elite consensus

**Film component:**

```
(0.40 × 0.40) + (0.35 × 0.40) + (0.15 × 0.45) + (0.10 × 0.35)
= 0.160 + 0.140 + 0.0675 + 0.035
= 0.4025
```

S-curve(0.4025, 0.50, 12.0, 0.05):
- arg = 12.0 × (−0.0975) = −1.170
- σ(−1.170) ≈ 0.237
- output factor = −0.526
- film_raw = 1 + 0.05 × (−0.526) = **0.974**
- film_effective = **0.974**

**RAS component (SL-018 Year 2+, High-tier):**

- RAS_normalized = 6.5 / 10 = 0.650
- S-curve(0.650, 0.50, 11.0, 0.08):
  - σ(11.0 × 0.150) = σ(1.650) ≈ 0.839
  - output factor = 0.678
  - RAS_raw = 1 + 0.08 × 0.678 = **1.054**
- Year 2+ → RAS_position_weight = 0.10
- RAS_effective = 1.0 + (1.054 − 1.0) × 1.0 × 0.10 = **1.005**

**Breakout component (SL-019 modulators applied):**

- Breakout age 22 → base 0.50, modulated = 0.50 + 0.50 × 0.35 × 0.65 = **0.614**
- School tier G5 → 0.70
- College usage 16% → 0.571 (linear interp: 15% = 0.50, 22% = 1.00)
- Age trajectory 33 → base 0.00, modulated = 0.00 + 1.00 × 0.35 × 0.65 = **0.228**

Composite:
```
(0.35 × 0.614) + (0.20 × 0.70) + (0.30 × 0.571) + (0.15 × 0.228)
= 0.2149 + 0.1400 + 0.1713 + 0.0342
= 0.5604
```

Composite is in the **Average zone** (0.40 < x < 0.80).

S-curve(0.5604, 0.50, 11.0, 0.05):
- σ(11.0 × 0.0604) = σ(0.664) ≈ 0.660
- output factor = 0.320
- breakout_raw = 1 + 0.05 × 0.320 = **1.016**
- breakout_effective = **1.016**

**Layer 4 combined:**

```
Layer_4_Output = 0.974 × 1.005 × 1.016 = 0.994
```

**Multiplier: ~0.99** — pull case (mild).

Higbee's pull comes primarily from the film component (declining PFF/RSP signals) and is only partially offset by SL-019 modulator help. His mid-RAS (6.5) means modulators provide modest assistance but don't bail him out the way they would a Kittle profile. Static breakout signals (G5 school + 16% college usage + breakout age 22) anchor his breakout composite in Average zone rather than Late zone — but they're weak enough that the composite doesn't push above 0.80.

**Full Layer 3 × Layer 4 chain for Higbee:**

Layer 3 age_pull at 33 (4 years past peak 29) = 0.97⁴ = 0.885. SL-019 amplified buffer at RAS_normalized 0.65 = 0.30 × 0.65 = 19.5%. Buffered age_pull = 1.0 + (0.885 − 1.0)(1 − 0.195) = **0.907**.

```
Layer 3 × Layer 4 (Higbee) = 0.907 × 0.994 = 0.901
Layer 3 × Layer 4 (Bowers) = 1.000 × 1.109 = 1.109
```

A 23% spread.

---

### Case 2B — Comparative pull: Hunter Henry

Included as supplementary verification to demonstrate the **profile-strength differentiator** SL-019 produces between two aging veterans at the same position.

**Profile:**
- Age 31 (June 2026), Year 11 NFL veteran (drafted 2016 R2 by LAC)
- College: Arkansas (Power Four — SEC)
- Breakout age: 20 (junior-year peak — 51 catches/739 yards at Arkansas)
- Junior-year college usage: ~20% of team passing yards
- RAS: ~6.0 *(estimated — same mid-tier as Higbee)*
- PFF: declining but still TE1-tier production in NE
- Static profile is genuinely strong — P4 school, elite college usage, breakout at 20

**Film component:**

```
(0.40 × 0.50) + (0.35 × 0.45) + (0.15 × 0.55) + (0.10 × 0.40)
= 0.200 + 0.1575 + 0.0825 + 0.040
= 0.4800
```

S-curve(0.4800, 0.50, 12.0, 0.05):
- arg = 12.0 × (−0.020) = −0.240
- σ(−0.240) ≈ 0.440
- output factor = −0.120
- film_raw = **0.994**

**RAS component (SL-018 Year 2+):**

- RAS_normalized = 0.600 → RAS_raw 1.040 → RAS_effective = **1.004**

**Breakout component (SL-019 modulators applied):**

- Breakout age 20 → base 1.00 (already maxed, no modulator effect)
- School tier P4 → 1.00
- College usage 20% → 0.857
- Age trajectory 31 → base 0.20, modulated = 0.20 + 0.80 × 0.35 × 0.60 = **0.368**

Composite:
```
(0.35 × 1.00) + (0.20 × 1.00) + (0.30 × 0.857) + (0.15 × 0.368)
= 0.350 + 0.200 + 0.257 + 0.055
= 0.862
```

Composite is in the **Elite zone** (≥ 0.80) — driven by three static signals at or near max plus modulator-boosted age trajectory.

S-curve(0.862, 0.50, 11.0, 0.05):
- σ(11.0 × 0.362) = σ(3.982) ≈ 0.982
- output factor = 0.963
- breakout_raw = **1.048**

**Layer 4 combined:**

```
Layer_4_Output = 0.994 × 1.004 × 1.048 = 1.046
```

**Multiplier: ~1.05** — mild push, not pull.

**Full Layer 3 × Layer 4 chain for Henry:**

Layer 3 age_pull at 31 (2 years past peak) = 0.97² = 0.941. SL-019 buffer = 0.30 × 0.60 = 18%. Buffered age_pull = 1.0 + (0.941 − 1.0)(0.82) = **0.952**.

```
Layer 3 × Layer 4 (Henry) = 0.952 × 1.046 = 0.996
```

Henry ends up essentially neutral overall — which probably matches his actual dynasty value as a still-useful but unexciting veteran asset.

### Structural finding from Cases 2A and 2B

Higbee and Henry have nearly identical RAS (6.0 vs 6.5) and similar age (33 vs 31), but their Layer 4 outputs differ by 5 percentage points (0.99 vs 1.05). **The architecture is differentiating them by rookie-era profile strength, not by current production alone.**

- **Higbee (0.99 pull):** G5 school + late breakout age 22 + moderate college usage. Static breakout signals are weak. Even with SL-019 helping age trajectory, the composite stays in Average zone. Layer 4 pulls.
- **Henry (1.05 push):** P4 school + early breakout age 20 + elite college usage. Static breakout signals are strong and don't decay over career. SL-019 adds modulator help on top. Composite stays in Elite zone. Layer 4 pushes.

This is consistent with the WR/RB pattern (Lockett-pattern strong-profile veteran vs. Herbert-pattern weak-profile veteran). The Veteran Scouting Layer Extension (Deliverable 3, SL-008) is the proper venue to evaluate whether static breakout signals should decay or be re-weighted for veterans. For TE, with peak limit 29 and many veterans reaching 32-34, this question is most acute.

---

## 6. Open Questions Surfaced

WR + RB sessions surfaced SL-OQ-013 through SL-OQ-021. TE adds:

- **SL-OQ-022:** SL-019 modulation strength calibration. Current defaults are TE-specific (0.35 breakout age, 0.35 age trajectory, 0.30 × RAS_normalized Layer 3 buffer). These are starting positions. Empirical calibration requires multi-season athletic-TE longevity data — how does a 9.5-RAS TE actually age relative to a 6.0-RAS TE? CAL-018 backlog item.

- **SL-OQ-023:** Other positions that should apply SL-019. DE/Edge is the strongest candidate per the handoff (RAS highly predictive). LB and CB/S may benefit from partial application. Each position rubric specifies whether modulators apply and at what strengths. Resolves during DE rubric build (position 5).

- **SL-OQ-024:** SL-019 strength SYMMETRY between breakout-age (development) and age-trajectory (career length) modulators. Currently both at 0.35 for TE. Should they always be equal? Or should some positions have asymmetric values (e.g., RB might want stronger career-length modulator than development modulator)? Worth examining as more positions adopt SL-019.

- **SL-OQ-025 (from Gemini, renumbered):** Split PFF TE Grade into independent receiving and blocking vectors. Current rubric uses composite PFF Overall TE Grade weighted by snap distribution. TE roles are heterogeneous in a way other positions aren't — pure receivers, inline blockers, hybrid move-TEs. Composite grade washes out specialty signals: a great receiver who blocks poorly gets averaged down; an inline blocker who can't catch gets averaged up. For dynasty value specifically, receiving grade is what matters. Proposed v1.1+ refinement: PFF_receiving (~0.85 weight within PFF) + PFF_blocking (~0.15 weight). Not blocking for current lock.

**Calibration Backlog additions from TE build:**

- **CAL-018:** SL-019 modulation strengths empirical tune for TE — breakout age modulator (currently 0.35), age trajectory modulator (currently 0.35), Layer 3 buffer multiplier (currently 0.30 × RAS_normalized). Requires longitudinal athletic-TE longevity data.

- **CAL-019 (from Gemini, renumbered):** PFF EMA 0.12 blend rate validation for athletic pass-catcher early-career noise. Slow blend (α=0.12) was intended to smooth TE production noise but has a side effect — a Year 1 grade often dragged by run-blocking adjustment persists in the EMA at ~11% weight after one season, which can suppress a Year 2 athletic-TE breakout composite. **Linked to SL-OQ-025:** if PFF splits into receiving/blocking vectors, the receiving signal stops being contaminated by early-career blocking noise and CAL-019 partially auto-resolves. Resolve these together in the v1.1+ branch.

---

## 7. Position-Specific Notes

- TE is the first position to apply SL-019 RAS-modulator interactions. Mechanic is generalizable — codified for use by other position rubrics as needed.
- Three verification cases included rather than the standard two. TE is structurally weirder than WR/RB (longest development curve, most RAS-dependent longevity, blocking/receiving role split). The Higbee-Henry contrast demonstrates SL-019 differentiating by rookie-era profile strength, which is a finding worth documenting in-rubric rather than burying in roadmap notes.
- SL-004 is amended at session close: TE moves from Medium-tier RAS to High-tier RAS. Cross-tier consistency check needed during DE rubric build (DE may also need High-tier confirmation per handoff notes).
- All RAS values in verification cases are estimates pending ras.football verification.

---

## 8. Cross-Pollination Source

This rubric synthesized from:
- Universal Rubric Template v1.1 (structural skeleton — to be amended at session close to add SL-019 as available mechanic)
- Engine Specification v2.1 (Layer 4 mechanics — to be amended at session close to add SL-019)
- Gemini's TE rubric draft (sub-signal weight allocations including PFF/RSP inversion, base breakout-age curve shifted later for TE, RAS steepness 11.0, breakout sub-signal weights)
- Gemini's TE open questions (reconciled — PFF receiving/blocking split accepted as SL-OQ-025 for v1.1+ branch; PFF EMA noise calibration accepted as CAL-019, linked dependency with SL-OQ-025 documented)
- Christopher's SL-019 directive (TE moves to High-tier RAS, RAS modulates development curve AND career length)
- Sixth Madden mapping row added (Move TE / H-back versatility)
- SL-018 applied with High-tier 1.00 baseline (vs. Gemini's hedged Medium-tier 0.80)
- TDN reclassified from dynamic α=1.0 to STATIC (consistency with WR/RB fixes)
- RSP and Sharp EMA α refined to 0.50
- Age trajectory plateau (≥31=0.20) tightened to gradual decline through 0.00 at ≥33
- Third verification case (Hunter Henry) added at Christopher's direction to demonstrate profile-strength differentiation alongside the primary Higbee pull case

---

*Built by: Christopher Campbell + Claude (Anthropic)*

| Version | Date | Changes |
|---|---|---|
| 0.9 | June 2026 | Draft. Introduces SL-019 RAS-modulator interactions as generalized mechanic, applied first at TE. SL-004 amended to move TE from Medium to High RAS tier. Three verification cases (Bowers push, Higbee primary pull, Henry comparative). All math worked end-to-end. Pending Gemini's TE open questions for reconciliation pass before v1.0 lock. |
| 1.0 | June 2026 | Locked. Gemini's TE open questions reconciled. PFF receiving/blocking split accepted as SL-OQ-025 for v1.1+ branch. PFF EMA noise calibration accepted as CAL-019 with linkage to SL-OQ-025 documented — these resolve together in v1.1+. Renumbered Gemini's local SL-OQ-015 and CAL-017 to avoid collision with already-used global numbers. |
