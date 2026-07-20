# FILM Calibration — Planning (Thread C)

**Status:** CALIBRATION COMPLETE (2026-07-20). C-0→C-3 DONE: reachability verified, live distributions
sampled, expert-panel gate cleared, all 5 weight knobs LOCKED (see §4 C-3). Remaining = C-4 WIRING
(execution, unblocked). Weights are now SET — this is no longer planning-only.
**Created:** 2026-07-20 · Supersedes the Thread-C planning portion of handoff 48.
**Posture:** planning-first, decision-gated, expert-panel candidate. NO blind film weights — ever.
**Prereq facts verified on disk this session** (main `971ae50`+): all four film-feed fetchers exist
(`internal/ingestion/{madden,nflproduction,pfrcoverage,veteranfilm}`); the three source maps
(`docs/data-layer/{Offense,Defense,Kicker}_Scouting_Source_Map.md`) are the calibration ground truth.

---

## 1. What FILM is (and is NOT)

FILM is the **last unwired scouting group** and the only one that is **not a plumbing clone**. The
S-Phase arc (RAS→SchoolTier→CollegeShare→BreakoutAge, offense+IDP) was fetch → crosswalk join →
Profile → a per-position curve the engine already owned. FILM is different: the **fetchers exist and
the sources are verified**, but the **blend WEIGHTS are UNSET on purpose**. Setting them is a
**calibration-against-live-data job (CAL-series)**, not a number to lock blind.

The three `types.go` film groups name ELIMINATED sources (PFF/DraftNetwork/RSP/Sharp/IDPShow/…): all
were subjective/paywalled with no clean Go-reachable feed. The redesign (Option D, already decided +
recorded in the source maps) rebinds each onto data **already fetched**. This planning pass does NOT
re-litigate the source decisions — those are locked. It plans the **calibration** that turns those
fetched signals into engine-normalized Layer-4 sub-signals with defensible weights.

---

## 2. The redesigned FILM signals (sources LOCKED, weights UNSET)

| Group (types.go) | Positions | Redesigned source (fetcher) | Weight status |
|---|---|---|---|
| `OffenseFilm` | QB/RB/WR/TE | **Veterans:** FTN charting (`veteranfilm`, primary) + Madden sub-attrs (fallback). **Rookies:** consensus-rank (MANUAL, the 1 surviving manual input) + Madden-rookie + combine/RAS + CFBD college *advanced* (success rate/havoc, **NOT PPA**). | UNSET |
| `IDPFilm` | DT/DE/LB/CB/S | Madden defense sub-attrs (`madden`, primary) + `nflproduction` + `pfrcoverage` (supporting). No new fetcher. | UNSET |
| `Coverage` (NGSCoverage) | CB/S **only** (hard boundary) | PFR advanced-defense coverage-allowed (`pfrcoverage`) — passer-rating-allowed headline; emitted RAW, engine normalizes + enforces the CB/S boundary. | UNSET |
| `MaddenFilm` (universal) | all + **K majority** | Madden sub-attribute composite. K is the ONE exception with weights ALREADY set (DECISION-011): **Madden 0.60 / NFLProduction 0.40**, mechanics (cap/curve) deferred to B5b-K. | K set; others UNSET |
| `NFLProduction` (universal) | all | nflverse `player_stats_season` (`nflproduction`). **VOLUME signal** — clean of fantasy points but MUST be capped so it can't dominate a *scouting* rank (identity risk). | UNSET |

**Disjointness** mirrors the S-Phase proof: `OffenseFilm` is nil at defense/K, `IDPFilm` nil at
offense/K, `Coverage` nil everywhere but CB/S. Same slot never clobbers.

---

## 3. The locked calibration PRINCIPLES (constrain every weight we pick)

These are already decided in the source maps — they are the rails, not open questions:

1. **Durability NEVER sets a weight.** Uptime/staleness is handled by SQLite caching + stale-data
   preservation + a manual fallback — never by down-weighting a football signal. (Do not down-weight
   Madden because m24 is two seasons stale.)
2. **Quality enters ONLY as a fidelity discount on a like-for-like proxy** (e.g. ~15% for a tracking
   proxy standing in for direct film), and even that is **resolved by calibration against real data
   (CAL-series)**, not by a from-scratch "rank sources by confidence" scheme. "Weight by source
   confidence" is a **rejected category error** — it lets the data pipeline decide the football.
3. **Madden already has an architectural seat** as a *regulator* (α≈0.20 in every blend, mapping
   subjective claims to sub-attrs). Promoting it from regulator to a weighted signal is consistent
   with the existing design, not foreign to it.
4. **Zero-leak is absolute.** NFLProduction/TouchShare are volume-clean but capped; CFBD college =
   success rate/havoc, **never PPA** (points-based → leak). No fantasy/EPA/MFL-scored field binds.

---

## 4. The calibration PROGRAM (the ordered, planning-first sequence)

