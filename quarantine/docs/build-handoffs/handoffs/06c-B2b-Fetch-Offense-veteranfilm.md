HANDOFF — Session 6 (cont. 2): B2b-Fetch-Offense — Veteran-Film + residual offense fetchers
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- Mid-module. B2b-Fetch-Offense on branch session/b2b-fetch-offense (NOT merged, 8 commits ahead of main).
  Done + reviewed + LIVE-verified on CT105 this arc:
    · crosswalk (MFL->gsis foundation)                 7f9dd32  (7979 / 7783 entries)
    · NFLProduction + extcsv.go seam                    990c0ae  (607 REG)
    · byte-cap M3 gate-proof                            0d4372d
    · shared IntCell/FloatCell extraction (seam)        eac431f  ← SEAM CHANGE, see below
    · TouchShare (RB, snap_counts)                      bc842ed  (637)
    · AgeTrajectory (players.csv birth_date)            94e3d44  (24961)
    · RAS (combine.csv measurables)                     bb15e01  (4832)
  All: GOMEMLIMIT=1500MiB GOGC=20 make lint = 0, go test -race ./... green, env-gated live PASS.
- Working tree: clean, pushed. STAY ON session/b2b-fetch-offense (module unfinished; do not cut a new branch).
- THREE of the four nflverse offense fetchers are done. Veteran-Film is the last + hardest. Then CFBD pair, then Madden.

== TWO SEAM CHANGES INHERITED (both flagged for the module-close agy/Gemini re-review) ==
1. ALREADY DONE — ingestion.IntCell / ingestion.FloatCell now live in extcsv.go (missing-is-zero / malformed-
   fails-loud parsing), and nflproduction was refactored onto them. Reason: TouchShare needed the same parsing;
   M17 says extract on the 2nd instance, don't copy-paste. Behaviour unchanged; nflproduction live re-verified 607.
2. TO DO THIS SESSION — the pbp streaming/cap evolution (see Veteran-Film below). This is the deliberate
   template-evolution item carried since 06b. Do it FIRST, prove the new path, then build the join on top.

== THE SEAM YOU INHERIT (clone it; do not re-invent) ==
- internal/ingestion/extcsv.go: FetchCSV(ctx, client, url, maxBytes)->[][]string (status-checked, byte-capped,
  fails loud over cap), CSVColumns(header, names...) (by-NAME, BOM-stripped, case-SENSITIVE fail-loud),
  IsMissing("" or "NA"), IntCell/FloatCell (missing->0, malformed->fail loud), ValidatePlayerID (RISK-003).
- The fetcher shape (see ras/touchshare/agetrajectory): FetchCSV -> CSVColumns -> loop records[1:] ->
  resolve/skip -> typed parse (fail loud) -> dedup guard -> errEmpty guard -> typed map keyed by gsis.
- pfr-keyed sources (snap_counts, combine) take an INJECTED pfrToGSIS map[string]string (built by the assembly
  layer from db_playerids.csv); the live test builds it inline via livePfrToGSIS (copy from ras/touchshare).
  pfr_id format matches between db_playerids.pfr_id and snap_counts.pfr_player_id / combine.pfr_id (verified).

== READ FIRST ==
- docs/data-layer/Offense_Scouting_Source_Map.md §3 (FTN->PBP join, VERIFIED) + §7 (zero-leak ledger).
- internal/ingestion/extcsv.go (the seam you must extend for pbp) + internal/ingestion/ras/fetcher.go
  (the most recent worked example — optional fields, injected map, drop-ambiguous dedup).
