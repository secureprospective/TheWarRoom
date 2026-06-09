# Legacy NFL Fantasy — Project CLAUDE.md
**Version:** 1.0 — June 2026
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

**Immediate next steps (in order):**
1. ~~Session 3 audit pass~~ — **COMPLETE.** Engine_Specification.md v3.0, Universal_Rubric_Template.md v1.2. All SL-018–SL-021, NGS anchor, and document cleanup applied.
2. Build and validate Layer 4 testing harness per `docs/build-handoffs/Testing_App_Specification.md`.
3. First code build: Go engine skeleton + MFL data pipeline.

**No blocking Christopher decisions. Layer 4 build can proceed.**
- SL-022: WR SL-019 excluded for v1.0 (SL-OQ-043 closed, Option A). Layer 3 carries WR aging. Calibration revisit flagged for v1.1.

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

---

## Open Items at Last Session Close

| Item | Status |
|---|---|
| OQ-001 MFL League ID | OPEN — Christopher supplies |
| OQ-004 EDGE position mapping source | OPEN |
| OQ-005 Salary adjustment line item | OPEN |
| OQ-006 Cap tier calibration | OPEN — resolve after live data |
| OQ-007 Scouting layer weight | OPEN — resolve after testing |
| OQ-008 Franchise tag calculation timing | OPEN |
| OQ-009 RFA eligibility window | OPEN |
| OQ-010 Playoff bid rules trigger | OPEN — needs commissioner confirmation |
| SL-OQ-043 WR SL-019 status | CLOSED — Option A; SL-022 assigned; v1.1 calibration revisit flagged |
| Session 3 audit pass | COMPLETE — Engine_Spec v3.0, Universal_Rubric_Template v1.2 |

---

*Built by: Christopher Campbell + Claude (Anthropic)*
