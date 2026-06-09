Legacy NFL — Scoring Engine Specification
Version: 3.0 — June 2026
Status: Foundation document. Algorithm design only. Go implementation is a separate build session.
Changelog: Version 3.0 is the Session 3 audit pass. Formally documents SL-018 (RAS time-decay schedule), SL-019 (RAS modulator interactions at TE/DE/CB/S), SL-020 (Low-tier exclusion at QB/K), SL-021 (DT hybrid tier + Cushion Guard), and the SL-OQ-027 SL-019 gating rule with position applicability table. NGS Coverage Metrics anchor pattern added to film component. Layer 2 True Position split updated: CB and S now have separate formulas with MFL-confirmed premium values (OQ-011 resolved). Layer 2 scoring corrected: FG range updated (no separate 70+ tier), PAT/XP added, missed FG values confirmed (OQ-002 resolved), long play bonus format confirmed (OQ-003 resolved), INT and fumble return yards corrected to 0.1/yard. SL-004 tier table amended: TE moved to High-tier, DT reclassified as Hybrid per SL-021. Cross-rubric decision queue added for Session 3 open architectural questions. Version 2.1 captured year-over-year configuration flexibility. Version 2.0 locked the Layer 4 architecture. Version 1.x was the original superseded skeleton.

Overview
The ranking engine converts player assets into a single comparable score that accounts for:

Raw statistical production (rulebook scoring values, MFL-sourced)
Positional True Position splits (DL premium)
Contract efficiency (cap tier scaling, percentage-based)
Age depreciation (position-specific decay curves, admin-tunable)
Qualitative scouting layer (film, athletic measurement, developmental signals with Madden regulation)
Tiebreaker protocol (deterministic, multi-key, admin-tunable scarcity matrix)

The rulebook is authoritative for the CURRENT year's configuration. The engine reads league-configurable values from MFL on startup and at season transitions, so year-over-year rulebook changes do not require code releases.

Configuration Sourcing (DECISION-009)
League-configurable values are pulled from MFL on engine startup and refreshed at season transitions, not hardcoded in the engine. The following values are MFL-sourced:

Layer 2 scoring values (every line item in the scoring matrix)
Salary cap amount
Minimum salary scale by experience tier
Roster size limits (active, IR, practice squad)
Starting lineup format
Position-specific extension floors
Tender prices for RFA
Rookie pick slot pricing
True Position split tackle values (where MFL exposes them; manual override in admin console otherwise)

The values documented in the layers below reflect the 2026 rulebook. They serve as documentation of current rules and as defaults for the admin console when MFL data is unavailable. The engine implementation reads from MFL config, not from this document.
Admin console can override MFL-sourced values for testing and what-if analysis. Overrides are flagged in the player record so it is clear which scores reflect MFL config versus admin overrides.

Historical Preservation (DECISION-010)
Historical season scores are preserved as scored at the time. Each season's player record is stored with the scoring config that was used to generate it. The engine does NOT retroactively re-score prior seasons when scoring config changes.
If the league votes to change passing TD from 5 pts to 4 pts in 2027, the 2026 season scores stay as they were calculated under the 2026 rules. The 2027 season scores use the new 2027 rules. Historical comparisons between seasons reflect the actual rules in effect for each season.
Re-calculation of historical seasons requires an explicit override flag (admin-only). This is for cases where a data correction is needed, not for adjusting to current scoring rules.
This applies to all engine outputs — Adjusted Score, component multipliers, rankings — not just raw stat scoring.

Layer 1: Data Hygiene & Contract Floor Enforcement
Minimum Salary Scale
The current 2026 rulebook values (documentation, not implementation values):
ExperienceMinimum0 years$0.33M1 year$0.38M2 years$0.43M3 years$0.48M4-6 years$0.53M7-9 years$0.58M10+ years$0.63M
Per DECISION-009, the engine reads these values from MFL config, not from this document.
Data Hygiene Rules

