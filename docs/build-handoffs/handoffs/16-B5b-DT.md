HANDOFF — Session 16: B5b-DT (the DEFENSE skeleton-setter — the architecture stress-test)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B5b-QB (row 14, squash `7048881`) is MERGED. The OFFENSE skeleton is set:
  `internal/engine/l4/offense` (QB rubric), the pure `engine.Layer4` contract, and the
  "version the boundary" pattern — `engine.ScoutingInput` (with per-sub-signal `Has*`
  presence flags), `Layer4Input.Scouting`, `Score(p, sc, c)`, and the composition+harness
  ripple. Shared S-curve / interp / Data-Parity helpers live in `offense/curves.go`.
- Case 3C (SL-020) is GREEN. The harness registry (`harness_app.go` `rubrics()`) holds QB
  (real) + K (identity PLACEHOLDER). Module 1 differentiates QBs by breakout.
- B5b-DT is Build_Tracker row 15 — DT is deliberately SECOND (AD-15: QB first, DT second)
  to STRESS-TEST the architecture on a defensive rubric before mass-producing the rest.
  It is the DEFENSE skeleton-setter (LB/DE/CB/S reuse its shape). Branch fresh off main:
  git checkout main && git pull && git checkout -b session/b5b-dt. Confirm scope first.

== WHY DT IS THE STRESS-TEST (this is the session's real architectural work) ==
DT exercises three mechanics QB did not, and at least one of them may touch the ENGINE
SPINE, not just a new L4 package. Resolve these at the GATE before coding:

1. **SL-021 Cushion Guard — an L3 DECAY modulator, not a pure L4 multiplier.** At DT,
   RAS ≥ 8.00 → ~10% deceleration of the age-decay pull (it protects elite-athlete DTs
   from aging as fast). That modifies `ApplyDecay`'s output as a function of RAS — i.e. it
   lives at the L3 seam in the engine, which today takes only (age, peak, rate). The pure
   engine must stay pure; the cleanest path is a per-position L3 modulator passed via
   `Calibration` (or a new pluggable seam mirroring Layer4), NOT a store reach. **Case 3F
   gates on this.** Decide: does the cushion guard ride on `Calibration` (a RAS-threshold +
   strength the rubric ships) consumed by `ApplyDecay`, or a separate L3 dispatch? Either
   way the engine stays depguard-pure. This is the load-bearing decision of the session.

2. **SL-005 film compression — ±3% at DT/LB vs ±5% elsewhere.** The film component cap is
   position-specific (offense used ±5%). So `capBand` must become a rubric constant per
   position (it already is a scurve PARAMETER — just pass 0.03 for DT). **Case 3D gates on
   this AND needs `film_raw` (pre-effective) exposed** — Layer4Output does not carry it yet;
   3D's encoded detail says "film_raw (pre-effective) not on Layer4Output". Adding a debug
   `FilmRaw` to Layer4Output (sandbox-only, never UI per Hard Constraint) is the likely move.
   3D gates on DT+LB+WR, so it will not fully flip this session — but build the hook.

3. **Dynamic PFF alpha (0.50 Y1 → 0.10 Y2+) — a film-blend INTERNAL.** This is an EMA blend
   rate inside the film sub-signal composite. **Case 3G gates on it** and its encoded detail
   says "PFF blend alpha is a rubric internal, no hook yet" — so 3G needs a test hook
   exposing the alpha (or the blended intermediate). Decide where that hook lives (a
   rubric-introspection method, NOT on Layer4Output which is the production surface).

== WHAT B5b-DT BUILDS ==
- A new pure package `internal/engine/l4/defense` (Build_Tracker row 15) implementing
  `engine.Layer4` for DT. It is IDP: it reads `scouting.IDPFilm` (Madden defense
  sub-attrs + NFLProduction + pfrcoverage — the redesigned IDP film, weights UNSET pending
  calibration, so expect Data-Parity neutral like QB film today) and the breakout sub-signals.
  **The NGS Coverage anchor is CB/S ONLY — it MUST NOT appear at DT** (Hard Constraint).
- **SL-019 is EXCLUDED at DT** (Cushion Guard replaces it — running both double-protects;
  Hard Constraint). DE (row 19) is where SL-019 first applies.
- **Escape hatch allowed (AD-14):** DT may diverge from the offense skeleton where the IDP
  mechanics require it — but document every divergence and keep the engine pure.
- Contract growth, same localized pattern: any new sub-signal fields → `engine.ScoutingInput`
  + composition + harness; any new L3 modulator → `engine.Calibration` + `ApplyDecay`.

== THE HARNESS IS YOUR ACCEPTANCE GATE ==
- Register DT in `rubrics()` (`harness_app.go`): `domain.PosDT: defense.NewDT(...)`.
- Cases that begin evaluating once DT registers: 3F (cushion guard — needs the L3 modulator
  built), 3G (PFF alpha — needs DE too + the hook), 3D (SL-005 — needs LB+WR too + FilmRaw).
  NONE may be made green by weakening its assertion — match the spec. Several will stay
  PENDING until their co-gated positions land; that is correct (three-state model).
- Add planted-failure tests for every new gate (M3): the cushion guard must be SEEN to
  change the decay output at RAS 8.00 vs 7.99; SL-005 ±3% must be SEEN to bind tighter than
  ±5%.

== READ FIRST ==
- docs/scoring-engine/DT_Rubric.md — the authoritative DT mechanics: SL-021 hybrid RAS tier,
  Cushion Guard (RAS threshold + decel strength), dynamic PFF alpha schedule, SL-005 film
  compression, the IDP film composite, breakout. Triage the literal numbers vs the code.
- docs/scoring-engine/Engine_Specification.md — L3 decay + where a per-position modulator
  attaches; L4 IDP mechanics; component caps.
- docs/scoring-engine/Scouting_Schema.md + internal/scouting — IDPFilm field set (Madden
  defense sub-attrs / NFLProduction / pfrcoverage; weights UNSET — Data-Parity neutral).
- internal/engine/{types.go,pipeline.go,decay.go} — Layer4Input/Output, Score, ApplyDecay
  (the L3 seam the cushion guard touches). internal/engine/l4/offense/{curves.go,qb.go} —
  the skeleton + shared helpers to MIRROR (scurve takes capBand as a param already → pass
  0.03 for SL-005). internal/composition + internal/harness/{cases.go,fixtures.go,
  validation.go} — the ripple + the DT-gated cases (3D/3F/3G).
- docs/roadmap/Roadmap_and_Open_Questions.md — SL-OQ-037, SL-OQ-039 (AD-22, carried by DT).
- DECISION-011 (K Madden-driven) is B5b-K, not here.

== RECON (Haiku fan-out — run before design/build) ==
Spin a Haiku Explore subagent for, VERBATIM: the SL-021 Cushion Guard mechanics in
DT_Rubric + Engine_Specification (the RAS threshold, the decel strength, and EXACTLY which
layer it modifies — L3 decay vs L4); the SL-005 film-compression cap value for DT/LB; the
dynamic PFF alpha schedule; the IDP film sub-signal field names in internal/scouting
(IDPFilm); the current engine.Calibration + ApplyDecay shape and signature; and how the QB
rubric + the offense curves helpers are structured (to mirror). Claude VERIFIES the
load-bearing numbers (cushion RAS threshold + decel %, SL-005 ±3%) against the literal spec
line before any code — recon never overrides source.

== GATE CHECK (confirm with Christopher before writing code) ==
1. Cushion Guard home: per-position L3 modulator on `Calibration` consumed by `ApplyDecay`
   (recommended — keeps the engine pure, one seam) vs a separate L3 dispatch. THE decision.
2. FilmRaw exposure: add a sandbox-only `FilmRaw` to Layer4Output for case 3D? (never UI.)
3. PFF-alpha hook: where the rubric exposes its blend alpha for case 3G (introspection
   method, not the production Layer4Output).
4. Scope: DT ONLY (defense skeleton-setter). LB/DE/CB/S reuse the shape later. Escape hatch
   (AD-14) used only where IDP mechanics force it — documented.
5. Acceptance: 3F goes green (cushion guard built + planted-tested); 3D/3G get their hooks
   but stay PENDING on their co-gated positions. Agree the harness is the gate.

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch session/b5b-dt. Never git --no-verify.
- CT105 build: export PATH=$PATH:/usr/local/go/bin; go build ./... ; GOMEMLIMIT=1500MiB
  GOGC=20 make lint (repo ROOT); go test -race ./... . wails on the Beelink uses
  `wails dev -tags webkit2_41`. Files < 400 lines (AD-17); pre-split.
- Engine stays PURE — `engine/l4/defense` imports NO store/db/IO; depguard `engine-is-pure`
  must stay green (PROVE it with a planted import, as B5b-QB did). The cushion guard touches
  L3 but via PARAMETERS only.
- NGS Coverage anchor at CB/S ONLY — must not bleed to DT (Hard Constraint).
- SL-019 NOT applied at DT — Cushion Guard replaces it (Hard Constraint).
- Confidence scores INTERNAL — never on Layer4Output/Result; sandbox debug only.
- Every custom gate proven by a planted failure (M3). Shared logic extracted (M17).
- Review gate: GLM 5.2 (OpenCode on bird), BLIND, output is LEADS — triage every finding vs
  source. INFRA: GLM/opencode HANGS on repo exploration — INLINE source files into the
  prompt (scp a self-contained prompt to bird, `opencode run --agent review "$(cat ...)"`);
  review in focused slices. bird: ssh -i /root/.ssh/bird x@192.168.1.195.

== CARRIED FORWARD — leads, not blockers ==
- The scouting `[0,1]`-normalized schema convention (scouting/types.go) vs the "engine
  normalizes from raw" contract (engine takes raw, normalizes per-position) needs explicit
  reconciliation when REAL scouting.Profile data is wired into composition (today the
  harness PlayerSpec supplies raw facts directly). Not a DT blocker; flag at the data-wiring
  session.
- IDP film weights are UNSET (calibration pass) — DT film will be Data-Parity neutral like
  QB film today. Module 1 will differentiate DTs by breakout + cushion guard, not film yet.
- K stays an identity PLACEHOLDER in rubrics() until B5b-K (DECISION-011, row 23).
- Admin stepper allows sub-zero entry (UI polish) — the B4 range gate correctly REJECTS it
  (proven live), so it is a cosmetic follow-up, not a correctness issue.

== CLOSE GATE FOR THIS SESSION ==
- go build + make lint 0 + go test -race green; engine depguard still green (new defense
  pkg pure, PROVEN by planted import; the cushion guard rides Calibration, not a store).
- Case 3F GREEN (cushion guard built + planted failure proven); 3D/3G hooks in place.
- GLM 5.2 BLIND review (sliced, inlined prompts); triage every finding vs source.
- Functional gate: Christopher OPERATES the harness — registers DT, sees 3F flip, sees a
  high-RAS DT decay slower than an equivalent low-RAS DT. Squash-merge after he confirms.
- Then write handoff 17 (B5b-RB — resume offense; confirm sequencing).
