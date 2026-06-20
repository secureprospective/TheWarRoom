# Offense Scouting Source Map & Automation Decision (B2b-Fetch-Offense)
**Date:** 2026-06-19 · **Status:** Decision locked (Option D). Implementation pending.
**Owner sign-off:** Christopher (tie-break vote) · **Review gate this session:** Gemini 3.1 (agy out)
**Supersedes:** the "manual manifest for PFF/RSP/DraftNetwork/Sharp/RAS" plan in the first version of OQ-015.

---

## TL;DR — what changed this session

We set out to pick ingestion paths for the offense scouting fetchers (QB/RB/WR/TE) and ended up making a **strategic pivot**: **eliminate manual data entry** and automate the scouting inputs. After 4 recon passes (Claude + Gemini), 2 quality/durability rating rounds (which converged), and a dual architectural vote, the decision is **Option D — Pragmatic Scouting Hybrid**:

- **Veterans → 0 manual inputs.** All scouting signals sourced from free, Go-reachable APIs / data files.
- **Rookies → exactly 1 manual input per year** — a consensus draft-rank CSV (~45 min once, pre-rookie-draft).
- **PFF, Matt Waldman RSP, The Draft Network, Sharp Football are ELIMINATED** from the rubric (paywalled, no clean automatable source — verified across two independent sweeps).

This **forces a Film-component redesign** (see §5) because for QB/WR/TE those four eliminated sources WERE 100% of the Film component. The new film signal is **FTN charting (primary, verified) + Madden (fallback)** for veterans; **consensus-rank + Madden-rookie + combine + CFBD college** for rookies. **The new rubric weights are NOT set — they are a calibration-against-data job, not a number to lock blind (see §6).**

---

## 1. The decision and why (Option D)

Both independent votes converged on D. The deciding argument (the "rookie problem"): NFL-derived signals (production, snaps, FTN) **do not exist for incoming rookies** — they have no NFL snaps. Strip manual scouting from rookies and the prospect film signal collapses to "Madden-rookie-rating + combine + college box score." Madden rookie ratings are backwards-engineered from draft capital/hype, and college box scores can't see separation-vs-press or blown safety reads — exactly the judgments a dynasty *scouting* engine exists to make. So D keeps ONE minimal manual rookie input to protect the engine's core job. Full automation (Option B) was rejected for rookies; a deliberate down-weight of film (Option C) was rejected as turning a scouting engine into a stats model.

**Escape hatch checked and FAILED:** the one thing that would have automated even the rookie input — a maintained repo auto-compiling a clean per-player consensus *scouting* board into a fetchable CSV — does not exist today (`benjackson-data/NFL-Draft-Class-Analysis` is class-level, and its board input is rank-only AND manual/gitignored; `JackLich10` is dead since 2021; `underdogmockdraft` is fantasy/DFS = zero-leak reject). If such a source appears, the last manual input automates too (future OQ).

---

## 2. Verified source map (all endpoints hit live on CT105, 2026-06-19)

