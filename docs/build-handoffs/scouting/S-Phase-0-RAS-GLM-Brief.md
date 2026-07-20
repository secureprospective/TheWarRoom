# GLM CHUNK BRIEF — Scouting Wire-Up, S-Phase 0: RAS end-to-end

**Project:** TheWarRoom · Go · module `github.com/secureprospective/TheWarRoom` · Go at `/usr/local/go/bin/go`
**Branch to work on:** `session/scouting-sphase0-ras` (already cut off main `eafd379`)
**Your role:** You (GLM 5.2) WRITE this code. Claude is head-brain and will review every line against source before anything merges. Write the plumbing to the method pinned in §3 EXACTLY — **do not invent, "improve," or deviate from the normalization math.** If something in this brief is ambiguous or contradicts the code you see, STOP and say so in your output rather than guessing — flag it as a question, do not fill the gap.

---

## 1. Mission (one sentence)

Wire the RAS scouting signal end-to-end for the first time: raw combine measurables → per-position RAS-equivalent (0–10) → `scouting.Profile.RAS` → `composition.PlayerSpec.RAS/HasRAS` → the already-built engine RAS modulator → the M1 board. This is the **first-instance TEMPLATE**; every later signal clones its assembly shape, so correctness and clean seams matter more than speed.

## 2. The verified gap (read these first, confirm against source)

- `internal/scouting/types.go` — `Profile` (keyed by MFL `playerid.PlayerID`) has a flat `RAS float64` field. **`scouting.Profile` is constructed NOWHERE in the codebase today** (only named in fetcher comments). You will construct it for the first time.
- `internal/ingestion/ras/fetcher.go` — `Fetch(ctx, client, url, pfrToGSIS map[string]string) (map[string]RawCombine, error)`. Returns **raw combine measurables** keyed by **gsis_id**: `RawCombine{ GSISID string; HeightIn, WeightLb, Forty, Bench, Vertical, BroadJump, Cone, Shuttle *float64 }`. A `*float64` is nil when the player did not perform that drill (ABSENT ≠ 0 — never treat nil as 0). It does NOT compute a RAS number. Producing the single 0–10 RAS is YOUR job here (§3).
- `internal/ingestion/crosswalk/fetcher.go` — `Fetch(ctx, client, url) (Map, error)`; `Map.Lookup(mflID) (gsis string, ok bool)` and `Map.PFRMap() map[string]string` (pfr→gsis, a defensive copy). `crosswalk.SourceURL` is the CSV address. Use `PFRMap()` to feed the `ras` fetcher, and `Lookup` to join a rostered MFL id to its gsis.
- `internal/composition/playerspec.go` — `PlayerSpec.RAS float64` + `HasRAS bool` already exist. `Validate()` already fail-louds on `HasRAS && !finite(RAS)`. When `HasRAS` is false the assembler zeroes RAS and L1 imputes `DefaultRASFallback = 5.00` (`internal/composition/defaults.go`). **Do not touch the engine or the assembler's RAS math** — the consumer is done and harness-tested.
- `internal/rankings/rankings.go` — `Runner` holds ports incl. `dir Directory`; `New(...)` nil-guards every dependency. `scorePlayer` (~L218-230) builds `PlayerSpec` and leaves every scouting `Has*` false (see the comment at L228-229). This is the injection point.
- `m1_app.go` `ScoreLeague()` (~L96-108) — builds `lk` (directory), `base`, `asm`, then calls `rankings.New(...)`. This is where the profile map gets built and the new port injected.
- Engine RAS consumer (reference only, DO NOT MODIFY): `internal/engine/l4/offense/rb.go` uses `curve.Scurve(p.RAS/10.0, ...)` — RAS is on a **0–10 scale**, fallback 5.0 is neutral. Harness coverage: `internal/harness/cases_eval.go` 3A/3B.

## 3. THE PINNED RAS v1 METHOD (Christopher-approved 2026-07-20 — implement EXACTLY, do not deviate)

This is a **provisional** v1 mapping. The real z-score calibration is a SEPARATE, deferred, decision-gated pass — so label it provisional in code comments, but implement it precisely as written.

Given the full set of rostered, scored players and their `RawCombine` (joined to MFL id + resolved position):

1. **Cohort = resolved position.** For each position, the cohort is all players at that position who have ≥1 measurable present. Position comes from the players-DB `Directory` (the same `normalize.PlayerFacts.Position` rankings already uses). Use the RUBRIC-resolved position if available, else the players-DB position (v1: players-DB position is fine — EDGE routing is a later concern).
2. **Per measurable, per cohort** — compute mean μ and **sample** standard deviation σ (n−1 denominator) over the PRESENT values only.
   - **Sign convention (higher z = better athlete):**
     - Time drills — `forty`, `cone`, `shuttle`: `z = (μ − x) / σ` (lower time is better).
     - All others — `vertical`, `broad_jump`, `bench`, `weight`, `height`: `z = (x − μ) / σ` (bigger is better). *(Crude size handling is a known v1 simplification; real per-position size treatment is deferred.)*
