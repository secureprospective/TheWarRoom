HANDOFF — Session 45: Scouting Data Integration — wire fetchers → Profile → engine Layer 4
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

Origin: the "deferred scouting scaffolding" ruling (SLIM_MAP §4). Christopher elevated it
from "keep + document" to "wire it up" (2026-07-20). This is the real B2b-Fetch MODULE
CLOSE — the ~2000 lines of fetchers + scouting.Profile finally connect to the engine.
**PLANNING-FIRST session** (enter plan mode): there are hard gates + a sequencing decision
+ a calibration decision to settle BEFORE code. Do not dive straight into wiring.

== EXECUTION MODEL (locked 2026-07-20, Christopher) ==
INVERTED from the usual review gate. GLM 5.2 does the HEAVY-LIFTING CODE — the scouting
wiring is loadable in 100–200k-token chunks at cheap price, so GLM WRITES the assembly
(fetchers → crosswalk join → normalize → Profile → PlayerSpec) per chunk. **Claude is
HEAD-BRAIN, not the typist:** read/review GLM's output against source, own all judgment —
sequencing, layer/depguard correctness, the film-weight calibration decision, gate sign-off,
harness lockstep — and triage GLM's work leads-not-findings (GLM runs without the full context;
Claude sees the code). Feed GLM self-contained chunk briefs (one signal / one assembly seam at
a time, per the S-phases below). Do NOT let GLM own a decision, a gate, or a locked-decision
reversal. First-instance TEMPLATE (RAS) especially: Claude verifies the shape before every other
signal clones it.

== WHERE WE ARE ==
- Just completed: SLIM_MAP tiers 1–3 on session/slim-cleanup (pushed). Scouting sub-system
  documented as deferred scaffolding (internal/scouting package doc + SYSTEM_MAP.md).
- Working tree: assume clean off main after slim-cleanup merges (confirm at start).
- This session's branch: session/scouting-integration-plan (planning) → then per-phase branches.

== THE GAP (verified against source 2026-07-20 — read this first) ==
- Fetchers EXIST + are unit-tested (Layer 1), each keyed by its SOURCE id (nflverse gsis,
  PFR, espn, CFBD): agetrajectory, collegeshare, collegedefense, crosswalk, kicking, madden,
  nflproduction, pfrcoverage, ras, touchshare, veteranfilm.
- `crosswalk` (internal/ingestion/crosswalk) builds the JOIN: MFL PlayerID → gsis_id (unique
  by design; gsis→MFL is one-to-many) + an espn_id → gsis bridge for CFBD. This is the
  foundation leaf.
- `scouting.Profile` (internal/scouting, leaf keyed by MFL PlayerID) is the UNIFIED target
  shape. **It is constructed NOWHERE in the codebase** (only referenced in fetcher comments).
- `rankings.Runner.scorePlayer` (internal/rankings/rankings.go ~L218-231) builds PlayerSpec
  with base/age/salary/veteran and leaves EVERY scouting Has* false → the rubrics run the
  Data-Parity neutral path. Comment L228-229 says so explicitly.
- **So the wiring = build the missing middle:** an assembly layer that runs the fetchers,
  joins each source signal to the MFL id via crosswalk, normalizes each to its [0,1] scale,
  and produces map[mflID]scouting.Profile; inject a scouting Directory into rankings.Runner
  (a new dependency, mirroring `dir Directory`); populate the PlayerSpec scouting fields from
  the Profile in scorePlayer.

== READ FIRST ==
- SLIM_MAP.md §4 (the ruling) + internal/scouting package doc (WIRING STATUS block)
- internal/rankings/rankings.go (scorePlayer — the injection point) + internal/composition/{ports,defaults,playerspec}.go
- internal/scouting/types.go (Profile + the per-type SOURCE-DRIFT notes — the retained-field rationale)
- internal/ingestion/crosswalk/fetcher.go (the id join) + one signal fetcher end to end (ras/fetcher.go)
- docs/data-layer/{Offense,Defense}_Scouting_Source_Map.md (source status, Option D, eliminated sources)
- handoffs/06-B2b-Fetch-Offense.md · 09-B2b-Fetch-module-close.md (what the fetchers were built to do)
- internal/harness/cases_eval.go (asserts "film Data-Parity neutral" — WILL change when film flows)

== RECON (Explore fan-out — run before design) ==
- Ask for: for EACH of the 11 fetchers — its Fetch() signature, its Raw* output type, its
  source id key, and which scouting.Profile field(s) it feeds (per the source maps). Return a
  table: fetcher → source id → normalize rule → Profile slot → PlayerSpec field.
- Also: which external sources need credentials/are blocked (CFBD key, Madden current-season),
  and which PlayerSpec scouting fields already have a live engine path + harness coverage.
