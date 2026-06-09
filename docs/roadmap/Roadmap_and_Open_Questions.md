# Legacy NFL — Development Roadmap & Open Questions
**Version:** 2.0 — June 2026
**Status:** Living document. Update at the start and end of every build session.
**Delta absorbed:** Roadmap_Delta_Session2.md (Deliverable 2, Session 2 — all 10 rubrics complete)

---

## Build Philosophy

This application has no defined endpoint. It grows as the league grows. The build follows a deliberate phase structure: personal tool first, alpha when stable, beta when hardened, public when ready. No artificial deadlines. Quality over speed.

Christopher directs. Claude executes. Both flag problems. Neither proceeds on wrong assumptions.

---

## Phase 1 — Personal Tool (Current)

**Goal:** A working application Christopher can use as his competitive edge tool processing all 32 teams.

**Required for Phase 1 completion:**
- [ ] MFL API data pipeline — fetches all 32 rosters, contracts, players, scoring
- [ ] Scoring engine Go implementation — rulebook-verified values
- [ ] 32-team asset rankings output
- [ ] Per-team roster view
- [ ] Basic transaction validation layer (cap check, roster check, deadline check)
- [ ] Dead cap auto-calculator

**Nice to have in Phase 1:**
- [ ] Weekly power rankings (basic)
- [ ] Free agent pool with ranking scores

---

## Phase 2 — Alpha (League-Wide Access)

**Goal:** League mates use the application. Christopher's league becomes the test environment.

**Required for Phase 2:**
- [ ] Multi-user access with individual team authentication
- [ ] In-app transaction system (UFA bidding, RFA, waivers, trades)
- [ ] 24-hour bid clock management
- [ ] Snipe detection and enforcement
- [ ] DOT voting interface
- [ ] Trade deadline hard block
- [ ] Commissioner dashboard
- [ ] Mobile-optimized interface

**Testing approach:**
League mates report back. Bugs, edge cases, and rule interpretation questions surface through actual use. Build fixes. Repeat.

---

## Phase 3 — Beta (Public)

**Goal:** Available to other leagues running similar formats.

**Required for Phase 3:**
- [ ] Multi-league support (configurable scoring rules, roster settings)
- [ ] League setup wizard (import scoring config from MFL)
- [ ] MFL write access integration (authenticated transaction submission)
- [ ] Documentation for league commissioners to onboard
- [ ] Performance hardening for multi-tenant load

---

## Phase 4 — Long-Term Scope (No Timeline)

- Computer-use / screen-interaction layer (Codex-style click-through automation on MFL)
- Historical analytics and franchise valuation
- Offseason cap planner
- Draft pick value model (built from actual league draft history)
- Scouting layer full calibration (3-5% weight tested and locked)
- Support for additional host platforms beyond MFL

---

## Open Questions

These must be resolved before the relevant build session begins.

### Critical — Resolve Before First Build

**OQ-001: MFL League ID**
The Legacy NFL MFL league ID is required for all API calls.
Status: RESOLVED — League ID 14432, host server www47. Documented in docs/data-layer/MFL_API_Reference.md.

**OQ-002: MFL Scoring Configuration Verification**
The rulebook specifies missed FG as "-3 pts / -1 pt" — two values. Need to confirm which MFL uses and under what conditions. Likely -1 for short misses and -3 for longer, but the MFL config is the authority.
Status: RESOLVED — MG 0-29 yards = -3 pts. MG 30-99 yards = -1 pt. Confirmed via live rules endpoint. See docs/data-layer/MFL_Scoring_Rules_Decode.md.

**OQ-003: Long Play Bonus Data in MFL Exports**
The rulebook scores discrete long play bonuses (20+ yard rush = +1pt, 40+ = +1pt additional, etc.). Need to confirm whether MFL exports include these as separate stat events or whether they are embedded in total points. This affects how the scoring engine reads the data.
Status: RESOLVED — Discrete separate events (P40, R20, R40, C20, C40) in playerScores endpoint. Not embedded in yardage totals. A 43-yard run triggers both R20 and R40. See docs/data-layer/MFL_Scoring_Rules_Decode.md.

**OQ-004: EDGE Position Mapping**
MFL may classify some players as EDGE. The scoring engine requires explicit DE or LB before Layer 2 (True Position split) executes. What is the authoritative mapping source — MFL position tags, NFL.com, or a manually maintained league list?
Status: OPEN

