# Fable — TheWarRoom Code Companion Plan
**Version:** 1.0 — 2026-06-12
**Author:** Claude Fable 5 (orchestrating 4 read-only Haiku fetch agents + 2 agy cross-reference agents on CT 104)
**Status:** ADVISORY COMPANION. Additive. Rewrites nothing.

---

## What this document is, and how to use it

This is a **field guide for the models that will write TheWarRoom's code** across the 38 sessions in `docs/build-handoffs/Build_Tracker.md`. It keeps weaker models on the rails: copy-ready skeletons that already satisfy the strict standards, session-by-session traps flagged, and an answer to *"I am a model in the middle of session N — how do I do this well, in idiomatic Go, without breaking the architecture?"*

**It does not replace anything.** The build still runs off:
- `docs/build-handoffs/Build_Tracker.md` — the 38-session sequence (the *what* and *when*).
- `session-3-audit-build-sequencing.md` — the locked decisions AD-01..AD-25 (the *why*).
- The six structural wireframes — the module shapes (the *anti-spaghetti contracts*).
- `christopher-coding-standards/AGENTS.md` — the rules every diff must satisfy.
- This project's `CLAUDE.md` Hard Constraints.

**How a building model should use it:**
1. At session start, read the matching **per-session brief** in Section 6.
2. Before writing a file, copy the matching **Go skeleton** from Section 4 — do not invent structure.
3. Keep the **Never-Hallucinate constants** (Section 5) open; every domain number lives there.
4. Run the **pre-flight and pre-commit checklist** (Section 7) every session, no exceptions.

A note on tone for the reader-model: when this guide and a hook disagree, **the hook wins** — fix the violation, never bypass it. When this guide and a locked decision (AD-xx) disagree, **the decision wins** — flag the conflict to Christopher, do not route around it. This guide is the most junior voice in the room. It is here to help you, not to override the people above it.

---

## The five things that will sink a weaker model — read these first

These are the highest-leverage findings from cross-referencing every planning artifact, the architecture docs, and the coding standards. If a building model internalizes only five things, these are the five.

### 1. There are TWO conflicting package layouts on disk. Pick one and never mix them.
The planning wireframes (`very-good-now-i-replicated-feigenbaum.md`) specify Go packages like `internal/engine/l4/offense/`, `internal/mfl/`, `internal/store/...`, `internal/transactions/...`. The Backend Architecture doc (`Backend_Architecture.md`) specifies a *different* tree: `core/engine/layer4/`, `infrastructure/mfl/`, `interface/bindings/`. **These are two different blueprints for the same app.** A weaker model handed both will blend them and produce a tree no one can navigate.

