# Legacy NFL — Layer 4 Temporary Testing Application
**Version:** 1.0 — June 2026
**Purpose:** Lightweight testing harness for validating Layer 4 scouting engine outputs before building the full application. Claude Code builds this. Christopher operates it. The goal is to verify that the engine produces rankings and component outputs that align with real-world consensus, identify any calibration problems early, and confirm all architectural decisions (SL-005, SL-019, SL-020, SL-021, Cushion Guard, EMA, Lockett Pattern) are executing correctly in code before the full build begins.
**Scope:** This is NOT the production application. It is a testing sandbox. Prioritize correctness of output and debuggability over UI polish. Every component output must be visible and inspectable. Admin parameters must be adjustable live.

---

## Application Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    TESTING HARNESS UI                        │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  Test Module │  │  Test Module │  │   Test Module    │  │
│  │  Selector    │  │  Output View │  │   Admin Panel    │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           │
              ┌────────────┴────────────┐
              │     LAYER 4 ENGINE      │
              │  (film × RAS × breakout)│
              └────────────┬────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
   ┌─────▼──────┐  ┌──────▼─────┐  ┌───────▼──────┐
   │ MFL API    │  │  Manual    │  │  Hardcoded   │
   │ (rosters,  │  │  Entry     │  │  Test Cases  │
   │  players)  │  │  Form      │  │  (rubric-    │
   └────────────┘  └────────────┘  │  verified)   │
                                   └──────────────┘
