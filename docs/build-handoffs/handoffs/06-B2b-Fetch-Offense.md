HANDOFF — Session 6: B2b-Fetch-Offense — Scouting fetchers for offense sources
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- Just completed: Session 5 (B2b-Schema — Scouting Schema). Build green: yes
  (make lint 0; go test -race ./... green). Branch merged: <confirm
  session/b2b-scouting-schema is squash-merged to main before starting>.
- Working tree: clean on main. The unified scouting shape is LOCKED at
  internal/scouting (Profile + OffenseFilm/IDPFilm/NGSCoverage conditional groups
  + SchoolTier/SafetyRole enums). AD-16 walk passed. It is a leaf (imports only
  playerid); it does NOT fetch and does NOT score.
- This session's branch: session/b2b-fetch-offense (branch from main).

== READ FIRST ==
- internal/scouting/types.go + constants.go (the shape you populate — do NOT change
  it without cause; if a source needs a field that isn't there, flag it, don't
  silently widen the schema).
- docs/scoring-engine/Scouting_Schema.md (the per-position coverage table + zero-leak
  invariant + scale convention [0,1]).
- docs/scoring-engine/{QB,RB,WR,TE}_Rubric.md (the offense sub-signals + weights).
- docs/data-layer/MFL_API_Reference.md + MFL_API_Specification.md (only if a source
  routes through MFL; most scouting sources do NOT — see below).
- internal/ingestion/ (the boundary-helper pattern: MFLList, ValidatePlayerID,
  CheckAPIError — reuse, do not re-implement).

== GATE CHECK (confirm before writing code) ==
- Upstream complete: B2b-Schema (scouting.Profile). Verified: yes.
- Open questions that block this session: none hard. SOURCE ACCESS is the real
  question — most scouting sources (PFF, RSP, Sharp Football, The Draft Network,
  Madden, RAS, college production) are NOT MFL endpoints. Confirm with Christopher
  the ingestion path for each offense source (manual CSV import? scrape? API?)
  BEFORE building a fetcher against an assumed source. Do not invent a data source.
- DECISION-011 (K Madden-majority) does NOT touch this session — K is a later
  Kicker/Archival fetch session, not offense.

== WHAT THIS SESSION BUILDS ==
- Output: the scouting fetcher(s) that populate scouting.Profile for the OFFENSE
  positions (QB/RB/WR/TE): the universal core (PFFGrade, DraftNetwork, MaddenFilm,
  NFLProduction, RAS, BreakoutAge, SchoolTier, CollegeProductionShare, AgeTrajectory)
  plus the offense-conditional group OffenseFilm (RSPQualitative, SharpFootball) and
  RB-only TouchShare.
- Each input is normalized to [0,1] at the fetcher boundary (scale convention).
- Layer: 1 (data population). Fetchers live under internal/ingestion (or a scouting
  subpackage) and import scouting for the output type — they never score.

== CONSTRAINTS ACTIVE THIS SESSION ==
- ZERO-LEAK (hard): no fetched field may encode fantasy points, projected volume,
  MFL scoring config, or format-dependent volume stats. Touch share is OK (it is
  snap/opportunity, not fantasy points) — but verify each source field.
- Set ONLY the offense-relevant fields. Leave IDPFilm / Coverage nil (those are
  defense; B2b-Fetch-Defense owns them). TouchShare is RB only.
- Any MFL-sourced fetch MUST call ingestion.CheckAPIError first (B3 learning: MFL
  returns HTTP 200 with {"error":{...}} on failure — empty-as-valid silently wipes).
- Standards: <250-line target / 400 cap; leaf-ish; gofmt -w before make lint
  (comment alignment trips the gate); on CT105 run GOMEMLIMIT=1500MiB GOGC=20 make lint.
- RUN AGAINST LIVE DATA before declaring done (B3 learning — units alone missed the
  commissioner-created players). If a source isn't live-reachable from CT105, gate
  the live test (TWR_LIVE_*) and verify on the Beelink.

== CARRIED FROM LAST SESSION (B2b-Schema learnings) ==
- The pointer-group design means "position doesn't use this" = nil group, not zero
  fields. Populate OffenseFilm as a non-nil struct for QB/RB/WR/TE; leave it nil only
  if that genuinely shouldn't exist (it shouldn't be nil for any offense position).
- CollegeProductionShare is ONE field; the position-specific definition (QB start %,
  RB touch %, WR/TE target %) is computed in the fetcher, not the schema.
- Gemini is the review gate (agy still out); it found a real money BLOCKER in B3.
  Triage every finding against source — it suggested a ToUpper that would have
  broken MFL's case-specific codes. You can see the code; Gemini can't.

== CLOSE GATE FOR THIS SESSION ==
- Offense fetchers populate scouting.Profile for QB/RB/WR/TE with every offense field
  set and the correct conditional groups; defense/coverage groups left nil.
- Zero-leak verified field-by-field on the fetched data.
- Build green: make lint 0 + go test -race. Live smoke run where reachable.
- Handoff: write Session 7's handoff (B2b-Fetch-Defense) before clearing.
