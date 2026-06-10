# TheWarRoom — Build Tracker
**Version:** 1.0 — June 2026
**Status:** Operational. The forward source of truth for the build. Check a session off only when its close gate fully passes.
**Origin:** Session 2 sequencing plan (`very-good-now-i-replicated-feigenbaum.md`) as corrected by the Session 3 audit (`session-3-audit-build-sequencing.md`). Audit decisions referenced as `AD-##`.

This is the document opened at the start and end of every build session. One line per session. Read the legend once; it governs the whole build.

---

## Legend

**Status markers** (the checkbox):

| Mark | Meaning |
|---|---|
| `[ ]` | Not started |
| `[~]` | In progress (branch open, code being written) |
| `[x]` | **Complete** — all three close gates passed (see below) |
| `[!]` | Blocked — an open question or upstream gate is unresolved |

**A session is `[x]` only when all three close gates pass:**
1. **Build green** — `golangci-lint` clean, `go test ./...` (and `pnpm test` for frontend) pass. No `--no-verify`, ever.
2. **Functional verification** — Christopher used the actual behavior in the real app, not just "build succeeded."
3. **Handoff written** — the next session's handoff prompt is produced per `Handoff_Protocol.md` and saved, ready to paste when the session is cleared.

**Column meaning:**
- **Gate** — what must be true *before* the session starts (upstream dependency + any open question that blocks it).
- **WF** — the wireframe the session's code must match (anti-spaghetti contract). `—` = no code wireframe (dev artifact / scaffold has its own).
- **L** — architectural layer (1 = Real Football read-only, 2 = Logic Engine, 3 = User Interaction, — = infra/dev).

**Standing rules every session inherits** (from AGENTS.md + the three-layer law):
- File target < 250 lines, hard cap 400. Refactor over.
- All external input schema-validated at the boundary before business logic.
- Parameterized SQL only. No hardcoded secrets/IPs.
- No layer mixing. Modules never write B3c except through B7. Stores never import each other.

---

## Pre-Build Gate (must close before Session 1)

- `[ ]` **G0 — Go overlay** — Author and merge the Go overlay in `christopher-coding-standards` (golangci-lint ruleset, Go pre-commit config, Go naming conventions). **Net-new authoring — does not exist yet** (AD-19). B0's tooling spec cannot be locked without it. Fix the Phase-label mismatch (CLAUDE.md "Phase 1D" vs repo "Phase 2") as housekeeping.

---

## Build Sequence (38 sessions)

### Tier 0 — Scaffold

| # | Status | Session | Scope | Gate | WF | L |
|---|---|---|---|---|---|---|
| 1 | `[ ]` | **B0 — Project Scaffold** | Go module, Wails v2, React+Tailwind+Zustand, SQLite WAL lifecycle hooks, golangci-lint + pre-commit, AGENTS.md + SYSTEM_MAP.md, IPC ping-pong. Locks the 5 patterns (AD-03). | G0 closed | B0 | — |

### Tier 1 — Data Pipeline (Layer 1)

| # | Status | Session | Scope | Gate | WF | L |
|---|---|---|---|---|---|---|
| 2 | `[ ]` | **B1 — MFL API Client** | HTTP transport only: rate limiting, backoff, host routing. One exported `Do()`. No domain types. | B0 | 1A | 1 |
| 3 | `[ ]` | **B2 — MFL Data Ingestion** | Fetch schedule, raw record storage. Consumes B1. | B1 · **OQ-005** | 1B | 1 |
| 4 | `[ ]` | **B3 — MFL Data Normalization** | Raw → typed internal records. RISK-003 via `internal/playerid` (AD-06). Internal types locked — **reviewed deliverable** (AD-18). | B2 · **OQ-004, OQ-005** | 1C | 1 |
| 5 | `[ ]` | **B2b-Schema — Scouting Schema** | Design + lock the unified scouting schema, all 10 positions. **Human review gate before Session 14** — walk every position's inputs; decide SL-OQ-035/036 field reservation (AD-16). | B1 · B3 *(design-ordering, not code dep — AD-13)* | 1B | 1 |
| 6 | `[ ]` | **B2b-Fetch-Offense** | Scouting fetchers for offense-relevant sources (unblocks QB/RB/WR/TE). | B2b-Schema | 1B | 1 |
| 7 | `[ ]` | **B2b-Fetch-Defense** | Scouting fetchers for defense-relevant sources (NGS for CB/S, PFF defense, college production). | B2b-Schema | 1B | 1 |
| 8 | `[ ]` | **B2b-Fetch-Kicker/Archival** | K sources + Madden archival-only (SL-OQ-042). | B2b-Schema | 1B | 1 |

