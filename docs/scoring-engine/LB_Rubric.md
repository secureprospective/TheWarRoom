# Legacy NFL Position Group Rubric — Linebacker (LB)
**Version:** 1.0 — June 2026
**Status:** Locked. First non-SL-019 IDP rubric. SL-005 compression applied to film component only (cap ±3% + steepness 10.0, position_weight retained at 1.00). RAS and breakout components untouched at standard Medium-tier values — film weakness is not compensated by RAS overconfidence at a scheme-dependent position. Gemini's LB open questions reconciled into SL-OQ-032 and CAL-025.
**Companion:** Engine_Specification.md Layer 4 is authoritative on mechanics. This rubric specifies position-specific values, the SL-005 compression mechanic at LB, and the SL-OQ-027 gating-rule exclusion of LB from SL-019.

---

## 1. Architectural Baseline

- **Layer 4 RAS Tier:** **Medium** (per SL-004). RAS is one factor among many at LB; scheme-fit and pre-snap processing share predictive weight with athletic profile.
- **Layer 3 Peak Limit:** 29 years.
- **SL-005 application:** **YES — film component only.** IDP data gap at LB compresses film signal: cap ±3% (vs. standard ±5%), film_steepness 10.0 (vs. standard 12.0). film_position_weight retained at 1.00 (no triple-compression — cap + steepness already express the compression).
- **SL-019 application:** **NO.** Per the SL-OQ-027 gating rule (closure documented in DE rubric Section 7), SL-019 requires BOTH High-tier RAS AND a position-specific predictive relationship between athletic profile and longevity. LB fails both: Medium tier per SL-004, and LB longevity is scheme-driven (coverage LBs and run-defender thumpers follow different decline curves — no clean RAS→longevity signal at the position). Layer 3 buffer stays at SL-018 standard 0.10 × RAS_normalized.
- **RAS treatment under SL-005:** **NOT elevated.** Cap stays at standard Medium ±4%, RAS_position_weight follows Medium-tier SL-018 schedule (0.60 / 0.30 / 0.06). SL-005 is film-specific per the handoff text. Compensating film weakness by inflating RAS would create false confidence in athletic-profile-strong-but-scheme-mismatched players — LB is the most scheme-dependent dynasty position and this risk is real.
- **Layer 2 Base Points drivers** (per Official Rulebook lines 88–117, MFL-sourced per DECISION-009):
  - **True Position split (LB/CB/S):** Tackle 1.5, Assist 1.0
  - **Pass-rush production:** Sack 4.5, QB Hit 1, Tackle for Loss 2.5 (direct stat — no proxy)
  - **Coverage:** Pass Defensed 2.5, Interception Caught 5, Interception Return Yards 0.025/yd, Interception Return TD 6
  - **Turnover production:** Forced Fumble 4, Opponent Fumble Recovery 3, Opponent Fumble Recovery Yards 0.025/yd
  - **Special teams adjacent:** Blocked Kick 7
  - **Game-state:** Safety 10
- **Data Parity Rule:** Missing/Unknown sub-signal data collapses component deviation to 0.0 via confidence weighting, returning neutral 1.00 fallback.

---

## 2. Film Component Configuration

**Cap (asymptote):** **±3%** (SL-005 compression — vs. standard ±5%).

### Sub-signal weights

| Source | Weight | Classification |
|---|---|---|
| PFF Linebacker Defense Grade (analytical anchor) | 0.40 | Analytical — self-regulates |
| The IDP Show (subjective anchor) | 0.30 | Subjective — Madden-regulated |
| The IDP Guru (analytical modifier) | 0.20 | Analytical — self-regulates |
| The Draft Network (pre-draft trait eval) | 0.05 | Subjective — Madden-regulated |
| Dynasty Nerds | 0.05 | Subjective — Madden-regulated |

Sums to 1.00.

TDN and Dynasty Nerds split from Gemini's combined 0.10 "TDN/Nerds" bundle into separate sub-signals with distinct EMA behaviors. PFF anchor weight (0.40) and IDP Show (0.30) dominate — IDP analytical coverage at LB, while thin compared to offensive positions, is strongest at the two anchor sources.

