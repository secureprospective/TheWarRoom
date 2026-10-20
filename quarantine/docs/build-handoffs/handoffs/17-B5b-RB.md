HANDOFF — Session 17: B5b-RB (resume OFFENSE — the running back rubric)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B5b-DT (row 15, squash `c12bd98`) is MERGED. The DEFENSE skeleton + the architecture
  stress-test are done: `internal/engine/l4/defense` (DT), the SL-021 cushion guard (L3
  `engine.ApplyCushionGuard` riding `Calibration` + the L4 breakout-trajectory half in
  `DT.Apply`), SL-005 film compression (capBand param), the active-RAS pattern, and the
  M17 extraction of shared curve helpers into `internal/engine/l4/curve`.
- Both skeletons now exist: `engine/l4/offense` (QB) and `engine/l4/defense` (DT), both
  built on the shared `curve` package (Scurve / Interp / SubSignal / Present / Breakpoint /
  NeutralNorm). The "version the boundary" pattern is established: add a sub-signal only
  when a rubric needs it (ScoutingInput + composition + harness), don't forward-design.
- Case status: 3C (QB+K) and 3F (DT cushion) GREEN; 3A/3B/3D/3E/3G/3H/3I/3J/3K PENDING on
  their later positions; 3L runs. Suite 0 FAIL.

== WHY RB IS NEXT (sequencing) ==
AD-15 ordered QB-first / DT-second deliberately (offense skeleton, then defense stress-test).
With both proven, the rest are mass-production on the two skeletons. Build_Tracker row 16 is
B5b-RB. RB is a STANDARD offense rubric (Film × RAS × Breakout, no SL-020/SL-021 exotica) —
the cleanest re-use of the QB offense skeleton — and it flips case 3B (the Herbert pattern,
"L4 pulls below 1.00 for weak-profile vets"). CONFIRM the sequence with Christopher before
building; if priorities changed (e.g. a defensive position first), re-confirm at the gate.

== WHAT B5b-RB BUILDS ==
- A new RB rubric in the EXISTING `internal/engine/l4/offense` package (RB/WR/TE live with QB;
  it is NOT a new package). `func NewRB() *RB` + `Apply(engine.Layer4Input) engine.Layer4Output`,
  mirroring `offense/qb.go`. Use the shared `curve` helpers — do NOT duplicate them.
- RB mechanics from RB_Rubric.md (READ IT — triage every number vs the literal spec):
  - RAS tier / cap / steepness / position-weight schedule (is RB RAS active like DT, or a
    different tier? verify — do not assume).
  - Film component config + whether any RB film source is populated (likely Data-Parity
    neutral like QB/DT — weights UNSET pending the film redesign; confirm in the source map).
  - Breakout sub-signal weights + the four normalization curves (breakout age / school tier /
    college share / age trajectory) + RB peak limit (composition.peakLimit(PosRB) = 25 today —
    confirm vs spec).
- RB-UNIQUE sub-signal: TouchShare (`scouting.Profile.TouchShare *float64`, RB ONLY — snap-count
  workload share, Option D). If the RB breakout/film uses it, that is a NEW sub-signal → grow
  the boundary the localized way: `engine.ScoutingInput` (+ `Has*` flag) + `composition`
  (PlayerSpec field, validate, map in `scouting()`) + `harness` fixtures. "Version the boundary"
  — add it ONLY because RB needs it.
- Register RB in `harness_app.rubrics()` AND `harness/realrubric_test.go` `realRegistry()`.

== THE HARNESS IS YOUR ACCEPTANCE GATE ==
- Case 3B (Herbert pattern) gates on B5b-RB alone — it should FLIP to PASS/FAIL this session
  (the QB+DT pattern: convert the `gatedPending("3B", …)` entry into a real `eval3B` once the
  rubric + any needed sub-signal inputs exist). NONE may be made green by weakening an
  assertion — match the spec.
- Add planted-failure tests (M3) for any new mechanic; if TouchShare is added, prove ABSENT ≠
  a real zero (the QB B1 / DT regression — `Has*` + neutral-0.50 substitution in the rubric).
