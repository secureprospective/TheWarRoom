I'll treat this as a task (no boot). Before answering, let me ground the load-bearing claims about *existing* code — a design review on a faulty premise is worthless. Quick parallel checks.
Let me confirm the load-bearing specifics: the `TxWriter` interface shape, how `Store.season` is set/refreshed, and the exact `transaction_counts` PK + the `Season()` impl.
Let me read the interface definitions and the Store struct/`Season()` impl — these are the crux of Q1.
Need the `Season()` impl and the `load()` reload path — the crux of whether the season refreshes post-commit.
Let me see the `season_phases` schema and the genesis-seed logic — this determines whether the phase log can *be* the season source of truth (Q1 option b).
I grounded the brief against source before answering — three load-bearing facts the panel's recommendations hinge on, verified not assumed:

1. **`s.season` is frozen and never refreshed by reload.** `state.go:64` sets it only in `New()`; `txWriter.Season()` returns `w.s.season` (`writes.go:243`); and `load()` (the post-commit reload, `state.go:259+`) scopes *every* read — rosters, contracts, cells, dead-cap, cap-relief, phase — by `s.season` **without re-reading it**. The reload refreshes derived cap memory but with the *frozen* year.
2. **`season_phases` is partitioned by season.** `CurrentPhase` and `seedInitialPhase` query `WHERE league_id=? AND season=?` (the frozen value), not league-global latest. The phase log is not yet a league-wide lifecycle cursor.
3. **`is_restructured` is a per-*contract* guard**, distinct from the per-*season* `transaction_counts` guard. §11 ships *both*: flag = "one restructure per contract," counter = "one per team per year."

These reshape two of the five answers. Full review below.

---

## Q1 — Where the runtime season lives, and how it moves

**Recommendation: (b) — derive `current_season` from the `season_phases` log (the `season` of the league-global latest row); have `load()` re-derive `s.season` from it *before* scoping any cap read. Reject (a) and (c).**

This codebase's governing discipline is *one source of truth; derived values are computed, never stored as competing truth* — cap is derived, dead-cap is derived, relief is derived. A mutable `league_meta.current_season` (a) is a stored competing truth in exactly the system that has rejected them, and it creates *two* lifecycle sources (the mutable cursor + the append-only phase log) that must agree and can drift under rollback. (c) is (a) with extra indirection: the brief itself says the config value must never be re-read after the first rollover, so the "delta" *is* the only live value — you've reinvented a cursor for nothing. (b) makes the season an *attribute of the latest transition row*, so the season moves *because* the phase log moved — no separate write, no sync surface. The cost is real but localized and worth paying: `CurrentPhase`/`seedInitialPhase` must stop being scoped to the frozen `s.season` and read the **league-global latest row** (`ORDER BY seq DESC LIMIT 1`), and `load()` must set `s.season` from that row *first*, then scope all cap/dead-cap/relief reads by it. The config value is demoted to the *initial floor only* — it seeds the first OFFSEASON genesis row and is never trusted again, which (b) makes natural rather than enforced.

**Invariant risk:** a chicken-and-egg seam. Today the season *scopes* the phase read; under (b) the phase read must *produce* the season. Any code path that still reads the phase (or any ledger) under the old frozen-season scoping after a rollover has committed will silently compute against the wrong year. This is a one-line-per-query change with system-wide blast radius — it must be exhaustive, and a drift test (cap reads must track the derived season) is mandatory.

---

## Q2 — What rollover does to the ledgers

**Recommendation: confirm correct — nothing needs an explicit carry. Contract cells are absolute-year keyed so the cap naturally points at the next year's cells; `dead_cap`/`cap_relief` are absolute-`league_year`-scoped so a new season orphans the old rows (they persist for audit, contribute 0 to the new year), and the new year starts at 0. No multi-year dead-cap spread exists to carry — the locked model charges the entire §8/§12 figure in the cut year, flat.**

This is the cleanest part of the design precisely *because* the ledgers were built absolute-year-keyed and this-season-scoped. The dead-cap/cap-relief ledgers don't "reset" — they simply don't match the new `league_year`, so `CapUsed = Σ(this-year cells) + Σ(dead_cap league_year=N+1) − Σ(relief league_year=N+1)` starts clean. Old rows remain for history and never vanish (append-only, immutable triggers). The single-season charge model means no rollover-time dead-cap propagation logic is required or desirable.

**Invariant risk:** Q2's correctness is *entirely downstream of Q1*. Every dead-cap/relief charge writes `league_year = w.s.season` (`writes.go:171`, `cap_relief.go`). If the season int is stale after a rollover, a buyout in OFFSEASON(N+1) silently charges `league_year = N` — dead cap lands in the *wrong year*, invisible until someone notices the cap is off. "Nothing to carry" is true *only if* the scoping season is advanced correctly. This dependency must be stated explicitly in the design, not left implicit.

---

## Q3 — Per-season op counts

**Recommendation: confirm the intended and safe behavior — advancing the season makes every franchise's counts read 0 for the new year automatically (no row exists for `(league, franchise, N+1, op_kind)`, and `OpCount` returns 0 on no-row). Do *not* add an explicit reset/audit row.**

The §10 rulebook ruling ("each season resets the counter") is realized *literally* by the season key — `transaction_counts` PK `(league_id, franchise_id, season, op_kind)` confirmed at `schema.go:87-94`. An explicit reset row would be a write that accomplishes nothing: writing 0-count rows for the new season is meaningless noise, and the "audit" that a new season began already lives in the rollover's own phase-log transition. Absence-of-row = 0 is the *natural* representation, matching the ledger-is-king ethos (don't store what's implied by the key structure).

