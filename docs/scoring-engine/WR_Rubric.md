# Legacy NFL Position Group Rubric — Wide Receiver (WR)
**Version:** 1.0 — June 2026
**Status:** Locked. Validates Universal Rubric Template v1.1 against the data-richest offensive position. Gemini's WR open questions reconciled into SL-OQ-013 and SL-OQ-014.
**Companion:** Engine_Specification.md Layer 4 is authoritative on mechanics. This rubric specifies position-specific values that fill in the Universal Rubric Template.

---

## 1. Architectural Baseline

- **Layer 4 RAS Tier:** High (per SL-004). RAS directly shapes baseline multiplier at year 0; collapses to residual by year 2+ per SL-018.
- **Layer 3 Peak Limit:** 29 years.
- **Layer 2 Base Points drivers:** Receptions (1.0), Rec Yards (0.1/yd), Rec TDs (6.0), stackable long-play threshold bonuses (+1 at 20+ yards, +1 additional at 40+ yards), 2PT conversions (2.0). Per DECISION-009 these values are MFL-sourced.
- **Data Parity Rule:** Missing/Unknown sub-signal data collapses component deviation to 0.0 via confidence weighting, returning neutral 1.00 fallback.

---

## 2. Film Component Configuration

**Cap (asymptote):** ±5% (high end of SL-002 range).

### Sub-signal weights

| Source | Weight | Classification |
|---|---|---|
| Matt Waldman RSP (subjective anchor) | 0.40 | Subjective — Madden-regulated |
| PFF Receiver Grade (analytical anchor) | 0.40 | Analytical — self-regulates |
| The Draft Network | 0.10 | Subjective — Madden-regulated |
| Sharp Football Analysis | 0.10 | Subjective — Madden-regulated |

Sums to 1.00.

### Madden regulation parameters

- **Threshold:** 0.15 (normalized scale). Disagreement below this holds the subjective claim weight at full strength.
- **Blend scaling:** Linear gradient over 0.10 delta beyond threshold. At threshold + 0.10, full blend toward Madden-implied weight.

Both values are defaults pending calibration (CAL-008, CAL-009).

### Madden attribute mapping

| Subjective Expert Claim | Madden Sub-Attribute / Composite | Formula |
|---|---|---|
| "Elite speed / vertical threat" | Speed (SPD) + Acceleration (ACC) | Average(SPD, ACC) |
| "Good hands / contested catch" | Catching (CTH) + Catch in Traffic (CIT) + Spectacular Catch (SPC) | Average(CTH, CIT, SPC) |
| "Route technician / separation" | Short Route Running (SRR) + Medium Route Running (MRR) + Deep Route Running (DRR) | Average(SRR, MRR, DRR) |
| "Press win / physical release" | Release (RLS) + Strength (STR) | (0.8 × RLS) + (0.2 × STR) |
| "YAC threat / elusiveness" | Break Tackle (BTK) + Juke Move (JUK) + Agility (AGI) | Average(BTK, JUK, AGI) |
| "High-point / contested-catch specialist" | Jumping (JMP) + Spectacular Catch (SPC) + Catch in Traffic (CIT) | Average(JMP, SPC, CIT) |
| "Power-after-catch" | Trucking (TRK) + Break Tackle (BTK) + Strength (STR) | Average(TRK, BTK, STR) |

### Signal mechanics

- `film_position_weight`: 1.00
- `film_inflection`: 0.50
- `film_steepness`: 12.0

### EMA blend rates (dynamic sub-signals)

| Sub-signal | α | Classification |
|---|---|---|
| RSP | 0.50 | Dynamic — annual publication, half-weight blend preserves prior interpretation for veterans |
| PFF | 0.15 | Dynamic — weekly updates with slow blend |
| TDN | N/A | **STATIC** — locked at rookie evaluation; no re-publication for veterans |
| Sharp | 0.50 | Dynamic — annual rankings + occasional in-season analysis |
| Madden | 0.20 | Dynamic — multiple mid-season updates with moderate blend |

### Season transition behavior

CONTINUATION across all dynamic sub-signals. Prior-season values blend with new-season first observation per the standard EMA formula.

### Sub-signal normalization (0.0–1.0 mapping)

- **RSP:** Percentile rank within the annual RSP WR rankings
- **PFF:** PFF receiver grade / 100 (grades are already 0–100)
- **TDN:** TDN composite grade mapped to percentile within draft class WRs
- **Sharp:** Sharp dynasty ranking inverted to percentile within Sharp's WR pool (rank 1 = 1.0)

---

## 3. RAS Component Configuration

**Cap (asymptote):** ±8% (High-tier per SL-004 — wider cap reflects RAS predictive weight at WR).

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