```

## Data Input Strategy

The full data pipeline (PFF, NGS, RSP, ras.football) is not built yet. This testing harness uses three data sources in order of priority:

1. **MFL API** — rosters, contracts, player metadata, position tags. Pull fresh on load.
2. **Hardcoded test cases** — pre-verified inputs for architectural validation (Section 4). These are fixed reference inputs, not live data.
3. **Manual entry form** — for any player not in the hardcoded set, Christopher enters film grades and RAS manually. The form mirrors the sub-signal structure of each position rubric exactly.

---

## Module 1: 2026 Rookie Layer 4 Rankings

### Purpose

2026 rookies are the purest test of the Layer 4 breakout and RAS components. No NFL film data exists yet. PFF grades and NFL production are Unknown. The engine must rank rookies using only:

- Pre-draft RAS score (ras.football)
- Breakout Age (college)
- School Tier
- College Production / Usage Share (position-specific definition)
- Age Trajectory (relative to position peak limit)
- Film component = TDN static grade only (RSP annual in April, Sharp static entry, no PFF yet)

This is the cleanest early test because the expected inputs are well-defined and the output should roughly correspond to draft capital consensus (first-round picks with elite profiles should score higher than late-round picks with weak profiles).

### Data Requirements

Pull from MFL API: all players flagged as rookies in the current season (`experience_years = 0`). For each rookie, the harness needs:

| Field | Source | Fallback |
|---|---|---|
| Player name | MFL | — |
| MFL player ID (string) | MFL | — |
| Position (true position) | MFL tag + EDGE consensus rule | Manual override |
| Age | MFL | — |
| RAS score | Manual entry (ras.football lookup) | Position-group mean (Unknown flag) |
| Breakout age | Manual entry | 21.0 (Unknown flag) |
| School tier | Manual entry | Group of Five (Unknown flag) |
| College usage/production share | Manual entry (position-specific definition) | Unknown |
| TDN grade | Manual entry (0–100) | Unknown |
| RSP rank (where available) | Manual entry | Unknown |
| Draft round | MFL / manual | — |
| Draft pick number | MFL / manual | — |

### Engine Configuration for Rookie Mode

```
film_confidence     = sourced fields only (TDN primary, RSP if available)
RAS_confidence      = 1.0 if RAS entered; 0.0 if fallback used
breakout_confidence = sourced fields only
RAS_position_weight = Year 0 baseline (maximum tier weight — rookie has no NFL data yet)
PFF                 = Absent (no NFL grades yet)
NFL production      = Absent
Madden              = Absent (rookies not yet in Madden, or unrated)
```

### Output Table (per position group, sortable)

| Column | Description |
|---|---|
| Rank | Engine rank within position group |
| Player | Name |
| Age | Current age |
| Draft Slot | Round + pick |
| RAS | Raw RAS score (or "Imputed" if missing) |
| Breakout Age | College breakout age |
| School | Tier (P4/G5/FCS/Non-FCS) |
| Production Share | College usage/share % |
| Film (TDN) | Normalized TDN score |
| film_eff | Film component multiplier |
| RAS_eff | RAS component multiplier |
| breakout_eff | Breakout component multiplier |
| Layer 4 | Combined output (film × RAS × breakout) |
| Confidence | Breakout confidence score (internal debug only) |
| Flags | Unknown fields listed |

### Comparison Column

Add a **KTC Rank** column (KeepTradeCut dynasty rookie rankings — public data, pull via KTC's public endpoint or manual entry if API access is unavailable). The delta between engine rank and KTC rank shows where the model diverges from market consensus. Large deltas are not necessarily errors — they are calibration signals.

### Position Group Tabs

Separate views for: QB | RB | WR | TE | DE | LB | CB | S | DT | K

Each tab shows only that position group's rookies ranked by Layer 4 output.

### Validation Target

Expected behavior: First-round picks with elite RAS scores and early college breakout ages should cluster at the top of each position group. Late-round picks with late breakout ages and G5/FCS school tiers should cluster at the bottom. Not a perfect 1:1 with draft capital — the engine is measuring traits, not draft capital directly — but the correlation should be visible and directionally correct.

---

## Module 2: Team Composition Analyzer

### Purpose

For each of the 32 Legacy NFL teams, display a team score, positional strength/weakness breakdown, and a league-wide ranking. Tests that the engine produces coherent team-level intelligence — the core value proposition for GMs using the tool.

### Data Requirements

Pull from MFL API: full rosters for all 32 teams including contracts. For each rostered player:

- Layer 4 output (from Module 1 if rookie, from manual entry or hardcoded cases if veteran)
- Layer 3 age_pull (calculated from age and position peak limit)
- Layer 5 cap multiplier (calculated from salary vs. league cap)
- Final Adjusted Score (`Base_Points_stub × age_pull × Layer_4_Output × cap_multiplier`)

**Note on Base_Points stub:** The full Layer 2 scoring matrix is not implemented in this testing harness. Use a position-group median Base_Points proxy for display purposes, flagged clearly as a stub. The test is validating Layer 4 output and the integration of Layers 3–5, not Layer 2 accuracy. Layer 2 build is a separate session.

**Position-group median Base_Points stubs (for display only — replace with real Layer 2 values when built):**

| Position | Stub Base_Points |
|---|---|
| QB | 280 |
| RB | 160 |
| WR | 140 |
| TE | 120 |
| DE | 110 |
| LB | 90 |
| CB | 75 |
| S | 80 |
| DT | 85 |
| K | 100 |

### Team Score Calculation

```
player_score        = stub_base × age_pull × Layer_4_Output × cap_multiplier
position_group_score = sum of player_scores for all rostered players at position group
team_score          = sum of all position_group_scores
```

### Output — Team Dashboard

For each team, display:

**Team Header**
- Team name / GM name (from MFL)
- Overall team score
- League-wide rank (1–32)
- Salary cap used / cap remaining (from MFL)

**Position Group Grid**

| Position | # Rostered | Group Score | League Rank | Tier |
|---|---|---|---|---|
| QB | — | — | — | Elite / Strong / Average / Weak |
| RB | — | — | — | — |
| WR | — | — | — | — |
| TE | — | — | — | — |
| DE | — | — | — | — |
| LB | — | — | — | — |
| CB | — | — | — | — |
| S | — | — | — | — |
| DT | — | — | — | — |
| K | — | — | — | — |

Tier thresholds (based on league percentile): Elite = top 20%, Strong = top 40%, Average = middle 40%, Weak = bottom 20%.

**Strength/Weakness Summary**

Auto-generate two lists:
- Top 3 position group strengths (highest group scores relative to league average)
- Top 3 position group weaknesses (lowest group scores relative to league average)

**Age Curve**

Simple text display showing: "X players past peak limit" and "average age across roster." No chart required for testing phase.

**Contract Risk Flags**

List players where:
- Age > position peak limit AND Layer 4 < 1.00 (declining veteran with weak profile — roster clearance signal)
- Salary in Hot tier AND Layer 4 < 1.02 (overpaid asset)
- Salary in Cold tier AND Layer 4 > 1.05 (undervalued asset — hold)

### League-Wide Rankings View

A separate table listing all 32 teams sorted by total team score. Columns: Rank, Team, GM, Team Score, Best Position, Worst Position, Cap Remaining.

---

## Module 3: Architectural Validation Tests

### Purpose

These are fixed, hardcoded test cases with known inputs and expected output ranges. The engine must produce outputs within the expected range for each test. If any test fails, it is a **code error**, not a calibration question. These are non-negotiable structural requirements.

Claude Code should implement these as automated assertions that run on page load and display pass/fail status with actual vs. expected values.

---

### Test 3A — Lockett Pattern (Layer 4 stays near neutral for declining elite veterans)

**Rule being tested:** Static breakout signals at veteran entry hold Layer 4 near 1.00 even when current film grades decline and age trajectory drops to zero. Layer 3 carries the aging.

**Tyler Lockett — WR, Age 33, Year 11 NFL**

```python
# Breakout component (static, locked at 2015 draft entry)
breakout_age_input: 0.75         # early breakout at UW
school_tier_input:  1.00         # Power Four
college_usage_input: 0.85        # strong target share at UW
age_trajectory_input: 0.00       # 4+ years past peak (peak 29)

