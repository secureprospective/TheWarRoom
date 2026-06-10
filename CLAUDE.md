# Legacy NFL Fantasy — Project CLAUDE.md
**Version:** 1.2 — June 2026
**Project path:** `/mnt/storage/claudebox/projects/legacy-nfl-fantasy/`
**Pillars:** Business, Technical

---

## What This Project Is

A 32-team dynasty fantasy football ranking engine and full-stack desktop application for the Legacy NFL league. Six-layer scoring engine (Go), Wails v2 desktop shell, React + Tailwind + Zustand frontend, SQLite WAL storage, MFL API integration.

**Three-phase delivery:** Personal tool (Phase 1, current) → League-wide alpha (Phase 2) → Public beta (Phase 3).

---

## Current Build State (June 2026)

**Documentation phase complete. Pre-build phase.**

All architecture documents are on disk. Engine specification, 10 position rubrics, UI architecture, backend architecture, Layer 4 pre-build audit, and testing harness specification are complete. MFL scoring rules decoded and verified.

**Session 4 completions:**
- ~~Token-tightening pass~~ — **COMPLETE.** 7 edits across 4 files. All 28 docs surveyed.
- ~~SL-OQ-043~~ — **CLOSED.** Option A. SL-022 assigned. WR SL-019 excluded for v1.0; calibration revisit flagged for v1.1.
- ~~Folder restructure~~ — **COMPLETE.** All numbered folders → `docs/` with semantic kebab-case names. All path references updated across all files.
- ~~GitHub repo~~ — **LIVE.** https://github.com/secureprospective/TheWarRoom (main, initial commit, 28 files).

**Session 5 completions:**
- ~~Planning arc Session 1 (Logical Building Blocks)~~ — **COMPLETE.** 26 blocks defined across 4 tiers. Three-layer architectural law locked. All structural decisions recorded. Full block list at `/root/.claude/plans/yes-each-module-is-immutable-wigderson.md`.

**Session 6 completions:**
- ~~Planning arc Session 2 (Build Sequencing)~~ — **COMPLETE.** 32-session ordered build plan with dependency gates, six structural wireframes, and open flags. Full plan at `/root/.claude/plans/very-good-now-i-replicated-feigenbaum.md`. Self-audited (11 corrections applied) before handoff to Opus 4.8 for Session 3 audit.

**Session 7 completions:**
- ~~Planning arc Session 3 (Build Sequencing Audit, Opus 4.8)~~ — **COMPLETE.** Verdict PASS WITH FLAGS; 25 decisions locked (AD-01–AD-25), reviewed piece-by-piece with Christopher. Audit at `/root/.claude/plans/session-3-audit-build-sequencing.md`. Build plan grew 32→38 sessions (B7 split into B7a–d; B2b split schema-first + 3 fetcher groups; QB-first/DT-second rubric order).
- ~~Three Session 2 open flags~~ — **RESOLVED.** Per-team roster view → M1 drill-down (AD-20); B7 sub-session design → foundation-first split (AD-02); Go overlay label → corrected, reframed as net-new authoring (AD-19).

**Forward source of truth:** `docs/build-handoffs/Build_Tracker.md` — the 38-session checkable sequence. Read at the start and end of every build session. Every session closes by writing the next session's handoff per `docs/build-handoffs/Handoff_Protocol.md`.

**Immediate next steps (in order):**
1. Author the **Go overlay** in `christopher-coding-standards` (golangci-lint ruleset, Go pre-commit config, Go naming conventions). Net-new — does not exist yet; repo has the TypeScript overlay only. **B0 cannot start until this is merged.**
2. Resolve OQ-004 (EDGE mapping) and OQ-005 (salary adjustment) before B3.
3. Build and validate the Layer 4 testing harness per `docs/build-handoffs/Testing_App_Specification.md` (Session 13).
4. First code build: B0 — Project Scaffold (Session 1).

**Next branch:** TBD at next session start (this audit closed on `session/planning-arc-audit`).

**SL-022 still active:** WR SL-019 excluded for v1.0 (SL-OQ-043 closed, Option A). Layer 3 carries WR aging. Calibration revisit flagged for v1.1.

---

## Session Start Protocol

1. Confirm branch: `git branch --show-current` — never work on main
2. Read `docs/roadmap/Roadmap_and_Open_Questions.md` — check open OQs that affect the session's work
3. Load the folder(s) relevant to the task:
   - Engine work → `docs/scoring-engine/` + `docs/build-handoffs/Layer4_PreBuild_Audit.md`
   - UI work → `docs/ui/UI_Direction_Document.md` + `docs/backend/Backend_Architecture.md`
   - Data pipeline work → `docs/data-layer/`
   - Transaction work → `docs/transactions/`
   - Testing → `docs/build-handoffs/Testing_App_Specification.md`