**Invariant risk:** the documented `OpCount` footgun (reads committed state, not this-tx's own `IncOpCount`) is safe for rollover *only because* rollover must be the **sole occupant** of its `WriteTx`. Every op reads `w.s.season` monotonically throughout a transaction; if any contract op were co-resident in the rollover tx, its count bump would land at whichever season `s.season` held — corrupted. **Rollover must be a standalone tx.** Encode that as a design rule, not a hope.

---

## Q4 — The `is_restructured` reset

**Recommendation: push back on the premise — it is under-specified and likely conflates two guards. `is_restructured` is a per-*contract* lifetime guard ("one restructure per contract"); the per-*season* "restructure again next year" capability is *already* provided by `transaction_counts` resetting (Q3). Whether the *same contract* may be restructured in a new season is a rulebook question for Christopher, not a machinery given. Flag it; do not assume the reset.**

Today §11 enforces *both*: the flag blocks a second restructure of one contract; the season-keyed counter blocks a second restructure by one team in one year. The brief's claim that "flags must reset so franchises can restructure again next season" describes the *team-level* capability, which Q3 already delivers for *different* contracts — no flag reset needed. Resetting `is_restructured` on rollover would instead permit re-restructuring the *same* contract yearly, which may or may not be the rule intent. **If** the ruling is "per-contract-per-season": prefer deriving "restructured this season" from a per-season restructure event (append-only, like the counters) over sweeping a stored boolean — a stored flag that must be bulk-reset is a denormalized competing truth in a derived-truth system, and a missed/partial sweep silently misfires the guard. **If** v1 pragmatism demands the sweep, it belongs inside the rollover's `WriteTx` (satisfying the single-writer law) behind a new embedded sub-interface method so `TxWriter` stays at 10 — and it should be the *only* non-append-only write in the rollover besides the phase append, loudly documented.

**Invariant risk:** a stored boolean reset by sweep is the exact "stored competing truth" pattern this codebase exists to kill. If a season passes with no rollover run (or a partial one), the flags drift and the restructure guard silently allows or blocks the wrong moves. Derive it instead.

---

## Q5 — Op shape & phase coupling

**Recommendation: rollover is a NEW op (`ROLLOVER_SEASON`), gated PLAYOFFS-only, commissioner-confirmed — *not* an implicit side effect of `ADVANCE_PHASE→OFFSEASON`. Under Q1(b), it appends the `PLAYOFFS→OFFSEASON` transition row with `season = N+1` (the bump lives in the appended row's season column). Plain `ADVANCE_PHASE` always appends with `season = current`; the season moves *only* via a ROLLOVER op. Treat rollover as ONE-WAY in v1: in-season phase rollback (within a season) stays fully free via `ADVANCE_PHASE`, but backward *season* crossing is disallowed.**

A season rollover is high-consequence and irreversible-in-spirit — it moves the entire league's cap, counts, and restructure eligibility forward. Piggybacking that onto a generic phase advance would hide a season change inside an op the commissioner reads as "advance the phase" — surprising and dangerous. A dedicated op makes the intent explicit, gives it its own audit row and confirmation, and localizes the `is_restructured` sweep (if Q4 rules yes). This stays consistent with Q1(b): the season is always "latest row's season," and ROLLOVER is the *only* op that writes a row whose season column is bumped — so the derivation stays clean and the bump is never accidental. On rollback: the phase log is append-only, so "undo" means appending a reverse transition — but reverse-rolling `OFFSEASON(N+1)→PLAYOFFS(N)` reverts the season pointer while *not* relocating N+1 ledger charges, *not* reversing N+1 counts, and *not* un-sweeping restructure flags. That multi-state inconsistency has no clean auto-reversal; the least-surprising rule is that the season boundary is a committed door. In-season phase correction (REGULAR↔PLAYOFFS within N) remains unrestricted, which is all `ADVANCE_PHASE` rollback was ever for.

**Invariant risk:** if backward season crossing *is* permitted later, you get orphaned ledger charges (N+1 `league_year` rows that no scoped cap read can see), stranded counts, and stale restructure flags — silent cap corruption with no loud failure. Forbid it in v1 and document the manual-correction path, extending the existing "rollback does not auto-reverse `transaction_counts`" posture to "rollback does not cross the season boundary."

---

## Biggest risk in this design

**The frozen `s.season` refresh seam — it is a silent-corruption single point of failure, and it does not exist today.** Every cap read, every dead-cap/relief charge, every `OpCount`, every `CurrentPhase`, and every `transaction_counts` bump is scoped by `s.season`, which is set *once* in `New()` and *never* touched by `load()` (verified: `state.go:64`, `writes.go:243`, all reads in `load()`/`season_phase.go`/`cap_relief.go` use the frozen value). The entire rollover design reduces to one requirement: after the rollover tx commits, `load()` must re-derive `s.season` from the phase log *before* scoping any derived read. If that refresh is missed, partial, reordered, or racy, **nothing fails loud** — the cap engine simply computes every figure against the wrong year: dead cap in the wrong `league_year`, counts that reset (or fail to) at the wrong boundary, a phase gate reading the wrong lifecycle position, restructure eligibility for the wrong season. Off-by-one-year, silently, across the whole league. The rollover *op* is straightforward; this *seam* is where the design lives or dies. It demands (a) an exhaustive sweep of every `s.season` reader to prove none caches the old value across the reload, and (b) a planted drift test that commits a rollover then asserts a subsequent cap read reflects season N+1 — fail that test and the store should *poison*, exactly as the GLM-B7a MAJOR taught for the committed-but-stale-memory class.
