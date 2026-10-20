# Layer 4 Scouting Engine — Pre-Build Audit
**Version:** 1.0 — June 2026
**Purpose:** Definitive handoff to Claude Code. Everything in Section 1 is verified against locked rubrics at 95%+ confidence. Section 2 are confirmed corrections to Gemini blueprint errors. Section 3 requires Christopher's decision before build. Section 4 is deferred and does not block initial build.

---

## Section 1: CONFIRMED FOR BUILD

### 1A. Engine Structural Mechanics (Code-Locked — Never Expose to Admin UI)

> **Note:** Code examples in this section use Python syntax for readability. The implementation language is Go. Type handling — particularly `None`/nil and the data hygiene flag logic — must be translated carefully, not copied mechanically.

#### Layer 4 Formula Chain

```
Layer_4_Output = film_effective × RAS_effective × breakout_effective

component_effective = 1.0 + (component_raw - 1.0) × confidence × position_weight

Scouting_Adjusted_Points = Base_Points × age_pull × Layer_4_Output
```

No overall Layer 4 cap. Each component's asymptote is the natural bound.

#### Shape B S-Curve (all components, all positions)

```python
import math

def calculate_scurve(input_val, steepness, inflection, cap):
    arg = max(-500.0, min(500.0, steepness * (input_val - inflection)))  # overflow guard
    sigma = 1.0 / (1.0 + math.exp(-arg))
    raw_output = 1.0 + cap * (2.0 * sigma - 1.0)
    return max(1.0 - cap, min(1.0 + cap, raw_output))  # hard boundary clamp
```

Formula: `output = 1 + cap × (2 × σ(steepness × (input − inflection)) − 1)`

Boundary behavior:
- Input = 1.00 → output approaches 1 + cap (upper asymptote)
- Input = 0.00 → output approaches 1 - cap (lower asymptote)
- Input = inflection → output = 1.000 (neutral, no push/pull)

#### Madden Regulation (Approach D — subjective sub-signals only)

```
if |expert_claim - madden_normalized| < madden_threshold:
    effective_weight = rubric_weight  (no regulation)
else:
    blend_factor = (disagreement - threshold) / 0.10  [linear, caps at 1.0]
    effective_weight = blend(rubric_weight, madden_implied_weight, blend_factor)
```

- Madden threshold default: 0.15 (admin-tunable per position)
- Blend scaling: linear gradient over 0.10 delta beyond threshold (admin-tunable)
- Applies to: RSP, TDN, IDP Show, Sharp, Dynasty Nerds — ALL subjective sub-signals
- Does NOT apply to: PFF, NGS, IDP Guru, NFL production data — analytical signals self-regulate

#### EMA Blending (Option A)

```python
def blend_dynamic_signal(previous_value, new_observation, alpha, season_stage="In-Season"):
    if season_stage == "Off-Season":
        return previous_value  # hold — no decay during silent window
    return (1.0 - alpha) * previous_value + alpha * new_observation
```

- First observation becomes the sub-signal's starting value; EMA begins on second observation
- Default season transition behavior: **CONTINUATION** (prior season final value blends with new season first observation)
- Position rubrics may override to RESET per sub-signal with rationale

#### Three-State Data Flags

```python
def enforce_data_hygiene(field_value, position, signal_type):
    if field_value is None or field_value == "Unknown":
        return get_neutral_baseline(position, signal_type), "Unknown", 0.0
    elif field_value == "Absent":
        return get_positional_mean(position, signal_type), "Absent", 0.0
    else:
        return field_value, "Present", 1.0
```

