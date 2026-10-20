HANDOFF — Session 22: B5b-CB (cornerback rubric — first NGS-anchor position + THIRD SL-019 instance)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B5b-LB (row 20, squash `2be23f1`) is MERGED. SEVEN real rubrics live: QB (SL-020-forced RAS),
  RB (Medium RAS), WR (High RAS), TE (High RAS @ steep 11.0 + first SL-019), DT (SL-021 cushion +
  SL-005 film ±3%), DE (High RAS @ steep 10.0 + second SL-019), LB (SL-005 film ±3% + Medium RAS @
  steep 11.0, first non-SL-019 IDP + EDGE dispatch resolver).
- The shared `internal/engine/l4/curve` helpers are the spine: Scurve, Interp, SubSignal, Present,
  SL019. The defense pkg is `internal/engine/l4/defense` (dt.go/de.go/lb.go).
- Case status: 3A(WR+QB) 3B(RB) 3C(QB+K) 3D(LB/DT/WR) 3E(DE) 3F(DT) 3J(EDGE dispatch) 3K(WR)
  3M(TE) GREEN; 3G/3H/3I PENDING; 3L runs. Suite 0 FAIL, 13 cases.
- EDGE routing: `composition.ResolveRubricPosition` routes DE/LB by pass_rush_snap_share ≥0.50; it
  is GENERAL — CB/S are not in the ambiguous pair, so they pass through unchanged (no CB change).

== WHY CB IS NEXT (sequencing) ==
Build_Tracker row 21. CB is the FIRST of the two NGS-anchor positions (CB row 21, S row 22). It
unlocks case 3I (NGS anchor present only at CB & S) and is the THIRD SL-019 instance — proving the
shared curve.SL019 transfers a THIRD time, now at REDUCED strengths. CONFIRM CB-before-S with
Christopher (both are NGS positions; the rubrics are near-identical — S reuses everything CB builds).

== WHAT B5b-CB BUILDS ==
- A new CB rubric in the EXISTING `internal/engine/l4/defense` package (cb.go), mirroring de.go's
  SL-019 structure. `func NewCB() *CB` + `Apply(engine.Layer4Input) engine.Layer4Output`. Use the
  shared `curve` helpers — REUSE curve.SL019 (do NOT re-implement; M17).
