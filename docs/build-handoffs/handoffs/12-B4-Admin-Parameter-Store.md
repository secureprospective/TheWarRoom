HANDOFF — Session 12: B4 — Admin Parameter Store (third Layer-2 store)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B3c — League State Store is COMPLETE + merged to main (2026-06-26, squash `52c9760`).
  Second Layer-2 store. `internal/store/state`: rosters + contracts + derived cap
  usage; Reader/Writer split (Writer→B7a only, un-assertable Reader); Initialize
  seeds ONCE from normalize, NO Reload (runtime state is B7-owned); fail-loud
  throughout (empty-seed, RowsAffected==1, orphan reconcile); franchise identity
  player-derived in v1.0 (documented). GLM 5.2 reviewed (2 fail-loud gaps fixed).
- B3b — League Rulebook (first Layer-2 store, the TEMPLATE) merged `12afc3c`.
- THE TWO STORE SHAPES NOW EXIST:
  - B3b rulebook = CONFIG store: immutable versioned snapshots + active pointer +
    Reload/Promote (side-load + commissioner gate) + override layer.
  - B3c state = MUTABLE store: single-writer split, seed-once, no Reload.
  B4 is a CONFIG store like B3b, NOT a mutable store — but SIMPLER than B3b: it
  ships built-in DEFAULTS (no external fetch source), with an admin override layer.
- THIS SESSION builds B4, the THIRD Layer-2 store. Branch fresh off main:
  git checkout main && git pull && git checkout -b session/b4-admin-parameter-store.
  Confirm scope with Christopher first.

== READ FIRST ==
- /root/.claude/plans/very-good-now-i-replicated-feigenbaum.md → "Wireframe 2 — Layer 2
  Config Stores", the **B4 row** (admin-only write path, ships defaults).
- /root/.claude/plans/session-3-audit-build-sequencing.md → AD-05 (admin write path,
  NOT through B7), AD-21 (cap-tier PERCENTAGES are B4 calibration — distinct from the
  cap AMOUNT, which is B3b config), and the OQ-006 note.
- docs/backend/Backend_Architecture.md — search for the engine calibration / parameter
  surface and the cap-tier (Cold/Neutral/Hot) thresholds. Get the verbatim parameter
  list + default values the engine expects.
- internal/store/rulebook/ — the CONFIG-store template B4 clones (override layer,
  validateOverride gate, parameterized SQL, db.Pools split, two-lock concurrency).
- internal/store/state/ — the most recent store; reuse its idioms where they fit.

== RECON (Haiku fan-out — run before design/build) ==
Spin a Haiku Explore subagent over the READ FIRST docs; ask for: the EXACT engine
calibration parameter set B4 must hold (names + types + default values + valid
ranges), VERBATIM — especially the cap-tier percentage thresholds (OQ-006 / AD-21);
which params are scalars vs. tables; what READS B4 (engine L-? / which modules);
and any OQ/AD touching calibration. Claude VERIFIES load-bearing claims against source
before code (the B3c handoff's scoping held, but the B3b "scoring in `league`" premise
was WRONG — never trust a handoff claim over live source).

== GATE CHECK (confirm with Christopher before writing code) ==
1. B4 v1.0 parameter scope: which calibration params ship as defaults this session
   (cap-tier percentages at minimum — AD-21). Confirm the full list from source.
2. Override model: clone B3b's scalar override layer (admin SetOverride + a
   validateOverride gate with ranges), OR a simpler defaults+override map? Recommend
   the B3b override pattern — it already encodes the admin-write + validation gate.
3. Is there ANY external source, or is B4 purely shipped defaults? (Plan says ships
   defaults — confirm no fetch source, which simplifies Initialize: seed from a Go
   defaults table, not a network/normalize pull.)

== WHAT THIS SESSION BUILDS (Build_Tracker row 11) ==
B4 — Admin Parameter Store: engine calibration parameters (cap-tier percentages and
any other tunables), SQLite-backed, with shipped defaults + an admin override layer.
  - CONFIG store (like B3b), admin-only write path — NEVER through B7 (AD-05). This
    is the M9a calibration surface's backing store.
  - Initialize seeds from BUILT-IN Go defaults on a fresh DB (no external fetch);
    loads existing on restart (stability, like B3b/B3c).
  - Typed getters for the engine (e.g. GetCapTiers() returning the Cold/Neutral/Hot
    thresholds); override-aware reads; validateOverride with RANGE checks (a tier
    percentage must be in a sane bound — the M3 planted-failure gate).
  - AD-21 discipline: B4 holds cap-tier PERCENTAGES (calibration); the cap AMOUNT
    stays in B3b. Do not duplicate the amount here.

