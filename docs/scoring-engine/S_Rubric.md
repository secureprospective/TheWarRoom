# Legacy NFL Position Group Rubric — Safety (S)
**Version:** 1.0 — June 2026
**Status:** Locked. Fourth position to apply SL-019 RAS-modulator interactions (after TE, DE, CB), at CB-matching reduced modulator strengths (0.30 / 0.30 / 0.25). Second rubric to apply NGS coverage metrics as dedicated analytical anchor. Structurally parallel to CB — same peak limit, same tier, same gating criterion, same modulator strengths. Gemini's S open questions reconciled into SL-OQ-036 and CAL-029.
**Companion:** Engine_Specification.md Layer 4 is authoritative on mechanics. This rubric specifies position-specific values, the SL-019 mechanic at S, and the second NGS-anchor implementation.

---

## 1. Architectural Baseline

- **Layer 4 RAS Tier:** **High** (per SL-004). Athletic profile drives baseline multiplier at year 0; collapses to residual by year 2+ per SL-018.
- **Layer 3 Peak Limit:** 28 years (same as CB — reactive speed and range decay onset similar between coverage positions).
- **SL-019 application:** **YES, at CB-matching reduced strengths (0.30 / 0.30 / 0.25).** S meets the SL-OQ-027 gating rule criteria (High-tier RAS + position-specific athletic-profile-to-longevity predictive relationship — elite-RAS safeties maintain coverage range and tackle radius into their 30s; mid-RAS safeties lose range earlier). Matching CB strengths for structural consistency at parallel positions; calibration may diverge with live data.
- **Layer 2 Base Points drivers** (per Official Rulebook lines 88–117, MFL-sourced per DECISION-009):
  - **True Position split (LB/CB/S):** Tackle 1.5, Assist 1.0
  - **Pass-rush production (occasional on blitzes):** Sack 4.5, QB Hit 1, Tackle for Loss 2.5
  - **Coverage:** Pass Defensed 2.5, Interception Caught 5, Interception Return Yards 0.025/yd, Interception Return TD 6
  - **Turnover production (frequent at S — safeties strip often):** Forced Fumble 4, Opponent Fumble Recovery 3, Opponent Fumble Recovery Yards 0.025/yd
  - **Special teams adjacent:** Blocked Kick 7
  - **Game-state:** Safety 10
- **Data Parity Rule:** Missing/Unknown sub-signal data collapses component deviation to 0.0 via confidence weighting, returning neutral 1.00 fallback.

---

## 2. Film Component Configuration

**Cap (asymptote):** ±5% (standard).

### Sub-signal weights

| Source | Weight | Classification |
|---|---|---|
| PFF Safety Overall Grade (analytical anchor) | 0.35 | Analytical — self-regulates |
| NFL Next Gen Stats Coverage/Range Metrics (analytical anchor) | 0.30 | Analytical — self-regulates |
| The IDP Show (subjective anchor) | 0.10 | Subjective — Madden-regulated |
| The IDP Guru (analytical modifier) | 0.10 | Analytical — self-regulates |
| The Draft Network (pre-draft trait eval) | 0.08 | Subjective — Madden-regulated |
| Dynasty Nerds | 0.07 | Subjective — Madden-regulated |

Sums to 1.00. Structurally parallel to CB.

**NGS as dedicated anchor:** Second rubric to elevate NGS coverage metrics to dedicated 0.30-weight anchor status. Safety-specific NGS metrics differ from CB metrics — at S, the valuable signals are average depth of alignment (deep vs. box snap distribution), tackle radius / coverage range, closing speed on receivers, and coverage snap % vs. box snap %. These distinguish centerfielders from box safeties from hybrid players in a way no other sub-signal can. CAL-028 flags the specific S-NGS metric bundle definition for empirical calibration.

### Madden regulation parameters

- **Threshold:** 0.15 (normalized scale)
- **Blend scaling:** Linear gradient over 0.10 delta beyond threshold

### Madden attribute mapping

