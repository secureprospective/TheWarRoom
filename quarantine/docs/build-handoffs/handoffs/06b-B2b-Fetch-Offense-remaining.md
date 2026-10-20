HANDOFF — Session 6 (cont.): B2b-Fetch-Offense — remaining offense scouting fetchers
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- Mid-module. B2b-Fetch-Offense is PARTIALLY built on branch session/b2b-fetch-offense (NOT merged to main).
  Done + reviewed this session:
    · crosswalk fetcher (foundation leaf)            commit 7f9dd32
    · NFLProduction fetcher + shared extcsv.go seam  commit 990c0ae
    · byte-cap M3 gate-proof + helper tests          commit 0d4372d
  All three: make lint 0 issues, go test -race green, live-verified on CT105.
  Both new fetchers passed an agy First-Instance Template Review (triaged).
- Working tree: clean. Branch pushed to origin (last = 0d4372d). 4 commits ahead of main.
- This session's branch: STAY ON session/b2b-fetch-offense (do not cut a new one; the module is unfinished).

== THE SEAM YOU INHERIT (already built + twice-reviewed — CLONE it, do not re-invent) ==
- internal/ingestion/extcsv.go — the shared external-HTTP-CSV plumbing. Exact surface:
    · ingestion.FetchCSV(ctx, *http.Client, url string, maxBytes int64) ([][]string, error)
        GETs a static CSV (NOT MFL — no mfl.Client), status-checks, returns ALL records (header = [0]).
        FAILS LOUD if the body exceeds maxBytes (no silent truncation). Pass ingestion.DefaultMaxCSVBytes (64 MiB)
        for the small files; see the pbp WARNING below for the large one.
    · ingestion.CSVColumns(header []string, names ...string) (map[string]int, error)
        Binds columns BY NAME (strips a leading BOM), errors naming the first missing column. Case-SENSITIVE
        on purpose (an upstream rename must fail loud, not fuzzy-match — agy suggested case-insensitive; REJECTED).
    · ingestion.IsMissing(s string) bool — true for "" and the literal "NA" (R-export sentinel).
    · ingestion.ValidatePlayerID(raw) — RISK-003 site #1; routes a bare MFL id through playerid.New (zero-pads).
- Pattern to copy (see crosswalk/fetcher.go and nflproduction/fetcher.go):
    Fetch(ctx, client, url) → FetchCSV → CSVColumns(records[0], …) → loop records[1:] → IsMissing skip →
    typed parse (fail loud on malformed, missing→0) → dedup/conflict guard → errEmpty guard → typed result.
- Each fetcher owns ONLY: its column-name consts, its output type, its filter/conflict rules. Plumbing is shared.

== READ FIRST ==
- docs/data-layer/Offense_Scouting_Source_Map.md  ← THE source-of-truth (§2 endpoint table, §3 FTN→PBP join,
  §7 zero-leak ledger). The crosswalk row + "GAP CLOSED" note were added this session.
