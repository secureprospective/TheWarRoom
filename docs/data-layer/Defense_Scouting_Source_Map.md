# Defense Scouting Source Map
**Status:** Locked (decisions recorded 2026-06-26). Companion to `Offense_Scouting_Source_Map.md`.
**Scope:** the data sources behind the defensive/IDP scouting inputs (DT, DE, LB, CB, S) — the B2b-Fetch-Defense module. Built on the same seams as offense (extcsv, cfbd.go, crosswalk, the injected-resolver pattern).

---

## §1. Summary of what feeds each defensive scouting component

| Component | Source (LOCKED) | Fetcher | Keying |
|---|---|---|---|
| Coverage anchor (CB/S only) | PFR advanced defense (nflverse `pfr_advstats`, `advstats_season_def.csv.gz`) | `pfrcoverage` | pfr_id → gsis (injected) |
| College Production Share (breakout) | CFBD season stats (`defensive` + `interceptions`) | `collegedefense` | espn id → gsis (injected) |
| RAS (athletic) | nflverse `combine.csv` | `ras` (already built — position-independent) | pfr_id → gsis |
| Age Trajectory | nflverse `players.csv` | `agetrajectory` (already built) | gsis |
| School Tier | CFBD `/teams` | `schooltier` (already built — position-independent) | by school |
| NFL Production | nflverse `player_stats_season` | `nflproduction` (already built) | gsis |
| IDP Film | **ELIMINATED → Madden redesign** (see §3) | `madden` (already built) + NFLProduction + `pfrcoverage` | gsis |

No defensive fetcher remains to build: the two new ones (`pfrcoverage`, `collegedefense`) plus the position-independent/already-built fetchers cover every defensive input.

---

## §2. DECISION — Coverage anchor rebind (NGSCoverage → PFR)

**The premise in the handoff was wrong.** The CB/S Coverage anchor (`scouting.NGSCoverage`) was specified as nflverse Next Gen Stats "defender coverage (separation allowed, target rate)." Recon (2026-06-26, all 25 nflverse release tags swept) found **nflverse publishes no defender NGS file** — the `nextgen_stats` release ships only `passing`/`receiving`/`rushing`, all offense-keyed; `avg_separation` there is the *receiver's* separation.

**Christopher's call: rebind onto PFR advanced defense** (`pfr_advstats` → `advstats_season_def.csv.gz`), the clean, Go-reachable, gsis-bridgeable defender-coverage source. It carries genuine coverage-**allowed** quality:

- `tgt` (targets → target rate), `cmp` / `cmp_percent` (completions allowed), `yds` / `yds_cmp` / `yds_tgt` (yards allowed), **`rat` (passer rating allowed — the headline coverage metric)**, `dadot` (avg depth of target).

This is coverage-allowed quality, **not** tracking-based separation. The `NGSCoverage` field name is retained; the source substitution is documented here (parallel to the offense Option-D pivot). Emitted RAW, gsis-keyed; the engine normalizes and applies the CB/S boundary. Verified live on CT105: 775 gsis-keyed 2024 coverage records, all passer ratings in [0, 158.3].

---

## §3. DECISION — IDP film eliminated, redesigned on Madden (Option-D parallel)

The IDP film component (per the locked LB/DT/DE/CB/S rubrics §4 film tables) names five sources: **PFF (0.40), The IDP Show (0.30), The IDP Guru (0.20), The Draft Network (0.05), Dynasty Nerds (0.05).**

- PFF and The Draft Network were **already eliminated on offense** (no clean source).
- Recon (2026-06-26) of the three `IDPFilm` sources: **The IDP Show** (theidpshow.com — Substack content + paid projections), **The IDP Guru** (idpguru.com — subscription resource), **Dynasty Nerds** (dynastynerds.com — Premium $69.99/yr, DynastyGM behind login). All are subjective content/paywalled brands with **no clean, free, machine-readable feed or API.**

So **100% of the IDP film component has no clean automatable source.**

**Christopher's call: eliminate IDP film, redesign around Madden** (mirrors offense Option D — veteran film = Madden fallback). The redesigned IDP film signal draws on data **already fetched**:
- **Madden defense sub-attributes** (primary) — already in `madden` `RawMaddenRating.Attributes` (m24): `tackle`, `manCoverage`, `zoneCoverage`, `hitPower`, pursuit, play-recognition, etc. The rubric Madden-regulation tables already map subjective claims to these sub-attrs (e.g. LB "sideline-to-sideline" → avg(SPD,PUR); "coverage LB" → avg(ZCV,PRC)).
- **NFLProduction** + **pfrcoverage** (supporting).

**No new fetcher needed.** The `IDPFilm` struct fields are retained-pending-redesign (populated by no fetcher), annotated in `internal/scouting/types.go`.

**The film reweight is a SEPARATE CALIBRATION pass — weights UNSET here.** Durability never sets a weight; quality enters only as a calibration fidelity discount (the locked offense/defense principle). This decision records the *source* elimination and the *fallback data*, not new numbers.

---

## §4. Zero-leak ledger (defensive sources)

| Source | Fields bound | Leak risk | Disposition |
|---|---|---|---|
| PFR advanced defense | targets, completions, yards/TDs allowed, cmp%, yds/cmp, yds/tgt, passer rating allowed, dadot | none — coverage counting/rates; passer rating is a football stat, not fantasy | clean |
| CFBD defensive + interceptions | tackles, sacks, TFL, PD, INT (+ within-team shares) | **PPA** (points-based predicted points) | PPA NEVER fetched/bound; only counting categories requested |
| Madden defense sub-attrs | integer `_rating` attributes | none — athletic/skill grades | clean (already verified at madden build) |

All defensive fetchers emit RAW and structurally cannot hold a fantasy/projected-volume/MFL-scoring value.

---

## §5. Carry-forward (module-close items)

- **pfr→gsis bridge promotion (Codex M17):** now a 3rd consumer (`touchshare`, `ras`, `pfrcoverage`). The bridge lives only in live-test helpers (`livePfrToGSIS`). Promote a `GSISForPFR` into `crosswalk.Map` (one db_playerids fetch, N maps) and refactor the three onto it.
- **Shared CFBD long-format helpers (M17):** `collegeshare` and `collegedefense` duplicate `statRow`/`fetchCategory`/parsers/poison-drop-emit. Extract into the shared `ingestion` package.
- **agy/Gemini seam re-review:** the gzip seam (`FetchCSVGz`/`StreamCSVGz`) joins the offense-arc seams (IntCell/FloatCell, StreamCSV, cfbd.go) awaiting re-review. Triage every finding against source.
- **The Film reweight calibration pass** (offense AND defense) — weights UNSET; a separate calibration job, not a fetcher session.