# RAS component (Year 2+, position_weight = 0.10)
RAS_normalized: 0.95             # high RAS WR

# Film component (current — declining)
film_composite: 0.48             # near-inflection, reflecting decline

Expected:
  film_effective:     0.97–1.01  (near neutral — declining grades near inflection)
  RAS_effective:      1.001–1.005 (residual contribution, Year 2+ weight)
  breakout_effective: 1.02–1.05  (positive — strong static profile)
  Layer_4_Output:     1.00–1.07  (net near-neutral to mild push)

PASS if Layer_4_Output between 0.99 and 1.08
FAIL if Layer_4_Output < 0.95 (static signals not holding) or > 1.10 (overcounting)
```

**Derek Carr — QB, Age 35, Year 13 NFL**

```python
# Breakout component (static)
breakout_age_input: 0.80         # age 21 at Fresno
school_tier_input:  0.70         # G5
college_usage_input: 0.85        # strong starter share
age_trajectory_input: 0.15       # 3 years past peak (peak 32)

# RAS: forced 1.000 (SL-020)

# Film component (declining)
film_composite: 0.40             # below inflection, hard decline

Expected:
  film_effective:     0.96–0.98  (pull — declining grades)
  RAS_effective:      1.000       (forced — SL-020)
  breakout_effective: 1.03–1.05  (positive — good static profile)
  Layer_4_Output:     0.99–1.04  (mild pull to mild push — net near neutral)

PASS if Layer_4_Output between 0.97 and 1.06
FAIL if Layer_4_Output < 0.94 (static signals not holding) or > 1.08
```

---

### Test 3B — Herbert Pattern (Layer 4 pulls below 1.00 for weak-profile veterans)

**Rule being tested:** Players with genuinely weak rookie-era profiles should produce Layer 4 < 1.00 when profile has declined. Distinguishes from Lockett Pattern — the engine CAN pull below neutral, it just doesn't for players with strong draft-era signals.

**Khalil Herbert — RB, Age 28, Year 6 NFL**

```python
# Breakout component (static, weak profile)
breakout_age_input: 0.20         # late breakout age 22
school_tier_input:  1.00         # P4 (Virginia Tech)
college_usage_input: 0.92        # strong senior-year workload
age_trajectory_input: 0.05       # 3 years past peak (peak 25)

# RAS (Year 2+, position_weight = 0.06)
RAS_normalized: 0.85             # solid RAS but residual weight

# Film component (depth role)
film_composite: 0.38             # below inflection, depth-back profile

Expected:
  film_effective:     0.96–0.98  (pull)
  RAS_effective:      1.001–1.003 (near-zero contribution)
  breakout_effective: 0.99–1.02  (near-neutral — late breakout drags despite other signals)
  Layer_4_Output:     0.96–1.00  (net pull)

PASS if Layer_4_Output between 0.94 and 1.01
FAIL if Layer_4_Output > 1.02 (engine not distinguishing weak profile from Lockett)
```

---

### Test 3C — SL-020 Enforcement (QB and K must output 1.000 at Layer 4 RAS)

**Rule being tested:** Regardless of any QB or K's RAS score — from 0.00 to 9.99 — their RAS component must output exactly 1.000. Layer 4 RAS has zero influence.

```python
QB Test A:
  RAS_normalized: 0.10  (very low — pocket passer)
  Expected RAS_effective: 1.0000
  PASS if RAS_effective == 1.0000 exactly (not 1.0001, not 0.9999)

QB Test B:
  RAS_normalized: 0.99  (elite athlete — Lamar Jackson type)
  Expected RAS_effective: 1.0000
  PASS if RAS_effective == 1.0000 exactly

K Test:
  All three components forced to 1.000
  Expected Layer_4_Output: 1.0000
  PASS if Layer_4_Output == 1.0000 exactly
```

---

### Test 3D — SL-005 Film Compression at LB and DT

**Rule being tested:** Regardless of film composite input, the film component at LB and DT cannot exceed ±3% from neutral. Film at WR/QB/DE etc. can reach ±5%.

```python
LB Film Test:
  film_composite: 1.00  (perfect input — all film sources elite)
  Expected film_raw: ≤ 1.030  (hard cap at +3%)
  PASS if film_raw <= 1.031
  FAIL if film_raw > 1.031 (SL-005 not applied)

  film_composite: 0.00  (zero input)
  Expected film_raw: ≥ 0.970  (hard floor at -3%)
  PASS if film_raw >= 0.969
  FAIL if film_raw < 0.969

