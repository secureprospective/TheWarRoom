# Legacy NFL Fantasy Application — North Star
Version: 1.2 — June 2026
Status: Strategic navigation document. Read first in every future session. The technical specifications are authoritative on implementation detail. This document is authoritative on purpose, scope, and how the pieces connect.

Changelog: Version 1.2 corrects source count and updates Phase 3 for self-hosted model. Version 1.1 nuanced Principle 1 and added Principle 2 on year-over-year configurability.

## Mission
Build a fantasy football application that processes the Legacy NFL's 32-team dynasty format with the depth its ruleset deserves. Surface intelligence GMs cannot get from MyFantasyLeague's stock interface. Remove the mechanical friction that consumes commissioner and DOT time. Make ownership of a Legacy NFL franchise easier, more rewarding, and more likely to retain engaged owners.

The application has no defined endpoint. It grows as the league grows.

## Who This Serves
- Phase 1 — Christopher (personal tool). Competitive edge for one GM processing all 32 teams.
- Phase 2 — Legacy NFL league mates (alpha). Multi-user access. The league becomes the test environment.
- Phase 3 — Any MFL league (self-hosted release). Each league installs and runs their own server instance. Configurable for any MFL league's scoring and roster rules. League members connect to their own league's self-hosted instance.
- Phase 4 — Long-term scope (no timeline). Computer-use automation, historical analytics, offseason cap planner, full draft pick value model.

## The Four Pillars
The application has four distinct systems. Each one is independently valuable. Together they replace the Proboards-plus-spreadsheets stack that GMs use today.

### Pillar 1: The Scoring Engine
The valuation brain. Six layers process MFL data plus external sources into a single Adjusted Score per player.

- Layer 1 — Data hygiene and contract floor enforcement
- Layer 2 — Rulebook scoring matrix (MFL-sourced, not hardcoded)
- Layer 3 — Age decay (position-specific, admin-tunable)
- Layer 4 — Scouting layer (film × RAS × breakout with Madden regulation)
- Layer 5 — Contract efficiency scaling (cap tier multiplier, percentage-based)
- Layer 6 — Deterministic tiebreaker (tenure → RAS → positional scarcity)

The engine is the value proposition. Every module surfaces some view of its output.

### Pillar 2: The Transaction System
Rule enforcement and mechanical workflow replacement. Bid validation, dead cap auto-calculation, 24-hour clock management, snipe detection, trade deadline hard block, DOT vote tracking.

Phase 1 generates correctly formatted transaction outputs that GMs execute manually on MFL. Phase 2 adds authenticated MFL write access. Phase 4 considers full computer-use automation.

The transaction system does not replace human judgment. It removes the mechanical tasks that consume commissioner and DOT time so they can focus on governance.

### Pillar 3: The Application Modules
The views into the engine's output and the transaction system's state.

- Module 1 — 32-team asset rankings
- Module 2 — Weekly power rankings
- Module 3 — Matchup score predictions
- Module 4 — In-app transaction system (the Pillar 2 interface)
- Module 5 — Free agency intelligence
- Module 6 — Rookie draft intelligence
- Module 7 — Trade analyzer
- Module 8 — Commissioner dashboard

Each module is a separate build session. Module specifications live in docs/modules/Module_Specifications.md.

### Pillar 4: The Admin Console
The calibration interface. The engine's accuracy depends on parameters that cannot be perfectly set from theory alone — they must be tuned against real data and real results. The admin console exposes those parameters so calibration happens through the UI, not through code changes.

**Exposed in standard admin UI (per SL-017):**
- Sub-signal weights across all engine components
- S-curve cap values and shape parameters
- Madden regulation threshold and blend parameters
- EMA blend rates and season transition behavior per dynamic sub-signal
- Madden attribute mapping tables per position
- Position weight values per component
- Cap tier boundary percentages (Layer 5)
- Layer 2 scoring value overrides (default MFL-sourced)
- Layer 3 peak limits and decay rate
- Layer 6 positional scarcity matrix
- Sub-signal normalization parameters

