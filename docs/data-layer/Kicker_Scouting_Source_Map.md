# Kicker Scouting Source Map
**Status:** Locked (decisions recorded 2026-06-26). Companion to `Offense_Scouting_Source_Map.md` and `Defense_Scouting_Source_Map.md`.
**Scope:** the data sources behind the K scouting inputs — the B2b-Fetch-Kicker module. Built on the same seams as offense/defense (extcsv, crosswalk, the injected-resolver pattern).

---

## §1. Summary of what feeds each K scouting component

Under **DECISION-011** K's Layer-4 Film component is ACTIVE with two sub-signals: **Madden 0.60 / NFLProduction 0.40** (RAS and Breakout remain excluded — SL-020 / not-translatable). The mechanics rewrite (Film cap/curve calibration, AD-10/SL-020 reconciliation) executes at **B5b-K**; this module only feeds the data.

| Component | Source (LOCKED) | Fetcher | Keying |
|---|---|---|---|
| Madden K ratings (0.60) | EA ratings-api `m24-ratings` (`kickPower_rating`, `kickAccuracy_rating`, `awareness_rating`) | `madden` (**already built** — no new code) | name + birthdate → gsis (injected) |
| NFL Production (0.40) | nflverse `player_stats_kicking_season` | `kicking` (**new this module**) | gsis |
| Age Trajectory | nflverse `players.csv` | `agetrajectory` (already built) | gsis |
| RAS / Film / Breakout | — | excluded at K (SL-020) | — |

**One new fetcher** (`kicking`). Madden K is covered by the existing fetcher.

---

## §2. Madden K — already fetched (recon finding, verified live)

The handoff flagged "verify K rating attrs are present in the m24 pull before writing anything." Recon (2026-06-26, live EA `m24-ratings`) confirmed:

- m24 carries `kickPower_rating` and `kickAccuracy_rating` (plus `awareness_rating`), and K-position records exist.
- The `madden` fetcher reads **every** integer `*_rating` field into `RawMaddenRating.Attributes` dynamically — so `kickPower` / `kickAccuracy` / `awareness` are already captured for kickers under their suffix-stripped names.
- K records carry `firstName` / `lastName` / `plyrBirthdate`, so they resolve through the existing name+birthdate injected resolver, same path as every other position.
- Live spot-checks: Cairo Santos (kickPower 94 / kickAccuracy 85 / awareness 63), Tyler Bass (96/80/65), Dustin Hopkins, Matt Prater — sane values, birthdates present.

**No new Madden code.** (Madden m24 is two seasons stale — the same lower-durability fallback caveat that applies everywhere; durability never weights, per the locked principle.)

---

## §3. NFL Production for K — new `kicking` fetcher

**The handoff premise that nflverse `player_stats_season` carries kicking columns is FALSE.** That file is offense-only — no FG/XP columns at all (recon: its header has no `fg_*`/`pat_*`/`xp` fields). The existing `nflproduction` fetcher binds only passing/rushing/receiving and cannot surface kicking.

Kicker production lives in a **separate** nflverse release file: `player_stats_kicking_season_<year>.csv` (live HTTP 200). It is rich and gsis-keyed:

- `player_id` (gsis), `season_type` (REG filter), `games`, `team`
- `fg_made` / `fg_att` / `fg_missed` / `fg_blocked` / `fg_long`
- FG-by-distance buckets: `fg_made_0_19 … fg_made_60_` and the `fg_missed_*` mirror (6 buckets each)
- `pat_made` / `pat_att` / `pat_missed` / `pat_blocked`
- (also `fg_pct` / `pat_pct` rate columns and `fg_*_list` / `fg_*_distance` string columns — all deliberately **unbound**)

The `kicking` fetcher emits RAW gsis-keyed **counting stats only** (THE CONTRACT — engine normalizes, Approach A). It does NOT bind `fg_pct` / `pat_pct`: a rate is made ÷ attempted, i.e. analysis the engine derives from the raw counts; emitting it here would pre-bake an engine decision. ZERO-LEAK is structural — `RawKicking` has no field able to hold a fantasy/scoring value.

### §3a. DECISION — traded kickers: SUM the per-team REG splits (Christopher, 2026-06-26)

This file's shape **differs from the offense file**: it is one row per **(player, team, season_type)**, and `season_type` ∈ {`REG`, `POST`, `REG+POST`}. A kicker traded mid-season has a REG row per team, and there is **NO source-provided cross-team aggregate row** (no `2TM`-style combined row, unlike PFR). Live 2024: 45 distinct REG kickers, 7 traded (2–3 rows each).

**Christopher's call: sum a kicker's per-team REG rows into one season record** — counting stats and distance buckets add, `fg_long` takes the max (the season's longest make).

- This is legitimate season aggregation of **raw counts**, distinct from the pfrcoverage prefer-aggregate dedup: pfrcoverage had a source aggregate row to prefer and a rate-recompute risk to avoid; here no aggregate row exists and this fetcher emits no rates.
- Rejected alternatives: **drop traded kickers** (RAS precedent) — loses 7 of 45 incl. real traded starters, bad for a 32-kicker league; **use `REG+POST`** — still per-team splits AND folds in playoffs, reintroducing the playoff-team bias the REG-only filter avoids.
- Live verification: Greg Joseph (NYG 13/16 + NYJ 1/1 + WAS 2/3) → games 8, FGMade 16, FGAtt 20, FGLong = max across teams. 45 distinct records after summing.

---

## §4. Build status

- `kicking` fetcher: built, `make lint` 0 / `go test -race ./...` green / live `TWR_LIVE_NFLVERSE=1` PASS (45 kickers). Commit `887fda4`.
- Madden K: no new code (covered by existing `madden` fetcher).
- SL-OQ-042: RESOLVED (see Roadmap). CAL-032: open (calibration).
- **B2b-Fetch arc NOT yet fully closed** — the deferred module-close refactors (M17 pfr→gsis bridge promotion, shared CFBD long-format helper extraction, agy/Gemini seam re-review) remain → handoff `09-B2b-Fetch-module-close.md`.
