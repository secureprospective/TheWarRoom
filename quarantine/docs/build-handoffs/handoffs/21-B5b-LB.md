HANDOFF — Session 21: B5b-LB (linebacker rubric — first non-SL-019 IDP rubric + EDGE-vs-LB dispatch)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B5b-DE (row 19, squash `202b0e0`) is MERGED. SIX real rubrics live: QB (SL-020-forced RAS),
  RB (Medium-tier RAS), WR (High RAS), TE (High RAS @ steep 11.0 + first SL-019), DT (SL-021
  cushion + SL-005 film ±3%), DE (High RAS @ steep 10.0 + second SL-019, reusing curve.SL019).
- The shared `internal/engine/l4/curve` helpers are the spine: Scurve, Interp, SubSignal,
  Present, and SL019. LB introduces NO new mechanic — it COMBINES two existing ones (see below).
- Case status: 3A(WR+QB) 3B(RB) 3C(QB+K) 3E(DE) 3F(DT) 3K(WR) 3M(TE) GREEN; 3D/3G/3H/3I/3J
  PENDING; 3L runs. Suite 0 FAIL, 13 cases.

== WHY LB IS NEXT (sequencing) ==
Build_Tracker row 20. LB is the LAST defense-pkg rubric before CB/S (the NGS positions). It
proves the architecture composes — LB reuses DT's SL-005 film-compression idiom AND RB's
Medium-tier RAS idiom with NO new engine code, only new constants/curves. It ALSO unlocks the
two cases that have been waiting on a second compression/dispatch position: 3D (SL-005 ±3% at
LB/DT vs ±5%) and 3J (EDGE classification routing). CONFIRM the sequence with Christopher.

== WHAT B5b-LB BUILDS ==
- A new LB rubric in the EXISTING `internal/engine/l4/defense` package (lb.go), mirroring dt.go's
  structure (defense pkg) but with NO SL-021 cushion and NO SL-019. `func NewLB() *LB` +
  `Apply(engine.Layer4Input) engine.Layer4Output`. Use the shared `curve` helpers.
- LB mechanics from LB_Rubric.md (READ IT — triage every number vs the literal spec):
  - **Layer 4 RAS Tier: MEDIUM** (§1, SL-004). cap **±4%** (Medium standard — NOT elevated by
    SL-005), `RAS_steepness` **11.0**, inflection 0.50, raw RAS / 10. RAS_position_weight follows
    the **Medium-tier SL-018 schedule 0.60 / 0.30 / 0.06** — v1.0 ships the ROOKIE weight **0.60
    ONLY** (same deferral as every prior position; RB was the first 0.60-rookie position). NOTE:
    LB RAS is Medium ±4% like RB, but at steepness 11.0 (RB was 8.0) — verify against §3.
  - **Film: SL-005 COMPRESSED** (§2) — cap **±3%** (NOT ±5%), `film_steepness` **10.0** (NOT
    12.0), `film_position_weight` **1.00** (compression expressed via cap+steepness only, no
    triple-compression). This is IDENTICAL to DT's film block (dtFilmCap 0.03 / dtFilmSteepness
    10.0) — reuse that pattern. Data-Parity neutral (IDP film weights UNSET).
  - **SL-019: EXCLUDED** (§1/§4, SL-OQ-027 gating rule from DE §7). LB is Medium-tier AND its
    longevity is scheme-driven — NO RAS→longevity signal. Breakout sub-signals use BASE curves
    (no curve.SL019 calls). The L3 buffer stays SL-018 standard 0.10×RAS_norm (L3 — not built).
  - **Breakout** (§4): cap ±5% (standard), steepness 11.0, inflection 0.50. Weights
    **0.25 / 0.20 / 0.40 / 0.15** (breakout age / school / College Production Share / age traj).
    College Production Share elevated to **0.40** (highest of any position — tackle+sack+TFL
    market share; richness compensates for thin LB pre-draft scouting). Breakout age dropped to
    0.25 (LBs break out later — junior emergence is normative).
  - **Breakout Age** base curve (§4): ≤20.0→1.00, 21.0→0.75, 22.0→0.45, ≥23.0→0.15 (DT-shaped).
  - **College Production Share** (§4): ≤10%→0.15, 18%→0.55, [top anchor — READ §4 for the ≥X→1.00].
  - **Age Trajectory** base curve (§4): peak 29 — ≤25→1.00 (READ §4 for the ≤ side), 26→0.85,
    27→0.70, 28→0.55, 29(peak)→0.50, 30→0.35, 31→0.20, 32→0.10, ≥33→0.00.
    composition.peakLimit(PosLB)=29 (already correct — confirm).
  - **School Tier**: TEMPLATE (P4 1.00 / G5 0.70 / FCS 0.40 / Non-FCS 0.10) — boundary default,
    NO LB branch (schoolTierNorm is already position-aware; verify LB uses the template path).