**OQ-005: Salary Adjustment Line Item**
The Arizona Cardinals MFL export showed a $5.49 salary adjustment. The API must account for these adjustments in cap calculations. Need to confirm what endpoint exposes salary adjustments and how they are applied.
Status: OPEN

### Important — Resolve Before Relevant Module Build

**OQ-006: Cap Tier Calibration**
Current cap tiers (Cold < $1.50M, Neutral $1.50M–$6.00M, Hot > $6.00M) are from the rubrics — not the rulebook. Once MFL data is live, recalibrate around actual salary distribution across all 32 teams. Flag for Phase 1 testing.
Status: OPEN

**OQ-007: Scouting Layer Weight**
The scouting/film layer is weighted at 3-5% pending long-term testing. The exact weight and which component metrics carry how much influence has not been defined. Initial implementation should be configurable so testing can determine the right values.
Status: OPEN

**OQ-008: Franchise Tag Calculation Timing**
The rulebook states the tag price equals the average salary of the top 5 players at that position across the league. Does "across the league" mean all 32 rostered players at that position, or only the top 5 based on current salaries at the time of tagging? And does it include practice squad and IR salaries?
Status: OPEN

**OQ-009: RFA Player Eligibility Window**
The rulebook states "A player must be in their 3rd year or earlier on the last year of their contract." Need to confirm: does the year count from the original contract signing date, or from the player's NFL experience level?
Status: OPEN

**OQ-010: Playoff Bid Rules Trigger**
The rulebook notes: "After Week 12: playoff teams may place 1-year bids only. Non-playoff teams may not bid at all." And: "During the playoffs, the $12M cap on 1-year bids is lifted." There is a noted clarification flag in the rulebook itself: "confirm exact trigger — start or conclusion of Week 12?" Christopher to confirm with league commissioner.
Status: OPEN — needs commissioner confirmation

**OQ-011: True Position Tackle / Assist Split in MFL Rules**
MFL rules endpoint must confirm whether position-specific tackle/assist values are applied additively on top of a universal base or as standalone overrides.
Status: RESOLVED — Additive mechanic confirmed. Universal base: TK 1.5 / AS 1.0. DT/DE +1.0/+0.5 additive → 2.5/1.5. CB/S +0.5/+0.0 additive → 2.0/1.0. LB base only. See docs/data-layer/MFL_Scoring_Rules_Decode.md.

---

## Known Technical Risks

**RISK-001: MFL Rate Limiting**
The MFL API returns 429 on overload. The pipeline must implement backoff logic and caching. Player database should be fetched once daily maximum.

**RISK-002: MFL Host Routing**
MFL leagues can be moved between servers (wwwXX format). The league host should be fetched via the league API call and stored, but must be re-fetched periodically as leagues can move between hosts.

**RISK-003: Player ID Type Enforcement**
MFL player IDs are strings. IDs under 1000 require leading zeros. Any part of the codebase that handles player IDs must enforce string type. Integer treatment will cause data integrity failures.

**RISK-004: Proboards Integration Boundary**
Proboards does not have a public API and automated scraping violates their TOS. The transaction system in Phase 1 is a standalone application layer — it does not integrate with or read from Proboards. Historical content is manually curated.

**RISK-005: Multi-Tenant Data Isolation**
When Phase 2 opens to league-wide use, each team's sensitive data (bid strategy, draft lists) must be isolated. No GM should see another GM's private transaction drafts or bid intentions before they are posted.

---

## Completed Decisions (Locked)

**DECISION-001: Kicker is a full position.**
PK scores, starts weekly, has an extension floor ($3M), appears in the scarcity matrix at rank 0. Included in all engine layers.

**DECISION-002: TFL is a direct stat, not a proxy.**
Tackle for Loss scores 2.5 pts directly per the rulebook. No solo tackle multiplication proxy. The rubrics' TFL proxy is retired.

**DECISION-003: Long play bonuses are discrete threshold events.**
Not continuous multipliers. Each qualifying play triggers the bonus independently. Multiple qualifying plays in one game stack.

**DECISION-004: Scouting layer is a first-class component.**
Not theoretical. Not a future enhancement. It is part of the engine. Weight is 3-5% pending testing. Data sources are approved and documented.

**DECISION-005: Application is read-only against MFL in Phase 1.**
All transactions are generated by the application and executed manually on MFL. Write integration is Phase 2.

