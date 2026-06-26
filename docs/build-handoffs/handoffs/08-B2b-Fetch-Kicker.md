HANDOFF — Session 8: B2b-Fetch-Kicker/Archival — K sources (closes the B2b-Fetch arc)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B2b-Fetch-Offense COMPLETE + merged (main). 9 offense fetchers.
- B2b-Fetch-Defense COMPLETE + merged to main `677102a` (2026-06-26). Every defensive
  scouting input is fed; no defensive fetcher remains. New this module:
    · gzip seam: extcsv FetchCSVGz / StreamCSVGz (shared openCappedCSV; cap bounds the
      DECOMPRESSED bytes; a non-gzip body FAILS LOUD at gunzip).
    · pfrcoverage — CB/S Coverage anchor. nflverse has NO defender NGS file (recon swept
      all 25 release tags); REBOUND (Christopher) onto PFR advanced defense
      (advstats_season_def.csv.gz): targets/cmp%/yds allowed, passer rating allowed.
      Traded-player dedup prefers PFR's [0-9]TM aggregate row. Live 775/2024.
    · collegedefense — defensive CollegeProductionShare. CFBD defensive+interceptions
      categories; emits per-component within-team market shares (tackle/sack/TFL/PD/INT)
      RAW; engine combines per position (CB=PD+INT, S=INT+Tackle, LB=Tackle+Sack+TFL,
      DT/DE=TFL+Sack). Live 665/2023, all shares in [0,1].
    · DECISION: IDP film ELIMINATED (Option-D parallel) — no clean source for any IDP
      film sub-signal; redesigned on Madden defense sub-attrs (already in m24) +
      NFLProduction + pfrcoverage. NO new fetcher; weights UNSET (calibration).
  Build gate every commit: GOMEMLIMIT=1500MiB GOGC=20 make lint = 0, go test -race ./...
  green, env-gated live PASS (TWR_LIVE_NFLVERSE=1 / TWR_LIVE_CFBD=1).
- THIS SESSION cuts a fresh branch off main: git checkout main && git pull &&
  git checkout -b session/b2b-fetch-kicker. Confirm scope with Christopher first.

== WHAT THIS SESSION IS (Build_Tracker row 8) ==
B2b-Fetch-Kicker/Archival — K sources + Madden archival-only (SL-OQ-042). This closes
the entire B2b-Fetch arc (offense + defense + kicker). Scope is SMALL — K is the
simplest position.

== K LAYER-4 CONTEXT (do not re-litigate — locked) ==
- DECISION-011: K Layer-4 is Madden-driven, Madden 0.60 / NFLProduction 0.40. This
  reverses K's structural exclusion + AD-10 + SL-020-at-K (mechanics rewrite is carried
  to B5b-K, NOT this session). For the FETCHER session this means K's two inputs are:
    1. Madden K ratings — ALREADY FETCHED. The madden fetcher reads every `_rating`
       sub-attr into RawMaddenRating.Attributes (kick power/accuracy live in m24). K may
       need NO new Madden code — VERIFY the K attributes are present in the m24 pull
       before writing anything (kickPower, kickAccuracy).
    2. NFLProduction for K — nflverse player_stats carries kicking columns (FGM/FGA, by
       distance bucket, XPM/XPA). CONFIRM whether the existing nflproduction fetcher
       already surfaces (or can surface) the kicking columns, or whether K needs a small
       kicking-stats fetcher. RECON player_stats_season columns live FIRST.
- SL-OQ-042 (Madden archival-only): recon what this open question requires for K
  archival. Read it in docs/roadmap/Roadmap_and_Open_Questions.md before scoping.

== RECON PHASE (standing build pattern — do this first) ==
1. Read docs/roadmap/Roadmap_and_Open_Questions.md → SL-OQ-042 and any K OQs.
2. Read K_Rubric.md §Layer-4 (Madden 0.60 / NFLProduction 0.40 under DECISION-011).
3. LIVE on CT105: confirm m24 carries K rating attrs (kickPower/kickAccuracy) in
   RawMaddenRating.Attributes; confirm nflverse player_stats_season kicking columns.
   Verify load-bearing claims against source before building (Gemini's confident-but-
   wrong rate is high; agy is the review gate when it returns, else Gemini 3.1).

