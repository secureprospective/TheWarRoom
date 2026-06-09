# Legacy NFL Position Group Rubric — Quarterback (QB)
**Version:** 1.0 — June 2026
**Status:** Locked. Introduces SL-020 Low-tier RAS implementation rule (zero Layer 4 contribution, Layer 6 tiebreaker only). Validates Universal Rubric Template v1.1 against the position where film consensus dominates and RAS is poorly predictive. Gemini's QB open questions reconciled into SL-OQ-028 and CAL-021.
**Companion:** Engine_Specification.md Layer 4 is authoritative on mechanics.

---

## 1. Architectural Baseline

- **Layer 4 RAS Tier:** **Low** (per SL-004). Per SL-020 (locked this session), Low-tier RAS contributes nothing to Layer 4 — RAS_position_weight forced to 0.00, cap forced to 0%, Layer 4 RAS output forced to 1.000 for all QBs. RAS value is sourced and routed to Layer 6 tiebreaker storage only.
- **Layer 3 Peak Limit:** 32 years. Longest peak window of any position — reflects "arms last longer than legs" biological reality.
- **SL-019 application:** **Does NOT apply to QB.** RAS is poorly predictive at this position, so RAS-modulator interactions on breakout-age and age-trajectory curves are disabled. Breakout sub-signals operate on base curve values only.
- **SL-018 Layer 3 buffer:** Standard formula applies (`buffer_pct = 0.10 × RAS_normalized`). Flagged for empirical validation since RAS does not predict QB longevity well (see CAL-020).
- **Layer 2 Base Points drivers** (per Official Rulebook, MFL-sourced):
  - **Passing:** TDs (5.0), Yards (0.05/yd), INT (−2.0), Long Pass bonus (+1 at 40+ yds, single tier), 2PT Conversions (2.0)
  - **Rushing (for mobile QBs):** TDs (6.0), Yards (0.1/yd), Attempts (0.15 each), Long Rush bonuses (+1 at 20+ yds, +1 additional at 40+ yds, stackable), 2PT Conversions (2.0)
  - **No passing attempt scoring. No sack scoring. No passing yard milestones beyond 40+ bonus.**
  - QB rushing follows the standard rushing scoring table — meaningful for mobile QBs (Daniels, Allen, Lamar archetypes).
- **Data Parity Rule:** Missing/Unknown sub-signal data collapses component deviation to 0.0 via confidence weighting, returning neutral 1.00 fallback.

---

## 2. Film Component Configuration

**Cap (asymptote):** ±5%.

### Sub-signal weights

| Source | Weight | Classification |
|---|---|---|
| PFF Passing Grade (analytical anchor) | 0.45 | Analytical — self-regulates |
| Matt Waldman RSP (subjective anchor) | 0.35 | Subjective — Madden-regulated |
| The Draft Network | 0.10 | Subjective — Madden-regulated |
| Sharp Football Analysis | 0.10 | Subjective — Madden-regulated |

Sums to 1.00. PFF elevated to 0.45 (vs. WR's 0.40, RB's 0.35) — film consensus is the dominant Layer 4 signal at QB since RAS is excluded. PFF passing grade captures play-by-play decision quality more robustly than RSP can at this position.

### Madden regulation parameters

- **Threshold:** 0.15
- **Blend scaling:** Linear gradient over 0.10 delta beyond threshold

### Madden attribute mapping

| Subjective Expert Claim | Madden Sub-Attribute / Composite | Formula |
|---|---|---|
| "Elite arm talent / velocity" | Throw Power (THP) | Direct |
| "Short-to-medium precision" | Short Accuracy (SAC) + Medium Accuracy (MAC) | Average(SAC, MAC) |
| "Deep ball accuracy / touch" | Deep Accuracy (DAC) | Direct |
| "Off-platform / mobile playmaking" | Throw on the Run (RUN) + Acceleration (ACC) | Average(RUN, ACC) |
| "Pocket presence / processing" | Awareness (AWR) + Play Action (PAC) | Average(AWR, PAC) |
| "Clutch / throws under pressure" | Throw Under Pressure (TUP) + Awareness (AWR) | Average(TUP, AWR) |

Six rows. Sixth row added — Throw Under Pressure (TUP) is a distinct Madden attribute capturing the clutch/collapse-under-pressure dimension that Mahomes-type QBs are elite at and some QBs collapse on. Gemini's five rows miss this.