**DECISION-006: Trade deadline enforcement is a hard block.**
No trade submission accepted after Week 9 deadline. Timestamp check. No exceptions. No human needed to catch it.

**DECISION-007: DOT retains full judgment authority on trade value.**
The trade analyzer surfaces data. It does not make veto recommendations. Human judgment on league competitive health is irreplaceable.

**DECISION-008: Proboards historical content is manually curated.**
No automated scraping. League History document is populated by Christopher from direct access. This is the sustainable and TOS-compliant approach.

**DECISION-009: All league-configurable values are MFL-sourced from the league endpoint.**
No hardcoded cap amounts, scoring values, or roster limits in the engine. All configurable parameters pulled from MFL at runtime per DECISION-009. Admin UI reads from MFL; overrides stored as deltas, not replacements.

**DECISION-010: Historical season scores are preserved under the original scoring config.**
Changing engine parameters does not retroactively re-score prior seasons. Every engine output record carries a `scoring_config_id`. Historical scores are immutable. New config → new records only.

---

---

## Architectural Locked Decisions (SL series)

SL-001 through SL-017 are documented in `docs/scoring-engine/Engine_Specification.md` and the individual position rubrics. Key decisions from Session 2:

**SL-018 — RAS position weight time-decay schedule**
Three-tier decay by career stage (Year 0 / Year 1 / Year 2+): High-tier 1.00/0.50/0.10, Medium-tier 0.60/0.30/0.06, Low-tier 0.00 (excluded). Applies at all positions per their RAS tier.

**SL-019 — RAS modulator interactions (TE / DE / CB / S only)**
Three distinct modulated values per applicable position: breakout age modulator, age trajectory modulator, Layer 3 buffer. Not applied at QB, RB, LB, DT, K. WR excluded for v1.0 (SL-022 — Option A). Flagged for calibration revisit in v1.1 if live data shows elite-RAS WRs are systematically under-protected.

**SL-020 — Low-tier RAS exclusion (QB and K)**
QB and K RAS component forced 1.000. K all three Layer 4 components forced 1.000. Layer 3, Layer 5, and Layer 2 carry all ranking work for these positions.

**SL-021 — DT Hybrid Tier Resolution Package**
Resolves SL-004 vs. SL-005 contradiction at DT. Three mechanics: (1) Medium film tier with SL-005 compression (cap ±3%, steepness 10.0), High-tier RAS treatment (cap ±8%, schedule 1.00/0.50/0.10); (2) Dynamic Year 1 / Year 2+ PFF EMA alpha (0.50 → 0.10 after first full NFL season); (3) Late-Career Cushion Guard — if Raw RAS ≥ 8.00, late-career penalty velocity reduced 10% (`cushioned = peak − (peak − base) × 0.90`). Applies to Layer 3 age_pull and breakout Age Trajectory sub-signal beyond peak. DT-unique replacement for SL-019.

**SL-022 — WR SL-019 exclusion (v1.0)**
WR excluded from SL-019 for v1.0 build. Layer 3 carries all WR aging. Closed from SL-OQ-043 (Option A). Flagged for calibration revisit in v1.1 — if live testing shows elite-RAS WRs (High-tier, longevity archetypes) are systematically under-protected, re-evaluate enabling SL-019 at CB/S strength (Option B) or reduced strength (Option C).

---

## Scoring Layer Open Questions (SL-OQ series)

SL-OQ-001 through SL-OQ-028 documented in individual rubrics and `Engine_Specification.md`. Session 2 additions:

### Closed

| ID | Resolution |
|---|---|
| SL-OQ-027 | SL-019 applicability gating rule. Applies when: (a) High-tier RAS AND (b) predictive relationship between athletic profile and longevity arc. Confirmed: TE, DE, CB, S. Excluded: QB, RB, LB, K (Low/Medium tier or scheme-driven), DT (Cushion Guard replaces). See DE_Rubric.md Section 7. |
| SL-OQ-040 | PAT/XP scoring. EP made = +1 pt, EM missed = -1 pt. Confirmed via MFL rules endpoint. |
| SL-OQ-041 | FG 70+ threshold. No separate tier. FG 60-99 covers all 60+ attempts at 6 pts. Confirmed via MFL rules endpoint. |

### Open

