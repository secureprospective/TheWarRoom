HANDOFF — Session 6 (cont. 3): B2b-Fetch-Offense — CollegeProductionShare + Madden + module close
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- Mid-module. B2b-Fetch-Offense on branch session/b2b-fetch-offense (NOT merged, 17 commits ahead of main).
  Done + reviewed + LIVE-verified on CT105 this arc:
    · crosswalk / NFLProduction+extcsv / TouchShare / AgeTrajectory / RAS   (06b — 3 of 4 offense fetchers)
    · StreamCSV row-by-row reader (the seam)            f4eb542  (M3 over-cap + callback-abort proven)
    · Veteran-Film (FTN->PBP join)                      afc90a4  (live 2025: 209 recv / 45 pass; rates in [0,1])
    · SchoolTier (CFBD /teams)                          a19d285  (live: 681 schools, P4=68/G5=66/FCS=129/nonFCS=418)
  All: GOMEMLIMIT=1500MiB GOGC=20 make lint = 0, go test -race ./... green, env-gated live PASS.
- Working tree clean, pushed. STAY ON session/b2b-fetch-offense (module unfinished; do not cut a new branch).
- TWO of the offense scouting fetchers remain: CollegeProductionShare (CFBD) + Madden (EA). Then module close.

== HARD GATE BEFORE CollegeProductionShare CODE — the rookie player-id keying question (UNRESOLVED) ==
- Every fetcher built so far keys on gsis (NFL ids). CollegeProductionShare is a ROOKIE/college signal: those
  players may have NO gsis yet, and the dynastyprocess db_playerids.csv crosswalk does NOT cover college-only
  prospects. CFBD has its own integer playerId + (player name, team, position).
- DECISION NEEDED (Christopher): how does college production attach to a TheWarRoom player?
  Candidate paths to recon BEFORE building: (a) CFBD has a /draft/picks endpoint that may map college player ->
  draft slot -> an nfl id; (b) name+college fuzzy match to the rosters feed (brittle); (c) defer college-share to
  when a rookie-import path exists (the 1 manual consensus-rank CSV already planned may carry the id bridge).
  This is the SAME class of problem the crosswalk solved for veterans, for a population the crosswalk omits.
- Do NOT build CollegeProductionShare until this is decided. It is a better fresh-session opener than a tail task.

== WHAT THIS SESSION BUILDS (once the gate clears) ==
1. CollegeProductionShare — internal/ingestion/collegeshare/ (or similar).
   · CFBD GET /stats/player/season?year=&team=&category=rushing|receiving — AUTHED JSON, bearer, NOT a CSV
     (does NOT use extcsv.go). LONG-FORMAT: one row per stat {playerId, player, position, team, conference,
     category, statType, stat}; a player's share = Σ(their REC or YDS) ÷ Σ(team REC or YDS), pivot across rows.
   · CFBD has NO targets (verified): receiver share = reception/yardage share, not target share.
   · Query is per-team — iterate FBS teams (from the same /teams call SchoolTier uses) and sum within each team.
   · EXTRACT the shared CFBD client NOW (M17 — this is the 2ND CFBD caller). Move schooltier.NewClient (HTTP/2
     DISABLED — CT105 h2 PROTOCOL_ERROR) + the bearer-auth + lenient-JSON-GET into a shared ingestion/cfbd.go,
     and refactor schooltier onto it (mirror how IntCell/FloatCell were extracted to extcsv.go). Re-verify
     schooltier live after the refactor.
   · Key the output per the keying decision above. RAW + the keying-native id per the CONTRACT (engine normalizes).
   · CALLER MUST TrimSpace the CFBD_API_KEY — the CT105 env var carries trailing newlines Go's header
     validation rejects (schooltier live test does this).
2. Madden (EA ratings-api) — internal/ingestion/madden/ (after CollegeShare or a fresh session).
   · Current-season EA API is BLOCKED (all m26 variants 500); historical slug works for birthdate. Current
     MaddenFilm = LOWER-DURABILITY fallback. JSON, no-auth, no extcsv. Build last.

