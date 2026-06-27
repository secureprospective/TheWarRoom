HANDOFF — Session 14: Testing Harness (the Layer-4 validation sandbox)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- THE ENGINE SPINE IS MERGED. B5a (`internal/engine`, squash `6a47ef6`) ships the
  pure-function pipeline: L1 hygiene, L3 decay, L5 cap scaling, L6 tiebreaker as pure
  functions, L4 a pluggable `Layer4` interface with an IDENTITY default (Combined 1.0).
  `Pipeline.Score(PlayerInput, Calibration) (Result, error)` chains them and accumulates
  ScoutingAdjusted = BasePoints × AgePull × Combined, then × CapMultiplier.
- The three Layer-2 stores are merged (B3b rulebook, B3c state, B4 params).
- THIS SESSION builds row 13, the Testing Harness — a HARD GATE: no B5b rubric
  (row 14+) starts until the harness exists. Branch fresh off main:
  git checkout main && git pull && git checkout -b session/testing-harness.
  Confirm scope with Christopher first.

== THE SPEC ==
- docs/build-handoffs/Testing_App_Specification.md is the authority. It is a TESTING
  SANDBOX, not the production app: prioritize correctness + debuggability over UI polish.
  Every component output must be visible/inspectable; admin params adjustable LIVE.
- Three input sources in priority order: (1) MFL API (rosters/players/contracts),
  (2) hardcoded rubric-verified test cases, (3) manual entry form mirroring each
  position rubric's sub-signal structure. Module 1 is the 2026-rookie L4 ranking test.

== THE LOAD-BEARING SEQUENCING FACT (resolve with Christopher FIRST) ==
- CHICKEN-AND-EGG: the harness validates Layer 4 OUTPUTS, but the real per-position L4
  rubrics are B5b (rows 14+), which come AFTER this gate. So the harness CANNOT validate
  a real rubric on day one — there isn't one yet. The harness ships against B5a's
  pipeline with the IDENTITY L4, plus the hardcoded test-case scaffold, and becomes the
  tool that validates each B5b rubric AS it lands (plug the new Layer4 impl in, compare
  output to consensus). Confirm this framing: the harness's job at build time is the
  HARNESS + the composition wiring + the identity-L4 end-to-end path + the inspection UI;
  real-rubric validation is exercised per B5b block thereafter.

== THE OTHER LOAD-BEARING FACT — THE COMPOSITION BOUNDARY ==
- B5a DELIBERATELY did not build the composition boundary (the layer that reads the
  stores + manual input and fills PlayerInput + Calibration). The engine is pure; inputs
  are parameters. THE HARNESS IS THE FIRST CONSUMER THAT MUST BUILD THAT BOUNDARY:
  MFL pull / hardcoded case / manual form  →  PlayerInput + Calibration  →
  Pipeline.Score  →  display Result + every component output.
  - Calibration globals come from B4 params (GetCapTiers, GetGlobal); per-position
    values that B4 does NOT ship arrive from the harness's manual/hardcoded input until
    B5b's per-position tables exist.
  - This boundary lives OUTSIDE internal/engine (depguard forbids the engine importing
    stores). Decide where: likely an app/composition package or the Wails app layer.
    Confirm the package location with Christopher at the gate.

== READ FIRST ==
- docs/build-handoffs/Testing_App_Specification.md — full module + UI + data spec.
- internal/engine/types.go — PlayerInput, Calibration, Result, Layer4 interface,
  Layer4Output. The harness fills the inputs and renders the outputs.
- internal/engine/pipeline.go — Pipeline.Score is the single entry point.
- internal/store/params/ — GetCapTiers / GetGlobal / Definitions for the live admin panel
  (Definitions backs "how much input is still on placeholder defaults").
- internal/store/{rulebook,state}/ — the cap amount (rulebook) + rosters/contracts (state)
  the composition boundary reads.
