# Legacy NFL — Universal Rubric Template
**Version:** 1.2 — June 2026
**Status:** Session 3 audit pass. All 10 position rubrics complete at v1.0. Template updated with SL-018/019/020/021 mechanics and NGS anchor pattern.
**Companion:** Engine_Specification.md Layer 4 is the authoritative implementation spec. This document is the structural skeleton that every position rubric fills in with position-specific values.

---

## Purpose

This is the blank form. Every position group rubric — wide receiver, running back, tight end, quarterback, defensive end/edge, linebacker, cornerback/safety, defensive tackle, kicker — fills in the same structural fields with position-specific values.

The template defines:
- What structural elements every component has
- What logic every component runs
- What signals every component requires
- What outputs every component produces
- What position rubrics must specify

The template does not define:
- Position-specific weights, thresholds, or numerical defaults
- Position-specific normalization functions
- Position-specific Madden attribute mappings

Those live in position rubrics.

---

## Architecture Overview

Layer 4 has three multiplicative components:

```
Layer_4_Output = film_effective × RAS_effective × breakout_effective
```

Each component:
- Is a multiplier centered on 1.00
- Bounded by its own S-curve asymptote (Shape B)
- Scaled by per-component confidence and per-component position weight

Components combine multiplicatively. No overall Layer 4 cap. Each component's cap acts as the natural bound.

---

## Component 1: Film Component

The film component is the engine's integration point for expert evaluation. Two anchors (RSP subjective + PFF analytical) carry the highest weights. Other approved sources contribute as directional modifiers. Madden regulates subjective sub-signals at the attribute level.

### Sub-signal categories

**Subjective sub-signals** — qualitative expert descriptions vulnerable to enthusiasm and over-reporting. Regulated by Madden via Approach D.
- RSP qualitative claims (primary subjective anchor)
- TDN scouting language
- IDP Show analyst takes
- Sharp Football qualitative analysis
- Dynasty Nerds prospect analysis (where applicable)
- DLF analyst rankings (where applicable)

**Analytical sub-signals** — numerical methodologies that self-regulate. Not regulated by Madden.
- PFF grades (primary analytical anchor)
- IDP Guru weekly grades
- NFL Next Gen Stats metrics
- NFL production data
- FantasyPros snap counts

**NGS Coverage Metrics Anchor — CB and S only**

At CB and S, NGS coverage metrics earn a dedicated sub-signal anchor at weight 0.30. Coverage-grade tracking data (separation allowed, completion % allowed, target rate, etc.) is the primary post-draft analytical signal at these positions — measuring the core skill directly.

This anchor is **not available at other positions**. Interior defenders, pass rushers, offensive positions, and kickers lack equivalent coverage-grade tracking data. Applying it outside CB/S introduces noise. Position rubrics at LB, DE, DT, QB, RB, WR, TE, K must NOT include an NGS coverage anchor.

### Madden regulation (Approach D at attribute level)

For each subjective sub-signal in the position rubric:

```
1. Position rubric specifies the Madden sub-attribute (or composite) 
   that corresponds to the expert claim.

2. Madden sub-attribute is normalized within position group (relative 
   rating across all true-position peers).

3. Approach D regulation applies:
     If disagreement < madden_threshold:
       claim_weight = full position rubric weight (no regulation)
     Else:
       claim_weight = blend(rubric_weight, madden_implied_weight)
       blend strength scales with magnitude of disagreement beyond threshold
```

`madden_threshold` and the blend scaling function are admin-tunable, with defaults per position rubric.

### Madden attribute mapping table

Every position rubric MUST include this table:

| Subjective Expert Claim | Madden Sub-Attribute / Composite | Composite Formula (if applicable) |
|------------------------|----------------------------------|-----------------------------------|
| (e.g., "fast")         | (e.g., Speed)                    | (e.g., direct)                    |
| (e.g., "elite hands")  | (e.g., Catching)                 | (e.g., direct)                    |
| (e.g., "good route runner") | (e.g., Route Running)       | (e.g., direct)                    |
| (e.g., "high football IQ") | (e.g., Awareness + Play Recognition + Pursuit) | (e.g., average of three) |
| ...                    | ...                              | ...                               |