4. Build command: not yet defined (pre-first-build). Will be added when the Go module is initialized.

---

## Hard Constraints (Never Route Around)

- **No work on main. Ever.** Branch naming: `session/<short-description>`.
- **MFL Player IDs are always string type.** IDs under 1000 require leading zeros. Enforce at ingestion boundary. All SQLite columns TEXT.
- **Confidence scores are internal engine flags only.** Never surface in UI.
- **Layer 2 / Layer 4 zero scoring leaks.** No sub-signal within Film, RAS, or Breakout may reference fantasy points, projected volume, MFL scoring config, or format-dependent volume stats.
- **Layer 4 structural mechanics never exposed in Admin UI.** Only admin-tunable parameters per SL-017.
- **NGS Coverage Metrics anchor applies at CB and S only.** Excluded at all other positions — critical check.
- **SL-019 not applied at DT.** Cushion Guard replaces it. Running both would double-protect.
- **Do not reopen locked decisions.** If a locked decision creates a technical constraint that feels wrong, flag it to Christopher — do not route around it silently.
- **Do not add features not in the documents** without Christopher's explicit direction.
- **Never use `git --no-verify`.**

---

## Key Document Map

| Need | Document |
|---|---|
| Full engine architecture | `docs/scoring-engine/Engine_Specification.md` |
| Any position's rubric | `docs/scoring-engine/<POSITION>_Rubric.md` |
| All rubric parameters in one place | `docs/build-handoffs/Layer4_PreBuild_Audit.md` (Section 1C) |
| Testing harness spec | `docs/build-handoffs/Testing_App_Specification.md` |
| MFL API calls | `docs/data-layer/MFL_API_Specification.md` |
| MFL scoring rules | `docs/data-layer/MFL_Scoring_Rules_Decode.md` |
| UI architecture | `docs/ui/UI_Direction_Document.md` |
| Backend architecture | `docs/backend/Backend_Architecture.md` |
| Open questions + decisions | `docs/roadmap/Roadmap_and_Open_Questions.md` |
| Build sequence + progress (checkable) | `docs/build-handoffs/Build_Tracker.md` |
| Session handoff routine + template | `docs/build-handoffs/Handoff_Protocol.md` |
| Session 3 audit (decisions AD-01–25) | `/root/.claude/plans/session-3-audit-build-sequencing.md` |

---

## Open Items at Last Session Close

| Item | Status |
|---|---|
| OQ-001 MFL League ID | RESOLVED — League ID 14432, host www47 (synced to roadmap, AD-24) |
| OQ-004 EDGE position mapping source | OPEN |
| OQ-005 Salary adjustment line item | OPEN |
| OQ-006 Cap tier calibration | OPEN — resolve after live data |
| OQ-007 Scouting layer weight | OPEN — resolve after testing |
| OQ-008 Franchise tag calculation timing | OPEN |
| OQ-009 RFA eligibility window | OPEN |
| OQ-010 Playoff bid rules trigger | OPEN — needs commissioner confirmation |
| SL-OQ-043 WR SL-019 status | CLOSED — Option A; SL-022 assigned; v1.1 calibration revisit flagged |
| Session 3 audit pass | COMPLETE — Engine_Spec v3.0, Universal_Rubric_Template v1.2 |
| Planning arc Session 1 | COMPLETE — 26 blocks defined. Plan at `/root/.claude/plans/yes-each-module-is-immutable-wigderson.md` |
| Planning arc Session 2 | COMPLETE — 32-session build plan + 6 wireframes. Plan at `/root/.claude/plans/very-good-now-i-replicated-feigenbaum.md` |
| Planning arc Session 3 | COMPLETE — PASS WITH FLAGS; 25 decisions AD-01–25 locked; `/root/.claude/plans/session-3-audit-build-sequencing.md` |
| Go overlay (Phase 2 in coding-standards repo) | OPEN — net-new authoring (does not exist yet; repo has TS overlay only); critical-path prereq before B0 (AD-19) |
| Per-team roster view — block assignment | RESOLVED — M1 drill-down (already specced as M1 output); M1b split only if scope tightens (AD-20) |
| B7 sub-session design | RESOLVED — split foundation-first into B7a (coordinator) + B7b/c/d by mechanic group (AD-02) |

---

*Built by: Christopher Campbell + Claude (Anthropic)*