3. **Per player** — `RAS_z = mean of that player's AVAILABLE measurable z-scores` (average only over measurables the player actually has). A player with **zero** measurables present → `HasRAS = false` (do NOT emit a RAS; L1 imputes 5.0 downstream).
4. **Scale to 0–10** — `RAS = clamp(5.0 + 2.0 * RAS_z, 0.0, 10.0)`. Cohort-average athlete → 5.0 (== neutral fallback); +2.5σ → 10.0; −2.5σ → 0.0.
5. **σ = 0 or n < 2 guard** — if a measurable's cohort σ is 0 (all present values identical) or the cohort has n<2 present, that measurable contributes **z = 0** (neutral), NEVER NaN/Inf. If ALL of a player's available measurables resolve to z=0 → `RAS_z = 0 → RAS = 5.0`, `HasRAS = true` (present-but-neutral — documented, distinct from absent).

**Determinism:** the output must be deterministic for a given input set (no map-iteration-order dependence in the math). **No NaN/Inf may ever reach `Profile.RAS`** — the composition boundary will reject it, but you must not produce it.

## 4. What to build

### 4a. New assembly leaf package — `internal/scouting/assembly` (or `internal/scoutasm` if depguard prefers; see §5)
A composition-class assembler that imports `internal/ingestion/ras`, `internal/ingestion/crosswalk`, `internal/scouting`, `internal/domain`, `internal/playerid`, and `internal/numeric`. It does NOT import the engine, any store, or normalize's write side. Suggested surface (adjust names for idiom, keep the shape):

```go
// PositionLookup resolves a rostered MFL id to its position (players-DB fact).
// A narrow injected port so the assembler is fake-testable with no live DB.
type PositionLookup interface {
    Position(mflID string) (domain.Position, bool)
}

// BuildRAS fetches combine measurables + the crosswalk, joins each rostered MFL
// id to its gsis, computes the per-position RAS-equivalent (§3), and returns a
// map[mflID]scouting.Profile with ONLY RAS/HasRAS populated (every other field
// zero/absent this phase). rosterMFLIDs is the set of ids to score.
func BuildRAS(ctx context.Context, client *http.Client, combineURL, crosswalkURL string,
    rosterMFLIDs []string, pos PositionLookup) (map[playerid.PlayerID]scouting.Profile, error)
```
- Fetch crosswalk once → `PFRMap()` → feed `ras.Fetch`. For each roster MFL id: `crosswalk.Lookup` → gsis → look up `RawCombine` by gsis. Cohort by `pos.Position(mflID)`.
- Two-pass: pass 1 accumulate per-position per-measurable μ/σ over present values; pass 2 compute each player's RAS_z → RAS.
- A player with no gsis, no combine row, or zero measurables → NOT in the map (or in the map with `HasRAS=false` — pick one, document it; the rankings side treats "absent from map" and "HasRAS=false" identically). Prefer: **absent from the map** = clean miss.
- Fail loud only on a genuine fetch failure (network/parse). A player-level miss is ordinary, never an error.

### 4b. `rankings.Runner` — new scouting Directory port
- Add an interface, e.g. `ScoutingDirectory interface { Profile(mflID string) (scouting.Profile, bool) }`, and a `scout ScoutingDirectory` field on `Runner`.
- Add it as a REQUIRED param to `New(...)` and to the nil-guard (mirror the existing guard exactly — a nil scouting dir is a wiring error, an explicitly-empty map is legal).
- Provide a tiny concrete impl over `map[playerid.PlayerID]scouting.Profile` (a map-backed lookup) — this is what the app wires. Keep it in the assembly package or rankings; your call, document it.
- In `scorePlayer`: after building `spec`, look up the profile; if present and `HasRAS`, set `spec.RAS = profile.RAS; spec.HasRAS = true`. If absent, leave `HasRAS=false` (unchanged behavior — L1 imputes 5.0). Update the L228-229 comment to reflect that RAS now flows.

### 4c. `m1_app.go` `ScoreLeague()` — wire the build
- After `lk` is built, build the profile map: call `assembly.BuildRAS(...)` with a `PositionLookup` backed by `lk` (adapt the existing `directory`/`Lookup` to the `Position(mflID)` shape), pass `crosswalk.SourceURL` and `ras.SourceURL`, the roster MFL id set, and the shared `http.Client`/timeout already in scope.
- Pass the resulting map-backed `ScoutingDirectory` into `rankings.New(...)`.
- On a BuildRAS error: surface it like the other `ScoreLeague` error returns (don't silently score a RAS-less league — a failed scouting fetch should be visible, matching the app's fail-loud posture). If Christopher later wants graceful degradation that's a separate decision — for v1, surface it.