Missing RAS → Layer 1 position-group mean imputation (~6.5 default for WR pending real data). Confidence flag set to Unknown.

### Late-career interaction (SL-018 buffer)

Once player age > 29 (Layer 3 peak limit), Layer 3 age_pull is RAS-buffered:

```
buffer_pct       = 0.10 × RAS_normalized
buffered_age_pull = 1.0 + (raw_age_pull − 1.0) × (1 − buffer_pct)
```

A WR with RAS = 9.0 gets a 9% buffer against age decay. A WR with RAS = 5.0 gets a 5% buffer. This is a cross-layer mechanic (Layer 4 informs Layer 3 output) — implementation lives in the Layer 3 / Layer 4 boundary code, not as a rubric override.

---

## 4. Breakout Component Configuration

**Cap (asymptote):** ±5%.

### Sub-signal weights

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.40 |
| School Tier | 0.25 |
| College Usage Rate | 0.20 |
| Age Trajectory | 0.15 |

Sums to 1.00.

### Parameters

- `breakout_position_weight`: 1.00
- `breakout_inflection`: 0.50
- `breakout_steepness`: 11.0

### Normalization functions

**Breakout Age** (age at first college season with ≥20% team receiving production — see WR-SL-OQ-001):

| Breakout Age | Normalized |
|---|---|
| ≤19.0 | 1.00 |
| 20.0 | 0.75 |
| 21.0 | 0.40 |
| ≥22.0 | 0.10 |

Linear interpolation between defined points.

**School Tier** (template defaults):

| Tier | Normalized |
|---|---|
| Power Four | 1.00 |
| Group of Five | 0.70 |
| FCS | 0.40 |
| Non-FCS | 0.10 |

**College Usage Rate** (final-year target share — see WR-SL-OQ-002):

| Target Share | Normalized |
|---|---|
| ≥35% | 1.00 |
| 25% | 0.50 |
| ≤15% | 0.10 |

Linear interpolation between defined points.

**Age Trajectory** (current age relative to WR peak limit of 29):

| Age | Normalized |
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

### Three-zone classification

Thresholds on the composite input (weighted sum before S-curve):

- **Elite zone:** composite_input ≥ 0.80
- **Average zone:** 0.40 < composite_input < 0.80
- **Late zone:** composite_input ≤ 0.40

Zones are surfaced downstream for context. The multiplier value is the S-curve output regardless of zone.

---

## 5. Verification Cases

S-curve formula used throughout (Shape B / sigmoid family):

```
output = 1 + cap × (2 × σ(steepness × (input − inflection)) − 1)
where σ(x) = 1 / (1 + e^(−x))
```

### Case 1 — Push: Justin Jefferson

**Profile:**
- Age 27 (June 2026), Year 7 NFL veteran (drafted 2020)
- College: LSU (Power Four)
- Breakout age: 19 (true sophomore 2019 — Heisman-finalist season)
- Senior-year target share: ~32%
- RAS: ~9.40 *(estimated pending ras.football verification)*
- PFF career grades: top-tier (~90s)
- RSP / TDN / Sharp: consensus elite at entry; production has confirmed

**Film component:**

Composite weighted sub-signals (all near upper-end normalization):
```
(0.40 × 0.95) + (0.40 × 0.92) + (0.10 × 0.90) + (0.10 × 0.90)
= 0.380 + 0.368 + 0.090 + 0.090
= 0.928
```

S-curve(0.928, 0.50, 12.0, 0.05):
- arg = 12.0 × (0.928 − 0.50) = 5.136
- σ(5.136) ≈ 0.9941
- output factor = 2 × 0.9941 − 1 = 0.9883
- film_raw = 1 + 0.05 × 0.9883 = **1.049**
- film_effective = 1.0 + (1.049 − 1.0) × 1.0 × 1.0 = **1.049**

**RAS component (SL-018 applied):**

- RAS_normalized = 9.40 / 10 = 0.940
- S-curve(0.940, 0.50, 10.0, 0.08):
  - σ(10 × 0.44) = σ(4.40) ≈ 0.9879
  - output factor = 0.9759
  - RAS_raw = 1 + 0.08 × 0.9759 = **1.078**
- Year 2+ → RAS_position_weight = 0.10
- RAS_effective = 1.0 + (1.078 − 1.0) × 1.0 × 0.10 = **1.008**

**Breakout component:**

- Breakout age 19 → 1.00
- School tier P4 → 1.00
- College usage ~32% → 0.85 (linear interp: 25% = 0.50, 35% = 1.00)
- Age trajectory 27 → 0.70

Composite:
```
(0.40 × 1.00) + (0.25 × 1.00) + (0.20 × 0.85) + (0.15 × 0.70)
= 0.400 + 0.250 + 0.170 + 0.105
= 0.925
```

