# Legacy NFL Fantasy — Project Master Index
Version: 1.0 — June 2026
Built by: Christopher Campbell + Claude (Anthropic)

## What This Is
This is the project folder for the Legacy NFL 32-Team Dynasty Ranking Engine and Application. The end goal is a full-stack fantasy football application that ingests live data from MyFantasyLeague (MFL), processes it through a custom valuation engine built on this league's official ruleset, and surfaces intelligence that helps GMs make better decisions — and helps commissioners run a cleaner league.

The build follows three phases:
- Personal tool — Christopher's competitive edge, processing all 32 teams
- Alpha — League-wide access, league mates as testers
- Beta / Public — Open to other leagues running similar formats

The build has no defined endpoint. It grows as the league grows.

## Stack

Go (backend, scoring engine) + Wails v2 (desktop shell) + React + Tailwind + Zustand (frontend). SQLite WAL. Phase 1 is a desktop-only personal tool. Phase 2 adds Cloudflare Tunnel for league-wide access.

## Folder Structure

| Folder | Contents |
|---|---|
| docs/league-rules/ | Official rulebook, scoring reference, contract mechanics |
| docs/scoring-engine/ | Ranking algorithm, scoring math, scouting layer, age decay, 10 position rubrics |
| docs/data-layer/ | MFL API reference, data pipeline specs, scoring rules decode |
| docs/transactions/ | Bidding, RFA, waivers, trades, DOT process — rules + human behavior |
| docs/modules/ | Power rankings, matchup predictions, future module specs |
| docs/sources/ | Approved external data sources with descriptions and use cases |
| docs/league-history/ | Curated historical examples — trades, bids, vetoes, human behavior |
| docs/roadmap/ | Build phases, open questions, known risks, future features |
| docs/ui/ | UI architecture, density system, nav structure, component specs, locked decisions |
| docs/backend/ | Backend architecture, schema domains, service layer, API contracts |
| docs/build-handoffs/ | Pre-build audits, testing harness specs, ready-to-build packages |

## How to Use This Folder in a New Session

1. Read docs/league-rules/Official_Rulebook.md before any scoring or contract work.
2. Load the specific folder relevant to the session's task.
3. Check docs/roadmap/Roadmap_and_Open_Questions.md for unresolved items that may affect the work.
4. For engine build sessions: load docs/build-handoffs/Layer4_PreBuild_Audit.md — it is the authoritative pre-build reference for all 10 position rubrics.
5. For UI/frontend sessions: load docs/ui/UI_Direction_Document.md and docs/backend/Backend_Architecture.md.
6. For testing sessions: load docs/build-handoffs/Testing_App_Specification.md.

Do not assume prior session knowledge. Load the documents. Then work.

## Project Status

**Documentation phase complete. Pre-build phase.**

- [x] Context brief complete
- [x] Official rulebook ingested and verified
- [x] Scoring engine specification complete (six-layer architecture)
- [x] All 10 position rubrics built and locked at v1.0
- [x] MFL API reference and scoring rules decoded
- [x] Source library approved and locked
- [x] Transaction system documented
- [x] Human transaction behavior documented
- [x] UI architecture complete (docs/ui/)
- [x] Backend architecture complete (docs/backend/)
- [x] Layer 4 pre-build audit complete (docs/build-handoffs/)
- [x] Testing harness specification complete (docs/build-handoffs/)
- [x] Session 3 audit pass — Engine_Specification.md v3.0 + Universal_Rubric_Template.md v1.2 (SL-018 through SL-022)
- [ ] Testing harness build (Layer 4 validation before full build)
- [ ] MFL API data pipeline — first build
- [ ] Scoring engine — Go implementation against rulebook
- [ ] Transaction validation layer — first build
- [ ] Power rankings module — first build
- [ ] In-app bidding system — first build
