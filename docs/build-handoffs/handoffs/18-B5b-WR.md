HANDOFF — Session 18: B5b-WR (wide-receiver rubric — the High-tier RAS offense position)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B5b-RB (row 16, squash `f9875f6`) is MERGED. Three real rubrics now live: QB (offense
  skeleton, SL-020-forced RAS), DT (defense skeleton + SL-021 stress-test, active High-tier
  RAS), and RB (standard offense, active Medium-tier RAS). All on the shared
  `internal/engine/l4/curve` helpers.
- The "version the boundary" pattern is well-worn: add a sub-signal only when a rubric needs
  it (ScoutingInput + composition + harness); don't forward-design. RB grew NOTHING new
  (TouchShare deferred) — WR likewise should grow the boundary only if it consumes a signal
  the current ScoutingInput lacks (it should NOT — WR reuses film/RAS/breakout-age/school/
  college-share/age, all present).
- B5b-RB added the position-aware `schoolTierNorm(pos, tier)` at the boundary: RB gets a
  softer non-P4 penalty (0.75/0.45/0.15); QB/WR/DT keep the template (0.70/0.40/0.10).
  **WR uses the TEMPLATE values — verified vs WR_Rubric §4 — so NO boundary change needed.**
- Case status: 3B (RB), 3C (QB+K), 3F (DT) GREEN; 3A/3D/3E/3G/3H/3I/3J/3K PENDING; 3L runs.
  Suite 0 FAIL.

== WHY WR IS NEXT (sequencing) ==
Build_Tracker row 17. WR is the canonical offense-skill template (WR_Rubric §7: "WR provides
the cleanest template for offensive skill positions"). It is a STANDARD offense rubric like RB
but with HIGH-tier RAS (not Medium). It flips case 3A (the Lockett pattern) AND it is a
co-gate position for 3D and 3K. CONFIRM the sequence with Christopher before building; TE is
the only other unbuilt offense position (row 18), DE/LB/CB/S remain on defense.

== WHAT B5b-WR BUILDS ==
- A new WR rubric in the EXISTING `internal/engine/l4/offense` package (wr.go), mirroring
  rb.go/qb.go. `func NewWR() *WR` + `Apply(engine.Layer4Input) engine.Layer4Output`. Use the
  shared `curve` helpers — do NOT duplicate them.
- WR mechanics from WR_Rubric.md (READ IT — triage every number vs the literal spec):
  - **RAS: ACTIVE, HIGH tier** (WR_Rubric §3) — cap ±8%, steepness 10.0, inflection 0.50,
    raw RAS / 10. Same shape as DT's active RAS (mirror dt.go's RAS block, NOT qb.go's
    SL-020 force). **AD-09: do NOT force RAS to 1.000.** SL-018 schedule is 1.00 → 0.50 →
    0.10; ship the **rookie 1.00 weight ONLY** (defer the receding schedule — same
    "version the boundary" call as DT/RB; no consumer needs the NFL-career-stage input yet).
  - **Film** ±5%, steepness 12.0, inflection 0.50 — **Data-Parity neutral** (offense film
    weights UNSET pending the redesign; confirm in the source map — likely identical to QB/RB).
  - **Breakout** weights 0.40/0.25/0.20/0.15 (breakout age / school tier / college usage /
    age trajectory), inflection 0.50, steepness 11.0, cap ±5%.
  - **Breakout Age** curve (≥20% team receiving production threshold): ≤19→1.00, 20.0→0.75,
    21.0→0.40, ≥22→0.10 — confirm the ≤19 anchor vs spec.
  - **College Usage** curve (final-year target share): confirm the anchors in WR_Rubric §4.
  - **Age Trajectory** curve relative to WR **peak 29** (composition.peakLimit(PosWR)=29 —
    already correct; confirm vs spec).
  - **School Tier**: TEMPLATE values, already handled by the boundary's default branch — do
    NOT add a WR branch to schoolTierNorm.
- **SL-019 NOT applied for v1.0 (SL-022)** — Layer 3 carries WR aging; the breakout sub-signals
  use base curve values with no RAS modulation (like QB, unlike DT which excludes SL-019 via
  the cushion guard). Do NOT add an SL-019 modulator.
- Register WR in `harness_app.rubrics()` AND `harness/realrubric_test.go` `realRegistry()`.

== THE HARNESS IS YOUR ACCEPTANCE GATE ==
- **Case 3A (Lockett pattern — L4 NEAR-NEUTRAL for declining elite vets)** gates on BOTH WR
  AND QB (already registered). Registering WR should let 3A flip. Convert the
  `gatedPending("3A", …)` entry into a real `eval3A`. Lockett's point (WR_Rubric §5, the
  contrast to RB's Herbert): a declining vet with a STRONG draft-era profile stays ~1.00–1.01
  (static breakout signals positive), NOT below 1.0 — the opposite of Herbert. NOTE: with film
  Data-Parity neutral, reproduce this via the LIVE components (breakout near-neutral/positive +
  active RAS) exactly as 3B did for the thin case; the film-driven half lands with the film
  source. CONFIRM the assertion shape with Christopher at the gate (same call as 3B).
- **Case 3D (SL-005 film compression ±3% at LB/DT vs ±5% elsewhere)** co-gates DT (built) +
  LB + WR. Registering WR removes WR from its missing-list but 3D still PENDS on LB — it will
  NOT flip this session. That is correct three-state behavior.