The mapping itself is admin-tunable through the admin console. Rubric provides defaults.

### Aggregation (Approach A)

After Madden regulation (where applicable), all sub-signals feed into Approach A:

```
For each sub-signal in film component:
  normalized_signal = position_specific_normalization(raw_signal)  # → 0.0–1.0
  weighted_contribution = normalized_signal × rubric_weight
  (subjective sub-signals: rubric_weight is post-Madden-regulation weight)

composite_input = Σ(weighted_contribution across all film sub-signals)
film_raw = S_curve(composite_input, film_inflection, film_steepness, film_cap)
```

S-curve cap default range: 3–5% per SL-002. Admin-tunable.

### Scaling

```
film_effective = 1.0 + (film_raw - 1.0) × film_confidence × film_position_weight
```

### Output

```
film_multiplier ∈ [1.0 - film_cap, 1.0 + film_cap]
```

### What the position rubric MUST specify for the film component

- [ ] Sub-signal weights for RSP, PFF, and each additional approved film source
- [ ] Madden attribute mapping table (claim → Madden sub-attribute or composite)
- [ ] Madden regulation threshold default value (`madden_threshold`)
- [ ] Madden blend scaling function default parameters
- [ ] `film_position_weight` default value (default 1.0 unless position warrants reduction per SL-005)
- [ ] `film_cap` default value (within 3–5% range per SL-002, admin-tunable)
- [ ] S-curve `film_inflection` and `film_steepness` defaults
- [ ] Sub-signal normalization functions (one per sub-signal)
- [ ] EMA blend rate `α` for each dynamic film sub-signal

---

## Component 2: RAS Component

The RAS component is a single-signal multiplier. RAS score normalized within true position group, translated through the position's S-curve, scaled by confidence and position weight tier.

### Position weight tier assignment (per SL-004)

| Tier   | Positions          | Layer 4 Role                                                                 |
|--------|--------------------|------------------------------------------------------------------------------|
| High   | WR, TE, DE, CB, S  | Meaningful push/pull contribution                                            |
| Medium | RB, LB             | Secondary signal                                                             |
| Low    | QB, K              | Forced 1.000 at Layer 4 per SL-020; RAS value used in Layer 6 tiebreaker    |
| Hybrid | DT (SL-021)        | Medium film (SL-005 compression) + High-tier RAS treatment; see SL-021      |

TE amendment: Moved from Medium to High-tier (Session 2).
DT hybrid: SL-021 resolution package — Medium film, High-tier RAS, dynamic PFF α, Cushion Guard.
For Low-tier positions (QB, K), RAS component is forced to exactly 1.000 per SL-020. K has all three Layer 4 components forced to 1.000. RAS value still recorded for Layer 6 tiebreaker at both positions.

**SL-018 — RAS Position Weight Time-Decay Schedule**

Every position rubric's RAS_position_weight follows this schedule by career stage:

| Tier   | Year 0 (draft) | Year 1 | Year 2+ |
|--------|----------------|--------|---------|
| High   | 1.00           | 0.50   | 0.10    |
| Medium | 0.60           | 0.30   | 0.06    |
| Low    | 0.00           | 0.00   | 0.00    |

DT (Hybrid) uses the High-tier schedule for its RAS component per SL-021. Year 0 = draft year. Year 1 = first full NFL season completed. Year 2+ = subsequent seasons.

**SL-019 — RAS Modulator Applicability**

SL-019 modulation applies at: **TE, DE, CB, S** only (High-tier positions with established athletic-profile-to-longevity-arc predictive relationship).