Same discipline that produced the 0.12 IDP breakout line (25k-season live pull → candidate reads →
Christopher decides, per `lesson_inference_vs_calibration`). Executed as its own program, NOT rushed
into a wiring session:

- **C-0 · Reachability + zero-leak re-verify.** For each of the four fetchers, confirm live 200s +
  the exact columns the redesign names are present + the emitted struct carries no leak field.
  (Most already verified in the source-map recon; re-confirm at build time, don't trust staleness.)
  Deliverable: a one-line PASS per source.
- **C-0 · DONE (2026-07-20).** All four fetchers re-verified live via their existing `TestLive_*`
  reachability tests (the token-efficient re-check): `madden` 200 / 1404 gsis-keyed (m24);
  `nflproduction` 200 / 607 REG records (2024); `pfrcoverage` 200 / 775 coverage records (2024,
  7782 pfr→gsis); `veteranfilm` 200 / 209 receivers + 45 passers above floor (2025 FTN). Zero-leak
  confirmed by struct shape — no PPA/EPA/fantasy field on any emitted record. PASS ×4.
- **C-1 · Live distribution sampling** (throwaway sampler, like `cmd/defsample` was for 4b, deleted
  after). Pull each signal across the real player population and report per-position distributions
  (p50/p75/p90/p95, spread, correlation with the already-wired signals so we don't double-count).
  Deliverable: a distributions sheet — the evidence Christopher's weight decision rests on.
- **C-2 · DONE (2026-07-20) — candidate schemes framed as 5 decision KNOBS** (evidence:
  `docs/data-layer/FILM_C1_Distributions.md`). Each knob carries options + a Claude recommendation
  (a LEAD for the panel, not a locked number). These are the AskUserQuestion + expert-panel inputs:

  - **K1 · Madden composite recipe.** Equal-weight mean of the per-position curated sub-attrs (all
    validated live present). **man/zone coverage MUST be averaged into ONE coverage term** (C-1 #3:
    r≈0.85–0.95 collinear — counting both double-weights coverage). → *Rec: adopt; low controversy.*
  - **K2 · NFLProduction cap + weight.** C-1 #2: volume/role signal, right-skewed, r≈0.6–0.84 with
    Madden (partial double-count). Options: (a) DROP from scouting entirely (Madden already captures it);
    (b) cap at group p75, weight ≤0.05 as a tiebreaker; (c) cap at p90, weight ≤0.10. → *Rec: (b) — keep
    a thin, capped role signal, never primary; leans on the "identity risk" mandate.*
  - **K3 · Offense film composition (FTN vs Madden).** C-1 #6: FTN sign-checks correct but is thin
    (veteran-only, n=38–94 above floor). Options: (a) Madden primary always, FTN as a confirmation
    overlay; (b) FTN primary for players above the charting floor + Madden fallback, with a ~15% FTN
    fidelity discount; (c) fixed blend. → *Rec: (b) — matches the locked source map, but FTN cannot be
    the SOLE veteran driver given n; the discount + fallback protect the low-population tail.*
  - **K4 · Coverage (CB/S) weight.** C-1 #4 (the headline IDP finding): pfr passer-rating-allowed is
    INDEPENDENT of Madden coverage (|r|<0.30) → a genuine additive signal, deserves a real (not token)
    weight. Engine must INVERT (lower rating allowed = better). Options: 0.15 / 0.20 / 0.25 of the CB/S
    Layer-4 film budget. → *Rec: 0.20 — real seat, still Madden-anchored overall.*
  - **K5 · Rookie vs veteran offense split.** Rookies have no NFL film (no snaps) → the locked Option-D
    path: consensus-rank (the 1 surviving manual input) + Madden-rookie + combine/RAS + CFBD college
    advanced (success rate/havoc, NOT PPA). Veterans → K3. → *Rec: keep the manual rookie input; do not
    collapse rookies to box-score (source map §4 "rookie problem").*

  Locked by C-1 evidence (not open): Madden earns/keeps its promoted-regulator seat and can carry the
  film anchor at every position; RAS weight need not change (C-1 #5, complementary not redundant);
  college-vs-film double-count check DEFERRED (structural season-overlap gap — C-1 caveats).
- **C-3 · DONE (2026-07-20) — expert-panel gate CLEARED, all 5 knobs LOCKED.** Independent panel = Claude
  lead + DeepSeek + Gemini (same self-contained brief, `/root/paste.md`); triaged vs each other + the
  source maps; Christopher decided the one split. Panel converged on 4/5; K3 was flipped by BOTH
  panelists independently naming the same risk.

  **LOCKED DECISIONS:**
  - **K1 — Madden composite = equal-weight mean of the curated per-position sub-attrs, man+zone coverage
    AVERAGED into ONE coverage term.** (Unanimous; collinearity r≈0.85–0.95 → averaging kills the implicit
    coverage double-count without information loss.)
  - **K2 — KEEP NFLProduction, capped at group p75, weight ≤0.05 as a pure tiebreaker** (Christopher's
    call, 2026-07-20; DeepSeek+Claude). Gemini dissented (drop entirely for layer purity) — recorded, not
    adopted. Implementation MUST hold the cap hard so volume can never dominate a scouting rank
    (identity-risk mandate); it is a middle-of-distribution tiebreaker only, never primary.
  - **K3 — Madden is the UNIVERSAL offense-film backbone; FTN is a BOUNDED DELTA-OVERLAY** (with the ~15%
    fidelity discount) applied only where the charting floor is met. **REJECTED the FTN-primary approach**
    — both panelists flagged the population regime-switch discontinuity at the charting floor (n≈38–94) as
    the single biggest risk: two near-identical talents must not rank materially apart because one cleared
    the FTN floor. Revisit a fixed blend only if FTN sample depth passes ~200 with breadth.
  - **K4 — Coverage (CB/S only) weight = 0.20** of the CB/S film budget; engine INVERTS (lower passer
    rating allowed = better). (Unanimous; pfr coverage is independent of Madden at CB/S, |r|<0.30 → a real
    additive seat, not a token.)
  - **K5 — keep the multi-input rookie framework** (consensus-rank manual + Madden-rookie + combine/RAS +
    CFBD advanced success-rate/havoc, NOT PPA). **ADDED pre-ship guard (DeepSeek):** before the rookie
    tower ships, check CFBD success-rate/havoc vs Madden-rookie and RAS — if any pair r>0.5, apply a
    non-redundancy cap. The ~0-overlap college-vs-film result is a MEASUREMENT gap, not proof of
    independence — do not treat college production as automatically non-redundant.
- **C-4 · NEXT (unblocked — weights LOCKED at C-3). The remaining work is EXECUTION, not calibration.**
  Do the §5 `GSISForPFR` bridge promotion FIRST (mechanical), then wire ONE group at a time, each its own
  reviewed + live-gated increment, using the merged S-Phase shape (caller-supplied crosswalk Map +
  birthdates, assembly leaf owns any multi-season loop, engine stays pure, Profile a leaf, 400-line cap /
  funlen 40 / zero-leak). Recommended order (smallest blast radius first):
  1. **`Coverage` (CB/S)** — single source (pfrcoverage), hard CB/S boundary, K4=0.20, engine inverts.
     Reuses the new `GSISForPFR` bridge.
  2. **`IDPFilm`** — Madden defense composite (K1) + capped NFLProduction (K2) + pfrcoverage supporting.
  3. **`OffenseFilm`** — Madden backbone + bounded FTN delta-overlay (K3) + capped NFLProduction (K2).
  4. **Rookie tower** — multi-input (K5) with the pre-ship redundancy check (r>0.5 → non-redundancy cap).
  Each increment: automated gates green → GLM review if back else waive w/ note → live Beelink gate.

---

## 5. Prerequisite refactor (do BEFORE C-4, flagged in Defense map §5)

**`pfr→gsis` bridge promotion.** `pfrcoverage` is now the 3rd consumer of an ad-hoc pfr→gsis bridge
that currently lives only in live-test helpers (`livePfrToGSIS`), alongside `touchshare` and `ras`.
Promote a `GSISForPFR` method into `crosswalk.Map` (one `db_playerids` fetch, N maps — mirrors the
`GSISForESPN`/`PFRMap` pattern Thread B already established) and refactor the three consumers onto it.
This is mechanical/behavior-preserving (a Thread-B-style consolidation) and should land before the
Coverage wiring so C-4 reuses the shared bridge instead of re-cloning the helper.

---

## 6. What is NOT in scope here / carried

- **AgeTrajectory** — Profile field vestigial (age derives from `spec.Age`); confirm before any wiring,
  independent of FILM.
- **K film mechanics (B5b-K)** — weights already set (0.60/0.40); only the cap/curve mechanics rewrite
  remains, a separate small module, not part of this multi-source calibration.
- **CAL-series open items** carried from the source maps: offense film-weight recalibration (Madden
  regulator mechanics + FTN fidelity discount), Madden current-season durability OQ (accept
  scraper/historical-only), CAL-032 (K).
- ⚠️ ROTATE the free CFBD key + the z.ai `GLM_API_KEY` at full beta.

---

## 7. One-line summary for the next session

FILM calibration is DONE — C-0→C-3 complete, expert-panel gate cleared, all 5 weight knobs LOCKED (§4
C-3): K1 equal-weight Madden composite w/ man+zone averaged · K2 NFLProduction capped p75 / ≤0.05
tiebreaker · K3 Madden backbone + bounded FTN delta-overlay (NOT FTN-primary) · K4 Coverage CB/S 0.20
(engine inverts) · K5 multi-input rookie + pre-ship redundancy check. Remaining = C-4 EXECUTION: promote
the `GSISForPFR` bridge, then wire Coverage (CB/S) → IDPFilm → OffenseFilm → rookie tower, each a
reviewed + live-gated increment cloning the S-Phase shape. All obeys the locked principles.