**Resolution (this guide's recommendation — confirm with Christopher at B0):** adopt the **`internal/`-rooted layout** as canonical (Section 3), because Go's `internal/` rule gives the compiler-enforced import boundary the three-layer law needs, and a single flat `internal/<domain>/` taxonomy is far easier for a small model to keep straight than a three-tier `core`/`infrastructure`/`interface` hexagonal split. The Backend Architecture's *concepts* (transport client, normalizer, service layer, bindings) all map cleanly onto it — Section 3 shows the mapping. **B0 locks the tree. After B0, the tree is law.**

### 2. The 400-line cap is a *design* constraint, not a cleanup step.
`AGENTS.md` mandates files under 250 lines, hard cap 400. A weaker model writes the whole feature, *then* notices it's 600 lines, then hacks it apart badly. **Invert this.** Decide the file split *before* writing, from the wireframe. Section 4 gives every file's shape so the split is pre-decided. AD-14 (DT escape hatch) and AD-17 (L2 pre-split) exist precisely because two sessions were already identified as cap-breakers — Section 6 flags every other one.

### 3. The type system is your anti-leak weapon. Use it, don't narrate it.
Three of this project's hardest rules — zero scoring leak (L4), confidence-never-in-UI, player-IDs-are-strings — are enforceable at **compile time**, not by a model "remembering." AD-08 locks the L4 signature so a scoring value *cannot* be passed in. AD-07 adds a guard test that fails the build on any `confidence` field in a `*Response`. Section 3 shows the `PlayerID` newtype that makes a bare string ID a compile error. A weaker model should lean on these — if it's relying on willpower to avoid a leak, it's already lost.

### 4. Stores never import each other. The coordinator is the only writer.
The single most common structural bug in systems like this (Christopher's own words, quoted in the three-layer law) is layer confusion. Two mechanical rules prevent 90% of it: **(a)** no store package imports another store package; **(b)** only the B7 transaction coordinator holds a `StateWriter` — every other consumer gets a `StateReader` and physically cannot mutate state. Section 3 shows the Go idiom that makes (b) a compile-time guarantee, not a code-review hope.

### 5. Every domain number is already decided. Inventing one is a bug.
Salary cap is $125M. Bid increment is 0.1, snipe is 1.0 past the 20-hour mark, trade deadline is Week 9, dead cap is 35% × salary × years remaining, TFL is 2.5 points, DT/DE tackles are 2.5 (not the 1.5 universal base), CB/S interceptions are 6 (not 5). A weaker model *will* approximate these from training-data fantasy football knowledge and get them wrong. **MFL live data is authoritative over the rulebook** where they disagree (the rulebook had at least five confirmed errors). Section 5 is the canonical table. Read from it; never recall.

---

## Section 1 — The G0 Go overlay: it does not exist, and nothing can start without it

`AD-19` makes the Go overlay in `christopher-coding-standards` a **critical-path prerequisite**: B0 cannot start until it is authored and merged. Today the repo has only TypeScript (Phase 1C) and Astro (Phase 1D) overlays. This section specifies exactly what the Go overlay must contain so it mirrors the existing standard. It is written so the G0 session can build the overlay without re-deriving the pattern.

The TypeScript overlay is the template. Its anatomy → the Go equivalent:

| TS overlay file | Purpose | Go overlay equivalent | Notes |
|---|---|---|---|
| `biome.json` | lint + format rules | `.golangci.yml` | linter set below |
| `tsconfig.json` (strict flags) | compiler strictness | *N/A at config level* | Go strictness comes from vet + linters, documented in Makefile build flags |
| `.pre-commit-config.yaml` | pre-commit hooks | `.pre-commit-config.yaml` | golangci-lint + gitleaks hooks |
| `vitest.config.ts` (coverage gates) | test + coverage thresholds | Makefile `test-coverage` target + threshold script | Go has no single config file; enforce in CI |
| `stryker.config.mjs` (mutation) | mutation thresholds 80/60/60 | `gremlins` config (or documented deferral) | see open question below |
| `package.json.snippet` (scripts) | task scripts | `Makefile.snippet` | targets below |
| `Makefile` | task runner | `Makefile.snippet` | targets below |
| `schemas/example.ts` (Zod boundary) | boundary validation pattern | `schema/example.go` | hand-written decoder pattern, Section 3 |
| `README.md` | adoption walkthrough + verification | `README.md` | with the "deliberate-violation" verification test |

**`.golangci.yml` — the linter set (provisional; agy web-research confirms current 2026 advice in Section 8):**
Must-enable, security and correctness first:
- `gosec` — security SAST (the Go analogue of Semgrep's OWASP pass; catches SQL string-building, weak crypto, path traversal).
- `errcheck` — unchecked errors. Non-negotiable in a financial/scoring engine.
- `govet` — the standard vet suite.
- `staticcheck` — the high-value correctness linter (SA checks).
- `ineffassign` — ineffectual assignments (catches a class of weaker-model bug directly).
- `unused` — dead code (supports the "no commented-out / no dead code" AGENTS.md rule).
- `gocyclo` or `cyclop` — cyclomatic complexity ceiling (pushes the model toward small functions).
- `funlen` — function length limit, tuned to support the 250/400 file rule at the function grain.
- `revive` — style/naming (the `gofmt`-plus layer).
- `bodyclose` — unclosed HTTP response bodies (B1 will make HTTP calls; this is a real leak class).
- `sqlclosecheck` / `rowserrcheck` — unclosed DB rows, unchecked row errors (B-anything-touching-SQLite).
- `containedctx`, `contextcheck` — context misuse (B1 fetchers take `context.Context`).

**`.pre-commit-config.yaml`** — two hooks, both pinned to **full commit SHAs, never tags** (AGENTS.md hard rule, the trivy-action lesson): `golangci-lint` run-on-staged, and `gitleaks`. Mirror the TS overlay structure exactly.

**`Makefile.snippet` targets** (mirror TS overlay names so muscle memory transfers across languages):
- `lint` → `golangci-lint run ./...`
- `fmt` → `gofmt -w . && goimports -w .`
- `vet` → `go vet ./...`
- `test` → `go test ./...`
- `test-coverage` → `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` + a threshold gate (fail under the line below)
- `build` → `wails build` (or `go build ./...` for the engine-only phases before Wails is wired)
- `mutation-test` → `gremlins unleash ./...` (see open question)

**Coverage / mutation thresholds — mirror the TS overlay's intent** (line 80 / branch 75 / mutation high-80 low-60 break-60). The TS overlay's reasoning is the operative quote: *"80% line coverage with a 40% mutation score means tests observe but don't assert."* For a deterministic scoring engine this matters more than usual — the engine's whole value is correct math, so the rubric/engine packages should target the *high* mutation band.

**Open question for the G0 session (flag to Christopher, do not guess):** Go mutation testing is less mature than Stryker. `gremlins` is the leading option but is less active than it was; `go-mutesting` is the older alternative. The G0 author should confirm current viability (agy's web research in Section 8 has a current read) and, if neither is production-ready, **document a deliberate deferral** with mutation testing replaced by an explicit "tests must assert, not just execute" review gate on the engine packages — rather than silently dropping the standard.

**Verification test the overlay README must ship** (mirrors the TS overlay's deliberate-violation test): a `bad.go` containing an unchecked error, a string-concatenated SQL query, and a hardcoded AWS-style key. Committing it must fire: golangci-lint (errcheck + gosec), gitleaks (the key), and `make lint` non-zero. If any gate stays silent, the overlay is not wired correctly — stop and fix before B0.

**Housekeeping the G0 session also closes:** the Phase-label mismatch (`christopher-coding-standards/CLAUDE.md` calls the next overlay "Phase 1D" in one place; the Go overlay is Phase 2). Trivial, but AD-19 names it.

**Expanded linter set (from the agy Go-architect cross-reference, Section 8B).** The Section 1 list above is the floor. Add these — each one closes a *specific* weaker-model failure mode, not generic hygiene:
- `gochecknoglobals` — fails the build on package-level variables. This is the mechanical enforcement of "pure functions, no hidden state" (failure mode #5). The single highest-value linter for this project.
- `depguard` — restricts which packages may import which. This is how the three-layer law becomes a **build error** instead of a code-review note: configure it so `internal/ingestion/...` cannot import `internal/engine/...`, stores cannot import each other, and nothing imports `internal/transactions` except through its interface. (failure mode #9, cyclic-import scrambling).
- `gocritic` + `interfacebloat` — reject `interface{}`/`any` escapes where a model dodges writing a typed parser for MFL data (failure mode #1).
- `wrapcheck` — alongside `errcheck`, forces error propagation/wrapping rather than `_ = err` silent drops (failure mode #8).
- `-race` flag on `go test ./...` in the pre-commit and CI test target — catches concurrent map access from multiple Wails IPC goroutines hitting B3c (failure mode #2). Not a linter; a test-runner flag. Easy to forget; put it in the Makefile `test` target so it is never optional.
- `go.uber.org/goleak` in test teardown for the networking packages (B1/B2) — catches goroutine leaks from fetchers that ignore context cancellation (failure mode #7).

---

## Section 2 — The ten ways a weaker model produces slop here, and the guardrail for each

These are Go-specific failure modes, cross-referenced against this architecture by the agy Go-architect agent and confirmed against the coding standards. For each: the trap, and the mechanical guardrail that catches it (a linter, a flag, or a type idiom — never "remember not to").

| # | The trap a weaker model falls into | The guardrail that catches it |
|---|---|---|
| 1 | **`interface{}` / `any` escape.** Model dodges writing a typed parser for raw MFL JSON and shovels `map[string]any` through the app. | `gocritic` + `interfacebloat` linters reject it. Force a typed `RawRecord` struct at the B1/B2 boundary (WF 1B). |
| 2 | **Concurrent map write panic.** Multiple Wails IPC goroutines read/write B3c's 32-team state; model used a bare `map` with no lock. | `go test -race` in the Makefile `test` target. Plus: B3c never exposes a raw map (Section 3); access is through methods that own the lock. |
| 3 | **Leaky pointer mutation.** A read-only method returns `*PlayerRecord`; the caller mutates internal state through it. | Read methods return **values** (struct copies), never pointers to internal state. `PlayerRecord` is a pure value type (Section 3). |
| 4 | **Integer ID casting.** Model casts a player ID to `int` for "a calculation," stripping the leading zero — `"0531"` becomes `531`. | `type PlayerID string` newtype with a validating constructor (Section 3). A bare string or int ID becomes a compile error at the boundary. |
| 5 | **Package-level global state.** Model adds a package var or cache to "simplify" a scoring layer — instantly breaks pure-function determinism. | `gochecknoglobals` fails the build. The engine takes all state as parameters; there is nowhere to hide a global. |
| 6 | **SQL string concatenation.** Model builds a dynamic query with `fmt.Sprintf` for a "complex filter." | `gosec` + `sqlclosecheck` + `rowserrcheck`. Parameterized queries only (AGENTS.md hard rule). `depguard` keeps SQL out of non-repository packages. |
| 7 | **Goroutine leak.** Fetcher spawns goroutines for MFL calls without honoring `context.Context` cancellation. | `context.Context` required as first arg in all of `internal/mfl` and `internal/ingestion` (it already is, per WF 1B). `goleak` in test teardown. |
| 8 | **Silent error drop.** `_ = err`, or returning a zero-value struct with `nil` error to make a signature "work." | `errcheck` + `wrapcheck`. Every error is checked and wrapped with context. |
| 9 | **Cyclic-import scramble.** Ingestion layer imports an engine package "just to reuse a struct," creating a dependency cycle the model then can't untangle. | `depguard` enforces unidirectional flow: Layer 1 → never up. Shared types live in a leaf package (e.g. `internal/playerid`, `internal/domain`) imported by both, importing neither. |
| 10 | **Logic inside a DB transaction.** Model puts an HTTP call or a heavy scoring loop inside a SQLite write transaction, holding the single writer lock. | Repository functions wrap raw SQL only — no domain logic, no I/O. The transaction coordinator (B7a) opens the tx, calls pure calculators, writes, commits. Calculators never see the DB. |

**The meta-rule behind all ten:** *the model should never be the thing standing between a mistake and production.* A linter, a compiler error, or a `-race` failure should catch it first. When you (the building model) find yourself thinking "I'll just be careful here," stop and ask whether a guardrail should be doing the being-careful for you. If the G0 overlay is missing one of these linters, that is a finding to surface — not a reason to proceed on willpower.

---

## Section 3 — The canonical package layout and the four compile-time enforcement idioms

### 3.1 The canonical tree (resolves the two-layout contradiction)

This is the **`internal/`-rooted layout**, recommended and corroborated by the agy web-research agent (the `internal/` convention remains the 2026 idiom precisely because the Go compiler enforces the import boundary — `go.dev/doc/modules/layout`). B0 locks it. The Backend Architecture doc's concepts map onto it as shown in the right column.

```
thewarroom/
├── main.go                     # Wails bootstrap only
├── app.go                      # App struct; IPC-bound methods; thin adapters (validate → route → format)
├── wails.json
├── Makefile                    # lint / fmt / vet / test (with -race) / build / mutation-test
├── .golangci.yml               # the expanded linter set (Section 1 + 2)
├── AGENTS.md                   # copied from christopher-coding-standards, placeholders filled
├── SYSTEM_MAP.md               # the anti-drift map — UPDATE IT every session
├── internal/
│   ├── domain/                 # leaf types shared across layers (PlayerRecord, EngineRecord). Imports NOTHING internal.
│   ├── playerid/               # PlayerID newtype + validating constructor (RISK-003 single source, AD-06)
│   ├── mfl/                    # [WF 1A] transport only. one exported Do(). ← Backend doc's infrastructure/mfl
│   ├── ingestion/             # [WF 1B] fetchers → RawRecord. ← infrastructure/mfl/endpoints + csv_ingestion
│   │   ├── schema.go          #   string-ID enforcement at the boundary (RISK-003, second call site)
│   │   └── scouting/          #   the 21 external sources, grouped offense/defense/kicker
│   ├── normalize/             # [WF 1C] RawRecord → domain types. ← infrastructure/mfl/normalizer
│   ├── store/                 # [WF 2] three sibling stores; NONE imports another
│   │   ├── rulebook/          #   B3b — league rules + delta overrides (read-only data access)
│   │   ├── state/             #   B3c — 32-team mutable state; StateReader/StateWriter split (3.2)
│   │   └── params/            #   B4 — admin calibration params; ships defaults
│   ├── engine/                # [WF 3] pure-function pipeline. ← core/engine
│   │   ├── pipeline.go        #   orchestrator: L1→L2→L3→[L4 dispatch]→L5→L6
│   │   ├── layer1..layer6     #   each: Apply(EngineRecord, params) (EngineRecord, error)
│   │   ├── l4/                # [WF 4] offense/ defense/ kicker/ — one file per position
│   │   └── mathx/             #   scurve.go — the one S-curve all components call
│   ├── output/                # [WF B6] B6 append-only store; immutability trigger + no Update/Delete API
│   ├── transactions/          # [WF 6] B7. ← core/transactions
│   │   ├── coordinator.go     #   B7a — the SOLE StateWriter holder; opens tx, calls calculators, commits
│   │   ├── deadcap/           #   shared dead-cap / cap math (pure)
│   │   ├── acquisitions/      #   B7b   trades/ B7c   contracts/ B7d — each a handler; StateReader only
│   └── db/                    # SQLite: split read/write pools (3.4), migrations, parameterized queries
└── frontend/
    └── src/
        ├── modules/<name>/    # [WF 5] <Name>.tsx + <name>.store.ts + <name>.types.ts per module
        └── store/             # Zustand slices; NONE imports another (3.5)
```

**Why this and not the `core/`+`infrastructure/`+`interface/` tree:** both are valid Go, but the three-tier hexagonal split asks a small model to correctly classify every new file into one of three abstract buckets *and* remember the allowed direction between them. The `internal/<domain>/` tree makes the boundary concrete (a package name) and lets `depguard` + the compiler's `internal/` rule do the enforcement. One taxonomy, machine-checked. If Christopher prefers the hexagonal tree, that is his call at B0 — but **the two must not coexist.**

### 3.2 Idiom #1 — the StateReader/StateWriter split that makes a bad write a compile error

The single most important anti-spaghetti idiom in the build. Only the B7 coordinator may mutate league state; every handler and module gets a reader that *physically lacks* write methods. From the agy Go-architect cross-reference:

```go
// internal/store/state/state.go
package state

import "context"

// StateReader — what handlers, modules, and the engine receive.
type StateReader interface {
	GetRoster(ctx context.Context, franchiseID string) (domain.Roster, error)
	GetCapSpace(ctx context.Context, franchiseID string) (float64, error)
}

// StateWriter — held ONLY by the B7 coordinator. Embeds the reader.
type StateWriter interface {
	StateReader
	UpdateRoster(ctx context.Context, franchiseID string, r domain.Roster) error
	CommitTransaction(ctx context.Context) error
}

// ReadOnlyStore is a CONCRETE wrapper exposing only reads.
// A handler holding this cannot type-assert its way to a StateWriter —
// the methods do not exist on the type, so it fails at COMPILE time.
type ReadOnlyStore struct{ r StateReader }

func NewReadOnlyStore(r StateReader) ReadOnlyStore { return ReadOnlyStore{r: r} }
func (s ReadOnlyStore) GetRoster(ctx context.Context, f string) (domain.Roster, error) { return s.r.GetRoster(ctx, f) }
func (s ReadOnlyStore) GetCapSpace(ctx context.Context, f string) (float64, error)     { return s.r.GetCapSpace(ctx, f) }
```

Handlers accept the **concrete `ReadOnlyStore`**, not the `StateReader` interface — because a concrete type with no write methods cannot be asserted up to `StateWriter`:

```go
// internal/transactions/acquisitions/ufa.go
type UFAHandler struct{ db state.ReadOnlyStore } // concrete read-only type

func (h *UFAHandler) Validate(ctx context.Context, bid Bid) error {
	cap, err := h.db.GetCapSpace(ctx, bid.FranchiseID)
	if err != nil { return fmt.Errorf("ufa: cap lookup: %w", err) }
	if bid.Points > cap { return ErrInsufficientCap }
	return nil
	// h.db.UpdateRoster(...)            // ← does not compile: method does not exist
	// w, ok := h.db.(state.StateWriter) // ← does not compile: ReadOnlyStore can't satisfy StateWriter
}
```

The coordinator is the *only* code that ever touches a `StateWriter`. Handlers return a `TransactionResult`; the coordinator applies it. This is AD-02's "sole-writer coordinator" made real in the type system.

### 3.3 Idiom #2 — the PlayerID newtype (RISK-003, AD-06)

A bare `string` ID invites the integer-cast bug (failure mode #4). Make the ID a type with a validating constructor, in the single canonical package both ingestion boundaries import:

```go
// internal/playerid/playerid.go
package playerid

type PlayerID string // newtype: not interchangeable with a bare string

// New validates and normalizes to MFL's leading-zero string form.
// IDs under 1000 are zero-padded to 4 chars. Team-aggregate IDs (0151–0782) are rejected by the caller, not here.
func New(raw string) (PlayerID, error) {
	s := strings.TrimSpace(raw)
	if s == "" { return "", errors.New("empty player id") }
	if _, err := strconv.Atoi(s); err != nil { return "", fmt.Errorf("non-numeric player id %q", s) }
	if len(s) < 4 { s = strings.Repeat("0", 4-len(s)) + s } // 531 → 0531
	return PlayerID(s), nil
}
```

Every domain struct uses `playerid.PlayerID`, never `string`, for an ID field. Every SQLite ID column is `TEXT`.

**Honest limitation (surfaced by agy's calibration audit, 2026-06-12):** a string newtype does **not** stop a model from writing `playerid.PlayerID(raw)` directly and skipping `New()` entirely — Go permits the direct conversion. The newtype catches *accidental* `string`/`int` mixing, not deliberate-or-lazy bypass. So the constructor must be **backed by a linter**, not trusted alone: add a `forbidigo` rule (or a small custom analyzer) banning the `playerid.PlayerID(` conversion expression anywhere outside the `playerid` package itself. With that rule, `New()` becomes the only legal construction path and the bypass is a build error. Without it, this idiom is a convention, not a guarantee — do not oversell it. (A heavier alternative: wrap the value in a struct with an unexported field so external packages physically cannot construct it; the linter ban is lighter and sufficient here.)

### 3.4 Idiom #3 — split DB pools enforce the single writer at the driver level

The type-level StateWriter split (3.2) prevents *code* from writing through the wrong path. The *driver* split prevents *SQLite* from deadlocking under concurrent access. Both agy agents independently recommended this. **Driver choice:** `modernc.org/sqlite` (pure Go, CGo-free) — chosen over `mattn/go-sqlite3` because Wails cross-compilation is dramatically simpler without a C toolchain. (Note: the two drivers use slightly different DSN parameter names — confirm the exact `_pragma=` / `_busy_timeout=` syntax for `modernc.org/sqlite` at B0; the agy code sample used the mattn syntax.)

```go
// internal/db/pools.go  — DSN params shown are mattn-style; translate to modernc at B0
func NewWritePool(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_journal_mode=WAL&_busy_timeout=5000&_txlock=immediate")
	if err != nil { return nil, err }
	db.SetMaxOpenConns(1)          // ONE writer. Serializes all writes; no SQLITE_BUSY races.
	db.SetConnMaxLifetime(0)       // never recycle the writer connection
	return db, nil
}
func NewReadPool(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil { return nil, err }
	db.SetMaxOpenConns(10)         // many concurrent readers (Wails IPC goroutines)
	return db, nil
}
```

The coordinator (B7a) holds the write pool. Everything else gets the read pool. The architecture is now enforced at *two* layers — types and driver — which is exactly the "both ways" rigor AD-04 applied to B6 immutability.

**Caveat — this is sound but error-prone; verify it, do not assume it (my two agy consultations disagreed here, which is the signal).** The agy web-research agent recommended exactly this split-pool pattern; the agy calibration agent flagged it as a contention risk (two `sql.DB` pools against one SQLite file *can* still produce `SQLITE_BUSY` if the WAL pragma doesn't actually apply or `busy_timeout` is missing). Both are right: the pattern is a known-good WAL idiom **and** it is the kind of thing a weaker model implements subtly wrong (forgetting `busy_timeout`, or the pragma silently not taking on the read-only pool). Treat it as **needs-verification at B0**, not settled: write the parallel-readers-plus-single-writer concurrency bench (agy-expert's own verification step) and confirm zero `SQLITE_BUSY` under load before this pattern is blessed as the template. A WAL pragma that fails to apply is invisible until it deadlocks in production.

### 3.5 Idiom #4 — pure-function pipeline with value semantics (zero side effects)

The engine's value is correct, deterministic math. Protect that by making side effects structurally impossible. From the agy cross-reference: `EngineRecord` carries no pointers, slices, or maps — pass-by-value means a layer *cannot* mutate a parent's state even by accident.

```go
// internal/engine/pipeline.go
type ApplyFunc func(rec domain.EngineRecord, p domain.RubricParams) (domain.EngineRecord, error)

type Pipeline struct{ layers []ApplyFunc }

func New(layers ...ApplyFunc) *Pipeline { return &Pipeline{layers: layers} }

func (pl *Pipeline) Run(in domain.EngineRecord, p domain.RubricParams) (domain.EngineRecord, error) {
	cur := in
	for _, layer := range pl.layers {
		next, err := layer(cur, p) // value in, value out — no shared memory
		if err != nil { return domain.EngineRecord{}, err }
		cur = next
	}
	return cur, nil
}
```

Each layer is `func Apply(rec, params) (rec, error)` — `gochecknoglobals` guarantees it reads no package state, and value semantics guarantee it writes none. L4 dispatch by position slots into the pipeline as one `ApplyFunc` that switches on `rec.Position` to the right rubric. **Caveat for whoever wires this:** if `EngineRecord` ever genuinely needs a slice (e.g. a list of film sub-signals), treat that slice as immutable-by-convention and never append to a parent's backing array — or, better, pass it through `RubricParams` (config) rather than the mutable record. Flag this to Christopher if it comes up; do not silently add a slice to the record.

---

## Section 4 — Copy-ready Go skeletons, one per wireframe

A weaker model that invents structure produces slop. A weaker model that copies a correct skeleton and fills the body produces maintainable code. These skeletons already satisfy the wireframes, the 400-line cap (by being pre-split), and the standards. **Copy the matching one at the start of the relevant session; do not design from scratch.**

Every file follows the same internal order so the next session's model finds things where it expects: `package` → imports → types → constructor → exported method(s) → unexported helpers. Hold that order everywhere.

### WF 1A — transport client (B1)
```go
// internal/mfl/client.go — one exported method, no domain types
package mfl

type Client struct {
	http    *http.Client
	limiter *rate.Limiter // token bucket; back off on 429, do NOT retry-storm
	host    string        // discovered league host (e.g. www47), cached
}

func New(host string, rps float64) *Client { /* ... */ }

// Do and DiscoverHost are the two sanctioned exported surfaces for B1.
// Transport in, transport out. No Player, no Schedule — no domain types.
func (c *Client) Do(ctx context.Context, req Request) (Response, error) {
	// rate-limit wait → execute → on 429: exponential backoff 1→2→4…→60s, then return error (caller decides)
	// close resp.Body (bodyclose linter enforces)
}

// DiscoverHost is host-discovery-specific (the "do better" instruction for B1,
// Section 6) and MUST itself route through Do, so it inherits rate-limiting and
// backoff. It is the ONLY other permitted exported method; no third exported
// surface may be added without a plan amendment.
func (c *Client) DiscoverHost(ctx context.Context, year, leagueID string) error {
	// query api.myfantasyleague.com/{year}/export?TYPE=league&L={id}&JSON=1 via Do,
	// read league.baseURL, extract+cache the subdomain. For the TYPE=league
	// discovery call specifically, force host="api" regardless of c.host so a
	// stale cached host can't block re-discovery (T3 finding #2, fixed in B1).
}
```
> **Reconciliation note (Confidence-80 session, Gate 5 / Friction #13 finding #5,
> 2026-06-13):** this skeleton previously read "Do is the ONLY exported surface,"
> which contradicted B1's own "do better" guidance below (line ~549, "make host
> discovery a first-class method"). Building B1 surfaced the contradiction. Both
> are now sanctioned, with `DiscoverHost` explicitly constrained to route through
> `Do`. Recorded as a plan-fidelity fix, not a scope change.
**Trap:** the host (`www47`) is not constant — discover it from the league endpoint on startup, cache it, re-verify weekly/at season change, fall back to `api.myfantasyleague.com` if it fails. Never hardcode `www47`.

### WF 1B — fetcher (B2 / B2b)
```go
// internal/ingestion/rosters/fetcher.go — one exported func per file, returns RAW records
package rosters

func Fetch(ctx context.Context, c *mfl.Client) ([]RawRoster, error) {
	// build Request → c.Do() → schema-validate the response → return raw. NO transformation here.
}
```
```go
// internal/ingestion/schema.go — string-ID enforcement at the boundary (RISK-003 call site #1)
func validatePlayerID(raw string) (playerid.PlayerID, error) { return playerid.New(raw) }
```
**Trap:** fetchers transform nothing. Validate the shape, store/return raw. Transformation is WF 1C's job. Filter team-aggregate IDs (range 0151–0782) here before they pollute downstream.

### WF 1C — normalizer (B3)
```go
// internal/normalize/roster.go — RawRoster → domain.Roster; pure transformation
package normalize

func Roster(raw ingestion.RawRoster) (domain.Roster, error) {
	id, err := playerid.New(raw.PlayerID) // RISK-003 call site #2 (AD-06: one impl, two sites)
	if err != nil { return domain.Roster{}, fmt.Errorf("normalize roster: %w", err) }
	// parse salary STRING → float ("7"→7.0, "1.30"→1.30). contractStatus is DIRTY: trim, then starts-with parse.
}
```
**Trap (B3 is a reviewed deliverable, AD-18):** the internal type system locks here; a mistake cascades to every module. Salary and cap fields arrive as *strings* — parse them, don't assume float. `contractStatus` has confirmed dirty values (`"UFA "`, `"YFA"` typo, `"EXT (2024)"`, trailing spaces) — normalize defensively.

### WF 2 — store (B3b / B3c / B4)
```go
// internal/store/state/state.go — see Section 3.2 for the full StateReader/StateWriter/ReadOnlyStore idiom
package state

type Store struct {
	read  *sql.DB        // read pool
	write *sql.DB        // write pool — only the coordinator constructs a Store with this set
	mu    sync.RWMutex   // guards any in-memory snapshot; NEVER expose a raw map
}

func (s *Store) GetRoster(ctx context.Context, f string) (domain.Roster, error) { /* parameterized query */ }
func (s *Store) Initialize(ctx context.Context, src normalize.Source) error      { /* PULLS from B3; B3 never pushes */ }
// No Reload() on B3c. No store imports another store. depguard enforces.
```
**Trap:** B3c has no `Reload()` (planning decision). It `Initialize()`s by *pulling* from B3 — B3 never pushes into it. Never return a raw `map`; concurrent IPC goroutines will panic (failure mode #2). The `-race` flag is your proof.

### WF 3 — engine layer + pipeline (B5a)
```go
// internal/engine/layer1/layer1.go — every layer is this exact shape
package layer1

func Apply(rec domain.EngineRecord, p domain.RubricParams) (domain.EngineRecord, error) {
	// pure: read rec + p, return a NEW rec. No globals (gochecknoglobals), no I/O, no pointers to internal state.
	return rec, nil
}
```
Pipeline composition and value-semantics rules: Section 3.5. **Trap (AD-17):** before B5a, check that L2 (the full scoring matrix — every position's event values + True Position split + special teams) fits under 400 lines. If not, pre-split into `engine/scoring/{offense,defense,special}.go` *before* writing the orchestrator. Decide at Session 11 close, not mid-B5a.

### WF 4 — position rubric (B5b-*) — the most precisely specified shape
```go
// internal/engine/l4/offense/qb.go — ALWAYS these four private funcs, this order, every position
package offense

// Score is the only export. Receives ONLY scouting inputs — never an L2 score, never MFL config (AD-08).
// That signature is the compile-time guarantee of zero scoring leak.
func Score(player domain.PlayerRecord, scouting domain.ScoutingData, p domain.RubricParams) domain.L4Score {
	return combine(scoreFilm(player, scouting, p), scoreRAS(player, scouting, p), scoreBreakout(player, scouting, p))
}

func scoreFilm(player domain.PlayerRecord, scouting domain.ScoutingData, p domain.RubricParams) float64 { /* */ }
func scoreRAS(player domain.PlayerRecord, scouting domain.ScoutingData, p domain.RubricParams) float64  { return 1.000 } // QB: SL-020
func scoreBreakout(player domain.PlayerRecord, scouting domain.ScoutingData, p domain.RubricParams) float64 { /* */ }

func combine(film, ras, breakout float64) float64 { return film * ras * breakout } // identical everywhere; NOT special-cased (AD-10)
```
**Per-position deltas live ONLY inside the four functions** — never change the signatures, never add a fifth function to the main file (use the AD-14 escape hatch `qb_<mechanic>.go` in the same package for helpers). The position-specific rules (which positions force RAS to 1.000, where NGS anchors, where SL-019 applies/excludes) are in Section 5.3 — read them there, they are landmines.

### WF 5 — module (M1–M9b): three frontend files + one backend response
```go
// internal/api/rankings_response.go — NO confidence field (AD-07 guard test fails the build if present)
package api
type RankingsResponse struct {
	Players []PlayerRow `json:"players"`
	// confidence stays internal. It must not appear on any *Response type.
}
```
```ts
// frontend/src/modules/rankings/rankings.store.ts — IPC call lives in the store, never the component
export const useRankingsStore = create((set) => ({
  rankings: null,
  setRankings: (d) => set({ rankings: d }),
  fetchRankings: async () => set({ rankings: await window.go.main.App.GetRankingsData() }),
})) // never imports another module's store
```
```go
// app.go — thin adapter: validate → route to service → format. NO direct SQL here.
func (a *App) GetRankingsData() (api.RankingsResponse, error) { /* validate args → a.engineSvc → format */ }
```
**Trap:** modules never call a store's write path. All writes route through a B7 `App` method. The component reads its own Zustand slice only. Theme/density is CSS-variable driven (`data-density` attribute), never React state. Lists are virtualized (`react-window`) — non-negotiable for the 1,500+ player list.

### WF 6 — transaction handler + coordinator (B7a–d)
```go
// internal/transactions/coordinator.go — the ONLY StateWriter holder (AD-02)
package transactions

type Coordinator struct {
	w        state.StateWriter // exclusive
	rules    rulebook.Reader
	handlers map[Kind]Handler
}

func (c *Coordinator) Execute(ctx context.Context, req Request) (Result, error) {
	h := c.handlers[req.Kind]
	res, err := h.Handle(ctx, req, c.rules, state.NewReadOnlyStore(c.w)) // handler gets READ-ONLY
	if err != nil { return Result{}, err }
	// open ONE sqlite tx on the write pool → apply res via c.w → commit. No I/O, no scoring inside the tx.
	return res, nil
}
```
```go
// internal/transactions/acquisitions/ufa.go — handler: validate + calculate, NEVER write (Section 3.2)
func (h *UFAHandler) Handle(ctx, req, rules, db state.ReadOnlyStore) (Result, error) { /* ... */ }
```
**Trap:** the handler calculates and returns a `Result`; the coordinator writes it. Nothing else in the app may construct a `StateWriter`. The Week-9 trade deadline, bid clocks, and cap math are domain logic — they run in the handler *before* the tx opens, never inside it (failure mode #10).

---

## Section 5 — Never-Hallucinate constants (the canonical numbers)

A weaker model *will* fill these from generic fantasy-football training data and be wrong. Every number here is sourced from the project docs. **Read from this table; never recall.** Where MFL live data and the rulebook disagree, **MFL wins** (the rulebook had ≥5 confirmed errors, listed in 5.4). Put these in code as named constants in one `internal/domain/constants.go`, not as magic numbers scattered across files.

### 5.1 League configuration
| Constant | Value | Note |
|---|---|---|
| League ID | `14432` | Legacy NFL |
| MFL host | `www47` | **discover at runtime, cache, re-verify** — not constant |
| Salary cap | `$125M` | arrives as string `"125"`; parse to float |
| Roster size | `80` | |
| Taxi squad | `8` slots | counts 100% toward cap |
| Injured Reserve | `12` slots | counts 100% toward cap |
| Weekly starters | `21` | 8 offense, 12 IDP, +1 |
| Regular season | weeks `1–17` | last regular week = Week 13 |
| Score decimals | `2` | |

### 5.2 Weekly starter limits by position
QB `1` · RB `1–3` · WR `2–5` · TE `1–3` · PK `1` · DT `2–4` · DE `2–4` · LB `2–4` · CB `2–4` · S `2–4`

### 5.3 Engine landmines — position-specific rules (the ones a model gets wrong)
| Rule | Applies | Does NOT apply | Source |
|---|---|---|---|
| `scoreRAS` forced to **1.000** | QB, K | everyone else | SL-020 |
| WR `scoreRAS` computes **live** High-tier curve (±8%, SL-018 decay 1.00/0.50/0.10); **do NOT force 1.000** | WR | — | AD-09 / SL-022 |
| NGS Coverage Metrics anchor | **CB and S only** | all other positions | CLAUDE.md hard constraint |
| SL-019 modulator | TE, DE, CB, S | **excluded at DT** (Cushion Guard replaces it — running both double-protects) and LB | CLAUDE.md / DT rubric |
| Cushion Guard | DT | — | SL-021; carries SL-OQ-037 (threshold) |
| Dynamic PFF α | DT | — | carries SL-OQ-039 (down-shift trigger) |
| SL-005 compression in `scoreFilm` | LB | — | LB rubric |
| K: all three components return 1.000 → `combine` yields 1.000, **not special-cased** | K | — | AD-10 |
| `combine` = `film × ras × breakout` | every position, identical | — | AD-10 |

### 5.4 Scoring values (MFL-authoritative)
**True Position tackle/assist/INT/PD split** — base is universal; DT/DE/CB/S *stack* on top:
| Position | Tackle | Assist | Interception | Pass Defensed |
|---|---|---|---|---|
| DT | **2.5** | **1.5** | 5 | 2.5 |
| DE | **2.5** | **1.5** | 5 | 2.5 |
| LB | 1.5 (base) | 1.0 (base) | 5 | 2.5 |
| CB | **2.0** | 1.0 | **6** | **3.0** |
| S | **2.0** | 1.0 | **6** | **3.0** |

Other key values: **TFL = 2.5** (universal, all positions) · **Sack = 4.5** (not 5) · QB Hit = 1 · Safety = 10 · Pass TD = 5 · Rush TD = 6 · Rec TD = 6 · PPR reception = 1 · Pass yd = 0.05 · Rush yd = 0.1 · Rec yd = 0.1.

**Long-play bonuses are cumulative discrete events, NOT derived from yardage:** a 43-yard rush fires R20 (+1) AND R40 (+1) = +2. Read them as stat events from `playerScores`, never compute from raw yards (OQ-003).

**Rulebook errors (MFL is truth):** Interception Return Yards = **0.1/yd** (not 0.025) · Fumble Recovery Return Yards = **0.1/yd** (not 0.025) · CB/S tackles = **2.0** (not 1.5) · CB/S INT = **6** (not 5) · CB/S PD = **3.0** (not 2.5).

### 5.5 Engine numeric defaults
Salary floor: per experience tier (B4 default); missing-RAS imputation = position-group mean, fallback **5.00**; missing Breakout Age fallback **21.0**; missing School Tier fallback **Group of Five**. Age decay default rate **0.03**, peak limit position-specific. Cap-tier (Layer 5): Cold `<1.2%` of cap → ×**1.15**; Neutral `1.2–4.8%` → ×**1.00**; Hot `>4.8%` → ×**0.85**. S-curve = Shape-B sigmoid with overflow clamp (arg clamped ±500), hard-bounded to `[1-cap, 1+cap]`. These are B4-tunable; ship the defaults, never invent replacements.

### 5.6 Transaction constants
| Rule | Value |
|---|---|
| Bid increment | `0.1` points |
| Snipe window / increment | after the **20-hour** mark, must exceed by a full **1.0** point |
| Bid clock | `24` hours unchallenged to win |
| Year multipliers | 1yr ×1.00 · 2yr ×1.20 · 3yr ×1.40 · 4yr ×1.60 |
| Max contract | `4` years |
| Max 1-yr bid | `$12M` (lifted during playoffs) |
| Salary→years gates | $12–24M needs ≥2yr; >$24M needs ≥3yr |
| RFA offer window | `7` days; baseline re-sign +0/+15/+30/+45% for 1/2/3/4yr |
| Waiver dead cap | **35% × annual salary × remaining years**; ends if claimed |
| **Trade deadline** | **Week 9** (hard timestamp block) |
| DOT review | **3 approves OR 3 vetoes** (binary, not majority) |
| Pick trading | ≤2 years ahead; compensatory picks **not** tradeable |
| Franchise tag | top-5 positional average; floor 120% of prior year; max 2 consecutive (2nd = 120% of 1st) |
| Extension | 1/GM/season; +≤3 yrs; ≤6 total; priced at 150% of highest remaining year |
| Restructure | 1/team/year; 50% dead-cap penalty if later waived |
| Buyout | 2/team/season, offseason only; 60/75/90% for 2/3/4 yrs remaining |

### 5.7 MFL API cache discipline (MFL enforces some of this)
`players` endpoint: **once per day MAX** (cache it; all lookups read cache) · `league`/`rosters`/`contracts`/`salaries`: daily · `standings`/`playerScores`/`draftResults`: weekly/event · `transactions`/`tradeBait`: event-triggered · `liveScoring`: **NO cache**, poll only in active game windows. Always append `JSON=1` (else XML). All calls server-side (MFL blocks cross-domain JS). On 429: back off, do **not** retry-storm.

---

## Section 6 — Per-session field briefs (G0 + 38 sessions)

One brief per session. Format: **Trap** (the specific way a weaker model breaks this session) · **Contingency** (what to do when it goes sideways) · **Do better** (an improvement over a naive read of the plan). Sessions flagged 🔴 are the four the agy Go-architect rated highest-risk — read those twice. Gates and scope come from `Build_Tracker.md`; this is the *how*, not the *what*.

**G0 — Go overlay** (pre-build, in `christopher-coding-standards`). **Trap:** treating it as a config-copy job and shipping a thin `.golangci.yml`. **Contingency:** if `gremlins` mutation testing won't run cleanly, document a deliberate deferral with an engine-package "tests must assert" review gate — don't silently drop the standard. **Do better:** ship the expanded linter set (Section 1 + 2) *with `depguard` import rules pre-written for TheWarRoom's tree* so B0 inherits the three-layer law as build errors on day one. Run the deliberate-violation verification test before declaring done.

**B0 — Scaffold (S1).** **Trap:** the five locked patterns (AD-03) get written loosely, and every later session clones the looseness. **Contingency:** if Wails + the split DB pools fight during setup, stand up the engine packages (pure Go, no Wails) first and wire Wails second — the engine has no Wails dependency. **Do better:** lock the canonical tree (Section 3.1) and commit `SYSTEM_MAP.md` *fully populated* now; an accurate map is the single biggest force-multiplier for every weaker model after you.

**B1 — MFL client (S2).** **Trap:** leaking domain types into the transport layer; hardcoding `www47`. **Contingency:** if rate limits bite during testing, the backoff path is the feature — verify it, don't disable it. **Do better:** make host discovery a first-class method, not an afterthought (Section 4 WF 1A).

**B2 — Ingestion (S3).** Gate: OQ-005. **Trap:** transforming data in the fetcher. **Contingency:** if OQ-005 (salary adjustment) is unresolved, stop — it blocks B2/B3, ask Christopher. **Do better:** filter team-aggregate IDs (0151–0782) at ingestion so they never reach normalization.

**B3 — Normalization (S4) — reviewed deliverable (AD-18).** Gate: OQ-004, OQ-005. **Trap:** assuming salary/cap arrive as numbers (they're strings) and that `contractStatus` is clean (it's dirty — `"UFA "`, `"YFA"`, trailing spaces). **Contingency:** the type system locks here; if a type feels wrong, flag it before downstream sessions build on it — a B3 type error cascades to all modules. **Do better:** write the dirty-value normalizer as a tested table (input→output) so the next model can see every known dirty case.

**B2b-Schema (S5) — reviewed gate (AD-16).** **Trap:** designing the scouting schema for only the positions in front of you. **Contingency:** walk all 10 positions' inputs in the review; decide SL-OQ-035/036 (box/deep safety field reservation) now or explicitly accept a controlled later revision. **Do better:** reserve fields for the deferred SL-OQs so a later add is a column, not a migration.

**B2b-Fetch Offense/Defense/Kicker (S6–S8).** **Trap:** one giant fetcher file over 400 lines. **Contingency:** one file per source; if a source is flaky, stub it behind the same function signature and flag it. **Do better:** group by the position-need map (Section 5 / Approved_Sources) so offense fetchers unblock QB (S14) on time.

**B3b — Rulebook (S9).** **Trap:** putting rule *logic* in the rulebook store (it's pure data access + delta overrides). **Do better:** model deltas as separate records over MFL defaults, never edit-in-place.

🔴 **B3c — League State Store (S10).** The agy agent's #1 risk. **Trap:** exposing a raw `map` (concurrent IPC reads panic), adding a `Reload()` that bypasses transaction logic, or a thread-unsafe implementation. **Contingency:** if concurrency bugs appear, `go test -race` is the diagnosis — make it pass, don't paper over it. **Do better:** implement the full StateReader/StateWriter/ReadOnlyStore idiom (Section 3.2) *here*, and construct the in-memory snapshot behind a `sync.RWMutex` with no raw map ever returned. No `Reload()` on B3c — it `Initialize()`s by pulling from B3.

**B4 — Param store (S11).** **Trap:** baking calibration values as magic numbers. **Do better:** ship all Section 5.5 defaults from this store; at S11 close, run the AD-17 pre-check (does L2 fit 400 lines?) and decide the B5a split before S12 starts.

**B5a — Engine pipeline (S12).** Gate: B3b·B3c·B4 + L2 pre-check. **Trap:** a layer reaching for package state or mutating a shared record. **Contingency:** if L2 is over 400 lines, pre-split into `engine/scoring/{offense,defense,special}.go` *before* writing the orchestrator (AD-17). **Do better:** build the `ApplyFunc` pipeline (Section 3.5) with value-semantics `EngineRecord`; `gochecknoglobals` proves purity.

**Testing Harness (S13) — hard gate.** **Trap:** skipping ahead to a rubric without the harness. **Contingency:** no rubric session starts until this passes — it's a hard gate, full stop. **Do better:** build the harness to assert on *outputs* (worked cases like Jefferson/Lockett), not just to execute — this is where the mutation-score intent lives even if mutation tooling is deferred.

🔴 **B5b-QB (S14) — skeleton-setter.** **Trap:** a flawed four-function structure here is cloned into all 9 remaining rubrics. **Contingency:** if the shape feels cramped, fix it *now* — every later rubric inherits it. **Do better:** match Section 4 WF 4 exactly; QB forces `scoreRAS`→1.000 (SL-020) but keep the function present and returning 1.000, don't delete it — uniformity is what lets the next model read all 10 rubrics the same way.

🔴 **B5b-DT (S15) — stress-test (AD-15).** The hardest rubric, placed second on purpose. **Trap:** applying SL-019 at DT (forbidden — Cushion Guard replaces it; running both double-protects), or a naive Cushion Guard formula. **Contingency:** if DT blows the 400-line cap, use the AD-14 escape hatch — `dt_cushionguard.go`, `dt_pff_alpha.go` in the same package; the four core functions stay in `dt.go`. **Do better:** if DT fits the wireframe, all positions do. Carry SL-OQ-037/039 as B4-default flags rather than blocking.

**B5b-RB/WR/TE/DE/LB/CB/S/K (S16–S23).** **Trap (the big one):** WR — do NOT force `scoreRAS` to 1.000 (AD-09); it computes the live High-tier curve. CB/S — NGS anchor only here. K — all components 1.000, `combine` yields 1.000, not special-cased (AD-10). **Contingency:** each rubric's open SL-OQ (Section 5.3) defaults to a B4 value with a CAL flag rather than blocking. **Do better:** verify each position against Section 5.3 before writing — these are the most landmine-dense sessions.

**B6 — Output store (S24).** Gate: B5a + all rubrics. **Trap:** an `Update`/`Delete` method sneaking onto an append-only store. **Do better:** AD-04 immutability *both ways* — no mutating API methods AND a SQLite `UPDATE`/`DELETE` trigger that rejects historical-row changes. Tag every row with `scoring_config_id` (DECISION-010).

**M1 — Asset Rankings (S25).** Gate: B6·B3c. **Trap:** an unvirtualized 1,500-row list (UI death). **Contingency:** if M1 scope balloons, split the per-team drill-down to M1b (AD-20, pre-authorized). **Do better:** `react-window` from the first line; density via CSS variables, not React state. First visible engine validation — make it trustworthy.

🔴 **B7a — Transaction Foundation + Coordinator (S26).** The agy agent's core-risk session — the whole three-layer law hinges here. **Trap:** a coordinator that doesn't enforce atomicity, or lets handlers write state directly. **Contingency:** if atomicity is unclear, the rule is: one SQLite tx on the write pool, opened by the coordinator, after the handler has validated/calculated; commit or roll back as a unit. **Do better:** build + test the coordinator *first*, alone, before any handler (AD-02). It is the only `StateWriter` holder in the entire app (Section 4 WF 6). Prove with a test that a handler cannot obtain a writer.

**B7b — Acquisitions (S27).** Gate: OQ-009, OQ-010. **Trap:** the micro-bid bypass (validating against `+0.001` instead of the 0.1 rulebook minimum). **Contingency:** OQ-009/010 unresolved → stop, ask. **Do better:** validate increments against Section 5.6 exactly; enforce the 20-hour snipe (1.0 point) rule.

🔴 **B7c — Trades (S28).** agy-flagged. **Trap:** validating only one team's cap; mis-windowing multi-year picks; timezone-fragile Week-9 deadline parsing. **Contingency:** if date parsing is fragile, anchor the deadline to an explicit league-timezone timestamp, not local time. **Do better:** validate *both* teams' post-trade cap and roster compliance; reject compensatory picks and >2-year-out picks explicitly.

**B7d — Contracts (S29).** Gate: OQ-008. **Trap:** mixing the tag/extension/restructure/buyout math. **Do better:** share the dead-cap calculator (`deadcap/`); each mechanic is a thin handler over it. Buyout is offseason-only, 2/team/season.

**M4 — Transaction UI (S30).** Gate: B7a–d. **Trap:** a module writing state outside B7. **Do better:** every submission routes through a B7 `App` method — the UI never touches B3c.

**M2/M5/M3/M6/M7/M8 (S31–36).** Mostly read-only views over B6/B3c. **Trap:** importing another module's Zustand store. **Do better:** each module is self-contained (Section 4 WF 5); M7 (Trade Analyzer) surfaces analysis but makes no DOT decision; M8 reads compliance across 32 teams.

**M9a — Calibration UI (S37) / M9b — Rules UI (S38).** **Trap:** routing admin writes through B7 (wrong — AD-05). **Do better:** admin writes go direct to B4 (calibration) / B3b (rules), schema-validated at the App boundary, Christopher-only. Structural mechanics stay code-locked; only SL-017 params are exposed.

---

## Section 7 — Per-session pre-flight and pre-commit checklist

A weaker model that runs this every session cannot drift far. Copy it into the session handoff.

**Pre-flight (before writing code):**
1. Read this session's brief (Section 6) and its `Build_Tracker.md` row. Confirm the upstream gate is `[x]` and any blocking OQ is resolved — if not, **stop and ask Christopher**.
2. Confirm branch: `git branch --show-current`. Never `main`. Branch `session/<short-desc>`.
3. Open the matching skeleton (Section 4) and the constants (Section 5). Decide the file split *now*, from the wireframe, before writing a line.
4. Read `SYSTEM_MAP.md` — reuse existing utilities; do not re-implement.

**Pre-commit (every commit, no exceptions):**
1. `make fmt && make lint` — golangci-lint clean (the full set: gosec, errcheck, depguard, gochecknoglobals, …).
2. `make test` — **with `-race`**; `go test ./...` green.
3. Every file under 400 lines (target 250). Over? Refactor *before* commit, per the wireframe — never after.
4. No `interface{}`/`any` escapes, no package globals, no raw maps returned from stores, no `fmt.Sprintf` into SQL, no `_ = err`.
5. Player IDs are `playerid.PlayerID`, all SQLite ID columns `TEXT`. No `confidence` on any `*Response`.
6. New top-level dir? Update `SYSTEM_MAP.md` in the same commit (AGENTS.md rule).
7. Conventional commit prefix + `[AI-assisted]` tag. **Never `--no-verify`.**

**Session close:**
1. Functional verification — Christopher uses the actual behavior, not just "build green."
2. Write the next session's handoff per `Handoff_Protocol.md`.
3. Update `SYSTEM_MAP.md` if the surface changed. Check the session's box in `Build_Tracker.md` only when all three close gates pass.

---

## Section 8 — What this guide adds over the previous plan, and the cross-reference behind it

### 8A — Concrete improvements over a naive read of the existing plan
The Session 1–3 planning arc (older models) produced an excellent *sequencing* plan. It did not produce *implementation guardrails*. This guide adds, and recommends the build adopt:

1. **Resolve the two-layout contradiction (Section 3.1).** The wireframes and the Backend Architecture doc specify different package trees. The previous plan never noticed. Left unresolved, every weaker model picks a different one. → Lock `internal/`-rooted at B0.
2. **Make the three-layer law a build error, not a guideline.** `depguard` import rules + the `internal/` compiler boundary turn "don't cross layers" into "won't compile if you do." The previous plan stated the law in prose; prose doesn't stop a 7B model at 2am.
3. **Compile-time write lockout (Section 3.2).** The `ReadOnlyStore` concrete-wrapper idiom means a handler *cannot* obtain a `StateWriter` — the sole-writer rule is enforced by the type system, not by hoping the model read AD-02.
4. **Driver-level single-writer (Section 3.4).** Split read/write SQLite pools (`SetMaxOpenConns(1)` on the writer) enforce the same rule a second way and kill `SQLITE_BUSY` deadlocks the previous plan never addressed.
5. **The G0 overlay specified, not just named (Section 1).** AD-19 flagged the Go overlay as critical-path but no one wrote down what it must contain. This guide does — file by file, linter by linter — so G0 isn't a re-derivation.
6. **The ten weaker-model failure modes mapped to specific linters (Section 2).** Generic "write good code" advice doesn't help a small model. "`gochecknoglobals` will fail your build if you add a package var" does.
7. **A never-hallucinate constants table (Section 5).** The previous plan's numbers were spread across a dozen docs. A weaker model won't find them all and will invent the rest. One table fixes that, and records the 5 rulebook errors where MFL overrides.
8. **Pure-function value semantics (Section 3.5).** `EngineRecord` with no pointers/slices/maps makes side effects structurally impossible — a stronger guarantee than "please write pure functions."

None of this changes the sequence, the decisions (AD-01..25), or the wireframes. It makes them *enforceable by a weaker hand.*

### 8B — agy cross-reference (CT 104, read-only, 2026-06-12)
Two agy agents on CT 104 independently corroborated and extended this guide. Both ran read-only; neither modified TheWarRoom.

**agy web-research agent — current 2026 best practices (with sources):**
- **Layout:** `internal/` convention remains idiomatic *because* the compiler enforces the import boundary. → `https://go.dev/doc/modules/layout`
- **Maintainable LLM Go:** consumer-defined single-method interfaces; manual DI via constructors in `main.go`; package-per-capability not per-layer. → `https://go.dev/wiki/CodeReviewComments#interfaces`
- **SQLite:** pure-Go `modernc.org/sqlite` (CGo-free → trivial Wails cross-compile); split writer (`SetMaxOpenConns(1)`, `_txlock=immediate`) / reader pools, `_busy_timeout=5000`. → driver pool guidance.
- **Validation:** hand-written `Validate() error` methods over reflection-based `go-playground/validator` — compile-time safe, no struct tags an LLM can hallucinate. Matches AGENTS.md's "no opaque struct tags." → `https://go.dev/blog/json`
- **Linters:** `gosec` + `govet` + `staticcheck` + `errcheck` + `unused` as the core. → `https://golangci-lint.run/usage/linters/`
- **Mutation:** `gremlins` over `go-mutesting`, restricted to pure-logic packages (`internal/engine`). → `https://github.com/go-gremlins/gremlins`

**agy Go-architect agent — design stress-test:** delivered the 10 failure-mode→linter map (Section 2), the `ReadOnlyStore` compile-time lockout (Section 3.2), value-semantics `PlayerRecord`/pipeline (Section 3.5), the driver-pool split (Section 3.4), and the four highest-risk sessions (B3c/S10, B5b-QB/S14, B5b-DT/S15, B7a/S26, B7c/S28 — flagged 🔴 in Section 6). Full report archived on CT 104 at `/root/.gemini/antigravity-cli/brain/<session>/build_plan_audit_report.md`.

**One discrepancy surfaced and resolved:** agy-web recommended `modernc.org/sqlite` (pure Go); agy-expert's sample code used `mattn/go-sqlite3` (CGo). Resolution: **use `modernc.org/sqlite`** for CGo-free Wails cross-compilation; translate the DSN parameter names at B0 (the two drivers differ slightly — verify `_pragma=busy_timeout` vs `_busy_timeout` syntax against modernc's docs). Flagged in Section 3.4.

---

## Section 9 — Best Practices: the Claude ↔ agy collaborative code-hygiene workflow

This section is the product of a direct, multi-round negotiation between Claude Code (Builder, CT 105) and agy/Antigravity (Recon/Audit, CT 104) on 2026-06-12, at Christopher's direction. The brief was explicit: *collaboration is stronger than a narrow guessing tree; use agy's expertise, do not just believe Claude 100% of the time.* What follows is the workflow the two agents actually agreed on — including the points where agy changed Claude's mind and the points where agy declined authority Claude offered it. It is represented as agy stated it, not as Claude would have paraphrased it.

**The governing principle.** A single model reasoning alone follows a narrow path and cannot see what its own context has already normalized as fine. The second agent's value is not throughput — it is a *different vantage*: agy is Go-native (Antigravity is built in Go), carries live web search, and was trained differently, so it catches what Claude's context has smoothed over. The workflow below institutionalizes that second vantage as a standing part of the build, not an optional garnish. It does not flatten agy into fetch-and-confirm, and it does not let Claude overrule by role where agy is the better judge.

### 9.1 Authority, split by domain (not by rank)
The line is not "Builder outranks Recon." It is "whoever holds the better instrument decides." agy *declined* a blanket deference Claude offered — correctly noting that language-nativity alone does not justify overriding the Builder's ownership of runtime and architecture. The agreed split:

| Domain | Default authority | Override rule |
|---|---|---|
| **Tooling & dependency versions** (Go toolchain, linter flags, library versions, 2026 practice) | **agy** — it holds live web search | Builder adopts agy's version/tooling recommendation unless it has a specific, stated reason. |
| **Idiomatic Go patterns** (error wrapping, goroutine lifetimes, context use, interface shape) | **Builder-final**, but agy-informed | agy flags non-idiomatic structure; Builder may override but **must justify the override in writing** (relay note or code comment). |
| **Architecture & runtime design** (boundaries, the three-layer law, state ownership) | **Builder** — it owns the system's dynamic state | Structural *disagreements* go to the arbiter in 9.4, not to Builder fiat. |
| **Security / auth / crypto / external-input boundaries** | **Builder-final** (per `multi-agent-roles.md`) | agy audits and reports; Builder signs off using the same checklist every time. |

### 9.2 Uniformity in parallel work — the shared enforcement layer is the toolchain, not a doc
agy cannot (and should not) edit the living standard docs — that boundary stays. But uniformity does not require shared write access; it requires a **shared, committed enforcement file** that the compiler and linter apply to both agents equally. That file is **`.golangci.yml` in the repo root** (plus the pinned toolchain, 9.6). The flow:
1. agy proposes a standard improvement as a **copy-pasteable diff or config snippet** — never as a doc edit.
2. Claude reviews it, and if adopted, **commits it** to the living docs / `.golangci.yml` / ADR.
3. Both agents thereafter run audits against the *same committed config*. Drift becomes a build failure, not a style argument.

**Two channels, deliberately split (clarified after Frictions #11/#12,
2026-06-13):**
- **PRIMARY — git.** The shared, committed enforcement layer (`.golangci.yml`,
  the pinned toolchain, `tools/ifaceguard/`) lives in the repo; both agents
  pull it and audit against identical config. This is the channel that makes
  "drift = build failure" true for all 38 sessions. It depends on CT105 being
  able to **push** — the gap Friction #12 found and the Confidence-80 session's
  Gate 3 (fine-grained PAT) closes. Until a push lands, this channel is not
  actually exercised, only assumed.
- **SECONDARY — SSH file relay.** For fast, ad-hoc, **read-only** review
  requests, `scp` the specific artifact (a new file or diff) to a clearly
  separate `/tmp/<topic>/` path on the target machine — **never** into agy's
  tracked clone — prompt agy at that path, and `scp` its written findings back.
  Measured at ~6 min round trip (T3). This is the right channel for "review this
  one new thing now," and it is **not** a substitute for the shared committed
  config (it carries no toolchain parity guarantee). Use git for "what both
  agents build against"; use SSH-relay for "look at this specific output."

### 9.3 Cross-pollination cadence — three triggers, not "review everything"
Post-commit review on all 38 sessions is relay fatigue and noise. agy's review fires on three triggers instead:

**(a) First-Instance Template Review Gate — the highest-leverage trigger.** This was Claude's pushback and agy agreed: mechanical checklists only catch code that *looks* dangerous, and miss the costliest error class — a flaw in a template that later sessions clone. **Before any session that inherits a template begins coding, agy audits and signs off on the template implementation.** The template-setting sessions agy identified from the Build_Tracker:
- **B0** (S1) — system boundaries, WAL hooks, IPC pattern → every session inherits.
- **B1** (S2) — HTTP transport / rate-limit / backoff → all fetchers inherit.
- **B5b-QB** (S14) — the four-function rubric skeleton → the other 9 rubrics clone it.
- **B5b-DT** (S15) — the defensive rubric pattern (Cushion Guard, dynamic α, SL-019 exclusion) → LB/CB/S/DE inherit.
- **B7a** (S26) — the atomic write-pipeline / sole-writer coordinator → B7b/c/d inherit.
- **M1** (S25) — the Zustand + Wails IPC binding pattern → all modules inherit.

**(b) High-risk mechanical checklist — per-session, auto-trigger.** Any session whose diff: introduces concurrency (`go`, `select`, `chan`, `sync`); modifies connection pooling or DB transaction boundaries; parses external data (the MFL boundary); or uses unsafe/reflection.

**(c) Open Audits at tier boundaries — the blind-spot catcher (9.5).**

### 9.4 Disagreement resolution — the machine arbitrates first; Christopher arbitrates the rest
Escalating every disagreement to Christopher is wrong; so is letting Claude win by role. The resolution ladder:
1. **Machine-decidable disputes resolve by machine.** Performance/safety → write a `go test -bench` and let the number decide. Style/pattern → conform to `.golangci.yml`. No human, no role-pulling.
2. **Structurally undecidable disputes go to Christopher — not to Claude, and not to a fresh agy.** This is where agy declined authority Claude offered it. Claude proposed a fresh-frame agy instance as a neutral tiebreak; **agy rejected that** — *"a clean-frame model has no stakes and will output generic software design arguments, adding noise rather than a decision."* agy's position: Christopher is the only correct arbiter for boundary/abstraction calls because he owns the complexity budget and the long-term maintenance stakes. To keep that fast for him, the two agents hand him a **Complexity-vs-Benefit matrix** — exactly two columns (complexity cost: lines of boilerplate, dependency count — vs. functional capability), max three items per column — and he picks. Builder-decides-with-a-rationale-comment is explicitly **not** sufficient for a genuine structural disagreement.
3. **Security disputes** stay Builder-final per `multi-agent-roles.md`.

### 9.5 Open Audits — scheduled, unscoped, to catch what Claude has normalized
Mechanical triggers cannot catch normalized drift *by definition*: if Claude could specify the trigger, it would already see the problem. So at four tier boundaries, agy runs an **Open Audit** — no checklist from Claude, full repository read access, agy looks for whatever smells wrong:
- **Audit 1 — Ingestion boundary** (after S8): transport, raw-storage schema, normalizer drift.
- **Audit 2 — Scoring engine** (after S23): scoring-leak into lower layers, rubric-formula drift, perf hotpaths — *before* state mutation begins.
- **Audit 3 — Transaction mechanics** (after S29): sole-writer lock patterns, state transitions — *before* UI freezes the API.
- **Audit 4 — Pre-handoff final** (after S38): cross-layer imports, UI/backend state mismatch, dead code.

agy classifies every finding as **Structural Drift** (ADR / Technical-Pillar violation), **Normalized Complexity** (abstraction not earning its keep, boilerplate bloat), or **Invisible Risk** (undocumented assumption, missing boundary check). Claude triages; disagreements follow 9.4.

### 9.6 Engagement framing, memory symmetry, toolchain parity (the operational hygiene)
- **Framing — defensive invariants, not euphemisms.** agy was clear: do not disguise a security ask. Phrase it as a concrete, closed-ended invariant. *Not* "audit for SQL injection" → *"verify every SQLite query uses parameterized placeholders and performs no string interpolation."* *Not* "find memory leaks" → *"check slice growth bounds and verify goroutine lifetimes close."* This engages agy's expertise without tripping its safety guard, because it is what Claude actually means.
- **Memory symmetry — corrected (Friction #11, 2026-06-13).** The original framing
  assumed agy is a stateless, prompt-only reviewer blind to repo state. That is
  **false in practice.** agy on CT104 has a **standing, self-updating clone of
  the real TheWarRoom repo** plus live web research, and it **will** read that
  clone (`Build_Tracker.md`, `AGENTS.md`, the MFL specs, conventions) regardless
  of how the prompt is framed. T3 proved this is a **feature, not a leak**:
  agy's host-discovery logic was correct *because* it read the real
  `MFL_API_Specification.md`, which was not in the brief. Two consequences:
  - **Do NOT assume agy is blind to repo state.** Review prompts should not be
    written as if agy can only see what is pasted in; it sees the committed repo
    too, and that context improves repo-aware reviews.
  - **The self-contained discipline still applies to the *artifact under
    review*** — the new file or diff agy does not yet have. Send that exactly
    (path + contents), because it is the one thing not in agy's clone. What is
    already committed, agy already has; do not waste relay budget re-sending it,
    and do not rely on "as we discussed" for prior-turn *conversation* (that, it
    genuinely does not retain — it truncates ~135k tokens, no cross-session
    chat memory). Repo state: yes. Chat history: no.
- **Toolchain parity (CT 104 vs CT 105).** The two agents run on separate LXCs with potentially different Go/linter/SQLite versions — a source of false-positive findings. **B0 commits a pinned toolchain definition and `.golangci.yml`; both agents run audits with matching commands.** Without this, a "finding" may just be a version skew.

### 9.7 What this changes about how to read the rest of this guide
Section 8's line that "this guide is the most junior voice in the room" stands for the *building model* against hooks and locked decisions — but it does **not** describe the Claude↔agy relationship. Between the two agents, neither is junior; authority is split by instrument (9.1), and on tooling and the open-audit blind-spot work, agy leads. A building model that hits a structural fork it cannot resolve should not guess — it should produce the 9.4 Complexity-vs-Benefit matrix and surface it for Christopher.

### 9.8 Triage Protocol — every concrete agy finding is checked against source before it escalates
Friction #13 (2026-06-13) measured why 9.4's "Claude triages" clause is
load-bearing, not ceremony: in a single First-Instance Template Review, agy
returned 8 findings — 6 real, 1 overstated, and **1 confident, line-specific,
"blocking"-severity finding that was simply wrong** (it cited `client.go:125`,
a specific variable, and a plausible failure mode; the cited line was actually
`} else {`). In *form* it was indistinguishable from the two findings that were
genuine logic bugs. Triage is what separated them, and it was cheap (~5 min)
precisely because the claim was concrete enough to check directly. The protocol:

1. **Every agy finding that cites a `file:line` or a specific symbol MUST be
   checked against the actual source before it is escalated to Christopher as a
   defect or acted on.** Read the cited range; confirm the claim describes what
   the code actually does, on the exact bytes agy read (verify cross-machine if
   the relay introduced any copy).
2. A finding that **survives** is escalated/fixed with the triage note attached
   ("confirmed at `client.go:NN`").
3. A finding that **does not survive** is logged in the friction record (for
   agy-calibration tracking) but **not** escalated and **not** acted on. It is
   not bounced back to a fresh agy for argument — a clean-frame model has no new
   information and will only add noise (9.4.2).
4. Vague findings ("there may be a host-resolution issue near line 125") are the
   expensive case — neither confirmable nor dismissible by direct read. Ask agy
   to make the claim concrete (exact line, exact mechanism) or drop it; never
   escalate an unfalsifiable finding.

This applies to all three review triggers (9.3) and to Open Audits (9.5). It is
the single most important operational lesson of the T1–T3 friction tests:
**peer review has real, measured value AND its output is not self-certifying —
both at once.** (T3: agy's review caught two genuine logic bugs that the 10/10
golangci-lint pass had no mechanism to see — invisible to linters, visible to a
second model reading for what the code *does*.)

---

## Section 10 — Locked Enforcement Decisions (Confidence-80 session, 2026-06-13)

Two enforcement mechanisms left open by the G0 overlay (Frictions #6 and #10)
were decided by Christopher this session and are now **LOCKED** — implemented,
tested, and committed in `christopher-coding-standards` (`session/go-overlay-g0`
@ `e48e352`). They are no longer "documented gaps." B0 inherits them as built.

### 10.1 AD-06 / RISK-003 — `playerid.PlayerID` bypass → struct-wrap (Gate 1)

**Decision:** `internal/playerid` defines `type PlayerID struct { id string }`
with an **unexported** field, not `type PlayerID string`.

**Rationale:** the original forbidigo rule could not distinguish a bypass
conversion (`playerid.PlayerID("99")`) from a legitimate type reference in a
signature, so it was unusable (Friction #6). A string newtype can always be
bypassed; a struct with an unexported field cannot — `playerid.PlayerID("99")`
from any other package fails to **compile**, making `New()` the only path to a
value and guaranteeing leading-zero normalization ("99" → "0099") can never be
skipped. This converts a lint-time social contract into a compile-time
guarantee. Cost was near-zero: no `playerid` code existed yet, so this is just
how the template is written the first time. Reference implementation:
`templates/go/playerid/example.go`.

**Consequence for B0 skeletons:** `PlayerID` is constructed via `playerid.New`,
stringified via `.String()`, and JSON round-trips through `New`
(`UnmarshalJSON`). It deliberately does **not** implement `driver.Valuer` /
`sql.Scanner` — the store layer serializes via `String()`/`New(text)` so the
domain package imports no `database/sql` (the depguard `sql-confined-to-data-
layer` rule confirmed this by rejecting a `Scan`/`Value` draft). All SQLite id
columns remain TEXT.

### 10.2 `interface{}`/`any` escapes → `ifaceguard` custom vettool (Gate 2)

**Decision:** a small custom `go/analysis` vettool (`tools/ifaceguard/`) flags
`interface{}`/`any` in **exported** signatures; it is run via `go vet -vettool`,
wired into `make lint` and pre-commit. `//ifaceguard:allow` is the escape hatch
for deliberate generic boundaries.

**Rationale:** no enabled golangci-lint linter catches a bare empty interface in
a signature (Friction #10) — `interfacebloat` only checks interface
declarations. A bare `interface{}` at a layer boundary turns off the type
checker there, which for the "Layer 2/4 zero scoring leak" rules is a silent
correctness hole. This was the one gap a checklist could not close at the
metric target, so a tool was proportionate — kept narrow (exported-only,
own-module, x/tools pinned by `go.sum`, `analysistest` regression guard) to stay
inside the same supply-chain discipline the overlay holds everywhere.

---

*End of companion plan. This document is additive: it modifies none of the existing planning artifacts, decisions, or wireframes. It is meant to be read by the models building TheWarRoom, session by session, to help them build code a human can maintain — and to prove that an AI build pipeline, properly instrumented and genuinely collaborative, produces engineering, not slop. Built by Claude Fable 5 with 4 Haiku fetch agents and 2 agy cross-reference agents, and refined through a direct Claude↔agy negotiation, 2026-06-12.*