Excluded: QB and K (SL-020), RB and LB (Medium-tier, scheme-driven longevity), DT (Cushion Guard per SL-021 is the DT-unique replacement), WR (SL-022 — excluded for v1.0, Layer 3 carries aging; calibration revisit flagged for v1.1).

Position rubrics at TE, DE, CB, S MUST specify three SL-019 values (see Breakout Component section).

**SL-020 — Low-Tier RAS Exclusion**

QB and K rubrics MUST force RAS component to exactly 1.000. K rubric MUST force all three Layer 4 components to exactly 1.000. No sub-signal calculation needed — the output is structurally locked.

### Aggregation

```
RAS_normalized = position-group relative rating of player's RAS
RAS_raw = S_curve(RAS_normalized, RAS_inflection, RAS_steepness, RAS_cap)
```

### Scaling

```
RAS_effective = 1.0 + (RAS_raw - 1.0) × RAS_confidence × RAS_position_weight

RAS_confidence: 1.0 if RAS Present, 0.0 if Absent (player did not test)
                Position-group mean fallback applied at Layer 1 when missing
```

### Output

```
RAS_multiplier ∈ [1.0 - RAS_cap, 1.0 + RAS_cap]
```

### What the position rubric MUST specify for the RAS component

- [ ] Position weight tier (High / Medium / Low per SL-004)
- [ ] `RAS_position_weight` default value (scaled by tier)
- [ ] `RAS_cap` default value (scaled by tier — wider for High, narrower for Medium, near-zero for Low)
- [ ] S-curve `RAS_inflection` and `RAS_steepness` defaults
- [ ] RAS normalization function (typically relative-to-position-group; specify reference group if non-standard)

---

## Component 3: Breakout Component

The breakout component aggregates four mechanical college-era inputs. No Madden regulation (mechanical not subjective). No NIL recency engineering (per SL-003 as amended and SL-014).

### Sub-signal inputs (per SL-003)

1. **Breakout age** — continuous numerical (years at which player produced at college level)
2. **School tier** — categorical (Power Four / Group of Five / FCS / Non-FCS per SL-001)
3. **College usage rate** — continuous numerical (% of team production)
4. **Age trajectory** — derived position on age curve relative to Layer 3 peak limit for the position

### Normalization

Each sub-signal normalized to 0.0–1.0 scale via position-specific function:

- **Breakout age:** position rubric defines mapping from raw years to normalized score (e.g., WR breakout age 18 = 1.0, 21+ = 0.4)
- **School tier:** lookup table (e.g., Power Four = 1.0, Group of Five = 0.7, FCS = 0.4, Non-FCS = 0.2 — exact values per rubric)
- **College usage rate:** position rubric defines mapping from raw % to normalized score
- **Age trajectory:** derived from Layer 3 peak limit — e.g., 4+ years until peak = 1.0, at peak = 0.5, 4+ years past peak = 0.0

### Aggregation (Approach A)

```
composite_input = Σ(normalized_signal × weight) across all four sub-signals
breakout_raw = S_curve(composite_input, breakout_inflection, breakout_steepness, breakout_cap)
```

### Three-zone classification (per SL-003)

The S-curve output produces natural zones:
- **Elite zone:** breakout_raw at upper plateau (≥ Elite_threshold)
- **Average zone:** breakout_raw in middle range (Late_threshold < breakout_raw < Elite_threshold)
- **Late zone:** breakout_raw at lower plateau (≤ Late_threshold)

Zone thresholds per position rubric, admin-tunable. Zones are surfaced downstream for context but do not change the multiplier value — the multiplier is the S-curve output regardless of zone.

### Scaling

```
breakout_effective = 1.0 + (breakout_raw - 1.0) × breakout_confidence × breakout_position_weight
```

### SL-019 Breakout Modulator (TE / DE / CB / S only)

At applicable positions, RAS modulates two of the four breakout sub-signals. The un-modulated composite is replaced by the modulated composite before the S-curve is applied.