- Verify load-bearing claims (file:line) before they move the plan.

== GATE CHECK (settle in plan mode BEFORE any wiring code) ==
1. SEQUENCING (recommend, get Christopher's ok): this is B5b-sized — split it. Proposed:
   - **S-Phase 0 (template, RECOMMENDED FIRST): RAS end-to-end.** Simplest signal — single
     value, one fetcher (ras), crosswalk pfr→gsis→MFL, PlayerSpec.RAS/HasRAS already exist,
     the engine RAS modulator is already built AND harness-tested. Wire fetch→crosswalk→
     Profile.RAS→PlayerSpec→engine→M1 board for RAS ONLY. This is the first-instance TEMPLATE
     (every other signal clones the assembly shape) + it proves the whole pipe on real data.
   - **S-Phase 1..n:** remaining signals in groups (breakout/college; offense film; IDP film;
     coverage; K film) — each clones the template.
   - **S-Phase C (calibration):** see gate 3.
2. SOURCE GATES: CFBD needs a key (collegefootballdata.com/key — HARD GATE from B2b-Fetch;
   401 without) → gates collegeshare/collegedefense only, NOT RAS. Madden current-season EA API
   is BLOCKED (historical only) → K/IDP film fallback. Durability is ENGINEERING (cache/fallback),
   NEVER a weight (locked B2b principle). Confirm which sources are reachable this phase.
3. CALIBRATION DECISION (durable → expert-panel gate): the film-component weights are UNSET BY
   DESIGN (source maps: "calibration against data, NOT a blind number"). Wiring the DATA is
   separate from setting the WEIGHTS. **Do NOT ship blind film weights.** RAS/breakout/college
   have existing engine treatment; FILM (offense + IDP) is the one whose impact is
   uncalibrated — flowing that data changes scores only once weights are set, which is its own
   decision-gated calibration pass. Keep film Data-Parity neutral until then.

== WHAT S-PHASE 0 BUILDS (the template) ==
- A scouting assembly layer (new pkg — likely internal/normalize scope or internal/scouting-
  adjacent; RESPECT depguard: fetchers are Layer 1 no-upward, engine stays pure, Profile stays
  a leaf). It: runs the fetchers, joins via crosswalk to MFL id, normalizes RAS to [0,1],
  emits map[mflID]scouting.Profile (RAS populated, rest absent).
- A scouting Directory port + injection into rankings.Runner (New() gains a dependency; nil-guard
  it like the others — a silent unwired scouting is the failure mode to prevent).
- scorePlayer populates spec.RAS/HasRAS from the Profile.
- Harness: update the RAS-relevant cases if the neutral-RAS assertions move (keep 3G/3H in mind —
  still pending).

== CONSTRAINTS ACTIVE ==
- Layer law (depguard, build errors): ingestion never imports up; engine imports domain only;
  scouting.Profile imports only playerid. The assembly layer is the ONLY new seam — place it so
  no boundary breaks (mirror how internal/normalize relates to ingestion + domain).
- Standards: 400-line file cap; parameterized SQL; no globals; typed IPC; error-wrap; no
  interface{} at exported boundaries.
- ZERO-LEAK invariant (scouting package doc): no scouting field may hold fantasy points /
  projected volume / MFL scoring config. Every signal is film/athletic/college.
- id reconciliation: crosswalk drops ambiguous ids (documented policy); OQ-013 (created→official
  id ramp) is the related open item — do not silently mis-join.

== CARRIED FROM LAST SESSION ==
- Decisions: scouting scaffolding KEPT + documented (2026-07-20), now being wired. M2/M4 refactor
  handoffs (43/44) staged in parallel — independent of this.
- Mistakes/learnings: live-verify every external-API boundary EARLY (B3 lesson: league feed ⊇
  global feed caught only by running the live pipeline; units were green). CT105 CAN reach MFL/
  nflverse; CFBD needs the key.
- Open items carried: OQ-013 (id ramp), OQ-014 (Money type), harness 3G/3H pending, film weights UNSET.

== CLOSE GATE FOR S-PHASE 0 ==
- Build green: make lint 0 + go test -race ./... green (incl. new assembly pkg tests).
- First-instance TEMPLATE review (GLM 5.2 blind, leads-not-findings, triage vs source) — this is
  the shape every other signal inherits, so it gets the template gate.
- Functional check (Beelink live gate): run the M1 board live; a player with a real RAS shows a
  non-neutral RAS modulation vs a player with absent RAS (exactly 1.000) — RAS actually moved a
  score, sourced from the live fetch, joined by crosswalk. Reset Wails clone before/after.
- Handoff: write S-Phase 1's handoff (first signal group) before clearing.
