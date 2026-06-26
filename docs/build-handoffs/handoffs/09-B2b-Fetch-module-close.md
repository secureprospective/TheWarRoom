HANDOFF — Session 9: B2b-Fetch Module Close (M17 refactors + seam re-review) — closes the B2b-Fetch arc
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B2b-Fetch-Offense COMPLETE + merged (main, 520eb0a). 9 offense fetchers.
- B2b-Fetch-Defense COMPLETE + merged (main, 677102a). All defensive inputs fed.
- B2b-Fetch-Kicker COMPLETE on branch session/b2b-fetch-kicker (887fda4, NOT merged):
    · NEW kicking fetcher (internal/ingestion/kicking) — K's Layer-4 NFLProduction
      signal (0.40, DECISION-011). nflverse player_stats_season is offense-only;
      kicking lives in the separate player_stats_kicking_season file. RAW gsis-keyed
      counts only (FG made/att/missed/blocked, FG-by-distance buckets, fg_long, PAT);
      rates unbound; REG-only; structural zero-leak.
    · TRADED KICKERS summed across per-team REG splits (no source aggregate row;
      Christopher's call — raw-count aggregation, distinct from pfrcoverage's
      prefer-aggregate). Live 45 distinct kickers, Greg Joseph sum spot-checked.
    · Madden K (0.60): NO new code — m24 kickPower/kickAccuracy already captured by
      the madden fetcher; recon-verified live.
    · SL-OQ-042 RESOLVED. Doc: docs/data-layer/Kicker_Scouting_Source_Map.md.
  Build gate every commit: GOMEMLIMIT=1500MiB GOGC=20 make lint = 0, go test -race
  ./... green, env-gated live PASS (TWR_LIVE_NFLVERSE=1 / TWR_LIVE_CFBD=1).
- THIS SESSION: decide whether to build on session/b2b-fetch-kicker (so the arc lands
  as one merge) or merge the K fetcher first then cut session/b2b-fetch-module-close.
  Confirm with Christopher. Either way these are the LAST items before the arc closes.

== STATUS (updated 2026-06-26) ==
Items 1 and 2 are DONE on branch session/b2b-fetch-kicker (built on the K fetcher so the
arc lands as one squash-merge):
  - Item 1 (M17 pfr->gsis bridge promotion) — DONE 506c541. crosswalk.Map now carries the
    pfr bridge (PFRMap() defensive copy + LenPFR); addESPN generalized to addBridge (one
    drop-ambiguous policy for espn AND pfr); the three live tests build the crosswalk and
    inject PFRMap() — the duplicated livePfrToGSIS helper is gone. Behavior-preserving
    (PROVEN on current live data): touchshare 637, pfrcoverage 775 unchanged; ras = 4831
    under BOTH old last-write-wins and new drop-ambiguous maps (the prior 4832 was 5-day
    combine drift; the 3 ambiguous pfr ids are not in combine). crosswalk pfr = 7779.
  - Item 2 (M17 shared CFBD long-format helpers) — DONE 354b49c. Extracted CFBDStatRow,
    FetchCFBDCategory, CFBDInt/CFBDFloat, Share, and EmitDropAmbiguous[P,T] into cfbd.go;
    collegeshare/collegedefense refactored onto them. Behavior-preserving (unit tests pin
    parsing/share/drop-ambiguous on fixtures, all pass unchanged); live collegeshare 408
    (was 407) / collegedefense 667 (was 665) = CFBD+crosswalk data drift, not logic.
  Gate green throughout: make lint 0, go test -race ./... green, env-gated live PASS.
Item 3 (Gemini seam re-review) — DONE 2026-06-26. Gemini 3.1 reviewed blind (no file access);
all 6 findings triaged to FALSE POSITIVE against source — each was already correctly
implemented: (1) lr.N over-cap uses the maxBytes+1 sentinel; (2) EmitDropAmbiguous build
returns value structs into map[string]T (no aliasing); (3) gzip path closes the body on
NewReader error; (4) PFRMap returns a defensive copy (unit-tested); (5) pfrIdx>=0 guards the
optional-column access; (6) mergeKicking covers all 13 RawKicking fields. Zero code changes.
ARC CLOSED: squash-merged to main 2026-06-26. Next = B3b (handoff 10-B3b-League-Rulebook.md).

== WHAT THIS SESSION IS ==
The deferred B2b-Fetch arc module-close refactors (Christopher scoped the K session to
the fetcher only). Pure cleanup/consolidation — NO new scouting data, NO new rubric
weights. After this the B2b-Fetch arc is COMPLETE → next module is B3b (League Rulebook).