== SHARED SEAMS AVAILABLE (clone, do not re-invent) ==
- ingestion/extcsv.go — FetchCSV / FetchCSVGz / StreamCSV / StreamCSVGz / CSVColumns /
  IsMissing / IntCell / FloatCell. (Gz variants new this module.)
- ingestion/cfbd.go — NewCFBDClient (h2 disabled) + GetCFBD (bearer, byte-capped).
- ingestion/crosswalk — Map.Lookup (MFL->gsis), Map.GSISForESPN (espn->gsis).
- THE INJECTED-RESOLVER PATTERN (collegeshare/collegedefense/madden/pfrcoverage) — for
  any source that keys on something other than gsis.
- Fetcher shape: ctx first; inject client+url(+key/resolver); own subpackage; fail loud;
  by-name binding; RAW + gsis-keyed (THE CONTRACT — engine normalizes, Approach A);
  every gate proven by a deliberate violation in the same commit (M3); shared logic
  extracted not copy-pasted (M17).

== MODULE-CLOSE ITEMS (the B2b-Fetch arc closes after THIS session — do them here) ==
- pfr->gsis bridge promotion (M17): now 3 consumers (touchshare, ras, pfrcoverage). The
  bridge lives only in live-test helpers (livePfrToGSIS). Promote a GSISForPFR into
  crosswalk.Map (one db_playerids fetch, N maps) and refactor the three onto it.
- Shared CFBD long-format helpers (M17): collegeshare + collegedefense duplicate
  statRow/fetchCategory/parsers/poison-drop-emit. Extract into shared ingestion.
- agy/Gemini RE-REVIEW of all arc seam changes (agy was out): IntCell/FloatCell->extcsv,
  StreamCSV, cfbd.go, AND the new gzip seam (FetchCSVGz/StreamCSVGz / openCappedCSV).
  Triage every finding against source.
- The rubric FILM REWEIGHT (offense AND defense) is a SEPARATE CALIBRATION pass —
  weights UNSET; durability never weights; quality = fidelity discount. The eliminated-
  source fields (OffenseFilm RSP/Sharp, PFFGrade, DraftNetwork, IDPFilm *) are
  retained-pending-redesign, populated by no fetcher. Do NOT set weights here.
- OQ-013 (placeholder/created gsis -> official gsis reconciliation ramp) — collegeshare
  AND collegedefense emit dynastyprocess PLACEHOLDER gsis for not-yet-played rookies; a
  consistent join key today, reconciled when the NFL assigns the real id.

== CONSTRAINTS ACTIVE (do not re-litigate) ==
- THE CONTRACT: fetchers emit RAW, gsis-keyed; the engine normalizes (Approach A).
- ZERO-LEAK (hard, structural): no field references fantasy points / projected volume /
  MFL scoring. CFBD: raw counting/havoc/success-rate, NEVER PPA.
- NGS/Coverage anchor applies at CB and S ONLY (scouting.NGSCoverage nil elsewhere).
- Do not reopen locked decisions; flag, don't route around.
- CT105: GOMEMLIMIT=1500MiB GOGC=20 make lint (warm cache: go build ./... first).
  Go 1.26.4 at /usr/local/go/bin. CT105 reaches nflverse/CFBD/EA (CFBD needs the key,
  h2 disabled). Beelink is the Wails/GUI dev machine.
- Review gate: Gemini 3.1 (agy out; re-auth via pct enter 104 if it returns).

== CLOSE GATE FOR ANY SESSION ==
- Build: GOMEMLIMIT=1500MiB GOGC=20 make lint = 0 + go test -race ./... green + env-gated
  live PASS.
- Functional (live): the new/confirmed K inputs yield sane data; a deliberate violation
  proves each new gate. Write the next handoff before clearing.
- After this session the B2b-Fetch arc is COMPLETE → next module is B3b (League Rulebook).
