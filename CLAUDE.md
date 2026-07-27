# TheWarRoom — Project CLAUDE.md
**Version:** 2.0 — July 2026
**Project path:** `/mnt/storage/claudebox/projects/TheWarRoom/`
**Pillars:** Business, Technical

---

## What This Project Is

A 32-team dynasty fantasy football ranking engine and full-stack desktop application for the Legacy NFL league. Six-layer scoring engine (Go), Wails v2 desktop shell, React + Tailwind + Zustand frontend, SQLite WAL storage, MFL API integration.

**Three-phase delivery:** Personal tool (Phase 1, current) → League-wide alpha (Phase 2) → Public beta (Phase 3).

---

## Current Build State (2026-07-27, main HEAD)

**Alpha Gate passed and the full UI build track (B-1→B-5) is done — TheWarRoom is genuinely live for Christopher's real 32-team league, not just merged-and-tested.** Coding now runs on the GLM Build Workflow (below). Session 0 (taxi/IR cap-math fix) merged `5800a05` — the last Claude-authored session before the workflow switch.

**Session 1 (Activity/Transaction Feed + OQ-013 reconciliation) — PUSHED, gates green, NOT YET MERGED.** First session run end-to-end on the GLM Build Workflow — a genuine test of full Beelink autonomy (GLM implements, DeepSeek blind-reviews, GLM triages/fixes). Branch `session/session-1-activity-transaction-feed`, `origin` HEAD `998eedf`. Built a read-only chronological feed (`internal/store/state/feed.go`) — a single parameterized UNION ALL across the five existing append-only ledger tables (trade_notes/player_status_events/dead_cap_ledger/cap_relief_ledger/contract_year_changes), no schema change, no writes. DeepSeek's blind review found 1 real MAJOR (a silent `ELSE 'RELEASE'` default that would have mis-labeled any future/unknown player status) + 3 real MINOR (dead code, a lexicographic-vs-numeric ID tiebreaker bug, an SQL/Go case-sensitivity mismatch) — zero false positives, all four fixed with new pinning tests. OQ-013's reconciliation seam lives in `transactions_feed_app.go`'s `GetFeed`: a stale commissioner-created id renders as `PlayerUnknown=true` rather than hard-failing, and the same Lookup call will pick up the real MFL id once a future write-side auto-match lands — no rendering change needed then. Frontend: new `FeedBoard.tsx` as a third summon-strip facet (alongside comms/calendar), reusing the Session-D event grammar on backward-looking data. **Independently re-verified on CT105** (not taken on GLM's or DeepSeek's word): `make lint` 0 issues, full `go test -race ./...` green, frontend build clean. **Process note:** GLM's z.ai quota ran out mid-triage-retry on a PATH environment bug (the pre-commit hook's subshell couldn't find `go`) — Claude finished the commit+push manually once quota was gone; the actual fix work and its self-verification (lint/test/build all green) were already complete at that point. `orchestrate.sh` should export the full PATH before Stage 3 to avoid this next time. **Next: Christopher runs the live/functional gate on the Beelink binary before this merges** — do not merge on gate-green alone.

