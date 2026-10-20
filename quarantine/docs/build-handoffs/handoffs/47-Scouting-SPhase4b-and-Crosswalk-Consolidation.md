HANDOFF — Session 47: Scouting S-Phase 4b (IDP BreakoutAge) + the Crosswalk Consolidation
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

Origin: the scouting wire-up arc (handoffs 45/46). S-Phase 0 (RAS), 1 (SchoolTier), 2 (offense
CollegeShare), 3 (IDP CollegeDefense) and 4 (OFFENSE BreakoutAge) are all MERGED (main HEAD
`71d6394`). BreakoutAge was the first MULTI-season signal and the first with a genuine
Christopher-decision v1-method (0.20 dominator threshold / Sept 1 reference / 6-season scan /
offense-first). Two clean next threads, either order — Christopher picks.

== WHERE WE ARE (main HEAD 71d6394) ==
- WIRED end-to-end: RAS, SchoolTier, offense CollegeShare, IDP CollegeDefense, OFFENSE BreakoutAge.
- The assembly template is `internal/scouting/assembly/{ras,schooltier,collegeshare,
  collegedefense,breakoutage}.go` + `m1_scouting.go` (the per-signal merge helpers — extracted
  from m1_app.go this phase when it crossed the 400-line cap).
- STILL-DEFERRED-NEUTRAL: IDP BreakoutAge; AgeTrajectory; the FILM components (OffenseFilm +
  IDPFilm + NGSCoverage) and K film — those wait for the decision-gated CALIBRATION pass. Do NOT
  ship blind film weights.

== THREAD A — S-Phase 4b: IDP BreakoutAge (the clone, WITH its own calibrated threshold) ==
BreakoutAge applies at defense too (every IDP §4 rubric defines a breakout-age curve). The wiring
is a near-clone of S-Phase 4 with ONE real decision:
- The offense scan uses `collegeshare` yardage shares vs a 0.20 threshold. Defense must use the
  `collegedefense` feed's AVERAGED component share (the same `collapseCollegeDefense` output:
  CB=mean(PassDef,INT), S=mean(INT,Tackle), LB=mean(Tackle,Sack,TFL), DT/DE=mean(TFL,Sack)).
- THE DECISION (Christopher, planning-first — do NOT lock blind): a defender almost never
  averages 20% across those events, so 0.20 would mark ~nobody a breakout. Propose a
  defense-appropriate threshold (likely position- or share-distribution-specific — sample the
  live 2019-2024 collegedefense shares FIRST and bring 2-3 reads, per [[lesson_inference_vs_calibration]]).
- Shape: a `BuildBreakoutAgeIDP` (or generalize `BuildBreakoutAge` to take a share-selector +
  threshold + the college feed) that scans multi-season `collegedefense.Fetch` the same way. Same
  birthdate join, same Sept 1 reference, same 6-season window, same `Profile.BreakoutAge` slot —
  BUT note the offense and IDP breakout populate DISJOINT positions (offense {WR,TE,RB} vs IDP
  {CB,S,LB,DT,DE}), so the slot never clobbers (same disjointness proof as S-Phase 2 vs 3).
- Presence: `Profile.HasBreakoutAge` already exists — IDP just sets it for defensive positions.

== THREAD B — the CROSSWALK CONSOLIDATION (the cost is now real) ==
The crosswalk (dynastyprocess db_playerids, ~1 MB) is fetched **FIVE times** per Score League:
BuildRAS + BuildSchoolTier(teams, no crosswalk actually — verify) + BuildCollegeShare +
BuildCollegeDefense + BuildBreakoutAge. Each `Build*` fetches its own. Consolidate:
- Fetch the crosswalk ONCE in `m1_scouting.buildScoutingDirectory` (and birthdates once too — S-Phase 4
  already fetches `agetrajectory` inside BuildBreakoutAge; a second breakout thread would fetch it
  twice). Pass the `crosswalk.Map` (and `map[string]agetrajectory.RawAge`) DOWN into each `Build*`.
- This CHANGES the signatures of FOUR already-merged assemblers (BuildRAS/BuildCollegeShare/
  BuildCollegeDefense/BuildBreakoutAge) + BuildSchoolTier if it uses crosswalk. Keep it a pure
  mechanical refactor: same behavior, same tests (adjust fixtures to pass an already-fetched Map),
  fail-loud on the single fetch. This is the moment to do it — a 6th fetch (Thread A) makes it worse.
- Depguard stays legal (Map is a Layer-1 type; assembly already imports crosswalk).

RECOMMENDATION: do Thread B FIRST (mechanical, shrinks every later signal's cost and diff), then
Thread A on the consolidated shape. But if Christopher wants the IDP signal first, A stands alone.

== THREAD C (larger, decision-gated, NOT this session unless named) — FILM CALIBRATION ==
The last unwired scouting group. OffenseFilm/IDPFilm/NGSCoverage sources were all ELIMINATED (see
types.go SOURCE-DRIFT notes); the redesign is around Madden sub-attributes + NFLProduction +
pfrcoverage with weights UNSET. This is a WEIGHTS/calibration decision, not a plumbing clone —
frame it as its own planning-first program (like Versioning), do NOT ship blind weights.

== SOURCE GATES ==
- CFBD key still required for any college feed (multi-season) — env `CFBD_API_KEY` (the free
  zero-risk key is in Christopher's hands; ROTATE at full beta — flagged in memory).
- agetrajectory = nflverse players.csv (birthdates), no key, reachable from CT105.
- Live-verify EARLY (B3 lesson: external-API boundaries surprise you; units stay green while a live
  join finds nobody). For Thread A, sample the live collegedefense share distribution before picking
  a threshold.

== CONSTRAINTS ACTIVE (unchanged) ==
- Layer law / depguard: assembly imports Layer-1 fetchers + domain/playerid/math only; engine pure;
  Profile a leaf. Multi-season loops live in assembly, never as upward imports in the fetcher.
- 400-line file cap (this phase already forced the m1_app→m1_scouting split), funlen ≤40 stmts,
  parameterized SQL, no globals, typed IPC, zero-leak (no fantasy/EPA/PPA).

== EXECUTION MODEL ==
INVERTED gate (handoff 45): GLM 5.2 writes heavy wiring, Claude head-brain reviews vs source + owns
the method decision + gates. GLM has been DOWN through S-Phase 1-4 → if still down, Claude writes
directly. Thread A's threshold decision is Christopher's regardless (planning-first). Thread B is a
mechanical refactor — no method decision, just careful signature surgery + green tests.

== CLOSE GATE ==
- Build green: make lint 0 + go test -race ./... green.
- Review (GLM 5.2 blind, leads-not-findings, triage vs source) if GLM is back; else waive w/ a
  self-review note (same posture as S-Phase 1-4).
- Functional check (Beelink live gate): Thread A — a real early-breakout DEFENDER shows non-neutral
  vs an absent-breakout defender; Thread B — a clean Score League with identical board to pre-refactor
  (behavior-preserving). Reset clone before/after. Steps to /root/paste.md.
- Handoff: write the next thread's (whichever of A/B/C remains, or the FILM framing).

== CARRIED ==
- Open items: OQ-013 (created→official id ramp), OQ-014 (Money type), harness 3G/3H pending, FILM
  weights UNSET (calibration pass), crosswalk multi-fetch consolidation (Thread B here), ROTATE the
  CFBD key at beta.
- Memory: [[project_warroom_review_week]], [[project_thewarroom]], [[lesson_inference_vs_calibration]].