All financial values rounded to nearest $10,000 (two decimal places in millions)
Missing RAS values imputed using position-group arithmetic mean; fallback to 5.00 if group has no valid records
Missing breakout age defaults to 21.0 (flagged as Unknown per Layer 4 data flag mechanics)
Missing school tier defaults to Group of Five (flagged as Unknown)
Missing expert film agreement defaults to neutral baseline (flagged as Unknown)


Layer 2: Scoring Matrix
Scoring values below reflect the 2026 rulebook configuration. Per DECISION-009, the engine reads these values from MFL config — not from this document. The values shown are current defaults and serve as documentation of what the rules are today.
Offensive Scoring
pass_points = (pass_yds × 0.05) + (pass_td × 5) + (pass_int × -2)
            + long_pass_bonus + (pass_2pt × 2)

rush_points = (rush_att × 0.15) + (rush_yds × 0.1) + (rush_td × 6)
            + long_rush_bonus + (rush_2pt × 2)

rec_points  = (rec × 1.0) + (rec_yds × 0.1) + (rec_td × 6)
            + long_rec_bonus + (rec_2pt × 2)
Long Play Bonuses (Discrete Thresholds — Not Continuous Multipliers)
long_pass_bonus:  +1 pt if any single pass attempt ≥ 40 yards
long_rush_bonus:  +1 pt if any single rush ≥ 20 yards
                  +1 pt additional if any single rush ≥ 40 yards (stackable)
long_rec_bonus:   +1 pt if any single reception ≥ 20 yards
                  +1 pt additional if any single reception ≥ 40 yards (stackable)
Critical: These are event-based bonuses, not per-yard multipliers. Each qualifying play triggers the bonus. Multiple qualifying plays in a game each trigger independently.
Special Teams Scoring
fg_points:    3 pts (0-39 yd) | 4 pts (40-49 yd) | 5 pts (50-59 yd) | 6 pts (60-99 yd, no separate 70+ tier — OQ-041 resolved)
missed_fg:   -3 pts (under 30 yd) | -1 pt (30+ yd) — OQ-002 resolved
pat_made:    +1 pt
pat_missed:  -1 pt — OQ-040 resolved
punt_ret_td: 6 pts
punt_ret_yds: 0.025/yard
ko_ret_td:   10 pts
ko_ret_yds:  0.025/yard
Fumble Scoring
fumble_lost:        -2 pts
own_fum_recovery:   +2 pts
opp_fum_recovery:   +3 pts
opp_fum_rec_yds:    0.1/yard — MFL confirmed (rulebook had 0.025, MFL config is correct)
forced_fumble:      +4 pts
Defensive Scoring — True Position Split
# Defensive Line (DT and DE)
def_points_DL = (solo_tackles × 2.5) + (assist_tackles × 1.5)
              + (sacks × 4.5) + (interceptions × 5)
              + (passes_defensed × 2.5) + (qb_hits × 1.0)
              + (tackles_for_loss × 2.5)
              + (blocked_kicks × 7) + (safeties × 10)

# Cornerback and Safety (CB, S) — MFL additive premium confirmed (OQ-011 resolved)
def_points_CBS = (solo_tackles × 2.0) + (assist_tackles × 1.0)
               + (sacks × 4.5) + (interceptions × 6)
               + (passes_defensed × 3.0) + (qb_hits × 1.0)
               + (tackles_for_loss × 2.5)
               + (int_return_td × 6) + (int_return_yds × 0.1)
               + (blocked_kicks × 7) + (safeties × 10)

# Linebacker (LB; QB if they record defensive stats — rare)
def_points_LB = (solo_tackles × 1.5) + (assist_tackles × 1.0)
              + (sacks × 4.5) + (interceptions × 5)
              + (passes_defensed × 2.5) + (qb_hits × 1.0)
              + (tackles_for_loss × 2.5)
              + (int_return_td × 6) + (int_return_yds × 0.1)
              + (blocked_kicks × 7) + (safeties × 10)