- CB mechanics from CB_Rubric.md (READ IT — triage every number vs the literal spec):
  - **Layer 4 RAS Tier: HIGH** (§3, SL-004). cap **±8%**, `RAS_steepness` **11.0**, inflection 0.50,
    raw RAS / 10. RAS_position_weight High-tier SL-018 schedule 1.00 / 0.50 / 0.10 — v1.0 ships the
    ROOKIE weight **1.00 ONLY** (same deferral as every prior position). AD-09 NOT forced (active RAS).
  - **Film: STANDARD ±5%** (§2 — NOT SL-005 compressed; CB is NOT a thin-data IDP, NGS is rich).
    `film_steepness` standard 12.0, inflection 0.50, weight 1.00. **Data-Parity neutral in v1.0**
    (film weights UNSET — same as every IDP so far). NOTE: the NGS dedicated anchor (0.30 weight)
    lives INSIDE the film composite; it does not change the engine's film input (still a single
    [0,1] composite) — see the 3I note below for how NGS-presence is asserted.
  - **SL-019: YES — THIRD instance, REDUCED strengths** (§1/§3/§4). Strengths drop from TE/DE's
    0.35/0.35/0.30 to **0.30 (breakout-age) / 0.30 (age-traj) / 0.25 (L3 buffer)**. The L4 rubric
    applies the **0.30** strength to breakout-age (present-only) AND age-trajectory via curve.SL019.
    The 0.25 L3 buffer is L3 (NOT built — carry-forward). Worked examples (§4): breakout-age base
    0.15 @ RAS 9.99 → 0.405, @ RAS 5.00 → 0.278; base 1.00 → 1.00 (inert).
  - **Breakout** (§4): cap ±5%, steepness **10.0** (NOT DE/LB's 11.0 — CB college data is noisier),
    inflection 0.50. Weights **0.20 / 0.25 / 0.40 / 0.15** (breakout age / school / College
    Production Share / age traj). School Tier ELEVATED to 0.25 (NFL CBs come from P4 reliably);
    College Production Share 0.40 (PD + INT market share, cleanest college-to-NFL predictor);
    breakout age dropped to 0.20.
  - **Breakout Age** base curve (§4, half-year DE-shaped): ≤19.5→1.00, 20.5→0.75, 21.5→0.45,
    ≥22.5→0.15.
  - **College Production Share** (§4): READ §4 for the three anchors (PD + INT market share).
  - **Age Trajectory** base curve (§4): peak **28** (one earlier than WR/TE/LB's 29) — READ §4 for
    the full table. composition.peakLimit(PosCB)=28 (verify — already set in defaults.go).
  - **School Tier**: TEMPLATE (P4 1.00 / G5 0.70 / FCS 0.40 / Non-FCS 0.10) — boundary default, NO
    CB branch (schoolTierNorm is already position-aware; verify CB uses the template path).
  - **SL-021 cushion: NOT applied** (DT-specific). cushionGuard(PosCB) already returns (0,0).

== THE CASE CB UNLOCKS — 3I (the real design question this session) ==
Case 3I — "NGS anchor present only at CB & S." Currently `gatedPending("3I", … , PosCB, PosS)` with
detail "rubric sub-signal weights not introspectable yet." The film composite is a single [0,1]
value at the engine boundary (Data-Parity neutral, weights unset) — so NGS-presence is NOT visible
on Layer4Output. **GATE-CHECK with Christopher:** how should 3I flip?
  (a) RECOMMENDED — add a rubric INTROSPECTION hook (like DT.PFFAlpha for 3G): e.g.
      `func (cb *CB) HasNGSAnchor() bool { return true }` and the offense/other-defense rubrics
      return false (or an enum/weight accessor). eval3I asserts the hook is true at CB (and S once
      built) and false at a non-NGS position (e.g. WR). This stays a RUBRIC INTERNAL — never on
      Layer4Output (the production surface), exactly like PFFAlpha. It is the minimal honest way to
      assert a sub-signal-architecture fact the engine boundary deliberately hides.
  (b) Defer 3I until S is built too (both NGS positions present) — but the hook is cheap and CB
      alone can assert "CB has it, WR doesn't," so deferring leaves a permanent PENDING needlessly.
  (c) Surface NGS weight on a debug hook (heavier; not recommended — film weights are unset in v1.0
      so the number is 0.00, making a weight accessor misleading until the calibration pass).
3I co-gates CB AND S, so even with the hook it stays PENDING until S registers UNLESS eval3I is
written to assert the property per-registered-position (recommended: assert at whichever NGS rubric
is registered, like eval3A asserts the non-negative property at both WR and QB). Decide the shape.

== THE HARNESS ==
- Register CB in `harness_app.rubrics()` AND `realrubric_test.go realRegistry()`.
- If the 3I hook is built: convert `gatedPending("3I")` → real `eval3I` (mirror eval3F/eval3M
  structure; assert the NGS-anchor introspection true at CB / false at a non-NGS position). Add
  TestRealCBRegistryFlips3I. If 3I is written to require BOTH CB+S, it stays PENDING until S — note
  that explicitly so the close gate expects it.
- Add a real `eval3?` for the SL-019 reuse at CB if a NEW case is warranted — but NOTE 3E (DE) and
  3M (TE) already assert SL-019; CB's reduced-strength variant is best proven by a UNIT test
  (cb_test.go worked examples at strength 0.30) + the ranking differentiation, NOT a new suite case
  (suite stays 13 unless 3I splits). Confirm no new case is added.
- Add CB fixtures to fixtures.go (CB Alpha / CB Bravo — High-tier RAS + SL-019, so athletic profile
  separates them like DE Alpha/Bravo) + TestRealCBRankingDifferentiates. CB Alpha (0601) already
  exists as a bare identity fixture — UPGRADE it with scouting sub-signals + add CB Bravo (0602).
- WATCH: TestRealDTRegistryFlips3F / TestRealLBRegistryFlips3DAnd3J / TestRealDERegistryFlips3E /
  TestRealWRRegistryFlips3AAnd3K each assert specific cases STAY PENDING (3G, etc). grep every test
  asserting 3I PENDING before you commit — registering CB may flip 3I (if eval3I is per-position)
  and those assertions go stale (same pattern as the B5b-LB update to the DT/WR tests).
- Suite count: if 3I flips without splitting, stays 13. The `validation_test.go` empty-registry
  tally (13 cases, 1 pass, 12 pending) does NOT change.

== GATE CHECK (confirm with Christopher before writing code) ==
1. Sequence: CB next (recommended — first NGS anchor, third SL-019) vs S first.
2. CB = High-tier RAS (±8%/steep 11.0) + standard ±5% film + SL-019 at REDUCED strength 0.30 +
   breakout steepness 10.0 + peak 28. Confirm.
3. **Case 3I: build a rubric introspection hook (HasNGSAnchor, option a) and flip 3I now asserting
   per-registered-position, vs defer 3I until S.** (The real call this session.)
4. RAS ships High rookie weight 1.00 only; L3 SL-019 buffer (0.25×RAS_norm) deferred to L3.
5. No new suite case for CB's SL-019 (proven by cb_test.go unit + ranking); suite stays 13.

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch session/b5b-cb. Never git --no-verify.
- CT105: export PATH=$PATH:/usr/local/go/bin; go build ./...; GOMEMLIMIT=1500MiB GOGC=20 make lint
  (repo ROOT); go test -race ./... . Beelink GUI: wails dev -tags webkit2_41.
- Files < 400 lines (AD-17). cases_eval.go and cases_eval_dispatch.go are the split template if a
  new eval pushes a file over 400. Engine stays PURE — defense pkg imports only engine + curve;
  depguard engine-is-pure must stay green (PROVE with a planted USED store import — a blank `_`
  import trips revive first and masks depguard; import + REFERENCE a real exported symbol, e.g.
  `var _ = params.ValueType("")`).
- Confidence scores INTERNAL — never on Layer4Output/Result (FilmRaw is debug-only; an NGS/PFF
  introspection hook is a RUBRIC method, not a Layer4Output field — same as DT.PFFAlpha).
- Lint gotcha: never name a param/var `cap` (revive redefines-builtin) — use `capBand`. Avoid a bare
  `switch` on domain.Position unless you enumerate ALL members (exhaustive linter) — an if/else
  chain sidesteps it (the B5b-LB resolver lesson).
- Every custom mechanic proven by a planted failure (M3); shared logic reused, not copied (M17 — CB
  must REUSE curve.SL019 at strength 0.30, not re-implement). Pin MAGNITUDE through the consumer
  Apply, not just the helper (the standing GLM lesson — add a TestCBMagnitudeThroughApply).
- Review gate: GLM 5.2 (OpenCode on bird), BLIND, output is LEADS — triage every finding vs source.
  INLINE a self-contained prompt AND the composition/playerspec/defaults BOUNDARY files (every GLM
  false positive came from inlining a definition but not its consumer). Run headless:
  opencode run -m zai/glm-5.2 "$(cat /tmp/prompt.md)". bird: ssh -i /root/.ssh/bird x@192.168.1.195.
  scp/ssh have a standing allow rule in /root/.claude/settings.json (data-exfil classifier cleared).
  GLM track record: LB 0 hard / DE 0 / TE 0 / WR 1 / RB 1 / DT 3 / QB 2 / B5a 2 — earns its keep,
  still triage. (GLM has misread no-op guards before — verify its "defects" against the actual code.)

== CARRIED FORWARD — leads, not blockers ==
- SL-018 receding RAS position-weight schedules ship the ROOKIE weight only across ALL positions —
  wire the schedule when an NFL-career-stage input lands.
- L3 RAS-buffers (Layer-3 age-pull): WR/RB/LB standard 0.10×RAS_norm, TE/DE 0.30×RAS_norm, CB
  0.25×RAS_norm — all Layer-3 mechanics, NOT built. Build at the L3 / data-wiring session.
- Offense/IDP film weights UNSET (calibration pass) — film stays Data-Parity neutral everywhere,
  including CB's NGS anchor (weight 0.30 but the composite source is unpopulated in v1.0).
- 3G (DT PFFAlpha assertion-wiring) still gatedPending though DT+DE+LB registered — wire when needed.
- 3H (confidence floor) needs component confidence inputs on Layer4Input — not built.
- EDGE dispatch threshold (SL-OQ-030, 0.50 default) pends calibration; resolver is live.
- SL-OQ-031/CAL-024/CAL-025 (LB), SL-OQ-034/CAL-027 (CB) college-data/NGS-source questions pend the
  pipeline session. SL-OQ-035/036 carried for S (row 22).
- The scouting [0,1]-normalized schema vs "engine normalizes from raw" reconciliation pends the
  real-data-wiring session.
- After CB: S (row 22) is near-identical (NGS anchor + SL-019), then K stays an identity PLACEHOLDER
  until B5b-K (DECISION-011, row 23). CB+S complete the IDP set.

== CLOSE GATE FOR THIS SESSION ==
- go build + make lint 0 + go test -race green; engine depguard green (planted USED import).
- CB constants/curves verified vs CB_Rubric §2/§3/§4 (incl. the SL-019 worked examples at strength
  0.30: breakout-age base 0.15 → 0.405 @ RAS 9.99, 0.278 @ RAS 5.00); CB reuses curve.SL019 (no
  re-implementation — confirm by reading cb.go vs de.go) at the REDUCED 0.30 strength.
- TestCBMagnitudeThroughApply pins RAS (High ±8%/steep 11.0) AND the SL-019 0.30 strength AT the
  call site (the standing "pin magnitude through the consumer" lesson).
- Case 3I resolved per the gate-check (flips PENDING→PASS if the hook is built + eval3I is
  per-position, or stays PENDING pending S — whichever Christopher chooses; every test asserting 3I
  PENDING updated either way). Module 1 separates CB Alpha from CB Bravo.
- GLM 5.2 BLIND review (sliced, inlined incl. boundary files; pin magnitude through Apply); triage.
- Functional gate: Christopher operates the harness — sees 3I flip (or correctly stay PENDING) and
  CBs differentiate. Squash-merge after he confirms. Then write handoff 23 (B5b-S).