- internal/scouting/types.go — OffenseFilm group (QB/RB/WR/TE) is where veteran film ultimately lands.
- docs/scoring-engine/Engine_Specification.md (Layer-4 Approach-A: normalization is the engine's job).

== RECON (Haiku fan-out — run before design/build) ==
- VERIFY live (curl | head -1 | tr ',' '\n') the EXACT column names in ftn_charting_{year}.csv and
  play_by_play_{year}.csv before coding. Source map §3 lists them but agy/Gemini fabricate names — confirm bytes.
  Known-good from §3: FTN has ftn_game_id, nflverse_game_id, ftn_play_id, nflverse_play_id, is_catchable_ball,
  is_contested_ball, is_created_reception, is_drop, is_interception_worthy, is_throw_away, read_thrown.
  PBP has play_id (col 1), game_id (col 2), receiver_player_id, rusher_player_id, passer_player_id (all gsis).
- Confirm the live BYTE SIZE of play_by_play_2024.csv (curl -sIL ... | grep -i content-length) so the cap is set
  from a real number, not a guess. It is hundreds of MB — DefaultMaxCSVBytes (64 MiB) WILL fail-loud by design.

== WHAT THIS SESSION BUILDS ==
1. extcsv.go pbp evolution (DO FIRST — prove before the join).
   · The hundreds-of-MB pbp file cannot use FetchCSV (it csv.ReadAll-buffers the whole body). Add a STREAMING
     variant: e.g. StreamCSV(ctx, client, url, maxBytes, func(header []string, rec []string) error) that reads
     row-by-row (csv.Reader.Read in a loop), still byte-capped via io.LimitedReader, still BOM/by-name binding,
     fails loud on cap. Do NOT just raise the 64MiB cap and ReadAll — that buffers ~hundreds of MB on a 2GB box
     (OOM risk; CT105 already OOMs lint). Stream + accumulate only the per-player aggregates.
   · M3: ship a deliberate-violation test for the new path in the SAME commit (over-cap fails loud; a malformed
     row fails loud). Re-review the seam change (agy/Gemini) at module close.
2. Veteran-Film fetcher — internal/ingestion/veteranfilm/  (gsis-keyed, RAW trait RATES — clean zero-leak).
   · Fetch ftn_charting_{year}.csv (small, FetchCSV is fine) into memory keyed by (nflverse_game_id, nflverse_play_id).
   · STREAM pbp_{year}.csv; for each play, join on play_id==FTN.nflverse_play_id AND game_id==FTN.nflverse_game_id;
     attribute the charted flags to the play's receiver_player_id / rusher_player_id / passer_player_id (gsis).
   · Aggregate per gsis into RATES, not counts: contested-catch %, drop %, created-reception %, etc.
     ("how", never "how much" — a rate is leak-clean; a raw count is volume and risks the identity leak.)
   · RAW per the CONTRACT: do NOT normalize per position and do NOT weight — the engine does Approach-A.
   · Output type holds rates + the denominators (targets/plays charted) so the engine can confidence-gate small
     samples; emit nothing (or a clearly-flagged low-n) for players below a sane charted-play floor — DECISION
     for Christopher at build (floor value, and whether to emit low-n at all).
   · Env-gated live test TWR_LIVE_NFLVERSE=1: assert per-player trait rates resolve for a charted season; spot-
     check a known high-volume WR's contested-catch rate is in [0,1].
3. THEN the residual offense fetchers (after Veteran-Film, or a fresh session):
   · CFBD CollegeProductionShare + SchoolTier — GATED on the CFBD key (in CT105 env CFBD_API_KEY; key minted +
     /teams 200 verified 2026-06-20). CFBD has NO targets -> CollegeProductionShare = reception/yardage share,
     long-format (one row/stat, share = Sigma player / Sigma team). NOT a static CSV — it's an authed JSON API,
     so it does NOT use extcsv.go; new client pattern (header auth). See source map §2/§4.
   · Madden (EA ratings-api) — current-season BLOCKED (all m26 variants 500); historical slug works for
     birthdate; current MaddenFilm = lower-durability fallback. Build after the above. JSON, no-auth, no extcsv.
   · Rookie consensus-rank CSV — the 1 surviving MANUAL input; a manual-import path, NOT a fetcher. Separate slot.

== CONSTRAINTS ACTIVE (do not re-litigate) ==
- THE CONTRACT: scouting fetchers emit RAW, gsis-keyed; the ENGINE normalizes (position-specific, Approach-A).
- ZERO-LEAK (hard): no field references fantasy points / projected volume / MFL scoring. Make it STRUCTURAL.
  For Veteran-Film: emit RATES not counts; never bind pbp's fantasy/EPA/WP columns.
- MISSING POLICY by source type: counting stats missing->0 (IntCell); optional measurables missing->absent/nil
  (ras pattern, *float64); a missing JOIN (a charted play whose player_id is empty) -> skip that play.
- DEDUP by source: a gsis appearing 2+x where it shouldn't -> nflproduction/agetrajectory FAIL LOUD (trusted
  shape); RAS DROPS-ambiguous-and-continues (combine shares pfr across distinct players — Christopher's call,
  lenient external boundary). Pick per source; for Veteran-Film aggregation, many plays per gsis is EXPECTED
  (accumulate), like touchshare.