== THE TWO CASES LB UNLOCKS ==
1. **Case 3D — SL-005 film compression ±3% at LB/DT vs ±5% elsewhere.** Currently
   `gatedPending("3D", … , PosLB, PosDT, PosWR)`. With LB registered alongside DT+WR, convert it
   to a real `eval3D`: assert that an elite film composite at LB AND DT lands inside ±3% (≤1.03)
   and STRICTLY BELOW where the same composite lands at WR's ±5% (the registered WR rubric is the
   ±5% control). The FilmRaw hook is already on Layer4Output. Registering LB flips 3D PENDING→PASS.
   (dt_test.go TestDTFilmCompression is the unit-level template; eval3D asserts it cross-position
   through the registered rubrics.)
2. **Case 3J — EDGE classification routing (pass_rush_snap_share → DE vs LB).** This was DEFERRED
   from B5b-DE precisely because LB did not exist to route to. It is a **position-resolution /
   dispatch** concern (NOT a rubric): per OQ-004 / SL-OQ-030, a pass-rush-primary defender routes
   to the DE rubric regardless of MFL tag (DE/EDGE/3-4 OLB); a coverage/run-stop off-ball LB
   routes to LB. **GATE-CHECK with Christopher (the real design call this session):** build the
   dispatch now (both DE+LB exist), or defer 3J again to a dedicated dispatch/ingestion session.
   The DE handoff's recommendation was "build it when LB exists" — that is now. If built, it lives
   at the composition/harness boundary (a role-classification field on PlayerSpec or a resolver),
   NOT in either pure rubric. Recommend SCOPING it this session since deferring twice leaves a
   permanent PENDING with both positions present.

== THE HARNESS ==
- Register LB in `harness_app.rubrics()` AND `realrubric_test.go realRegistry()`.
- Convert `gatedPending("3D")` → real `eval3D` (the SL-005 cross-position assertion). Mirror the
  structure of eval3F/eval3M. Add TestRealLBRegistryFlips3D.
- Add LB fixtures to fixtures.go (LB Alpha / LB Bravo contrast — Medium RAS so the separation is
  driven more by breakout/college-share than by RAS; LB is the scheme-dependent position) +
  TestRealLBRankingDifferentiates.
- If 3J is scoped: convert `subSignalPending("3J")` → a real eval over the dispatch resolver.
- NOTE the suite count stays 13 (3D and 3J already exist — they flip, no new case added). The
  `validation_test.go` empty-registry tallies (13 cases, 1 pass, 12 pending) DO NOT change.
- WATCH: TestRealDTRegistryFlips3F asserts 3D stays PENDING ("co-gate LB absent"). Registering LB
  flips 3D → that assertion goes stale. UPDATE it (same pattern as the B5b-DE update to the TE
  test). Also TestRealWRRegistryFlips3AAnd3K asserts 3D stays PENDING — UPDATE that too. grep for
  every test asserting 3D PENDING before you commit.

== GATE CHECK (confirm with Christopher before writing code) ==
1. Sequence: LB next (recommended — last defense-pkg rubric before NGS; unlocks 3D + 3J) vs CB/S.
2. LB combines existing mechanics: SL-005 film (DT pattern) + Medium-tier RAS (RB pattern, but
   steepness 11.0) + NO SL-019. Confirm.
3. Harness: convert `gatedPending("3D")` → real `eval3D` (SL-005 cross-position). Confirm.
4. **Case 3J EDGE dispatch: scope it now (recommended — both DE+LB exist) vs defer again.**
5. RAS ships Medium rookie weight 0.60 only; L3 standard buffer (0.10×RAS_norm) deferred to L3.

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch session/b5b-lb. Never git --no-verify.
- CT105: export PATH=$PATH:/usr/local/go/bin; go build ./...; GOMEMLIMIT=1500MiB GOGC=20 make
  lint (repo ROOT); go test -race ./... . Beelink GUI: wails dev -tags webkit2_41.