| Field(s) | Source & exact access | Quality / Durability | Status |
|---|---|---|---|
| **MFL↔gsis crosswalk (FOUNDATION)** | dynastyprocess `https://raw.githubusercontent.com/dynastyprocess/data/master/files/db_playerids.csv` (`mfl_id`,`gsis_id`) | High / Med (3rd-party maintained) | ✅ BUILT + live-verified (`internal/ingestion/crosswalk`, 7979 entries, 2026-06-20) |
| NFLProduction | nflverse `…/releases/download/player_stats/player_stats_season_{year}.csv` | High / High | ✅ verified 200 |
| TouchShare (RB) | nflverse `…/releases/download/snap_counts/snap_counts_{year}.csv` | High / High | ✅ verified 200 |
| CollegeProductionShare | CFBD `GET api.collegefootballdata.com/stats/player/season?year=&team=&category=rushing\|receiving` + local Go team-sum | High / High | ⚠ needs key (401) + targets check (§4) |
| SchoolTier | CFBD `GET /teams` → conference→tier map (static logic ours) | High / High | ⚠ needs key |
| AgeTrajectory (NFL vets) | nflverse `…/releases/download/players/players.csv` (`birth_date`) | High / High | ✅ verified 200 |
| RAS (homebrew equivalent) | nflverse `…/releases/download/combine/combine.csv` + per-position z-score in Go | Med (proxy, NOT Platte's exact #) / High | ✅ file verified 200 |
| Veteran Film | nflverse `…/releases/download/ftn_charting/ftn_charting_{year}.csv` joined to PBP (§3) | Clean zero-leak / High (needs join layer) | ✅ columns + join key verified |
| MaddenFilm + NFL birthdate | EA `GET ratings-api.ea.com/v2/entities/m{NN}-ratings` (JSON, no-auth) | High value / LOW-MED | ⚠ see §4 — current season BLOCKED |
| College birthdate (rookies) | chain: combine.csv DOB → Wikidata SPARQL → manual seed | Low-Med / Med | partial; incomplete coverage |
| **Rookie consensus film** | **MANUAL** consensus-rank CSV (`Rank, Player, Position, College`), normalized [0,1] within class | — | the 1 surviving manual input |

**ELIMINATED (no clean automatable source — two sweeps):** PFFGrade, RSPQualitative, DraftNetwork, SharpFootball.

**GAP CLOSED (2026-06-20, B2b-Fetch build):** the original §2 had no gsis→MFL crosswalk, yet `scouting.Profile` is keyed by MFL id and every nflverse source keys on `gsis_id` — so as written no nflverse signal could attach to a rostered player. nflverse `players.csv` does NOT carry `mfl_id` (verified live; it has gsis/pfr/espn only). Fix: dynastyprocess `db_playerids.csv` (row above). **Direction is MFL→gsis** (keyed by the unique side): gsis→MFL is genuinely one-to-many because MFL keeps duplicate player records (live: gsis `00-0031320` → mfl `12459` AND `12571`). The R-generated source encodes missing cells as the literal string `"NA"`, not empty — both are skipped at the boundary. Built as the foundation leaf all other offense fetchers join through.

**Defense-session note (B2b-Fetch-Defense, do not build here):** NGS receiving = `…/releases/download/nextgen_stats/ngs_receiving.csv.gz` (GZIPPED) is the CB/S Coverage anchor.

---

## 3. FTN → PBP join (the veteran film mechanism) — VERIFIED

FTN charting is **play-level, no player_id** (cols: `ftn_game_id, nflverse_game_id, season, week, ftn_play_id, nflverse_play_id, is_catchable_ball, is_contested_ball, is_created_reception, is_drop, is_interception_worthy, is_throw_away, read_thrown, …`). To make it a per-player film signal:

1. Fetch `ftn_charting_{year}.csv` and nflverse `pbp/play_by_play_{year}.csv`.
2. Join `FTN.nflverse_play_id = PBP.play_id` AND `FTN.nflverse_game_id = PBP.game_id` (both keys verified present in PBP: `play_id` col 1, `game_id` col 2).
3. Attribute each charted play to `PBP.receiver_player_id` / `rusher_player_id` / `passer_player_id` (all verified present).
4. Aggregate per player into trait rates (e.g. contested-catch rate, drop rate, created-reception rate).

This is the durable, current, clean veteran-film anchor — it replaces the role Madden-current was meant to fill (Madden-current is blocked, §4).

---

## 4. Open verification items (close before locking the build)

- **CFBD API key — RESOLVED (2026-06-20).** Key minted + live-verified on CT105: `GET /teams?year=2024` → **HTTP 200** (gate cleared; both CFBD fetchers unblocked). Key lives in CT105 env (`CFBD_API_KEY`), not committed. **Targets question ANSWERED: CFBD does NOT expose targets.** Live `/stats/player/season?year=2024&team=Georgia&category=receiving` returns statTypes `[LONG, REC, TD, YDS, YPR]` — no `TGT`. **→ receiver `CollegeProductionShare` is reception/yardage share, NOT target share** (the prior held). FETCHER NOTE: the response is **long-format** — one row per stat (`{playerId, player, position, team, conference, category, statType, stat}`) — so a player's share = Σ(their REC or YDS) ÷ Σ(team REC or YDS), pivoting across rows. (Rate limit: Gemini's "240 req/min" still unverified; daily college fetch is well under any plausible cap.)
- **Madden current-season is BLOCKED:** the EA entities API serves **historical only**. `m23-ratings`/`m24-ratings` return full records (m24 = 1992 players, full sub-attrs + `plyrBirthdate` + `age`); **every current-season variant 500s** — `m26-ratings`, `madden-nfl-26`, `m26`, etc. (the live ea.com page itself names `madden-nfl-26`, and it still 500s). So: **birthdates are safe** (DOB doesn't go stale — use a historical slug), but **current MaddenFilm ratings are NOT cleanly automatable here.** Fallback for current Madden = third-party scrape (`theedgepredictor/nfl-madden-data` pipeline, but stale Aug-2025; or madden.tools) = lower durability. Decision: FTN is the PRIMARY veteran film signal; Madden-current is a lower-durability fallback, not the anchor.
- **College birthdate coverage:** combine.csv covers only ~320 invitees/yr → Wikidata SPARQL fallback → manual seed for the rest. Incomplete; the chain is upkeep.

---

## 5. Rubric impact — the Film component MUST be redesigned

Verified Film sub-signal weights (current, locked rubrics) — all the eliminated manual sources:

| Pos | Film component (current) | Automated remaining after elimination |
|---|---|---|
| QB | PFF 0.45 · RSP 0.35 · DraftNet 0.10 · Sharp 0.10 | **0.00 — empty** |
| WR | RSP 0.40 · PFF 0.40 · DraftNet 0.10 · Sharp 0.10 | **0.00 — empty** |
| TE | PFF 0.40 · RSP 0.35 · DraftNet 0.15 · Sharp 0.10 | **0.00 — empty** |
| RB | RSP 0.35 · PFF 0.35 · TouchShare 0.20 · DraftNet 0.05 · Sharp 0.05 | only TouchShare 0.20 |

Note: the rubric ALREADY uses **Madden as a "regulator"** of subjective film grades (every subjective sub-signal tagged "Madden-regulated"; Madden in every blend table at α 0.20). So Madden has an existing architectural seat — promoting it from regulator to a weighted signal is consistent with the design, NOT foreign to it. (This corrects the Haiku recon, which reported MaddenFilm as "unused at offense" — it's the regulation anchor.)

**New Film signal composition (Option D):**
- **Veterans:** FTN-charting trait composite (primary) + Madden sub-attrs (fallback, lower durability).
- **Rookies:** consensus-rank (manual, normalized) + Madden-rookie-rating + combine/RAS + CFBD college *advanced* metrics (success rate, havoc — NOT PPA, which is points-based → zero-leak risk).

---

## 6. The rubric REWEIGHT is a calibration job — not done, do not fake it

Two principles locked by the dual vote (unanimous):
- **Durability must NEVER influence a rubric weight.** It's an uptime concern → handled by SQLite caching + stale-data preservation + a manual fallback, never by down-weighting a football signal. (Down-weighting RAS because the combine file might break someday is irrational.)
- **Quality/fidelity MAY inform weights** — but only as a **fidelity discount on a like-for-like proxy** (Gemini suggested ~15% for tracking-data proxies of direct film), resolved by **calibration against real data (CAL-series)**, NOT a from-scratch "rank sources by confidence" scheme. "Weight by source confidence" was rejected as a category error that lets the data pipeline decide the football.

→ The new weights are a deliberate calibration pass with live data. Until then, weights are UNSET. Confidence on the source/architecture path is ≥85% (verified); confidence on final weights is explicitly NOT — that's calibration, not a blind number.

---

## 7. Zero-leak ledger (hard constraint)

- ✅ FTN charting — verified clean at the column level ("how", not "how much").
- ✅ combine, snap_counts, players.csv, CFBD season stats (raw counting/share) — clean.
- ⚠ Raw NFLProduction / TouchShare are VOLUME signals — clean of fantasy points, but cap/watch so they don't dominate a *scouting* rank (identity risk).
- ❌ REJECT: FantasyPros `ecrData` (Expert Consensus *fantasy* Rankings = leak; also redundant w/ snap_counts). CFBD `PPA` (points-based predicted-points = likely leak — use success rate/havoc instead). `underdogmockdraft` consensus (fantasy/DFS operator).
- ✅ Rookie consensus-rank = draft-analyst SCOUTING consensus (Brugler/Jeremiah-type), NOT fantasy ADP — clean.

---

## 8. Follow-on tracking (open questions / calibration)

- **OQ-015** (roadmap) — this decision; now resolved to Option D, links here.
- **NEW (assign at build):** rubric film-weight recalibration = a CAL pass against live data (Madden-regulator mechanics + FTN-proxy fidelity discount).
- **NEW (future OQ):** auto-ingest the rookie consensus-rank board if a clean, maintained source appears (NFL Mock Draft Database is the likely target; API unverified).
- **OQ (durability):** Madden current-season source — accept scraper fallback or historical-only-for-film.

---

## Provenance
4 recon passes (Claude web/GitHub sweeps + Gemini rounds 1–2), 2 quality/durability rating rounds (Claude & Gemini converged), 1 dual architectural vote (Claude voted B then updated to D after verification; Gemini voted D at 95%; owner = tie-break). Every endpoint in §2 hit live on CT105 this session. Findings that did NOT survive verification were discarded (Gemini's stale `JackLich10` "updated every cycle" claim, its repeated 404 URL, its `filter=iteration:1` Madden fix falsified live 3×).
