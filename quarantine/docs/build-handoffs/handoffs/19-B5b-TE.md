HANDOFF — Session 19: B5b-TE (tight-end rubric — the FIRST SL-019 position)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B5b-WR (row 17, squash `448a2bf`) is MERGED. Four real rubrics now live: QB (offense
  skeleton, SL-020-forced RAS), DT (defense skeleton + SL-021 cushion, High-tier active RAS),
  RB (standard offense, Medium-tier active RAS), and WR (offense skill template, High-tier
  active RAS). All on the shared `internal/engine/l4/curve` helpers.
- The "version the boundary" pattern is well-worn: add a sub-signal ONLY when a rubric needs
  it; don't forward-design. WR grew NOTHING new (reused film/RAS/breakout-age/school/
  college-share/age, all present). **TE likewise reuses the SAME ScoutingInput — it adds a
  MECHANIC (SL-019), not a sub-signal. No boundary field change expected.**
- `schoolTierNorm(pos, tier)` at the boundary is position-aware (RB softer non-P4; QB/WR/DT/TE
  use the TEMPLATE). **TE uses the template (verified vs TE_Rubric §4 line 166-173) — NO TE
  branch.**
- Case status: 3A (WR+QB), 3B (RB), 3C (QB+K), 3F (DT), 3K (WR) GREEN; 3D/3E/3G/3H/3I/3J
  PENDING; 3L runs. Suite 0 FAIL.

== WHY TE IS NEXT (sequencing) ==
Build_Tracker row 18. TE is the FIRST position to apply **SL-019 RAS-modulator interactions**
(TE_Rubric §1/§3/§4, §7: "the mechanic is generalizable — codified for use by other rubrics").
DE/LB (SL-OQ-023) will reuse it, so building TE first establishes the shared SL-019 helper
(M17) the defense positions inherit — the same reason DT was built before the rest of the IDP
group. CONFIRM the sequence with Christopher before building.

== WHAT B5b-TE BUILDS ==
- A new TE rubric in the EXISTING `internal/engine/l4/offense` package (te.go), mirroring
  wr.go/rb.go. `func NewTE() *TE` + `Apply(engine.Layer4Input) engine.Layer4Output`. Use the
  shared `curve` helpers — do NOT duplicate them.