### Signal mechanics

- `film_position_weight`: 1.00
- `film_inflection`: 0.50
- `film_steepness`: 12.0

### EMA blend rates (dynamic sub-signals)

| Sub-signal | α | Classification |
|---|---|---|
| RSP | 0.50 | Dynamic — annual publication |
| PFF | 0.15 | Dynamic — weekly updates |
| TDN | N/A | **STATIC** — locked at rookie evaluation |
| Sharp | 0.50 | Dynamic — annual + occasional updates |
| Madden | 0.20 | Dynamic — multiple mid-season updates |

### Season transition behavior

CONTINUATION across all dynamic sub-signals.

### Sub-signal normalization (0.0–1.0 mapping)

- **PFF:** PFF Passing Grade / 100
- **RSP:** Percentile rank within annual RSP QB rankings
- **TDN:** TDN composite grade mapped to percentile within draft class QBs
- **Sharp:** Sharp dynasty ranking inverted to percentile within Sharp's QB pool

---

## 3. RAS Component Configuration

**Per SL-020: Low-tier RAS = zero Layer 4 contribution.**

### Parameters (forced values)

| Parameter | Value | Notes |
|---|---|---|
| `RAS_position_weight` | 0.00 | Year 0, Year 1, Year 2+ — all 0.00 (SL-018 schedule does not apply) |
| `RAS_cap` | 0.00 | No asymptote applies |
| `RAS_inflection` | 0.50 | Stored but inoperative |
| `RAS_steepness` | 0.0 | Stored but inoperative |
| Layer 4 RAS output | **1.000 forced** | Component contributes nothing to multiplier |

### Normalization

`RAS_normalized = raw_RAS / 10.0` — still computed, used for Layer 3 buffer and Layer 6 tiebreaker.

### Late-career interaction (SL-018 standard buffer)

Standard SL-018 Layer 3 buffer applies at QB:
```
buffer_pct        = 0.10 × RAS_normalized
buffered_age_pull = 1.0 + (raw_age_pull − 1.0) × (1 − buffer_pct)
```

Same formula as WR/RB — NOT the SL-019 amplified version (TE-specific). Flagged for validation: RAS does not predict QB longevity cleanly (mobile QBs like RG3, Vick had high RAS but short careers due to playing style). See CAL-020.

### Layer 6 tiebreaker routing

RAS value is sourced from ras.football data and stored in the Layer 6 scarcity matrix as a tiebreaker attribute. When two QBs produce identical Layer 4 multipliers, Layer 6 can break the tie using RAS as a downstream signal (per SL-020 rule).

---

## 4. Breakout Component Configuration

**Cap (asymptote):** ±5%.

### Sub-signal weights

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.30 |
| School Tier | 0.25 |
| College Offensive Share Index | 0.30 |
| Age Trajectory | 0.15 |

Sums to 1.00. Breakout age weight reduced from WR's 0.40 — QB breakout age is less predictive than WR breakout age because most QB development happens later (junior/senior years). School tier elevated to 0.25 — college program quality matters more for QB scheme exposure and pro readiness.

### Parameters

- `breakout_position_weight`: 1.00
- `breakout_inflection`: 0.50
- `breakout_steepness`: 11.0

### Normalization functions

**Breakout Age** — "first season as starting QB at FBS school" — see SL-OQ-015 dependency:

| Breakout Age | Normalized |
|---|---|
| ≤20.0 | 1.00 |
| 21.0 | 0.80 |
| 22.0 | 0.50 |
| ≥23.0 | 0.10 |

Linear interpolation between defined points. Shifted later than WR (which uses ≤19=1.00) — QB true-freshman starters at FBS are rare and elite signal, but the curve doesn't penalize 21-year-old breakouts as severely as WR does because junior-year-starter is the standard QB development arc.

**No SL-019 modulation applies at QB.**

**School Tier** (template defaults):

| Tier | Normalized |
|---|---|
| Power Four | 1.00 |
| Group of Five | 0.70 |
| FCS | 0.40 |
| Non-FCS | 0.10 |

**College Offensive Share Index** — operational definition: starts as percentage of college career games available (see SL-OQ-026):