== MODULE CLOSE (when CollegeShare + Madden done) ==
- Zero-leak verified FIELD-BY-FIELD across ALL offense fetchers.
- FIX DOC DRIFT: scouting/types.go — Profile.TouchShare comment still says "FantasyPros touch share" (it's
  snap_counts / Option D now); AND the OffenseFilm group still names the ELIMINATED RSP/Sharp sources (the
  Film-component redesign is a separate calibration job, but the comment/field set is stale — flag or fix).
- agy/Gemini RE-REVIEW of the seam changes this module: IntCell/FloatCell->extcsv (06b), StreamCSV (this arc),
  and the new shared ingestion/cfbd.go (when extracted).
- The rubric FILM REWEIGHT is a separate CALIBRATION pass — durability never weights; quality = fidelity discount.
  Do NOT set film weights here. Veteran-Film floors (30 targets / 100 attempts) are PROVISIONAL params, a
  calibration decision, not constants.
- Update CLAUDE.md commit-count (it drifted; was "8 commits ahead", real = 17 after this session).
- Write the B2b-Fetch-Defense handoff: NGS receiving = nextgen_stats/ngs_receiving.csv.gz (GZIPPED — needs a
  gzip step the current extcsv path lacks), the CB/S Coverage anchor. Tick Build_Tracker.md.

== SEAM / PATTERN NOTES (clone, do not re-invent) ==
- extcsv.go: FetchCSV (buffered, byte-capped) + StreamCSV (row-by-row, for the >64MiB pbp) + CSVColumns +
  IsMissing + IntCell/FloatCell. CSV sources only.
- schooltier.NewClient: the CFBD HTTP/1.1 (h2-disabled) bearer client — EXTRACT to ingestion/cfbd.go at the 2nd
  CFBD caller (see above). Lenient JSON decode for external 3rd-party (no DisallowUnknownFields).
- Fetcher shape: ctx first; inject client+url(+key/map); own subpackage (depguard layer1-no-upward-import for
  free); fail loud; columns/fields by name; each gate proven by a deliberate violation in the same commit (M3);
  shared logic extracted not copy-pasted (M17). SchoolTier is the one fetcher that imports internal/scouting and
  emits a FINAL value (tier is position-INDEPENDENT) — that exception is documented in its package doc; the
  gsis-keyed RAW rule still holds for every position-specific signal.

== CONSTRAINTS ACTIVE (do not re-litigate) ==
- THE CONTRACT: scouting fetchers emit RAW, gsis-keyed; the ENGINE normalizes (Approach A). (SchoolTier exception
  above — position-independent.)
- ZERO-LEAK (hard, structural): no field references fantasy points / projected volume / MFL scoring. CFBD: use
  raw counting/share, NOT PPA (points-based predicted-points = leak risk; success-rate/havoc are clean).
- Source decisions LOCKED (Option D, 2026-06-19): PFF/RSP/DraftNetwork/Sharp ELIMINATED; do not rebuild a manual
  manifest. Pac-12 year-aware (P4 <=2023), Notre Dame=P4 (SchoolTier, Christopher 2026-06-21).
- CT105: GOMEMLIMIT=1500MiB GOGC=20 make lint (warm cache: go build ./... first). Go 1.26.4 at /usr/local/go/bin.
  Live tests reach nflverse/CFBD/EA directly from CT105 (CFBD needs the key, h2 disabled). CT105 link to GitHub
  is ~310 KB/s — the pbp live test legitimately takes ~4.5 min (timeout set to 12 min).
- Review gate: Gemini 3.1 (agy out). agy re-auth via pct enter 104 if it returns.

== CLOSE GATE FOR ANY SESSION ==
- Build: GOMEMLIMIT=1500MiB GOGC=20 make lint = 0 + go test -race ./... green + env-gated live PASS.
- Functional (Christopher / live): the new fetcher's live run yields sane data; a deliberate violation proves each
  new gate. Write the next handoff before clearing.
