# TheWarRoom Go Backend — Read-Only Review

## 1. CODEBASE MAP

| Package | Responsibility |
|---|---|
| `main` (root `*.go`) | Wails adapter: wires deps, routes IPC, owns process-lifetime resources |
| `internal/domain` | Leaf domain value types (Position, Money, Phase, PlayerStatus, PlayerRecord) |
| `internal/playerid` | Validated MFL player ID struct-wrapper (RISK-003 enforcement) |
| `internal/scouting` | Leaf scouting-input types (Profile, SchoolTier, SafetyRole) |
| `internal/schema` | Hand-written boundary validation for external/IPC JSON |
| `internal/engine` | Pure scoring pipeline (L1→L6), no store/db imports |
| `internal/engine/l4/curve` | Shared L4 normalization primitives (Scurve, Interp, SL019) |
| `internal/engine/l4/offense` | QB, RB, WR, TE Layer-4 rubrics |
| `internal/engine/l4/defense` | DT, DE, LB, CB, S Layer-4 rubrics |
| `internal/engine/l4/kicker` | K Layer-4 rubric (DECISION-011 Madden-driven) |
| `internal/composition` | Boundary: reads stores + harness data, assembles engine inputs |
| `internal/db` | SQLite split read/write pool (single-writer enforcement) |
| `internal/ingestion` | Layer-1 shared plumbing (MFL decode, CSV/CFBD helpers, ID validation) |
| `internal/ingestion/*` | Individual Layer-1 fetchers (players, rosters, league, scouting feeds) |
| `internal/mfl` | MFL HTTP transport client (rate limit, host discovery, 429 backoff) |
| `internal/normalize` | Raw→domain boundary: joins rosters with players DB |
| `internal/store/params` | B4 admin calibration parameter store |
| `internal/store/rulebook` | B3b league rulebook (versioned config + commissioner overrides) |
| `internal/store/state` | B3c mutable runtime state (rosters, contracts, ledger, cap) |
| `internal/output` | B6 per-season engine output (double-immutable score store) |
| `internal/harness` | Testing sandbox: 13-case validation suite + rookie ranking flow |
| `internal/transactions` | B7a transaction coordinator + handler packages |
| `internal/rankings` | M1 scoring orchestrator (scores league, persists to B6) |
| `internal/powerrankings` | M2 blend math (z-score normalization, weighted blend) |

**Dependency spine:**
```
main → {db, store/*, output, transactions, rankings, composition, powerrankings, ingestion/*, normalize, mfl}
transactions → {store/state, normalize, domain} + {acquisitions, contracts, deadcap, freeagency}
rankings → {composition, engine, normalize, output, store/state}
composition → {engine, domain, scouting, store/params}
engine → domain only
engine/l4/* → {engine, curve}
ingestion/* → {ingestion, mfl}
normalize → {domain, ingestion/players, ingestion/rosters, playerid}
```

**Divergence from stated purpose:** `internal/scouting` declares itself the "single unified field set every scouting fetcher populates and the engine reads," but no code in the provided source produces or consumes `scouting.Profile` — the engine reads `engine.ScoutingInput` (built by composition from `composition.PlayerSpec`), not `scouting.Profile`. The package's `SchoolTier` enum is consumed (by composition and harness), but the `Profile`, `OffenseFilm`, `IDPFilm`, and `NGSCoverage` types are unwired scaffolding (see §4).

---

## 2. DEAD CODE (semantic — beyond staticcheck U1000)

**`internal/ingestion/extcsv.go:61` — `StreamCSVGz`**
- (a) Exported function, gzip-streaming CSV reader.
- (b) No caller in the provided source. `pfrcoverage` uses `FetchCSVGz` (buffered); `veteranfilm` uses `StreamCSV` (non-gz). The gz-stream variant was built for a consumer that never arrived.
- (c) **hi**
- (d) `grep -rn 'StreamCSVGz' --include='*.go'` across the repo; confirm the only hit is the definition + its test (if any).