== LOCKED DECISIONS (do not relitigate) ==
- AD-05: B4 is an ADMIN-write store. Its write path is admin-only and is NOT routed
  through B7 (B7 writes league STATE, not calibration config).
- AD-21: cap-tier percentages = B4 (calibration); cap amount = B3b (config). OQ-006
  (the actual tier values) is POST-LIVE calibration, not a build gate — B4 ships
  sensible defaults and the engine runs on them; tuning happens later via M9a.
- Layer law: B4 is Layer 2 — imports `db` (and `domain` if it needs typed params);
  imports NO other store; modules reach it read-only; M9a writes it via the admin path.

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch session/b4-admin-parameter-store. Never git --no-verify.
- CT105 build: warm cache `go build ./...`, then GOMEMLIMIT=1500MiB GOGC=20 make lint
  (run from repo ROOT — a stray cd into a package dir breaks `make`), then
  go test -race ./... . Go at /usr/local/go/bin (NOT on PATH).
- Every custom gate proven by a planted failure (M3 — e.g. an out-of-range override
  rejected). Shared logic extracted, not copy-pasted (M17). File < 400 lines
  (filelen); pre-split helpers into helpers.go if needed (AD-17, both B3b and B3c did).
- Review gate: GLM 5.2 (Z.ai Coding Plan, OpenCode on bird). agy/Gemini RETIRED.
  Reviewer works BLIND; output is LEADS — TRIAGE every finding vs source. (B3c's GLM
  review found 2 REAL fail-loud gaps; B3b's found 2 REAL concurrency bugs — it earns
  its keep; still triage. bird clone lives at ~/qa/repos/TheWarRoom; sync the branch
  with `git fetch origin <branch>:<branch>` — a plain fetch can race the push.)

== CARRIED FROM LAST SESSION (B3c) ==
- The CONFIG-store override pattern (SetOverride + validateOverride + a separate
  override record layered at read time) is proven in B3b — clone it for B4's admin
  layer; B4's validateOverride adds RANGE checks (percentages must be bounded).
- The two-lock concurrency idiom (wmu outer write lock + mu reader RWMutex; reads
  deep-copy; write then in-memory reload) is proven in both B3b and B3c — clone it.
- Fail-loud is the house style: guard empty/0-row/orphan paths and prove each with a
  planted test. requireOneRow (B3c) is a reusable shape if B4 does row updates.

== OPEN ITEMS CARRIED (not B4-blocking) ==
- B3c franchise identity is player-derived (v1.0). A future franchise-registry
  integration owns the canonical 32-team list; revisit when B6/M1 need "always 32".
- B3c cross-call read snapshots are not isolated across two reader calls (B7 holds
  wmu so it is safe; engine/IPC readers compose two calls at their own risk). Add a
  league-wide snapshot read API if a consumer needs it.
- OQ-013 (created→official player-id reconciliation) and OQ-014 (money float64 vs a
  Money type) remain open; OQ-014 will matter when cap math compares against B4 tiers.

== CLOSE GATE FOR THIS SESSION ==
- Build: make lint 0 + go test -race ./... green.
- B4 ships defaults: a fresh DB seeds the calibration params from Go defaults; the
  engine-facing typed getters return them; a planted out-of-range override is rejected.
- Initialize does NOT reseed an existing DB (stability, like B3b/B3c).
- The admin write path is admin-only (no B7 coupling); override layer works and
  survives restart.
- GLM 5.2 BLIND review; triage every finding vs source.
- Squash-merge to main after Christopher confirms; write the next handoff
  (B5a — Engine Pipeline, row 12) before clearing. NOTE: B5a depends on B3b · B3c · B4
  all three — with B4 done, the Layer-2 config floor is COMPLETE and the engine can
  start. Pre-check internal/store/* and any L2 file < 400 lines before B5a (AD-17).
