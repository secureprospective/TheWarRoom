# Season-Rollover Design (B7c §14)
**Status:** LOCKED by expert panel (GLM 5.2 + Gemini + DeepSeek, blind, 2026-07-11) + commissioner rulings.
**Invariant carried:** offseason = START of season N (panel-unanimous, §12).

## Panel outcome
3/3 convergence on Q1–Q3 + the biggest-risk seam. GLM dissented on Q4/Q5; both escalated to
Christopher. Rulings recorded below. Panel brief: `Season_Rollover_Panel_Brief.md`. GLM's full
grounded review (line-level source verification) is the tie-breaking record.

## Decisions

### D1 — The runtime season is DERIVED from `season_phases` (Q1, 3/3)
- Current season = the `season` column of the **league-global latest** phase row
  (`ORDER BY seq DESC LIMIT 1`) — NOT scoped to a frozen season.
- `CurrentPhase` / `seedInitialPhase` must stop scoping by `s.season` and read league-global latest.
- `load()` (post-commit reload) **re-derives `s.season` from the phase log FIRST**, then scopes
  every cap / dead-cap / relief / count read by it.
- MFL config (`ingestion.SeasonYear`) is demoted to **genesis-seed-only**: it seeds the first
  OFFSEASON row and is never trusted as runtime truth again. On boot, a DB with phase rows overrides
  config (fixes the boot-time split-brain risk all three flagged).
- **No `league_meta` table** — a mutable season cursor is a stored competing truth in a
  derived-truth system.

### D2 — Rollover touches NO ledger data (Q2, 3/3)
- Contract cells are absolute-year keyed → advancing the season points the cap at next year's cells.
- `dead_cap` / `cap_relief` are `league_year`-scoped → new season sums to 0 naturally; old rows
  persist for audit (append-only). No carry step. No multi-year dead-cap spread exists (single-season
  charge model). **Correctness is entirely downstream of D1** — a stale season charges the wrong year.

### D3 — No op-count reset (Q3, 3/3)
- `transaction_counts` PK `(league_id, franchise_id, season, op_kind)` → new season reads 0 by
  absence. No reset row (meaningless write). The rollover's own phase transition IS the audit.

### D4 — `is_restructured` is a LIFETIME guard — NO reset on rollover (Q4, commissioner ruling)
- **Ruling: once per contract, ever.** A contract restructured in 2026 can never be restructured
  again. Rollover does NOT touch `is_restructured`.
- The "restructure again next season" capability applies to OTHER contracts and is delivered by D3's
  per-season op-count reset. GLM's push-back vindicated: the flag (per-contract) and the counter
  (per-team-per-season) are distinct guards; the handoff conflated them.
- Consequence: the rollover tx does NOT need a sweep or a reset method. Simpler.

### D5 — Dedicated `ROLLOVER_SEASON` op (Q5, commissioner ruling, GLM position)
- A distinct commissioner-confirmed op, **PLAYOFFS-gated** (legal only from PLAYOFFS). Season moves
  ONLY via this op; plain `ADVANCE_PHASE` always appends with `season = current`.
- It appends the `PLAYOFFS → OFFSEASON` transition row with `season = N+1` — the bump lives in the
  appended row's season column, so D1's "latest row's season" derivation stays clean and the bump is
  never accidental.
- **Season is MONOTONIC / one-way** (3/3): backward season crossing is forbidden in v1. In-season
  phase correction (REGULAR ↔ PLAYOFFS within a season) stays free via `ADVANCE_PHASE`. Extends the
  existing "rollback does not auto-reverse `transaction_counts`" posture to "rollback does not cross
  the season boundary." Assert `newSeason == currentSeason + 1`.

### D6 — The refresh seam is the make-or-break (biggest risk, 3/3)
- Every cap read / charge / count / phase read is scoped by `s.season`, set once in `New()` and never
  refreshed by `load()` today. The whole design reduces to: after the rollover commits, `load()`
  re-derives `s.season` from the phase log before any scoped read.
- **Mandatory:** (a) exhaustive sweep — no `s.season` reader caches the old value across reload;
  (b) a PLANTED DRIFT TEST — commit a rollover, assert a later cap read reflects N+1; on failure the
  store **poisons** (the GLM-B7a committed-but-stale-memory class).
- Rollover must be the **SOLE occupant of its `WriteTx`** (the `OpCount`-reads-committed-state
  footgun corrupts if any op co-resides).

### D7 — Advance the snapshot; season stays single-valued (commissioner ruling, implementation gap the panel brief missed)
- The `rosters`/`contracts` tables carry a fixed `season` column (the snapshot season); roster rows
  are inserted ONLY at seed (`helpers.go:68`) and every roster mutation scopes by `season`
  (`writes.go:124/144/236`). The ledgers (cells `league_year`, dead-cap, relief, counts) are the
  per-year dimension.
- **Ruling: advance the snapshot.** The rollover tx does `UPDATE rosters/contracts SET season=N+1`
  (they are mutable runtime state, not append-only) so `s.season` stays ONE derived value and every
  read scopes by it uniformly — truest to D1. Dead-cap/relief rows are UNTOUCHED (stay `league_year=N`
  as history); `transaction_counts` untouched (N+1 reads 0 by absence). History lives in the
  append-only ledgers + absolute-year cells.
- **Boot-order fix (D1 corollary):** `s.season` must be derived from the phase log BEFORE `hasState`
  in `Initialize`. Otherwise a rolled-over DB rebooted with unchanged config would find no rosters at
  the config season and RE-SEED. Derive first (league-global latest phase row), then hasState/seed/load.

## Interface impact
- New op groups into an embedded `SeasonScope` sub-interface on `TxWriter` (at its 10-member cap) —
  v1 needs only the rollover primitive (no reset method, per D4).
