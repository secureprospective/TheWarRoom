# FILM Calibration — Planning (Thread C)

**Status:** PLANNING-ONLY. No code, no weights written by this document.
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
- **C-1 · Live distribution sampling** (throwaway sampler, like `cmd/defsample` was for 4b, deleted
  after). Pull each signal across the real player population and report per-position distributions
  (p50/p75/p90/p95, spread, correlation with the already-wired signals so we don't double-count).
  Deliverable: a distributions sheet — the evidence Christopher's weight decision rests on.
- **C-2 · Candidate weight schemes.** From the distributions, draft 2–3 candidate blends per position
  group (NOT one blind number): what share of Layer-4 each film sub-signal carries, the NFLProduction
  cap, the FTN-vs-Madden fidelity discount, the rookie-vs-veteran split for offense. Frame as reads,
  with the trade-off each encodes.
- **C-3 · DECISION GATE (Christopher + expert panel).** Because this sets a durable scouting-weight
  convention, it goes through the **independent expert-panel decision gate**
  (`feedback_expert_panel_decision_gate`): same self-contained brief to each panelist, triage answers
  vs each other + source, surface a recommendation → Christopher picks via AskUserQuestion. **This is
  where Christopher is needed.** Nothing is wired before this gate clears.
- **C-4 · Wire ONE group first** (recommend `Coverage`/CB-S — single source, hard boundary, smallest
  blast radius) using the merged S-Phase shape: caller-supplied crosswalk Map + birthdates, assembly
  leaf owns any multi-season loop, engine stays pure, Profile a leaf, 400-line cap / funlen 40 /
  zero-leak. Live-gate it. Then the next group. Each group is its own reviewed+gated increment.

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

FILM = the calibration frontier. Fetchers exist, sources locked, **weights UNSET by design**. Do NOT
clone-and-wire: run C-0→C-3 (reachability → live distributions → candidate schemes → expert-panel +
Christopher gate) FIRST, promote the `GSISForPFR` bridge, then wire `Coverage` (CB/S) as the first
reviewed increment. Everything obeys the locked principles: durability never weights, quality only as a
calibrated fidelity discount, Madden is a promoted regulator, zero-leak absolute.