- Two contrasting RB fixtures so Module 1 visibly separates them (already have RB Alpha/Bravo
  in fixtures.go — extend them with the breakout/touch-share fields RB reads).

== READ FIRST ==
- docs/scoring-engine/RB_Rubric.md — authoritative RB mechanics. Triage every number.
- internal/engine/l4/offense/qb.go + internal/engine/l4/curve/curve.go — the skeleton + shared
  helpers to mirror (scurve takes capBand as a PARAM — pass RB's cap).
- internal/engine/l4/defense/dt.go — the active-RAS + Data-Parity-neutral-film pattern, if RB
  RAS is active.
- internal/engine/types.go (ScoutingInput / Layer4Input / Layer4Output / Calibration),
  internal/composition/{playerspec.go,composition.go,defaults.go}, internal/harness/{cases.go,
  fixtures.go,realrubric_test.go} — the ripple surface (mirror the B5b-DT diff exactly).
- internal/scouting/types.go — TouchShare (RB-only) + the OffenseFilm group.
- docs/data-layer/Offense_Scouting_Source_Map.md — whether any RB film source is live.

== GATE CHECK (confirm with Christopher before writing code) ==
1. Sequence: RB next (recommended — standard offense rubric, flips 3B) vs a different position.
2. TouchShare: does the RB rubric consume it this session, or is it deferred? (Drives whether
   the boundary grows.)
3. Film: Data-Parity neutral (likely, weights unset) vs a live RB film source — confirm in the
   source map.
4. Acceptance: 3B flips this session; the harness is the gate; no assertion weakened.

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch session/b5b-rb. Never git --no-verify.
- CT105: export PATH=$PATH:/usr/local/go/bin; go build ./...; GOMEMLIMIT=1500MiB GOGC=20 make
  lint (repo ROOT); go test -race ./... . Beelink GUI: wails dev -tags webkit2_41.
- Files < 400 lines (AD-17). Engine stays PURE — offense pkg imports only engine + curve;
  depguard engine-is-pure must stay green (PROVE with a planted import).
- Confidence scores INTERNAL — never on Layer4Output/Result (FilmRaw is debug-only, allowed).
- Lint gotcha: never name a param/var `cap` (revive redefines-builtin) — use `capBand`.
- Every custom gate proven by a planted failure (M3); shared logic reused, not copied (M17).
- Review gate: GLM 5.2 (OpenCode on bird), BLIND, output is LEADS — triage every finding vs
  source. INLINE source into the prompt (scp self-contained prompt; GLM hangs on repo
  exploration) AND inline the composition/playerspec boundary files too (B5b-DT review flagged
  3 "unverified wiring" leads purely because they were not inlined). bird: ssh -i /root/.ssh/bird
  x@192.168.1.195. GLM track record: B5b-DT 3 real / B5b-QB 2 real / B5a 2 real — earns its keep.

== CARRIED FORWARD — leads, not blockers ==
- The scouting `[0,1]`-normalized schema convention vs the "engine normalizes from raw"
  contract still needs reconciliation when REAL scouting.Profile data is wired into
  composition (today fixtures supply raw facts directly). Flag at the data-wiring session.
- DT RAS receding position-weight schedule (0.50 after 1 NFL season / 0.10 year 2+) ships only
  the rookie 1.00 weight — wire the schedule when an NFL-data-stage input lands (a consumer
  needs it first). Same pattern likely applies to RB RAS.
- IDP/offense film weights UNSET (calibration pass) — film stays Data-Parity neutral until the
  film-source redesign lands.
- K stays an identity PLACEHOLDER in rubrics() until B5b-K (DECISION-011, row 23).

== CLOSE GATE FOR THIS SESSION ==
- go build + make lint 0 + go test -race green; engine depguard green (planted import).
- Case 3B GREEN (or a real FAIL caught + fixed); no assertion weakened.
- GLM 5.2 BLIND review (sliced, inlined prompts incl. the boundary files); triage vs source.
- Functional gate: Christopher operates the harness — sees 3B flip, sees RBs differentiate.
  Squash-merge after he confirms. Then write handoff 18 (B5b-WR — confirm sequencing).