| Starter Share | Normalized |
|---|---|
| ≥65% | 1.00 |
| 50% | 0.55 |
| ≤35% | 0.15 |

Linear interpolation between defined points.

**Age Trajectory** — slower decay than other positions reflecting QB longevity:

| Age | Normalized |
|---|---|
| ≤28 | 1.00 |
| 29 | 0.90 |
| 30 | 0.80 |
| 31 | 0.65 |
| 32 (peak) | 0.50 |
| 33 | 0.35 |
| 34 | 0.25 |
| 35 | 0.15 |
| 36 | 0.10 |
| ≥37 | 0.00 |

Linear interpolation between defined points. **No SL-019 modulation applies at QB.**

### Three-zone classification

Aligned to WR/RB/TE:
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

### Case 1 — Push: Jayden Daniels

**Profile:**
- Age 25 (June 2026), Year 3 NFL (drafted 2024 #2 overall by WAS, Heisman winner 2023)
- College: Arizona State 2019-2021 (Pac-12 = Power Four at time of his starts), LSU 2022-2023 (SEC = P4)
- Breakout age: 19 (ASU true-freshman starter 2019)
- Career starts: ~45 of ~50 possible games ≈ 90% — elite college offensive share
- PFF career grades: top-tier
- RSP / TDN / Sharp: consensus elite at entry

**Film component:**

```
(0.45 × 0.90) + (0.35 × 0.92) + (0.10 × 0.95) + (0.10 × 0.90)
= 0.4050 + 0.3220 + 0.0950 + 0.0900
= 0.9120
```

S-curve(0.9120, 0.50, 12.0, 0.05):
- arg = 12.0 × 0.4120 = 4.944
- σ(4.944) ≈ 0.9929
- output factor = 0.9858
- film_raw = **1.049**
- film_effective = **1.049**

**RAS component (SL-020):**

Forced to **1.000** (Low-tier, no Layer 4 contribution). Daniels' actual RAS (estimated ~9.0+ given speed/athleticism) is routed to Layer 6 tiebreaker storage and used in the standard SL-018 buffer formula at Layer 3, but contributes nothing to Layer 4.

RAS_effective = **1.000**

**Breakout component (no SL-019 modulators):**

- Breakout age 19 → 1.00 (≤20 threshold)
- School tier P4 → 1.00
- College Offensive Share 90% → 1.00 (≥65% threshold)
- Age trajectory 25 → 1.00 (≤28 threshold)

Composite:
```
(0.30 × 1.00) + (0.25 × 1.00) + (0.30 × 1.00) + (0.15 × 1.00)
= 1.000
```

Composite is at the **Elite zone ceiling** (1.00).

S-curve(1.000, 0.50, 11.0, 0.05):
- σ(11.0 × 0.500) = σ(5.500) ≈ 0.9959
- output factor = 0.9918
- breakout_raw = **1.050**
- breakout_effective = **1.050**

**Layer 4 combined:**

```
Layer_4_Output = 1.049 × 1.000 × 1.050 = 1.101
```

**Multiplier: ~1.10** — clear push case.

Daniels registers near-cap film and maxed breakout component. RAS contributes nothing (SL-020) but he wouldn't need it — film and breakout combined push him to the same range as Bowers and Jefferson at their respective positions. The Low-tier RAS doesn't disadvantage elite QBs because the architecture compensates with broader film + breakout caps.

---

### Case 2 — Pull: Derek Carr

**Profile:**
- Age 35 (June 2026), Year 13 NFL veteran (drafted 2014 R2 by OAK)
- College: Fresno State (Mountain West = Group of Five)
- Breakout age: 21 (first full-time starter season at Fresno 2012, redshirted 2009 + freshman backup 2010 + injured 2011)
- Career starts: ~30 of ~50 possible games ≈ 60% — moderate (linear interp: 0.85)
- PFF: declining hard recent years, borderline benchable 2025
- RSP / TDN / Sharp: Round 2 prospect grade at entry, recent assessments emphasize decline

**Film component:**

```
(0.45 × 0.40) + (0.35 × 0.40) + (0.10 × 0.50) + (0.10 × 0.30)
= 0.1800 + 0.1400 + 0.0500 + 0.0300
= 0.4000
```

S-curve(0.4000, 0.50, 12.0, 0.05):
- arg = 12.0 × (−0.1000) = −1.200
- σ(−1.200) ≈ 0.231
- output factor = 2 × 0.231 − 1 = −0.538
- film_raw = 1 + 0.05 × (−0.538) = **0.973**
- film_effective = **0.973**

**RAS component (SL-020):**

Forced to **1.000**. Carr's RAS (~6.5 estimate) routed to Layer 3 buffer + Layer 6 tiebreaker, contributes nothing to Layer 4.

RAS_effective = **1.000**

**Breakout component (no SL-019 modulators):**

- Breakout age 21 → 0.80
- School tier G5 → 0.70
- College Offensive Share 60% → 0.85 (linear interp: 50% = 0.55, 65% = 1.00)
- Age trajectory 35 → 0.15

Composite:
```
(0.30 × 0.80) + (0.25 × 0.70) + (0.30 × 0.85) + (0.15 × 0.15)
= 0.2400 + 0.1750 + 0.2550 + 0.0225
= 0.6925
```

Composite is in the **Average zone** (0.40 < x < 0.80) — close to Elite boundary.

S-curve(0.6925, 0.50, 11.0, 0.05):
- σ(11.0 × 0.1925) = σ(2.1175) ≈ 0.892
- output factor = 0.784
- breakout_raw = 1 + 0.05 × 0.784 = **1.039**
- breakout_effective = **1.039**

**Layer 4 combined:**

```
Layer_4_Output = 0.973 × 1.000 × 1.039 = 1.011
```

**Multiplier: ~1.01** — essentially neutral with mild push.

### Structural finding — fourth instance of the Lockett pattern

Carr produces the same architectural result as Lockett (WR ~1.01), Henry (TE ~1.05), and now Carr (QB ~1.01): **declining veterans with strong rookie-era profiles do not get pulled below 1.0 at Layer 4.** The static breakout sub-signals (school tier, college offensive share, breakout age) don't decay over career. Layer 4 captures the scouting/development signal; Layer 3 captures aging.

This is now a four-position pattern (WR, TE, QB at Layer 4 ≈ 1.0; RB Herbert was the exception because his rookie profile was genuinely weak). **Deliverable 3 — Veteran Scouting Layer Extension — is the proper venue to evaluate whether veteran rubrics should re-weight or decay static signals.** Documented as SL-OQ-017 since the WR session. The QB instance reinforces this should be a priority for Deliverable 3.

**Full Layer 3 × Layer 4 chain for Carr:**

Layer 3 age_pull at 35 (3 years past peak 32) = 0.97³ ≈ 0.913. Standard SL-018 buffer at RAS_normalized 0.65 = 6.5%. Buffered age_pull = 1.0 + (0.913 − 1.0)(0.935) = **0.919**.

```
Layer 3 × Layer 4 (Carr)     = 0.919 × 1.011 = 0.929
Layer 3 × Layer 4 (Daniels)  = 1.000 × 1.101 = 1.101
```

A 17% spread — Layer 3 doing the heavy lifting on the decline, as expected at QB where Layer 4 RAS contributes nothing.

---

## 6. Open Questions Surfaced

Prior sessions surfaced SL-OQ-013 through SL-OQ-025. QB adds:

- **SL-OQ-026:** College Offensive Share Index operational definition for QB. Candidates: (a) starts as % of college career games available, (b) starts as % including redshirt year denominator, (c) starts at primary school only (ignoring transfers). Currently defaulted to (a). Resolves with multi-season validation data.

- **SL-OQ-027:** SL-019 applicability gating. TE explicitly uses SL-019; QB explicitly does not. Each position rubric must declare whether SL-019 applies. Need a structural rule: how is "SL-019 applies" determined? Candidate criterion: SL-019 applies when RAS tier is High AND empirical data shows RAS predicts longevity for that position. Refines during DE rubric build (position 5, where SL-019 likely applies).

- **SL-OQ-028 (from Gemini, repositioned):** Rushing QB scouting signal in Layer 4. Mobile QBs derive significant Layer 2 value from rushing, but Layer 4 only captures their mobility indirectly through Madden mobility attributes in the "off-platform / mobile playmaking" subjective claim. Gemini proposed a secondary component modifier triggered by rushing share > 30% of weekly Layer 2 output, but that risks double-counting (Layer 2 already rewards rushing volume). **Repositioned to architectural refinement:** add a 5th sub-signal "PFF QB Rushing Grade" to the QB film component at ~0.10 weight, with PFF Passing reduced from 0.45 to ~0.35. Captures rushing EFFICIENCY (yards per carry, broken tackles, big-play rate) as a scouting signal distinct from volume. PFF publishes a separate QB Running Grade — data source exists. Same pattern as SL-OQ-025 (TE PFF receiving/blocking split). Deferred to v1.1+ branch.

**Calibration Backlog additions from QB build:**

- **CAL-020:** SL-018 standard Layer 3 buffer validation at QB. RAS does not predict QB longevity cleanly — mobile QBs with high RAS sometimes have shorter careers due to playing style. Buffer formula may need QB-specific reduction or removal. Requires empirical longevity data by RAS tier at QB.

- **CAL-021 (from Gemini):** SL-020 RAS-collapse market-skew verification. Empirically confirm that collapsing RAS to zero Layer 4 contribution prevents under-rating of low-RAS pocket passers (Mahomes, Brady, Burrow archetypes) with elite processing traits. Not a new architectural question — verification that SL-020 achieves its intended effect in practice.

---

## 7. Position-Specific Notes

- QB is the first Low-tier RAS position. SL-020 implementation rule (zero Layer 4 contribution, Layer 6 tiebreaker only) locked this session. Applies to K when we reach position 9.
- QB applies neither SL-019 modulators (RAS not predictive enough at this position) nor the SL-019 amplified Layer 3 buffer (uses standard 0.10 × RAS_normalized). Most architecturally distinct position rubric so far.
- Long peak window (32) with slower age trajectory decay reflects QB-specific longevity reality — "arms last longer than legs."
- Fourth instance confirming the Lockett pattern — strong rookie profile + declining current production = Layer 4 stays near or above 1.0, Layer 3 does the aging work. Confirms this is structural across positions, not position-specific.
- All RAS values stored for Layer 3 buffer and Layer 6 tiebreaker; estimates pending ras.football verification.

---

## 8. Cross-Pollination Source

This rubric synthesized from:
- Universal Rubric Template v1.1 (structural skeleton)
- Engine Specification v2.1 (Layer 4 mechanics)
- Gemini's QB rubric draft (Low-tier RAS clean implementation — most important architectural call, accepted as SL-020 baseline; PFF elevation to 0.45; QB-specific renaming of "College Usage" to "College Offensive Share Index"; breakout age curve shifted later for QB)
- Gemini's QB open questions (reconciled — rushing QB modifier repositioned from Layer 4 multiplier to film sub-signal architecture, deferred to v1.1+ as SL-OQ-028; market-skew verification accepted as CAL-021)
- Christopher's SL-020 confirmation (Low-tier = 0 Layer 4 weight, Layer 6 tiebreaker only)
- Sixth Madden mapping row added (Clutch / throws under pressure)
- SL-019 explicitly disabled for QB (RAS not predictive — does not modulate breakout-age or age-trajectory curves)
- TDN reclassified from dynamic α=1.0 to STATIC (consistency with prior fixes)
- RSP and Sharp EMA α refined to 0.50
- Age trajectory curve tightened with QB-specific slower decay
- Layer 2 driver list expanded to include QB rushing scoring (per rulebook verification — standard rushing scoring table applies to QB carries)

---

*Built by: Christopher Campbell + Claude (Anthropic)*

| Version | Date | Changes |
|---|---|---|
| 0.9 | June 2026 | Draft. Introduces SL-020 Low-tier RAS implementation rule (zero Layer 4 contribution, Layer 6 tiebreaker only). SL-019 explicitly disabled for QB. Two verification cases (Daniels push, Carr pull). Fourth instance confirming the Lockett pattern. All math worked end-to-end. Pending Gemini's QB open questions for reconciliation pass before v1.0 lock. |
| 1.0 | June 2026 | Locked. Gemini's QB open questions reconciled. Rushing QB Layer 4 capture repositioned from multiplicative modifier to film sub-signal architecture (SL-OQ-028, deferred to v1.1+). RAS-collapse market-skew verification accepted as CAL-021. Renumbered Gemini's local SL-OQ-016 and CAL-018 to avoid collision with already-used global numbers. |
