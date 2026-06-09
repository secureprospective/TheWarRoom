# Legacy NFL Position Group Rubric — Running Back (RB)
**Version:** 1.0 — June 2026
**Status:** Locked. Validates Universal Rubric Template v1.1 against the most opportunity-dependent skill position with the shortest peak window. Gemini's RB open questions reconciled into SL-OQ-021 and CAL-017.
**Companion:** Engine_Specification.md Layer 4 is authoritative on mechanics. This rubric specifies position-specific values that fill in the Universal Rubric Template.

---

## 1. Architectural Baseline

- **Layer 4 RAS Tier:** Medium (per SL-004). RAS contributes meaningfully at year 0 but is not the elite predictor it is for WR. SL-018 collapses contribution to near-zero by year 2+.
- **Layer 3 Peak Limit:** 25 years. Shortest peak window of any skill position — Layer 3 decay kicks in at age 26.
- **Layer 2 Base Points drivers** (per Official Rulebook, MFL-sourced per DECISION-009):
  - Rushing: Attempts (0.15 each), Yards (0.1/yd), TDs (6.0), Long Rush bonuses (+1 at 20+ yds, +1 additional at 40+ yds, stackable)
  - Receiving: Receptions (1.0), Yards (0.1/yd), TDs (6.0)
  - 2PT Conversions: 2.0
  - Per-attempt scoring is a Legacy NFL custom — RB volume matters more than in standard PPR
- **Data Parity Rule:** Missing/Unknown sub-signal data collapses component deviation to 0.0 via confidence weighting, returning neutral 1.00 fallback.

---

## 2. Film Component Configuration

**Cap (asymptote):** ±5% (high end of SL-002 range).

### Sub-signal weights

| Source | Weight | Classification |
|---|---|---|
| Matt Waldman RSP (subjective anchor) | 0.35 | Subjective — Madden-regulated |
| PFF Rushing/Receiving Grade (analytical anchor) | 0.35 | Analytical — self-regulates |
| FantasyPros Snap Counts / Touch Share | 0.20 | Analytical — opportunity signal, self-regulates |
| The Draft Network | 0.05 | Subjective — Madden-regulated |
| Sharp Football Analysis | 0.05 | Subjective — Madden-regulated |