- internal/mfl + internal/ingestion/* — the MFL pull path for rosters/players.
- Wails app entry (cmd/ or root main.go + app.go) — how IPC methods are wired today.

== RECON (Haiku fan-out — run before design/build) ==
Spin a Haiku Explore subagent for, VERBATIM: the harness spec's module list + each
module's required inputs/outputs (Testing_App_Specification.md); the current Wails app
wiring (where IPC-exposed methods live, how the React front end calls Go); the existing
MFL pull entry points (function signatures to fetch rosters/players); and how the React/
Zustand/Tailwind front end is structured today (is there an existing component scaffold
or is this the first UI surface?). Claude VERIFIES load-bearing claims against source
before code — a handoff/recon claim never overrides live source.

== GATE CHECK (confirm with Christopher before writing code) ==
1. Sequencing framing (the chicken-and-egg above): harness ships against identity L4 +
   scaffold now; real-rubric validation is per-B5b. Agree?
2. Composition-boundary package location + that it (not the engine) reads the stores.
3. Harness scope for v1: which modules from the spec are in THIS session vs deferred
   (Module 1 rookie rankings is the natural first; the spec lists more).
4. UI fidelity bar: spec says debuggability over polish — confirm "every component
   output inspectable + live-adjustable admin params" is the bar, not visual design.

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch session/testing-harness. Never git --no-verify.
- CT105 build: warm cache `go build ./...`, then GOMEMLIMIT=1500MiB GOGC=20 make lint
  (from repo ROOT), then go test -race ./... . Go at /usr/local/go/bin (NOT on PATH).
  Wails has its own build/dev path — confirm `wails build` / `wails dev` works on CT105
  (headless: the dev server may need a display or a build-only verification path).
- AD-17 file-cap: every file < 400 lines; pre-split. The engine stays UNTOUCHED and PURE
  (depguard must stay green — the harness imports the engine, never the reverse).
- Confidence scores: the spec's engine-config shows film/RAS/breakout confidence as
  INTERNAL debug values. They may be shown in the HARNESS's debug view (it is a sandbox),
  but they remain OFF the production output structs — do not leak them into engine.Result.
- Every custom gate proven by a planted failure (M3). Shared logic extracted (M17).
- Review gate: GLM 5.2 (Z.ai Coding Plan, OpenCode on bird), BLIND, output is LEADS —
  TRIAGE every finding vs source. NOTE: GLM/opencode HUNG TWICE on B5a when asked to
  explore the repo; the reliable path was INLINING the source files into the prompt
  (scp a self-contained prompt to bird, `opencode run --agent review "$(cat ...)"`).
  For a larger UI surface, review in focused slices (composition boundary; one module at
  a time) rather than one giant exploration. bird clone: ~/qa/repos/TheWarRoom; sync with
  `git fetch origin <branch>:<branch>`. Reach bird: `ssh -i /root/.ssh/bird x@192.168.1.195`.

== CARRIED FROM B5a — forward-risk leads, not harness blockers ==
- The SL-018 RAS buffer + SL-021 cushion guard are L3 MODULATORS not yet built (they
  consume per-position modulator strengths that ship with B5b). B5a computes the RAW age
  pull only. The harness's L3 output is raw until B5b layers the modulators in.
- Layer4Input currently carries only Player; B5b adds the sub-signal fields it reads. The
  harness's manual form will need to fill those when the first real Layer4 lands.
- engine.Result intentionally exposes intermediates (AgePull, ScoutingAdjusted,
  CapMultiplier, Layer4Output components) for exactly this inspection use.

== OPEN ITEMS CARRIED (older, not harness-blocking) ==
- B3c franchise identity is player-derived (v1.0); a future registry owns "always 32".
- B3c cross-call reader snapshots aren't isolated across two reader calls.
- B4: hasDefaults is row-count-only; SetOverride reloads both tables per write; per-
  position param tables land WITH their B5b layer.
- OQ-013/OQ-014 (player-id reconciliation; money float64 vs a Money type vs B4 tiers).
- pfrcoverage aggregate-NA silent drop + veteranfilm join-key — verify at calibration.

== CLOSE GATE FOR THIS SESSION ==
- Build: go build ./... + make lint 0 + go test -race ./... green; engine depguard
  still green (harness imports engine, not vice versa).
- The harness runs end to end: a player (hardcoded case or manual entry) flows through
  the composition boundary into Pipeline.Score and the UI shows the final AdjustedScore
  PLUS every component output, with admin params adjustable live.
- Confidence values, if shown, are sandbox-only and absent from engine.Result.
- GLM 5.2 BLIND review (sliced, inlined prompts); triage every finding vs source.
- Squash-merge to main after Christopher confirms (functional gate: Christopher OPERATES
  the harness — sees component outputs, adjusts a param, sees the score move). Then write
  the NEXT handoff (15 — B5b-QB, the first real Layer 4). Confirm sequencing at close.
