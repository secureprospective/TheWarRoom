HANDOFF — Session 7: B2b-Fetch-Defense — defense + IDP scouting fetchers
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B2b-Fetch-Offense is COMPLETE + merged-pending (branch session/b2b-fetch-offense, NOT merged,
  ahead of main). All 9 offense-relevant fetchers built, reviewed, and LIVE-verified on CT105:
    · crosswalk (MFL->gsis FOUNDATION + espn->gsis bridge)   7979 / 7885 entries
    · nflproduction · touchshare · agetrajectory · ras       (the nflverse counting/measurable set)
    · veteranfilm (FTN->PBP join, streamed)                  209 recv / 45 pass
    · schooltier (CFBD /teams)                               681 schools
    · collegeshare (CFBD season stats, espn->gsis keyed)     407 records (2023)
    · madden (EA m24, name+birthdate keyed, INJECTED)        1404 records
  Build gate every commit: GOMEMLIMIT=1500MiB GOGC=20 make lint = 0, go test -race ./... green,
  env-gated live PASS. Zero-leak re-verified FIELD-BY-FIELD across all 9 (no fantasy/PPA/MFL-scoring
  binding anywhere).
- STAY ON session/b2b-fetch-offense if continuing the B2b arc, OR cut session/b2b-fetch-defense if
  Offense is merged first — confirm with Christopher which.

== SHARED SEAMS NOW AVAILABLE (clone, do not re-invent) ==
- internal/ingestion/extcsv.go — FetchCSV (buffered, byte-capped), StreamCSV (row-by-row, for >64MiB),
  CSVColumns (by-name, BOM-stripped), IsMissing, IntCell/FloatCell. CSV sources only, NO gzip (see gate).
- internal/ingestion/cfbd.go — NewCFBDClient (HTTP/1.1 pinned; CT105 h2 PROTOCOL_ERRORs) + GetCFBD
  (bearer, byte-capped, returns body bytes for a concrete lenient decode). CFBD_API_KEY needs TrimSpace.
- internal/ingestion/crosswalk — Map.Lookup (MFL->gsis), Map.GSISForESPN (espn->gsis). The id-bridge
  package; if defense needs a new id system, add the bridge HERE (one db_playerids fetch, N maps).
- THE INJECTED-RESOLVER PATTERN (collegeshare, madden) — when a source keys on something other than
  gsis, define a `type GSISResolver func(...) (string, bool)` and inject it; the fetcher passes raw
  source ids and trusts the result, staying decoupled and fake-testable. Unresolved = clean miss;
  two-to-one = drop the ambiguous gsis (poison). Reuse this for any non-gsis defense source.
- Fetcher shape: ctx first; inject client+url(+key/resolver); own subpackage (depguard layer1 for free);
  fail loud; columns/fields BY NAME; RAW + gsis-keyed (THE CONTRACT — engine normalizes, Approach A);
  every gate proven by a deliberate violation in the same commit (M3); shared logic extracted not
  copy-pasted (M17). schooltier is the lone FINAL-value exception (position-independent tier).

== HARD GATE BEFORE NGS CODE — the GZIP seam ==
- NGS Coverage (the CB/S Coverage anchor — scouting.NGSCoverage, non-nil at CB/S ONLY) is sourced from
  nflverse next_gen_stats, which is served GZIPPED (`…/releases/download/nextgen_stats/ngs_*.csv.gz`).
  extcsv's FetchCSV/StreamCSV do NOT gunzip. FIRST TASK: add a gzip step — wrap the response body in a
  gzip.Reader before the csv.Reader. Cleanest: a `FetchCSVGz`/`StreamCSVGz` sibling in extcsv.go (or a
  `gzipped bool` on the existing two). Prove it with a deliberate violation (a non-gzip body must fail
  loud, not silently mis-parse). VERIFY the exact ngs file + columns live on CT105 before binding —
  the 06d note said "ngs_receiving" but confirm which ngs_* file carries the DEFENDER coverage metric
  (separation allowed, target rate) vs the receiver-side file; bind the defender metric.