**Breakout age modulator:**
```
modulated_breakout_age = base + (1.0 - base) × breakout_age_modulator_strength × RAS_normalized
```

**Age trajectory modulator:**
```
modulated_age_trajectory = base + (1.0 - base) × age_trajectory_modulator_strength × RAS_normalized
```

Modulator strengths are position-specific and admin-tunable. Position rubrics at TE, DE, CB, S MUST specify both. At all other positions, the un-modulated sub-signal values pass through directly.

### Veteran behavior (per SL-007, SL-010)

All four breakout sub-signals are static. Set at draft entry and unchanged afterward. Veteran breakout component holds at rookie-time value. Injuries, redshirt time, missed snaps do not reset the model.

Age trajectory updates as the player ages but is captured automatically via the Layer 3 peak limit comparison.

### Output

```
breakout_multiplier ∈ [1.0 - breakout_cap, 1.0 + breakout_cap]
```

### What the position rubric MUST specify for the breakout component

- [ ] Sub-signal weights (breakout age, school tier, college usage, age trajectory)
- [ ] `breakout_position_weight` default value (default 1.0 unless warranted otherwise)
- [ ] `breakout_cap` default value
- [ ] S-curve `breakout_inflection` and `breakout_steepness` defaults
- [ ] Breakout age normalization function (position-specific)
- [ ] School tier lookup values (or confirmation of template defaults)
- [ ] College usage rate normalization function (position-specific)
- [ ] Age trajectory normalization (relative to Layer 3 peak limit)
- [ ] Three-zone boundary thresholds (Elite_threshold, Late_threshold)

---

## Cross-Component Logic

### Multiplicative combination

```
Layer_4_Output = film_effective × RAS_effective × breakout_effective
```

No overall Layer 4 cap. Combined swing equals product of component caps.

### Confidence calculation per component

```
film_confidence     = Σ(film_field_present_weights)     / Σ(film_field_expected_weights)
RAS_confidence      = 1.0 if Present else 0.0
breakout_confidence = Σ(breakout_field_present_weights) / Σ(breakout_field_expected_weights)
```

Field weights inside each confidence calculation can be uniform (each field counts as 1.0 toward expected) or non-uniform (high-importance fields count more toward expected). Position rubrics may specify non-uniform field weighting where some sub-signals are more critical than others to a position's evaluation.

### Position weight per component

Each component has its own `position_weight` value specified per position rubric. Defaults to 1.0 unless the position calls for reduction:

- DT and LB carry reduced film weight per SL-005 (IDP data gap)
- Low-tier RAS positions (DT, QB, K) carry near-zero RAS weight
- Position-specific reductions documented in the rubric with rationale

---

## Data Flag Mechanics (per SL-006)

Each input field carries one of three states:

| State   | Meaning                                            | Engine Behavior                                  |
|---------|----------------------------------------------------|--------------------------------------------------|
| Present | Data exists and has been sourced                   | Use raw value                                    |
| Absent  | Data will never exist (player didn't test, N/A)    | Use positional fallback (Layer 1)                |
| Unknown | Data not yet collected, may exist                  | Use neutral placeholder; flag for recalculation  |

Confidence score per SL-011 is an internal engine flag — used to scale component deviations from 1.00, not surfaced in the user interface.

---

## Running Blend (per SL-007, Option A EMA)

### Static sub-signals (set once, never update)

- School tier
- Pre-NFL breakout age
- College usage rate
- RAS score
- Age trajectory (technically derived from age which updates, but the trajectory function handles this automatically)

### Dynamic sub-signals (update repeatedly, EMA blend)

- PFF grades (weekly during NFL season)
- IDP source weekly grades (IDP Guru, IDP Show)
- RSP annual refresh (April publication)
- NFL production data (game-by-game accumulation)
- Madden ratings (multiple updates per NFL season)
- Other expert source updates as they publish

### EMA formula

```
new_value = (1 - α) × previous_value + α × new_observation
```

`α` per dynamic sub-signal per position rubric. Lower α = slower blend (history weighted). Higher α = faster blend (recent observation weighted).

### Initial value rule

First observation becomes the sub-signal's starting value. EMA blending begins on the second observation. Position rubrics may override for specific sub-signals.

### Offseason hold

When no new observations arrive for a dynamic sub-signal (e.g., NFL offseason for PFF), the value holds. No decay during the silent window. Blending resumes when new data arrives.

### Season transition behavior

When a new NFL season begins, dynamic sub-signals default to CONTINUATION — the prior season's final value blends with the new season's first observation per the standard EMA formula. Career production patterns matter for dynasty evaluation, so accumulated PFF grades and NFL production from prior seasons remain influential.

Position rubrics may override the default to RESET for specific sub-signals where the prior season's value is genuinely less relevant to the new context (e.g., a sub-signal tracking scheme fit in a specific year's offensive system, where a coaching change makes prior data noise). Override and rationale documented per sub-signal in the rubric.