| State | Meaning | Confidence Impact |
|---|---|---|
| Present | Data exists and sourced | 1.0 |
| Absent | Data will never exist (didn't test, N/A) | 0.0 — positional fallback used |
| Unknown | Not yet collected, may exist | 0.0 — neutral placeholder used |

Layer 1 data hygiene defaults:
- Missing RAS: position-group mean, confidence = 0.0 (Unknown)
- Missing Breakout Age: 21.0, flagged Unknown
- Missing School Tier: Group of Five, flagged Unknown

#### Confidence Calculation

```
film_confidence     = Σ(film_field_present_weights) / Σ(film_field_expected_weights)
RAS_confidence      = 1.0 if Present else 0.0
breakout_confidence = Σ(breakout_field_present_weights) / Σ(breakout_field_expected_weights)
```

When all fields in a component are Unknown: confidence = 0.0 → effective = 1.0 + (raw - 1.0) × 0.0 × weight = 1.000 (clean neutral fallback, no distortion).

Confidence scores are INTERNAL ENGINE FLAGS ONLY. Never surface in UI.

#### Lockett Pattern — Breakout Component Veteran Immutability

```python
def process_breakout_component(player):
    if player.is_veteran:
        breakout_age_signal   = player.historical_draft_breakout_age_normalized
        school_tier_signal    = player.historical_draft_school_tier_normalized
        college_usage_signal  = player.historical_draft_college_usage_normalized
        # NOTE: store normalized scores at draft entry, not raw values
    else:
        breakout_age_signal   = normalize_breakout_age(player.current_breakout_age, player.position)
        school_tier_signal    = normalize_school_tier(player.current_school_tier, player.position)
        college_usage_signal  = normalize_college_usage(player.current_college_usage, player.position)

    age_trajectory_signal = calculate_age_trajectory(player.age, player.position)
    # Age trajectory is always live — derived from current age vs. peak limit

    return aggregate_approach_a(
        breakout_age_signal, school_tier_signal, 
        college_usage_signal, age_trajectory_signal, 
        player.position
    )
```

Static sub-signals (locked at draft entry, never update):
- Pre-NFL Breakout Age
- School Tier
- College Usage / Workload / Production Share

Age Trajectory is always live (not static) — updates automatically as player ages relative to position peak limit. Layer 3 manages the aging timeline. Layer 4 static signals preserve the talent fingerprint.

#### MFL Player ID Type Enforcement

```python
assert isinstance(player_record.mfl_id, str), "MFL Player ID must be string type"
# IDs under 1000 require leading zero: "0999", not 999
```

#### Layer 2 / Layer 4 Separation (Zero Scoring Leaks)

No sub-signal within Film, RAS, or Breakout may reference: fantasy points per game, projected volume, MFL scoring config, format-dependent volume stats. Layer 2 handles rulebook scoring. Layer 4 handles real-world physical and qualitative traits. These are mathematically isolated.

---

### 1B. Cross-Position Mechanics

#### SL-018 — RAS Position Weight Time-Decay Schedule

RAS contribution decays as NFL data accumulates. Schedule scales proportionally by tier:

| Tier | Year 0 (Pre-NFL) | After Year 1 | Year 2+ |
|---|---|---|---|
| High | 1.00 | 0.50 | 0.10 |
| Medium | 0.60 | 0.30 | 0.06 |
| Low | 0.00 | 0.00 | 0.00 |

SL-018 governs the RAS COMPONENT weight only. SL-018 is INDEPENDENT of SL-019 modulator interactions — the two mechanics operate on separate channels.

#### SL-018 Layer 3 Buffer (Standard — all SL-018 positions)

Once player age exceeds position peak limit:

```
buffer_pct        = 0.10 × RAS_normalized
buffered_age_pull = 1.0 + (raw_age_pull − 1.0) × (1 − buffer_pct)
```

Positions using STANDARD buffer (0.10 × RAS): QB, RB, WR, LB. Buffer is admin-tunable.

#### SL-019 — RAS Modulator Interactions (Applicable Positions Only)

SL-019 applies when BOTH: (a) High-tier RAS, AND (b) position-specific predictive relationship between athletic profile and longevity.

**Confirmed applicable:** TE, DE, CB, S
**Confirmed NOT applicable:** QB (Low-tier), RB (Medium-tier), LB (Medium-tier + scheme-dependent), DT (uses Cushion Guard instead), K (SL-020 exclusion)
**WR: unresolved** — see Section 3.

SL-019 has THREE distinct interaction values per position (not one "strength"):

| Position | Breakout Age Modulator | Age Trajectory Modulator | Layer 3 Buffer |
|---|---|---|---|
| TE | 0.35 | 0.35 | **0.30** |
| DE | 0.35 | 0.35 | **0.30** |
| CB | 0.30 | 0.30 | **0.25** |
| S | 0.30 | 0.30 | **0.25** |

SL-019 breakout modulator formula:
```
modulated_value = base + (1.0 − base) × modulator_strength × RAS_normalized
```

Applied to both breakout age and age trajectory sub-signals in the breakout component.

SL-019 amplified Layer 3 buffer (replaces standard 0.10 rate):
```
buffer_pct        = sl019_buffer_strength × RAS_normalized
buffered_age_pull = 1.0 + (raw_age_pull − 1.0) × (1 − buffer_pct)
```

SL-019 modulator interactions are INDEPENDENT of SL-018. RAS as a modulator of other curves remains active across the career — an aging TE's/DE's/CB's/S's athletic profile is structural, not residual.

#### SL-020 — Low-Tier RAS Exclusion (QB and K)

For QB:
- `RAS_position_weight` = 0.00 at all career stages (SL-018 schedule does NOT apply)
- `RAS_cap` = 0.00
- Layer 4 RAS output **forced to 1.000** for all QBs
- RAS value still sourced and stored → routed to Layer 3 standard buffer and Layer 6 tiebreaker

For K:
- Layer 4 hardcoded to **1.000** for all three components (full structural exclusion)
- RAS still sourced → Layer 6 tiebreaker routing only
- Layer 3 + Layer 5 + Layer 2 carry all kicker ranking work

#### SL-021 — DT Hybrid Tier Architecture

DT is the only position with this pattern. Three-part resolution:

**Part 1 — Tier classification:**
- Film component: Medium-tier with SL-005 compression (cap ±3%, steepness 10.0)
- RAS component: High-tier treatment (cap ±8%, schedule 1.00/0.50/0.10)
- SL-019: NOT applied (Cushion Guard replaces it — running both would double-protect)

**Part 2 — Dynamic Year 1 / Year 2+ PFF EMA:**

```python
def get_dt_pff_alpha(nfl_experience_years):
    if nfl_experience_years <= 1:
        return 0.50  # aggressive blend — forces new NFL grades to overwrite rookie-era RAS anchor quickly
    else:
        return 0.10  # slow blend — stable veteran signal
```

Year 1 ends at conclusion of first NFL regular season. All other DT sub-signals use standard α values.

**Part 3 — Late-Career Cushion Guard:**

Binary gate: if Raw RAS ≥ 8.00, apply 10% flat deceleration on decay velocity. Below 8.00: no protection.

Applied at TWO points (both use the same formula):

```python
def apply_cushion_guard(raw_value, peak_value, raw_ras):
    if raw_ras >= 8.00:
        return peak_value - (peak_value - raw_value) * 0.90
    else:
        return raw_value
```

Point 1 — Layer 3 age_pull beyond peak:
```
cushioned_age_pull = 1.0 − (1.0 − raw_age_pull) × 0.90
```

Point 2 — Breakout component Age Trajectory sub-signal beyond peak:
```
cushioned_age_trajectory = peak − (peak − base_value) × 0.90
```
Where peak = 0.50 (the value at the peak age boundary in the Age Trajectory table).

Cushion Guard is conservative vs. SL-019 (~1pp protection per year past peak vs. SL-019's ~5pp at elite-RAS S). Reflects that DT athletic-profile-to-longevity is less cleanly predictive than coverage positions.

#### SL-005 — Film Compression at Data-Thin Positions (LB and DT)

| Parameter | Standard | SL-005 Compressed |
|---|---|---|
| film_cap | ±5% | **±3%** |
| film_steepness | 12.0 | **10.0** |
| film_position_weight | 1.00 | **1.00 (RETAINED)** |

Compression is expressed through cap and steepness ONLY. position_weight is not reduced (triple-compression would over-suppress). RAS component is NOT elevated to compensate — doing so would create false confidence in athletically strong but scheme-mismatched players.

---

### 1C. Per-Position Parameter Tables

#### School Tier Normalization

Default values (template) — used at QB, TE, WR, DE, LB, CB, S, DT, K:

| Tier | Normalized |
|---|---|
| Power Four | 1.00 |
| Group of Five | 0.70 |
| FCS | 0.40 |
| Non-FCS | 0.10 |

RB-specific (softer non-P4 penalty — small-school workhorse production translates more reliably):

| Tier | Normalized |
|---|---|
| Power Four | 1.00 |
| Group of Five | **0.75** |
| FCS | **0.45** |
| Non-FCS | **0.15** |

---

#### QUARTERBACK (QB)

| Parameter | Value |
|---|---|
| Layer 4 RAS Tier | Low — SL-020 forced 1.000 |
| Layer 3 Peak Limit | 32 |
| SL-019 | Disabled |
| SL-020 | Active — RAS_position_weight = 0.00, cap = 0.00, output forced 1.000 |
| Film Cap | ±5% |
| film_inflection | 0.50 |
| film_steepness | 12.0 |
| film_position_weight | 1.00 |
| RAS Cap | 0% (forced) |
| Breakout Cap | ±5% |
| breakout_inflection | 0.50 |
| breakout_steepness | 11.0 |
| breakout_position_weight | 1.00 |
| Layer 3 Buffer | Standard 0.10 × RAS_normalized (RAS low predictive value — CAL-020 flagged) |

Film sub-signal weights:

| Source | Weight | Type |
|---|---|---|
| PFF Passing Grade | 0.45 | Analytical |
| Matt Waldman RSP | 0.35 | Subjective |
| The Draft Network | 0.10 | Subjective — **STATIC** |
| Sharp Football Analysis | 0.10 | Subjective |

EMA α values:

| Sub-signal | α | Note |
|---|---|---|
| PFF | 0.15 | Dynamic |
| RSP | 0.50 | Dynamic |
| TDN | N/A | **STATIC** — locked at rookie evaluation |
| Sharp | 0.50 | Dynamic |
| Madden | 0.20 | Dynamic |

Madden attribute mapping (6 rows):

| Expert Claim | Madden Composite | Formula |
|---|---|---|
| "Elite arm talent / velocity" | Throw Power (THP) | Direct |
| "Short-to-medium precision" | SAC + MAC | Average |
| "Deep ball accuracy / touch" | Deep Accuracy (DAC) | Direct |
| "Off-platform / mobile playmaking" | Throw on the Run (RUN) + ACC | Average |
| "Pocket presence / processing" | AWR + Play Action (PAC) | Average |
| "Clutch / throws under pressure" | Throw Under Pressure (TUP) + AWR | Average |

Breakout sub-signal weights:

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.30 |
| School Tier | 0.25 |
| College Offensive Share Index | 0.30 |
| Age Trajectory | 0.15 |

Breakout age normalization (QB-specific — shifted later than WR):

| Breakout Age | Normalized |
|---|---|
| ≤20.0 | 1.00 |
| 21.0 | 0.80 |
| 22.0 | 0.50 |
| ≥23.0 | 0.10 |

Age trajectory normalization:

| Age | Normalized |
|---|---|
| ≤28 | 1.00 |
| 32 (peak) | 0.50 |
| 35 | 0.15 |
| ≥37 | 0.00 |

Linear interpolation between defined points. **No SL-019 modulation.**

Three-zone boundaries: Elite ≥ 0.80, Late ≤ 0.40.

---

#### RUNNING BACK (RB)

| Parameter | Value |
|---|---|
| Layer 4 RAS Tier | Medium |
| Layer 3 Peak Limit | 25 |
| SL-019 | Disabled |
| Film Cap | ±5% |
| film_inflection | 0.50 |
| film_steepness | 12.0 |
| film_position_weight | 1.00 |
| RAS Cap | ±4% |
| RAS_inflection | 0.50 |
| RAS_steepness | 8.0 |
| RAS_position_weight schedule | Medium-tier: 0.60 / 0.30 / 0.06 |
| Layer 3 Buffer | Standard 0.10 × RAS_normalized |
| Breakout Cap | ±5% |
| breakout_inflection | 0.50 |
| breakout_steepness | 11.0 |
| breakout_position_weight | 1.00 |

Film sub-signal weights:

| Source | Weight | Type |
|---|---|---|
| Matt Waldman RSP | 0.35 | Subjective |
| PFF Rushing/Receiving Grade | 0.35 | Analytical |
| FantasyPros Touch Share | 0.20 | Analytical (opportunity signal) |
| The Draft Network | 0.05 | Subjective — **STATIC** |
| Sharp Football Analysis | 0.05 | Subjective |

EMA α values:

| Sub-signal | α | Note |
|---|---|---|
| RSP | 0.50 | Dynamic |
| PFF | 0.20 | Dynamic (faster than WR — reflects RB usage volatility) |
| Touch Share | 0.25 | Dynamic (week-to-week usage swings) |
| TDN | N/A | **STATIC** |
| Sharp | 0.50 | Dynamic |
| Madden | 0.20 | Dynamic |

Madden attribute mapping (5 rows):

| Expert Claim | Madden Composite | Formula |
|---|---|---|
| "Elite speed / home-run threat" | SPD + ACC | Average |
| "Power / contact balance" | Trucking (TRK) + Break Tackle (BTK) + STR | Average |
| "Elusiveness / open-field agility" | Juke Move (JUK) + Spin Move (SPN) + AGI | Average |
| "Vision / creative instincts" | Ball Carrier Vision (BCV) | Direct |
| "Passing-down utility / hands" | CTH + Pass Blocking (PBLK) | (0.7 × CTH) + (0.3 × PBLK) |

Breakout sub-signal weights:

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.35 |
| School Tier | 0.20 |
| College Workload | 0.30 |
| Age Trajectory | 0.15 |

Breakout age normalization (aggressive dropoff — short career window compounds penalty):

| Breakout Age | Normalized |
|---|---|
| ≤19.5 | 1.00 |
| 20.0 | 0.80 |
| 20.5 | 0.50 |
| ≥21.0 | 0.20 |

College Workload normalization (touches as share of team RB touches — SL-OQ-018 default):

| Workload Share | Normalized |
|---|---|
| ≥40% | 1.00 |
| 30% | 0.60 |
| ≤20% | 0.15 |

Age trajectory normalization:

| Age | Normalized |
|---|---|
| ≤21 | 1.00 |
| 25 (peak) | 0.50 |
| 28 | 0.05 |
| ≥29 | 0.00 |

Linear interpolation. **No SL-019 modulation.**

Three-zone boundaries: Elite ≥ 0.80, Late ≤ 0.40.

School tier: **RB-specific values** (G5=0.75, FCS=0.45, Non-FCS=0.15).

---

#### TIGHT END (TE)

| Parameter | Value |
|---|---|
| Layer 4 RAS Tier | High (amended SL-004) |
| Layer 3 Peak Limit | 29 |
| SL-019 | Enabled — three values: BA=0.35, AT=0.35, L3=0.30 |
| Film Cap | ±5% |
| film_inflection | 0.50 |
| film_steepness | 12.0 |
| film_position_weight | 1.00 |
| RAS Cap | ±8% |
| RAS_inflection | 0.50 |
| RAS_steepness | 11.0 |
| RAS_position_weight schedule | High-tier: 1.00 / 0.50 / 0.10 |
| Layer 3 Buffer | SL-019 amplified: 0.30 × RAS_normalized |
| Breakout Cap | ±5% |
| breakout_inflection | 0.50 |
| breakout_steepness | 11.0 |
| breakout_position_weight | 1.00 |

Film sub-signal weights:

| Source | Weight | Type |
|---|---|---|
| PFF Overall TE Grade | 0.40 | Analytical |
| Matt Waldman RSP | 0.35 | Subjective |
| The Draft Network | 0.15 | Subjective — **STATIC** |
| Sharp Football Analysis | 0.10 | Subjective |

EMA α values:

| Sub-signal | α | Note |
|---|---|---|
| RSP | 0.50 | Dynamic |
| PFF | 0.12 | Dynamic (slightly slower than WR — TE production noise) |
| TDN | N/A | **STATIC** |
| Sharp | 0.50 | Dynamic |
| Madden | 0.20 | Dynamic |

Madden attribute mapping (6 rows — 6th covers Move TE/H-back hybrid):

| Expert Claim | Madden Composite | Formula |
|---|---|---|
| "Seam stretcher / elite speed" | SPD + ACC | Average |
| "Inline blocker / point of attack" | Run Block (RBK) + Pass Block (PBK) + STR | (0.4×RBK) + (0.4×PBK) + (0.2×STR) |
| "Contested catch / red-zone weapon" | CTH + Catch in Traffic (CIT) + Jumping (JMP) | Average |
| "Route runner / separation" | Short Route Running (SRR) + Medium Route Running (MRR) | Average |
| "YAC threat / elusive" | Break Tackle (BTK) + AGI | Average |
| "Move TE / H-back versatility" | RBK + CTH + AGI | Average |

Breakout sub-signal weights:

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.35 |
| School Tier | 0.20 |
| College Usage Rate | 0.30 |
| Age Trajectory | 0.15 |

Breakout age normalization (base curve — SL-019 modulated):

| Breakout Age | Base Normalized |
|---|---|
| ≤20.0 | 1.00 |
| 21.0 | 0.80 |
| 22.0 | 0.50 |
| ≥23.0 | 0.15 |

SL-019 modulation (strength 0.35):
```
modulated = base + (1.0 − base) × 0.35 × RAS_normalized
```

College Usage Rate normalization (lower thresholds than WR — TE share structurally lower):

| Target Share | Normalized |
|---|---|
| ≥22% | 1.00 |
| 15% | 0.50 |
| ≤8% | 0.10 |

Age trajectory normalization (base curve — SL-019 modulated, strength 0.35):

| Age | Base Normalized |
|---|---|
| ≤25 | 1.00 |
| 29 (peak) | 0.50 |
| 32 | 0.10 |
| ≥33 | 0.00 |

Three-zone boundaries: Elite ≥ 0.80, Late ≤ 0.40.

---

#### WIDE RECEIVER (WR)

| Parameter | Value |
|---|---|
| Layer 4 RAS Tier | High |
| Layer 3 Peak Limit | 29 |
| SL-019 | See Section 3 — unresolved; default Disabled for initial build |
| Film Cap | ±5% |
| film_inflection | 0.50 |
| film_steepness | 12.0 |
| film_position_weight | 1.00 |
| RAS Cap | ±8% |
| RAS_inflection | 0.50 |
| RAS_steepness | 10.0 |
| RAS_position_weight schedule | High-tier: 1.00 / 0.50 / 0.10 |
| Layer 3 Buffer | Standard 0.10 × RAS_normalized |
| Breakout Cap | ±5% |
| breakout_inflection | 0.50 |
| breakout_steepness | 11.0 |
| breakout_position_weight | 1.00 |

Film sub-signal weights:

| Source | Weight | Type |
|---|---|---|
| Matt Waldman RSP | 0.40 | Subjective |
| PFF Receiver Grade | 0.40 | Analytical |
| The Draft Network | 0.10 | Subjective — **STATIC** |
| Sharp Football Analysis | 0.10 | Subjective |

EMA α values:

| Sub-signal | α | Note |
|---|---|---|
| RSP | 0.50 | Dynamic |
| PFF | 0.15 | Dynamic |
| TDN | N/A | **STATIC** |
| Sharp | 0.50 | Dynamic |
| Madden | 0.20 | Dynamic |

Madden attribute mapping (7 rows):

| Expert Claim | Madden Composite | Formula |
|---|---|---|
| "Elite speed / vertical threat" | SPD + ACC | Average |
| "Good hands / contested catch" | CTH + CIT + Spectacular Catch (SPC) | Average |
| "Route technician / separation" | SRR + MRR + Deep Route Running (DRR) | Average |
| "Press win / physical release" | Release (RLS) + STR | (0.8×RLS) + (0.2×STR) |
| "YAC threat / elusiveness" | BTK + JUK + AGI | Average |
| "High-point / contested-catch specialist" | JMP + SPC + CIT | Average |
| "Power-after-catch" | TRK + BTK + STR | Average |

Breakout sub-signal weights:

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.40 |
| School Tier | 0.25 |
| College Usage Rate | 0.20 |
| Age Trajectory | 0.15 |

Breakout age normalization:

| Breakout Age | Normalized |
|---|---|
| ≤19.0 | 1.00 |
| 20.0 | 0.75 |
| 21.0 | 0.40 |
| ≥22.0 | 0.10 |

College Usage Rate normalization (final-year target share):

| Target Share | Normalized |
|---|---|
| ≥35% | 1.00 |
| 25% | 0.50 |
| ≤15% | 0.10 |

Three-zone boundaries: Elite ≥ 0.80, Late ≤ 0.40.
School tier: template defaults (G5=0.70, FCS=0.40, Non-FCS=0.10).

---

#### DEFENSIVE END / EDGE (DE)

| Parameter | Value |
|---|---|
| Layer 4 RAS Tier | High |
| Layer 3 Peak Limit | 30 |
| SL-019 | Enabled — three values: BA=0.35, AT=0.35, L3=0.30 |
| EDGE classification | Pass-rush-primary defenders route through this rubric regardless of MFL tag (DE / EDGE / 3-4 OLB). Coverage/run-stop-primary off-ball LBs → LB rubric. |
| Film Cap | ±5% |
| film_inflection | 0.50 |
| film_steepness | 12.0 |
| film_position_weight | 1.00 |
| RAS Cap | ±8% |
| RAS_inflection | 0.50 |
| RAS_steepness | 10.0 |
| RAS_position_weight schedule | High-tier: 1.00 / 0.50 / 0.10 |
| Layer 3 Buffer | SL-019 amplified: 0.30 × RAS_normalized |
| Breakout Cap | ±5% |
| breakout_inflection | 0.50 |
| breakout_steepness | 11.0 |
| breakout_position_weight | 1.00 |

Film sub-signal weights:

| Source | Weight | Type |
|---|---|---|
| PFF Edge Defense Grade | 0.40 | Analytical |
| The IDP Show | 0.30 | Subjective |
| The Draft Network | 0.15 | Subjective — **STATIC** |
| Dynasty Nerds / IDP Guru combined | 0.15 | Subjective |

EMA α values:

| Sub-signal | α | Note |
|---|---|---|
| PFF | 0.15 | Dynamic |
| IDP Show | 0.30 | Dynamic |
| TDN | N/A | **STATIC** |
| Dynasty Nerds / IDP Guru | 0.50 | Dynamic |
| Madden | 0.20 | Dynamic |

Madden attribute mapping (6 rows — 6th covers complete power-finesse hybrid):

| Expert Claim | Madden Composite | Formula |
|---|---|---|
| "Elite speed rush / first-step explosion" | SPD + ACC | Average |
| "Power rusher / heavy hands / bull rush" | Power Moves (PMV) + STR | Average |
| "Technical rusher / counter-flex / hand fighter" | Finesse Moves (FMV) + AGI | Average |
| "Elite edge setter / run squeezer" | Block Shedding (BSH) + Tackle (TAK) | Average |
| "High motor / relentless pursuit" | Pursuit (PUR) + Play Recognition (PRC) | Average |
| "Complete edge / power-finesse hybrid" | PMV + FMV + BSH | Average |

Breakout sub-signal weights:

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.30 |
| School Tier | 0.20 |
| College Production Share (Sack + TFL market share) | 0.35 |
| Age Trajectory | 0.15 |

Breakout age normalization (base curve — SL-019 modulated, strength 0.35):

| Breakout Age | Base Normalized |
|---|---|
| ≤19.5 | 1.00 |
| 20.0 | 0.80 |
| 20.5 | 0.50 |
| ≥21.0 | 0.15 |

College Production Share normalization (final-year sack + TFL market share):

| Market Share | Normalized |
|---|---|
| ≥28% | 1.00 |
| 20% | 0.55 |
| ≤12% | 0.15 |

Age trajectory normalization (base curve — SL-019 modulated, strength 0.35):

| Age | Base Normalized |
|---|---|
| ≤26 | 1.00 |
| 30 (peak) | 0.50 |
| 33 | 0.10 |
| ≥34 | 0.00 |

Three-zone boundaries: Elite ≥ 0.80, Late ≤ 0.40.
School tier: template defaults.

---

#### LINEBACKER (LB)

| Parameter | Value |
|---|---|
| Layer 4 RAS Tier | Medium |
| Layer 3 Peak Limit | 29 |
| SL-005 | YES — film only (cap ±3%, steepness 10.0, position_weight retained at 1.00) |
| SL-019 | Disabled (Medium-tier + scheme-dependent longevity) |
| Film Cap | **±3%** |
| film_inflection | 0.50 |
| film_steepness | **10.0** |
| film_position_weight | **1.00** |
| RAS Cap | ±4% |
| RAS_inflection | 0.50 |
| RAS_steepness | 11.0 |
| RAS_position_weight schedule | Medium-tier: 0.60 / 0.30 / 0.06 |
| Layer 3 Buffer | Standard 0.10 × RAS_normalized (NO SL-019 amplification) |
| Breakout Cap | ±5% (standard — NOT compressed) |
| breakout_inflection | 0.50 |
| breakout_steepness | 11.0 |
| breakout_position_weight | 1.00 |

Film sub-signal weights:

| Source | Weight | Type |
|---|---|---|
| PFF Linebacker Defense Grade | 0.40 | Analytical |
| The IDP Show | 0.30 | Subjective |
| The IDP Guru | 0.20 | Analytical |
| The Draft Network | 0.05 | Subjective — **STATIC** |
| Dynasty Nerds | 0.05 | Subjective |

EMA α values:

| Sub-signal | α | Note |
|---|---|---|
| PFF | 0.15 | Dynamic |
| IDP Show | 0.30 | Dynamic |
| IDP Guru | 0.20 | Dynamic |
| TDN | N/A | **STATIC** |
| Dynasty Nerds | 0.50 | Dynamic |
| Madden | 0.20 | Dynamic |

Madden attribute mapping (5 rows):

| Expert Claim | Madden Composite | Formula |
|---|---|---|
| "Sideline-to-sideline / elite range" | SPD + PUR | Average |
| "Thumper / interior run plugger" | TAK + Hit Power (HPW) + STR | Average |
| "Coverage LB / content in space" | Zone Coverage (ZCV) + PRC | Average |
| "Block take-on / sheds cleanly" | BSH + AWR | Average |
| "Three-down hybrid / blitzing playmaker" | SPD + PUR + PMV | Average |

Breakout sub-signal weights:

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.25 |
| School Tier | 0.20 |
| College Production Share (Tackle + Sack + TFL market share) | 0.40 |
| Age Trajectory | 0.15 |

College Production Share normalization (average across tackle, sack, TFL event shares):

| Market Share | Normalized |
|---|---|
| ≥25% | 1.00 |
| 18% | 0.55 |
| ≤10% | 0.15 |

Three-zone boundaries: Elite ≥ 0.80, Late ≤ 0.40.
School tier: template defaults.

---

#### CORNERBACK (CB)

| Parameter | Value |
|---|---|
| Layer 4 RAS Tier | High |
| Layer 3 Peak Limit | 28 |
| SL-019 | Enabled — three values: BA=0.30, AT=0.30, L3=**0.25** |
| Film Cap | ±5% |
| film_inflection | 0.50 |
| film_steepness | 12.0 |
| film_position_weight | 1.00 |
| RAS Cap | ±8% |
| RAS_inflection | 0.50 |
| RAS_steepness | 11.0 |
| RAS_position_weight schedule | High-tier: 1.00 / 0.50 / 0.10 |
| Layer 3 Buffer | SL-019 amplified: **0.25** × RAS_normalized |
| Breakout Cap | ±5% |
| breakout_inflection | 0.50 |
| breakout_steepness | 10.0 |
| breakout_position_weight | 1.00 |
| NGS Coverage Metrics | Dedicated 0.30-weight analytical anchor (excluded from all other positions) |

Film sub-signal weights:

| Source | Weight | Type |
|---|---|---|
| PFF CB Coverage Grade | 0.35 | Analytical |
| NFL Next Gen Stats Coverage Metrics | 0.30 | Analytical |
| The IDP Show | 0.10 | Subjective |
| The IDP Guru | 0.10 | Analytical |
| The Draft Network | 0.08 | Subjective — **STATIC** |
| Dynasty Nerds | 0.07 | Subjective |

EMA α values:

| Sub-signal | α | Note |
|---|---|---|
| PFF | 0.18 | Dynamic |
| NGS Coverage Metrics | 0.20 | Dynamic |
| IDP Show | 0.30 | Dynamic |
| IDP Guru | 0.20 | Dynamic |
| TDN | N/A | **STATIC** |
| Dynasty Nerds | 0.50 | Dynamic |
| Madden | 0.20 | Dynamic |

Madden attribute mapping (5 rows — press row uses asymmetric weighting):

| Expert Claim | Madden Composite | Formula |
|---|---|---|
| "Elite recovery speed / vertical match" | SPD + ACC | Average |
| "Physical press / disruptive jam" | Press (PRS) + STR | **(0.8×PRS) + (0.2×STR)** |
| "Sticky man coverage / fluid hips" | Man Coverage (MCV) + AGI | Average |
| "Zone instincts / spatial awareness" | ZCV + PRC | Average |
| "Elite ball skills / catch point dominance" | CTH + JMP | Average |

Breakout sub-signal weights:

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.20 |
| School Tier | 0.25 |
| College Production Share (PD + INT market share) | 0.40 |
| Age Trajectory | 0.15 |

Breakout age normalization (base curve — SL-019 modulated, strength 0.30):

| Breakout Age | Base Normalized |
|---|---|
| ≤19.5 | 1.00 |
| 20.5 | 0.75 |
| 21.5 | 0.45 |
| ≥22.5 | 0.15 |

College Production Share normalization (PD + INT market share):

| Market Share | Normalized |
|---|---|
| ≥24% | 1.00 |
| 16% | 0.55 |
| ≤8% | 0.15 |

Age trajectory normalization (base curve — SL-019 modulated, strength 0.30):

| Age | Base Normalized |
|---|---|
| ≤24 | 1.00 |
| 28 (peak) | 0.50 |
| 31 | 0.10 |
| ≥32 | 0.00 |

NGS Coverage Metrics normalization: composite z-score of (target separation, inverse completion % allowed, inverse ADOT allowed) → position percentile → 0–1 scaled. Specific bundle definition deferred to CAL-026 post-live-data.

Three-zone boundaries: Elite ≥ 0.80, Late ≤ 0.40.
School tier: template defaults.

---

#### SAFETY (S)

| Parameter | Value |
|---|---|
| Layer 4 RAS Tier | High |
| Layer 3 Peak Limit | 28 |
| SL-019 | Enabled — three values: BA=0.30, AT=0.30, L3=**0.25** |
| Film Cap | ±5% |
| film_inflection | 0.50 |
| film_steepness | 12.0 |
| film_position_weight | 1.00 |
| RAS Cap | ±8% |
| RAS_inflection | 0.50 |
| RAS_steepness | 10.0 |
| RAS_position_weight schedule | High-tier: 1.00 / 0.50 / 0.10 |
| Layer 3 Buffer | SL-019 amplified: **0.25** × RAS_normalized |
| Breakout Cap | ±5% |
| breakout_inflection | 0.50 |
| breakout_steepness | 11.0 |
| breakout_position_weight | 1.00 |
| NGS Coverage/Range Metrics | Dedicated 0.30-weight analytical anchor (excluded from all other positions) |

Film sub-signal weights:

| Source | Weight | Type |
|---|---|---|
| PFF Safety Overall Grade | 0.35 | Analytical |
| NFL Next Gen Stats Coverage/Range Metrics | 0.30 | Analytical |
| The IDP Show | 0.10 | Subjective |
| The IDP Guru | 0.10 | Analytical |
| The Draft Network | 0.08 | Subjective — **STATIC** |
| Dynasty Nerds | 0.07 | Subjective |

EMA α values:

| Sub-signal | α | Note |
|---|---|---|
| PFF | 0.18 | Dynamic |
| NGS Coverage/Range Metrics | 0.20 | Dynamic |
| IDP Show | 0.30 | Dynamic |
| IDP Guru | 0.20 | Dynamic |
| TDN | N/A | **STATIC** |
| Dynasty Nerds | 0.50 | Dynamic |
| Madden | 0.20 | Dynamic |

Madden attribute mapping (6 rows — 6th covers ball-hawk takeaway archetype):

| Expert Claim | Madden Composite | Formula |
|---|---|---|
| "Elite range / centerfield eraser" | SPD + ACC | Average |
| "Enforcer / box run support" | TAK + HPW + STR | Average |
| "Zone coverage / over-the-top anticipation" | ZCV + PRC | Average |
| "Diagnostic speed / downhill trigger" | PUR + AWR | Average |
| "Slot utility / man match capability" | MCV + AGI | Average |
| "Ball-hawk centerfielder / takeaway producer" | CTH + JMP + AWR | Average |

Breakout sub-signal weights:

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.20 |
| School Tier | 0.25 |
| College Production Share (INT + Tackle market share) | 0.40 |
| Age Trajectory | 0.15 |

Breakout age normalization (same base curve as CB — SL-019 modulated, strength 0.30):

| Breakout Age | Base Normalized |
|---|---|
| ≤19.5 | 1.00 |
| 20.5 | 0.75 |
| 21.5 | 0.45 |
| ≥22.5 | 0.15 |

College Production Share normalization (INT + Tackle market share — lower thresholds than CB):

| Market Share | Normalized |
|---|---|
| ≥20% | 1.00 |
| 14% | 0.55 |
| ≤8% | 0.15 |

Age trajectory normalization (same base curve as CB — SL-019 modulated, strength 0.30):

| Age | Base Normalized |
|---|---|
| ≤24 | 1.00 |
| 28 (peak) | 0.50 |
| 31 | 0.10 |
| ≥32 | 0.00 |

NGS Coverage/Range Metrics normalization at S: composite z-score of (tackle radius percentile, coverage range index, closing speed, snap distribution box vs. deep) → position percentile → 0–1 scaled. Specific S-NGS bundle definition deferred to CAL-028 post-live-data.

Three-zone boundaries: Elite ≥ 0.80, Late ≤ 0.40.

---

#### DEFENSIVE TACKLE (DT)

| Parameter | Value |
|---|---|
| Layer 4 RAS Tier | Hybrid: Medium classification, High-tier RAS treatment (SL-021) |
| Layer 3 Peak Limit | 30 |
| SL-019 | **NOT applied** (Cushion Guard replaces) |
| SL-021 Dynamic PFF α | 0.50 Year 1 → 0.10 Year 2+ |
| SL-021 Cushion Guard | Binary: Raw RAS ≥ 8.00 → 10% flat decay deceleration |
| Film Cap | **±3%** (SL-005) |
| film_inflection | 0.50 |
| film_steepness | **10.0** (SL-005) |
| film_position_weight | **1.00** (RETAINED) |
| RAS Cap | **±8%** (High-tier per SL-021) |
| RAS_inflection | 0.50 |
| RAS_steepness | 10.0 |
| RAS_position_weight schedule | High-tier SL-021: 1.00 / 0.50 / 0.10 |
| Layer 3 Buffer | Cushion Guard (binary, not standard or SL-019) |
| Breakout Cap | ±5% (standard — NOT compressed) |
| breakout_inflection | 0.50 |
| breakout_steepness | 11.0 |
| breakout_position_weight | 1.00 |

Film sub-signal weights:

| Source | Weight | Type |
|---|---|---|
| PFF Interior DL Grade | **0.50** | Analytical — dynamic α per SL-021 |
| The IDP Show | 0.20 | Subjective |
| The IDP Guru | 0.15 | Analytical |
| The Draft Network | 0.08 | Subjective — **STATIC** |
| Dynasty Nerds | 0.07 | Subjective |

EMA α values:

| Sub-signal | α | Note |
|---|---|---|
| PFF | **0.50 Year 1, 0.10 Year 2+** | **DT-unique dynamic α (SL-021)** |
| IDP Show | 0.30 | Dynamic |
| IDP Guru | 0.20 | Dynamic |
| TDN | N/A | **STATIC** |
| Dynasty Nerds | 0.50 | Dynamic |
| Madden | 0.20 | Dynamic |

Madden attribute mapping (5 rows — Space Eater row uses asymmetric weighting):

| Expert Claim | Madden Composite | Formula |
|---|---|---|
| "Interior push / pocket collapser" | PMV + STR | Average |
| "Space eater / double-team anchor" | BSH + STR | **(0.6×BSH) + (0.4×STR)** |
| "Elite lateral quickness / gap shooter" | FMV + ACC | Average |
| "Run stop utility / high tackle rate" | TAK + PRC | Average |
| "Hybrid pass-rush specialist / multi-move disruptor" | PMV + FMV + BSH | Average |

Breakout sub-signal weights:

| Sub-signal | Weight |
|---|---|
| Breakout Age | 0.20 |
| School Tier | 0.20 |
| College Production Share (TFL + Sack market share) | **0.45** (highest of any position) |
| Age Trajectory | 0.15 |

Breakout age normalization (base curve — NO SL-019 modulation):

| Breakout Age | Normalized |
|---|---|
| ≤20.0 | 1.00 |
| 21.0 | 0.75 |
| 22.0 | 0.45 |
| ≥23.0 | 0.15 |

College Production Share normalization (TFL + Sack market share):

| Market Share | Normalized |
|---|---|
| ≥22% | 1.00 |
| 15% | 0.55 |
| ≤8% | 0.15 |

Age trajectory normalization with Cushion Guard (applied if Raw RAS ≥ 8.00):

| Age | Base Normalized |
|---|---|
| ≤26 | 1.00 |
| 30 (peak) | 0.50 |
| 33 | 0.10 |
| ≥34 | 0.00 |

Cushion Guard on Age Trajectory: `cushioned = 0.50 − (0.50 − base) × 0.90`

Three-zone boundaries: Elite ≥ 0.80, Late ≤ 0.40.

---

#### KICKER (K)

| Parameter | Value |
|---|---|
| Layer 4 Output | **1.000 hardcoded** (full structural exclusion — SL-020) |
| film_effective | 1.000 |
| RAS_effective | 1.000 |
| breakout_effective | 1.000 |
| RAS value | Still sourced → Layer 6 tiebreaker only |
| Ranking mechanism | Layer 3 + Layer 5 + Layer 2 carry all kicker differentiation |

---

### 1D. Admin-Tunable Parameters (Standard Admin UI)

All of the following are admin-tunable without code changes. Defaults above are starting values pending empirical calibration:

- All sub-signal weights per position
- S-curve cap, inflection, steepness per component per position
- Madden regulation threshold and blend scaling parameters per position
- Madden attribute mapping table per position
- EMA α value per dynamic sub-signal per position
- film_position_weight, RAS_position_weight, breakout_position_weight per position
- Breakout zone thresholds (Elite_threshold, Late_threshold) per position
- School tier lookup values
- Layer 3 peak limits per position
- Layer 3 decay rate (default 3%)
- Layer 5 cap tier boundary percentages
- Layer 6 positional scarcity matrix
- SL-019 modulator strengths (three values per position)
- Cushion Guard RAS threshold (8.00) and reduction strength (10%)

---

### 1E. Pre-Flight Audit Checklist (Claude Code must implement and run before compile)

```python
def run_preflight_audit(layer_4_output, layer_2_config, player_record):
    # Check 1: Zero scoring leaks
    assert not scoring_dependencies_exist(layer_4_output, layer_2_config), \
        "CRITICAL: Layer 4 references Layer 2 scoring or volume metrics"

    # Check 2: Sigmoid boundary clamping
    assert sigmoid_clamping_guards_present(), \
        "CRITICAL: S-curve lacks explicit min/max clamp"

    # Check 3: Overflow guard on exp()
    assert exp_overflow_guard_present(), \
        "CRITICAL: math.exp(-arg) can overflow at extreme inputs — guard with max(-500, min(500, arg))"

    # Check 4: Veteran static signal immutability
    if player_record.is_veteran:
        assert verify_breakout_signals_match_draft_entry(player_record), \
            "CRITICAL: Veteran breakout signals have drifted from draft-entry values"

    # Check 5: SL-021 DT enforcement
    if player_record.position == "DT":
        assert sl019_not_applied(), "CRITICAL: SL-019 applied at DT — use Cushion Guard only"
        assert cushion_guard_is_binary(player_record.raw_ras), \
            "CRITICAL: Cushion Guard must be a binary RAS >= 8.00 check, not continuous"

    # Check 6: SL-005 compression at LB and DT
    if player_record.position in ["LB", "DT"]:
        assert film_cap == 0.03, "CRITICAL: SL-005 film cap must be 0.03 at LB/DT"

    # Check 7: NGS anchor only at CB and S
    ngs_positions = get_positions_with_ngs_anchor()
    assert set(ngs_positions) == {"CB", "S"}, \
        "CRITICAL: NGS anchor applied outside CB/S — not valid at interior or run-stop positions"

    # Check 8: MFL Player ID string type
    assert isinstance(player_record.mfl_id, str), \
        "CRITICAL: MFL Player ID must be string type with leading zeros preserved"

    # Check 9: Confidence floor produces neutral fallback
    if all_fields_unknown(player_record, component):
        assert component_effective == 1.000, \
            "CRITICAL: All-unknown confidence must resolve to 1.000 neutral"

    return "ALL CHECKS PASSED"
```

---

## Section 2: CORRECTIONS TO GEMINI BLUEPRINTS

These errors are in Gemini's blueprint documents. Implement the correct values, not Gemini's.

### Correction 1 — SL-019 "Strength" is Three Values, Not One (Affects TE, DE, CB, S)

Gemini shows a single "Strength" value per position. The actual architecture has three distinct values per position. The Layer 3 buffer value is always lower than the breakout modulator values:

| Position | Breakout Age | Age Trajectory | Layer 3 Buffer | Gemini (Wrong) |
|---|---|---|---|---|
| TE | 0.35 | 0.35 | **0.30** | "Strength: 0.35" |
| DE | 0.35 | 0.35 | **0.30** | "Strength: 0.35" |
| CB | 0.30 | 0.30 | **0.25** | "Strength: 0.30" |
| S | 0.30 | 0.30 | **0.25** | "Strength: 0.30" |

Using Gemini's value for the Layer 3 buffer at all four positions over-protects late-career age decay: TE/DE by ~17%, CB/S by ~20%.

### Correction 2 — DT EMA Function NameError Bug

Gemini's `sync_defensive_token` function references `player.nfl_experience_years` but `player` is not in the function signature. Fix:

```python
def sync_defensive_token(previous, observation, alpha_tier, position, 
                          nfl_experience_years, season_stage="In-Season"):
    if season_stage == "Off-Season":
        return previous
    if alpha_tier == "PFF_Interior_DL":
        alpha = 0.50 if nfl_experience_years <= 1 else 0.10
    else:
        alpha = alpha_tier
    return (1.0 - alpha) * previous + alpha * observation
```

Note: condition is `<= 1` (Year 0 pre-NFL has no PFF data yet, but covers Year 1); Gemini used `== 1`.

### Correction 3 — SL-OQ-028 Is Deferred, Not Implemented

Gemini's offensive blueprint presents SL-OQ-028 as an active architectural safeguard in the v1.0 QB rubric. It is not. SL-OQ-028 is **deferred to v1.1+**. 

v1.0 QB film component is: PFF Passing (0.45), RSP (0.35), TDN (0.10), Sharp (0.10). Sum = 1.00. No PFF QB Rushing Grade sub-signal in v1.0. Do not add it.

### Correction 4 — Math.exp Overflow Guard Required

The sigmoid throws OverflowError at extreme inputs (arg < ~-709) — the boundary clamp catches the output but the exception fires first. Guard required:

```python
arg = max(-500.0, min(500.0, steepness * (input_val - inflection)))
```

This is already included in the confirmed `calculate_scurve` function in Section 1A.

---

## Section 3: NEEDS CHRISTOPHER'S DECISION BEFORE BUILD

### Decision 3A — WR SL-019 Status (SL-OQ-043)

**Issue:** WR was built before SL-019 was generalized. WR is absent from the SL-OQ-027 applicability resolution table — it was never formally adjudicated.

**WR meets both gating criteria:**
- High-tier RAS ✓
- Athletic profile correlates with longevity (Tyreek Hill, DeSean Jackson archetypes play significantly longer than low-RAS possession receivers) ✓

**Options:**

| Option | What It Does | Trade-off |
|---|---|---|
| A: Keep Disabled | WR v1.0 as-built; no SL-019 | Layer 3 carries all aging. Fast path to build. WR may under-protect elite-athletic veterans. |
| B: Enable at CB/S strength (0.30/0.30/0.25) | Adds breakout + age trajectory modulation and amplified Layer 3 buffer | More protection for Tyreek Hill archetype; adds complexity; requires WR rubric update to v1.1 |
| C: Enable at reduced strength (0.20/0.20/0.15) | Conservative modulation acknowledging WR longevity has more variance than TE/DE | Middle path; unique values; calibration-dependent |

**RESOLVED — Option A. SL-022 assigned.**
WR excluded from SL-019 for v1.0. Layer 3 carries all aging. Options B and C remain documented above — if live testing shows elite-RAS WRs (High-tier, longevity archetypes) are systematically under-protected, revisit as a v1.1 calibration task.

### Decision 3B — SL-OQ-040: PAT/XP Scoring Value

Rulebook does not specify PAT/XP point value. Commissioner confirmation required before any kicker stat processing can include PAT events.

**Blocking for:** kicker Layer 2 stat ingestion completeness.

### Decision 3C — SL-OQ-041: FG 70+ Scoring Threshold

Rulebook specifies FG tiers through 60–69 yards at 6 pts. No value specified for 70+ yards. Commissioner confirmation required.

**Blocking for:** kicker Layer 2 stat processing at extreme distances (unlikely in v1.0 but required before any kicker hits it).

---

## Section 4: DEFERRED — Does Not Block Initial Build

These items are tracked and confirmed deferred. Do not implement in v1.0.

| ID | Topic | Default in v1.0 | Notes |
|---|---|---|---|
| SL-OQ-026 | QB College Offensive Share definition | Starts as % of career games | Validate post-pipeline |
| SL-OQ-018 | RB college workload definition | Touches as % of team rushes + RB targets | Validate post-pipeline |
| SL-OQ-020 | RB touch share placement | Standalone film sub-signal at 0.20 | vs. confidence-modifier; CAL-016 |
| SL-OQ-028 | QB PFF Rushing Grade in film | NOT in v1.0 | v1.1 branch only |
| SL-OQ-031 | LB SL-005 compression depth | Current defaults | Calibration backlog |
| SL-OQ-035/036 | S box/deep branching or role-conditional weighting | Monolithic S rubric | Post-live-data decision |
| SL-OQ-037 | Cushion Guard binary vs. continuous RAS scaling | Binary (threshold 8.00) | CAL-030 |
| SL-OQ-038 | Dynamic PFF α propagation beyond DT | DT only | Session 3 decision pending |
| SL-OQ-039 | DT Year 1 end trigger | End of first NFL regular season | Confirm with implementation |
| CAL-020 | QB SL-018 buffer validity | Apply as-built | RAS doesn't predict QB longevity cleanly |
| CAL-022–032 | All calibration items from Session 2 | Apply defaults | Admin-console task, not code |
| CAL-026 | CB NGS bundle definition | Z-score composite, CAL pending | Post-live-data |
| CAL-028 | S NGS bundle definition | Z-score composite, CAL pending | Post-live-data |

---

## Section 5: NUMBERING STATE AT AUDIT CLOSE

| Series | Next Available | Notes |
|---|---|---|
| SL- (locked decisions) | **SL-022** | Reserved for WR SL-019 decision when made |
| SL-OQ- (open questions) | **SL-OQ-043** | Assigned to WR SL-019 status per this audit |
| CAL- (calibration items) | **CAL-033** | No new CAL items added this audit session |

---

*Built by: Christopher Campbell + Claude (Anthropic)*
*Audit session: June 2026 — Pre-Build Layer 4 Synthesis*
*Verified against: WR, RB, TE, QB, DE, LB, CB, S, DT, K rubrics (all v1.0)*
