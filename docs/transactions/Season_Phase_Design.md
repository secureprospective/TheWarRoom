# Season-Phase Design (D3) + §12 Buyout — Gate-Check Locked

**Written:** 2026-07-10 · **Status:** DESIGN LOCKED with Christopher; pending expert-panel gate before code.
**Scope:** the season-phase machinery (Vision-2026 D3) built *with* Buyout §12 (Build_Tracker row 27, B7c).
**Sources triaged:** `docs/league-rules/Official_Rulebook.md` §5/§12/§13/§14; `docs/roadmap/Vision_2026.md` D3; the shipped transaction layer (`internal/transactions/*`, `internal/store/state/*`).

---

## Why this exists

§12 Buyout is "two per team per season, **offseason only**." "Offseason" is a season-phase concept
that does not exist in the runtime today — the state store loads a single `season` int and has no
phase. D3 pre-decided the architecture for it; this build executes D3, sized minimally but honestly.

The rulebook does **not** enumerate a clean phase machine. §5 "Season Structure" gives boundaries
(reg season ends Wk13, playoffs), §14 a Wk9 trade deadline, §6 FA windows around Wk12–13 — scattered
temporal facts, not named phases. So we build the full D3 *machinery* and seed only the phases the
rulebook justifies.

---

## Locked decisions (Christopher, 2026-07-10)

**GQ1 — Phase enum: MINIMAL-BUT-STRUCTURED.** Build the full D3 machinery (append-only transitions
table, `ADVANCE_PHASE` op, declarative gate map), but seed the enum with only rulebook-justified
phases: **`OFFSEASON`, `REGULAR_SEASON`, `PLAYOFFS`.** Adding a finer phase later (e.g. a Wk9
trade-deadline sub-phase for the planner) is one enum constant + gate-map rows, not a rearchitecture.
Honest to source; unblocks §12 now.

**GQ2 — Existing ops under the new gate: NO-REGRESSION.** Introduce and populate the gate map, but
set every shipped op (Trade, RosterStatus, Waiver, Restructure, Tag, Extension) to **allowed-in-all-
phases** in v1, with an in-code comment that real windows (Wk9 trade deadline, offseason-only tag/
extension) get pinned once in-season sub-phases exist. **Only `BUYOUT` is phase-restricted (OFFSEASON).**
Machinery lands with zero behavior change to verified ops.

**GQ3 — Buyout rate edges: FAIL LOUD / COMMISSIONER.** §12 defines rates for 2/3/4 years remaining
only. A buyout with `<2` or `>4` remaining years (1-yr deals; up to 6 total via §10 extensions) is
**rejected** as outside §12's defined range — surfaced as a §13 commissioner cap-relief situation, not
an invented rate. No fabricated number in a money path.

---

## Architecture

### 1. State: `season_phases` (append-only transitions)
- New table in `internal/store/state`, keyed conceptually per `(league_id, season)`; each row is one
  transition: `from_phase, to_phase, at, note`. **Current phase = the latest row** for the league-year.
- No CRUD surface (D3): the Go API exposes only append + read-latest. Consider the B6/dead-cap
  double-immutability idiom (BEFORE UPDATE/DELETE `RAISE(ABORT)` triggers) so the audit line can't be
  rewritten even by a raw SQL bypass — confirm at build whether it's warranted here (panel question).
- **Fresh-DB seed = `OFFSEASON`** (the league is built/managed in the offseason; makes §12 immediately
  testable). Seeded once, like the other stores; an existing DB loads its latest row, never reseeds.
- `TxWriter` gains **`CurrentPhase() (Phase, error)`** (read inside the tx, alongside `Season()`/
  `OpCount`), so the gate check is atomic with the mutation it guards.