### Tier 2 — Logic Engine (Layer 2)

| # | Status | Session | Scope | Gate | WF | L |
|---|---|---|---|---|---|---|
| 9 | `[ ]` | **B3b — League Rulebook** | MFL-sourced rules + delta overrides. Pure data access. No rule logic. | B3 | 2 | 2 |
| 10 | `[ ]` | **B3c — League State Store** | 32-team mutable state. `StateWriter` to B7 only; `StateReader` to all else. `Initialize()` **pulls** from B3 (B3 never pushes). No `Reload()`. | B3 | 2 | 2 |
| 11 | `[ ]` | **B4 — Admin Parameter Store** | Engine calibration params (incl. cap-tier **percentages** — calibration, not cap amounts; AD-21). Ships defaults. | B0 | 2 | 2 |
| 12 | `[ ]` | **B5a — Engine Pipeline** | L1–L3 + L5–L6 orchestration; L4 pluggable dispatch. Pure functions. **Pre-check L2 < 400 lines; pre-split if not** (AD-17). | B3b · B3c · B4 | 3 | 2 |
| 13 | `[ ]` | **Testing Harness** | Layer 4 test harness per `Testing_App_Specification.md`. **Hard gate — no rubric starts without it.** | B5a | — | — |

### Tier 2 — L4 Position Rubrics (QB first, DT second, then resume — AD-15)

| # | Status | Session | Scope | Gate | WF | L |
|---|---|---|---|---|---|---|
| 14 | `[ ]` | **B5b-QB** | `engine/l4/offense`. `scoreRAS` forced 1.000 (SL-020); Film + Breakout active. Skeleton-setter. | Harness · B5a · B2b-Off | 4 | 2 |
| 15 | `[ ]` | **B5b-DT** | `engine/l4/defense`. SL-021 hybrid tier + Cushion Guard + dynamic PFF α + SL-005. SL-019 excluded. **Escape hatch** allowed (AD-14). Carries **SL-OQ-037, SL-OQ-039** (AD-22). Stress-test. | Harness · B5a · B2b-Def | 4 | 2 |
| 16 | `[ ]` | **B5b-RB** | `engine/l4/offense`. Standard Film × RAS × Breakout. | Harness · B5a · B2b-Off | 4 | 2 |
| 17 | `[ ]` | **B5b-WR** | `engine/l4/offense`. **`scoreRAS` computes the High-tier RAS curve with SL-018 decay; SL-019 NOT applied (SL-022); do NOT force to 1.000** (AD-09). | Harness · B5a · B2b-Off | 4 | 2 |
| 18 | `[ ]` | **B5b-TE** | `engine/l4/offense`. SL-019 modulators active in `scoreRAS`/`scoreBreakout`. | Harness · B5a · B2b-Off | 4 | 2 |
| 19 | `[ ]` | **B5b-DE** | `engine/l4/defense`. SL-019 modulator. SL-OQ-029 carried. | Harness · B5a · B2b-Def | 4 | 2 |
| 20 | `[ ]` | **B5b-LB** | `engine/l4/defense`. SL-005 compression in `scoreFilm`. SL-019 excluded. | Harness · B5a · B2b-Def | 4 | 2 |
| 21 | `[ ]` | **B5b-CB** | `engine/l4/defense`. NGS anchor (**CB+S only**). SL-019 modulator. | Harness · B5a · B2b-Def | 4 | 2 |
| 22 | `[ ]` | **B5b-S** | `engine/l4/defense`. NGS anchor. SL-019 modulator. SL-OQ-035/036 carried. | Harness · B5a · B2b-Def | 4 | 2 |
| 23 | `[ ]` | **B5b-K** | `engine/l4/kicker`. All three components return 1.000; **`combine` yields 1.000, not special-cased** (AD-10). | Harness · B5a · B2b-K/A | 4 | 2 |

### Tier 2 — Output Store