**Code-locked (structural — not exposed in standard UI):**
- Multiplicative combination of three Layer 4 components
- Six-layer engine pipeline order
- S-curve mechanic itself (Shape B / sigmoid family)
- Approach A aggregation structure
- Approach D Madden regulation mechanism
- Three-state data flag mechanic (Present / Absent / Unknown)
- MFL-sourcing of league-configurable values (DECISION-009)
- Historical preservation policy (DECISION-010)

Developer-mode access: All structural parameters tunable for advanced calibration. Not exposed to standard league users.

The admin console is what makes the engine adaptive without recoding. It is also what makes Phase 2 and Phase 3 viable — different leagues running different formats need different defaults, and the admin console is how each league tunes its instance.

## The North Star Principles

### 1. The Rulebook Is Authoritative for Current-Year Rules
All scoring values, contract mechanics, and transaction rules derive from docs/league-rules/Official_Rulebook.md for the current year. The rulebook is updated whenever the league votes on changes. The engine adapts to those changes by pulling configurable values from MFL on startup and at season transitions (see Principle 2). The rulebook stays authoritative because MFL's league config IS the rulebook expressed in machine-readable form.

When in doubt about a current rule, the rulebook wins. When in doubt about an implementation, MFL's league endpoint wins. The two should agree; when they do not, the rulebook is corrected to match what the league has actually voted on.

### 2. The Engine Adapts to Year-Over-Year Configuration Changes Without Code Releases
Leagues vote on rule changes between seasons. Scoring values shift. Salary caps grow. Position-specific contract floors get adjusted. The engine must absorb those changes without requiring a code release. League-configurable values (Layer 2 scoring, cap, minimums, roster sizes, lineup format, contract floors, tender prices, rookie slots) are pulled from MFL's league endpoint, not hardcoded.

Engine-tunable values that are NOT in MFL config (Layer 3 age decay, Layer 5 cap tier boundaries as percentages, Layer 6 scarcity matrix, Layer 4 component parameters) are exposed through the admin console for calibration without code changes.

Historical scores are preserved per DECISION-010 — when scoring rules change, the prior season's scores stay as they were calculated under the old rules. Year-over-year comparisons reflect the actual rules in effect for each season.

### 3. The Engine Integrates Expert Output; It Does Not Replicate Expert Reasoning (SL-014)
Where the approved expert sources already account for real-world dynamics — NIL, transfer portal, scheme changes, league-wide production shifts — the engine does not engineer parallel mechanisms. The film component is the integration point for those dynamics. Mechanical components (breakout age, school tier, RAS, college usage rate) stay mechanical. The expert community does the dynamic interpretation work. The engine ingests it.

### 4. Stats and Analytics Self-Regulate; Subjective Claims Need Confirmation (SL-016)
Numerical signals (NFL production, PFF grades, IDP Guru analytics) flow through the film component without external regulation. They self-regulate by being measurements rather than judgments. Subjective expert claims (RSP qualitative descriptions, TDN/Sharp/IDP Show qualitative takes) are regulated by Madden at the attribute level via Approach D. Human enthusiasm and over-reporting are the things Madden checks; numerical methodology is not.

### 5. The Application Frees Human Judgment to Do What Only Humans Can Do
The trade deadline timestamp is binary. The dead cap calculation is arithmetic. The bid increment is a comparison. The application handles all of those. It does not replace DOT judgment on trade value, commissioner judgment on league health, or GM judgment on team direction. Mechanical tasks are automated. Judgment tasks are surfaced with the best possible information.

### 6. Phase 1 Is Read-Only Against MFL
All transactions are generated by the application and executed manually on MFL. Authenticated write integration is Phase 2. Computer-use click-through automation is Phase 4 scope.

### 7. Approved Sources Are Locked
The twenty-one sources in docs/sources/Approved_Sources.md are the data foundation. Adding new sources requires explicit approval from Christopher. Scraping Proboards is permanently off the table.

