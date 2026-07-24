# Handoff 51 — Case 3G wired (α-schedule-only), PFF retired, pass-rush weight deferred to panel

**From:** the 2026-07-24 session (pfrpassrush → IDP film calibration).
**For:** the next build session.
**Type:** STATE MARKER. Module 3 is COMPLETE. No task in-flight; pick from NEXT.

## WHAT WORKS (verified this session — merged to main, HEAD `e149c57`)
Every item below was checked, not assumed:
- **Case 3G is LIVE and PASSES** under the real registry (`TestRealRegistryFlips3G`, uncached `-race`). Module 3 (3A–3M) now has **zero PENDING cases** — 3H was already done (`ebe8287`), 3G is this session.
- **SL-021 α mechanic proven:** DT dynamic α 0.50 (Y1) → 0.10 (Y2+), DE fixed-0.15 control that never switches by year; `SL021Blend` EMA → spec's 0.75 / 0.63 / 0.645 on synthetic inputs; the DE≠0.75 guard fires. (`internal/harness/cases_eval_3g.go`, `internal/engine/l4/defense/sl021.go` + `dt.go`/`de.go`.)
- **`make lint` = 0 issues; `go test -race ./...` = EXIT 0** (full suite, uncached).
- **GLM 5.2 review gate CLEARED** (blind, verified `glm-5.2`): H1 evidence-integrity fix applied + re-verified; L3/L4 clarity applied; purity / no-production-weight / zero-leak / EMA-math / type-assert-safety / test-gating all confirmed clean.
- **C-1 evidence reproduced LIVE** (CT105 reaches nflverse directly): DT r≈0.75, DE r≈0.82, LB r≈0.47 vs the locked Madden IDP composite — identical before and after the H1 fix.
- **PFF fully retired in code:** `DT.PFFAlpha`→`DT.SL021Alpha`; remaining "PFF" strings are historical prose only.

## WHAT IS DELIBERATELY NOT DONE (deferred, not dropped)
- **The live DT/DE pass-rush WEIGHT is NOT wired into scoring — by decision.** `pfrpassrush` (the fetcher, merged earlier `b6b7899`) computes raw pass-rush counts but feeds NOTHING in the scoring path. Routing decision (Christopher, this session): feed the SL-021 EMA only; **leave the locked Madden IDP film budget untouched** (0.95·Madden + 0.05·neutral). The α mechanic is proven; the *weight* the EMA output would carry is an unallocated, panel-gated knob.
- **Why deferred:** C-1 shows the pressure composite is largely redundant with Madden exactly where SL-021 lives (DT/DE, r 0.75–0.82). A heavy DT/DE weight would double-count. Evidence sheet: `docs/data-layer/PassRush_C1_Distributions.md`.

## NEXT (nothing gated on Module 3 any more)
- **Live pass-rush weight — EXPERT-PANEL GATE.** Take the C-1 evidence to the panel: options are (a) small-or-zero DT/DE weight, or (b) site the signal at LB (the only additive spot, r≈0.47) under its own calibration. **No blind film weight.** Only after the panel does the EMA get a live `new_observation` and a production weight.
- B-2 UI module restyle · league-calendar B-3 frontend board · M2/M4 refactors (handoffs 43/44) · M2 slice-2.

## CARRIED / HOUSEKEEPING
- **Throwaway C-1 sampler KEPT on main** (`internal/ingestion/pfrpassrush/c1sample_test.go`, opt-in `TWR_C1_SAMPLE=1`, skips in CI) so the panel can re-run it. Delete when the weight decision lands (defsample precedent).
- **Ops flag:** opencode's zai auth is currently broken on the Beelink ("Authentication parameter not received in Header") though the key is valid — the GLM gate ran via the **direct z.ai coding-endpoint curl** path instead. See memory `reference_glm_review_over_ssh`.
- ⚠️ ROTATE the free CFBD key + the z.ai `GLM_API_KEY` at beta (unchanged).
