HANDOFF — Session 20: B5b-DE (edge-rusher rubric — the FIRST SL-019 REUSE)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B5b-TE (row 18, squash `6d7a1b3`) is MERGED. FIVE real rubrics now live: QB (SL-020-forced
  RAS), DT (SL-021 cushion), RB (Medium-tier RAS), WR (High-tier RAS), TE (High-tier RAS at
  steepness 11.0 + the FIRST SL-019 RAS-modulator). All on the shared `internal/engine/l4/curve`
  helpers, including the new **`curve.SL019(base, rasNorm, strength, hasRAS)`** — the generalized
  modulator TE introduced and DE now REUSES verbatim (M17 paid off).
- "Version the boundary" is well-worn: DE adds NO new sub-signal — SL-019 is a MECHANIC, and DE's
  "College Production Share (Sack + TFL market share)" maps onto the existing `CollegeShare`
  field. No boundary field change expected.
- `peakLimit(PosDE)=30` and the template school tier (no DE branch) are ALREADY correct — confirm.
- Case status: 3A(WR+QB) 3B(RB) 3C(QB+K) 3F(DT) 3K(WR) 3M(TE) GREEN; 3D/3E/3G/3H/3I/3J PENDING;
  3L runs. Suite 0 FAIL.

== WHY DE IS NEXT (sequencing) ==
Build_Tracker row 19. DE is the FIRST SL-019 REUSE — it proves the `curve.SL019` extraction was
the right M17 call (DE inherits the helper with zero new mechanic code). LB (SL-OQ-023) reuses it
after. CONFIRM the sequence with Christopher before building.