Composite is in the **Elite zone** (≥ 0.80).

S-curve(0.925, 0.50, 11.0, 0.05):
- arg = 11.0 × 0.425 = 4.675
- σ(4.675) ≈ 0.9907
- output factor = 0.9814
- breakout_raw = 1 + 0.05 × 0.9814 = **1.049**
- breakout_effective = **1.049**

**Layer 4 combined:**

```
Layer_4_Output = 1.049 × 1.008 × 1.049 = 1.109
```

**Multiplier: ~1.11** — clear push case.

Jefferson registers near-top film, near-cap RAS that is collapsed to residual by SL-018 year-2+ weighting, and Elite-zone breakout. Layer 3 age_pull is essentially 1.0 at age 27. Layer 5 cap efficiency depends on contract terms.

---

### Case 2 — Pull: Tyler Lockett

**Profile:**
- Age 33 (June 2026), Year 12 NFL veteran (drafted 2015 R3 by Seattle)
- College: Kansas State (Power Four — Big 12)
- Breakout age: 21 (junior year 2013 — first season as lead receiver)
- Junior-year target share: ~30%
- RAS: ~8.50 *(estimated pending ras.football verification)*
- PFF: declining recent grades, ~70s in 2025
- RSP / TDN / Sharp: subjective takes were moderate at entry; recent assessments emphasize decline

**Film component:**

Composite weighted sub-signals (mixed, declining):
```
(0.40 × 0.50) + (0.40 × 0.40) + (0.10 × 0.50) + (0.10 × 0.40)
= 0.200 + 0.160 + 0.050 + 0.040
= 0.450
```

S-curve(0.450, 0.50, 12.0, 0.05):
- arg = 12.0 × (−0.05) = −0.60
- σ(−0.60) ≈ 0.354
- output factor = 2 × 0.354 − 1 = −0.292
- film_raw = 1 + 0.05 × (−0.292) = **0.985**
- film_effective = **0.985**

**RAS component (SL-018 applied):**

- RAS_normalized = 8.50 / 10 = 0.850
- S-curve(0.850, 0.50, 10.0, 0.08):
  - σ(10 × 0.35) = σ(3.50) ≈ 0.9707
  - output factor = 0.9414
  - RAS_raw = 1 + 0.08 × 0.9414 = **1.075**
- Year 2+ → RAS_position_weight = 0.10
- RAS_effective = 1.0 + (1.075 − 1.0) × 1.0 × 0.10 = **1.008**

**Breakout component:**

- Breakout age 21 → 0.40
- School tier P4 → 1.00
- College usage ~30% → 0.75
- Age trajectory 33 → 0.00

Composite:
```
(0.40 × 0.40) + (0.25 × 1.00) + (0.20 × 0.75) + (0.15 × 0.00)
= 0.160 + 0.250 + 0.150 + 0.000
= 0.560
```

Composite is in the **Average zone** (0.40 < x < 0.80).

S-curve(0.560, 0.50, 11.0, 0.05):
- arg = 11.0 × 0.060 = 0.660
- σ(0.660) ≈ 0.659
- output factor = 0.319
- breakout_raw = 1 + 0.05 × 0.319 = **1.016**
- breakout_effective = **1.016**

**Layer 4 combined:**

```
Layer_4_Output = 0.985 × 1.008 × 1.016 = 1.009
```

**Multiplier: ~1.01** — Layer 4 sits essentially neutral.

**Structural finding (important):** Layer 4 alone does NOT pull a declining veteran below 1.0. This is intentional architecture:

1. Lockett's underlying scouting profile is moderate-positive — P4 school, decent college usage, Madden ratings still alive
2. SL-018 collapses RAS contribution at year 2+, so even his lower RAS does not drag much
3. The age trajectory penalty in the breakout component (0.00 at age 33) is offset by static signals (school, usage) that reflect what he was at draft entry, not what he is now

Layer 4 captures the scouting/development signal — "what kind of player is this?" Layer 3 captures aging — "how long is the runway?" They do different jobs.

**Full Layer 3 × Layer 4 chain for Lockett:**

Layer 3 age_pull at age 33 = 0.97^4 ≈ 0.885. SL-018 buffer at his RAS (0.85 normalized) = 0.10 × 0.85 = 0.085. Buffered age_pull = 1.0 + (0.885 − 1.0) × (1 − 0.085) = 1.0 + (−0.115)(0.915) = **0.895**.

```
Layer 3 × Layer 4 (Lockett) = 0.895 × 1.009 = 0.903
Layer 3 × Layer 4 (Jefferson) = ~1.00 × 1.109 = 1.109
```

A 22% spread between an elite in-peak WR and a declining veteran — produced by the engine separating "is he good?" from "how long does he have?"

