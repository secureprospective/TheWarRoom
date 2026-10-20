HANDOFF — Session 15: B5b-QB (the first REAL Layer 4 — the offense skeleton-setter)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- THE ENGINE SPINE (B5a, `6a47ef6`) + THE TESTING HARNESS (row 13, `0f421ed`) ARE MERGED.
  The engine is pure; L4 is a pluggable `engine.Layer4` interface with an IDENTITY default
  (Combined 1.0). The harness is the acceptance gate: it has the composition boundary
  (`internal/composition`), Module 1 rookie rankings, and the Module 3 three-state runner.
- B5b-QB is Build_Tracker row 14 — the FIRST real per-position Layer-4 rubric, and the
  SKELETON-SETTER for all offense rubrics (RB/WR/TE follow its shape). Branch fresh off main:
  git checkout main && git pull && git checkout -b session/b5b-qb. Confirm scope first.

== WHAT B5b-QB BUILDS ==
- A real `engine.Layer4` implementation for QB in a NEW pure package — Build_Tracker says
  `engine/l4/offense`. It computes Layer4Output = film_effective × RAS_effective ×
  breakout_effective (Backend_Architecture:256). SL-020: `scoreRAS` is FORCED to exactly
  1.000 for QB (RAS has zero Layer-4 influence at QB). Film + Breakout are active.
- It MUST stay pure: `engine/l4/**` is under the engine's depguard `engine-is-pure` rule —
  no store/db/IO; all inputs arrive as parameters. Confirm the depguard rule covers the new
  sub-package (it globs `internal/engine/**`; verify l4 lives under it or add the rule).