== WHAT B5b-DE BUILDS ==
- A new DE rubric in the EXISTING `internal/engine/l4/defense` package (de.go), mirroring dt.go's
  structure (defense pkg) but WITHOUT SL-021 cushion and WITH SL-019 (like te.go's breakout half).
  `func NewDE() *DE` + `Apply(engine.Layer4Input) engine.Layer4Output`. Use the shared `curve`
  helpers (incl. `curve.SL019`) — do NOT duplicate them.
- DE mechanics from DE_Rubric.md (READ IT — triage every number vs the literal spec):
  - **RAS: ACTIVE, HIGH tier** (§3, SL-004) — cap ±8%, **steepness 10.0** (standard High-tier —
    NOT TE's 11.0; DE is the same shape as WR/DT). inflection 0.50, raw RAS / 10. Rookie weight
    1.00 ONLY (SL-018 receding schedule deferred — same call as every prior position). NOT forced.
  - **Film** ±5% (standard SL-002 — NOT DT's SL-005 ±3% compression; DE film cap is ±5%, §2),
    steepness 12.0 — **Data-Parity neutral** (IDP film weights UNSET, same as DT; confirm in
    Defense_Scouting_Source_Map — DE film redesigned onto Madden defense sub-attrs, weights unset).
  - **Breakout** weights **0.30 / 0.20 / 0.35 / 0.15** (breakout age / school tier / College
    Production Share / age trajectory) — College Production Share elevated to 0.35 (sack+TFL market
    share is the cleanest pre-NFL pass-rush signal). inflection 0.50, steepness 11.0, cap ±5%.
  - **Breakout Age** base curve (§4): ≤19.5→1.00, 20.0→0.80, 20.5→0.50, ≥21.0→0.15 (the RB-shaped
    aggressive dropoff, NOT TE's later curve). Linear interp between points.
  - **Age Trajectory** base curve (§4): peak 30 — …29→0.55, 30(peak)→0.50, 31→0.35, 32→0.20,
    33→0.10, ≥34→0.00 (READ the full table in §4 for the ≤ side). composition.peakLimit(PosDE)=30.
  - **School Tier**: TEMPLATE — boundary default, NO DE branch.

== THE SL-019 REUSE (the point of this build) ==
SL-019 applies at DE to the SAME two breakout sub-signals as TE (§4): (a) breakout-age and
(b) age-trajectory, BOTH at strength **0.35** (held at TE values pending calibration). This is a
DIRECT reuse of `curve.SL019` — copy te.go's pattern exactly:
  - `breakoutAgeNorm` modulated ONLY when `HasBreakoutAge` (don't modulate the absent-neutral);
  - `ageTrajectoryNorm` always modulated (age always present);
  - absent/out-of-range RAS and non-finite base contribute nothing (the helper already guards).
Worked examples to VERIFY against (§4):
  - breakout age base 0.15: RAS 9.99 → 0.447 · RAS 7.50 → 0.373
  - age trajectory base 0.50 (age 30): RAS 9.99 → 0.675 · base 0.20 (age 32): RAS 7.50 → 0.410 ·
    base 0.00 (age 34): RAS 4.18 → 0.146
DESIGN NOTE: the SL-019 strength (0.35) is already a param on the helper — pass DE's 0.35, do NOT
re-extract anything. If the math + Data-Parity all match TE's pattern, this rubric is nearly
mechanical; the real work is the harness case + the routing question below.

== THE HARNESS ==
- **Case 3E is DE's canonical SL-019 instance** and is currently `gatedPending("3E", … , PosDE)`.
  Convert it to a real `eval3E` — mirror `eval3M`'s STRUCTURE (it's the proven template): assert
  breakout rises with RAS (hi>lo>absent) AND prove breakout-age and age-trajectory are EACH
  RAS-modulated in isolation AND that school/college are NOT modulated (the negative contract).
  Registering DE flips 3E PENDING→PASS.
- **Reproduce the §5 "Lockett pattern at DE" structural finding** (line 405): L4 alone does NOT
  pull a declining elite-draft DE below 1.0 — assert via the live breakout pathway the same way
  3A/3B/3M did (film neutral). Two contrasting DE fixtures (athletic-strong-profile vs
  non-athletic-thin) — DE Alpha (0501) exists; add a Bravo contrast.
- Register DE in `harness_app.rubrics()` AND `realrubric_test.go realRegistry()`. Add
  TestRealDERegistryFlips3E + TestRealDERankingDifferentiates (mirror the TE tests).
- NOTE the suite count goes 13→13 (3E already exists — no new case added; it just flips). The
  `validation_test.go` empty-registry tallies (13 cases, 1 pass, 12 pending) DO NOT change.

== THE OPEN QUESTION — EDGE CLASSIFICATION ROUTING (SL-OQ-023 / OQ-004) ==
DE_Rubric §1 + §7: the DE rubric encompasses ALL pass-rush-primary EDGE defenders regardless of
MFL tag (DE / EDGE / 3-4 OLB); coverage/run-stop off-ball LBs route to the LB rubric. This is a
**dispatch / position-resolution** concern (harness case 3J — `pass_rush_snap_share` → DE vs LB),
NOT the DE L4 rubric itself. CONFIRM with Christopher: build ONLY the DE rubric this session and
leave 3J/the routing dispatch for the B5b-LB session (recommended — "version the boundary": don't
build the DE-vs-LB router until LB exists to route to), OR scope the dispatch in now.

== GATE CHECK (confirm with Christopher before writing code) ==
1. Sequence: DE next (recommended — first SL-019 reuse, proves the M17 helper) vs another position.
2. SL-019 reuse: DE uses `curve.SL019` at strength 0.35 on breakout-age + age-trajectory, same as
   TE. Confirm (it is the consistent call; §4 holds strengths at TE values).
3. Harness: convert `gatedPending("3E")` → real `eval3E` (mirror eval3M). Confirm.
4. EDGE routing (SL-OQ-023): defer the DE-vs-LB dispatch to B5b-LB (recommended) vs scope it now.
5. RAS ships rookie weight only; L3 amplified buffer (0.30×RAS_norm, §3 — same as TE) deferred to
   the L3 / data-wiring session.

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch session/b5b-de. Never git --no-verify.
- CT105: export PATH=$PATH:/usr/local/go/bin; go build ./...; GOMEMLIMIT=1500MiB GOGC=20 make
  lint (repo ROOT); go test -race ./... . Beelink GUI: wails dev -tags webkit2_41.
- Files < 400 lines (AD-17). Engine stays PURE — defense pkg imports only engine + curve;
  depguard engine-is-pure must stay green (PROVE with a planted USED import — a blank `_` import
  trips revive first and masks depguard; import + reference a store symbol instead).
- Confidence scores INTERNAL — never on Layer4Output/Result (FilmRaw is debug-only, allowed).
- Lint gotcha: never name a param/var `cap` (revive redefines-builtin) — use `capBand`.
- Every custom mechanic proven by a planted failure (M3); shared logic reused, not copied (M17 —
  DE must REUSE curve.SL019, not re-implement it).
- Review gate: GLM 5.2 (OpenCode on bird), BLIND, output is LEADS — triage every finding vs
  source. INLINE a self-contained prompt (GLM hangs on repo exploration) AND inline the
  composition/playerspec/defaults BOUNDARY files too (the standing lesson — every false positive
  GLM has produced came from inlining a definition but not its consumer). Run headless:
  opencode run -m zai/glm-5.2 "$(cat /tmp/prompt.md)". bird: ssh -i /root/.ssh/bird
  x@192.168.1.195. **The bird scp/ssh now has a standing allow rule in /root/.claude/settings.json
  (added B5b-TE) — the data-exfil classifier no longer hard-blocks the dispatch.** GLM track
  record: TE 0 hard (coverage leads only) / WR 1 / RB 1 / DT 3 / QB 2 / B5a 2 — earns its keep,
  still triage.

== CARRIED FORWARD — leads, not blockers ==
- SL-018 receding RAS position-weight schedules ship the ROOKIE weight only across ALL positions —
  wire the schedule when an NFL-career-stage input lands.
- L3 RAS-buffers: WR/RB standard (0.10×RAS_norm), TE + **DE amplified (0.30×RAS_norm, §3)** are
  Layer-3 age-pull mechanics, NOT built — build at the L3 / data-wiring session.
- Offense/IDP film weights UNSET (calibration pass) — film stays Data-Parity neutral.
- Breakout three-zone classification (Elite/Average/Late) is NOT surfaced on Layer4Output — no
  consumer yet. Add when something downstream needs the zone.
- SL-OQ-024: SL-019 strength SYMMETRY (0.35 both sub-signals) — may go asymmetric per position
  later; ship symmetric for DE (matches TE).
- The scouting [0,1]-normalized schema vs "engine normalizes from raw" reconciliation still pends
  the real-data-wiring session.
- K stays an identity PLACEHOLDER in rubrics() until B5b-K (DECISION-011, row 23).

== CLOSE GATE FOR THIS SESSION ==
- go build + make lint 0 + go test -race green; engine depguard green (planted USED import).
- SL-019 reuse verified vs DE_Rubric §4 worked examples (0.447/0.373/0.675/0.410/0.146); DE uses
  the SHARED curve.SL019 (no re-implementation — confirm by grep).
- Case 3E flips PENDING→PASS (or a real FAIL caught + fixed); Module 1 separates an athletic DE
  from a non-athletic DE; no assertion weakened.
- GLM 5.2 BLIND review (sliced, inlined prompts incl. the boundary files); triage vs source.
- Functional gate: Christopher operates the harness — sees 3E flip and DEs differentiate on
  athletic profile. Squash-merge after he confirms. Then write handoff 21 (B5b-LB — the SL-019
  reuse #2 + the EDGE-vs-LB dispatch routing question 3J; confirm sequencing).
