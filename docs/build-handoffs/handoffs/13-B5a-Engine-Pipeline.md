HANDOFF — Session 13: B5a — Engine Pipeline (the pure-function scoring spine)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- THE LAYER-2 CONFIG FLOOR IS COMPLETE. All three Layer-2 stores are merged to main:
  - B3b rulebook (`internal/store/rulebook`, squash `12afc3c`) — CONFIG: versioned
    snapshots + active pointer + Reload/Promote + override layer. Holds the cap AMOUNT.
  - B3c state (`internal/store/state`, squash `52c9760`) — MUTABLE: single-writer
    split, seed-once, no Reload. Holds rosters + contracts + derived cap usage.
  - B4 params (`internal/store/params`, squash `9f2c675`) — CONFIG: shipped Go
    defaults + admin override, generic (key,position) typed store. Holds the engine
    CALIBRATION params: cap-tier PERCENTAGES (GetCapTiers → Cold 1.2 / Hot 4.8),
    Layer-3 decay (0.03), Cushion Guard (8.00/0.10). Typed getters: GetCapTiers(),
    GetGlobal(Key*), Definitions().
- With B3b · B3c · B4 all done, B5a is UNBLOCKED (Build_Tracker row 12).
- THIS SESSION builds B5a, the engine pipeline. Branch fresh off main:
  git checkout main && git pull && git checkout -b session/b5a-engine-pipeline.
  Confirm scope with Christopher first.

== THE ONE HARD ARCHITECTURAL FACT (do not violate) ==
- THE ENGINE IS A PURE-FUNCTION PIPELINE. depguard rule `engine-is-pure` (.golangci.yml)
  DENIES `internal/engine/**` from importing the stores, db, normalize, ingestion,
  transactions, output, net/http, net, os. EVERY input arrives as a PARAMETER.
  → The engine does NOT read B3b/B3c/B4. A composition boundary OUTSIDE the engine
    (the App/IPC layer, a later session) reads the stores and PASSES values in:
    PlayerRecord (B3c/normalize), ScoringConfig + cap amount (B3b), calibration
    params incl. CapTiers + decay (B4). B5a defines the engine's INPUT STRUCTS so
    that boundary has a typed contract to fill — but B5a itself imports no store.