### Madden regulation parameters

- **Threshold:** 0.15 (normalized scale)
- **Blend scaling:** Linear gradient over 0.10 delta beyond threshold

### Madden attribute mapping

| Subjective Expert Claim | Madden Sub-Attribute / Composite | Formula |
|---|---|---|
| "Sideline-to-sideline / elite range" | Speed (SPD) + Pursuit (PUR) | Average(SPD, PUR) |
| "Thumper / interior run plugger" | Tackle (TAK) + Hit Power (HPW) + Strength (STR) | Average(TAK, HPW, STR) |
| "Coverage LB / content in space" | Zone Coverage (ZCV) + Play Recognition (PRC) | Average(ZCV, PRC) |
| "Block take-on / sheds cleanly" | Block Shedding (BSH) + Awareness (AWR) | Average(BSH, AWR) |
| "Three-down hybrid / blitzing playmaker" | Speed (SPD) + Pursuit (PUR) + Power Moves (PMV) | Average(SPD, PUR, PMV) |

Five rows. Fifth added to cover the modern three-down hybrid archetype (Devin White / Patrick Queen / Tremaine Edmunds) — the LB who combines coverage range, blitz value, and turnover production. Gemini's four rows collapse this archetype awkwardly into either Range or Coverage.

### Signal mechanics

- `film_position_weight`: **1.00** (standard — SL-005 compression expressed via cap + steepness only)
- `film_inflection`: 0.50
- `film_steepness`: **10.0** (SL-005 compression — vs. standard 12.0)

### EMA blend rates (dynamic sub-signals)

| Sub-signal | α | Classification |
|---|---|---|
| PFF | 0.15 | Dynamic — weekly grades with slow blend |
| IDP Show | 0.30 | Dynamic — weekly podcast/content, moderate blend |
| IDP Guru | 0.20 | Dynamic — weekly analytical content |
| TDN | N/A | **STATIC** — locked at rookie evaluation; no re-publication for veterans |
| Dynasty Nerds | 0.50 | Dynamic — annual-publication-rate weekly content |
| Madden | 0.20 | Dynamic — multiple mid-season updates with moderate blend |

### Season transition behavior

CONTINUATION across all dynamic sub-signals.

### Sub-signal normalization (0.0–1.0 mapping)

- **PFF:** PFF Linebacker Defense Grade / 100 (grades are already 0–100)
- **IDP Show:** Tier ranking inverted to percentile within The IDP Show's LB pool
- **IDP Guru:** Weekly LB ranking inverted to percentile within IDP Guru's LB pool
- **TDN:** TDN composite grade mapped to percentile within draft class LB
- **Dynasty Nerds:** Dynasty LB ranking inverted to percentile within Dynasty Nerds' LB pool

---

## 3. RAS Component Configuration

**Cap (asymptote):** **±4%** (Medium-tier standard — NOT elevated by SL-005).

### Parameters

- `RAS_position_weight`: **Medium-tier SL-018 schedule** (NOT a flat 1.00):

| NFL career stage | RAS_position_weight |
|---|---|
| Rookie / pre-NFL data | 0.60 |
| After 1 NFL season | 0.30 |
| Year 2+ | 0.06 |

- `RAS_inflection`: 0.50 (equivalent to raw RAS = 5.00)
- `RAS_steepness`: 11.0

### Normalization

`RAS_normalized = raw_RAS / 10.0`

Missing RAS → Layer 1 position-group mean imputation. Confidence flag set to Unknown.

### Late-career interaction (SL-018 standard buffer, NO SL-019)

Once player age > 29 (LB peak limit), Layer 3 age_pull is RAS-buffered at the SL-018 standard rate:

```
buffer_pct        = 0.10 × RAS_normalized   ← standard (NO SL-019 amplification)
buffered_age_pull = 1.0 + (raw_age_pull − 1.0) × (1 − buffer_pct)
```

An LB with RAS = 9.0 gets a ~9% buffer against age decay. An LB with RAS = 5.0 gets a 5% buffer. The differential is real but modest — consistent with the position's scheme-driven longevity profile.