Note: Tackle for Loss (2.5 pts) is a direct stat. Do not approximate via solo tackle multiplication.
Note: Three distinct defensive formulas. DL premium on TK/AS only. CB/S carry a confirmed MFL additive premium on TK (+0.5 → 2.0), IC (+1 → 6), and PD (+0.5 → 3.0) relative to the universal base. LB uses universal base values only. All values MFL-sourced and confirmed — OQ-011 resolved via MFL rules endpoint (June 2026).

Layer 3: Age Decay Matrix
Position-specific peak longevity thresholds. Compounding annual depreciation past threshold.
age_pull = (1 - decay_rate) ^ max(0, player_age - peak_limit)
Default decay rate: 3% (0.03), yielding the 0.97 base used in prior versions. Per SL-017, both the peak limits and the decay rate are admin-tunable. As player longevity shifts over time (rule changes, training/sports-science improvements, position-specific evolution), these parameters can be recalibrated through the admin console without code changes.
Current Peak Limit Defaults
PositionPeak LimitQB32RB25WR29TE29DE30DT30LB29CB28S28K30
These defaults are starting points. Empirical calibration against actual league data is a Calibration Backlog item.

SL-019 Layer 3 Buffer (TE / DE / CB / S)
At applicable positions (see SL-019 Gating Rule in Layer 4 above), RAS modulates the age_pull output:
buffered_age_pull = age_pull + (1.0 - age_pull) × buffer_strength × RAS_normalized
This buffer slows effective decay velocity for players with elite athletic profiles. Buffer strength is position-specific and admin-tunable. Only activates past the position's peak limit — at or before peak, age_pull = 1.0 and the buffer has no effect.

SL-021 — DT Late-Career Cushion Guard
At DT only, when Raw RAS ≥ 8.00, the age_pull calculation applies a 10% decay velocity reduction:
  cushioned_age_pull = peak − (peak − standard_age_pull) × 0.90
  (where peak = 1.0 and standard_age_pull is the uncushioned Layer 3 value)

When Raw RAS < 8.00: no protection — standard age_pull applies.
Threshold = 8.00 (binary, admin-tunable). See SL-OQ-037 for open question on continuous RAS scaling alternative.
SL-019 is NOT applied at DT. Cushion Guard is the DT-unique replacement. Running both would double-protect elite-RAS veterans.

Layer 4: Scouting Layer
Architecture
Layer 4 is the engine's intelligence layer. It combines three multiplicative components:
Layer_4_Output = film_effective × RAS_effective × breakout_effective
Each component is a multiplier centered on 1.00, bounded by its own S-curve asymptote (Shape B per SL design), and scaled by per-component confidence and per-component position weight (SL-013).
There is no overall Layer 4 cap. Each component's cap acts as the natural bound. The combined swing equals the product of component caps (Option A from architecture decisions).
The output multiplies Base_Points after Layer 3 age decay:
Scouting_Adjusted_Points = Base_Points × age_pull × Layer_4_Output
Position group rubrics specify position-specific values for sub-signal weights, S-curve parameters, caps, Madden attribute mappings, and EMA blend rates. The structural mechanics defined here apply uniformly across all positions. See Universal_Rubric_Template.md for the structural skeleton every position rubric fills in.

Component 1: Film Component
The film component aggregates expert evaluation from approved sources. RSP and PFF act as paired anchors (subjective and analytical respectively, per SL-002 as amended). Other expert sources contribute as directional modifiers.
Sub-signal categories
Subjective sub-signals — qualitative expert descriptions vulnerable to enthusiasm and over-reporting:

RSP qualitative claims (Matt Waldman's Rookie Scouting Portfolio descriptions)
TDN scouting language (The Draft Network)
IDP Show analyst takes
Sharp Football qualitative analysis

Analytical sub-signals — numerical methodologies that self-regulate:

PFF grades (numerical scores from rule-based film evaluation)
IDP Guru weekly analytics
NFL Next Gen Stats metrics
NFL production data (accumulating stats)

NGS Coverage Metrics Anchor (CB and S only — SL-OQ-027 resolution extended)
At CB and S, NFL Next Gen Stats coverage metrics earn a dedicated sub-signal anchor at weight 0.30. This anchor reflects that coverage-grade tracking data (separation allowed, completion % allowed, target rate, etc.) is the primary post-draft analytical signal at these positions — directly measuring the core skill being valued. This anchor is explicitly excluded at all other positions. Interior defenders (DT, LB), pass rushers (DE), offensive positions, and kickers lack equivalent coverage-grade tracking data. Applying it outside CB/S would introduce noise, not signal.

Madden regulation (SL-015, SL-016)
Madden ratings regulate subjective sub-signals only. Analytical sub-signals flow through without Madden regulation (per SL-016 — measurements self-regulate; subjective judgments need confirmation).
For each subjective expert claim:

Position rubric specifies which Madden sub-attribute (or composite) corresponds to the claim
Madden sub-attribute is normalized within position group (relative rating across all true-position peers)
Approach D regulation applies:

If disagreement < threshold: subjective claim weight holds at full strength
If disagreement ≥ threshold: claim weight blends toward Madden-implied weight, blend strength scaling with disagreement magnitude beyond threshold



Threshold value and blend scaling function are admin-tunable.
Aggregation (Approach A)
After Madden regulation (where applicable), all sub-signals feed into Approach A:
For each sub-signal in film component:
  normalized_signal = position_specific_normalization(raw_signal)  # → 0.0–1.0
  weighted_contribution = normalized_signal × position_rubric_weight

composite_input = Σ(weighted_contribution across all film sub-signals)
film_raw = S_curve(composite_input)
S-curve cap (asymptote) per SL-002: configurable in admin console, default range 3–5% applied to the film component. Cap is admin-tunable.
Scaling
film_effective = 1.0 + (film_raw - 1.0) × film_confidence × film_position_weight

film_confidence    = Σ(film_fields_present_weight) / Σ(film_fields_expected_weight)
film_position_weight = specified per position rubric (default 1.0)
Output
film_multiplier ∈ [1.0 - film_cap, 1.0 + film_cap]

Component 2: RAS Component
The RAS component takes a single signal: the player's Relative Athletic Score normalized within their true position group.
Position weight tier (per SL-004, amended by Session 2 for TE and SL-021 for DT)

Tier     | Positions           | Layer 4 Role
---------|---------------------|----------------------------------------------
High     | WR, TE, DE, CB, S   | Meaningful push/pull contribution
Medium   | RB, LB              | Secondary signal
Low      | QB, K               | Forced 1.000 at Layer 4 (SL-020); RAS value used in Layer 6 tiebreaker only
Hybrid   | DT (SL-021)         | Medium film (SL-005 compression) + High-tier RAS treatment; see SL-021

TE amendment: Moved from Medium to High-tier based on Session 2 rubric work — athletic profile is highly predictive of TE longevity arc.
DT hybrid: Medium film classification with High-tier RAS treatment per SL-021 resolution package.

For Low-tier positions (QB, K), the RAS component is forced to exactly 1.000 per SL-020. The RAS value is still recorded for use in Layer 6 tiebreaker resolution. K has all three Layer 4 components forced to 1.000 — Layer 3, Layer 5, and Layer 2 carry all K ranking work.

SL-018 — RAS Position Weight Time-Decay Schedule
RAS_position_weight decays by career stage across all positions. The decay reflects that RAS predicts rookie athletic integration but is progressively superseded by accumulated NFL film grades.

Tier    | Year 0 (draft) | Year 1       | Year 2+
--------|----------------|--------------|--------
High    | 1.00           | 0.50         | 0.10
Medium  | 0.60           | 0.30         | 0.06
Low     | 0.00           | 0.00         | 0.00

Year 0 = draft year (no NFL data). Year 1 = first full NFL season completed. Year 2+ = all subsequent seasons.
DT (Hybrid tier) uses the High-tier schedule for its RAS component per SL-021.

SL-019 — RAS Modulator Interactions (TE / DE / CB / S only)
At positions where RAS is highly predictive of longevity arc, RAS modulates the breakout component sub-signals beyond its own S-curve contribution. Three distinct modulated values per applicable position:

1. Breakout age modulator (applied to normalized breakout age score in breakout component):
   modulated = base + (1.0 - base) × modulator_strength × RAS_normalized

2. Age trajectory modulator (applied to normalized age trajectory score in breakout component):
   modulated = base + (1.0 - base) × modulator_strength × RAS_normalized

3. Layer 3 buffer (applied to age_pull in Layer 3 for applicable positions):
   buffered_age_pull = age_pull + (1.0 - age_pull) × buffer_strength × RAS_normalized

Modulator strength values are position-specific and admin-tunable. See position rubrics for defaults.

SL-019 Gating Rule (SL-OQ-027 resolved — documented in DE_Rubric.md Section 7):
SL-019 applies when BOTH conditions are met:
  (a) Position is High-tier RAS per SL-004/SL-021 tier table, AND
  (b) Predictive relationship between athletic profile and longevity arc is established for the position

Confirmed applicable: TE, DE, CB, S
Explicitly excluded:
  QB — Low-tier per SL-020 (excluded by tier)
  RB — Medium-tier; scheme-driven longevity, not athletic-profile-driven
  LB — Medium-tier; scheme-dependent longevity, too variable
  K  — Low-tier per SL-020 (excluded by tier)
  DT — Cushion Guard per SL-021 is the DT-unique equivalent (running both would double-protect)
  WR — Excluded for v1.0 (SL-022 — Option A, SL-OQ-043 closed). Layer 3 carries all WR aging. Flagged for calibration revisit in v1.1 if elite-RAS WRs prove systematically under-protected.

SL-020 — Low-Tier RAS Exclusion (QB and K)
QB: RAS component forced to exactly 1.000. Film and breakout components operate normally.
K:  All three Layer 4 components forced to exactly 1.000. No scouting layer influence. Layer 3 + Layer 5 + Layer 2 carry all K ranking work.
RAS values still recorded at both positions for Layer 6 tiebreaker use.
Aggregation
RAS_normalized = relative rating within position group
RAS_raw = S_curve(RAS_normalized)
S-curve cap per position rubric, scaled by position weight tier. High-tier positions carry wider caps; Medium-tier narrower; Low-tier collapsed.
Scaling
RAS_effective = 1.0 + (RAS_raw - 1.0) × RAS_confidence × RAS_position_weight

RAS_confidence: 1.0 if RAS Present, 0.0 if Absent (player did not test).
                Position-group mean fallback applied at Layer 1 when missing.
RAS_position_weight: per position rubric, scaled by SL-004 tier
Output
RAS_multiplier ∈ [1.0 - RAS_cap, 1.0 + RAS_cap]

Component 3: Breakout Component
The breakout component aggregates four mechanical college-era inputs. No Madden regulation applies (the inputs are mechanical, not subjective). No NIL recency engineering applies (per SL-003 as amended and SL-014 — expert interpretation of these signals lives in the film component).
Sub-signal inputs (per SL-003 as amended)

Breakout age — continuous numerical (years)
School tier — categorical (Power Four / Group of Five / FCS / Non-FCS per SL-001)
College usage rate — continuous numerical (% of team production)
Age trajectory — derived position on age curve relative to Layer 3 peak limit

Normalization
Each sub-signal normalized to 0.0–1.0 scale via position-specific functions defined in the position rubric.
Aggregation (Approach A)
composite_input = Σ(normalized_signal × weight) across all four sub-signals
breakout_raw = S_curve(composite_input)
Three-zone classification (per SL-003)
The S-curve output produces natural zone classifications used downstream:

Elite zone: breakout_raw at upper plateau
Average zone: breakout_raw in middle range
Late zone: breakout_raw at lower plateau

Zone boundary thresholds defined per position rubric, admin-tunable.
Scaling
breakout_effective = 1.0 + (breakout_raw - 1.0) × breakout_confidence × breakout_position_weight

breakout_confidence    = Σ(breakout_fields_present) / Σ(breakout_fields_expected)
breakout_position_weight = per position rubric (default 1.0)
Veteran behavior (per SL-007, SL-010)
All four breakout sub-signals are static — set at draft entry and unchanged afterward. For veterans, the breakout component holds at the value calculated at draft entry. Injuries, redshirt time, and missed snaps do not reset the model.
Age trajectory is the one sub-signal that technically continues to update as the veteran ages, but this is captured automatically through Layer 3's peak limit comparison.
Output
breakout_multiplier ∈ [1.0 - breakout_cap, 1.0 + breakout_cap]

Data Flag Mechanics (per SL-006)
Each input field carries one of three states:
StateMeaningEngine BehaviorPresentData exists and has been sourcedUse raw valueAbsentData will never exist (player did not test, metric N/A)Use positional fallback (Layer 1)UnknownData not yet collected, may existUse neutral placeholder; flag for recalculation when data arrives
Confidence calculation per component
film_confidence     = Σ(film_field_present_weights)     / Σ(film_field_expected_weights)
RAS_confidence      = 1.0 if Present else 0.0
breakout_confidence = Σ(breakout_field_present_weights) / Σ(breakout_field_expected_weights)
Per SL-011, the confidence score is an internal engine flag. It scales the deviation of each component from 1.00 (per SL-013) but is not surfaced in the user interface. An overall player confidence score can be derived as a weighted aggregate of the three component confidences for internal debugging or developer-mode display.

Running Blend (per SL-007, Option A EMA)
Static sub-signals (set once, never update)

School tier
Pre-NFL breakout age
College usage rate
RAS score

These hold their initial values for the lifetime of the player record.
Dynamic sub-signals (update repeatedly, blend via EMA)

PFF grades (weekly during NFL season)
IDP source weekly grades
RSP annual refresh (April publication)
NFL production data (accumulating game by game)
Madden ratings (multiple updates per NFL season)

EMA formula
new_value = (1 - α) × previous_value + α × new_observation
α (blend rate) is configurable per sub-signal in the position rubric.
Initial value rule
The first observation of a dynamic sub-signal becomes that sub-signal's starting value. EMA blending begins on the second observation.
Offseason hold
When a dynamic sub-signal has no new observations for a period (e.g., NFL offseason for PFF grades), the value holds. The EMA does not decay during the silent window. The value resumes blending when new observations arrive.
Season transition behavior
When a new NFL season begins, dynamic sub-signals continue blending from their prior season's final value by default. Career production patterns matter for dynasty evaluation, so the prior season's accumulated PFF grade and NFL production blend with the new season's first observation per the standard EMA formula.
Position rubrics may override the default to RESET for specific sub-signals where the prior season's value is genuinely less relevant to the new context (e.g., a sub-signal that tracks scheme fit in a specific year's offensive system). Override and rationale documented in the rubric.