## 5. Constraints — NON-NEGOTIABLE

- **depguard (build errors):** engine stays pure (never import your new package). Your assembly leaf must NOT import `internal/engine`, `internal/store/*`, `internal/transactions`, `internal/output`, or `database/sql`. Note: `internal/ingestion/**` is under `layer1-no-upward-import` — so your package must live OUTSIDE `internal/ingestion/` (it imports ingestion, it isn't ingestion). `internal/scouting/assembly` or a new top-level `internal/scoutasm` both work; verify `go build ./... && make lint` passes and pick whichever the depguard rules accept cleanly. Do NOT edit `.golangci.yml` to make room — if a boundary blocks you, that's a design signal; flag it.
- **Standards:** 400-line file cap (split if needed); no globals (`gochecknoglobals`); parameterized SQL only (n/a here); typed IPC — no `interface{}`/`any` at exported boundaries (`ifaceguard` enforces); wrap every returned error with `fmt.Errorf("...: %w", err)` (`wrapcheck`); use `internal/numeric.Finite` for finite checks (don't hand-roll). No `fmt.Print*` (`forbidigo`).
- **ZERO-LEAK invariant** (`internal/scouting` package doc): no scouting field may hold fantasy points, projected volume, or MFL scoring config. RAS is pure athletic testing — structurally clean here; keep it that way.
- **Do NOT modify:** the engine, `composition.Assembler`'s RAS math, the `ras`/`crosswalk` fetchers, or the harness assertions — the consumer side is built and tested. (If a harness RAS assertion genuinely must move because real RAS now flows, FLAG it — do not edit it silently. For S-Phase 0 the harness uses synthetic RAS via PlayerInput, so it should NOT need to change.)

## 6. Tests you must write

- **Assembly unit tests (no network):** feed a fixture set of `RawCombine` + positions and assert the RAS math EXACTLY per §3 — including: sign convention (a fast forty → high RAS; a slow forty → low RAS), the σ=0 guard (identical cohort → RAS 5.0, HasRAS true), n<2 guard, a zero-measurable player → absent/HasRAS false, and the `clamp(5.0 + 2.0z, 0, 10)` rails. Pin at least one hand-computed numeric value end-to-end so a math regression is caught (the "prove the number through" discipline).
- **rankings port:** a fake `ScoutingDirectory` → assert `scorePlayer` sets `spec.RAS/HasRAS` when present and leaves HasRAS false when absent; assert `New` nil-guards the new port.
- Determinism: same input → same output across runs.
- `go test -race ./...` green including the new package.

## 7. Close gate (Claude verifies; you report)

- `go build ./...` clean; `make lint` 0 (run `GOMEMLIMIT=1500MiB GOGC=20` if on a small box); `go test -race ./...` green.
- Report: files added/changed, the exact RAS formula as implemented, your hand-computed test vector, and any place you had to make a judgment call (flag each — Claude triages them leads-not-findings).
- **Commit to THIS session branch and push it** (`git add -A && git commit && git push`). **Do NOT merge to main and do NOT move any gate** — Claude reviews your diff against source, owns the merge-to-main and the live functional gate on the Beelink.

## 8. Reference: exact current shapes (verified 2026-07-20)

- `scouting.Profile{ MFLID playerid.PlayerID; ... RAS float64; ... }` (types.go).
- `ras.RawCombine{ GSISID string; HeightIn, WeightLb, Forty, Bench, Vertical, BroadJump, Cone, Shuttle *float64 }`; `ras.SourceURL`; `ras.Fetch(ctx, *http.Client, url string, pfrToGSIS map[string]string) (map[string]RawCombine, error)`.
- `crosswalk.SourceURL`; `crosswalk.Fetch(ctx, *http.Client, url) (Map, error)`; `Map.Lookup(playerid.PlayerID)(string,bool)`; `Map.PFRMap() map[string]string`.
- `composition.PlayerSpec{ ... RAS float64; HasRAS bool; ... }`; `composition.DefaultRASFallback = 5.00`.
- `rankings.New(st state.Reader, dir Directory, base map[string]float64, cfg ConfigSource, out output.Writer, asm *composition.Assembler, reg Registry) (*Runner, error)` — you ADD one param.
- `playerid.PlayerID` is a struct with an unexported field; construct via `ingestion.ValidatePlayerID(string)` or the `playerid` constructor — never a raw conversion.