### 2. Coordinator: the declarative gate (D3)
- A **static `map[Kind][]Phase`** (allowed phases per op_kind) lives in the transactions root pkg.
- Checked as the **first step inside `Execute`'s `WriteTx` closure**: read `CurrentPhase()`, look up
  `req.Kind()`, reject if the current phase isn't in the allowed set. **Default-DENY** an op_kind
  absent from the map (fail loud) — a new op with no phase policy must be classified deliberately, not
  silently allowed. **Planted-test the deny** (a kind missing from the map is rejected).
- v1 map: every shipped kind → all three phases; `KindBuyout` → `{OFFSEASON}`; `KindAdvancePhase`
  → all phases (a phase transition is always legal).

### 3. `ADVANCE_PHASE` op
- New sealed `AdvancePhase{To Phase}` Request + `Coordinator.ExecuteAdvancePhase` (or plain `Execute`,
  since it needs no Directory/price). `apply` appends one transition row (`from` = current, `to` = To).
- **Commissioner-confirmed, no clock automation** (D3): the app suggests, the human confirms. Any
  target phase allowed in v1 (supports rollback/correction; the append-only log audits it). Reject a
  no-op transition (`to == current`) — the no-silent-no-op house rule.

### 4. Buyout op (§12)
- New sealed `Buyout{MFLID}` Request; handler in `internal/transactions/contracts` (or a `buyout`
  subpkg) behind the Coordinator depguard. Frontend sends only the id — every figure resolved in-tx.
- **Math (all from authoritative cells, flat, int64 cents):**
  - `remaining` = paid cells for years after the current `Season()` (the §8/§10 "remaining"
    convention: `expiration − season`, exclusive of the current year).
  - Reject if `remaining < 2 || remaining > 4` (GQ3 fail-loud).
  - `avgRemaining` = mean of those remaining cells' salaries (round half-up, `domain` helper).
  - `rate` = 60% (2) / 75% (3) / 90% (4) — a pinned step table, every fencepost tested.
  - `charge` = `avgRemaining × rate`, rounded to cents. Lands **whole in the current season** as a
    dead-cap entry (`AddDeadCap`, reason `"buyout §12"`) — the §8 pattern.
  - Release the player (`ReleasePlayer` + ledger `VoidCells`, exactly like §8 waiver).
- **Limits:** two per franchise per season via `transaction_counts` op_kind **`"BUYOUT"`**, ceiling 2,
  enforced in-handler (OpCount → check → mutate → IncOpCount), same as TAG. Offseason-only comes free
  from the Coordinator gate — the handler does NOT re-check the phase (single source of truth).

### 5. Carry-forward (explicitly OUT of scope v1)
- **"Cannot bid on a bought-out player until the following offseason"** needs the free-agent pool
  concept — deferred, same as the §8 claim path. Document in-code.
- Fine-grained in-season windows (Wk9 trade deadline, FA bid windows) — pinned when the planner forces
  sub-phases; the enum + gate map extend without rearchitecture.

---