Each dynamic sub-signal in the rubric specifies one of:
- CONTINUATION (default — no specification needed)
- RESET — with rationale

### Dynamic PFF EMA Alpha (DT only — SL-021; see SL-OQ-038 for cross-rubric question)

At DT, the PFF EMA blend rate α is not fixed — it varies by career stage:
- **Year 1 α = 0.50** — aggressive blend to rapidly displace rookie-era RAS dominance with real NFL film data
- **Year 2+ α = 0.10** — slow blend for veteran signal stability

The Year 1 / Year 2+ transition fires at the end of the first full NFL regular season. α is computed at the time of each EMA blend call — it is not stored.

All other positions: fixed α per rubric. SL-OQ-038 asks whether this mechanism should propagate cross-rubric. Not adopted elsewhere until Christopher resolves SL-OQ-038.

### Late-Career Cushion Guard (DT only — SL-021)

At DT, when Raw RAS ≥ 8.00, Layer 3 age_pull applies a 10% decay velocity reduction:
```
cushioned_age_pull = 1.0 − (1.0 − standard_age_pull) × 0.90
```
When Raw RAS < 8.00: standard age_pull applies unchanged.

The Cushion Guard also applies to the breakout component's Age Trajectory sub-signal at DT beyond the peak limit (same formula applied to the trajectory sub-signal value).

Threshold = 8.00 (binary, admin-tunable). See SL-OQ-037 for the open question on continuous RAS scaling as an alternative.

SL-019 Layer 3 buffer is the analogous mechanism at TE/DE/CB/S. At DT, Cushion Guard replaces SL-019 entirely — running both would double-protect elite-RAS veterans.

---

## Multi-Position Player Blending (per SL-009)

When a player qualifies for multiple position scores:

```
1. Calculate Layer 4 for each applicable position separately, using each 
   position's rubric.

2. Blend results:
   Layer_4_blended = Σ(Layer_4_per_position × role_weight)
   role_weight = snap_share_at_position / total_snap_share
```

If snap share data unavailable: depth chart designation determines primary position (weight 1.0) vs secondary (weight 0.0).

Position rubrics may override blend logic where the position warrants (e.g., a CB/S hybrid may use a custom blend that weighs both equally regardless of snap share). Overrides documented in the rubric with rationale.

---

## Admin-Tunable vs Structural Parameters (per SL-017)

### Admin-tunable (standard admin UI)

| Category | Parameters |
|----------|-----------|
| Component weights | All sub-signal weights per position |
| S-curve shape | Cap, inflection, steepness per component per position |
| Madden regulation | Threshold, blend scaling parameters per position |
| Madden mappings | Attribute mapping table per position |
| EMA blending | α value per dynamic sub-signal per position |
| Position weights | film_position_weight, RAS_position_weight, breakout_position_weight per position |
| Breakout zones | Elite_threshold, Late_threshold per position |
| Normalization | School tier values, normalization function parameters |