- **Case 3K (S-curve boundary safety — output clamped to [1-cap,1+cap])** gates on WR. Once WR
  registers, convert `gatedPending("3K", …)` into a real `eval3K` if its sub-signal inputs
  exist (extreme film/breakout inputs through the registered WR rubric → assert each component
  stays in band; the rb_test/qb_test M3 clamp test is the template). CONFIRM whether 3K flips
  this session or pends.
- Add planted-failure tests (M3) for the active RAS + clamp; B1 absent-neutral regression.
- Two contrasting WR fixtures already exist (WR Alpha/Bravo/Charlie in fixtures.go) — extend
  them with the breakout/college-usage fields WR reads so Module 1 visibly separates them.

== READ FIRST ==
- docs/scoring-engine/WR_Rubric.md — authoritative WR mechanics. Triage every number.
- internal/engine/l4/defense/dt.go — the ACTIVE High-tier RAS block to mirror (±8%/steep 10/
  rookie weight). internal/engine/l4/offense/rb.go — the standard-offense shape + Medium-tier
  RAS (WR is the same shape, High tier). internal/engine/l4/offense/qb.go — SL-019-absent
  breakout (WR matches QB here, not DT).
- internal/engine/l4/curve/curve.go — shared helpers (scurve takes capBand as a PARAM).
- internal/engine/types.go, internal/composition/{playerspec.go,composition.go,defaults.go},
  internal/harness/{cases.go,fixtures.go,realrubric_test.go} — the ripple surface (mirror the
  B5b-RB diff exactly; note schoolTierNorm is ALREADY position-aware — WR needs no branch).
- docs/data-layer/Offense_Scouting_Source_Map.md — whether any WR film source is live.

== GATE CHECK (confirm with Christopher before writing code) ==
1. Sequence: WR next (recommended — cleanest offense template, flips 3A) vs TE/a defense position.
2. Film: Data-Parity neutral (likely, weights unset) vs a live WR film source — confirm in the
   source map.
3. Case 3A assertion shape: with film neutral, assert the near-neutral Lockett pattern via the
   live breakout+RAS components (same principled substitution as 3B). Confirm.
4. Case 3K: does it flip this session (WR is its sole gate) or stay pending? Confirm.
5. SL-019 stays EXCLUDED (SL-022); RAS ships rookie weight only.

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch session/b5b-wr. Never git --no-verify.
- CT105: export PATH=$PATH:/usr/local/go/bin; go build ./...; GOMEMLIMIT=1500MiB GOGC=20 make
  lint (repo ROOT); go test -race ./... . Beelink GUI: wails dev -tags webkit2_41.
- Files < 400 lines (AD-17). Engine stays PURE — offense pkg imports only engine + curve;
  depguard engine-is-pure must stay green (PROVE with a planted USED import — a blank `_`
  import trips revive first and masks depguard; import + reference a store symbol instead).
- Confidence scores INTERNAL — never on Layer4Output/Result (FilmRaw is debug-only, allowed).
- Lint gotcha: never name a param/var `cap` (revive redefines-builtin) — use `capBand`.
- Every custom gate proven by a planted failure (M3); shared logic reused, not copied (M17).
- Review gate: GLM 5.2 (OpenCode on bird), BLIND, output is LEADS — triage every finding vs
  source. INLINE source into the prompt (scp self-contained prompt; GLM hangs on repo
  exploration) AND inline the composition/playerspec/defaults boundary files too (the B5b-RB
  review flagged the schoolTierNorm seam only because the boundary was inlined — that is the
  win; keep doing it). Run headless: opencode run -m zai/glm-5.2 "$(cat /tmp/prompt.md)".
  bird: ssh -i /root/.ssh/bird x@192.168.1.195. GLM track record: RB 1 real / DT 3 / QB 2 /
  B5a 2 — earns its keep, still triage.

== CARRIED FORWARD — leads, not blockers ==
- SL-018 L3 RAS-buffer (RB_Rubric §3 / WR_Rubric §3: `buffer_pct = 0.10 × RAS_normalized`,
  buffered_age_pull past peak) is a LAYER-3 age-pull mechanic, NOT built — out of L4 rubric
  scope. WR is where it was first specified; flag for the L3 / data-wiring session whether to
  build it (it would ride on Calibration like the DT cushion guard, but for the standard
  RAS-buffer formula, not the cushion-decline one).
- RAS receding position-weight schedules (QB n/a, RB 0.60→0.30→0.06, DT/WR 1.00→0.50→0.10) all
  ship the ROOKIE weight only — wire the schedule when an NFL-career-stage input lands (a
  consumer needs it first).
- Offense/IDP film weights UNSET (calibration pass) — film stays Data-Parity neutral until the
  film-source redesign lands.
- The scouting [0,1]-normalized schema convention vs "engine normalizes from raw" still needs
  reconciliation when REAL scouting.Profile data is wired into composition (today fixtures
  supply raw facts directly). Flag at the data-wiring session.
- K stays an identity PLACEHOLDER in rubrics() until B5b-K (DECISION-011, row 23).

== CLOSE GATE FOR THIS SESSION ==
- go build + make lint 0 + go test -race green; engine depguard green (planted USED import).
- Case 3A GREEN (or a real FAIL caught + fixed); 3K resolved (flip or documented pend); no
  assertion weakened.
- GLM 5.2 BLIND review (sliced, inlined prompts incl. the boundary files); triage vs source.
- Functional gate: Christopher operates the harness — sees 3A flip, sees WRs differentiate on
  the active RAS + breakout. Squash-merge after he confirms. Then write handoff 19 (B5b-TE —
  confirm sequencing; TE adds SL-019 modulators in scoreRAS/scoreBreakout per Build_Tracker).