DT Film Test: same as LB above.

WR Film Test (control — should NOT be compressed):
  film_composite: 1.00
  Expected film_raw: 1.040–1.050  (cap at +5%)
  PASS if film_raw >= 1.040
  FAIL if film_raw <= 1.031 (SL-005 incorrectly applied to WR)
```

---

### Test 3E — SL-019 Breakout Modulator Math

**Rule being tested:** SL-019 modulation correctly lifts the base breakout age and age trajectory scores when RAS is high. Two DEs with identical base curves but different RAS should produce different breakout component outputs.

**DE Test — Two players, identical college breakout age (21.0 = base 0.15):**

```python
Player A: RAS 9.99 (normalized 0.999)
  modulated = 0.15 + (1.0 - 0.15) × 0.35 × 0.999
  Expected = 0.15 + 0.85 × 0.35 × 0.999 = 0.15 + 0.297 = 0.447
  PASS if modulated between 0.443 and 0.451

Player B: RAS 4.18 (normalized 0.418)
  modulated = 0.15 + (1.0 - 0.15) × 0.35 × 0.418
  Expected = 0.15 + 0.85 × 0.35 × 0.418 = 0.15 + 0.124 = 0.274
  PASS if modulated between 0.270 and 0.278

Delta check: Player A must score higher than Player B.
FAIL if Player B modulated >= Player A modulated.
```

---

### Test 3F — DT SL-021 Cushion Guard

**Rule being tested:** Binary gate. RAS ≥ 8.00 triggers exactly 10% decay deceleration. RAS < 8.00 gets no protection.

**DT Age Trajectory Test (age 32, base_value = 0.20, peak = 0.50):**

```python
Player A: Raw RAS 9.00 (≥ 8.00 — triggers Cushion Guard)
  Expected = 0.50 - (0.50 - 0.20) × 0.90 = 0.50 - 0.27 = 0.23
  PASS if result between 0.228 and 0.232

Player B: Raw RAS 7.99 (< 8.00 — no Cushion Guard)
  Expected = 0.20 (unchanged base)
  PASS if result == 0.20 exactly

Player C: Raw RAS 8.00 (at threshold — triggers)
  Expected = 0.23 (same as Player A)
  PASS if result between 0.228 and 0.232
```

**Layer 3 Cushion Guard Test (age 33, raw_age_pull = 0.913):**

```python
Player A: Raw RAS 9.00 (≥ 8.00)
  Expected = 1.0 - (1.0 - 0.913) × 0.90 = 1.0 - 0.0783 = 0.9217
  PASS if result between 0.920 and 0.924

Player B: Raw RAS 7.50 (< 8.00)
  Expected = 0.913 (unchanged)
  PASS if result == 0.913
```

---

### Test 3G — DT Dynamic Pass-Rush Blend Alpha (SL-021)

> **Naming note (2026-07-24):** this test was originally "DT Dynamic **PFF** Alpha." **PFF is
> RETIRED** (TOS-restricted + paywalled). Only the SL-021 α *schedule* survives; the graded
> `new_observation` it smooths is now the **pfrpassrush pressure composite**, not a PFF grade. The
> spec math below is unchanged — the α schedule and blend are identical — but read `previous_pff` /
> `new_observation` as the SL-021 pass-rush grade.
>
> **How this is realized (WIRED — harness case 3G, `eval3G`, α-SCHEDULE-ONLY).** The SL-021 blend
> mechanic is `defense.SL021Blend(previous, observation, alpha)`; the α schedule is `DT.SL021Alpha`
> (dynamic 0.50→0.10) and the DE control `DE.SL021Alpha` (fixed 0.15). `eval3G` asserts the exact
> values below on the spec's synthetic grades and the DE≠0.75 guard. **No live pass-rush weight
> feeds production scoring yet:** the C-1 evidence (`docs/data-layer/PassRush_C1_Distributions.md`)
> found the pressure composite largely redundant with the locked Madden IDP film anchor at DT
> (r≈0.75) and DE (r≈0.82), so a live DT/DE pressure weight is DEFERRED to the expert-panel gate.
> Case 3G proves the mechanic; it sets no weight and does not touch the locked film budget.

**Rule being tested:** the SL-021 pass-rush-grade blend at DT uses α=0.50 in Year 1, drops to α=0.10 in Year 2+. At no other position does the α switch mid-career.

```python
DT Year 1:
  previous_pff: 0.60, new_observation: 0.90
  Expected new_value: (1 - 0.50) × 0.60 + 0.50 × 0.90 = 0.30 + 0.45 = 0.75
  PASS if result between 0.74 and 0.76