- Files < 400 lines (AD-17). cases_eval.go is at 356 — if eval3D/eval3J push cases_eval.go over
  400, split again (the cases.go/cases_eval.go split from B5b-DE is the template). Engine stays
  PURE — defense pkg imports only engine + curve; depguard engine-is-pure must stay green (PROVE
  with a planted USED store import — a blank `_` import trips revive first and masks depguard;
  import + REFERENCE a real exported symbol, e.g. `var _ = params.ValueType("")`).
- Confidence scores INTERNAL — never on Layer4Output/Result (FilmRaw is debug-only, allowed).
- Lint gotcha: never name a param/var `cap` (revive redefines-builtin) — use `capBand`.
- Every custom mechanic proven by a planted failure (M3); shared logic reused, not copied (M17 —
  LB must REUSE the DT film-compression + curve helpers, not re-implement).
- Review gate: GLM 5.2 (OpenCode on bird), BLIND, output is LEADS — triage every finding vs
  source. INLINE a self-contained prompt (GLM hangs on repo exploration) AND inline the
  composition/playerspec/defaults BOUNDARY files too (the standing lesson — every false positive
  GLM has produced came from inlining a definition but not its consumer; pin MAGNITUDE through the
  consumer Apply, not just the helper — the B5b-DE lead). Run headless:
  opencode run -m zai/glm-5.2 "$(cat /tmp/prompt.md)". bird: ssh -i /root/.ssh/bird
  x@192.168.1.195. The bird scp/ssh has a standing allow rule in /root/.claude/settings.json
  (added B5b-TE) — the data-exfil classifier no longer hard-blocks the dispatch. GLM track
  record: DE 0 hard (coverage leads only) / TE 0 hard / WR 1 / RB 1 / DT 3 / QB 2 / B5a 2 —
  earns its keep, still triage.

== CARRIED FORWARD — leads, not blockers ==
- SL-018 receding RAS position-weight schedules ship the ROOKIE weight only across ALL positions
  (LB = 0.60) — wire the schedule when an NFL-career-stage input lands.
- L3 RAS-buffers: WR/RB/LB standard (0.10×RAS_norm), TE + DE amplified (0.30×RAS_norm) are
  Layer-3 age-pull mechanics, NOT built — build at the L3 / data-wiring session.
- Offense/IDP film weights UNSET (calibration pass) — film stays Data-Parity neutral everywhere.
- 3G (DT PFFAlpha assertion-wiring) still gatedPending though DT+DE both registered — wire its
  assertion when a session needs it (PFFAlpha hook exists on DT).
- Breakout three-zone classification (Elite/Average/Late) NOT surfaced on Layer4Output — no
  consumer yet.
- SL-OQ-029 (DE) + SL-OQ-032/CAL-025 (LB) college-data source questions pend the pipeline session.
- The scouting [0,1]-normalized schema vs "engine normalizes from raw" reconciliation still pends
  the real-data-wiring session.
- After LB: CB + S are the NGS-anchor positions (case 3I — NGS present only at CB & S). K stays an
  identity PLACEHOLDER until B5b-K (DECISION-011, row 23).

== CLOSE GATE FOR THIS SESSION ==
- go build + make lint 0 + go test -race green; engine depguard green (planted USED import).
- LB constants/curves verified vs LB_Rubric §2/§3/§4 (incl. a worked-example end-to-end if §5
  provides one — the Warner case); LB reuses the DT film-compression idiom + curve helpers (no
  re-implementation — confirm by reading lb.go vs dt.go).
- Case 3D flips PENDING→PASS (SL-005 ±3% at LB+DT vs WR ±5%); every test asserting 3D PENDING
  updated. If 3J scoped, it flips too. Module 1 separates LB Alpha from LB Bravo.
- GLM 5.2 BLIND review (sliced, inlined prompts incl. the boundary files; pin magnitude through
  Apply); triage vs source.
- Functional gate: Christopher operates the harness — sees 3D (and 3J if scoped) flip and LBs
  differentiate. Squash-merge after he confirms. Then write handoff 22 (B5b-CB — first NGS
  anchor; confirm sequencing CB vs S).