### Structural (code-locked)

- Multiplicative combination of three Layer 4 components
- S-curve mechanic (Shape B / sigmoid family)
- Approach A aggregation structure (normalize → weighted sum → S-curve)
- Approach D Madden regulation mechanism (threshold + gradient blend)
- Madden regulates subjective sub-signals only (per SL-016)
- Per-component scaling (confidence × position weight on deviation)
- EMA blend mechanism (Option A)
- Three-state data flag mechanic (Present / Absent / Unknown)
- Confidence as internal flag (not UI-surfaced per SL-011)
- Static vs dynamic sub-signal categorization
- Initial value and offseason hold rules

### Developer-mode access

All structural parameters above are tunable via developer-mode admin access for advanced calibration. Standard admin UI does not expose them — protecting league users from breaking engine mechanics while preserving full calibration capability for maintainers.

---

## What Each Position Rubric MUST Specify (Checklist Summary)

Every position rubric is incomplete until all of these are specified or explicitly deferred with rationale.

### Film Component
- [ ] Sub-signal weights (RSP, PFF, and additional sources)
- [ ] Madden attribute mapping table
- [ ] Madden threshold default
- [ ] Madden blend scaling parameters
- [ ] film_position_weight default
- [ ] film_cap default
- [ ] film S-curve inflection and steepness defaults
- [ ] Sub-signal normalization functions
- [ ] EMA α per dynamic film sub-signal

### RAS Component
- [ ] Position weight tier (High / Medium / Low / Hybrid per SL-021)
- [ ] SL-018 decay schedule confirmed (table above — no per-rubric override needed unless position is Hybrid)
- [ ] SL-020 exclusion applied if QB or K (RAS component forced 1.000; K all three components forced 1.000)
- [ ] RAS_position_weight Year 0 default (from SL-018 schedule — specify if non-standard)
- [ ] RAS_cap default (scaled by tier)
- [ ] RAS S-curve inflection and steepness defaults
- [ ] RAS normalization function

### Breakout Component
- [ ] Sub-signal weights (breakout age, school tier, college usage, age trajectory)
- [ ] SL-019 applicability stated (applicable: TE/DE/CB/S; excluded: all others — with reason)
- [ ] If SL-019 applicable: breakout_age_modulator_strength specified
- [ ] If SL-019 applicable: age_trajectory_modulator_strength specified
- [ ] breakout_position_weight default
- [ ] breakout_cap default
- [ ] breakout S-curve inflection and steepness defaults
- [ ] Breakout age normalization function
- [ ] School tier lookup values (or template default confirmation)
- [ ] College usage rate normalization function
- [ ] Age trajectory normalization
- [ ] Three-zone boundary thresholds

### Layer 3 / Veteran Mechanics
- [ ] SL-019 Layer 3 buffer specified if TE/DE/CB/S (buffer_strength)
- [ ] SL-021 Cushion Guard documented if DT (threshold = 8.00, reduction = 10% — note if non-default)
- [ ] Dynamic PFF α documented if DT (Year 1 α, Year 2+ α, transition trigger)

### Notes
- [ ] Position-specific overrides or departures from universal logic (with rationale)
- [ ] IDP data gap flag (DT, LB per SL-005)
- [ ] NGS Coverage Metrics anchor present only if CB or S (0.30 weight)
- [ ] NGS Coverage Metrics anchor confirmed absent if not CB or S
- [ ] Any position-specific calibration priorities flagged for testing

---

## Veteran Scouting Layer Placeholder (per SL-008)

**Status: Open scope. Dedicated session pending after all nine position rubrics complete.**

The veteran layer must fit the same architectural shape as the rookie model:
- Same three-component multiplicative structure
- Same Approach A aggregation
- Same Approach D Madden regulation
- Same S-curve and scaling mechanics