---

## 6. Open Questions Surfaced

All items below are general-scope and land in `Roadmap_and_Open_Questions.md`. Two were reconciled from Gemini's WR open questions; three surfaced during this build.

- **SL-OQ-013 (from Gemini, refined):** School-tier-conditional imputation for missing breakout sub-signals. When college usage rate (or other breakout signals) is Absent for FCS/Non-FCS prospects, the standard Layer 1 position-group mean fallback under-credits a prospect who was high-usage at their small school but lacks tracking data. Refinement: per-position-rubric normalization functions allow school-tier-conditional fallbacks (e.g., FCS + missing usage → assume 0.70, since the prospect would not be on NFL radar without high college usage). Admin-console-tunable.

- **SL-OQ-014 (from Gemini, expanded):** Rookie Madden regulation behavior. Madden's pre-Week-4 rookie ratings are intentionally conservative — EA does not predict hot rookies that may bust. Madden therefore under-rates rookies relative to consensus expert claims, which would incorrectly trigger Approach D blend downward on high-consensus rookies. Two resolution paths: (a) widen Madden regulation threshold for rookies until first mid-season refresh, (b) switch Madden regulation OFF for rookies until first mid-season Madden update. Gemini's CAL-015 (gradient tuning) is the operational tune on top of whichever architectural path is chosen.

- **SL-OQ-015:** Breakout age operational definition. "First college season with ≥20% team receiving production" is one definition; "first season ranked top-3 in team receiving" is another; "first 100-target season" is a third. The pipeline needs the league to pick one and document it. Likely settles to the definition that best correlates with NFL production once 3+ seasons of pipeline data are live.

- **SL-OQ-016:** College usage rate signal type. Target share vs. yards-share vs. TD-share. Currently defaulted to target share as the most production-stable signal. Verify against best-available source data once pipeline is live.

- **SL-OQ-017:** Veteran breakout-component behavior. Lockett's verification shows three of four breakout sub-signals (breakout age, school tier, college usage) are static and reflect rookie-era development. Only age trajectory updates in veteran years, and at 0.15 weight it cannot pull the component below 1.0 on its own when the static signals are positive. **This is the rookie model behavior — the Veteran Scouting Layer Extension (Deliverable 3, SL-008) will evaluate whether veteran-era rubrics should re-weight or replace static signals.** Not a flaw of the WR rubric.

**Calibration Backlog addition surfaced from WR build:**
- **CAL-015:** Madden regulation gradient tuning for rookie WRs (and rookies in general) — operational tuning of whichever SL-OQ-014 resolution path is chosen.

---

## 7. Position-Specific Notes

- WR validates the High-tier RAS architecture. Year-0 RAS contribution is meaningful (potential ±8% cap). SL-018 collapses contribution to residual by year 2+.
- WR validates the breakout age signal — the gold-standard predictor for the position is captured at 0.40 weight, the highest among breakout sub-signals.
- WR provides the cleanest template for offensive skill positions. RB, TE, QB rubrics follow this skeleton with position-specific value shifts (peak limits, RAS tier weights, breakout age elite thresholds, position-relevant Madden attributes).
- All RAS values in verification cases are estimates pending ras.football verification. Final calibration cases will use sourced RAS values.

---

## 8. Cross-Pollination Source

This rubric synthesized from:
- Universal Rubric Template v1.1 (structural skeleton)
- Engine Specification v2.1 (Layer 4 mechanics)
- Gemini's WR rubric draft (baseline values, sub-signal weights, Madden mapping framework, S-curve parameters)
- Gemini's WR open questions (reconciled — SL-OQ-013 accepted with refinement; SL-OQ-014 expanded from Gemini's calibration framing into architecture question + calibration item pair)
- SL-018 (RAS time-decay) applied after Gemini's draft
- Two Madden mapping rows added for archetypal coverage (high-point specialist, power-after-catch)
- Age trajectory curve tightened past peak
- EMA α refined for RSP, TDN, Sharp

---

*Built by: Christopher Campbell + Claude (Anthropic)*

| Version | Date | Changes |
|---|---|---|
| 0.9 | June 2026 | Draft from cross-pollinated Gemini baseline + audit refinements. SL-018 applied. Both verification cases worked end-to-end with full math. Pending Gemini's WR open questions for reconciliation pass before v1.0 lock. |
| 1.0 | June 2026 | Locked. Gemini's WR open questions reconciled into SL-OQ-013 (school-tier-conditional imputation) and SL-OQ-014 (rookie Madden regulation behavior). Three WR-discovered OQs renumbered into global SL-OQ series (SL-OQ-015 through SL-OQ-017). CAL-015 added to Calibration Backlog. |