| Subjective Expert Claim | Madden Sub-Attribute / Composite | Formula |
|---|---|---|
| "Elite range / centerfield eraser" | Speed (SPD) + Acceleration (ACC) | Average(SPD, ACC) |
| "Enforcer / box run support" | Tackle (TAK) + Hit Power (HPW) + Strength (STR) | Average(TAK, HPW, STR) |
| "Zone coverage / over-the-top anticipation" | Zone Coverage (ZCV) + Play Recognition (PRC) | Average(ZCV, PRC) |
| "Diagnostic speed / downhill trigger" | Pursuit (PUR) + Awareness (AWR) | Average(PUR, AWR) |
| "Slot utility / man match capability" | Man Coverage (MCV) + Agility (AGI) | Average(MCV, AGI) |
| "Ball-hawk centerfielder / takeaway producer" | Catching (CTH) + Jumping (JMP) + Awareness (AWR) | Average(CTH, JMP, AWR) |

Six rows. Sixth added to cover the takeaway-machine ball-hawk archetype (Justin Reid, Antoine Winfield Jr., Justin Simmons in his prime) — distinct from the existing Zone Coverage row because ball-hawking emphasizes post-snap ball skills rather than pre-snap recognition. The full Madden bundle covers the full position spectrum from pure centerfielder (Range) to pure thumper (Enforcer) with three hybrid archetypes between them.

### Signal mechanics

- `film_position_weight`: 1.00
- `film_inflection`: 0.50
- `film_steepness`: 12.0

### EMA blend rates (dynamic sub-signals)

| Sub-signal | α | Classification |
|---|---|---|
| PFF | 0.18 | Dynamic — weekly grades with slow-moderate blend |
| NGS Coverage/Range Metrics | 0.20 | Dynamic — weekly tracking data with moderate blend |
| IDP Show | 0.30 | Dynamic — weekly subjective content, moderate blend |
| IDP Guru | 0.20 | Dynamic — weekly analytical content |
| TDN | N/A | **STATIC** — locked at rookie evaluation; no re-publication for veterans |
| Dynasty Nerds | 0.50 | Dynamic — annual-publication-rate content |
| Madden | 0.20 | Dynamic — multiple mid-season updates with moderate blend |

### Season transition behavior

CONTINUATION across all dynamic sub-signals.

### Sub-signal normalization (0.0–1.0 mapping)

- **PFF:** PFF Safety Overall Grade / 100
- **NGS Coverage/Range Metrics:** Composite z-score of (tackle radius percentile, coverage range index, closing speed on receivers, snap distribution between box and deep), mapped to position percentile, then 0–1 scaled. CAL-028 flags specific S-NGS bundle definition for empirical calibration.
- **IDP Show:** Tier ranking inverted to percentile within The IDP Show's S pool
- **IDP Guru:** Weekly S ranking inverted to percentile within IDP Guru's S pool
- **TDN:** TDN composite grade mapped to percentile within draft class S
- **Dynasty Nerds:** Dynasty S ranking inverted to percentile within Dynasty Nerds' S pool

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

- `RAS_inflection`: 0.50 (standard "universe of athletes" baseline)
- `RAS_steepness`: 10.0 (slightly less aggressive than CB's 11.0 — defensible because S athletic profiles are more continuous across the box-to-deep spectrum than CB's bimodal boundary/slot distribution)

### Normalization

`RAS_normalized = raw_RAS / 10.0`

Missing RAS → Layer 1 position-group mean imputation. Confidence flag set to Unknown.

### Late-career interaction (SL-018 + SL-019 reduced buffer)

Once player age > 28 (S peak limit), Layer 3 age_pull is RAS-buffered. **SL-019 amplifies the buffer at CB-matching strength (0.25× — same as CB):**

```
buffer_pct        = 0.25 × RAS_normalized   ← S-specific, matching CB
buffered_age_pull = 1.0 + (raw_age_pull − 1.0) × (1 − buffer_pct)
```

An S with RAS = 9.0 gets a 22.5% buffer against age decay. An S with RAS = 4.5 gets an 11.25% buffer. The differential is real but the absolute protection is smaller than at TE/DE (where the buffer is 0.30× rather than 0.25×) — reflecting that S longevity has more variance than TE/DE due to the box-vs-deep role split.

**SL-018 scope:** SL-019 modulator interactions are INDEPENDENT of SL-018 decay (same architecture as TE/DE/CB).