== THE THREE ITEMS ==

1. M17 — promote the pfr→gsis bridge into crosswalk.Map (GSISForPFR).
   Today the pfr→gsis bridge lives ONLY in live-test helpers (livePfrToGSIS),
   duplicated across THREE consumers: touchshare, ras, pfrcoverage. crosswalk already
   does one db_playerids fetch → N maps (GSISForESPN added that way). Add a GSISForPFR
   the same way (pfr_id is OPTIONAL there — must NOT break the foundation MFL→gsis map;
   ambiguous pfr→2-gsis dropped, per the RAS/espn precedent), then refactor the three
   fetchers' live tests onto crosswalk instead of their local helper. Verify the live
   counts are unchanged after the refactor.

2. M17 — extract the shared CFBD long-format helpers.
   collegeshare AND collegedefense duplicate statRow / fetchCategory / the long-format
   parsers / the poison-drop-emit logic. Extract into shared ingestion (alongside
   cfbd.go). Both fetchers refactored onto it; live re-verify both (collegeshare 407,
   collegedefense 665) are byte-identical after.

3. agy/Gemini RE-REVIEW of the arc seam changes (agy was out all arc; if it has
   returned, re-auth via `pct enter 104` and use it; else Gemini 3.1). Review:
     · extcsv IntCell/FloatCell extraction
     · extcsv StreamCSV
     · ingestion/cfbd.go
     · the gzip seam: FetchCSVGz / StreamCSVGz / openCappedCSV
     · (newly this arc) the kicking fetcher's traded-player sum + the new shared CFBD
       helper from item 2 and the GSISForPFR promotion from item 1
   TRIAGE every finding against source (Gemini's confident-but-wrong rate is high —
   this arc it called a 5-yr-stale repo "updated every cycle", gave a 404 URL twice,
   falsified a Madden fix live 3×). Output = leads, not findings; apply only survivors.

== NOT THIS SESSION (explicitly deferred) ==
- The Film reweight CALIBRATION pass (offense AND defense): weights UNSET; durability
  NEVER weights; quality = fidelity discount, set against live data not blind. The
  eliminated-source fields (OffenseFilm RSP/Sharp, PFFGrade, DraftNetwork, IDPFilm *)
  are retained-pending-redesign, populated by no fetcher. Do NOT set weights here.
- OQ-013 (placeholder/created gsis → official gsis reconciliation ramp) — refresh/sync
  layer; collegeshare/collegedefense emit placeholder gsis for not-yet-played rookies.
- Profile.TouchShare comment drift (cosmetic — "FantasyPros" → snap_counts); fix
  opportunistically if touching scouting/types.go.

== SHARED SEAMS (clone, do not re-invent) ==
- ingestion/extcsv.go — FetchCSV / FetchCSVGz / StreamCSV / StreamCSVGz / CSVColumns /
  IsMissing / IntCell / FloatCell.
- ingestion/cfbd.go — NewCFBDClient (h2 disabled) + GetCFBD (bearer, byte-capped).
- ingestion/crosswalk — Map.Lookup (MFL→gsis), Map.GSISForESPN (espn→gsis); ADD
  GSISForPFR here (item 1).
- The injected-resolver pattern for any non-gsis source.

== CONSTRAINTS ACTIVE (do not re-litigate) ==
- THE CONTRACT: fetchers emit RAW, gsis-keyed; the engine normalizes (Approach A).
- ZERO-LEAK (hard, structural): no field references fantasy points / projected volume /
  MFL scoring.
- crosswalk foundation MFL→gsis map must NOT break when adding optional bridges;
  ambiguous N→gsis is dropped (RAS/espn precedent).
- Do not reopen locked decisions; flag, don't route around.
- CT105: GOMEMLIMIT=1500MiB GOGC=20 make lint (warm cache: go build ./... first).
  Go 1.26.4 at /usr/local/go/bin. Beelink is the Wails/GUI dev machine.
- Review gate: agy if returned (pct enter 104), else Gemini 3.1.

== CLOSE GATE ==
- Build: GOMEMLIMIT=1500MiB GOGC=20 make lint = 0 + go test -race ./... green + the
  refactored fetchers' env-gated live counts UNCHANGED (regression check, not new data).
- Each refactor is behavior-preserving — prove it by the unchanged live count, not just
  a green compile.
- Squash-merge the arc to main. Write the B3b handoff before clearing.
- After this session the B2b-Fetch arc is COMPLETE → next module is B3b (League Rulebook).