== THE LOAD-BEARING FACT — THE CONTRACT NOW GROWS (this is the session's real work) ==
- The harness was built on the "VERSION THE BOUNDARY" decision: `engine.Layer4Input` today
  carries ONLY `Player PlayerInput`. The real QB rubric needs SUB-SIGNALS that are not on it
  yet — film composite + breakout-age/school-tier/college-usage inputs (the OffenseFilm /
  breakout fields from `internal/scouting`). B5b-QB is where those fields get ADDED to
  `Layer4Input`, and the change must ripple in exactly THREE places (the boundary localizes it):
    1. `engine.Layer4Input` — add the sub-signal fields the QB rubric reads.
    2. `internal/composition` — `PlayerSpec` gains the same sub-signals; `Assemble` fills the
       new `Layer4Input` fields (still fail-loud; still the only place inputs are assembled).
    3. `internal/harness` — `SampleRookies` fixtures + the (future) manual form fill them;
       `eval3A`/etc. can now feed real inputs.
  The pure engine contract change is small and localized BY DESIGN — that was the point of
  putting the seam in composition.

== THE HARNESS IS YOUR ACCEPTANCE GATE (use it, don't reinvent) ==
- Register the QB rubric in `harness.RubricRegistry` (today `harness_app.go` `rubrics()`
  returns `{}`; add `domain.PosQB: offense.NewQB(...)`). The moment it registers:
    • Module 3 case 3C (SL-020) AUTO-EVALUATES — it already asserts QB RASEffective==1.0000 at
      RAS 0.10 and 9.99 (the wired exemplar in `internal/harness/cases.go`, `eval3C`). A correct
      rubric flips 3C PENDING→PASS; a wrong one → FAIL. This is your first real green.
    • Case 3A (Lockett, QB portion) needs the new Layer4Input sub-signals — once you add them,
      replace its `gatedPending` body with a real assertion against the spec's expected ranges.
  Module 1 rankings will start DIFFERENTIATING QBs (L4 no longer 1.0) — visible in the UI.
- The three-state model is the contract: PENDING is never FAIL. As you implement, cases move
  PENDING→PASS. Do NOT make a case PASS by weakening its assertion; match the spec ranges.

== READ FIRST ==
- docs/scoring-engine/Engine_Specification.md — Layer 4 offense mechanics: the film S-curve,
  RAS curve (SL-020 forces QB to 1.000), breakout component, component caps (±5% offense).
- docs/scoring-engine/Scouting_Schema.md + internal/scouting — the OffenseFilm / breakout field
  set the sub-signals come from.
- internal/engine/types.go — Layer4Input/Layer4Output/Layer4; pipeline.go — how Score calls
  Apply. internal/engine/layer4.go — the identity impl to mirror.
- internal/composition/{playerspec.go,composition.go} — where the new sub-signals get filled.
- internal/harness/cases.go (eval3C exemplar), fixtures.go, validation.go (RubricRegistry).
- DECISION-011 (K is Madden-driven) is B5b-K, not here — but note QB sets the offense skeleton.

== RECON (Haiku fan-out — run before design/build) ==
Spin a Haiku Explore subagent for, VERBATIM: the QB Layer-4 mechanics in Engine_Specification
(film S-curve formula + caps, the SL-020 RAS rule, the breakout component math); the exact
OffenseFilm + breakout field names in internal/scouting; the current engine.Layer4Input shape
and every call site that constructs it; and how the identity Layer4 + NewPipeline wire today.
Claude VERIFIES load-bearing claims against source before code — recon never overrides source
(the SL-020/film-cap numbers especially: triage against the literal spec line).

== GATE CHECK (confirm with Christopher before writing code) ==
1. Package location: `engine/l4/offense` (a sub-package under the pure engine) vs a flat
   `internal/engine/offense` — confirm it falls under depguard `engine-is-pure`.
2. Layer4Input growth: confirm the sub-signal field set to add (film composite + the breakout
   inputs) and that it ripples only through composition + harness (the versioned-boundary plan).
3. Scope: QB ONLY this session (skeleton-setter). RB/WR/TE reuse the shape in later rows.
4. Acceptance: 3C must go green; 3A-QB gets a real assertion. Agree the harness is the gate.

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch session/b5b-qb. Never git --no-verify.
- CT105 build: export PATH=$PATH:/usr/local/go/bin; go build ./... ; GOMEMLIMIT=1500MiB GOGC=20
  make lint (repo ROOT); go test -race ./... . wails build needs webkit2gtk — on the Beelink
  use `wails dev -tags webkit2_41` (4.1, not 4.0). Files < 400 lines (AD-17); pre-split.
- Engine stays PURE — the new l4 package imports NO store/db/IO; depguard must stay green.
- Confidence scores are INTERNAL: they may inform the rubric but never appear on Layer4Output
  or engine.Result (Hard Constraint). The harness may show them as sandbox-only debug.
- Every custom gate proven by a planted failure (M3). Shared logic extracted (M17).
- Review gate: GLM 5.2 (OpenCode on bird), BLIND, output is LEADS — triage every finding vs
  source. INFRA: GLM/opencode HANGS on repo exploration — INLINE the source files into the
  prompt (scp a self-contained prompt to bird, `opencode run --agent review "$(cat ...)"`);
  review in focused slices. bird: ssh -i /root/.ssh/bird x@192.168.1.195 ; clone ~/qa/repos/TheWarRoom.

== CARRIED FROM THE HARNESS — leads, not blockers ==
- Module 1 v1 uses a SANDBOX league cap ("1000" in harness_app.go sandboxCap); the cap-tier %s
  and decay are the real admin-tunable B4 values. Wiring the real rulebook cap (needs a loaded
  MFL league) is deferred — not a B5b-QB concern unless you want real cap amounts.
- Module 3 cases 3D–3K remain encoded PENDING; each turns real as its B5b block (DT/LB/CB/S/...)
  lands. Several assert rubric INTERNALS (film_raw, PFF alpha, NGS weight) not on Layer4Output —
  B5b adds those test hooks WITH its rubric.
- SampleRookies fixtures are synthetic ("QB Alpha"); replace with rubric-verified cases as you
  build real assertions (versioned fixtures, not inline literals).

== CLOSE GATE FOR THIS SESSION ==
- go build ./... + make lint 0 + go test -race ./... green; engine depguard still green
  (new l4 package is pure; harness imports it, never the reverse).
- 3C is GREEN in the harness; 3A-QB has a real assertion; Module 1 differentiates QBs.
- GLM 5.2 BLIND review (sliced, inlined prompts); triage every finding vs source.
- Functional gate: Christopher OPERATES the harness — registers the QB rubric, sees 3C flip to
  PASS, sees the QB rankings move vs the scouting baseline. Squash-merge after he confirms.
- Then write the NEXT handoff (16 — B5b-DT or the next offense rubric; confirm sequencing).
