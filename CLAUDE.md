# TheWarRoom — Project CLAUDE.md
**Version:** 2.0 — July 2026
**Project path:** `/mnt/storage/claudebox/projects/TheWarRoom/`
**Pillars:** Business, Technical

---

## What This Project Is

A 32-team dynasty fantasy football ranking engine and full-stack desktop application for the Legacy NFL league. Six-layer scoring engine (Go), Wails v2 desktop shell, React + Tailwind + Zustand frontend, SQLite WAL storage, MFL API integration.

**Three-phase delivery:** Personal tool (Phase 1, current) → League-wide alpha (Phase 2) → Public beta (Phase 3).

---

## Current Build State (2026-07-24, main HEAD `b6b7899`)

**Latest:** Module-3 harness 3H wired + `pfrpassrush` IDP fetcher built (both merged). Only 3G remains PENDING in Module 3. pfrpassrush NOT yet wired into the engine — next step is IDP FILM calibration (Thread C IDP arm), decision-gated with an expert-panel weight gate.

**Alpha Versioning & Releases — ALL 3 TIERS DONE** (T1 build-stamp+tag `v0.5.0`, T2 ledger-safety `facb73c`, T3 self-protection `0b971ca`). Nothing downstream is gated on versioning anymore.

**UI Build Track:** B-1 shell & tokens DONE (`ce710bc`). Next = B-2 module restyle → B-3 calendar → B-4 Home+Inspector → B-5 harden → **ALPHA GATE**.

**Scouting engine:** S-Phase 0–4, 4b, and Thread B (crosswalk consolidation) all MERGED. FILM Thread C offense arm (`OffenseFilm`) already MERGED + LIVE (`4e4a0ef`); the IDP arm + the weight-calibration pass remain (`docs/roadmap/FILM_Calibration_Planning.md`) — no blind weights, expert-panel gate required.

**➡️ NEXT (pick one):** execute FILM Thread C IDP arm · B-2 module restyle · M2/M4 refactors (handoffs 43/44) · league-calendar backend · M2 slice-2.