**`internal/scouting/types.go:14` — `Profile`, `OffenseFilm`, `IDPFilm`, `NGSCoverage` types**
- (a) Four struct types (~70 lines total) defining a unified scouting field set.
- (b) Zero producers and zero consumers in the provided source. The composition boundary builds `engine.ScoutingInput` from `composition.PlayerSpec`, not from `scouting.Profile`. No fetcher returns a `Profile`. The rankings orchestrator states "no scouting fetcher is wired into the orchestrator yet." These types are entirely orphaned.
- (c) **hi**
- (d) `grep -rn 'scouting\.Profile\|scouting\.OffenseFilm\|scouting\.IDPFilm\|scouting\.NGSCoverage' --include='*.go' | grep -v '_test.go' | grep -v 'internal/scouting/types.go'`; confirm zero hits.

**`internal/scouting/constants.go:33-40` — `SafetyRole` type and its four constants**
- (a) Enum + 4 constants reserved for SL-OQ-035/036 safety role branching.
- (b) No code reads or sets a `SafetyRole` value. `Profile.SafetyRole` (the only field that holds one) is itself on the orphaned `Profile` type.
- (c) **hi**
- (d) `grep -rn 'SafetyRole' --include='*.go' | grep -v '_test.go' | grep -v 'internal/scouting/'`; confirm zero hits.

**`internal/schema/schema.go:50` — `RawPlayerRecord.Validate()`**
- (a) Exported validation method on `RawPlayerRecord`.
- (b) `DecodePlayerRecord` is called by harness eval3L, but `Validate()` is never invoked — eval3L reads `rec.ID` directly after decode. No other caller of `DecodePlayerRecord` or `Validate` exists in the provided source.
- (c) **med** — could be a genuine public seam intended for future callers; the package doc presents Validate as the pattern.
- (d) `grep -rn '\.Validate()' --include='*.go' | grep schema`; confirm zero hits outside schema's own definition.

**`internal/engine/l4/defense/dt.go:133` — `DT.PFFAlpha()`**
- (a) Exported introspection method returning the DT dynamic PFF blend rate.
- (b) The comment says it is "exposed for the harness only," but case 3G (the case that would call it) is still `pendingSubSignals` and never calls `PFFAlpha`. No other caller in the provided source.
- (c) **med** — it is a deliberate harness seam for case 3G; calling it dead depends on whether 3G's wiring is "imminent" vs "indefinitely deferred."
- (d) `grep -rn 'PFFAlpha' --include='*.go' | grep -v '_test.go' | grep -v 'func (d \*DT) PFFAlpha'`; confirm zero hits.

**`internal/store/state/writes.go:185-196` — Standalone mutators `MovePlayer`, `SetRosterStatus`, `ApplyContract` on `*Store`**
- (a) Three public methods that each wrap a one-op `WriteTx`.
- (b) The comment says they remain "for the direct/admin path and the store's own tests," but no production code calls them — all mutations flow through `Coordinator.Execute` → `WriteTx(fn)` → `txWriter`. They appear test-only.
- (c) **med** — they may be called from test files not in the provided source.
- (d) `grep -rn '\.MovePlayer\|\.SetRosterStatus\|\.ApplyContract' --include='*.go' | grep -v '_test.go' | grep -v 'func (s \*Store)'`; confirm whether any production caller exists.

---

## 3. DUPLICATE / NEAR-DUPLICATE CODE (semantic — beyond dupl)

**The 10 L4 position rubrics (`internal/engine/l4/offense/*.go`, `internal/engine/l4/defense/*.go`, `internal/engine/l4/kicker/k.go`)**
- dupl already flags the token-identical blocks (cb≈s, de≈te, lb≈rb≈wr). The **semantic** near-dup is the entire `Apply` method shape shared by 9 of 10 rubrics: (1) film Scurve-or-neutral, (2) RAS Scurve scaled by rookie weight, (3) weighted breakout composite, (4) optional SL-019 or SL-021 modulator, (5) `Combined = film × ras × breakout`. Only K deviates (forced RAS/breakout = 1.0).
- **Unify:** a single data-driven `Rubric` struct holding all the per-position constants (inflection, steepness, cap, weights, SL-019 strength, cushion params) plus an `Apply` method with two hooks: `modulateBreakoutAge(breakoutAgeNorm, rasNorm)` (no-op for QB/WR/DT/LB, SL-019 for TE/DE/CB/S) and `modulateAgeTrajectory(ageTraj, ras)` (SL-021 for DT, SL-019 for SL-019 positions, identity for others). This would collapse ~1500 lines to ~300 plus a constants table.
- **Risk:** lo — the dupl output proves the structure is identical; the only per-position variation is constants + which modulator applies.