---

## 4. Breakout Component Configuration

**Cap (asymptote):** ±5% (standard — NOT compressed by SL-005).

### Sub-signal weights

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.25 |
| School Tier | 0.20 |
| College Production Share (Tackle + Sack + TFL market share) | 0.40 |
| Age Trajectory | 0.15 |

Sums to 1.00.

College Production Share elevated to 0.40 (vs. WR's 0.20, TE's 0.30, DE's 0.35) — combined tackle + sack + TFL market share is the dominant college signal at LB, and its richness compensates for the position's thin pre-draft scouting depth. Breakout Age dropped to 0.25 reflecting that LBs typically break out later than offensive skill players (junior-year emergence is normative, not exceptional).

### Parameters

- `breakout_position_weight`: 1.00
- `breakout_inflection`: 0.50
- `breakout_steepness`: 11.0

### Normalization functions

**Breakout Age** — base curve (LB-specific, recognizes later-emergence norm):

| Breakout Age | Normalized |
|---|---|
| ≤20.0 | 1.00 |
| 21.0 | 0.75 |
| 22.0 | 0.45 |
| ≥23.0 | 0.15 |

Linear interpolation between defined points. **SL-019 modulation does NOT apply at LB.**

**School Tier** (template defaults):

| Tier | Normalized |
|---|---|
| Power Four | 1.00 |
| Group of Five | 0.70 |
| FCS | 0.40 |
| Non-FCS | 0.10 |

**College Production Share** (final-year tackle + sack + TFL market share, defined as the player's share of his team's total at each event class, averaged across the three event classes):

| Market Share | Normalized |
|---|---|
| ≥25% | 1.00 |
| 18% | 0.55 |
| ≤10% | 0.15 |

Linear interpolation between defined points.

**Age Trajectory** (current age relative to LB peak limit of 29):

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

**SL-019 modulation does NOT apply at LB.**

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

### Case 1 — Push: Fred Warner

**Profile:**
- Age 29 (born November 1996) — at peak limit
- College: BYU (Independent at the time — classified as G5 for tier purposes; BYU moved to Big 12 / P4 in 2023, well after Warner's college career)
- R3 #70 overall 2018
- Breakout age: 20 (junior year as starter, first-team All-Independent)
- Junior-year college production share: ~22% combined tackle + sack + TFL market share
- RAS: ~6.5 *(estimated, pending ras.football verification — 4.65 40 at 236lbs with strong agility numbers but average size/explosion grade)*
- PFF: elite recent grades, ~91
- Year 8 NFL veteran → SL-018 Year 2+ tier (RAS_position_weight = 0.06 at Medium)

**Film component (SL-005 compressed: cap ±3%, steepness 10.0):**

Sub-signal normalizations:
- PFF LB Grade ~91 → 0.91
- IDP Show top-tier LB consensus → 0.95
- IDP Guru top-3 LB → 0.95
- TDN (static, 2018 R3 grade) → 0.60
- Dynasty Nerds top dynasty rank → 0.90

Composite:
```
(0.40 × 0.91) + (0.30 × 0.95) + (0.20 × 0.95) + (0.05 × 0.60) + (0.05 × 0.90)
= 0.364 + 0.285 + 0.190 + 0.030 + 0.045
= 0.914
```

S-curve(0.914, 0.50, 10.0, 0.03):
- arg = 10.0 × 0.414 = 4.14
- σ(4.14) ≈ 0.9843
- output factor = 2 × 0.9843 − 1 = 0.9686
- film_raw = 1 + 0.03 × 0.9686 = **1.029**
- film_effective = **1.029**

Note: Even with elite signals across the board, SL-005 compression caps film at ~1.029 (vs. DE Garrett's 1.049 with identical-quality signals). This is the intended architectural effect — film can't drive Layer 4 as hard at IDP positions with thin data.

**RAS component (Medium-tier, Year 2+):**

- RAS_normalized = 6.5 / 10 = 0.65
- S-curve(0.65, 0.50, 11.0, 0.04):
  - arg = 11.0 × 0.15 = 1.65
  - σ(1.65) ≈ 0.8389
  - output factor = 0.6778
  - RAS_raw = 1 + 0.04 × 0.6778 = **1.027**
- Year 2+ → RAS_position_weight = 0.06
- RAS_effective = 1.0 + (1.027 − 1.0) × 1.0 × 0.06 = **1.002**

Note: RAS contribution is essentially residual at Year 8 / Medium tier (0.06 weight). This is expected per SL-018 — RAS is a development-era signal at Medium-tier positions.

**Breakout component (SL-019 NOT applied):**

- Breakout Age 20 → base 1.00 (Gemini's curve has ≤20.0 maxed)
- School Tier G5 (BYU Independent at the time) → 0.70
- College Production Share 22% → linear interp between 18% (0.55) and 25% (1.00): 0.55 + 4 × 0.0643 = **0.807**
- Age Trajectory 29 (peak) → 0.50

Composite:
```
(0.25 × 1.00) + (0.20 × 0.70) + (0.40 × 0.807) + (0.15 × 0.50)
= 0.250 + 0.140 + 0.323 + 0.075
= 0.788
```

Composite is in the **Average zone** (just below 0.80 Elite threshold).

S-curve(0.788, 0.50, 11.0, 0.05):
- arg = 11.0 × 0.288 = 3.168
- σ(3.168) ≈ 0.9596
- output factor = 0.9191
- breakout_raw = 1 + 0.05 × 0.9191 = **1.046**
- breakout_effective = **1.046**

**Layer 4 combined:**

```
Layer_4_Output = 1.029 × 1.002 × 1.046 = 1.078
```

**Multiplier: ~1.08** — clear push. Lower magnitude than DE Garrett (1.11) due to SL-005 film compression and residual Medium-tier Year 2+ RAS contribution. Architecture working as designed.

**Full Layer 3 × Layer 4 chain for Warner:**

Layer 3 age_pull at age 29 = 0.97^0 = 1.000 (at peak, no decay yet). No buffer applied.

```
Layer 3 × Layer 4 (Warner) = 1.000 × 1.078 = 1.078
```

---

### Case 2 — Pull-attempt: Lavonte David

**Profile:**
- Age 36 (born January 1990) — 7 years past peak limit
- College: Nebraska (Power Four — Big Ten); JUCO transfer from Fort Scott CC
- R2 #58 overall 2012
- Breakout age: 20 (junior year at Nebraska, first season as D1 starter, All-Big Ten)
- Senior-year college production share: ~26% combined tackle + sack + TFL market share at Nebraska
- RAS: ~6.0 *(estimated, pending ras.football verification — 4.65 40 at 230lbs with strong burst but undersized frame depressing the composite)*
- PFF: declining, ~65 recent grade (was 80s+ in 2013–2018 prime)
- Year 14 NFL veteran → SL-018 Year 2+ tier (RAS_position_weight = 0.06 at Medium)

**Film component (SL-005 compressed):**

Sub-signal normalizations:
- PFF LB Grade ~65 → 0.65
- IDP Show middle-tier vet → 0.35
- IDP Guru declining vet rank → 0.40
- TDN (static, 2012 R2 grade) → 0.55
- Dynasty Nerds low dynasty rank → 0.20

Composite:
```
(0.40 × 0.65) + (0.30 × 0.35) + (0.20 × 0.40) + (0.05 × 0.55) + (0.05 × 0.20)
= 0.260 + 0.105 + 0.080 + 0.0275 + 0.010
= 0.4825
```

S-curve(0.4825, 0.50, 10.0, 0.03):
- arg = 10.0 × (−0.0175) = −0.175
- σ(−0.175) ≈ 0.4564
- output factor = 2 × 0.4564 − 1 = −0.0872
- film_raw = 1 + 0.03 × (−0.0872) = **0.997**
- film_effective = **0.997**

Note: Even with substantial film decline (PFF 65 vs. peak 85+, IDP Show ranking dropped to mid-tier), SL-005 compression limits the film pull to ~0.003 below neutral. The architecture intentionally prevents film signals from driving Layer 4 at LB.

**RAS component (Medium-tier, Year 2+):**

- RAS_normalized = 6.0 / 10 = 0.60
- S-curve(0.60, 0.50, 11.0, 0.04):
  - arg = 11.0 × 0.10 = 1.10
  - σ(1.10) ≈ 0.7503
  - output factor = 0.5005
  - RAS_raw = 1 + 0.04 × 0.5005 = **1.020**
- Year 2+ → RAS_position_weight = 0.06
- RAS_effective = 1.0 + (1.020 − 1.0) × 1.0 × 0.06 = **1.001**

**Breakout component (SL-019 NOT applied):**

- Breakout Age 20 → base 1.00 (already maxed)
- School Tier P4 → 1.00
- College Production Share 26% → 1.00 (above 25% threshold)
- Age Trajectory 36 → ≥33 = 0.00

Composite:
```
(0.25 × 1.00) + (0.20 × 1.00) + (0.40 × 1.00) + (0.15 × 0.00)
= 0.250 + 0.200 + 0.400 + 0.000
= 0.850
```

Composite is in the **Elite zone** (≥ 0.80) — despite age 36. Three of four sub-signals maxed; only age trajectory pulls, and at 0.15 weight it cannot drag the composite below the inflection point.

S-curve(0.850, 0.50, 11.0, 0.05):
- arg = 11.0 × 0.350 = 3.85
- σ(3.85) ≈ 0.9791
- output factor = 0.9582
- breakout_raw = 1 + 0.05 × 0.9582 = **1.048**
- breakout_effective = **1.048**

**Layer 4 combined:**

```
Layer_4_Output = 0.997 × 1.001 × 1.048 = 1.046
```

**Multiplier: ~1.05** — **PUSH despite being 36 years old.**

**Structural finding (Lockett pattern at LB, amplified by SL-005):** This is the Lockett pattern in its purest expression so far. David's static breakout signals (breakout age 20, P4 Nebraska, 26% college market share) are all maxed; only age trajectory updates with current age, and at 0.15 weight it cannot move the composite below the inflection point. Meanwhile, SL-005 compresses the film component so the genuine film decline (PFF dropped 20+ points from his prime) produces only ~0.003 of pull at Layer 4.

The combined effect: **at LB, Layer 4 alone is incapable of producing a meaningful pull on aging veterans with strong rookie profiles.** Layer 3 carries essentially all of the aging work. This is the architecture working as designed — film at IDP positions is noisy enough that its veteran-decline signal should NOT drive valuation; the deterministic Layer 3 age decay is the cleaner mechanism.

The Lockett finding now formalized at six positions (WR, RB-with-Herbert-exception, TE, QB, DE, LB). SL-005 amplifies the pattern at LB specifically.

**Full Layer 3 × Layer 4 chain for David:**

Layer 3 age_pull at age 36 = 0.97^7 ≈ 0.808. SL-018 STANDARD buffer (no SL-019 amplification at LB) = 0.10 × 0.60 = 0.06 (6%). Buffered age_pull = 1.0 + (0.808 − 1.0) × (1 − 0.06) = 1.0 + (−0.192)(0.94) = **0.820**.

```
Layer 3 × Layer 4 (David)  = 0.820 × 1.046 = 0.858
Layer 3 × Layer 4 (Warner) = 1.000 × 1.078 = 1.078
```

A 25% spread between an in-peak elite LB and a seven-years-past-peak veteran — produced almost entirely by Layer 3. The engine correctly separates "what kind of player was he?" from "how long does he have?"

---

## 6. Open Questions Surfaced

Prior sessions surfaced SL-OQ-013 through SL-OQ-030 and CAL-015 through CAL-023. LB adds:

- **SL-OQ-031:** SL-005 compression depth calibration at LB. Current implementation uses two of three available knobs (cap ±3% + steepness 10.0, position_weight retained at 1.00). Empirical question: does the architecture want stronger SL-005 compression (additional position_weight reduction to ~0.70) once real league data is available? The Lavonte David case shows current compression already produces effectively zero film pull on aging vets — additional compression may be unnecessary or may further muddy the architecture. Defer to live-data calibration.

- **SL-OQ-032 (from Gemini, renumbered from local SL-OQ-018):** DL-dominated-system context normalization for LB college production share. A talented off-ball LB on a college team with elite DL (Alabama, Georgia, Clemson in their dominant cycles) may have suppressed tackle counts because the defensive line is making plays at the line of scrimmage. The 0.40-weighted College Production Share signal could under-rank such players. Refinement candidate: scheme-context adjustment that weights tackle market share relative to the team's DL pressure rate or QB-hit market share. Distinct from CAL-024 (which addresses intra-metric weighting between tackle/sack/TFL) and CAL-023 (snap-count normalization at DE) — this is specifically about defensive scheme suppression of off-ball opportunities. All three concerns share a root cause: college production share calculations need context-awareness, and all three resolve together once a college-data pipeline is settled.

**Calibration Backlog additions from LB build:**

- **CAL-024:** LB college production share weighting methodology. The current definition averages three event classes (Tackle market share, Sack market share, TFL market share) equally. Tackles are a volume stat that scales with snap participation and may inflate the metric for high-snap MIKEs vs. rotational LBs. Sacks and TFLs are discrete impact events. Refinement: test asymmetric weighting (e.g., 0.4 tackle / 0.3 sack / 0.3 TFL) or snap-count-normalized tackle market share. Linked dependency with CAL-023 (DE snap-count normalization) — same college data source question blocks both.

- **CAL-025 (from Gemini, renumbered and reframed from local CAL-020):** Empirical effectiveness tracking of SL-005 compression at dampening year-over-year IDP scheme-change noise. Gemini's original framing referenced film_position_weight = 0.70, but Christopher's call reset that parameter to 1.00; the underlying empirical concern remains valid and is here reframed to track current implementation (cap ±3% + steepness 10.0). Pairs with the decisional gate at SL-OQ-031 — CAL-025 is the empirical-tracking side (measure how often defensive scheme changes drive valuation noise), SL-OQ-031 is the decisional side (do we add more compression in response). Both require live league data.

---

## 7. Position-Specific Notes

- **SL-005 application at LB — film-only compression:** Three knobs are architecturally available for compressing scouting signal at thin-data positions: cap, steepness, and position_weight. Current LB implementation uses cap + steepness (±3% / 10.0). position_weight retained at standard 1.00 to keep the knob reserved for SL-018 NFL-career-stage scheduling. The Lavonte David verification case demonstrates that current compression is already aggressive — a vet with PFF dropped 20+ points from prime produces ~0.003 of film pull. Further compression via position_weight may not be necessary; flagged as SL-OQ-031 for live-data review.

- **RAS NOT elevated under SL-005:** LB is the most scheme-dependent dynasty position. Coverage LBs and run-defender thumpers age along different curves; a Cover-2 LB at one team can become a 3-4 ILB at another and lose his RAS-aligned role entirely. Inflating RAS to compensate for film weakness would create false confidence in athletic-profile-strong-but-scheme-mismatched players. RAS cap stays at standard Medium ±4%, RAS_position_weight on the Medium-tier SL-018 schedule (0.60 / 0.30 / 0.06).

- **SL-019 excluded by gating rule:** Per the SL-OQ-027 closure (DE Section 7), SL-019 requires High-tier RAS + position-specific athletic-profile-to-longevity predictive relationship. LB fails both criteria. No SL-019 modulators on breakout-age, age-trajectory, or Layer 3 buffer.

- **Lockett pattern amplified at LB:** SL-005 compression on film + strong static breakout signals at most LB rookie profiles + Medium-tier residual Year 2+ RAS = Layer 4 cannot meaningfully pull aging veterans below 1.0. Layer 3 carries essentially all aging work. This is by design — IDP film signal is noisy enough that its decline signal should not drive valuation. Veteran Scouting Layer Extension (Deliverable 3, SL-008) is the venue for re-weighting static breakout signals across NFL career stages.

- **College Production Share definition is multi-class at LB:** Combined tackle + sack + TFL market share is the dominant signal at the position. CAL-024 flags whether equal weighting across the three event classes is appropriate; live-data calibration will determine.

- **EDGE classification boundary:** Per OQ-004 resolution, this rubric covers off-ball linebackers only. Players consensus-classified as pass-rush primary (DE/EDGE/3-4 OLB by role) route through the DE rubric regardless of MFL position tag. The disambiguation is consensus role, not MFL string matching.

- All RAS values in verification cases are estimates pending ras.football verification.

---

## 8. Cross-Pollination Source

This rubric synthesized from:
- Universal Rubric Template v1.1 (structural skeleton)
- Engine Specification v2.1 (Layer 4 mechanics)
- Gemini's LB rubric draft (Medium-tier RAS recognition, ±3% film cap intuition, film steepness 10.0, archetype framework, base curves for breakout age and college production share, S-curve parameters)
- Gemini's LB open questions (reconciled — SL-OQ-032 DL-dominated-system context normalization accepted as novel concern distinct from CAL-024 weighting and CAL-023 snap-count refinements; CAL-025 reframed from Gemini's original CAL-020 because Christopher's call reset film_position_weight to 1.00, removing the specific parameter Gemini was tracking — empirical-tracking concern preserved against current SL-005 implementation and paired with SL-OQ-031 decisional gate)
- Christopher's calls (SL-005 = film cap + steepness only, position_weight retained at 1.00; RAS not elevated; SL-019 explicitly NO per gating rule; Fred Warner push, Lavonte David pull-attempt)
- SL-018 applied to RAS with Medium-tier schedule (Gemini's draft predated this)
- SL-019 explicitly excluded per SL-OQ-027 gating rule
- Layer 2 drivers expanded to include Forced Fumble (4), Opponent Fumble Recovery (3) + return yards — Gemini's list was incomplete
- EMA blend rates fixed: TDN → STATIC, Dynasty Nerds split from TDN at α=0.50, IDP Show split from IDP Guru with α=0.30 / 0.20 respectively
- Fifth Madden mapping row added (three-down hybrid / blitzing playmaker — Devin White / Patrick Queen / Tremaine Edmunds archetype)
- Age trajectory curve tightened from Gemini's "31+=0.15" plateau to gradual decline through 0.00 at age 33
- `[cite: N]` markers stripped throughout
- Verification cases replaced from generic placeholders with named real-player cases and full S-curve math

---

*Built by: Christopher Campbell + Claude (Anthropic)*

| Version | Date | Changes |
|---|---|---|
| 0.9 | June 2026 | Draft from cross-pollinated Gemini baseline + audit refinements. First non-SL-019 IDP rubric. SL-005 compression applied to film only (cap ±3% + steepness 10.0, position_weight retained at 1.00). RAS untouched at Medium-tier standard (cap ±4%, schedule 0.60/0.30/0.06). SL-019 explicitly excluded per SL-OQ-027 gating rule (Medium tier + scheme-dependent longevity). Layer 2 drivers expanded. Both verification cases worked end-to-end with full math (Warner ~1.08 push; David ~1.05 Layer 4 with strong Lockett-pattern finding amplified by SL-005, ~0.86 on full Layer 3 × Layer 4 chain). Pending Gemini's LB open questions for reconciliation pass before v1.0 lock. |
| 1.0 | June 2026 | Locked. Gemini's LB open questions reconciled. SL-OQ-032 (DL-dominated-system context normalization) renumbered from Gemini's local SL-OQ-018, accepted as a novel concern distinct from CAL-024 intra-metric weighting and CAL-023 snap-count normalization — though all three share a root cause (college production share needs context-awareness) and resolve together once the college-data pipeline is settled. CAL-025 reframed from Gemini's local CAL-020 because Christopher's call reset film_position_weight to 1.00, removing the specific parameter Gemini tracked; empirical-tracking concern preserved against current SL-005 implementation (cap ±3% + steepness 10.0) and explicitly paired with SL-OQ-031 as decisional-vs-empirical sides of the same compression-depth question. |