---

## 4. Breakout Component Configuration

**Cap (asymptote):** ±5%.

### Sub-signal weights

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.20 |
| School Tier | 0.25 |
| College Production Share (INT + Tackle market share) | 0.40 |
| Age Trajectory | 0.15 |

Sums to 1.00. Structurally parallel to CB but with **INT + Tackle** market share definition rather than CB's PD + INT — safeties are more tackle-involved than CBs (especially box safeties), and combined INT + Tackle market share is the cleanest single-metric capture of safety college production.

### Parameters

- `breakout_position_weight`: 1.00
- `breakout_inflection`: 0.50
- `breakout_steepness`: 11.0 (standard — slightly more aggressive than CB's 10.0, defensible because S college production data is cleaner than CB's)

### Normalization functions

**Breakout Age** — base curve (same as CB):

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

**School Tier** (template defaults):

| Tier | Normalized |
|---|---|
| Power Four | 1.00 |
| Group of Five | 0.70 |
| FCS | 0.40 |
| Non-FCS | 0.10 |

**College Production Share** (final-year INT + Tackle market share — average of player's share of team INTs and team tackles):

| Market Share | Normalized |
|---|---|
| ≥20% | 1.00 |
| 14% | 0.55 |
| ≤8% | 0.15 |

Linear interpolation. Threshold range slightly lower than CB (≥24%) because safeties have lower absolute market share ceilings (they share tackle volume with LBs and INT volume with CBs in any given defense).

**Age Trajectory** (current age relative to S peak limit of 28 — same as CB):

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

**SL-019 modulation applies (strength 0.30):**
```
age_trajectory_modulated = base + (1.0 − base) × 0.30 × RAS_normalized
```

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

### Case 1 — Push: Kyle Hamilton

**Profile:**
- Age 24 (born March 2001) — 4 years before peak limit
- College: Notre Dame (Power Four — Independent at the time, now ACC; treated as P4 historically)
- R1 #14 overall 2022
- Breakout age: 19 (true freshman impact at Notre Dame, immediate starter)
- Junior-year college production share: ~20% INT + Tackle market share
- RAS: ~9.0 *(estimated, pending ras.football verification — 4.59 40 at 6'4" 220lbs, exceptional size-adjusted athletic profile)*
- PFF: elite recent grades, ~90 (All-Pro consideration)
- Year 4 NFL veteran → SL-018 Year 2+ tier (RAS_position_weight = 0.10 at High)

**Film component:**

Sub-signal normalizations:
- PFF Safety Grade ~90 → 0.90
- NGS Coverage/Range Metrics (top-tier range, tackle radius, hybrid snap distribution) → 0.90
- IDP Show top-tier safety consensus → 0.92
- IDP Guru top-5 safety → 0.92
- TDN (static, 2022 R1 #14 elite grade) → 0.93
- Dynasty Nerds top dynasty rank → 0.93

Composite:
```
(0.35 × 0.90) + (0.30 × 0.90) + (0.10 × 0.92) + (0.10 × 0.92) + (0.08 × 0.93) + (0.07 × 0.93)
= 0.315 + 0.270 + 0.092 + 0.092 + 0.0744 + 0.0651
= 0.909
```

S-curve(0.909, 0.50, 12.0, 0.05):
- arg = 12.0 × 0.409 = 4.908
- σ(4.908) ≈ 0.9927
- output factor = 2 × 0.9927 − 1 = 0.9854
- film_raw = 1 + 0.05 × 0.9854 = **1.049**
- film_effective = **1.049**

**RAS component (SL-018 Year 2+):**

- RAS_normalized = 9.0 / 10 = 0.90
- S-curve(0.90, 0.50, 10.0, 0.08):
  - arg = 10.0 × 0.40 = 4.00
  - σ(4.00) ≈ 0.9820
  - output factor = 0.9640
  - RAS_raw = 1 + 0.08 × 0.9640 = **1.077**
- Year 2+ → RAS_position_weight = 0.10
- RAS_effective = 1.0 + (1.077 − 1.0) × 1.0 × 0.10 = **1.008**

**Breakout component (SL-019 modulators applied):**

- Breakout Age 19 → base 1.00 (already maxed)
- School Tier P4 → 1.00
- College Production Share 20% → 1.00 (at threshold)
- Age Trajectory 24 → 1.00 (≤24)

Composite:
```
(0.20 × 1.00) + (0.25 × 1.00) + (0.40 × 1.00) + (0.15 × 1.00)
= 1.000
```

All four sub-signals MAXED. SL-019 modulators have no effect on already-maxed bases.

S-curve(1.00, 0.50, 11.0, 0.05):
- arg = 11.0 × 0.50 = 5.50
- σ(5.50) ≈ 0.9959
- output factor = 0.9918
- breakout_raw = 1 + 0.05 × 0.9918 = **1.050**
- breakout_effective = **1.050**

**Layer 4 combined:**

```
Layer_4_Output = 1.049 × 1.008 × 1.050 = 1.110
```

**Multiplier: ~1.11** — clear push.

**Full Layer 3 × Layer 4 chain for Hamilton:**

Layer 3 age_pull at age 24 (pre-peak) = no decay yet, age_pull = 1.000.

```
Layer 3 × Layer 4 (Hamilton) = 1.000 × 1.110 = 1.110
```

---

### Case 2 — Pull-attempt: Tyrann Mathieu

**Profile:**
- Age 34 (born May 1992) — 6 years past peak limit
- College: LSU (Power Four — SEC)
- R3 #69 overall 2013 (off-field-driven draft fall from R1 talent grade)
- Breakout age: 19 (true freshman impact at LSU, Heisman-finalist sophomore year)
- Sophomore-year college production share: ~30% INT + Tackle market share (LSU's defensive playmaker — multiple takeaways + leading tackler from slot CB position)
- RAS: ~4.5 *(estimated, pending ras.football verification — undersized at 5'9" 186lbs, 4.50 40, size grade heavily depresses composite)*
- PFF: declining, ~65 recent grade (was 80+ in prime 2015–2020)
- Year 13 NFL veteran → SL-018 Year 2+ tier (RAS_position_weight = 0.10 at High)

**Film component:**

Sub-signal normalizations:
- PFF Safety Grade ~65 → 0.65
- NGS Coverage/Range Metrics (slot/box hybrid role, declining range) → 0.50
- IDP Show declining vet tier → 0.40
- IDP Guru mid-tier vet → 0.45
- TDN (static, 2013 R3 grade — off-field-discounted from a R1 talent grade, so static value sits at 0.60 rather than R3-typical 0.45) → 0.60
- Dynasty Nerds low dynasty rank → 0.20

Composite:
```
(0.35 × 0.65) + (0.30 × 0.50) + (0.10 × 0.40) + (0.10 × 0.45) + (0.08 × 0.60) + (0.07 × 0.20)
= 0.2275 + 0.150 + 0.040 + 0.045 + 0.048 + 0.014
= 0.5245
```

S-curve(0.5245, 0.50, 12.0, 0.05):
- arg = 12.0 × 0.0245 = 0.294
- σ(0.294) ≈ 0.5730
- output factor = 2 × 0.5730 − 1 = 0.1460
- film_raw = 1 + 0.05 × 0.1460 = **1.007**
- film_effective = **1.007**

Note: Film barely pushes despite genuine decline because TDN (static at 0.60 reflecting his R1-talent rookie eval grade, not his actual R3 draft slot) holds the composite up. Mathieu's rookie eval is a special case — his talent was R1, his draft slot was R3 due to off-field discount, so TDN evaluators graded him higher than his draft position would suggest.

**RAS component (SL-018 Year 2+):**

- RAS_normalized = 4.5 / 10 = 0.45
- S-curve(0.45, 0.50, 10.0, 0.08):
  - arg = 10.0 × (−0.05) = −0.50
  - σ(−0.50) ≈ 0.3775
  - output factor = 2 × 0.3775 − 1 = −0.2449
  - RAS_raw = 1 + 0.08 × (−0.2449) = **0.980**
- Year 2+ → RAS_position_weight = 0.10
- RAS_effective = 1.0 + (0.980 − 1.0) × 1.0 × 0.10 = **0.998**

**Breakout component (SL-019 modulators applied):**

- Breakout Age 19 → base 1.00 (already maxed)
- School Tier P4 → 1.00
- College Production Share 30% → 1.00 (above 20% threshold)
- Age Trajectory 34 → base 0.00 (≥32), modulated = 0.00 + 1.00 × 0.30 × 0.45 = **0.135**

Composite:
```
(0.20 × 1.00) + (0.25 × 1.00) + (0.40 × 1.00) + (0.15 × 0.135)
= 0.200 + 0.250 + 0.400 + 0.0203
= 0.870
```

Composite is in the **Elite zone** (≥ 0.80), just above the threshold. Three of four sub-signals MAXED. Note: SL-019 lifts the age trajectory base from 0.00 to 0.135 — substantially less lift than Peterson received at CB (0.279 with RAS 9.3) because Mathieu's lower RAS earns less longevity credit.

S-curve(0.870, 0.50, 11.0, 0.05):
- arg = 11.0 × 0.370 = 4.07
- σ(4.07) ≈ 0.9833
- output factor = 0.9666
- breakout_raw = 1 + 0.05 × 0.9666 = **1.048**
- breakout_effective = **1.048**

**Layer 4 combined:**

```
Layer_4_Output = 1.007 × 0.998 × 1.048 = 1.053
```

**Multiplier: ~1.05** — **PUSH despite age 34.**

**Structural finding (Lockett pattern at S, with SL-019 differentiating by RAS):** This is the eighth confirmed Lockett-pattern instance (WR, RB-with-Herbert-exception, TE, QB, DE, LB, CB, S). Mathieu's static breakout signals are all maxed; SL-019 lifts the age trajectory base from 0.00 to 0.135 (vs. Peterson's 0.279 lift at CB with RAS 9.3) — the mechanism correctly differentiates longevity protection by athletic profile. A low-RAS Lockett-pattern vet gets MUCH less Layer 4 buoyancy than a high-RAS one.

The pattern: at every position in the architecture, Layer 4 sits at 1.0 or above for veterans with strong rookie profiles because three or four of four breakout sub-signals are static. Layer 3 carries the aging.

**Full Layer 3 × Layer 4 chain for Mathieu:**

Layer 3 age_pull at age 34 = 0.97^6 ≈ 0.833. SL-019 reduced buffer (0.25 × RAS_normalized) = 0.25 × 0.45 = 0.1125 (11.25%). Buffered age_pull = 1.0 + (0.833 − 1.0) × (1 − 0.1125) = 1.0 + (−0.167)(0.8875) = **0.852**.

```
Layer 3 × Layer 4 (Mathieu)  = 0.852 × 1.053 = 0.897
Layer 3 × Layer 4 (Hamilton) = 1.000 × 1.110 = 1.110
```

A 21% spread on the full chain. SL-019 buffer pulled Mathieu's age_pull from 0.833 (without buffer) up to 0.852 — only ~2 percentage points of protection because his RAS is only 4.5. Compared to Peterson at CB (RAS 9.3, ~5pp protection), the architecture correctly differentiates: elite-RAS vets earn meaningful longevity credit; mid-RAS vets earn token credit.

This is the SL-019 mechanism working exactly as designed — athletic profile is rewarded for longevity, but the reward scales with the profile.

---

## 6. Open Questions Surfaced

Prior sessions surfaced SL-OQ-013 through SL-OQ-034 and CAL-015 through CAL-027. S adds:

- **SL-OQ-035:** Box-safety vs. deep-safety rubric branching. The current S rubric treats the position monolithically, applying identical sub-signal weights and curves to centerfielders (Earl Thomas archetype), box safeties (Kam Chancellor archetype), and hybrids (Kyle Hamilton, Tyrann Mathieu). Production profiles differ substantially: box safeties produce 1.5-2x more tackles but lower INT volume, deep safeties opposite. The current College Production Share definition (INT + Tackle market share, equal weight) treats them the same. Question: should S rubric split into S_BOX and S_DEEP sub-rubrics with role-specific College Production Share definitions and Madden mappings? Defer to empirical investigation — the simpler monolithic rubric is preferred for v1.0 if it doesn't systematically misrank archetypes.

- **SL-OQ-036 (from Gemini, renumbered from local SL-OQ-020):** Role-specific sub-signal weighting within a monolithic S rubric. This is the lighter alternative to SL-OQ-035 — instead of splitting S into S_BOX and S_DEEP rubrics, the question is whether sub-signal weights (specifically PFF Safety Overall vs. NGS Coverage/Range Metrics) should vary by role designation while keeping the rubric structure unified. A box safety's PFF Overall captures different work than a deep safety's; NGS coverage metrics are differentially relevant. Refinement candidate: a role-detection step that adjusts PFF weight between 0.30 and 0.40 (and NGS between 0.25 and 0.35) based on snap-distribution data from NGS itself. Links to SL-OQ-035 — these resolve together; the choice is whether to fully split rubrics or apply role-conditional weighting within one.

**Calibration Backlog additions from S build:**

- **CAL-028:** S NGS metric bundle definition (parallel to CB's CAL-026). The 0.30-weight S-NGS sub-signal is currently defined as composite of (tackle radius percentile, coverage range index, closing speed on receivers, snap distribution between box and deep). Empirical calibration needed to determine which S-specific NGS metrics best predict S future production, and how the bundle should weight metrics across box/deep/hybrid archetypes. Pairs with CAL-026 (CB NGS bundle) — same NGS data pipeline, position-specific metric selection.

- **CAL-029 (from Gemini, renumbered from local CAL-022 — global CAL-022 is already used for DE SL-019 modulator strengths):** College Production Share weighting and definition empirical calibration at S. Two concerns to track empirically: (a) does the 0.40-weight on College Production Share correctly surface elite slot-hybrid playmakers (Mathieu archetype — high INT + Tackle combined market share from a versatile college role), and (b) does the equal-weighted INT + Tackle market share definition over-weight box-tackle volume (which scales with team scheme and may inflate the metric for high-volume tacklers in poor pass defenses)? Refinement candidates: asymmetric weighting between INT and Tackle market shares; tackle-volume normalization by team total defensive snaps; or scheme-adjusted thresholds. Links to CAL-024 (LB college production share weighting) — both positions have the same Tackle market share problem.

---

## 7. Position-Specific Notes

- **NGS pattern extends to S:** Second position to apply NGS as dedicated 0.30-weight anchor (after CB). The pattern is established for coverage positions where tracking data is uniquely valuable. The pattern does NOT extend to LB, DE, or DT — interior and run-stop positions don't have a comparable tracking-data anchor. Decision documented here; if DT rubric reconsiders, it should not reach for NGS as a parallel.

- **SL-019 strengths held at CB values (0.30 / 0.30 / 0.25):** S is structurally parallel to CB — same peak, same tier, same gating criterion satisfaction. Holding modulator strengths at CB values for consistency. Live-data calibration may diverge if safety longevity-by-RAS shows different patterns than CB longevity-by-RAS (possible because safety roles are more continuous across the box-to-deep spectrum, while CB roles bifurcate sharply at boundary vs. slot).

- **RAS steepness 10.0 vs. CB's 11.0:** Slight reduction reflects continuous archetype distribution at S. CB athletic profiles are bimodal (elite-recovery boundary corners vs. technique-and-IQ slot corners), justifying sharper transition. S profiles span a continuous range from pure centerfielder to pure thumper without a clean bimodal split.

- **College Production Share thresholds lower at S (≥20% / 14% / 8%) vs. CB (≥24% / 16% / 8%):** Safeties share tackle volume with LBs and INT volume with CBs, so absolute market share ceilings are lower. Thresholds adjusted to recognize that 20% INT+Tackle market share at S represents the same caliber of college production as 24% PD+INT at CB.

- **EDGE classification reminder:** Per OQ-004 resolution, S rubric covers safeties only — hybrid safety/corner roles route through whichever rubric consensus role classification places them.

- **TDN special-case scoring (Mathieu):** The Mathieu verification case demonstrates a situation where TDN static value diverges from draft slot — a R1-talent prospect who fell to R3 due to off-field discount, where TDN evaluators graded him higher than his draft position would suggest. TDN normalization should reflect TDN's actual published grade rather than inferring from draft slot. Documented here; broader question of off-field-discount handling deferred to Veteran Scouting Layer Extension (Deliverable 3).

- All RAS values in verification cases are estimates pending ras.football verification.

---

## 8. Cross-Pollination Source

This rubric synthesized from:
- Universal Rubric Template v1.1 (structural skeleton)
- Engine Specification v2.1 (Layer 4 mechanics)
- CB Rubric v1.0 (structurally parallel template — peak limit, tier, NGS anchor pattern, SL-019 strengths)
- Gemini's S rubric draft (sub-signal weight allocations excluding NGS, Madden archetype framework with strong Slot Utility row, base curves for breakout age and college production share, S-curve parameters, RAS_inflection correctly set at 0.50)
- Gemini's S open questions (reconciled — SL-OQ-036 role-specific sub-signal weighting accepted as a lighter alternative to SL-OQ-035 full rubric branching, both resolve together; CAL-029 College Production Share weighting calibration renumbered from Gemini's local CAL-022 to avoid collision with global CAL-022 used for DE SL-019 strengths, accepted as an extension of the tackle-market-share calibration concern shared with LB's CAL-024)
- Christopher's calls (NGS anchor YES at 0.30; SL-019 strengths matching CB at 0.30 / 0.30 / 0.25; Kyle Hamilton push, Tyrann Mathieu pull-attempt)
- SL-018 applied (Gemini's draft predated this mechanic)
- SL-019 applied at CB-matching reduced strengths (Gemini's draft predated this mechanic — S is fourth instance after TE, DE, CB)
- NGS Coverage/Range Metrics added as 0.30-weight analytical anchor (Gemini's draft did not include NGS; CB rubric pattern carried over)
- Layer 2 drivers expanded to include QB Hits (1), Forced Fumble (4), Opponent Fumble Recovery (3) + return yards, INT Return Yards (0.025/yd), Blocked Kick (7) — Gemini's list was incomplete
- EMA blend rates fixed: IDP Show split from IDP Guru with α=0.30 / 0.20, TDN → STATIC at reduced weight 0.08, Dynasty Nerds split from TDN at α=0.50, NGS added at α=0.20
- Age trajectory curve tightened from Gemini's "≥28=0.10" plateau to gradual decline through 0.00 at age 32
- Sixth Madden mapping row added (Ball-hawk centerfielder / takeaway producer — Justin Reid / Antoine Winfield Jr. archetype)
- `[cite: N]` markers stripped throughout
- Verification cases replaced from generic placeholders with named real-player cases and full S-curve math

---

*Built by: Christopher Campbell + Claude (Anthropic)*

| Version | Date | Changes |
|---|---|---|
| 0.9 | June 2026 | Draft from cross-pollinated Gemini baseline + audit refinements. Fourth SL-019 instance after TE/DE/CB, at CB-matching reduced strengths (0.30 / 0.30 / 0.25). Second rubric to elevate NGS coverage/range metrics as dedicated 0.30-weight analytical anchor (Gemini did not include NGS — pattern carried from CB). SL-018 + SL-019 applied. Layer 2 drivers expanded with missing rulebook lines. Both verification cases worked end-to-end with full math (Hamilton ~1.11 push at age 24 pre-peak with all four breakout sub-signals maxed; Mathieu ~1.05 Layer 4 push despite age 34 — eighth confirmed Lockett-pattern instance, with SL-019 differentiating longevity protection by RAS — Mathieu's 0.135 age-trajectory lift vs. Peterson's 0.279 at CB demonstrates the architecture correctly scaling reward to athletic profile). Pending Gemini's S open questions for reconciliation pass before v1.0 lock. |
| 1.0 | June 2026 | Locked. Gemini's S open questions reconciled. SL-OQ-036 (role-specific sub-signal weighting within monolithic S rubric) renumbered from Gemini's local SL-OQ-020, accepted as the lighter alternative to SL-OQ-035 full-rubric-branching — both resolve together pending empirical investigation of whether box/deep/hybrid archetypes systematically diverge. CAL-029 (College Production Share weighting and definition calibration at S) renumbered from Gemini's local CAL-022 to avoid collision with the existing global CAL-022 (DE SL-019 modulator strengths) — accepted as an extension of the tackle-market-share calibration concern shared with LB's CAL-024. |