**`finite()` triplicated: `internal/engine/finite.go:8`, `internal/composition/playerspec.go:139`, `internal/output/helpers.go:121`**
- Three identical package-private functions (`math.IsNaN || math.IsInf`).
- **Unify:** move to `domain.Finite()` or a tiny `internal/finite` package. Saves ~12 lines and eliminates three copies of the same guard.
- **Risk:** lo.

**`boolToInt()` triplicated: `internal/store/state/helpers.go`, `internal/store/params/helpers.go`, `internal/output/helpers.go`**
- Three identical `func boolToInt(b bool) int` implementations.
- **Unify:** one shared helper. Saves ~9 lines.
- **Risk:** lo.

**`loadDeadCap` (state.go) / `loadCapRelief` (cap_relief.go) — near-identical query+scan+sum shape**
- dupl flags cap_relief.go:109-133 ≈ state.go:372-396. Both run `SELECT franchise_id, COALESCE(SUM(...), 0) FROM <ledger> WHERE league_id=? AND league_year=? GROUP BY franchise_id` then scan into `map[string]domain.Money`.
- **Unify:** one generic `sumLedgerByFranchise(ctx, table, centsCol string) (map[string]domain.Money, error)` helper.
- **Risk:** lo — the SQL differs only in table/column names.

**`parseStanding` / `atoiOrZero` / `atofOrZero` in m2_app.go mirror `SanitizeNumeric` + validation already done in leaguestandings**
- `leaguestandings.RawStanding.Validate()` already parses every numeric field; `m2_app.go:parseStanding()` re-parses the same fields with the same `SanitizeNumeric` call. The fetcher validates shape; the adapter re-parses for values. This is not just duplication — it's the same parsing logic in two layers.
- **Unify:** have the fetcher (or a normalize step) produce typed numeric fields once, or push `parseStanding` into a shared M2 helper package.
- **Risk:** med — the fetcher's contract is "transform nothing," so re-parsing at the consumer is architecturally consistent, just redundant.

---

## 4. STUBS WITHOUT A DESTINATION

**`internal/scouting/types.go` — entire `Profile`/`OffenseFilm`/`IDPFilm`/`NGSCoverage` type hierarchy**
- Described in the package doc as the "single unified field set every scouting fetcher populates and the engine reads." In reality: no fetcher produces it, no code consumes it. The engine's `ScoutingInput` and composition's `PlayerSpec` are the actual field sets used. The package comment itself flags fields as "retained pending Film-component redesign … populated by no fetcher today." This is orphaned scaffolding whose consumers were redesigned away.
- **Resolve:** confirm whether a future session still intends to route scouting data through `scouting.Profile`, or whether `engine.ScoutingInput` has permanently replaced it. If the latter, the types should be deleted.

**Eleven scouting fetchers built but unwired**
- `agetrajectory`, `collegeshare`, `collegedefense`, `crosswalk`, `kicking`, `madden`, `nflproduction`, `pfrcoverage`, `ras`, `touchshare`, `veteranfilm` — all have complete `Fetch()` implementations with tests (per the dupl output referencing their `_test.go` files) but no production caller in the provided source. The rankings orchestrator explicitly states scouting sub-signals are "Data-Parity ABSENT for every player this session (no scouting fetcher is wired into the orchestrator yet)."
- These are documented deferrals (each has extensive package-level rationale), not accidental orphans. But they represent ~2000+ lines of unwired code. The question is whether they are "next sprint" or "indefinitely deferred."