- internal/ingestion/extcsv.go + internal/ingestion/nflproduction/fetcher.go (the seam + the worked example)
- internal/scouting/types.go  (the Profile these fetchers ultimately populate; MFL-keyed, POSITION-BLIND)
- docs/scoring-engine/Engine_Specification.md  (Layer-4 Approach-A: normalization is POSITION-SPECIFIC = engine's job)
- docs/build-handoffs/handoffs/06-B2b-Fetch-Offense.md  (the original module handoff — Option D source decision)

== RECON (Haiku fan-out — run before design/build) ==
- Spin a Haiku Explore subagent over: the source map §2/§3, the RB rubric (TouchShare role), and the
  combine.csv + ftn_charting + pbp column headers. Ask for: "the exact column names each remaining fetcher
  needs, and any column that references fantasy points / projected volume (zero-leak)."
- VERIFY every column-name claim against the live header before coding (curl the file | head -1 | tr ',' '\n').
  This session burned real cycles because agy/Gemini FABRICATES column + file names — never trust a name
  from a review agent; confirm against the actual bytes. (See [[feedback-agy-fabricates-endpoint-specifics]].)

== GATE CHECK (confirm before writing code) ==
- Upstream complete: crosswalk + NFLProduction + extcsv seam. Verified: yes (live + lint + 2 reviews).
- CFBD API key: STILL OPEN (Christopher to mint at collegefootballdata.com/key). It gates ONLY the CFBD
  fetchers (CollegeProductionShare, SchoolTier) — it does NOT block the four nflverse fetchers below. Build those.
- Madden current-season EA API: BLOCKED (all m26 variants 500). Birthdates safe via a historical slug. Madden
  stays a lower-durability fallback; FTN is the primary veteran-film anchor. No change.

== WHAT THIS SESSION BUILDS (remaining offense fetchers — clone the seam; each is its own subpackage) ==
Order = easiest-first so the seam is exercised before the hard one. Each is Layer 1, gsis-keyed, RAW (see contract).
1. TouchShare (RB) — internal/ingestion/touchshare/
   · Source: nflverse snap_counts_{year}.csv  (live cols incl. offense_snaps, offense_pct, player/pfr_player_id, position).
   · NOTE: snap_counts keys on pfr_player_id, NOT gsis — confirm the id column live; may need the crosswalk or a
     pfr→gsis map. RESOLVE the id key before coding (don't assume gsis). RB-only in the Profile (TouchShare is *float64, RB only).
2. AgeTrajectory — internal/ingestion/agetrajectory/
   · Source: nflverse players.csv  (birth_date col 16, gsis_id col 1 — verified live this session).
   · Emit RAW birth_date / derived age keyed by gsis. Do NOT normalize against a position peak — that is
     position-specific = the engine's job, and the Profile is position-blind. (Same contract as NFLProduction.)
3. RAS — internal/ingestion/ras/
   · Source: nflverse combine.csv  (combine.csv, NOT combine_{year}.csv — verified live; agy 404'd on the wrong name).
   · Emit RAW combine measurables keyed by gsis. The per-position z-score "RAS-equivalent" is POSITION-SPECIFIC
     (engine), so the fetcher provides raw measurables; flag it as a RAS-EQUIVALENT, not Kent Platte's exact number.
4. Veteran-Film (FTN→PBP join) — internal/ingestion/veteranfilm/  ← THE HARD ONE, do last.
   · Sources: ftn_charting_{year}.csv + play_by_play_{year}.csv. Join FTN.nflverse_play_id = PBP.play_id AND
     FTN.nflverse_game_id = PBP.game_id (all four cols verified present live). Attribute each charted play to
     PBP.receiver_player_id / rusher_player_id / passer_player_id; aggregate per gsis into trait RATES
     (contested-catch %, drop %, created-reception %). Clean zero-leak ("how", never "how much").
   · ⚠ pbp is HUNDREDS of MB. ingestion.DefaultMaxCSVBytes (64 MiB) WILL fail-loud on it (by design). Before
     this fetcher: raise the cap for the pbp call AND/OR add a streaming variant to extcsv.go (the current
     FetchCSV buffers the whole file via csv.ReadAll — fine for the small files, not for pbp). This is the
     carried template-evolution item; do it deliberately, prove the new path, re-review the seam change.

== CONSTRAINTS ACTIVE THIS SESSION ==
- THE CONTRACT (decided this session — do not re-litigate): scouting fetchers emit RAW, gsis-keyed records and
  do NOT normalize. [0,1] normalization is the engine's POSITION-SPECIFIC job (Engine_Spec Approach-A); the
  scouting Profile is position-blind, so a fetcher structurally cannot normalize. (This corrects the original
  handoff's generic "[0,1] normalize at the boundary" line for these signals.)
- ZERO-LEAK (hard constraint): no field may reference fantasy points / projected volume / MFL scoring. Make it
  STRUCTURAL (no struct field to hold it) like RawProduction does — never bind fantasy_points / fantasy_points_ppr
  (player_stats cols 51/52), CFBD PPA (points-based), or FantasyPros ecrData. Raw NFLProduction/TouchShare are
  VOLUME — clean of fantasy points but cap/watch downstream so they don't dominate a SCOUTING rank.
- Standards: <250-line target / 400 cap (filelen gate); gofmt -w; columns bound by name; fail loud; ctx first;
  inject client+url (testable). Each fetcher: unit test (fixtures, prove every gate by a deliberate violation — M3)
  + env-gated live test (TWR_LIVE_NFLVERSE=1, opt-in, never in default suite).
- Anti-spaghetti: each fetcher is its own subpackage under internal/ingestion/ (so the layer1-no-upward-import
  depguard rule covers it for free). No fetcher imports another. Shared logic goes in extcsv.go, not copy-paste.
- CT105 lint OOM: run as  GOMEMLIMIT=1500MiB GOGC=20 make lint  (warm cache first with go build ./...).
  Go 1.26.4 is at /usr/local/go/bin (prepend to PATH). CT105 reaches all nflverse sources at HTTP 200.

== CARRIED FROM LAST SESSION ==
- Decisions made:
  · Crosswalk direction is MFL→gsis (the UNIQUE side). gsis→MFL is one-to-many (MFL keeps dup records;
    live: gsis 00-0031320 → mfl 12459 AND 12571). Lookup(mflID) → gsis. Do not flip it.
  · NFLProduction = RAW REG-season offensive box-score counting stats keyed by gsis (RawProduction). RawProduction
    FIELD SET is still REVIEWABLE by Christopher (he may want a narrower/different raw set) — flag at module review.
  · season_type filtered to "REG" (POST would bias toward playoff teams).
- Mistakes / learnings (the field that compounds — read these):
  · nflverse players.csv has NO mfl_id column (gsis/pfr/espn only) — the crosswalk MUST come from dynastyprocess
    db_playerids.csv. Verified live; do not assume an id column exists — curl the header.
  · R-exported CSVs encode missing cells as the literal "NA", not empty. ingestion.IsMissing handles both.
  · A literal UTF-8 BOM is ILLEGAL in Go SOURCE — use the escape "﻿", not a pasted BOM char. (Cost two
    build failures this session; sed -i 's/\xef\xbb\xbf/\\ufeff/g' fixes a pasted one.)
  · Live data caught two bugs fixtures never would (the NA sentinel + the one-to-many direction). RUN THE LIVE
    TEST before declaring a fetcher done — it is mandatory, not optional.
  · A gate isn't real until a deliberate violation fails it (M3): the byte-cap shipped without an over-cap test;
    agy caught it. Every new gate ships with the violation test in the same commit.
  · agy (Gemini) review caveats: (a) it FABRICATES exact file/column names — triage every name against source;
    (b) its severity labels are unreliable in BOTH directions (false BLOCKERs one pass, real finding marked MINOR
    the next) — mine the substance, ignore the severity; (c) Autonomy Mirage — it claims it wrote to the requested
    path but actually writes to /root/.gemini/antigravity-cli/brain/<uuid>/ — fetch the real file, verify it exists.
- Open items carried:
  · CFBD key (Christopher) → unblocks CollegeProductionShare + SchoolTier (separate sub-build).
  · Madden fetcher (EA historical slug for birthdate; current-season blocked) — lower-durability, do after the 4 above.
  · Rookie consensus-rank CSV (the 1 surviving manual input) — separate manual-import path, not a fetcher.
  · extcsv.go cap-raise / streaming for pbp (see Veteran-Film above).
  · Film-component rubric REWEIGHT is a separate CALIBRATION pass — NOT this session. Fetchers populate data only;
    do NOT set rubric weights.

== CLOSE GATE FOR THIS SESSION ==
- Build green: GOMEMLIMIT=1500MiB GOGC=20 make lint (0 issues) + go test -race ./...  ALL green.
- Functional check (Christopher): for each new fetcher, run its env-gated live test (TWR_LIVE_NFLVERSE=1) on
  CT105 and confirm a sane record count vs the real file (e.g. snap_counts has every offensive RB; combine ~320/yr;
  players.csv full DB; FTN→PBP yields per-player trait rates for charted seasons). A fetcher is not done until its
  live run passes.
- Module close (when all offense fetchers done): zero-leak verified field-by-field; consider a final agy review of
  any NEW seam change (the pbp cap/stream); then write the B2b-Fetch-Defense handoff (NGS receiving =
  nextgen_stats/ngs_receiving.csv.gz, GZIPPED, is the CB/S Coverage anchor) and update Build_Tracker.md.
- Handoff: write the next session's handoff before clearing.

== ENV / OPS QUICK REF ==
- Branch: session/b2b-fetch-offense (pushed, 4 commits ahead of main, NOT merged).
- agy (CT104) is authed + quota-clear (re-auth needed 2026-06-20 via pct enter 104 on the Proxmox host — SSH has
  no TTY for its login UI). Review dispatch = direct SSH; sync its clone (git pull) to the latest commit first.
- Live tests reach the internet from CT105 directly (no Beelink round-trip needed for plain HTTPS GET).