Sums to 1.00. Touch share elevated to 0.20 (vs. WR's 0.10 split among directional modifiers) reflects RB opportunity-dependence — workload is more dynasty-relevant for RB than for any other position.

### Madden regulation parameters

- **Threshold:** 0.15 (normalized scale). Same as WR baseline (CAL-008).
- **Blend scaling:** Linear gradient over 0.10 delta beyond threshold (CAL-009).

### Madden attribute mapping

| Subjective Expert Claim | Madden Sub-Attribute / Composite | Formula |
|---|---|---|
| "Elite speed / home-run threat" | Speed (SPD) + Acceleration (ACC) | Average(SPD, ACC) |
| "Power / contact balance" | Trucking (TRK) + Break Tackle (BTK) + Strength (STR) | Average(TRK, BTK, STR) |
| "Elusiveness / open-field agility" | Juke Move (JUK) + Spin Move (SPN) + Agility (AGI) | Average(JUK, SPN, AGI) |
| "Vision / creative instincts" | Ball Carrier Vision (BCV) | Direct |
| "Passing-down utility / hands" | Catching (CTH) + Pass Blocking (PBLK) | (0.7 × CTH) + (0.3 × PBLK) |

### Signal mechanics

- `film_position_weight`: 1.00
- `film_inflection`: 0.50
- `film_steepness`: 12.0

### EMA blend rates (dynamic sub-signals)

| Sub-signal | α | Classification |
|---|---|---|
| RSP | 0.50 | Dynamic — annual publication, half-weight blend |
| PFF | 0.20 | Dynamic — weekly updates, slightly faster blend than WR (0.15) reflecting RB usage volatility |
| Touch Share | 0.25 | Dynamic — week-to-week usage swings warrant faster blend |
| TDN | N/A | **STATIC** — locked at rookie evaluation |
| Sharp | 0.50 | Dynamic — annual rankings + occasional updates |
| Madden | 0.20 | Dynamic — multiple mid-season updates |

### Season transition behavior

CONTINUATION across all dynamic sub-signals.

### Sub-signal normalization (0.0–1.0 mapping)

- **RSP:** Percentile rank within annual RSP RB rankings
- **PFF:** PFF rushing + receiving composite grade / 100 (weighted by snap distribution)
- **Touch Share:** Touches per game / position-leader touches in the same week (rolling percentile)
- **TDN:** TDN composite grade mapped to percentile within draft class RBs
- **Sharp:** Sharp dynasty ranking inverted to percentile within Sharp's RB pool

---

## 3. RAS Component Configuration

**Cap (asymptote):** ±4% (Medium-tier per SL-004 — narrower than WR's ±8% reflecting RAS's reduced predictive weight at RB).

### Parameters

- `RAS_position_weight`: **0.60 at year 0 baseline (Medium-tier)**, modulated by SL-018 proportional scaling:

| NFL career stage | RAS_position_weight |
|---|---|
| Rookie / pre-NFL data | 0.60 |
| After 1 NFL season | 0.30 (50% of baseline) |
| Year 2+ | 0.06 (10% of baseline) |

- `RAS_inflection`: 0.50 (equivalent to raw RAS = 5.00)
- `RAS_steepness`: 8.0 (gentler than WR's 10.0 reflecting Medium-tier predictive weight)

### Normalization

`RAS_normalized = raw_RAS / 10.0`

Missing RAS → Layer 1 position-group mean imputation. Confidence flag set to Unknown.

### Late-career interaction (SL-018 buffer)

Once player age > 25 (Layer 3 peak limit for RB), Layer 3 age_pull is RAS-buffered:

```
buffer_pct        = 0.10 × RAS_normalized
buffered_age_pull = 1.0 + (raw_age_pull − 1.0) × (1 − buffer_pct)
```

Same mechanic as WR, different peak threshold. A RB with RAS = 9.0 gets a 9% buffer; RAS = 5.0 gets a 5% buffer.

### Note on SL-018 tier interaction

SL-018 was originally specified against WR's High-tier baseline of 1.00 ("100% → 50% → 10%"). For non-High-tier positions, the schedule scales proportionally to the position's year-0 baseline. Medium-tier RB baseline 0.60 → year 1 weight 0.30 (50%) → year 2+ weight 0.06 (10%). This interpretation is the cross-tier extension of SL-018 and gets documented in `Universal_Rubric_Template.md` at session close.

---

## 4. Breakout Component Configuration

**Cap (asymptote):** ±5%.

### Sub-signal weights

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.35 |
| School Tier | 0.20 |
| College Workload | 0.30 |
| Age Trajectory | 0.15 |

Sums to 1.00. Re-weighted from WR (0.40/0.25/0.20/0.15): 0.10 redistributed from breakout age + school tier into college workload, reflecting RB opportunity-dependence. Workload share at college is the strongest dynasty signal for RB transition to NFL.

### Parameters

- `breakout_position_weight`: 1.00
- `breakout_inflection`: 0.50
- `breakout_steepness`: 11.0

### Normalization functions

**Breakout Age** (age at first college season meeting the operational definition — see SL-OQ-015):

| Breakout Age | Normalized |
|---|---|
| ≤19.5 | 1.00 |
| 20.0 | 0.80 |
| 20.5 | 0.50 |
| ≥21.0 | 0.20 |

Linear interpolation between defined points. Aggressive 20-21 dropoff reflects RB-specific reality: short career window plus age-25 peak compounds the penalty for late college breakouts.

**School Tier:**

| Tier | Normalized |
|---|---|
| Power Four | 1.00 |
| Group of Five | 0.75 |
| FCS | 0.45 |
| Non-FCS | 0.15 |

Softer non-P4 penalty than WR (G5=0.75 vs WR's 0.70; FCS=0.45 vs WR's 0.40). Small-school RBs convert to NFL production more reliably than small-school WRs — workhorse usage at any FBS level is a meaningful signal.

**College Workload** (final-year touches as share of team RB touches — see SL-OQ-016):

| Workload Share | Normalized |
|---|---|
| ≥40% | 1.00 |
| 30% | 0.60 |
| ≤20% | 0.15 |

Linear interpolation. Higher thresholds than WR target share (≥35% / 25% / ≤15%) because workhorse RBs claim a much higher share of team rushing+RB-targets than alpha WRs claim of team targets.

**Age Trajectory** (current age relative to RB peak limit of 25):

| Age | Normalized |
|---|---|
| ≤21 | 1.00 |
| 22 | 0.85 |
| 23 | 0.75 |
| 24 | 0.60 |
| 25 (peak) | 0.50 |
| 26 | 0.30 |
| 27 | 0.15 |
| 28 | 0.05 |
| ≥29 | 0.00 |

Mirrors the WR taper structure, scaled to RB peak = 25. Linear interpolation between defined points.

### Three-zone classification

Aligned to WR for cross-position consistency:

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

### Case 1 — Push: Bijan Robinson

**Profile:**
- Age 24 (June 2026), Year 4 NFL veteran (drafted 2023 #8 overall by ATL)
- College: Texas (Power Four — Big 12 → SEC)
- Breakout age: 19 (true freshman 2020, 703 yards as ATX's lead back)
- Junior-year workload: ~53% of team rushing attempts (258 of 487) — historic workhorse usage
- RAS: ~9.55 *(estimated pending ras.football verification)*
- PFF career grades: top-tier
- NFL touch share: high — Atlanta's workhorse lead back
- RSP / TDN / Sharp: consensus elite at entry; production has confirmed

**Film component:**

Composite weighted sub-signals (all near upper-end):
```
(0.35 × 0.95) + (0.35 × 0.92) + (0.20 × 0.85) + (0.05 × 0.90) + (0.05 × 0.90)
= 0.3325 + 0.3220 + 0.1700 + 0.0450 + 0.0450
= 0.9145
```

S-curve(0.9145, 0.50, 12.0, 0.05):
- arg = 12.0 × 0.4145 = 4.974
- σ(4.974) ≈ 0.9931
- output factor = 0.9862
- film_raw = 1 + 0.05 × 0.9862 = **1.049**
- film_effective = **1.049**

**RAS component (SL-018 applied):**

- RAS_normalized = 9.55 / 10 = 0.955
- S-curve(0.955, 0.50, 8.0, 0.04):
  - σ(8.0 × 0.455) = σ(3.64) ≈ 0.9742
  - output factor = 0.9484
  - RAS_raw = 1 + 0.04 × 0.9484 = **1.038**
- Year 2+ → RAS_position_weight = 0.06 (Medium-tier proportional)
- RAS_effective = 1.0 + (1.038 − 1.0) × 1.0 × 0.06 = **1.002**

**Breakout component:**

- Breakout age 19 → 1.00
- School tier P4 → 1.00
- College workload 53% → 1.00 (≥40% threshold)
- Age trajectory 24 → 0.60

Composite:
```
(0.35 × 1.00) + (0.20 × 1.00) + (0.30 × 1.00) + (0.15 × 0.60)
= 0.350 + 0.200 + 0.300 + 0.090
= 0.940
```

Composite is in the **Elite zone** (≥ 0.80).

S-curve(0.940, 0.50, 11.0, 0.05):
- arg = 11.0 × 0.440 = 4.840
- σ(4.840) ≈ 0.9922
- output factor = 0.9843
- breakout_raw = 1 + 0.05 × 0.9843 = **1.049**
- breakout_effective = **1.049**

**Layer 4 combined:**

```
Layer_4_Output = 1.049 × 1.002 × 1.049 = 1.103
```

**Multiplier: ~1.10** — clear push case.

Bijan registers near-cap film (elite consensus + high touch share), high RAS that collapses to residual under SL-018 year-2+ weighting, and Elite-zone breakout (true-freshman breakout, P4 school, historic college workload). Layer 3 age_pull is 1.0 at age 24 (one year shy of peak). Full Layer 3 × Layer 4 ≈ 1.10.

---

### Case 2 — Pull: Khalil Herbert

**Profile:**
- Age 28 (June 2026), Year 6 NFL veteran (drafted 2021 R6 by CHI, currently depth role)
- College: Kansas (Big 12 — P4) 2017-2019, transferred to Virginia Tech (ACC — P4) for 2020 senior year
- Breakout age: 22 (Virginia Tech 2020 — 1183 yards graduate-transfer season). Using strict definition per SL-OQ-015; loose definition would mark him at age 19 freshman year at Kansas (663 yards on a bad team), which would change the math.
- Senior-year workload at VT: ~38% of team rushing attempts (155 of ~410)
- RAS: ~8.50 *(estimated — Herbert had a strong athletic profile despite Day 3 draft selection)*
- PFF: decent early career grades, declining 2023+ in depth role
- NFL touch share: low — buried on depth chart
- RSP / TDN / Sharp: Day 3 draft profile, never elite consensus

**Film component:**

Composite weighted sub-signals (mediocre profile, declining production):
```
(0.35 × 0.40) + (0.35 × 0.45) + (0.20 × 0.25) + (0.05 × 0.35) + (0.05 × 0.35)
= 0.1400 + 0.1575 + 0.0500 + 0.0175 + 0.0175
= 0.3825
```

S-curve(0.3825, 0.50, 12.0, 0.05):
- arg = 12.0 × (−0.1175) = −1.410
- σ(−1.410) ≈ 0.196
- output factor = 2 × 0.196 − 1 = −0.609
- film_raw = 1 + 0.05 × (−0.609) = **0.970**
- film_effective = **0.970**

**RAS component (SL-018 applied):**

- RAS_normalized = 8.50 / 10 = 0.850
- S-curve(0.850, 0.50, 8.0, 0.04):
  - σ(8.0 × 0.350) = σ(2.80) ≈ 0.9427
  - output factor = 0.8853
  - RAS_raw = 1 + 0.04 × 0.8853 = **1.035**
- Year 2+ → RAS_position_weight = 0.06
- RAS_effective = 1.0 + (1.035 − 1.0) × 1.0 × 0.06 = **1.002**

**Breakout component (using strict breakout-age definition):**

- Breakout age 22 → 0.20 (clamped at ≥21.0 floor per the RB curve)
- School tier P4 (Virginia Tech, ACC) → 1.00
- College workload 38% → 0.92 (linear interp: 30% = 0.60, 40% = 1.00)
- Age trajectory 28 → 0.05

Composite:
```
(0.35 × 0.20) + (0.20 × 1.00) + (0.30 × 0.92) + (0.15 × 0.05)
= 0.0700 + 0.2000 + 0.2760 + 0.0075
= 0.5535
```

Composite is in the **Average zone** (0.40 < x < 0.80).

S-curve(0.5535, 0.50, 11.0, 0.05):
- arg = 11.0 × 0.0535 = 0.589
- σ(0.589) ≈ 0.643
- output factor = 0.286
- breakout_raw = 1 + 0.05 × 0.286 = **1.014**
- breakout_effective = **1.014**

**Layer 4 combined:**

```
Layer_4_Output = 0.970 × 1.002 × 1.014 = 0.986
```

**Multiplier: ~0.99** — clear pull case.

**Structural finding — distinct from Lockett:**

Herbert's Layer 4 pulls genuinely below 1.0 (~0.99), unlike Lockett (~1.01). The difference shows the architecture working correctly. Lockett had a strong rookie scouting profile (high-WR draft pick, elite college usage) — his static breakout signals stayed positive, and his film component was only mildly declining. Herbert had a mediocre profile entering the league (Day 3 pick, late breakout, depth-back role from day one) — his film component pulls harder, and his breakout component is barely positive because the late breakout age (0.20) hammers what would otherwise be a decent composite.

**Layer 4 CAN pull below 1.0 when the underlying scouting profile is genuinely thin.** It just doesn't pull declining-veteran cases like Lockett that way because Lockett's draft-era development signals were strong. The Veteran Scouting Layer Extension (Deliverable 3) is the proper venue to evaluate whether veteran rubrics should reweight static signals to handle aging more aggressively at Layer 4 — for now this is correct architecture.

**Full Layer 3 × Layer 4 chain for Herbert:**

Layer 3 age_pull at age 28 (3 years past peak 25) = 0.97^3 ≈ 0.913. SL-018 buffer at his RAS_normalized of 0.85 = 0.10 × 0.85 = 8.5%. Buffered age_pull = 1.0 + (0.913 − 1.0) × (1 − 0.085) = 1.0 + (−0.087)(0.915) = **0.920**.

```
Layer 3 × Layer 4 (Herbert)  = 0.920 × 0.986 = 0.907
Layer 3 × Layer 4 (Bijan)    = 1.000 × 1.103 = 1.103
```

A 22% spread, similar to the WR Jefferson-vs-Lockett demonstration. Layer 3 (aging) and Layer 4 (scouting/development signal) doing different work, combining to differentiate elite-in-peak from declining-depth.

---

## 6. Open Questions Surfaced

The following items land in `Roadmap_and_Open_Questions.md` at session close. WR session already surfaced SL-OQ-013 through SL-OQ-017. RB adds:

- **SL-OQ-018:** College workload operational definition for RB. Three candidate definitions: (a) touches (rush + receptions) as share of team rushes + RB targets, (b) touches as share of team plays, (c) carries as share of team carries (ignoring receiving role). Currently defaulted to definition (a) as the most production-stable signal. Resolves once pipeline is live with multi-season validation data.

- **SL-OQ-019:** SL-018 tier-interaction documentation. Proportional scaling across RAS tiers (1.00 → 0.50 → 0.10 for High-tier; 0.60 → 0.30 → 0.06 for Medium-tier; 0.20 → 0.10 → 0.02 for Low-tier) is the cross-tier extension of SL-018. Needs explicit documentation in `Universal_Rubric_Template.md` RAS Component spec and `Engine_Specification.md` Layer 4 RAS section. Not a flaw — a structural clarification.

- **SL-OQ-020:** Touch share conceptual placement. Currently a film sub-signal at 0.20 weight. Alternative architecture: touch share modulates PFF/RSP confidence (low touches = small sample = low PFF/RSP confidence) rather than carrying standalone weight. Current placement defensible because touch share carries dynasty-relevant signal beyond just modulating other signals (high-volume RBs are likely to remain high-volume). Worth revisiting after multi-season calibration data is in. CAL-016 backlog candidate.

- **SL-OQ-021 (from Gemini, repositioned):** Injury-shortened touch share EMA handling. A primary back who plays 8 games at 80% snap share and misses 8 to injury would have his touch share averaged down by DNP weeks in a naïve EMA blend, misrepresenting his actual usage when active. Resolution: touch share computed per-active-game (touches / games played) rather than per-week (touches / total weeks). Requires the data ingestion layer to track snaps-available alongside touches. **This is a Layer 1 / data-hygiene specification, not a Layer 4 rubric question** — the Layer 4 component ingests whatever the data layer provides. Gets handed off to the data ingestion layer spec.

**Calibration Backlog additions from RB build:**

- **CAL-016:** Touch share standalone weight vs. confidence-modifier architecture — empirical comparison once 3+ seasons of RB pipeline data exist. Architecture-level decision (parent of CAL-017).

- **CAL-017 (from Gemini):** Committee-scheme touch share weight empirical calibration. Once architecture is locked (CAL-016), tune the 0.20 weight against actual backfield utilization splits across standard committee schemes. Parameter-level tune (child of CAL-016).

---

## 7. Position-Specific Notes

- RB is the first non-High-tier position rubric. SL-018 proportional scaling is the cross-tier extension validated here.
- RB validates that Layer 4 can pull below 1.0 (Herbert ~0.99) when the underlying scouting profile is genuinely mediocre — distinct from veteran-with-strong-draft-profile cases (Lockett ~1.01) where Layer 4 stays near neutral.
- RB breakout age curve is more aggressive than WR (steep 20-21 dropoff) reflecting compounding penalty: short career window + early peak (25) means late college breakout = compressed NFL runway.
- College workload reweighted to 0.30 in breakout component (vs. WR's 0.20). For RB, "what did he do at college" matters more than "where did he go" because workhorse usage signals pro-readiness regardless of conference.
- All RAS values in verification cases are estimates pending ras.football verification.

---

## 8. Cross-Pollination Source

This rubric synthesized from:
- Universal Rubric Template v1.1 (structural skeleton)
- Engine Specification v2.1 (Layer 4 mechanics)
- Gemini's RB rubric draft (sub-signal weight allocations, Madden mapping, breakout curve structure, three-zone framework)
- Gemini's RB open questions (reconciled — injury-shortened touch share repositioned to SL-OQ-021 as a Layer 1 / data-hygiene specification; committee-scheme touch share calibration accepted as CAL-017 child of architectural CAL-016)
- SL-018 (RAS time-decay) applied with proportional Medium-tier scaling
- TDN reclassified from dynamic α=1.0 to STATIC (consistency with WR fix)
- RSP and Sharp EMA α refined to 0.50 (consistency with WR fix)
- Age trajectory curve tightened past peak (Gemini's plateau at 0.10 expanded to gradual decline through 0.00)
- Three-zone boundaries aligned to WR (0.80 / 0.40)
- Layer 2 rushing-attempt scoring (0.15/attempt) verified against `Official_Rulebook.md`

---

*Built by: Christopher Campbell + Claude (Anthropic)*

| Version | Date | Changes |
|---|---|---|
| 0.9 | June 2026 | Draft from cross-pollinated Gemini baseline + audit refinements. SL-018 proportional Medium-tier scaling applied. Both verification cases worked end-to-end with full math. Pending Gemini's RB open questions for reconciliation pass before v1.0 lock. |
| 1.0 | June 2026 | Locked. Gemini's RB open questions reconciled. Injury-shortened touch share question repositioned from Layer 4 rubric concern to Layer 1 / data-ingestion specification (SL-OQ-021). Committee-scheme touch share calibration accepted as CAL-017, distinct from the architectural CAL-016 (standalone vs. confidence-modifier). Renumbered Gemini's local SL-OQ-014 and CAL-016 to avoid collision with already-used global numbers from WR session. |