DT Year 2+:
  previous_pff: 0.60, new_observation: 0.90
  Expected new_value: (1 - 0.10) × 0.60 + 0.10 × 0.90 = 0.54 + 0.09 = 0.63
  PASS if result between 0.62 and 0.64

DE Year 1 (control — should use fixed α=0.15, not dynamic):
  previous_pff: 0.60, new_observation: 0.90
  Expected new_value: (1 - 0.15) × 0.60 + 0.15 × 0.90 = 0.51 + 0.135 = 0.645
  PASS if result between 0.64 and 0.65
  FAIL if result == 0.75 (SL-021 dynamic α incorrectly applied to DE)
```

---

### Test 3H — Confidence Floor Returns 1.000

**Rule being tested:** When all fields in a component are Unknown, confidence = 0.00, and the component effective output must be exactly 1.000. The deviation collapses to zero. No distortion regardless of what the raw component calculated.

```python
Film Confidence Floor Test:
  All film sub-signals: Unknown
  film_confidence: 0.00
  film_raw: any value (e.g., 1.04 from partial S-curve calculation)
  film_effective = 1.0 + (1.04 - 1.0) × 0.00 × 1.00 = 1.0 + 0.00 = 1.000
  PASS if film_effective == 1.000

RAS Confidence Floor Test:
  RAS: Absent (player did not test)
  RAS_confidence: 0.00
  RAS_effective must = 1.000
  PASS if RAS_effective == 1.000

Breakout Confidence Floor Test:
  All breakout fields: Unknown
  breakout_effective must = 1.000
  PASS if breakout_effective == 1.000

Full Layer 4 Floor Test:
  All three components have confidence = 0.00
  Layer_4_Output = 1.000 × 1.000 × 1.000 = 1.000
  PASS if Layer_4_Output == 1.000 exactly
```

> **How this is realized (WIRED — harness case 3H, `eval3H`).** The engine has no explicit
> component-confidence multiplier; the floor is the **Data-Parity design**. For **film** and
> **RAS** the presence flag *is* the confidence gate — flagged absent ⇒ effective exactly 1.000,
> stray raw ignored (asserted across every registered rubric). **Breakout** has no whole-component
> flag because age-trajectory is always live, so "all breakout fields Unknown" is the point where
> the three flagged sub-signals are absent (neutral 0.50) **and** age sits at the position peak
> (WR = 29 ⇒ age-trajectory 0.50), driving the composite to the 0.50 inflection ⇒ breakout ==
> 1.000, and with film + RAS also absent, Combined == 1.000 exactly.

---

### Test 3I — NGS Anchor Boundary (CB and S only)

**Rule being tested:** NGS Coverage Metrics earn a dedicated 0.30-weight anchor at CB and S. They are explicitly absent at DE, LB, DT, QB, RB, WR, TE, K. The architecture must enforce this.

```python
CB Film Component:
  NGS sub-signal weight = 0.30
  PASS if NGS weight present and == 0.30

S Film Component:
  NGS sub-signal weight = 0.30
  PASS if NGS weight present and == 0.30

DE Film Component:
  NGS sub-signal must NOT be present
  PASS if NGS weight == 0.00 or field absent
  FAIL if DE film includes any NGS metric at non-zero weight

LB, DT, QB, RB, WR, TE, K: same as DE (NGS absent)
```

---

### Test 3J — EDGE Classification Routing

**Rule being tested:** Pass-rush-primary defenders route through the DE rubric regardless of MFL position tag. Off-ball linebackers route through LB.

```python
Test Player A: MFL tag = "EDGE", pass_rush_snap_share = 75%
  Expected rubric route: DE
  PASS if position_rubric_used == "DE"

Test Player B: MFL tag = "EDGE", pass_rush_snap_share = 25%
  Expected rubric route: LB
  PASS if position_rubric_used == "LB"

Test Player C: MFL tag = "LB", pass_rush_snap_share = 80%
  Expected rubric route: DE (pass-rush primary overrides LB tag)
  PASS if position_rubric_used == "DE"
```

Note: `pass_rush_snap_share` is manual entry in this testing harness. Production pipeline handles this from NGS data.

---

### Test 3K — S-Curve Boundary Safety

**Rule being tested:** The S-curve cannot produce outputs outside [1 - cap, 1 + cap] at any position for any input value. Overflow guard is active.

For each position, run the S-curve with extreme inputs:

```python
Input = 1.00 (theoretical ceiling):
  output must be ≤ 1.0 + film_cap (position-specific)
  PASS if output <= 1.0 + cap