| ID | Topic | Position | Note |
|---|---|---|---|
| SL-OQ-029 | DE college production share data source — sack + TFL market share requires team total data | DE | Open |
| SL-OQ-030 | Mid-season MFL position re-tagging for hybrid-scheme defenders; recommend season-start classification lock | DE | Gemini-originated |
| SL-OQ-031 | SL-005 compression depth empirical calibration at LB | LB | Open |
| SL-OQ-032 | DL-dominated-system context normalization for LB college production share | LB | Gemini-originated |
| SL-OQ-033 | CB subjective scouting anchor strength — PFF + NGS dominance vs. thin TDN/RSP CB coverage | CB | Open |
| SL-OQ-034 | Targets-starved elite CB NGS interpretation ("Revis Island" problem) | CB | Gemini-originated |
| SL-OQ-035 | Box-safety vs. deep-safety rubric branching — split S into S_BOX and S_DEEP sub-rubrics? | S | Open |
| SL-OQ-036 | Role-specific sub-signal weighting within monolithic S rubric (alternative to SL-OQ-035) | S | Gemini-originated |
| SL-OQ-037 | Cushion Guard threshold — binary 8.00 vs. continuous RAS scaling | DT | Open |
| SL-OQ-038 | Dynamic Year 1 / Year 2+ PFF α propagation — should this become cross-rubric? | DT | Session 3 venue |
| SL-OQ-039 | Dynamic α down-shift trigger — when exactly does Year 1 end? | DT | Gemini-originated |
| SL-OQ-042 | Madden API ingestion pipeline optimization at K — archival-only data | K | Gemini-originated |
| SL-OQ-043 | WR SL-019 status — **RESOLVED: Option A (excluded for v1.0).** SL-022 assigned. Calibration revisit flagged for v1.1. | WR | CLOSED |

---

## Calibration Items (CAL series)

CAL-001 through CAL-021 documented in individual rubrics. Session 2 additions:

| ID | Topic | Position | Linked To |
|---|---|---|---|
| CAL-022 | DE SL-019 modulator strength empirical calibration (0.35/0.35/0.30 vs. asymmetric values) | DE | — |
| CAL-023 | DE college production share snap-count normalization | DE | SL-OQ-029 |
| CAL-024 | LB college production share weighting methodology (Tackle vs. Sack vs. TFL asymmetry) | LB | CAL-023 |
| CAL-025 | LB SL-005 compression effectiveness empirical tracking | LB | SL-OQ-031 |
| CAL-026 | CB NGS Coverage Metrics bundle definition methodology | CB | — |
| CAL-027 | CB RAS inflection appropriateness across CB archetype spectrum | CB | — |
| CAL-028 | S NGS metric bundle definition (parallel to CAL-026 at CB) | S | CAL-026 |
| CAL-029 | S college production share weighting and definition calibration | S | CAL-024 |
| CAL-030 | DT Cushion Guard threshold (8.00) and reduction strength (10%) empirical calibration | DT | SL-OQ-037 |
| CAL-031 | DT Cushion Guard behavior across in-career role transitions | DT | — |
| CAL-032 | K Madden archival data utility — future predictive value beyond tiebreaker? | K | SL-OQ-042 |

---

## Numbering State

| Series | Next Available | Last Used | Last Source |
|---|---|---|---|
| OQ- | 012 | OQ-011 | True Position tackle split |
| DECISION- | 011 | DECISION-010 | Historical score preservation |
| SL- | 023 | SL-022 | WR SL-019 exclusion v1.0 (SL-OQ-043 closed) |
| SL-OQ- | 044 | SL-OQ-043 | WR SL-019 status (CLOSED — Option A, SL-022 assigned) |
| CAL- | 033 | CAL-032 | K Madden archival utility |

---

## Session Log

| Date | Session Focus | Key Outputs |
|------|--------------|-------------|
| June 2026 | Project foundation | Context, rules, scoring spec, data layer, transaction reference, source library, league history, roadmap |
| June 2026 | Deliverable 2 Session 1 | WR, RB, TE, QB rubrics at v1.0; SL-018, SL-019, SL-020 locked; SL-OQ-001–028 |
| June 2026 | Deliverable 2 Session 2 | DE, LB, CB, S, DT, K rubrics at v1.0; SL-021 locked; SL-OQ-027 closed; 14 new SL-OQs; 11 new CALs |
| June 2026 | Document intake | UI architecture, backend architecture, Layer 4 pre-build audit, testing harness spec, MFL scoring rules decode; OQ-002/003/011/040/041 resolved |