- Standards: <250-line target / 400 cap (filelen); gofmt -w; columns by name; fail loud; ctx first; inject
  client+url(+map); each fetcher its own subpackage (depguard layer1-no-upward-import for free); shared logic ->
  extcsv.go not copy-paste (M17); each gate proven by a deliberate violation in the same commit (M3).
- CT105 lint OOM: GOMEMLIMIT=1500MiB GOGC=20 make lint (warm cache: go build ./... first). Go 1.26.4 at
  /usr/local/go/bin (prepend PATH). Live tests reach the internet directly from CT105 (no Beelink round-trip).

== CARRIED FROM THIS ARC (the field that compounds — read these) ==
- LIVE DATA CATCHES WHAT FIXTURES CANNOT — run the live test before declaring any fetcher done. RAS's live run
  found combine.csv shares one pfr_id across two distinct players (CB+OLB "Derrick Johnson", gsis 00-0023449);
  drove the drop-ambiguous policy. snap_counts is PER-GAME (aggregate to season). combine ht is "F-I" not inches.
  players.csv has NO age column (durable raw = birth_date). snap_counts/combine key on pfr, NOT gsis.
- SL-OQ-021 is a Layer-1 spec: per-ACTIVE-game, not per-week. TouchShare honors it (GamesActive). Veteran-Film
  trait rates are inherently per-charted-play, which is the right denominator — keep it that way.
- DOC DRIFT TO FIX AT MODULE CLOSE: scouting/types.go Profile.TouchShare comment still says "FantasyPros touch
  share" — it's snap_counts now (Option D eliminated FantasyPros). Correct the comment.
- agy/Gemini review caveats unchanged: fabricates file/column names (triage every name vs bytes); severity
  unreliable both directions (mine substance); Autonomy Mirage (writes to /root/.gemini/.../brain/<uuid>/, not
  the path it claims — fetch the real file).

== CLOSE GATE FOR THIS SESSION ==
- Build: GOMEMLIMIT=1500MiB GOGC=20 make lint = 0 + go test -race ./... green.
- Functional (Christopher / live): the new StreamCSV proven (over-cap fails loud on real pbp); Veteran-Film live
  run yields sane per-player trait rates in [0,1] for a charted season; spot-check one known WR.
- MODULE CLOSE (when all offense fetchers done): zero-leak verified field-by-field across ALL offense fetchers;
  fix the Profile.TouchShare comment; agy/Gemini re-review of BOTH seam changes (IntCell/FloatCell + StreamCSV);
  then write the B2b-Fetch-Defense handoff (NGS receiving = nextgen_stats/ngs_receiving.csv.gz, GZIPPED, the
  CB/S Coverage anchor) and tick Build_Tracker.md.
- Handoff: write the next handoff before clearing.

== ENV / OPS QUICK REF ==
- Branch: session/b2b-fetch-offense (pushed, 8 commits ahead of main, NOT merged).
- Review gate: Gemini 3.1 (agy out). agy re-auth via pct enter 104 on the Proxmox host if it returns.
- CFBD_API_KEY is in CT105 env (not committed). Live tests reach nflverse/CFBD/EA from CT105 at HTTP 200.