== HARD GATE — IDP film source access (parallels the Offense Option-D pivot) ==
- scouting.IDPFilm names three sources: IDPShow, IDPGuru, DynastyNerds. Offense found its qualitative
  film sources (PFF/RSP/DraftNetwork/Sharp) had NO clean automatable source and ELIMINATED them
  (Option D), replacing them with FTN(primary)+Madden(fallback). EXPECT THE SAME for IDP: recon each
  IDP source for a clean Go-reachable feed BEFORE building; if none, the IDP film signal likely
  collapses to Madden(defense sub-attrs)+NFLProduction+NGS, a parallel redesign decision for Christopher.
  Do NOT build a manual IDP manifest without his call. Madden defense sub-attrs are ALREADY fetched
  (madden Attributes map: tackle, manCoverage, zoneCoverage, hitPower, … — all live in m24).

== WHAT THIS SESSION LIKELY BUILDS (confirm scope with Christopher first) ==
1. extcsv gzip seam (above) — prerequisite for NGS.
2. NGS Coverage fetcher — internal/ingestion/ngscoverage/ (or similar), gsis-keyed RAW, CB/S metric.
3. Defense college production / PFF-defense — likely CFBD defense stats (havoc/TFL/sacks — clean;
   NOT PPA) via the existing cfbd.go client, espn->gsis keyed via the existing bridge. Recon CFBD's
   defensive stat categories live first.
4. IDP film — pending the source-access decision above.
Madden + RAS + AgeTrajectory + SchoolTier + CollegeProductionShare already serve defense too (they are
position-independent or the engine selects per position) — do NOT rebuild them.

== MODULE-CLOSE ITEMS CARRIED FROM OFFENSE (do at the END of the B2b arc, not now) ==
- agy/Gemini RE-REVIEW of the seam changes introduced this arc (agy was out): IntCell/FloatCell ->
  extcsv (06b), StreamCSV (06c), the shared ingestion/cfbd.go (06d), and the upcoming gzip seam.
  Triage every finding against source (Gemini's confident-but-wrong rate is high here).
- The rubric FILM REWEIGHT (offense AND defense) is a SEPARATE CALIBRATION pass — durability never
  weights; quality = fidelity discount. The eliminated-source fields (PFFGrade, DraftNetwork,
  OffenseFilm RSP/Sharp; and whatever IDP eliminates) are retained-pending-redesign, populated by no
  fetcher. Do NOT set weights in a fetcher session.
- OQ-013 (created/placeholder gsis -> official gsis reconciliation ramp) now has a concrete instance:
  collegeshare emits dynastyprocess PLACEHOLDER gsis (e.g. "ALL092954") for not-yet-played rookies —
  a consistent join key today, reconciled when the NFL assigns the real id. Defense rookies inherit it.

== CONSTRAINTS ACTIVE (do not re-litigate) ==
- THE CONTRACT: fetchers emit RAW, gsis-keyed; the ENGINE normalizes (Approach A). schooltier is the
  one position-independent FINAL-value exception.
- ZERO-LEAK (hard, structural): no field references fantasy points / projected volume / MFL scoring.
  CFBD: raw counting/havoc/success-rate, NEVER PPA (points-based = leak).
- NGS Coverage anchor applies at CB and S ONLY (scouting.NGSCoverage nil elsewhere) — STRUCTURAL.
- SL-019 NOT applied at DT (Cushion Guard replaces it). Do not reopen locked decisions; flag, don't route.
- CT105: GOMEMLIMIT=1500MiB GOGC=20 make lint (warm cache: go build ./... first). Go 1.26.4 at
  /usr/local/go/bin. CT105 reaches nflverse/CFBD/EA directly (CFBD needs the key, h2 disabled). The
  pbp/large gz live tests run long on CT105's ~310 KB/s link — set generous timeouts.
- Review gate: Gemini 3.1 (agy out; re-auth via pct enter 104 if it returns).

== CLOSE GATE FOR ANY SESSION ==
- Build: GOMEMLIMIT=1500MiB GOGC=20 make lint = 0 + go test -race ./... green + env-gated live PASS.
- Functional (Christopher / live): the new fetcher's live run yields sane data; a deliberate violation
  proves each new gate (esp. the gzip seam). Write the next handoff before clearing.