Input = 0.00 (theoretical floor):
  output must be ≥ 1.0 - film_cap
  PASS if output >= 1.0 - cap

Input = 999.00 (overflow test):
  Must not throw error. Must return 1.0 + cap.
  PASS if output == 1.0 + cap (not exception)

Input = -999.00 (underflow test):
  Must not throw error. Must return 1.0 - cap.
  PASS if output == 1.0 - cap (not exception)
```

Run this for: film, RAS, and breakout components at QB, WR, LB, DT, CB, K.

---

### Test 3L — MFL Player ID String Enforcement

**Rule being tested:** MFL player IDs are always strings. Low-index IDs preserve leading zeros.

```python
Test cases:
  Player with MFL ID "0001" — must stay "0001", never 1 (integer)
  Player with MFL ID "0999" — must stay "0999", never 999
  Player with MFL ID "14263" — must stay "14263" (string, no leading zero needed)

PASS if all three IDs are string type and leading zeros are preserved.
FAIL if any ID is integer type or leading zeros are stripped.
```

---

## Module 4: Sensitivity / Admin Tuning Panel

### Purpose

Christopher adjusts a single parameter, the engine immediately re-runs and outputs update. Tests that the admin-tunable architecture (SL-017) works correctly — changes in the admin console affect output without requiring code changes. Also serves as calibration preview: Christopher can see what happens to rankings when he shifts an S-curve cap or an EMA alpha.

### Parameter Controls (all admin-tunable)

Display a panel beside the output table with sliders and number inputs for the currently selected position group:

**Film Component Controls**
- `film_cap`: slider 0.01 – 0.10, step 0.01
- `film_inflection`: slider 0.30 – 0.70, step 0.05
- `film_steepness`: slider 5.0 – 20.0, step 0.5
- Sub-signal weights: number inputs for each source (must sum to 1.00 — show live sum, warn if not 1.00)

**RAS Component Controls**
- `RAS_cap`: slider 0.00 – 0.10
- `RAS_position_weight` (Year 0): slider 0.00 – 1.00
- Madden threshold: slider 0.05 – 0.30
- SL-019 breakout modulator strength (applicable positions only): slider 0.00 – 0.50
- SL-019 Layer 3 buffer strength (applicable positions only): slider 0.00 – 0.40

**Breakout Component Controls**
- `breakout_cap`: slider 0.01 – 0.10
- Sub-signal weights: number inputs (must sum to 1.00)
- Three-zone boundaries: Elite threshold, Late threshold

**DT-Specific Controls (visible only when DT selected)**
- Cushion Guard RAS threshold: number input (default 8.00)
- Cushion Guard decay reduction: slider 0.00 – 0.25 (default 0.10)
- DT PFF α Year 1: slider 0.10 – 0.70 (default 0.50)
- DT PFF α Year 2+: slider 0.05 – 0.20 (default 0.10)

### Sensitivity Tests — Pre-Built Scenarios

Three pre-built scenario buttons that Christopher can fire to see model behavior:

**Scenario A — "Make it Athletic"**
- RAS caps to 110% of defaults at all High-tier positions
- SL-019 modulators to 110% of defaults
- Shows: How much does athletic profile shift rankings if we believe in it more?

**Scenario B — "Make it Film-First"**
- Film caps to 110% of defaults at all positions
- RAS Year 2+ weight drops to 0.05 across all positions
- Shows: If we trust human scouting more, how do rankings shift?

**Scenario C — "Vet-Friendly Settings"**
- Layer 3 decay rate drops from 3% to 2% globally
- Cushion Guard threshold drops to 7.00 at DT
- SL-019 Layer 3 buffer strengths +10% at TE/DE/CB/S
- Shows: How much do late-career veterans benefit from friendlier aging parameters?

Each scenario shows rankings before/after side by side. Reset button returns all parameters to defaults.

---

## Module 5: Cross-Position Value Comparison

### Purpose

The engine must produce outputs that allow meaningful cross-position comparisons — a ranking of all players regardless of position. This module tests whether the combined effect of Layer 3, Layer 4, and Layer 5 produces results that align with dynasty market consensus.

### KeepTradeCut (KTC) Comparison

KTC publishes dynasty value rankings that represent aggregate market consensus across the dynasty fantasy community. Pull KTC values via KTC's public data feed (ktc.app) or manual entry if API access is unavailable.

For each player in the test set:
- Show Engine Rank (overall, all positions combined)
- Show KTC Rank
- Show Delta (Engine Rank - KTC Rank)
- Flag large deltas (|delta| > 20) for Christopher's review

Expected outcome: The engine will not perfectly match KTC because KTC bakes in scoring-specific value (e.g., TE premium leagues affect KTC values) and the engine explicitly separates that. However, the top 20 overall players in the engine should overlap significantly with KTC's top 20, and QBs should rank appropriately given Low-tier RAS exclusion.

### Anomaly Detection

Auto-flag any player where:
- Engine rank is > 30 positions above KTC rank (possible over-scoring)
- Engine rank is > 30 positions below KTC rank (possible under-scoring)
- Layer 4 > 1.08 (extreme push — verify inputs are correct)
- Layer 4 < 0.92 (extreme pull — verify inputs are correct)
- `film_effective == 1.000` AND all film fields are Present (unexpected neutral — likely a data or weight error)

---

## Module 6: Contract Efficiency Stress Test (Layer 5)

### Purpose

Tests that Layer 5 cap tier calculation is working correctly and that changing the league cap value correctly scales all tier boundaries without code changes.

### Tests

**Tier Boundary Calculation:**

```
League Cap: $125M (default 2026)
Expected Cold boundary: $125M × 0.012 = $1.50M
Expected Hot boundary:  $125M × 0.048 = $6.00M