- TE mechanics from TE_Rubric.md (READ IT — triage every number vs the literal spec):
  - **RAS: ACTIVE, HIGH tier** (TE_Rubric §3, SL-004 AMENDED Medium→High) — cap ±8%,
    **steepness 11.0 (STEEPER than WR's 10.0 — TE-specific, §3)**, inflection 0.50, raw RAS /
    10. Same active block as WR/DT (mirror it), **NOT** SL-020-forced. Ship the **rookie 1.00
    weight ONLY** (defer the SL-018 receding schedule — same "version the boundary" call).
  - **Film** ±5%, steepness 12.0 — **Data-Parity neutral** (offense film weights UNSET, same
    as QB/RB/WR; confirm in Offense_Scouting_Source_Map §5 — TE row also "0.00 — empty").
  - **Breakout** weights 0.35/0.20/0.30/0.15 (SAME as RB), inflection 0.50, steepness 11.0,
    cap ±5%.
  - **Breakout Age base curve** (later than WR): ≤20→1.00, 21→0.80, 22→0.50, ≥23→0.15.
  - **College Usage base curve** (TE target share, lower than WR): ≥22%→1.00, 15%→0.50,
    ≤8%→0.10.
  - **Age Trajectory base curve**: TE peak 29, SAME shape as WR (≤25→1.00 … 29→0.50 …
    ≥33→0.00). composition.peakLimit(PosTE)=29 — ALREADY correct, confirm.
  - **School Tier**: TEMPLATE — boundary default, NO TE branch.

== THE NEW MECHANIC: SL-019 RAS-MODULATOR (the heart of this build) ==
SL-019 lifts TWO breakout sub-signals by the player's athletic profile BEFORE they enter the
breakout composite (TE_Rubric §4):

    modulated = base + (1.0 − base) × 0.35 × RAS_normalized        (RAS_normalized = raw RAS/10)

Applied to (a) the **breakout-age** sub-signal and (b) the **age-trajectory** sub-signal. NOT
applied to school tier or college usage. Worked examples to VERIFY your implementation against:
  - breakout age 22 (base 0.50): RAS 9.5 → 0.666 · RAS 7.0 → 0.623 · RAS 4.0 → 0.570 (§4:159-162)
  - age trajectory 32 (base 0.10): RAS 9.5 → 0.399 · RAS 5.4 → 0.270 (§4:205-207)
  - a base already at 1.00 (early breakout / young age) → modulator has NO effect (1.0−base=0).
DESIGN CALLS to make and surface at the gate:
  1. **Absent RAS:** SL-019 needs RAS_normalized. When HasRAS is false (Data-Parity), the
     modulator must contribute NOTHING — i.e. `modulated = base` (multiply the lift by 0, or
     skip it). Do NOT impute. This mirrors the rubric's existing "absent RAS → neutral" stance.
     Confirm this is the right call (it is the consistent one).
  2. **Where SL-019 lives (M17):** TE is the FIRST instance but DE/LB will reuse it. Put the
     modulator as a shared pure helper in the `curve` package (e.g.
     `curve.SL019(base, rasNorm float64, hasRAS bool, strength float64) float64`) so DE inherits
     it — the same M17 extraction discipline that moved scurve/interp to `curve`. Strength 0.35
     is TE-specific → pass it as a param, don't hardcode in the helper.
  3. **L3 amplified buffer (0.30 × RAS_normalized, §3:106-117) is LAYER 3, NOT L4** — same
     class as the WR/RB RAS-buffer carry-forward. Do NOT build it in the rubric; flag for the
     L3 / data-wiring session. (TE's is 3× WR's standard; note that when it lands.)

== THE HARNESS — NOTE: NO EXISTING CASE GATES ON TE ==
**Important:** scan validationCases() — 3A(WR,QB) 3B(RB) 3C(QB,K) 3D(LB,DT,WR) 3E(DE) 3F(DT)
3G(DT,DE) 3I(CB,S) 3K(WR) — **none gate on PosTE**, and 3E (SL-019 breakout modulator) is
gated on **DE**, not TE. So registering TE flips NO existing case. The acceptance gate is
therefore:
  1. **Unit tests in te_test.go** proving the SL-019 modulator math against the §4 worked
     examples above (this is the M3 proof for the new mechanic — a planted wrong strength must
     fail), plus the standard RAS-active / film-active / absent-neutral / clamp tests mirrored
     from wr_test.go. Add a SPECIFIC test that an absent-RAS TE gets base (no modulation) and a
     high-RAS TE gets the lifted value — the SL-019 contract.
  2. **A real harness case for SL-019 at TE** — RECOMMENDED: convert nothing existing (3E is
     DE's); instead ADD a new case (e.g. "3M — SL-019 modulator lifts breakout with RAS at TE")
     OR generalize 3E to co-gate {DE, TE} and flip its TE half now. CONFIRM with Christopher
     which: a new TE case is cleaner (3E stays DE's canonical instance), but co-gating 3E is
     less case-sprawl. Either way the case must assert: same TE, higher RAS → higher
     breakout-age + age-trajectory contribution → higher breakout component (the Kittle vs
     non-athletic-late-developer contrast, §4:164).
  3. **Module 1 differentiation** — register TE in `harness_app.rubrics()` AND
     `realrubric_test.go realRegistry()`; extend the TE fixture(s) (TE Alpha 0401 exists; add a
     contrasting athletic-vet vs non-athletic-vet pair so SL-019 visibly separates them — the
     Higbee≈0.99 vs Henry≈1.05 structural finding, §5 Cases 2A/2B).
- The Higbee/Henry §5 cases are the gold validation: two near-identical-RAS aging TEs whose L4
  differs by profile strength. Reproduce the STRUCTURAL claim (strong static profile → breakout
  stays up) the same film-neutral way 3A/3B did — assert via the live breakout pathway.

== READ FIRST ==
- docs/scoring-engine/TE_Rubric.md — authoritative. §3 (RAS steepness 11.0 + L3 buffer), §4
  (SL-019 modulator formula + worked examples), §5 (Bowers/Higbee/Henry), §7 (SL-019 is
  generalizable). Triage every number.
- internal/engine/l4/offense/wr.go — the High-tier active-RAS template to mirror (TE differs
  only in RAS steepness 11.0, the later breakout-age/usage curves, and +SL-019).
  internal/engine/l4/offense/rb.go — the 0.35/0.20/0.30/0.15 breakout weights (TE shares them).
- internal/engine/l4/curve/curve.go — shared helpers; ADD the SL-019 helper here (M17).
- internal/engine/types.go, internal/composition/{playerspec.go,composition.go,defaults.go},
  internal/harness/{cases.go,fixtures.go,realrubric_test.go} — the ripple surface. Expect NO
  boundary field change (SL-019 is a mechanic, not a sub-signal); peakLimit(PosTE)=29 + template
  school tier are already correct. INLINE the boundary files in the GLM prompt anyway (the
  standing lesson — B5b-WR's MED3 false-positive came from NOT inlining the consumer wiring).

== GATE CHECK (confirm with Christopher before writing code) ==
1. Sequence: TE next (recommended — establishes the shared SL-019 helper DE/LB reuse) vs a
   defense position.
2. SL-019 absent-RAS behavior: modulator contributes nothing (modulated = base). Confirm.
3. SL-019 helper location: shared in `curve` (M17, strength as a param). Confirm.
4. Harness: NEW TE case for SL-019 vs co-gating 3E {DE,TE}. Confirm which.
5. RAS ships rookie weight only; L3 amplified buffer (0.30×RAS_norm) deferred to L3 session.

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch session/b5b-te. Never git --no-verify.
- CT105: export PATH=$PATH:/usr/local/go/bin; go build ./...; GOMEMLIMIT=1500MiB GOGC=20 make
  lint (repo ROOT); go test -race ./... . Beelink GUI: wails dev -tags webkit2_41.
- Files < 400 lines (AD-17). Engine stays PURE — offense pkg imports only engine + curve;
  depguard engine-is-pure must stay green (PROVE with a planted USED import — a blank `_`
  import trips revive first and masks depguard; import + reference a store symbol instead).
- Confidence scores INTERNAL — never on Layer4Output/Result (FilmRaw is debug-only, allowed).
- Lint gotcha: never name a param/var `cap` (revive redefines-builtin) — use `capBand`.
- Every custom mechanic proven by a planted failure (M3); shared logic reused, not copied (M17).
- Review gate: GLM 5.2 (OpenCode on bird), BLIND, output is LEADS — triage every finding vs
  source. INLINE source into the prompt (scp self-contained prompt; GLM hangs on repo
  exploration) AND inline the composition/playerspec/defaults boundary files too. Run headless:
  opencode run -m zai/glm-5.2 "$(cat /tmp/prompt.md)". bird: ssh -i /root/.ssh/bird
  x@192.168.1.195. GLM track record: WR 1 real / RB 1 / DT 3 / QB 2 / B5a 2 — earns its keep,
  still triage. (The bird scp/ssh trips the sandbox data-exfil classifier — it is Christopher's
  standing review gate; proceed with the sandbox override when he directs, per his authorization.)

== CARRIED FORWARD — leads, not blockers ==
- SL-018 receding RAS position-weight schedules (1.00→0.50→0.10 at High tier) ship the ROOKIE
  weight only across ALL positions — wire the schedule when an NFL-career-stage input lands.
- L3 RAS-buffers: WR/RB standard (0.10×RAS_norm) AND **TE amplified (0.30×RAS_norm, §3)** are
  Layer-3 age-pull mechanics, NOT built — build at the L3 / data-wiring session.
- Offense/IDP film weights UNSET (calibration pass) — film stays Data-Parity neutral.
- Breakout three-zone classification (Elite/Average/Late, every rubric §4) is NOT surfaced on
  Layer4Output — no consumer yet. Add when something downstream needs the zone.
- The scouting [0,1]-normalized schema vs "engine normalizes from raw" reconciliation still
  pends the real-data-wiring session.
- K stays an identity PLACEHOLDER in rubrics() until B5b-K (DECISION-011, row 23).
- SL-OQ-024: SL-019 strength SYMMETRY (TE uses 0.35 for BOTH breakout-age and age-trajectory) —
  may go asymmetric per position later; ship symmetric for TE.

== CLOSE GATE FOR THIS SESSION ==
- go build + make lint 0 + go test -race green; engine depguard green (planted USED import).
- SL-019 modulator math verified vs TE_Rubric §4 worked examples (0.666/0.623/0.570 etc.);
  shared SL-019 helper in `curve` (M17); a planted wrong strength FAILS (M3).
- The TE SL-019 harness case GREEN (or a real FAIL caught + fixed); Module 1 separates an
  athletic-vet TE from a non-athletic-vet TE (Higbee/Henry finding); no assertion weakened.
- GLM 5.2 BLIND review (sliced, inlined prompts incl. the boundary files); triage vs source.
- Functional gate: Christopher operates the harness — sees the SL-019 case flip and sees TEs
  differentiate on athletic profile. Squash-merge after he confirms. Then write handoff 20
  (B5b-DE — the SL-019 reuse + the EDGE classification routing question, SL-OQ-023; confirm
  sequencing).