What changes for veterans (already partially captured in component sections):
- Breakout component sub-signals are STATIC. Locked at draft entry. Veteran breakout multiplier holds at rookie-time value.
- Dynamic activity centers on the film component: PFF grades, NFL production, in-season expert refresh, Madden updates.
- Age trajectory updates automatically through Layer 3 peak limit comparison.

Position rubrics flag any veteran-specific signals their position requires. The veteran layer extension session will consolidate those flags into a coherent veteran model.

---

## Position Group Build Order (Deliverable 2 — COMPLETE)

All 10 position rubrics built and locked at v1.0. Session 1 built WR/RB/TE/QB; Session 2 built DE/LB/CB/S/DT/K.

| # | Position | File | Status | Session |
|---|---|---|---|---|
| 1 | Wide Receiver | WR_Rubric.md | v1.0 locked | Session 1 |
| 2 | Running Back | RB_Rubric.md | v1.0 locked | Session 1 |
| 3 | Tight End | TE_Rubric.md | v1.0 locked | Session 1 |
| 4 | Quarterback | QB_Rubric.md | v1.0 locked | Session 1 |
| 5 | Defensive End | DE_Rubric.md | v1.0 locked | Session 2 |
| 6 | Linebacker | LB_Rubric.md | v1.0 locked | Session 2 |
| 7 | Cornerback | CB_Rubric.md | v1.0 locked | Session 2 |
| 8 | Safety | S_Rubric.md | v1.0 locked | Session 2 |
| 9 | Defensive Tackle | DT_Rubric.md | v1.0 locked | Session 2 |
| 10 | Kicker | K_Rubric.md | v1.0 locked | Session 2 |

Next: Session 3 audit pass complete (this update). Session 4 is the Layer 4 testing harness build per `docs/build-handoffs/Testing_App_Specification.md`.

---

## Template Verification (Deliverable 1 Closing Check)

The template was structurally verified against two positions before this version was finalized:

**WR (data-rich offensive position):**
- All three components have rich sub-signals
- Subjective film sources (RSP, TDN, Sharp) regulated by Madden Speed, Catching, Route Running, etc.
- Analytical sources (PFF receiver grades, NGS separation metrics, snap counts) flow through
- RAS High-tier weight, full cap range
- Breakout component fully populated with college data
- Template handles cleanly with no structural contradiction

**DT (data-thin defensive position):**
- Film component has reduced weight per SL-005 (IDP data gap)
- Fewer subjective sources reliably available; Madden regulation maps to fewer claims
- Analytical sources thinner (PFF DL grades, IDP Guru) but present
- RAS Low-tier weight, near-zero cap at Layer 4 (RAS contribution falls to Layer 6 tiebreaker)
- Breakout component fully populated (college data exists for DTs)
- Template handles cleanly with no structural contradiction

Both positions can be specified using this template without architectural change. Position-specific values differ; structural skeleton holds.

---

*Built by: Christopher Campbell + Claude (Anthropic)*

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | June 2026 | Initial release. Deliverable 1 of the Scouting Layer build branch. SL-001 through SL-017 reflected. Verified against WR and DT for structural integrity. |
| 1.1 | June 2026 | Added EMA season transition behavior section. Default CONTINUATION; position rubrics may override to RESET with rationale. |
| 1.2 | June 2026 | Session 3 audit pass. SL-004 tier table amended (TE→High, DT→Hybrid per SL-021). SL-018 time-decay schedule table added. SL-019 modulator applicability and equations added to RAS and Breakout components. SL-020 exclusion rule documented. SL-021 Cushion Guard and dynamic PFF α added as standard Layer 3 / EMA options. NGS Coverage Metrics anchor added to Film component (CB/S only, explicitly excluded elsewhere). Checklist expanded with SL-018/019/020/021 and NGS anchor required specifications. Position Group Build Order updated to show all 10 rubrics complete. |
