# Legacy NFL Position Group Rubric — Kicker (K)
**Version:** 1.0 — June 2026
**Status:** Locked. **Structural exclusion model** per Christopher's Option 1 call — Layer 4 scouting layer output hardcoded to flat 1.000 multiplier across all three components. True ranking at K is driven by Layer 2 raw production, Layer 3 age depreciation, and Layer 5 cap efficiency. This rubric is the cleanest application of SL-020 in the architecture: RAS-not-applicable, film-not-meaningfully-scoutable, breakout-framework-doesn't-translate. Gemini's K open questions reconciled into SL-OQ-042; Layer 6 scarcity-sort preservation confirmed by Gemini's verification check.
**Companion:** Engine_Specification.md Layer 4 is authoritative on mechanics. This rubric specifies the K-specific exclusion architecture.

---

## 1. Architectural Baseline

- **Layer 4 RAS Tier:** **Low** (per SL-004). No combine testing universe exists for kickers — the position genuinely lacks athletic measurement data analogous to other positions.
- **Layer 3 Peak Limit:** 30 years.
- **SL-020 application:** **YES — full strength.** This is the canonical SL-020 case: all three Layer 4 components forced to 1.000 multiplier output. RAS routed to Layer 6 tiebreaker only (and even there, RAS data for kickers is sparse to nonexistent — fallback to position-mean per Layer 1 imputation). Film, breakout, and RAS all collapsed.
- **SL-019 application:** **NOT applicable.** Low-tier RAS excludes by the SL-OQ-027 gating rule.
- **Structural Exclusion Rule:** Kickers bypass Layer 4 valuation shifts entirely. The scouting layer output is structurally hardcoded to 1.000 for all kickers regardless of profile. True valuation is driven by:
  - **Layer 2:** Raw production (FG made by distance, missed FG penalty)
  - **Layer 3:** Age depreciation past peak limit 30
  - **Layer 5:** Cap efficiency tier (with K's $3M extension floor per league rules)

- **Layer 2 Base Points drivers** (per Official Rulebook + MFL rules endpoint, confirmed June 2026):

| Stat | Points | Source |
|---|---|---|
| FG Made 0–39 yards | 3 | MFL confirmed |
| FG Made 40–49 yards | 4 | MFL confirmed |
| FG Made 50–59 yards | 5 | MFL confirmed |
| FG Made 60–99 yards | 6 | MFL confirmed — no separate 70+ tier (SL-OQ-041 resolved) |
| Missed FG 0–29 yards | −3 | MFL confirmed (OQ-002 resolved) |
| Missed FG 30–99 yards | −1 | MFL confirmed (OQ-002 resolved) |
| Extra Point (PAT) Made | +1 | MFL confirmed (SL-OQ-040 resolved) |
| Extra Point (PAT) Missed | −1 | MFL confirmed (SL-OQ-040 resolved) |

All values confirmed via live MFL rules endpoint June 2026. See docs/data-layer/MFL_Scoring_Rules_Decode.md.

- **OQ-002 resolved:** MG 0-29 yards = −3 pts, MG 30-99 yards = −1 pt. The engine reads active config from MFL on startup per DECISION-009; confirmed values for League 14432.
- **SL-OQ-040 resolved:** EP made = +1 pt, EM missed = −1 pt. Confirmed via MFL rules endpoint.
- **SL-OQ-041 resolved:** FG 60–99 yards = 6 pts. No separate 70+ tier in MFL config.

- **Data Parity Rule:** Not relevant at K — Layer 4 is hardcoded 1.000 regardless of data state.

---

## 2. Film Component Configuration

**Cap (asymptote):** **0%** (structurally excluded).

### Configuration

- `film_position_weight`: **0.00**
- `film_cap`: **0.00**
- `film_inflection`: 0.50 (preserved for engine consistency; mathematically irrelevant given cap = 0)
- `film_steepness`: 0.0
- **Film output: hardcoded to 1.000 multiplier** regardless of any sub-signal values

### Sub-signal weights

**None active for Layer 4 valuation.** Kickers do not have meaningful film coverage in the approved source library — no PFF position-specific grade exists for K, IDP sources don't cover K, RSP/TDN/Nerds don't cover K, and the kicker film analyst ecosystem is essentially nonexistent in dynasty content.

### Madden mapping (archival only — not Layer 4 valuation)

| Asset Attribute | Madden Sub-Attribute / Composite | Formula |
|---|---|---|
| "Leg strength" | Kick Power (KPW) | Direct |
| "Accuracy baseline" | Kick Accuracy (KAC) + Awareness (AWR) | (0.8 × KAC) + (0.2 × AWR) |

Madden values logged for database completeness and potential future use (CAL-032 flag), but **do not feed Layer 4 multiplier output**. The hardcoded 1.000 exclusion supersedes any Madden-derived signal.

### EMA blend rates

- Madden α = 0.20 (archival-only, no Layer 4 effect)
- All other α values: N/A (no other sub-signals)

### Season transition

CONTINUATION (mathematically inert at K — no dynamic signals affect Layer 4).

---

## 3. RAS Component Configuration

**Cap (asymptote):** **0%** (structurally excluded per SL-020).

### Configuration

- `RAS_position_weight`: **0.00** (forced across all NFL career stages — SL-018 schedule does not apply)
- `RAS_cap`: **0.00**
- `RAS_inflection`: 0.50 (preserved for engine consistency)
- `RAS_steepness`: 0.0
- **RAS output: hardcoded to 1.000 multiplier per SL-020**

### Normalization

`RAS_normalized = raw_RAS / 10.0` if RAS data exists.

Missing RAS (typical at K — most kickers don't combine-test) → Layer 1 fallback to position-mean (5.00 raw / 0.500 normalized).

### Layer 6 tiebreaker routing

Per SL-020, RAS data still flows to Layer 6 tiebreaker for use when Layer 4 outputs are identical across two kickers (a common outcome at this position given all Layer 4 outputs = 1.000). However, kicker RAS data is so sparse that Layer 6 will frequently fall back to positional scarcity rank (K = 0) per the standard tiebreaker chain.

### Late-career interaction

**None.** SL-019 not applicable (Low-tier excludes by gating rule). No Cushion Guard. Layer 3 age_pull applies in standard form (`0.97^(age − 30)`) with no buffer.

---

## 4. Breakout Component Configuration

**Cap (asymptote):** **0%** (structurally excluded).

### Configuration

- `breakout_position_weight`: **0.00**
- `breakout_cap`: **0.00**
- `breakout_inflection`: 0.50 (preserved for engine consistency)
- `breakout_steepness`: 0.0
- **Breakout output: hardcoded to 1.000 multiplier**

### Reasoning for full exclusion

The breakout framework does not translate to kicker evaluation:
- **Breakout Age** — college kicking conditions differ radically from NFL (collegiate FG distances often shorter; snap quality varies; hash marks differ; altitude effects). A "breakout" 19-year-old college kicker has weak predictive value for NFL longevity.
- **School Tier** — kicker prospects come from all school tiers including FCS and walk-on backgrounds (Jason Sanders from G5, Harrison Butker from P4, many UDFA kickers). School tier carries no signal at K.
- **College Production Share** — college FG market share is meaningless (only one kicker per team; "market share" is 100% by default if the player is the starter).
- **Age Trajectory** — handled at Layer 3 with standard age decay; no need for redundant Layer 4 representation.

### Three-zone classification

Not applicable (composite_input is meaningless when output is hardcoded).

---

## 5. Verification Cases

**S-curve formula:** Not invoked at K — all three components return hardcoded 1.000.

**Component combination:**
```
Layer_4_Output = 1.000 × 1.000 × 1.000 = 1.000 (always)
```

The verification cases below demonstrate that Layer 4 returns exactly 1.000 regardless of kicker profile, and that all meaningful differentiation comes from Layer 3 age decay (and downstream Layer 5 cap efficiency, outside this rubric's scope).

---

### Case 1 — Push case: Brandon Aubrey

**Profile:**
- Age 31 (born February 1995) — 1 year past peak limit
- College: Notre Dame (P4 — Independent at the time)
- Late-blooming kicker (drafted as soccer player, transitioned to football professionally; entered NFL at age 28 in 2023)
- Year 3 NFL veteran
- Production: elite — multiple 60+ yard FGs, top-tier accuracy

**Layer 4 components:**
- Film: hardcoded **1.000** (regardless of any signal)
- RAS: hardcoded **1.000** per SL-020 (no combine data anyway)
- Breakout: hardcoded **1.000** (framework not applicable)

**Layer 4 combined:** 1.000 × 1.000 × 1.000 = **1.000**

**Full Layer 3 × Layer 4 chain:**

Layer 3 age_pull at age 31 (1 year past peak 30) = 0.97^1 = **0.97**. No SL-019 buffer (not applicable at K).

```
Layer 3 × Layer 4 (Aubrey) = 0.97 × 1.000 = 0.970
```

Layer 5 cap efficiency and Layer 2 raw production carry the rest of the ranking work — and at Aubrey's production level (multiple 60+ FGs scoring 6 pts each), Layer 2 contribution is meaningful.

---

### Case 2 — Pull case: Justin Tucker

**Profile:**
- Age 36 (born November 1989) — 6 years past peak limit
- College: Texas (P4 — Big 12)
- Year 14 NFL veteran (2012–2024 with Baltimore; released 2025)
- Production: declining but historically elite — career accuracy leader, multiple All-Pro honors

**Layer 4 components:**
- Film: hardcoded **1.000**
- RAS: hardcoded **1.000** per SL-020
- Breakout: hardcoded **1.000**

**Layer 4 combined:** 1.000 × 1.000 × 1.000 = **1.000**

**Full Layer 3 × Layer 4 chain:**

Layer 3 age_pull at age 36 (6 years past peak) = 0.97^6 ≈ **0.833**. No SL-019 buffer.

```
Layer 3 × Layer 4 (Tucker)  = 0.833 × 1.000 = 0.833
Layer 3 × Layer 4 (Aubrey)  = 0.970 × 1.000 = 0.970
```

**Structural finding (exclusion model verified):** Both Aubrey and Tucker return identical Layer 4 = 1.000. The 14% spread on the full chain comes ENTIRELY from Layer 3 age decay (Aubrey's 1 year past peak vs. Tucker's 6 years past peak). This is the exclusion model working exactly as designed — at K, Layer 4 contributes zero to differentiation; Layer 3 (age) and Layer 5 (cap efficiency) and Layer 2 (raw production) do all the ranking.

**Comparison to other positions:** Every other position rubric shows Layer 4 contributing 5–11% of differentiation between push and pull cases. At K, that contribution is 0%. This is the architectural choice — kicker valuation should be driven by what's measurable (production, age, cap cost), not by extrapolation from absent profile data.

**Layer 5 commentary (outside this rubric's scope but worth noting):** K's $3M extension floor per league rules means even mid-tier kickers carry meaningful cap weight relative to their production ceiling. Layer 5 cap tier alignment is likely the dominant differentiator at K — a Tucker-class veteran on a Cold-tier deal (<$1.50M) is materially more valuable than the same production on a Neutral-tier deal ($1.50–6.00M). CAL-016 / CAL-017 calibration items (from prior sessions) on cap tier boundary calibration apply with full force at K.

---

## 6. Open Questions Surfaced

Prior sessions surfaced SL-OQ-013 through SL-OQ-039 and CAL-015 through CAL-031. K adds:

- **SL-OQ-040:** Extra Point (PAT/XP) scoring values not specified in the Official Rulebook. The Kicking section (lines 71–78) contains only FG and Missed FG. PAT is a routine scoring event in NFL play and standard in most fantasy formats (typically +1 made / −1 or −2 missed). Is this a rulebook gap requiring update, or an intentional design choice (PAT does not score in this league)? Commissioner confirmation needed before the engine ingests kicker production data.

- **SL-OQ-041:** FG 70+ yard scoring threshold not specified. The rulebook caps at "FG Made 60–69 yards" at 6 points. A 70+ yard FG is rare but possible (the current NFL record is 66 yards). Three resolution options: (a) extend the linear pattern to 7 points for 70+, (b) cap at 6 points (no additional bonus), (c) special milestone scoring (e.g., 8 or 10 points to reward the rarity). Commissioner confirmation needed before any kicker hits the threshold.

- **SL-OQ-042 (from Gemini, renumbered from local SL-OQ-022):** Madden API ingestion pipeline optimization at K. Since Madden K data is archival-only (does not affect Layer 4 valuation per the exclusion model), should the data pipeline disable automatic Madden pulling for K position group during in-season weekly cycles to conserve rate limits? Three resolution options: (a) keep K Madden in routine pulls (simple, consistent with other positions, marginal rate-limit cost), (b) disable in-season but pull at season-end and during roster moves (saves weekly rate budget, requires conditional logic in the ingestion module), (c) disable entirely until CAL-032 resolves whether Madden K data has future predictive utility. Recommendation: option (a) for v1.0 — pipeline simplicity outweighs marginal rate-limit savings; revisit if rate limits become binding. Concrete engineering decision for the ingestion module specification.

**OQ-002 (existing, applies here):** Missed FG penalty configuration — rulebook says "−3 / −1." Which value applies under what MFL config? The engine must read the active config from MFL endpoints and apply the correct penalty value.

**Calibration Backlog additions from K build:**

- **CAL-032:** Madden archival data utility at K. The Madden Kick Power and Kick Accuracy attributes are logged per the rubric's Section 2 archival mapping but do not affect Layer 4 valuation. Question: is there future value in surfacing Madden-derived kicker signal (e.g., as a tie-breaker beyond positional scarcity rank, or as an input to Layer 5 cap efficiency)? Current implementation keeps the data accessible without weighting it. Live-data calibration may reveal whether Madden K attributes carry predictive value worth incorporating in v2.0.

---

## 7. Position-Specific Notes

- **Exclusion model rationale:** Kickers are structurally unlike every other position in the engine:
  - No combine testing universe → RAS framework cannot apply meaningfully
  - No approved film source covers kickers analytically (PFF has K grades but they're not in the same predictive class as their position-specific grades; IDP sources skip K entirely; RSP/TDN/Nerds focus on offensive and defensive skill)
  - College production is binary (starter or backup; no market share signal)
  - Breakout-age framework doesn't translate (college kicking conditions differ from NFL)
  
  Forcing Layer 4 multipliers on a position with no genuine input signal would create artificial precision — the model would output false confidence in valuations driven by noise. The exclusion model is honest about what we know.

- **Kicker is a full scoring position per LOCKED-001:** This rubric's exclusion model applies only to Layer 4 forward-looking valuation. K remains a full scoring position with mandatory weekly starts, full Layer 2 raw scoring, the $3M extension floor, and full Layer 5 cap efficiency treatment. The exclusion concerns the SCOUTING-LAYER MULTIPLIER specifically, not the position's status in the league.

- **Layer 5 dominance at K:** Because Layer 4 is structurally inert, Layer 5 cap efficiency becomes the dominant differentiator between kickers of similar Layer 2 production. A Tucker-class veteran on a Cold-tier deal is materially more valuable than the same production on a Neutral-tier deal. CAL-016 / CAL-017 cap tier boundary calibrations apply with full force at K — possibly more than at any other position.

- **Layer 3 carries all aging at K:** Without SL-018 RAS-component-weight reduction (RAS_position_weight = 0.00 everywhere) and without SL-019 buffer (not applicable), Layer 3 age decay applies in pure form (`0.97^(age − 30)`). A kicker at 36 loses ~17% of value from age decay alone vs. a kicker at peak 30. This is sharper aging than at other positions where SL-018/SL-019 partially shield late-career value.

- **Two rulebook gaps surfaced during this rubric build:** SL-OQ-040 (PAT) and SL-OQ-041 (FG 70+) are flagged for commissioner confirmation. Until resolved, kicker production data from MFL needs interpretation rules — the engine should configure to handle both presence and absence of PAT scoring.

- **No SL-019, no Cushion Guard, no dynamic α:** K is the only position in the universe where none of these mechanisms apply. The architecture is at its simplest here, and intentionally so.

---

## 8. Cross-Pollination Source

This rubric synthesized from:
- Universal Rubric Template v1.1 (structural skeleton)
- Engine Specification v2.1 (Layer 4 mechanics, SL-020 mechanism)
- Gemini's K rubric draft (correctly recognized the exclusion model, Layer 2 FG-by-distance scoring, structural exclusion rule statement, Madden archival mapping retained for system completeness, OQ-002 reference for missed FG penalty configuration)
- Gemini's K open questions (reconciled — SL-OQ-042 Madden API pipeline optimization accepted as a concrete engineering decision with v1.0 recommendation to keep K in routine pulls for pipeline simplicity, revisit if rate limits become binding; Gemini's "Calibration Check" treated as a verification statement rather than a new CAL — confirms the exclusion model preserves Layer 6 deterministic scarcity-sort matrix integrity, which is the expected and architecturally-intended behavior, and provides cross-check assurance that the K = 1.000 baseline does not introduce artificial differentiation downstream)
- Christopher's call (Option 1 explicit exclusion model — confirmed during pre-draft architectural question)
- SL-020 invoked formally (Gemini's draft implemented the SL-020 mechanic but did not reference SL-020 explicitly — added)
- Layer 2 drivers verified against Official Rulebook lines 71–78
- Two rulebook scoring gaps surfaced during verification — SL-OQ-040 (PAT/XP not listed) and SL-OQ-041 (FG 70+ not listed)
- `[cite: N]` markers stripped throughout
- Verification cases replaced from generic Tucker/Aubrey placeholder language with explicit named-player cases and full Layer 3 × Layer 4 math demonstrating the exclusion model
- Section 7 expanded with rationale for why exclusion is the architecturally honest choice at K

---

*Built by: Christopher Campbell + Claude (Anthropic)*

| Version | Date | Changes |
|---|---|---|
| 0.9 | June 2026 | Draft from cross-pollinated Gemini baseline + Christopher's Option 1 confirmation. Structural exclusion model: Layer 4 hardcoded to 1.000 across all three components per SL-020. No SL-018, no SL-019, no Cushion Guard, no dynamic α — K is the architectural simplest case. Layer 2 drivers verified against rulebook; two rulebook gaps surfaced (SL-OQ-040 PAT/XP scoring not listed; SL-OQ-041 FG 70+ not listed). Verification cases (Aubrey age 31 push, Tucker age 36 pull) both return Layer 4 = 1.000 — the 14% spread on full chain comes entirely from Layer 3 age decay. Pending Gemini's K open questions for reconciliation pass before v1.0 lock. |
| 1.0 | June 2026 | Locked. Gemini's K open questions reconciled. SL-OQ-042 (Madden API ingestion pipeline optimization for K archival data) renumbered from Gemini's local SL-OQ-022, accepted as a concrete engineering decision with v1.0 recommendation to keep K in routine pulls for pipeline simplicity (revisit only if rate limits become binding). Gemini's "Calibration Check" verification statement noted in Section 8 — confirms the exclusion model preserves Layer 6 deterministic scarcity-sort matrix integrity, which is the architecturally-intended behavior and provides cross-check assurance that the K = 1.000 baseline does not introduce artificial downstream differentiation. **Deliverable 2 complete.** |