## Gate (unchanged house rules)
- Branch `session/b7c-buyout` off clean main. No work on main. depguard stays PROVEN (plant a rankings
  import, watch it fire, revert — don't commit the plant).
- `make lint` 0 / `go test -race ./...` green / tsc+vite clean / every formula fencepost pinned
  (rates 2/3/4, the `<2`/`>4` reject edges, the default-deny gate) / functional gate live on the
  Beelink before merge (advance to OFFSEASON → buyout succeeds; in REGULAR_SEASON → rejected;
  3rd buyout in a season → rejected; out-of-range remaining → rejected).
- Review = GLM 5.2 BLIND on the Beelink. Leads, not findings.

## Expert-panel outcomes — LOCKED 2026-07-10

Panel: GLM 5.2 · Gemini · DeepSeek (parallel, blind). Full responses + triage:
`docs/build-handoffs/reviews/b7c-buyout-panel.md`. **All three independently converged on the
season-int lifecycle as the critical must-fix.** Amendments to the design above:

**A1 — SEASON/PHASE INVARIANT (the unanimous must-fix), locked:** *the loaded `season` int is the
season the OFFSEASON belongs to — offseason sits at the START of its season's lifecycle.* Cycle:
`OFFSEASON(N) → REGULAR_SEASON(N) → PLAYOFFS(N) → [rollover to N+1] → OFFSEASON(N+1)`. So a buyout
in OFFSEASON(N) counts against N's `transaction_counts`, charges dead cap to N (the upcoming managed
season it clears cap for), and computes `remaining = expiration − N`. **Season-rollover machinery is
NOT built this session** — the PLAYOFFS→OFFSEASON increment stays the existing carry-forward (already
noted for §11's in-season restructure unlock). v1 correctness holds because the fresh-DB seed places
us in OFFSEASON at the loaded season int.

**A2 — Fresh-DB seed inserts a REAL transition row** `(from=NULL, to=OFFSEASON, note='seed')`.
`CurrentPhase()` reads the latest row and has NO hardcoded fallback — the table is the sole source of
truth (GLM: avoid the split-source-of-truth). Empty table = fail loud (should never occur post-seed).

**A3 — Final buyout charge takes ONE `domain.RoundToNearest10k` snap** (flat-$10k doctrine), same as
the §8 `Charge`. Not a per-step cent round. (The house helper is already half-up-away-from-zero, so the
panel's `math.Round` banker's-rounding concern does not apply.)

**A4 — DB-level UPDATE/DELETE `RAISE(ABORT)` triggers on `season_phases`** — parity with the dead-cap
ledger (all three panelists; GLM escalated the omission to a design-consistency defect). Added at table
creation, zero migration cost.

**A5 — `meta` TEXT/JSON column on `season_phases`** (nullable) so later finer phases carry granularity
(e.g. `{"week":9}`) without a schema change. (DeepSeek.)

**A6 — Document, don't guard (v1 postures):** (i) the 90% rate is only reachable on a §10-extended
contract — a normal 4-yr deal bought in year 1 has remaining≤3 → ≤75%; (ii) phase rollback does NOT
auto-reverse `transaction_counts` (commissioner-trusted manual correction — per-season ceiling of 2
bounds any abuse). Both in-code comments.

**Refuted vs source (NOT actioned):** §8 dead-cap spreading (lands whole), §8 waiver/bidding inheritance
(no claim path in v1), player↔buyout link scrubbed (`DeadCapEntry.MFLID` preserves it), voided-cells =
row-delete (`VoidCells` zeroes cap + preserves history). See the panel review doc.

**Buyout does NOT reuse `deadcap.Charge`** — that is the §8 formula (`35% × salary × remaining`). §12 has
its own charge (`avgRemaining × rate`); it reuses only the WRITE path (`AddDeadCap` + `ReleasePlayer` +
`VoidCells`).

## Test matrix (from the panel, pin before merge)
- Phase gate: op allowed/rejected per phase; an op_kind absent from the map is default-DENIED (planted).
- Buyout: rate table 2→60/3→75/4→90 every fencepost; `remaining < 2` and `> 4` both rejected; charge
  snaps to $10k; dead cap lands in season N; 3rd buyout in a season rejected; player released + cells voided.
- ADVANCE_PHASE: no-op (`to == current`) rejected; a rolled-back tx leaves NO partial `transaction_counts` bump.
- Season invariant: `remaining` and the dead-cap `LeagueYear` both computed off the loaded season N.

## Build plan (next)
Branch `session/b7c-buyout` off clean main. Suggested commit order:
1. `season_phases` table + triggers + `meta` col + seed row; `Phase` enum; `TxWriter.CurrentPhase()`.
2. `AdvancePhase` sealed Request + Coordinator wiring (no-op reject).
3. Static `map[Kind][]Phase` gate at the top of `Execute`'s WriteTx closure, default-deny (planted test).
4. `Buyout` sealed Request + handler (avgRemaining × rate, edge-reject, $10k snap, count limit, release+void).
5. IPC branch + React dev control (ADVANCE_PHASE control + Buyout control).
6. GLM 5.2 blind review of the diff → functional gate live on the Beelink → merge.