Test: player salary $1.49M → cap_tier = Cold  → multiplier = 1.15
Test: player salary $1.50M → cap_tier = Neutral → multiplier = 1.00
Test: player salary $6.00M → cap_tier = Neutral → multiplier = 1.00
Test: player salary $6.01M → cap_tier = Hot    → multiplier = 0.85

PASS for all four tier boundary assignments.
```

**Automatic Scaling Test (DECISION-009):**

```
Change league cap to $150M in admin panel.
Expected new Cold boundary: $150M × 0.012 = $1.80M
Expected new Hot boundary:  $150M × 0.048 = $7.20M

Verify player at $1.75M:
  At $125M cap → $1.75M > $1.50M → Neutral
  At $150M cap → $1.75M < $1.80M → Cold tier
  PASS if tier changes from Neutral to Cold when cap changes to $150M.
```

**Percentage vs. Dollar Display:**

UI must show both: `"Salary: $1.75M | Cap %: 1.17% | Tier: Cold"`

Switching cap input updates % column live without changing dollar amounts.

---

## Module 7: Historical Preservation Check (DECISION-010)

### Purpose

Ensures that changing any engine parameter does NOT retroactively change a player record that was scored under prior settings. Simpler validation — the full historical archive is not built yet, but the data model must be correct.

### Test

```
Create two player snapshots:

Snapshot A: Scored with film_cap = 0.05, Layer 4 = 1.045
  scoring_config_id: "config_2026_v1"
  Stored in player record.

Change film_cap to 0.04 in admin panel.

Snapshot B: Same player re-scored, Layer 4 = 1.036
  scoring_config_id: "config_2026_v2"

PASS if:
  - Snapshot A record still shows Layer 4 = 1.045 and config "config_2026_v1"
  - Snapshot B is a NEW record, not an overwrite
  - Viewing "2026 v1 score" still returns 1.045
  - Viewing "2026 v2 score" returns 1.036

FAIL if Snapshot A's Layer 4 has changed to 1.036.
```

---

## UI Specification

### General Requirements

- Single-page application with tab navigation for modules
- No framework required — React or plain HTML/JS both acceptable
- Color scheme: Dark background (`#1a1a2e`), white text, red accent (`#e63946`) consistent with the existing project aesthetic
- Tables must be sortable by any column (click header)
- Tables must be filterable by position group
- All Layer 4 outputs color-coded: > 1.05 = green, 1.01–1.05 = light green, 0.99–1.01 = white (neutral), 0.94–0.99 = light red, < 0.94 = red
- Pass/Fail test results: green checkmark for PASS, red X for FAIL, with actual vs. expected values shown

### Navigation Tabs

```
[Module 1: Rookie Rankings] [Module 2: Team Analyzer] [Module 3: Arch Tests]
[Module 4: Sensitivity] [Module 5: Cross-Position] [Module 6: Layer 5] [Module 7: History]
```

### Admin Panel (Persistent Right Sidebar)

Always visible. Shows current parameter values for the selected position group. Sliders update output in real time. A "Reset to Defaults" button per section. A "Save Config" button that stores the current parameter set with a label.

### Player Detail Modal

Click any player row → modal opens showing:

