HANDOFF — Session 46: Scouting S-Phase 4 — BreakoutAge (multi-season derivation)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

Origin: the scouting wire-up arc (handoff 45). S-Phase 0 (RAS), 1 (SchoolTier), 2 (offense
CollegeShare) and 3 (IDP CollegeDefense) are MERGED. Each was a data-integration clone of
the RAS assembly template. BreakoutAge is the FIRST remaining signal that is NOT a clean
clone — it carries a genuine pin-a-v1-method DECISION. **PLANNING-FIRST** (enter plan mode,
bring the method to Christopher BEFORE writing), same as S-Phase 2's collapse rule.

== WHERE WE ARE (main HEAD ae1593a) ==
- WIRED end-to-end (fetch → crosswalk join → assembly → Profile → engine Layer 4): RAS,
  SchoolTier, offense CollegeProductionShare, IDP CollegeProductionShare.
- The assembly template is `internal/scouting/assembly/{ras,schooltier,collegeshare,
  collegedefense}.go` + `m1_app.buildScoutingDirectory` (the merge loops, inside the CFBD
  key block). Clone this shape.
- STILL-DEFERRED-NEUTRAL after BreakoutAge: the FILM components (OffenseFilm + IDPFilm +
  NGSCoverage) and K film — those wait for the decision-gated CALIBRATION pass. Do NOT
  ship blind film weights.

== THE ENGINE SIDE IS ALREADY BUILT (verified) ==
- `engine.ScoutingInput.BreakoutAge float64` (raw age in years) + `HasBreakoutAge bool`
  (types.go:103-104). `composition.PlayerSpec.BreakoutAge/HasBreakoutAge` (playerspec.go
  :49-50) with a finite + non-negative validation guard (playerspec.go:138-142).
- Every breakout-position rubric §4 already normalizes BreakoutAge via its own base curve
  (e.g. CB ≤19.5→1.00 … ≥22.5→0.15; SL-019 RAS modulation applies at some positions, NOT
  at LB/DT). The engine consumes a RAW breakout age in YEARS and maps it — same raw-in /
  normalize-in-engine posture as RAS and CollegeShare.
- So the wiring = build the missing middle only: derive a raw breakout age per rostered
  player, populate `Profile.BreakoutAge` + set the presence flag, and let `applyScouting`
  copy it (extend applyScouting — it currently copies RAS/SchoolTier/CollegeShare; add a
  BreakoutAge block gated on the presence flag). NOTE: `Profile` has `BreakoutAge` but the
  presence flag needs checking — S-Phase 1 added explicit `HasRAS`; confirm whether a
  `HasBreakoutAge` exists on Profile or must be added (mirror the HasCollegeProductionShare
  discipline — 0/low age is a real value, needs its own flag, never a zero sentinel).

== THE DATA PROBLEM (why this is not a clone) ==
"Breakout age" = the player's AGE during his FIRST college season whose production crossed
a breakout threshold. Deriving it needs TWO things the current fetchers don't jointly give:
1. MULTI-SEASON college production. `collegeshare.Fetch` (and `collegedefense.Fetch`) pull
   ONE `year` per call (baseURL + year query param). To find the FIRST breakout season you
   must fetch several seasons and scan earliest-first. Decide: how many seasons back, and
   the loop/aggregation shape (a thin multi-year wrapper over the existing single-year
   Fetch, NOT a fetcher rewrite — keep Layer 1 pure).
2. BIRTH DATE. `agetrajectory.Fetch` returns `RawAge{GSISID, BirthDate}` (gsis-keyed,
   birthdate only — age is deliberately an as-of derivation, zero-leak). Join it via the
   crosswalk gsis bridge, exactly like the other signals.
Then: breakout age = (season reference date − birthdate) in years, for the earliest season
whose production share ≥ the breakout threshold.

== THE v1-METHOD DECISION (bring to Christopher BEFORE writing — expert-panel NOT required,
   this is a provisional v1 like the offense collapse; but it IS a Christopher call) ==
A) BREAKOUT THRESHOLD: what production share (and WHICH share — offense uses the collegeshare
   yardage share; defense the collegedefense averaged share) counts as a "breakout"? Common
   dynasty convention is a position-specific dominator threshold. Propose 2-3 reads; do NOT
   lock a number blind (mirror [[lesson_inference_vs_calibration]]).
B) SEASON REFERENCE DATE: age "during" a season — anchor to a fixed convention (e.g. Sept 1
   of the season year, or the player's age on the season's kickoff). Pick one, document it.
C) HOW FAR BACK / MISSING DATA: how many seasons to scan; what if no season crosses the
   threshold (→ absent, neutral — the safe default) or birthdate is missing (→ absent).
D) OFFENSE vs DEFENSE: breakout applies at BOTH (the rubrics define it for skill positions
   AND IDP). Confirm whether v1 does both signals or offense-first then IDP-clone.

== SOURCE GATES ==
- CFBD key still required (multi-season collegeshare/collegedefense) — same env `CFBD_API_KEY`.
- agetrajectory source = nflverse players.csv (birthdates), reachable from CT105 (no key).
- Live-verify the multi-season fetch EARLY (B3 lesson: external-API boundaries surprise you;
  units stay green while the live join finds nobody). Confirm several past seasons return.

== CONSTRAINTS ACTIVE (unchanged) ==
- Layer law / depguard: assembly imports Layer-1 fetchers + domain/playerid/math only; engine
  stays pure; Profile is a leaf. The multi-year wrapper lives in assembly (or a thin helper),
  NOT as upward imports in the fetcher.
- 400-line file cap; parameterized SQL; no globals; typed IPC; zero-leak (no fantasy/EPA/PPA).
- crosswalk is now fetched 4× per score run; if S-Phase 4 adds a 5th, seriously consider the
  deferred consolidation (share one fetched crosswalk across all assemblers) THIS session —
  the cost is becoming real. Flag it to Christopher as a possible in-scope cleanup.

== EXECUTION MODEL ==
INVERTED gate (handoff 45): GLM 5.2 writes the heavy wiring, Claude head-brain reviews vs
source + owns the method decision + gates. GLM has been DOWN through S-Phase 1/2/3 → if still
down, Claude writes it directly (the derivation is more novel than a clone, so Claude-authored
is defensible, but the METHOD decision is Christopher's regardless).

== CLOSE GATE FOR S-PHASE 4 ==
- Build green: make lint 0 + go test -race ./... green (incl. new assembly + multi-year tests).
- Review (GLM 5.2 blind, leads-not-findings, triage vs source) if GLM is back; else waive with
  a self-review note, same posture as S-Phase 1/2/3.
- Functional check (Beelink live gate): a rostered player with a real early breakout shows a
  non-neutral breakout modulation vs an absent-breakout player at neutral. Reset clone
  before/after. Steps to /root/paste.md.
- Handoff: write S-Phase 5's (the FILM calibration pass framing, OR the next non-film signal).

== CARRIED ==
- Open items: OQ-013 (created→official id ramp), OQ-014 (Money type), harness 3G/3H pending,
  FILM weights UNSET (calibration pass), crosswalk multi-fetch consolidation.
- Memory: [[project_warroom_review_week]], [[project_thewarroom]], [[lesson_inference_vs_calibration]].