**Full history archived, not lost:** every prior session's detailed build log (Alpha Gate, both roadmap-plan stages, all five UI build sessions B-1→B-5, Module 3, the scouting engine build, and the full historical "Open Items" table) lives in `docs/build-handoffs/Build_State_Archive_Through_Alpha.md`. This section had grown to ~25k tokens of permanent per-session history — big enough that a plain file read hit the harness's token cap mid-session. Genuinely open items (OQ-006 through OQ-016, film reweight calibration, the two flagged GLM leads on pfrcoverage/veteranfilm) live in `docs/roadmap/Roadmap_and_Open_Questions.md` — that's the authoritative open-items list going forward, not a table in this file.

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
4. Build command: `make lint` (ifaceguard + filelen + golangci-lint, all must pass) then `make test` (`go test -race ./...`). Go 1.26.4 is at `/usr/local/go/bin` (NOT on default `$PATH` — Friction #1; prepend it). **On CT105 (2GB RAM), `golangci-lint run ./...` OOM-kills unless you warm the build cache first (`go build ./...`) and cap memory — run lint as `GOMEMLIMIT=1500MiB GOGC=20 golangci-lint run ./...` (or via `make lint` with those env vars set). `.golangci.yml` already pins `concurrency: 1`.** `wails build` produces the desktop binary (needs the GUI libs; headless CT105 compiles it but cannot display it).

---

## Functional Verification on the Beelink (READ before asking Christopher to run anything)

CT105 is headless and its firewall may block outbound — GUI runs and live network tests happen on the **Beelink (192.168.1.190)**. To avoid the B1 round-trips, follow this every time:

1. **PUSH the session branch to origin from CT105 FIRST.** The Beelink pulls from origin; it cannot see an unpushed local branch. (B1 cost a round-trip because the branch wasn't pushed.)
2. The Beelink clone is **`/home/chris/opencode/TheWarRoom`** (verified 2026-07-04; the CT105 `/mnt/storage/...` path does NOT exist there; `/home/chris/.config/TheWarRoom` is just the SQLite data dir). Go is `/usr/local/go/bin/go`.
3. The Beelink may be on a **stale branch** — the paste.md batch must `git fetch origin` before `git checkout <branch>`.
4. Per [[feedback_paste_md_copypaste]]: any command Christopher runs goes in `/root/paste.md`, labeled with why + where (target machine, repo path) + what PASS looks like.

Live/network tests are **opt-in and env-gated** (e.g. `TWR_LIVE_MFL=1`) so they never fire in the default suite/CI; they still compile + lint with everything else (no build tag). Copy `internal/mfl/client_live_test.go` as the pattern.

---

## GLM Build Workflow (locked 2026-07-27)

**Coding for TheWarRoom now happens on GLM 5.2, not Claude.** Christopher's call, mirroring what's already working on Shadowbane: Claude = head brain (brief + triage/diagnosis), Beelink/GLM = workhorse (actual code-writing execution). Unlike Shadowbane, **there is no Hermes interview step here** — TheWarRoom's specs are already locked (rubrics, hard constraints, the 10-session Commissioner Suite plan), so there's no design gap for a human-interview relay to fill.

**Per-session loop:**
1. Claude drafts a self-contained brief for the session's scope — pulled from the locked build-sequence plan (`/root/.claude/plans/commissioner-suite-build-sequence.md`) + relevant docs + Hard Constraints (below) + the Agent Codex (`docs/agent-codex.md`) — naming the session branch.
2. Dispatched to the Beelink as a scripted, detached `opencode run --agent build --model zai/glm-5.2` call with a sentinel file (never piped via stdin — GLM is finicky over this path; see [[reference_glm_review_over_ssh]] / [[reference_opencode_script_prompt]]). GLM implements, runs `make lint` + `make test` itself, commits, and **pushes the session branch to origin directly** — git push credentials now live on the Beelink clone for exactly this.
3. Claude pulls the branch to CT105 and **independently re-runs lint/test** — GLM's self-report is a lead, not a fact ([[lesson_worker_status_reports_unverified]]) — then reviews the diff against Hard Constraints.
4. **Review gate: DeepSeek, not GLM.** GLM authored the code here, so GLM reviewing its own output would forfeit the independence the review gate exists for. This is a per-project carve-out from the standing GLM-is-the-reviewer default ([[feedback_glm_code_reviewer]]) — locked 2026-07-27, Christopher's call, applies to TheWarRoom specifically.
5. Live/functional gate is unchanged — Christopher runs the actual binary on the Beelink and confirms real behavior.
6. Merge to main only after Christopher confirms the live result, per standing branch discipline.

**One-time setup (2026-07-27):** the Beelink clone (`~/opencode/TheWarRoom`) had **no pre-commit/pre-push hooks installed at all** — the `make verify` safety net that's supposed to gate every push had nothing to run on. Fixed: pre-commit + pinned golangci-lint v2.12.2 + gitleaks v8.30.1 installed, `make setup` wires both hook stages, git push credentials cached via `credential.helper store` (Christopher entered the PAT directly on the Beelink — Claude never saw it). This workflow starts at **Session 1** of the Commissioner Suite sequence — Session 0 (taxi/IR fix) was still Claude-authored.

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
- **Run `make setup` once per clone.** Direct-to-main has no PR/CI backstop, so the pre-push `verify` hook (`make verify` — lint + `go test -race` + frontend build, `.pre-commit-config.yaml`) is the only gate between a local commit and origin. `make setup` wires both the commit-stage hooks (golangci-lint, gitleaks, ifaceguard) and the pre-push hook in one step — running only the plain `pre-commit install` leaves the pre-push gate uninstalled with no visible warning.

---

## Key Document Map

| Need | Document |
|---|---|
| **Build doctrine — LOAD BEFORE any build/review (both agents)** | `docs/agent-codex.md` — 17 motifs w/ receipts, slop catalog, security baseline, canon→motif map (canonical in coding-standards) |
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
| Code companion plan + Claude↔agy collab workflow (Section 9) | `Fable_TheWarRoom_code_plan.md` |
| Pre-build friction test results (T1-T3, 13 frictions, synthesis) | `Fable_Friction_Log.md` |
| Next session brief — path to 80%+ confidence on all 4 metrics | `Fable_Confidence_80_Brief.md` |

---

## Open Items

The full "Open Items at Last Session Close" table (resolved-item history back to B0) is archived in `docs/build-handoffs/Build_State_Archive_Through_Alpha.md`. **Genuinely open items live in `docs/roadmap/Roadmap_and_Open_Questions.md`** (OQ-006 through OQ-016, film reweight calibration, the two flagged GLM leads on pfrcoverage/veteranfilm) — check there, not here.

---

*Built by: Christopher Campbell + Claude (Anthropic)*