**`internal/ingestion/salaryadjustments/`, `internal/ingestion/schedule/` — league fetchers with no caller**
- Both clone the rosters template pattern but neither is called from any production path in the provided source. The salary-adjustments cap data is modeled via the `dead_cap_ledger` (populated by transactions, not by MFL ingest). The schedule fetcher has no consumer.
- **Resolve:** confirm whether these feed a future refresh/sync layer or are dead templates.

**`internal/ingestion/cfbd.go:31` — `NewCFBDClient`**
- Exported factory returning an HTTP/1.1-pinned client. The three CFBD fetchers (collegeshare, collegedefense, schooltier) take an `*http.Client` as a parameter but no caller in the provided source invokes `NewCFBDClient` to create one. It was extracted for reuse but its caller (the orchestration layer that wires CFBD fetchers) does not exist yet.
- **Resolve:** confirm whether a future orchestration session will call it, or if it's prematurely extracted.

---

## 5. SLIMMING OPPORTUNITIES

**Ranked by lines saved vs risk:**

1. **L4 rubric data-driven consolidation** — ~1200 lines saved, **lo** risk.
   - `internal/engine/l4/{offense,defense,kicker}/*.go`: 10 files × ~130 lines each = ~1300 lines of near-identical Apply methods. Replace with one `Rubric` struct + one `Apply` method + a constants table per position. The dupl output proves the structure is token-identical; only constants differ.
   - Check: run `diff` on any two Apply methods (e.g., cb.go:Apply vs s.go:Apply) and confirm the only differences are the constant names referenced.

2. **m2_app.go business logic extraction** — ~250 lines moved, **med** risk.
   - `m2_app.go` holds `parseStanding`, `aggregateScouting`, `buildBlendInputs`, `buildPowerRows`, `clampWeight`, `atoiOrZero`, `atofOrZero`, `resolveAggMode`, `starterCount`, `franchiseDisplayName` — substantial business logic in the adapter layer. Extract to an `internal/m2` (or extend `internal/powerrankings`) orchestrator package, leaving m2_app.go as a thin IPC wrapper (like `rankings.Runner` for M1).
   - Check: compare m2_app.go line count (~280 lines of logic) vs m1_app.go (~120 lines, most delegating to `rankings.Runner`).

3. **finite()/boolToInt() consolidation** — ~25 lines saved, **lo** risk.
   - Three identical copies of each across engine/composition/output and state/params/output. Move to a shared leaf package.
   - Check: `grep -rn 'func finite\|func boolToInt' --include='*.go'`.

4. **loadDeadCap/loadCapRelief generic helper** — ~30 lines saved, **lo** risk.
   - Two functions with identical query/scan/sum shape differing only in table and column names.
   - Check: diff the two function bodies.

5. **`Store.LedgerCells` on state.Store** — ~20 lines, **lo** risk.
   - `internal/store/state/ledger.go:62`: public method with no production caller (comment says "lets tests and an admin audit verify the ledger directly"). If it's test-only, move to a test helper.
   - Check: `grep -rn 'LedgerCells' --include='*.go' | grep -v '_test.go'`.

6. **Unused `scouting.Profile` type hierarchy** — ~70 lines, **lo** risk.
   - If confirmed as permanently superseded by `engine.ScoutingInput`, deleting `Profile`, `OffenseFilm`, `IDPFilm`, `NGSCoverage` removes dead type definitions and their "retained pending redesign" fields.
   - Check: confirm no planned consumer exists.

---

## 6. CODING-STANDARDS ADHERENCE

**Three-layer law — m2_app.go holds business logic (VIOLATION)**

`m2_app.go:135-166` (`buildBlendInputs`), `m2_app.go:169-185` (`aggregateScouting`), `m2_app.go:188-259` (`parseStanding`), `m2_app.go:262-275` (`atoiOrZero`/`atofOrZero`), `m2_app.go:278-284` (`clampWeight`), `m2_app.go:287-296` (`resolveAggMode`), `m2_app.go:299-305` (`starterCount`).
- **Rule violated:** "the Wails `App` is a THIN adapter — it wires dependencies and routes IPC calls; business logic lives in `internal/engine`, the `internal/store/*` packages, and `internal/transactions`."
- **Why:** M1 follows the law — `m1_app.go` delegates to `internal/rankings.Runner`. M2 does not — standings parsing, per-franchise scouting aggregation, weight clamping, and mode resolution all live in the adapter. `internal/powerrankings` holds only the blend math.
- **Confidence:** **hi**
- **Check:** compare `m1_app.go` (delegates to `rankings.New(...).Run(...)`) vs `m2_app.go` (implements the full orchestration inline). The ~250 lines of m2_app.go logic should be in a service package.