**Full history:** `docs/build-handoffs/handoffs/41-50-*.md` (chronological) + `docs/build-handoffs/Build_Tracker.md` (checkable sequence) + `docs/roadmap/Roadmap_and_Open_Questions.md`. Memory: [[project_thewarroom]], [[project_thewarroom_ui_roadmap]], [[project_thewarroom_versioning_releases]].

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
| B5a — Engine Pipeline (`internal/engine`) | **DONE + MERGED to main 2026-06-27 (squash `6a47ef6`).** Pure-function spine: L1/L3/L5/L6 + L4 pluggable interface (identity default). `Pipeline.Score` accumulates per Backend_Architecture:259,266. L2 deferred (BasePoints supplied). depguard `engine-is-pure` green (NO store/db/IO). Fail-loud non-finite guards L3+L5 (shared `finite()`). make lint 0 / race green (3 planted gates). GLM 5.2: 2 real fail-loud gaps fixed. **Next = Testing Harness (row 13, HARD GATE)** |
| B5a — L3 modulators (SL-018 buffer, SL-021 cushion guard) not built | **OPEN by design — they consume per-position modulator strengths that ship WITH B5b.** B5a computes the RAW age pull only; modulators layer on in B5b |
| B5a — `Layer4Input` carries only `Player` | **OPEN by design — B5b adds the sub-signal fields the first real `Layer4` reads.** The harness's manual form fills them when that rubric lands |
| B5a — composition boundary not built | **OPEN by design — B5a is pure, inputs are parameters.** The Testing Harness (row 13) builds the FIRST composition boundary: stores + manual/MFL input → `engine.PlayerInput`+`Calibration`. Lives OUTSIDE `internal/engine` |
| B4 — Admin Parameter Store (`internal/store/params`) | **DONE + MERGED to main 2026-06-26 (squash `9f2c675`).** Generic `(key,position)` typed calibration store, shipped Go defaults + admin override, seed-once/no-reseed, two-lock. v1.0 = cap-tier %s (AD-21) + global scalars (Option C). Admin-only write, never B7 (AD-05); `validateOverride` range gate. make lint 0 / race green (8 tests). GLM 5.2: 3 real defects fixed (NaN gate, torn read, updated_at). **Layer-2 config floor COMPLETE → B5a unblocked** |
| B4 — `hasDefaults` row-count-only seed check (GLM L2) | **OPEN — forward-risk, not a present defect** (seedDefaults is transactional, no partial seed possible). Revisit with `n >= len(defaultParams())` or a content hash if a param migration is added |
| B4 — SetOverride reloads both full tables per write (GLM L5) | **OPEN — fine at v1.0 row counts.** Revisit when per-position tables make the table large |
| B4 — `Definitions` sort-test checks Key only, not (Key,Position) (GLM L6) | **OPEN — add a position tiebreaker case once a per-position fixture exists** |
| B4 — per-position calibration tables (peak_limit, S-curve, EMA α, …) | **OPEN by design — ship WITH the B5b engine layer that consumes each, never pre-seed into B4** |
| B3c — League State Store (`internal/store/state`) | **DONE + MERGED to main 2026-06-26 (squash `52c9760`).** 32-team mutable state: rosters + contracts + derived cap. Reader/Writer split (Writer→B7a only, un-assertable Reader); Initialize seeds once from normalize, no Reload; fail-loud (empty-seed, RowsAffected==1, orphan reconcile); franchise identity player-derived v1.0. make lint 0 / race green (14 tests). GLM 5.2: 2 real fail-loud gaps fixed |
| B3c — franchise identity player-derived (GLM MED, Christopher: document) | **OPEN by design** — a franchise emptied of all players is absent from `Franchises()`/returns ok=false. Canonical 32-team list owned by a future franchise registry; revisit when B6/M1 need "always 32." Documented on `Franchises()` |
| B3c — cross-call reader snapshot not isolated (GLM LOW-MED) | **OPEN — add a league-wide snapshot read API if a consumer needs it.** B7 is safe (holds `wmu`); engine/IPC readers composing two reader calls can straddle a `load()` swap |
| B3c — `MovePlayer` accepts phantom target franchise (GLM LOW-MED) | **OPEN — target-franchise validity is B7's responsibility** (B3c holds no franchise registry to validate against). Documented as the writer's job |
| B3b — League Rulebook store + `league` fetcher | **DONE + MERGED to main 2026-06-26 (squash `12afc3c`).** `internal/store/rulebook` (versioned snapshots + active pointer, side-loading `Reload`→`ChangeSet`, `Promote` confirm/rollback, layered overrides + validation gate) + `internal/ingestion/league` (two-endpoint `league`+`rules` fetcher, `$t`/collapse decode). make lint 0 / race green / live PASS. GLM 5.2 review: 2 real concurrency fixes (slice-alias, admin-write race) |
| B3b verification add-on (rule-change detection) | **DONE 2026-06-26** — `Reload` side-loads + returns `ChangeSet` (commissioner-gate signal); `Promote` applies-or-rolls-back; overrides survive a re-pull. Notification POPUP = deferred thin UI (later admin session). New-PLAYER verification routed to OQ-013, NOT the rulebook |
| pfrcoverage aggregate-NA silent drop (GLM Layer-1 lead, HIGH) | **OPEN — verify at calibration live-fetch** (recorded in Open Items above) |
| veteranfilm join-key normalization (GLM Layer-1 lead, MED) | **OPEN — hardening at calibration** (recorded in Open Items above) |
| B2b-Fetch-Kicker — kicking fetcher (K NFLProduction 0.40) | **DONE 2026-06-26 (`887fda4`, branch `session/b2b-fetch-kicker`, not merged)** — `internal/ingestion/kicking`. nflverse `player_stats_kicking_season` (separate from offense file). RAW gsis-keyed counts (FG made/att/missed/blocked, distance buckets, fg_long, PAT); rates unbound; REG-only; structural zero-leak. Live 45 kickers |
| Traded-kicker dedup policy | **RESOLVED 2026-06-26 (Christopher) — SUM per-team REG splits** (counts+buckets add, fg_long=max). No source aggregate row exists (unlike PFR `[0-9]TM`); raw-count aggregation, distinct from pfrcoverage prefer-aggregate. Rejected: drop-traded (loses 7/45), REG+POST (still split + playoff bias). Greg Joseph spot-checked vs raw |
| Madden K (0.60 signal) | **NO new code 2026-06-26** — recon-verified m24 carries `kickPower_rating`/`kickAccuracy_rating`(+awareness) for K, already captured by `madden` fetcher's dynamic `*_rating` parse + name/birthdate resolver |
| SL-OQ-042 (Madden K archival pipeline) | **RESOLVED 2026-06-26** — superseded by DECISION-011 (Madden K is a live 0.60 signal, not archival); whole-roster pull not position-conditional, so "keep K in routine pulls" holds with zero new logic. CAL-032 (predictive utility) stays open (calibration) |
| B2b-Fetch arc module-close (M17 + seam review) | **COMPLETE + MERGED to main 2026-06-26.** Item 1 pfr→gsis bridge promotion (crosswalk `PFRMap()`+`addBridge`; behavior-preserving — touchshare 637/pfrcoverage 775 unchanged, ras 4831 under both old+new maps, pfr bridge 7779). Item 2 shared CFBD helpers (`CFBDStatRow`/`FetchCFBDCategory`/`CFBDInt`/`CFBDFloat`/`Share`/`EmitDropAmbiguous[P,T]` in cfbd.go; collegeshare/collegedefense refactored; unit tests unchanged = equivalence proof). Item 3 Gemini seam re-review ran blind → all 6 findings triaged to FALSE POSITIVE against source (lr.N=maxBytes+1 sentinel, value-type build, gzip cleanup-on-error, PFRMap copy+test, pfrIdx>=0 guard, mergeKicking total) → **zero code changes.** **B2b-Fetch arc CLOSED → next module B3b.** |
| B2b-Fetch-Defense — gzip seam | **DONE 2026-06-26 (`0b6d12b`)** — `extcsv.go` `FetchCSVGz`/`StreamCSVGz` over shared `openCappedCSV`; cap bounds decompressed bytes; non-gzip body fails loud at gunzip. **Flag: agy/Gemini seam re-review at arc close** |
| B2b-Fetch-Defense — pfrcoverage (CB/S Coverage anchor) | **DONE 2026-06-26 (`55d5c28`)** — `internal/ingestion/pfrcoverage`. NGSCoverage REBOUND onto PFR advanced defense (nflverse has no defender NGS file). Passer-rating-allowed headline; traded-player `[0-9]TM`-aggregate dedup; rates `*float64` absent≠0. Live 775/2024 |
| B2b-Fetch-Defense — collegedefense (CollegeProductionShare) | **DONE 2026-06-26 (`0b142c9`)** — `internal/ingestion/collegedefense`. CFBD defensive+interceptions; per-component within-team shares (tackle/sack/TFL/PD/INT) RAW, engine combines per position. Same espn→gsis injected resolver; PPA never fetched. Live 665/2023, shares in [0,1] |
| NGSCoverage rebind decision | **RESOLVED 2026-06-26 (Christopher)** — nflverse publishes no defender NGS; CB/S Coverage anchor sourced from PFR advanced defense. Field name retained. Doc: `docs/data-layer/Defense_Scouting_Source_Map.md` §2 |
| IDP film source access | **RESOLVED 2026-06-26 → ELIMINATED (Option-D parallel, Christopher)** — no clean source for any IDP film sub-signal (IDPShow/IDPGuru/DynastyNerds paywalled; PFF/TDN already gone). Redesigned on Madden defense sub-attrs (already in m24) + NFLProduction + pfrcoverage. NO new fetcher; weights UNSET (calibration). Schema `IDPFilm` annotated. Doc §3 |
| pfr→gsis bridge promotion (M17) | **OPEN (module-close)** — 3rd consumer now (touchshare/ras/pfrcoverage); bridge lives only in live-test `livePfrToGSIS`. Promote `GSISForPFR` into `crosswalk.Map`, refactor the three. Do at B2b-Fetch arc close (next session) |
| Shared CFBD long-format helpers (M17) | **OPEN (module-close)** — collegeshare + collegedefense duplicate statRow/fetchCategory/parsers/poison-emit. Extract into shared `ingestion`. Do at arc close |
| Film reweight calibration (offense + defense) | **OPEN — calibration pass.** Eliminated-source fields (OffenseFilm RSP/Sharp, PFFGrade, DraftNetwork, IDPFilm *) retained-pending-redesign, populated by no fetcher. Durability never weights; quality = fidelity discount. Weights set against live data, not blind |
| pfrcoverage aggregate-NA silent drop (GLM lead, HIGH) | **OPEN — verify at calibration live-fetch.** `pickSeasonRow` commits to the `[0-9]TM` aggregate row; if that row's `tgt` is NA while per-team split rows carry targets, `rowCoverage` returns `hasCov=false` and the traded defender is silently skipped (fetcher.go:181) — splits never consulted. Mechanism real; trigger (PFR aggregate with blank counting cols) unconfirmed/unlikely. CHECK against a live 2024 mid-season-traded CB/S: does the `2TM` row populate every counting column the splits do? If not → fall back to summing splits' counting cols. Cannot trip a test. From 2026-06-26 GLM 5.2 Layer-1 freeze review |
| veteranfilm join-key normalization (GLM lead, MED) | **OPEN — hardening at calibration.** FTN↔PBP join (accumulate.go:67 & :136) keys on RAW `game_id`/`play_id`, no `TrimSpace` either arm. Low risk (both nflverse files share id conventions; live run gave sane 209 recv/45 pass) but a partial format skew = silently-wrong rates over a wrong denominator (only an ALL-miss trips `errEmpty`). Add `TrimSpace` both arms as insurance; while there, a charted-but-unjoined miss-counter would surface skew. From 2026-06-26 GLM 5.2 Layer-1 freeze review |
| B2b-Fetch-Offense — TouchShare (RB) | **DONE 2026-06-21 (`bc842ed`)** — `internal/ingestion/touchshare`, snap_counts→season aggregate, injected pfr→gsis, SL-OQ-021 per-active-game (GamesActive). Live 637. RAW; engine normalizes |
| B2b-Fetch-Offense — AgeTrajectory | **DONE 2026-06-21 (`94e3d44`)** — `internal/ingestion/agetrajectory`, players.csv birth_date (raw DATE, not derived age). Live 24961 |
| B2b-Fetch-Offense — RAS | **DONE 2026-06-21 (`bb15e01`)** — `internal/ingestion/ras`, combine.csv measurables (*float64 absent≠0, ht "F-I"→inches, injected pfr→gsis). RAS-EQUIVALENT (engine z-scores), not Platte's number. Live 4832 |
| RAS dedup policy (combine pfr collisions) | **RESOLVED 2026-06-21 — Christopher's call: drop-ambiguous + continue (silent exclude).** Live data: combine.csv shares one pfr_id across two distinct players; a gsis with 2+ rows is dropped entirely (lenient external boundary; ~17 affected). NOT fail-loud, NOT first-wins |
| IntCell/FloatCell seam extraction | **DONE 2026-06-21 (`eac431f`)** — to `extcsv.go`, nflproduction refactored on (M17). **Flag: module-close agy/Gemini re-review** (seam change) |
| `Profile.TouchShare` comment drift | **OPEN (cosmetic)** — still says "FantasyPros touch share"; it's snap_counts now (Option D). Fix at module close |
| `StreamCSV` streaming seam | **DONE 2026-06-21 (`f4eb542`)** — row-by-row `extcsv.go` sibling to `FetchCSV` for files too big to buffer; byte-cap fail-loud, by-name binding, one reused row in memory. M3 over-cap + callback-abort proven same commit. **Flag: module-close seam re-review** |
| Veteran-Film (FTN→PBP) | **DONE 2026-06-21 (`afc90a4`)** — `internal/ingestion/veteranfilm/`, FTN→pbp join on (game,play), RAW per-player trait RATES (receiver: contested/created/drop; passer: int-worthy/throwaway), structural zero-leak. Multi-year-capable (default `[2025]`); floor-suppression (params 30/100, calibration-tunable). Live 2025: 209 recv / 45 pass, rates in [0,1] |
| SchoolTier (CFBD `/teams`) | **DONE 2026-06-21 (`a19d285`)** — `internal/ingestion/schooltier/`, conference→`scouting.SchoolTier` keyed by school. Imports scouting + emits FINAL value (position-independent; documented exception). Pac-12 year-aware (P4 ≤2023), Notre Dame→P4. CFBD = authed JSON, h2 DISABLED, key needs TrimSpace, lenient decode. Live: 681 schools, P4=68/G5=66/FCS=129/nonFCS=418. **CFBD client local — extract to shared `ingestion/cfbd.go` at 2nd CFBD caller (M17)** |
| CollegeProductionShare (CFBD) | **DONE 2026-06-21 (`6fe195f`)** — `internal/ingestion/collegeshare/`. Rookie keying RESOLVED by recon (not the 3 handoff candidates): CFBD `playerId`=espn_id, db_playerids bridges espn→gsis. Christopher: gsis-keyed + bridge in crosswalk pkg (`GSISForESPN`, espn OPTIONAL, ambiguous-drop per RAS precedent). 2 CFBD calls (all FBS/call), local team-sum, within-team REC/YDS/rush shares via INJECTED resolver. Live 2023: 407 records, Malik Washington 0.444 top. Placeholder gsis for rookies = OQ-013 instance; yardage shares can be tinily negative (raw) |
| Madden (EA ratings-api) | **DONE 2026-06-21 (`1a382ed`)** — `internal/ingestion/madden/`. PREMISE DEGRADED on recon (m26 500 AND m25 empty → only m24, 2 seasons stale); Christopher: build lean vs m24. Paged EA api, every numeric `*_rating`→name→int map (string ratings skipped, non-int number fails loud), INJECTED name+birthdate resolver (~77%, brittleness contained). Live m24: 1404 records, ratings [0,99]. Lower-durability fallback behind FTN; weight = calibration |
| CFBD shared client (M17) | **DONE 2026-06-21 (`0046674`)** — `internal/ingestion/cfbd.go`: `NewCFBDClient` (h2-disabled) + `GetCFBD` (bearer, byte-capped, returns bytes for concrete lenient decode). Extracted from schooltier at 2nd CFBD caller; schooltier refactored + live re-verified. **Flag: module-close seam re-review** |
| crosswalk espn→gsis bridge | **DONE 2026-06-21 (`0046674`)** — `GSISForESPN` added to crosswalk.Map from the same db_playerids fetch. espn OPTIONAL (foundation MFL→gsis must not break); ambiguous espn→2-gsis dropped (RAS precedent, 4 live). Live: 7885 espn→gsis entries |
| scouting/types.go source drift | **FIXED 2026-06-21 (`a79aa3c`)** — TouchShare comment = snap_counts (not FantasyPros); PFFGrade/DraftNetwork/RSP/Sharp marked ELIMINATED-retained-pending-Film-redesign. Comments only; field set rework belongs to the Film redesign |
| B2b-Fetch-Offense — source access | **DECISION LOCKED 2026-06-19 → Option D (no code yet).** Eliminate manual; PFF/RSP/DraftNetwork/Sharp eliminated; veterans 0 manual (FTN film + nflverse/CFBD/EA), rookies 1 manual consensus-rank CSV/yr. All endpoints verified live. Doc: `docs/data-layer/Offense_Scouting_Source_Map.md`. Forces Film-component redesign (separate calibration pass — weights UNSET). Gate: CFBD key + targets check; Madden current blocked (fallback) |
| B2b-Fetch-Offense — CFBD API key | **RESOLVED 2026-06-20** — key minted + live-verified on CT105 (`/teams` → HTTP 200; gate cleared, both CFBD fetchers unblocked). Key in CT105 env `CFBD_API_KEY`, not committed. Targets question ANSWERED: CFBD exposes `[LONG,REC,TD,YDS,YPR]`, no targets → receiver `CollegeProductionShare` = **reception/yardage share**. Response is long-format (one row/stat) — share = Σplayer ÷ Σteam. See `docs/data-layer/Offense_Scouting_Source_Map.md` §4 |
| B2b-Fetch-Offense — rubric film reweight | **OPEN — calibration pass.** Eliminated sources were 100% of QB/WR/TE Film. New film = FTN(primary)+Madden(fallback) veterans / consensus-rank+Madden-rookie+combine+CFBD-college rookies. Durability never weights; quality = fidelity discount via CAL. Weights set against live data, not blind |
| B2 — MFL Data Ingestion | **COMPLETE 2026-06-18** — rosters (template) + schedule fetchers + ingestion boundary helpers (`MFLList`, `ValidatePlayerID`, `IsTeamAggregateID`). Gemini review: 2 BLOCKERs fixed. Live PASS (rosters 32/1217, schedule 16). **Merged to main `2866bd4`** |
| B1 — MFL API Client | **COMPLETE 2026-06-18** — transport template locked + live-tested (www47, rosters 200). Merged to main. 2 Gemini reviews, 2 real BLOCKERs fixed |
| Aggregate-filter airtight check (B2 review #1) | **CLOSED 2026-06-20** — proven against LIVE position data: zero real playable-position players carry an id in [151,782] (the rosters ID-range filter never silently drops a real player). New `internal/normalize` tests: `auditAggregateFilter` pure cross-tab + `TestAuditAggregateFilter_*` (offline, plants a real-player-in-range so the gate is seen to fail — teeth) and `TestLive_AggregateFilterAirtight` (opt-in `TWR_LIVE_MFL=1`, hard-asserts on the real DB). FINDING (not a bug): individual `Coach`/`PN` aggregates carry high sequential ids OUTSIDE the reserved block — caught by normalize's position-code filter (the join backstop), not the range filter; range stays [151,782] (widening would swallow real players sharing that id space). Two filters, two concerns, defended in depth |
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
