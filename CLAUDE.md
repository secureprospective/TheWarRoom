# TheWarRoom — Project CLAUDE.md
**Version:** 1.5 — June 2026
**Project path:** `/mnt/storage/claudebox/projects/TheWarRoom/`
**Pillars:** Business, Technical

---

## What This Project Is

A 32-team dynasty fantasy football ranking engine and full-stack desktop application for the Legacy NFL league. Six-layer scoring engine (Go), Wails v2 desktop shell, React + Tailwind + Zustand frontend, SQLite WAL storage, MFL API integration.

**Three-phase delivery:** Personal tool (Phase 1, current) → League-wide alpha (Phase 2) → Public beta (Phase 3).

---

## Current Build State (June 2026)

**B2b-Schema — Scouting Schema COMPLETE + merged (2026-06-19, `4da4b04`). `internal/scouting` leaf locks the unified scouting field set for all 10 positions (`Profile` + `OffenseFilm`/`IDPFilm`/`NGSCoverage` pointer groups + `SchoolTier`/`SafetyRole` enums); position boundaries are STRUCTURAL (Coverage non-nil at CB/S only); zero-leak verified field-by-field; imports only playerid. AD-16 human-review gate passed (per-position walk). SL-OQ-035/036 reserved via `SafetyRole` (S monolithic v1.0). DECISION-011 recorded: K Layer-4 is Madden-driven (Madden 0.60 / NFLProduction 0.40) — reverses K structural-exclusion + AD-10 + SL-020-at-K; mechanics rewrite carried to B5b-K. Also resolved OQ-012 this session (League Schedule Methodology, `179176e`). Next: B2b-Fetch-Offense (scouting fetchers, QB/RB/WR/TE) — real gate there is SOURCE ACCESS (most scouting sources aren't MFL endpoints).**

**Standing build pattern (2026-06-19, all remaining sessions):** open each module session with a **Haiku Explore recon fan-out** over the handoff's READ FIRST docs (cheap recon/fetch tier — gathers an inventory, never owns design/gates/locked-decision reversals); verify its load-bearing claims against source before acting. Codified in `docs/build-handoffs/Handoff_Protocol.md` (Recon Phase + `== RECON ==` template field).

**B2b-Schema close (2026-06-19):** Haiku recon assembled the 10-position scouting-input inventory; Claude verified the load-bearing claims (SL-OQ-035/036 verbatim, NGS=CB/S-only) against source; Claude + Christopher did the design + AD-16 walk. K's Madden-majority direction (DECISION-011) absorbed by the schema with no structural change — `MaddenFilm` was already a core field; only K's characterization was corrected. make lint 0 / go test -race green. Design doc: `docs/scoring-engine/Scouting_Schema.md`.

**B3 — MFL Data Normalization COMPLETE (2026-06-18). Raw→typed domain records; the internal type system is LOCKED (AD-18 reviewed deliverable). OQ-004 (EDGE→DE) + OQ-005 (salaryAdjustments dead-cap) RESOLVED. Live-verified CT105→MFL end-to-end.**

**B3 close (2026-06-18):** `internal/domain` (leaf: `PlayerRecord`/`Roster` + `Position`/`ContractStatus`/`RosterStatus` enums + `PosFlag` admin sentinel) and `internal/normalize` (pure Layer-1: `NewLookup` cross-ref join, `classifyPosition` PK→K/EDGE→DE/XX→FLAG, dirty-`contractStatus` table, salary str→float, RISK-003 site #2 via `playerid.New`, fail-loud per record, deterministic intra-roster id sort). Plus TWO B2-addendum Layer-1 fetchers the join needed: **`players` — now LEAGUE-scoped** (the global `api` feed OMITS commissioner-created players: live ids 0816 Gosnell/0820 Roberts/0835 Childress/0838 Robinson, present in the 2621-record league feed, absent from the 2578 global) and **`salaryadjustments`** (the per-franchise dead-cap ledger resolving OQ-005). **TWO Gemini reviews** (agy still out): normalize (4/5 findings applied incl. determinism sort + reserved-range hardening; the suggested `ToUpper` REJECTED — would break MFL's case-specific aggregate codes Def/Coach/Off/ST) and the salaryAdjustments money path (**BLOCKER**: MFL returns HTTP 200 with `{"error":{"$t":…}}`, which — with this fetcher's deliberate no-empty-sentinel policy — would silently wipe every team's dead cap on an outage; fixed with shared `ingestion.CheckAPIError`). **Live PASS (CT105→MFL): 1217 records / 32 franchises (zero loss), reserved-range invariant clean on the 2621-player DB, salaryAdjustments franchise 0025 = $5.495 ≈ displayed $5.49.** Christopher's calls: reserved-range anomaly **flags+continues** (not halt); negative salary adjustments are **valid** (commissioner credits). New: **OQ-013** (created→official id reconciliation ramp — refresh/sync layer) + **OQ-014** (Money-type / cap-math precision — `Salary float64` kept for B3, decision deferred to cap-math). `salaryadjustments` aggregation into cap usage is downstream engine work (deferred).

**B2 close (2026-06-18):** `internal/ingestion` formalized — shared boundary helpers (`ValidatePlayerID` RISK-003 site #1, `IsTeamAggregateID` 0151–0782 filter, `MFLList[T]` array/object-collapse decoder) + two fetchers: `rosters` (template-setter, league-specific, DiscoverHost-first) and `schedule` (non-league `api`-host path, built against the live-captured `nflSchedule` shape). Gemini 3.1 Pro first-instance review of the rosters template caught TWO real BLOCKERs the linters could not: MFL collapses single-element arrays to bare objects (would crash a 1-player franchise) and an empty `{"rosters":{}}` payload decoded to a silent empty slice (downstream DB-wipe risk) — both triaged against source and fixed (the `MFLList` decoder is now shared so every fetcher inherits it; zero-length guarded with a sentinel). Christopher's two template-contract calls: **fail-loud per record** (don't add partial-success machinery until a dirty fetcher needs it) and **`Fetch` takes year/leagueID as args, not package globals** (data-driven, multi-league/season-ready). Live PASS on the Beelink: rosters 32 franchises / 1217 records (payload-size cross-check confirms the aggregate filter isn't eating real players); schedule 16 matchups, exactly one home team each. **Contracts/salaries fetcher DEFERRED (OQ-005). Aggregate-filter airtight verification (no real player in 151–782) carried to B3's players-join.**

**B1 close (2026-06-18):** `internal/mfl` formalized to WF 1A + 3 T4 fixes; the transport template (`New`/`Do`/`DiscoverHost`, rate-limit + 429 backoff + host routing) is locked — fetchers inherit it, never re-implement it. TWO Gemini 3.1 Pro first-instance reviews (client, then the live test) each caught real BLOCKERs the linters could not: a `NaN` rate slipping past `rps<=0` into the limiter (→ `Wait` blocks forever), and retries bypassing the limiter (→ concurrent storm). Both triaged against source (M13) and fixed. Live smoke test (`TWR_LIVE_MFL=1`, opt-in) PASSED on the Beelink: discovered host `www47`, `Do(rosters)` → 200 with a 140KB real rosters payload. Squash-merged to main. **Q2.1 deferred (Christopher blessed):** a league-specific call made before discovery falls back to the `api` host rather than hard-erroring — "discover first" is a B2 ingestion responsibility, not a transport constraint.

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

**Session 8 completions (2026-06-13, Pre-Build Friction Testing):**
- ~~G0 Go overlay~~ — **AUTHORED.** `.golangci.yml` (v2), `.pre-commit-config.yaml` (SHA-pinned), `Makefile.snippet`, `schema/example.go`, `README.md` added to `christopher-coding-standards` on `session/go-overlay-g0` (commit `86bcde6`, NOT YET MERGED). Found and fixed a SEVERE depguard glob bug (`files:` patterns need a leading `**/`, or the three-layer-law check fires on **zero files**).
- ~~T1/T2/T3 friction tests~~ — **COMPLETE.** Full log at `Fable_Friction_Log.md` (13 numbered frictions + final synthesis). Measured (not estimated) confidence: plan-fidelity 78%, realized-enforcement 72%, collab-plumbing 58%, end-goal ~60%.
- ~~B1 transport client (`internal/mfl`)~~ — agy (CT104) built it end-to-end from a self-contained brief, scored 10/10 on T2's lint-based conformance rubric. agy's own First-Instance Template Review (T3) then found 2 real LOGIC bugs the lint pass couldn't see (host-discovery couldn't recover from a stale cached host; a caller-supplied `JSON` param could override the mandatory `JSON=1`) — **both fixed and verified**, commit `d573420` on `session/prebuild-friction-testing`.
- **NEW BLOCKER FOUND:** Friction #12 — CT105's stored GitHub PAT can read but not push to `secureprospective/TheWarRoom` (403). Both session branches (`session/prebuild-friction-testing` @ `d573420`, `session/go-overlay-g0` @ `86bcde6`) are committed locally but **not on origin**.

**Session 9 completions (2026-06-13, Confidence-80 pass — all four metrics ≥80%):**
- ~~Gate 1 — AD-06 enforcement~~ — **LOCKED: struct-wrap.** `playerid.PlayerID` is now `struct { id string }` (unexported field); the bypass `playerid.PlayerID("99")` fails to **compile**. Template `templates/go/playerid/example.go` in coding-standards (`session/go-overlay-g0` @ `b77709b`).
- ~~Gate 2 — `interface{}`/`any` escapes~~ — **LOCKED: custom `ifaceguard` vettool** (`tools/ifaceguard/`), pinned + `analysistest`-guarded, wired into `make lint` + pre-commit. Closes the gap no golangci-lint linter covered.
- ~~Plan-fidelity sweep~~ — **third unenforced-claim found + closed:** the 400-line file cap had no gate (funlen caps functions, not files); added a `filelen` Makefile gate. Gate 5 WF1A contradiction reconciled; Gates 1/2 recorded LOCKED in companion plan Section 10.
- ~~Collab workflow (Section 9)~~ — **REBUILT + VALIDATED.** 9.8 Triage Protocol (new), 9.2 git/SSH-relay dual-channel, 9.6 memory-symmetry corrected. T1 re-run (7/7 gates fire), T4 (agy re-review, **0 hallucinations**, triage exercised), and **T3-as-designed PASSED** (agy's standing clone pulls the pushed branch from git — primary channel live).
- ~~Friction #12 (PAT)~~ — **FIXED.** New fine-grained PAT (TheWarRoom + coding-standards, Contents R/W). All branches pushed to origin.
- **STRATEGIC PIVOT (Christopher, 2026-06-13):** the **local-AI idea is cut.** CT106/AiderBox **retired** (hardware upgrade needed, no fix), Hermes/Ollama brain cut, weak-local-model conformance path **descoped** (not deferred). Collaboration refocuses on **Claude (CT105) + agy/CT104** only — agy is a capable agentic CLI, not a local model, so it is unaffected. See `[[strategic-pivot-claude-agy-only]]` in auto-memory.

**Forward source of truth:** `docs/build-handoffs/Build_Tracker.md` — the 38-session checkable sequence. Read at the start and end of every build session. Every session closes by writing the next session's handoff per `docs/build-handoffs/Handoff_Protocol.md`.

**Immediate next steps (in order) — pre-B0 confidence pass COMPLETE (all 4 metrics ≥80%):**
1. ~~Merge the G0 overlay~~ — **DONE 2026-06-13.** `session/go-overlay-g0` squash-merged to coding-standards `main` (PR #8, `cf45454`, all 6 CI checks green); TheWarRoom `session/confidence-to-80` merged to `main` (`bbe7cf0`). **B0 is unblocked.**
2. Resolve OQ-004 (EDGE mapping) and OQ-005 (salary adjustment) before B3.
3. Build and validate the Layer 4 testing harness per `docs/build-handoffs/Testing_App_Specification.md` (Session 13).
4. First code build: B0 — Project Scaffold (Session 1). **B0 carries forward:** copy the G0 overlay (`.golangci.yml`, `playerid/`, `tools/ifaceguard/`, Makefile, pre-commit) into the repo; commit a pinned toolchain for agy/Claude parity (9.6); apply the B1 fixes from T4 (stale `client.go:36` comment, `time.NewTimer`+`Stop`, `rps>0` guard).
- **DONE this session:** PAT/Friction #12 (FIXED, branches pushed) · AD-06 + interface{} enforcement (LOCKED) · Section 9 collab workflow (REBUILT + VALIDATED, both channels live).

**Next branch:** The next session is **B2b-Fetch-Offense** (`session/b2b-fetch-offense`, branch from main) — handoff ready at `docs/build-handoffs/handoffs/06-B2b-Fetch-Offense.md`. It builds the scouting fetchers that populate `scouting.Profile` for QB/RB/WR/TE. **Open the session with a Haiku recon fan-out** (standing pattern). **Real gate: SOURCE ACCESS** — most scouting sources (PFF, RSP, Sharp, Madden, RAS) are NOT MFL endpoints; settle each source's ingestion path (manual import / scrape / API) with Christopher before building a fetcher. Do not invent a data source. (B2b-Schema + OQ-012 merged to main 2026-06-19; branches deleted.) **Collaboration model: Claude + agy; until agy returns the review gate is Gemini 3.1 Pro (found a real money BLOCKER in B3 — but triage every finding against source; it suggested a `ToUpper` that would have broken MFL's case-specific aggregate codes).** CT105 CAN reach MFL (HTTP 200) — the B3 live pipeline ran there directly; the Beelink remains the Wails/GUI dev machine (Go 1.26.4 + Wails 2.12 + pnpm + WebKit 4.1; `-tags webkit2_41`).

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
4. Build command: `make lint` (ifaceguard + filelen + golangci-lint, all must pass) then `make test` (`go test -race ./...`). Go 1.26.4 is at `/usr/local/go/bin` (NOT on default `$PATH` — Friction #1; prepend it). **On CT105 (2GB RAM), `golangci-lint run ./...` OOM-kills unless you warm the build cache first (`go build ./...`) and cap memory — run lint as `GOMEMLIMIT=1500MiB GOGC=20 golangci-lint run ./...` (or via `make lint` with those env vars set). `.golangci.yml` already pins `concurrency: 1`.** `wails build` produces the desktop binary (needs the GUI libs; headless CT105 compiles it but cannot display it).

---

## Functional Verification on the Beelink (READ before asking Christopher to run anything)

CT105 is headless and its firewall may block outbound — GUI runs and live network tests happen on the **Beelink (192.168.1.190)**. To avoid the B1 round-trips, follow this every time:

1. **PUSH the session branch to origin from CT105 FIRST.** The Beelink pulls from origin; it cannot see an unpushed local branch. (B1 cost a round-trip because the branch wasn't pushed.)
2. The Beelink clone is **`/home/chris/TheWarRoom`** (the CT105 `/mnt/storage/...` path does NOT exist there; `/home/chris/.config/TheWarRoom` is just the SQLite data dir). Go is `/usr/local/go/bin/go`.
3. The Beelink may be on a **stale branch** — the paste.md batch must `git fetch origin` before `git checkout <branch>`.
4. Per [[feedback_paste_md_copypaste]]: any command Christopher runs goes in `/root/paste.md`, labeled with why + where (target machine, repo path) + what PASS looks like.

Live/network tests are **opt-in and env-gated** (e.g. `TWR_LIVE_MFL=1`) so they never fire in the default suite/CI; they still compile + lint with everything else (no build tag). Copy `internal/mfl/client_live_test.go` as the pattern.

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

## Open Items at Last Session Close

| Item | Status |
|---|---|
| B2 — MFL Data Ingestion | **COMPLETE 2026-06-18** — rosters (template) + schedule fetchers + ingestion boundary helpers (`MFLList`, `ValidatePlayerID`, `IsTeamAggregateID`). Gemini review: 2 BLOCKERs fixed. Live PASS (rosters 32/1217, schedule 16). **Merged to main `2866bd4`** |
| B1 — MFL API Client | **COMPLETE 2026-06-18** — transport template locked + live-tested (www47, rosters 200). Merged to main. 2 Gemini reviews, 2 real BLOCKERs fixed |
| Aggregate-filter airtight check (B2 review #1) | **CARRIED to B3** — confirm zero real offensive/defensive-position players carry an ID in [151, 782] at the players-join. Confirmed safe by spec + payload-size cross-check at B2; needs position data to be airtight |
| Players-endpoint fetcher | **NOT built** — B2 was rosters + schedule. B3's cross-reference join (and OQ-004 EDGE) needs it; build in B3 or as a B2-addendum (non-league `api` host, cache once/day) |
| OQ-012 League (fantasy) schedule source & schema | **RESOLVED (methodology) 2026-06-19** — Christopher supplied the deterministic NFL-mirrored schedule methodology; full spec at `docs/data-layer/League_Schedule_Methodology.md` (13-wk regular season + wks 14–17 playoffs, home/away + inaugural power-rank seeding). Generator build is a separate slot. Merged `179176e` |
| B2b-Schema — Scouting Schema | **COMPLETE + merged 2026-06-19 (`4da4b04`)** — `internal/scouting` leaf, all 10 positions, AD-16 walk passed, zero-leak verified, lint 0 / race green. SL-OQ-035/036 reserved via `SafetyRole`. Doc: `docs/scoring-engine/Scouting_Schema.md` |
| DECISION-011 — K Layer-4 Madden-driven (0.60/0.40) | **RECORDED 2026-06-19** — reverses K structural-exclusion + AD-10 + SL-020-at-K. Schema impact none (MaddenFilm already core). **Carried to B5b-K:** Film cap/curve calibration + K-rubric/AD-10/SL-020 mechanics rewrite. K rubric flagged on disk |
| OQ-004 EDGE position mapping source | **RESOLVED (B3, 2026-06-18) — EDGE→DE.** MFL labels edge rushers DE; 0 live EDGE records. `normalize` maps directly; no consensus-check machinery |
| OQ-005 Salary adjustment line item | **RESOLVED (B3, 2026-06-18)** — `salaryAdjustments` export (league-scoped) = per-franchise dead-cap ledger; `internal/ingestion/salaryadjustments` built + live-verified (0025=$5.495). Cap usage = Σ(salaries)+Σ(adjustments); negatives valid. Aggregation deferred to cap-math |
| B3 — MFL Data Normalization | **COMPLETE 2026-06-18** — `internal/domain` + `internal/normalize` (type system LOCKED, AD-18). 2 Gemini reviews (1 money BLOCKER fixed). Live PASS 1217/32. On `session/b3-mfl-normalization`, awaiting squash-merge |
| Players-endpoint fetcher | **BUILT (B3, 2026-06-18) — LEAGUE-scoped** (`internal/ingestion/players`). Global feed omitted commissioner-created players; league feed (2621) includes them. All 1217 rostered ids resolve |
| OQ-013 Created→official id reconciliation ramp | OPEN — auto-replace commissioner-created id with MFL's official id on refresh; deferred to refresh/sync layer, manual swap is the fallback. Noted 2026-06-18 |
| OQ-014 Money type / cap-math precision | OPEN — `domain.PlayerRecord.Salary` is `float64` (safe at scale; cap math must avoid exact `==`). Money-type decision deferred to cap-math layer. Noted 2026-06-18 |
| Q2.1 league-call-before-discovery | **SATISFIED at B2** — `rosters.Fetch` calls `DiscoverHost` first (ingestion owns "discover before any league-specific call"); schedule is a non-league `api` call, no discovery |
| OQ-006 Cap tier calibration | OPEN — resolve after live data |
| OQ-007 Scouting layer weight | OPEN — resolve after testing |
| OQ-008 Franchise tag calculation timing | OPEN |
| OQ-009 RFA eligibility window | OPEN |
| OQ-010 Playoff bid rules trigger | OPEN — needs commissioner confirmation |
| Session 3 audit pass | COMPLETE — Engine_Spec v3.0, Universal_Rubric_Template v1.2 |
| Go overlay (Phase 2 in coding-standards repo) | **MERGED to main** — PR #8 (squash `cf45454`), all 6 CI checks green. AD-06 struct-wrap, interface{} ifaceguard, file-cap filelen; 7/7 + 2 gates verified. **B0 unblocked.** |
| GitHub PAT push access (Friction #12) | **FIXED** — new fine-grained PAT (TheWarRoom + coding-standards, Contents R/W). All branches pushed; T3-as-designed validated the shared-repo channel live |
| AD-06 enforcement mechanism (`playerid.PlayerID` bypass) | **RESOLVED — struct-wrap** (Gate 1). Bypass fails to compile. `templates/go/playerid/example.go` |
| `interface{}`/`any` escape enforcement | **RESOLVED — `ifaceguard` vettool** (Gate 2). `tools/ifaceguard/`, pinned + tested, wired into `make lint` + pre-commit |
| 400-line file cap enforcement | **RESOLVED — `filelen` Makefile gate** (plan-fidelity sweep). funlen caps functions not files; this is the file gate |
| B1 transport client (`internal/mfl`) | T3 fixes pushed (`d573420`). T4 re-review flagged 3 more minor items for the *real* B1 (Session 2): stale `client.go:36` comment, timer leak (`time.NewTimer`+`Stop`), `rps>0` guard |
| Claude↔agy collab workflow (Section 9) | **REBUILT + VALIDATED** — 9.8 Triage Protocol, 9.2 dual-channel (git primary / SSH-relay secondary), 9.6 memory corrected. T3-as-designed + T4 passed (T4: 0 hallucinations) |
| Local-AI idea / CT106 / Hermes | **CUT (2026-06-13)** — local-AI dropped; CT106/AiderBox retired (hardware), weak-model path descoped; collab = Claude + agy only. See `[[strategic-pivot-claude-agy-only]]` |
| Cross-project cleanup (pivot follow-up) | OPEN — coding-standards Layer-11/Phase-3 (local-model guidance) now moot; Local AI Stack project CLAUDE.md needs a wind-down pass |

---

*Built by: Christopher Campbell + Claude (Anthropic)*