Multi-Position Player Blending (per SL-009)
When a player qualifies for multiple position scores:

Calculate Layer 4 for each applicable position separately (using each position's rubric)
Blend results weighted by primary role and snap share

Layer_4_blended = Σ(Layer_4_per_position × role_weight)
role_weight = snap_share_at_position / total_snap_share
If snap share data is unavailable, use depth chart designation (primary position weight 1.0, secondary 0.0). Position rubrics may override blend logic where the position calls for it.

Admin-Tunable vs Structural Parameters (per SL-017)
Admin-tunable (exposed in standard admin UI)

Sub-signal weights (per component, per sub-signal, per position)
S-curve cap values (film, RAS, breakout — per position)
S-curve inflection points and steepness (per position, per component)
Madden regulation threshold (per position)
Madden regulation blend scaling parameters (per position)
EMA blend rate α (per dynamic sub-signal)
EMA season transition behavior (continue vs reset, per dynamic sub-signal)
Madden attribute mapping table (per position)
Position weight per component (per position)
Breakout zone boundary thresholds (per position)
School tier lookup values
Sub-signal normalization function parameters (per position, per sub-signal)
Cap tier boundary percentages (Layer 5)
Layer 2 scoring values (default sourced from MFL; admin override available)
Layer 3 peak limits per position
Layer 3 decay rate
Layer 6 positional scarcity matrix

Structural (code-locked)

Multiplicative combination of three Layer 4 components
No overall Layer 4 cap (component caps only)
S-curve mechanic itself (Shape B / sigmoid family)
Per-component scaling structure (confidence × position weight applied to deviation)
Approach A aggregation structure (normalize → weighted sum → S-curve)
Madden regulation mechanism (Approach D — threshold + gradient blend)
Madden regulates subjective sub-signals only (analytical signals flow through)
EMA blend mechanism (Option A)
Three-state data flag (Present / Absent / Unknown)
Confidence score as internal flag (not UI-surfaced)
Static vs dynamic sub-signal categorization
Initial value rule and offseason hold mechanics
Six-layer engine pipeline order
MFL-sourcing of league-configurable values (DECISION-009)
Historical preservation policy (DECISION-010)
Cap tier expressed as percentage of league cap (not absolute dollars)

Developer-mode access
All structural parameters above are technically tunable through direct developer-mode admin access for advanced calibration and testing. The standard admin UI exposes only the operational tunable list. This protects league users from breaking the engine while preserving full calibration capability for the engine maintainers.

Layer 5: Contract Efficiency Scaling
Cap tier boundaries expressed as percentages of the current league salary cap. As the cap grows year over year, tiers scale automatically without requiring admin intervention. Boundaries remain admin-tunable per SL-017.
Default cap tier percentages
Cold Tier   = salary < 1.2% of league cap
Neutral     = 1.2% to 4.8% of league cap
Hot Tier    = salary > 4.8% of league cap
At the 2026 league cap of $125M, these percentages produce the equivalent dollar boundaries:

Cold: salary < $1.50M
Neutral: $1.50M to $6.00M
Hot: salary > $6.00M

If the cap grows to $150M, the boundaries automatically become $1.80M and $7.20M without any admin action.
Multipliers
Cold Tier multiplier  = 1.15
Neutral multiplier    = 1.00
Hot Tier multiplier   = 0.85

Adjusted_Score = Scouting_Adjusted_Points × cap_multiplier
Calibration
Tier boundary percentages and multiplier values are starting points pending empirical calibration once real MFL data flows through the pipeline (OQ-006). Calibration Backlog item.
Future consideration
Rebuild around bid point cost vs projected score output (cost-per-point efficiency). Flag for Phase 2 testing.

Layer 6: Tiebreaker Protocol (Deterministic)
When multiple players calculate to identical Adjusted Scores:

Tenure: Veteran status wins over rookie (is_veteran == True)
Athletic Score: Higher RAS wins (this is where Low-tier RAS positions get their RAS contribution)
Positional Scarcity: Sorted by market premium

Current scarcity matrix defaults
PositionScarcity RankQB9 (highest)DT8DE7LB6S5CB4RB3WR2TE1K0
Per SL-017, the scarcity matrix is admin-tunable. As league market dynamics evolve (TE valuation, IDP scarcity shifts), the matrix can be recalibrated through the admin console without code changes.

Known Data Risks
EDGE Classification (OQ-004): MFL data may tag players as EDGE. The engine requires explicit DE or LB classification before Layer 2 (True Position split) executes. Mapping logic needed in the data ingestion layer.
Stat Crew Variance: Tackle credits in IDP fantasy can vary by home stat crew. The IDP Guru tracks this. Flag as a calibration input for the scouting layer once real data is flowing.
Long Play Bonus Data (OQ-003 — RESOLVED): Confirmed discrete separate events (P40, R20, R40, C20, C40) in playerScores endpoint. Not embedded in yardage totals. Engine reads as individual stat events.
Madden Data Continuity: Madden ratings can change methodology between annual releases (Madden 26 → Madden 27, etc.). The engine should detect the current version each season and gracefully handle rating scale shifts if they occur.
MFL True Position Split Exposure (OQ-011 — RESOLVED): MFL rules endpoint confirmed. Additive mechanic: universal base (TK 1.5 / AS 1.0) plus position-specific modifiers. DT/DE: +1.0/+0.5. CB/S: +0.5/+0.0 for TK, +1.0 for IC, +0.5 for PD. LB: base only. See Layer 2 True Position Split formulas above and MFL_Scoring_Rules_Decode.md.

Cross-Rubric Architectural Decision Queue (Session 3)
The following open questions are queued for Christopher's decision before the relevant build sessions. They are documented here as architectural flags — not resolved.

SL-OQ-038 — Dynamic Year 1 / Year 2+ PFF α propagation
Currently DT-only (SL-021). Question: should this mechanism be adopted cross-rubric at positions where Year 1 NFL data should aggressively displace rookie-era signals? Status: Open. Venue: Session 3 review. If resolved, becomes SL-022.

SL-OQ-035/036 — Safety rubric branching
SL-OQ-035: Should S split into S_BOX and S_DEEP sub-rubrics with separate weights? SL-OQ-036 (alternative): role-conditional sub-signal weighting within a monolithic S rubric. Status: Open. Both options documented in S_Rubric.md.

SL-OQ-037 — Cushion Guard continuous scaling
Current: binary gate at RAS 8.00. Alternative: continuous RAS scaling for architectural consistency with SL-019 modulator approach. Status: Open. Christopher confirms binary vs. continuous.

SL-OQ-043 — WR SL-019 status
CLOSED — Option A (exclude for v1.0). SL-022 assigned. Calibration revisit flagged for v1.1.

Output Schema
FieldDescriptionFinal_RankLeague-wide asset rankplayer_idMFL player ID (string, not integer)namePlayer namepositionTrue position (DT/DE/LB/CB/S/QB/RB/WR/TE/K)ageCurrent ageis_veteranBooleanyears_expNFL experience yearssalaryAnnual salary in millionscontract_yearContract expiration yearcontract_statusUFA / RFAseasonSeason year — supports historical preservation per DECISION-010scoring_config_idReference to the scoring configuration used for this recordBase_PointsRaw scoring output (Layer 2)age_pullLayer 3 age decay multiplierfilm_multiplierLayer 4 film component outputRAS_multiplierLayer 4 RAS component outputbreakout_multiplierLayer 4 breakout component outputLayer_4_OutputCombined film × RAS × breakoutScouting_Adjusted_PointsBase_Points × age_pull × Layer_4_Outputcap_tierCold / Neutral / Hotcap_tier_basisPercentage-of-cap value at time of calculationAdjusted_ScoreFinal score after Layer 5 cap scalingRASRelative Athletic Scorefilm_confidenceInternal engine flag (per SL-011)RAS_confidenceInternal engine flagbreakout_confidenceInternal engine flagoverall_confidenceDerived weighted aggregate, internal only

Built by: Christopher Campbell + Claude (Anthropic)
Version | Date | Changes
--------|------|--------
1.0 | June 2026 | Initial release with skeleton Layer 4
1.1 | June 2026 | Layer 4 skeleton flagged as superseded after scouting layer architecture session
2.0 | June 2026 | Layer 4 replaced with locked architecture. SL-001 through SL-017 reflected. Output schema expanded for component-level debugging and admin display.
2.1 | June 2026 | Structural pass for year-over-year configuration flexibility. DECISION-009, DECISION-010. Layer 5 cap tiers as % of league cap. Layer 3 and Layer 6 admin-tunable. EMA season transition behavior. Output schema adds season and scoring_config_id.
3.0 | June 2026 | Session 3 audit pass. SL-018/019/020/021 formally documented. SL-004 tier table amended (TE→High, DT→Hybrid). SL-OQ-027 gating rule with position applicability table. NGS Coverage Metrics anchor (CB/S only). True Position split updated: CB and S separated from LB with MFL-confirmed premium values. Layer 2 scoring corrected: FG range, PAT/XP, missed FG values, INT and fumble return yards (0.1/yard). OQ-002, OQ-003, OQ-011 marked resolved. Cross-rubric decision queue added for SL-OQ-038, SL-OQ-035/036, SL-OQ-037, SL-OQ-043.