- **Film component:** each sub-signal's normalized value, weight, weighted contribution, Madden regulation status (regulated/not regulated), confidence flag
- **RAS component:** raw RAS, normalized, S-curve output, position_weight at current career stage, confidence flag
- **Breakout component:** each sub-signal raw value, normalized value (base + SL-019 modulated if applicable), weight, weighted contribution, confidence flag
- **Final Layer 4 calculation:** film × RAS × breakout with full chain shown
- **Layer 3:** age_pull calculation with Cushion Guard status (if DT)
- **Layer 5:** salary, cap %, tier, multiplier
- **Final Adjusted Score**

### Data Entry Form

Manual player entry form (for players not in hardcoded set or MFL):
- Position selector → loads position-specific sub-signal fields
- One input per sub-signal (labeled with source name)
- Unknown checkbox per field (sets confidence weight to 0, uses neutral fallback)
- Calculate button → runs engine and adds player to current module's table
- RAS lookup link: opens ras.football in new tab

---

## Data Requirements Summary

| Data Type | Source | How Obtained |
|---|---|---|
| Roster and contracts | MFL API | API call on load |
| Player metadata (age, exp, position tag) | MFL API | API call on load |
| RAS scores | ras.football | Manual entry (paste from lookup) |
| TDN grades | The Draft Network | Manual entry |
| PFF grades (veterans only) | PFF | Manual entry |
| KTC dynasty values | KTC public data | Auto-pull or manual entry |
| Breakout age | Manual entry | From college research |
| School tier | Manual entry | From college lookup |
| College production share | Manual entry | From college stats |
| EDGE snap share | Manual entry | From NGS or PFR |

### MFL API Calls Required

```
GET /export?TYPE=rosters&L={league_id}&JSON=1
GET /export?TYPE=players&JSON=1            (full player database — cache daily)
GET /export?TYPE=contracts&L={league_id}&JSON=1
GET /export?TYPE=league&L={league_id}&JSON=1  (for cap amount)
```

MFL League ID: Christopher supplies before first API session (OQ-001).

All API calls are server-side. MFL blocks cross-origin browser calls.

---

## Success Criteria

The testing harness is doing its job if Christopher can answer yes to all of these:

1. **Rookie rankings pass the smell test.** First-round picks with elite athletic profiles rank above late-round picks with weak profiles at each position group. The ranking isn't perfectly correlated with draft capital — the engine is measuring traits, not draft consensus — but it's directionally coherent.

2. **Team composition differentiates well-built from poorly-built rosters.** The 32-team ranking produces a spread, not a cluster. Teams with obvious dynasty strength (multiple elite young players, cap flexibility) should rank higher than teams in clear rebuilds.

3. **All 12 architectural validation tests pass.** Every test in Module 3 shows a green checkmark. Zero failures.

4. **Sensitivity panel produces intuitive results.** When Christopher moves the `film_cap` slider up, film-dominant players rise. When he increases the SL-019 modulator, elite-RAS aging veterans gain more protection. The engine responds predictably to parameter changes.

5. **Cross-position comparison is defensible.** When Christopher looks at the overall top 20, he can explain why each player is there. Players who appear wrong prompt a data investigation, not a code investigation.

6. **No silent failures.** If a field is Unknown, the UI shows it. If a player is missing RAS, the table flags it. If confidence is low, the row is marked. The engine never produces a false-precision output without surfacing the data gap.

---

## Technical Notes for Claude Code

- **Layer 2 Base_Points is a stub in this harness.** Use position-group median values from the stubs table. Do not build the full Layer 2 scoring matrix — that is a separate build session.
- **Build the Layer 4 engine as a standalone module.** All nine position rubric parameter sets should be stored as JSON config objects (one per position), loaded at startup. Admin panel writes to these configs in memory. "Save Config" persists to local storage.
- **The S-curve function should exist once** and be called by all components. No copy-pasted sigmoid implementations.
- **SL-019 modulation should be a single helper function** called by the breakout component where applicable. Not implemented inline per sub-signal.
- **Cushion Guard should be a single binary function** called by both Layer 3 and the DT breakout Age Trajectory sub-signal.
- **Dynamic DT PFF alpha** should be determined by checking `nfl_experience_years` at the time of each EMA blend call. The alpha is not stored — it is computed each time.
- **All position rubric parameter sets are in `Layer4_PreBuild_Audit.md` (Section 1C).** Use that document as the implementation reference. Do not derive values from Gemini's blueprints.
- **MFL Player IDs:** enforce string type at the API ingestion layer. Type-check on every write to player records.
- **Confidence scores:** calculate internally, display in the player detail modal, never show in the main ranking tables.

---

*Built by: Christopher Campbell + Claude (Anthropic)*
*Pre-build testing harness specification — June 2026*
*Companion to: `Layer4_PreBuild_Audit.md`*