| # | Status | Session | Scope | Gate | WF | L |
|---|---|---|---|---|---|---|
| 24 | `[ ]` | **B6 — Per-Season Output Store** | Persist engine outputs, `scoring_config_id` tagged. **Immutability both ways: append-only API + SQLite UPDATE/DELETE trigger** (AD-04, DECISION-010). | B5a · Sessions 14–23 | B6 | 2 |

### Tier 3/2 — First Module, then Transaction Engine

| # | Status | Session | Scope | Gate | WF | L |
|---|---|---|---|---|---|---|
| 25 | `[ ]` | **M1 — Asset Rankings** | 32-team rankings view + **per-team roster drill-down** (AD-20). First visible engine validation. Split **M1b** only if scope tightens. | B6 · B3c | 5 | 3 |
| 26 | `[ ]` | **B7a — Transaction Foundation + Coordinator** | `TransactionHandler` interface (designed to accept Phase-2 clock/snipe — AD-01); `deadcap/calculator.go`; **the sole-writer coordinator** (only `StateWriter`; atomic parameterized B3c write; handlers get `StateReader` only). Built + tested first (AD-02). | B6 · B3b · B3c | 6 | 2 |
| 27 | `[ ]` | **B7b — Acquisitions** | UFA, RFA, waivers. Validation + calc + formatted output. No live clock (Phase 2). | B7a · **OQ-009, OQ-010** | 6 | 2 |
| 28 | `[ ]` | **B7c — Trades** | Trades + trade block. Two-team cap, pick windows, Week-9 deadline hard block, format validation. DOT voting deferred (Phase 2). | B7a | 6 | 2 |
| 29 | `[ ]` | **B7d — Contract Mechanics** | Franchise tag, extension, restructure, buyout. Shared dead-cap/cap math. | B7a · **OQ-008** | 6 | 2 |

### Tier 3 — Remaining Modules

| # | Status | Session | Scope | Gate | WF | L |
|---|---|---|---|---|---|---|
| 30 | `[ ]` | **M4 — Transaction UI** | In-app transaction interface. Submissions route through B7 App methods only. Completes Phase 1 critical path. | B7a–d · B3c | 5 | 3 |
| 31 | `[ ]` | **M2 — Power Rankings** | Weekly power rankings view. | B6 · B3c | 5 | 3 |
| 32 | `[ ]` | **M5 — Free Agency Intelligence** | Free agent pool with scores. | B6 · B3c | 5 | 3 |
| 33 | `[ ]` | **M3 — Matchup Predictions** | Matchup score prediction view. | B6 · B3c | 5 | 3 |
| 34 | `[ ]` | **M6 — Rookie Draft Intelligence** | Rookie draft view + rankings. | B6 · B3c | 5 | 3 |
| 35 | `[ ]` | **M7 — Trade Analyzer** | Trade value analysis. | B6 · B3c · B7 | 5 | 3 |
| 36 | `[ ]` | **M8 — Commissioner Dashboard** | Commissioner tools, league health. | B7 · B6 · B3c | 5 | 3 |
| 37 | `[ ]` | **M9a — Engine Calibration UI** | B4 tuning surface; admin write path (not via B7 — AD-05). Christopher-only. | B4 | 5 | 3 |
| 38 | `[ ]` | **M9b — Commissioner Rules UI** | B3b governance surface; admin write path (not via B7 — AD-05). | B3b | 5 | 3 |

---

## Open Question Gates (must resolve before the named session)

| OQ | Topic | Blocks |
|---|---|---|
| OQ-004 | EDGE position mapping source | Session 4 (B3) |
| OQ-005 | Salary adjustment line item | Sessions 3–4 (B2, B3) |
| OQ-009 | RFA eligibility window | Session 27 (B7b) |
| OQ-010 | Playoff bid rules trigger | Session 27 (B7b) |
| OQ-008 | Franchise tag calculation timing | Session 29 (B7d) |

OQ-006 (cap tier calibration) and OQ-007 (scouting weight) are **post-live calibration, not build gates** — the engine runs on shipped B4 defaults.

---

## Conditional Splits (decide at scoping, not now)

- **M1b** — per-team roster view as its own session, only if M1 (Session 25) proves too large.
- **L2 sub-split** — `engine/scoring/{offense,defense,special}.go`, only if L2 exceeds 400 lines (checked before Session 12).

---

*Built by: Christopher Campbell + Claude Opus 4.8 (Anthropic). Audit basis: `session-3-audit-build-sequencing.md`.*