### 8. The Admin Console Is the Calibration Layer
Parameters that need to be set from theory alone are dangerous. Parameters that ship with sensible defaults and are tunable against real outputs through the admin console are safe. The default is to expose any parameter where real-world calibration will improve accuracy. Code-lock only structural mechanics.

### 9. Multi-AI Collaboration Is the Architecture
Christopher cannot code without collaboration. Claude cannot always help Christopher alone. Gemini is a legitimate tool in the practice. Christopher is the bridge between AI tools — his speed at moving information between them is an asset. Future sessions should expect to integrate output from other AI tools that Christopher brings in.

## How the Pieces Connect

```
                        [ MFL API ]
                              |
                              v
                   [ Data Ingestion Layer ]
                              |
                              v                     <-- League config (scoring,
                   [ Data Normalization Layer ]        cap, minimums) pulled
                              |                         here per DECISION-009
                              v
                  +------------------------+
                  |    SCORING ENGINE      |
                  |  L1: Data hygiene      |
                  |  L2: Rulebook scoring  |    <-- Values MFL-sourced
                  |  L3: Age decay         |    <-- Admin-tunable
                  |  L4: Scouting layer    |    <-- Admin-tunable
                  |  L5: Cap efficiency    |    <-- Percentage-of-cap, tunable
                  |  L6: Tiebreaker        |    <-- Scarcity matrix tunable
                  +------------------------+
                              |
                              v                     <-- Historical scores
                  [ Per-Season Output Store ]         preserved per
                              |                         DECISION-010
                              v
              +-----------------------------+
              |   APPLICATION MODULES       |
              |                             |
              |   - Rankings (M1)           |
              |   - Power Rankings (M2)     |
              |   - Matchups (M3)           |
              |   - Transactions (M4) ------|----> Transactions also feed
              |   - Free Agency (M5)        |      back into engine state
              |   - Rookie Draft (M6)       |
              |   - Trade Analyzer (M7)     |
              |   - Commissioner (M8)       |
              +-----------------------------+
                              |
                              v
                         [ Users ]
                              |
                              v
                    [ MFL (manual execution
                       in Phase 1) ]
```

## How to Navigate This Project

| Need | Document |
|---|---|
| Strategic overview (this) | North_Star.md |
| Rulebook (authoritative for current year) | docs/league-rules/Official_Rulebook.md |
| Scoring engine math | docs/scoring-engine/Engine_Specification.md |
| Layer 4 rubric structure | docs/scoring-engine/Universal_Rubric_Template.md |
| MFL API reference | docs/data-layer/MFL_API_Specification.md |
| Transaction rules + human behavior | docs/transactions/Transaction_Reference.md |
| Module designs | docs/modules/Module_Specifications.md |
| Approved data sources | docs/sources/Approved_Sources.md |
| Historical examples | docs/league-history/League_History_v1.md |
| Roadmap, open questions, locked decisions, calibration backlog | docs/roadmap/Roadmap_and_Open_Questions.md |

## What This Application Is Not

- It is not a generic fantasy football platform.
- It is not a replacement for human league governance.
- It is not a betting tool, daily fantasy tool, or season-long redraft tool.
- It is not a content site.
- It is not an AI agent that makes decisions for GMs.
- It is not version-frozen against the rulebook.

---

Built by: Christopher Campbell + Claude (Anthropic)

| Version | Date | Changes |
|---|---|---|
| 1.0 | June 2026 | Initial release. Captures strategic frame after Layer 4 scouting layer architecture session. Establishes the four pillars and the eight North Star principles. |
| 1.1 | June 2026 | Structural pass update. Principle 1 nuanced for current-year scope. New Principle 2 added on year-over-year configurability without code releases. References to DECISION-009 (MFL-sourced config) and DECISION-010 (historical preservation) integrated throughout. Connection diagram updated to reflect MFL config sourcing and per-season output store for historical preservation. |
| 1.2 | June 2026 | Document audit pass. Phase 3 updated to self-hosted per-league model. Christopher_in_Context.md navigation row removed (superseded by global v2.0). Source count corrected to twenty-one (Madden added to Approved_Sources.md; prior count of seventeen/eighteen was inaccurate). |
