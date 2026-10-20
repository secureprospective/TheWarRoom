HANDOFF — Session 6: B2b-Fetch-Offense — Scouting fetchers for offense
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B2b-Schema (Session 5) merged: internal/scouting shape LOCKED.
- 2026-06-19 SOURCE-ACCESS RECON SESSION (no fetcher code written yet): the source-access
  question — flagged as this session's "real gate" — is now RESOLVED with a strategic pivot.
  Branch session/b2b-fetch-offense exists; only DOCS changed (source map, OQ-015, this handoff,
  CLAUDE.md). No internal/ code yet.
- READ THIS FIRST, it changes the build: docs/data-layer/Offense_Scouting_Source_Map.md
  (the complete decision + every verified endpoint + zero-leak ledger + open items).

== THE PIVOT (supersedes the old "manual manifest" plan) ==
DECISION (Option D, owner-confirmed via dual Claude+Gemini vote): ELIMINATE manual entry.
- Veterans → 0 manual. Rookies → exactly 1 manual input/yr (a consensus draft-rank CSV).
- PFF / Matt Waldman RSP / The Draft Network / Sharp Football are ELIMINATED (no clean
  automatable source — verified two sweeps). Do NOT build a manifest importer for them.

== READ FIRST ==
- docs/data-layer/Offense_Scouting_Source_Map.md  ← THE canonical reference for this build
- internal/scouting/types.go + constants.go  (the output shape; do not widen without cause)
- docs/roadmap/Roadmap_and_Open_Questions.md → OQ-015 (resolved → Option D)
- internal/ingestion/ (boundary-helper pattern: MFLList, ValidatePlayerID, CheckAPIError — reuse)
- internal/mfl/ (transport template — NOTE: these new sources are NOT MFL; see below)

== HARD GATE — CLOSE BEFORE WRITING FETCHER CODE ==
1. CFBD API KEY: ask Christopher to mint one at collegefootballdata.com/key (endpoint 401s
   without it; none stored). Then run a live /stats/player/season?category=receiving call to
   settle whether CFBD exposes TARGETS — strong prior is NO, so receiver CollegeProductionShare
   becomes RECEPTION/YARDAGE share, not target share. Confirm before coding the share math.
2. Confirm the Madden decision still holds: current-season EA API is BLOCKED (all m26 variants
   500). Birthdates safe via a historical slug; current MaddenFilm = lower-durability scraper
   fallback. FTN is the PRIMARY veteran film signal, not Madden.

== WHAT THIS SESSION BUILDS (Layer 1 fetchers; import scouting for the type, never score) ==
These sources are NOT MFL — they are external HTTP file/REST sources. They still inherit the
ingestion discipline (fail-loud, shape-validate, [0,1] normalize at the boundary, live-verify).
- CFBD fetcher: /stats/player/season (rushing|receiving) → team-sum aggregation in Go →
  CollegeProductionShare; /teams → SchoolTier. (Bearer key from env, never hard-coded.)
- nflverse fetchers (fetch GitHub release CSVs directly; some are .csv.gz → compress/gzip):
  · player_stats_season_{year}.csv → NFLProduction
  · snap_counts_{year}.csv → TouchShare (RB only)
  · players.csv → NFL-vet birth_date → AgeTrajectory
  · combine.csv → homebrew RAS (per-position z-score in Go; it's a RAS-EQUIVALENT, flag it) +
    prospect DOB seed
  · ftn_charting_{year}.csv + play_by_play_{year}.csv → JOIN on (game_id, play_id), attribute to
    receiver/rusher/passer_player_id, aggregate to per-player film trait rates (VETERAN FILM).
- Madden fetcher: EA ratings-api.ea.com/v2/entities/m{NN}-ratings — use a working HISTORICAL
  slug for plyrBirthdate; current-season ratings via scraper fallback ONLY if needed (lower
  durability). CheckAPIError-equivalent: EA returns {count,docs[]}; treat empty docs as a glitch.
- Rookie consensus-rank: a MANUAL CSV import path (Rank,Player,Position,College) normalized
  [0,1] within the rookie class. This is the ONE manual surface. Keep it tiny and well-documented.

== CONSTRAINTS ACTIVE (zero-leak ledger in the source map §7) ==
- FTN verified clean (column-level). Raw NFLProduction/TouchShare are VOLUME — cap/watch, don't
  let them dominate a scouting rank. REJECT FantasyPros ecrData (fantasy leak) and CFBD PPA
  (points leak — use success rate / havoc for college "how").
- Set only offense-relevant scouting.Profile fields; leave IDPFilm/Coverage nil. TouchShare RB only.
- Standards: <250-line target / 400 cap; gofmt -w; on CT105 GOMEMLIMIT=1500MiB GOGC=20 make lint.
- RUN AGAINST LIVE DATA before declaring done.

== DO NOT DO IN THIS SESSION (it's a separate, deliberate job) ==
- Do NOT set new rubric weights. The Film component must be redesigned (eliminated sources were
  100% of QB/WR/TE Film) — but reweighting is a CALIBRATION-against-data pass, not a blind number.
  Two locked principles: durability NEVER influences a weight; quality only as a fidelity discount
  via calibration. The fetchers populate the data; the recalibration is its own session/CAL pass.

== CLOSE GATE ==
- Offense scouting.Profile populated from the automated sources for QB/RB/WR/TE; defense groups nil.
- Zero-leak verified field-by-field on fetched data. Live smoke run where reachable.
- Build green: make lint 0 + go test -race. Gemini review (agy out).
- Handoff: write Session 7 (B2b-Fetch-Defense) — note NGS receiving = ngs_receiving.csv.gz
  (gzipped) is the CB/S Coverage anchor.