== READ FIRST ==
- docs/backend/Backend_Architecture.md — the pipeline block (search "Layer 1:
  hygiene.Apply"). Verbatim layer order, each layer's signature, and the two
  accumulation points: ScoutingAdjusted = BasePoints × AgePull × Layer4Output.Combined,
  then AdjustedScore = ScoutingAdjusted × CapMultiplier.
- docs/scoring-engine/Engine_Specification.md — the full L1–L6 spec (the authority on
  each layer's math; the position rubric files are L4 detail, deferred to B5b).
- docs/build-handoffs/Layer4_PreBuild_Audit.md — §1D (the admin-tunable param list:
  which calibration values each layer consumes; cross-check what B4 actually ships
  vs. what is still a per-position table that arrives as a parameter, not from B4).
- internal/store/params/ — the calibration getters the COMPOSITION layer will call
  (GetCapTiers, GetGlobal). The engine receives their RESULTS, not the store.
- internal/domain/ — PlayerRecord and the enums the pipeline reads.

== RECON (Haiku fan-out — run before design/build) ==
Spin a Haiku Explore subagent over the READ FIRST docs; ask for, VERBATIM: each
layer's exact function signature + return type + formula (L1 hygiene, L2 scoring,
L3 decay, L5 capscaling, L6 tiebreaker); the EXACT shape of Layer4Output and the
TiebreakerKey; which calibration inputs each layer needs and WHETHER B4 ships them
(global) or they are PER-POSITION (arrive as a parameter, B5b's job); and how L4 is
meant to be "pluggable dispatch" (interface? function value? per-position registry?).
Claude VERIFIES load-bearing claims against source before code — the B3b "scoring in
`league`" premise was WRONG; never trust a handoff/recon claim over live source.

== GATE CHECK (confirm with Christopher before writing code) ==
1. L4 dispatch shape: B5a builds "L4 pluggable dispatch" but NOT any position's L4
   (that is B5b-QB onward). Confirm the seam: an interface the pipeline calls (e.g.
   `Layer4 interface { Apply(...) Layer4Output }`) injected per position, with a
   trivial identity/no-op default so B5a's pipeline is testable end-to-end before any
   rubric exists. Recommend interface-injection (mirrors B3c's Writer/Reader DI seam).
2. Input-struct boundary: confirm B5a OWNS the engine input structs (the typed
   contract the composition layer fills from the stores) but imports NO store. Where
   do the structs live — `internal/engine/` root, or an `engine/types.go`?
3. Per-position calibration (peak_limit, S-curve params, …) that B4 does NOT ship as
   globals: confirm these arrive as PARAMETERS on the layer call, sourced later by
   B5b, not added to B4 now (consistent with "ship per-position params WITH the layer
   that consumes them").

== WHAT THIS SESSION BUILDS (Build_Tracker row 12) ==
B5a — Engine Pipeline: `internal/engine/` — orchestrates L1–L3 + L5–L6 as pure
functions, with L4 as a pluggable dispatch seam (no position rubric yet).
  - Each layer a pure function (no I/O); the pipeline chains them in order and
    accumulates ScoutingAdjusted then AdjustedScore exactly per Backend_Architecture.
  - L4 is an injected interface with an identity default so the whole pipeline is
    unit-testable now; B5b-QB (row 14) provides the first real L4.
  - Engine input structs defined here (the composition-layer contract); NO store import.
  - Confidence scores stay INTERNAL (never surfaced) — a Hard Constraint.

== LOCKED DECISIONS (do not relitigate) ==
- Engine purity (depguard `engine-is-pure`): inputs are parameters; no store/db/IO.
- Layer order + the two accumulation points are fixed by Backend_Architecture — do not
  reorder or fold layers.
- AD-21: cap-tier percentages come from B4 (GetCapTiers); the cap AMOUNT from B3b. The
  engine receives both as plain values via the composition layer.
- L4 has no overall cap — component caps are the natural bounds (Backend_Architecture).
- Confidence scores are internal engine flags only — never in any output struct that
  reaches the UI (Hard Constraint).

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch session/b5a-engine-pipeline. Never git --no-verify.
- CT105 build: warm cache `go build ./...`, then GOMEMLIMIT=1500MiB GOGC=20 make lint
  (from repo ROOT), then go test -race ./... . Go at /usr/local/go/bin (NOT on PATH).
- AD-17 file-cap pre-check: every engine file < 400 lines; pre-split helpers. NOTE:
  `internal/store/rulebook/rulebook.go` is 378 lines — if B5a touches any L2 store
  surface (it should NOT need to), pre-split first. The engine package is fresh.
- Every custom gate proven by a planted failure (M3). Shared logic extracted (M17).
- Pure-function layers are a TESTING GIFT: table-driven unit tests per layer with no
  DB/network. Prove the accumulation math and the L4 identity-default end to end.
- Review gate: GLM 5.2 (Z.ai Coding Plan, OpenCode on bird). agy/Gemini RETIRED.
  Reviewer works BLIND; output is LEADS — TRIAGE every finding vs source. (B4's GLM
  review found a real NaN gate bypass; B3c's found 2 fail-loud gaps; B3b's found 2
  concurrency bugs — it earns its keep; still triage. bird clone: ~/qa/repos/TheWarRoom;
  sync with `git fetch origin <branch>:<branch>`. Reach bird: `ssh -i /root/.ssh/bird
  x@192.168.1.195`; run `opencode run --agent review "<brief>"`.)

== CARRIED FROM B4 (params store) — forward-risk leads, not B5a blockers ==
- L2 (params): `hasDefaults` is row-count-only (n>0). A partial seed would masquerade
  as complete; today seedDefaults is transactional so it can't happen. Revisit if a
  param migration is added (consider n >= len(defaultParams()) or a content hash).
- L5 (params): SetOverride reloads BOTH full tables on every write. Fine at v1.0 row
  counts; revisit when per-position tables make the table large.
- L6 (params): Definitions sort-test only checks the Key order, not the (Key,Position)
  tiebreaker — add a position case once a per-position fixture exists.
- B4 ships NO per-position params yet (by design). Per-position calibration tables
  (peak_limit, S-curve, EMA α, etc.) land in B4 WITH the B5b layer that consumes them.

== OPEN ITEMS CARRIED (older, not B5a-blocking) ==
- B3c franchise identity is player-derived (v1.0); a future registry owns "always 32".
- B3c cross-call reader snapshots aren't isolated across two reader calls (B7 holds
  wmu so it is safe; add a league-wide snapshot read API if a consumer needs it).
- OQ-013 (created→official player-id reconciliation), OQ-014 (money float64 vs a Money
  type — matters when cap math compares vs B4 tiers).
- pfrcoverage aggregate-NA silent drop + veteranfilm join-key normalization — verify at
  calibration live-fetch.

== CLOSE GATE FOR THIS SESSION ==
- Build: make lint 0 + go test -race ./... green.
- The pipeline runs end to end on a synthetic PlayerRecord with the L4 identity
  default, producing AdjustedScore = BasePoints × AgePull × 1.0 × CapMultiplier, with
  table-driven per-layer tests and a planted-failure gate on at least one layer.
- The engine imports NO store/db/IO (depguard green proves it).
- Confidence scores are not present in any UI-bound output struct.
- GLM 5.2 BLIND review; triage every finding vs source.
- Squash-merge to main after Christopher confirms; write the NEXT handoff
  (14 — Testing Harness OR B5b-QB; NOTE the Testing Harness, row 13, is a HARD GATE —
  no rubric/B5b starts without it). Confirm sequencing with Christopher at close.