**Scouting Profile / engine.ScoutingInput divergence — undocumented seam**

`internal/scouting/types.go:14` defines `Profile` as the "single unified field set every scouting fetcher populates and the engine reads." But the engine reads `engine.ScoutingInput` (types.go:99), and composition builds `ScoutingInput` from `composition.PlayerSpec` (playerspec.go), not from `Profile`. The stated purpose of `scouting.Profile` and its actual role (unused) diverge.
- **Confidence:** **hi**
- **Check:** `grep -rn 'scouting\.Profile' --include='*.go' | grep -v '_test.go' | grep -v types.go`; confirm zero consumers.

**`finite()` duplicated across three packages — implicit boundary leak**

`internal/engine/finite.go`, `internal/composition/playerspec.go:139`, `internal/output/helpers.go:121`.
- **Rule:** "shared logic extracted on the second use" (the codebase's own M17 principle, cited throughout). Three copies of the identical NaN/Inf guard is a direct M17 violation.
- **Confidence:** **hi**
- **Check:** `diff` the three function bodies — they are character-identical.

**`boolToInt()` duplicated across three packages — same M17 violation**

`internal/store/state/helpers.go`, `internal/store/params/helpers.go`, `internal/output/helpers.go`.
- **Confidence:** **hi**
- **Check:** identical 4-line bodies.

**Unwrapped errors — consistent adherence (NO VIOLATION FOUND)**

Spot-checked error wrapping across `db/pools.go`, `mfl/client.go`, `store/state/*.go`, `transactions/*.go`, all root app files — every error return uses `fmt.Errorf("...: %w", err)`. No bare `fmt.Errorf("...: %v", err)` or swallowed errors found.
- **Confidence:** **hi**

**Typed IPC boundary — consistent adherence (NO VIOLATION FOUND)**

Every IPC method returns a typed struct (`PingResult`, `RookiesResult`, `ScoreLeagueResult`, `RankingsResult`, `PowerRankingsResult`, `TransactionResult`, `RosterResult`, etc.). `TransactionRequest` is fully typed with `Kind` discriminator + named fields. No `any`/`interface{}` at the Go→JS boundary. The only `interface{}` usage is `main.go:33` (`Bind: []interface{}{app}`), which is the Wails framework API, not a bridge type.
- **Confidence:** **hi**

**Single-writer law — consistent adherence (NO VIOLATION FOUND)**

`app.go:131` wires `transactions.New(st.Writer())` exactly once, in `initStoreFloor`. The `state.Writer()` method is called only there. The `txWriter` type (the mutation surface inside a transaction) is never exposed outside `WriteTx`'s callback. The `readerView` wrapper prevents type-assertion back to `Writer`.
- **Confidence:** **hi**

**depguard engine-is-pure — consistent adherence (NO VIOLATION FOUND)**

`internal/engine` imports only `domain`. `internal/engine/l4/curve` imports only `math`. `internal/engine/l4/{offense,defense,kicker}` import only `engine` and `curve`. No store/db/ingestion import reaches the engine.
- **Confidence:** **hi**

**Ledger is KING — consistent adherence (NO VIOLATION FOUND)**

`domain.Money` is int64 cents throughout. `RoundToNearest10k` is applied at aggregation time (`loadCellCap` snaps per-cell at the franchise sum; dead-cap charges snap in `deadcap.Charge`). No rounding-at-write of base cells. Seed writes `RoundToNearest10k` on the flat-fill salary, which is documented as the "universal rule" applied once at birth. This matches the documented model.
- **Confidence:** **hi**