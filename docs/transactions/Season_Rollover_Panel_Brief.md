# Panel Brief — Season-Rollover Machinery (TheWarRoom B7c §14)
**Date:** 2026-07-11 · **Mode:** independent blind panel, parallel self-contained brief, triage vs source.
**Reviewers:** GLM 5.2 (SSH/Beelink) · Gemini · DeepSeek. Answer independently; do not assume house conventions — they are stated below.

You are reviewing a DESIGN, not code. No Go exists yet for this unit. Give a recommendation per question with reasoning; flag anything under-specified rather than guessing.

## System in one paragraph
TheWarRoom is a Go + SQLite (WAL) engine for a 32-team dynasty fantasy-football league with a hard salary cap. **Money = int64 cents**, universal FLAT `$10k` rounding on every stored figure. **The ledger is KING:** per-year `contract_years` cells (absolute-year keyed: 2026, 2027, …) are the sole money truth; `CapUsed(franchise) = Σ(cells for the current season) + Σ(dead_cap this season) − Σ(cap_relief this season)`, floored at 0. All mutations go through ONE spanning `WriteTx` (single-writer law). Two sibling append-only, double-immutable ledgers (`dead_cap_ledger`, `cap_relief_ledger`) and an append-only `season_phases` transition log (OFFSEASON / REGULAR_SEASON / PLAYOFFS) already exist. Per-season op ceilings (2 buyouts, 1 tag, 1 extension per franchise) live in `transaction_counts`, PRIMARY KEY `(league_id, franchise_id, season, op_kind)`.

## The one fact that reframes everything
**The `season` int is currently NOT persisted as league state.** It is parsed from MFL config (`ingestion.SeasonYear`) at startup and frozen into the `Store` struct (`state.New(pools, leagueID, season)`). Every scoped query (`… WHERE league_id=? AND season=?`) reads this frozen value. There is today **no runtime path to change the season** — the whole point of this unit.

**LOCKED INVARIANT (panel-unanimous from §12, do not relitigate):** *offseason = START of season N.* The loaded season int is the season OFFSEASON belongs to. After PLAYOFFS(N) → OFFSEASON, the season becomes N+1 and OFFSEASON is the start of N+1.

## Questions

**Q1 — Where does the runtime season live, and how does it move?**
Options on the table: (a) a new single-row `league_meta` table with a mutable `current_season` column, updated in the rollover tx; (b) DERIVE the current season from the `season_phases` log (the season of the latest row), making the phase log the sole lifecycle truth; (c) keep the config value as the *floor* and store only a delta. Which, and why? How does the in-memory `Store.season` field get refreshed after a rollover commits (the store reloads derived state post-commit)? Note the config value stays as-is — it must not be re-read as truth after the first rollover.

**Q2 — What does a rollover DO to the ledgers?**
Contract cells are absolute-year keyed, so they don't move — incrementing the season naturally rolls the cap to next year's cells. `dead_cap` and `cap_relief` are this-season-scoped, so a new season leaves them behind (their old-season rows stay, new-season sum starts at 0). Confirm this is correct and that NOTHING needs an explicit "carry" step. Is there any dead-cap that should persist across the boundary (e.g. a multi-year dead-cap spread)? (Current model: dead cap is a single-season charge.)

**Q3 — Per-season op counts.** `transaction_counts` is season-keyed, so advancing the season int makes every franchise's counts read 0 for the new season automatically (no reset write needed). Confirm this is the intended and safe behavior — or argue for an explicit reset/audit row.

**Q4 — The §11 in-season restructure unlock and §10 `is_restructured` reset.** An extension's in-season restructure unlock was "illusory" until rollover exists. On rollover, per-player `is_restructured` flags must reset so franchises can restructure again next season. These flags live on contract rows, not season-scoped tables. Design: does the rollover tx sweep-reset them, or should they be derived-per-season? Where does that write belong under the single-writer law?

**Q5 — Op shape & phase coupling.** Is rollover a NEW op, or an extension of the existing `ADVANCE_PHASE` op when the target is PLAYOFFS→OFFSEASON? It is commissioner-confirmed, no clock automation. Given `ADVANCE_PHASE` already permits any target (including rollback), how should a season-incrementing rollover interact with rollback — can you roll BACK a rollover, and if so what happens to the season int and the counts? Recommend the cleanest, least-surprising coupling.

## Locked decisions that carry (reuse, don't relitigate)
- int64 cents, FLAT `$10k` grid via `RoundToNearest10k`. Ledger is KING; cap/dead-cap/relief DERIVED.
- Single-writer law: all mutations through one `WriteTx`. `TxWriter` is at its 10-member interface cap — new season ops group into an embedded sub-interface, not bare methods.
- All lifecycle logs append-only + double-immutable (write-only Go API + BEFORE UPDATE/DELETE RAISE(ABORT) triggers).
- `ADVANCE_PHASE` is commissioner-confirmed, no clock, any target allowed, append-only audit; rollback does NOT auto-reverse `transaction_counts` (documented v1 posture).
- Build gates: `make lint` 0 (filelen 400-line cap, ifaceguard, golangci), `go test -race` green.

## What to return
For each Q: a recommendation + one-paragraph rationale + any invariant risk you see. Then one closing "biggest risk in this design" call. Blind — no coordination with the other reviewers.
