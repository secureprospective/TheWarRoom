HANDOFF — Session 5: B2b-Schema — Scouting Schema (design + lock, all 10 positions)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- Just completed: Session 4 (B3 — MFL Data Normalization). Build green: yes
  (make lint 0; go test -race ./... green). Branch merged: <yes once
  session/b3-mfl-normalization is squash-merged to main — confirm before starting>.
- Working tree: clean on main. Layer 1 now resolves the FULL pipeline against live
  MFL data: players (LEAGUE-scoped) → normalize.NewLookup → rosters → normalize
  .Rosters → []domain.Roster (typed). domain types are LOCKED (AD-18 reviewed
  deliverable). salaryAdjustments fetcher built (dead-cap source, OQ-005).
- This session's branch: session/b2b-scouting-schema (branch from main).
- NOTE: B2b-Schema is a DESIGN + HUMAN-REVIEW-GATE session (AD-16), not a code
  build. Its output is the locked unified scouting schema, walked with Christopher
  position-by-position. It depends on B3 by DESIGN-ORDERING only (AD-13), not code.

== READ FIRST ==
- docs/build-handoffs/Build_Tracker.md (row 5, B2b-Schema)
- docs/build-handoffs/Layer4_PreBuild_Audit.md (Section 1C — every rubric's inputs
  in one place; this is the raw material the scouting schema must cover)
- docs/scoring-engine/<POSITION>_Rubric.md for all 10 positions (the per-position
  scouting inputs the schema must hold)
- /root/.claude/plans/session-3-audit-build-sequencing.md — AD-16 (human review gate
  before this session) and AD-13 (B2b design-ordering rationale)
- docs/scoring-engine/Engine_Specification.md (Layer 2/3 boundaries: NO scoring leak
  — a scouting field must not encode fantasy points / projected volume)

== GATE CHECK (confirm before writing code) ==
- Upstream complete: B1, B3. Verified: yes (B3 live-tested CT105→MFL).
- Open questions that block this session: SL-OQ-035 / SL-OQ-036 (field reservation
  for S — safety-specific scouting inputs) MUST be decided here (AD-16). NOT yet
  resolved — walk them with Christopher at the review gate.
- The B3 carried OQs do NOT block this session: OQ-013 (created→official id ramp,
  refresh/sync layer), OQ-014 (Money type, cap-math layer). Ignore here.
- AD-16 HUMAN REVIEW GATE is mandatory: do not lock the schema without walking every
  position's inputs with Christopher. If he is not available to review: STOP.

== WHAT THIS SESSION BUILDS ==
- Output: the unified scouting schema spec (design doc) covering all 10 positions —
  the single field set every scouting fetcher (B2b-Fetch-*) will populate, with the
  zero-leak boundary enforced (no field references fantasy points / MFL scoring /
  format-dependent volume). Decide SL-OQ-035/036 field reservation for S.
- If code is produced, it is leaf Go types only (e.g. internal/scouting/schema.go,
  a domain-style leaf like internal/domain) — NO fetchers, NO scoring. Most of this
  session is the design + the human review, not code.
- Layer: 1 (data shape). Touches only the scouting schema's own definition.

== CONSTRAINTS ACTIVE THIS SESSION ==
- Standards: <250-line target / 400 cap; if Go types are written they are a leaf
  package importing nothing internal except (at most) playerid; depguard layer rules
  apply.
- Architectural (the landmines):
  * ZERO SCORING LEAK (hard constraint): no scouting field may reference fantasy
    points, projected volume, MFL scoring config, or format-dependent volume stats.
    This is the whole reason scouting is its own schema, not roster fields.
  * NGS Coverage Metrics anchor applies at CB and S ONLY — schema must reserve those
    fields for CB/S and not bleed them to other positions.
  * SL-019 NOT applied at DT (Cushion Guard replaces it) — reflect in field design.
  * Reserve SL-OQ-035/036 fields deliberately (AD-16) — do not invent fields.
- Anti-spaghetti: the scouting schema is a leaf data shape; it does not fetch and it
  does not score. Fetchers (B2b-Fetch-Offense/Defense/Kicker) come in later sessions
  and POPULATE it; the engine CONSUMES it.

== CARRIED FROM LAST SESSION (B3 learnings) ==
- Decisions made (locked, B3):
  * EDGE → DE (OQ-004): MFL labels edge rushers DE; no separate EDGE class. Any
    position-driven scouting design should treat edge rushers as DE.
  * players is a LEAGUE-scoped fetch (www47, L): the global feed omits
    commissioner-created players. Any new league-data fetcher: prefer the league
    host unless you have proven the global feed is complete for your need.
  * domain leaf types are the pattern to mirror for a scouting leaf package.
- Mistakes / learnings (apply these):
  * RUN AGAINST LIVE DATA EARLY. B3's unit tests were green but the live pipeline
    caught the missing commissioner-created players AND would have caught nothing
    if we'd trusted units alone. CT105 CAN reach MFL (HTTP 200) — a live smoke run
    is cheap and worth it before declaring done.
  * MFL returns HTTP 200 with {"error":{"$t":...}} on failure. Any fetcher that
    treats empty as valid MUST call ingestion.CheckAPIError first (added B3), or an
    outage silently reads as "no data". Reuse it.
  * Gemini is a strong review gate (agy still out): it found a real money BLOCKER
    in B3. But triage every finding against source — it suggested a ToUpper that
    would have broken MFL's case-specific aggregate codes; rejected with cause. You
    can see the code; Gemini can't. When in doubt, lean your own read.
  * gofmt comment-alignment trips the lint gate repeatedly — run `gofmt -w` on
    touched files before `make lint`.
- Open items carried (do not block this session): OQ-013 (created→official id
  reconciliation ramp — refresh/sync layer), OQ-014 (Money type / cap-math
  precision — cap-math layer), CheckAPIError rollout to rosters/players/schedule
  (they are sentinel-protected, so cosmetic — clearer error messages only).

== CLOSE GATE FOR THIS SESSION ==
- Schema locked: every one of the 10 positions walked with Christopher (AD-16);
  SL-OQ-035/036 field reservation decided; zero-leak boundary verified field-by-field.
- Build green (if code written): make lint 0 + go test -race on the scouting package.
- Functional check: Christopher confirms the schema covers each rubric's inputs with
  no scoring leak and no missing field — by reviewing the schema against the rubrics.
- Handoff: write Session 6's handoff (B2b-Fetch-Offense) before clearing.
