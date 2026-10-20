HANDOFF — Session 25: B6 — Per-Season Output Store (the FIRST persistence layer for scores)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B5b-K (row 23, squash on main 2026-06-28) is MERGED. **All 10/10 Layer-4 rubrics are now real**
  (QB RB WR TE DT DE LB CB S K). The Tier-2 Layer-4 arc is COMPLETE. K was the last: film ACTIVE
  (Madden 0.60 / NFLProduction 0.40, ±3% cap / steep 10.0 / inflection 0.50), RAS + breakout
  forced to exactly 1.000, Combined = film; AD-10 reversed + shipped.
- The engine pipeline (`internal/engine`, B5a) produces an `engine.Result` per player:
  BasePoints, AgePull, Layer4Output, ScoutingAdjusted, CapMultiplier, CapTier, AdjustedScore,
  Tiebreaker. It is PURE — it computes scores but persists NOTHING. Nothing has ever been written
  to durable storage from the score path; B6 is that first write.
- Three Layer-2 stores already set the STORE TEMPLATE B6 clones: `internal/store/rulebook` (B3b),
  `internal/store/state` (B3c), `internal/store/params` (B4). All share: a Reader/Writer split
  enforcing the single-writer law (AD-02/AD-05), `db.Pools` read/write separation (MaxOpenConns=1
  on the write pool), two-lock concurrency (`wmu` outer write-mutex + `mu` reader RWMutex),
  parameterized SQL, fail-loud on drift (`requireOneRow`), and planted-failure gates.
- Composition boundary (`internal/composition`) already maps PlayerSpec → engine inputs; the
  harness (`internal/harness`) runs the pipeline and exposes `Result` rows.

== WHY B6 IS NEXT / WHAT MAKES IT DIFFERENT ==
Build_Tracker row 24, WF B6. It is the FIRST store that holds ENGINE OUTPUT (the prior three hold
config + runtime state). Its defining constraint is DOUBLE immutability (AD-04, DECISION-010):
- **DECISION-010:** historical season scores are preserved under the ORIGINAL scoring config.
  Changing engine params NEVER retroactively re-scores a prior season. Every output record carries
  a `scoring_config_id`. New config → NEW records only; old records are immutable.
- **AD-04:** immutability is enforced BOTH ways — an append-only Go API (no Update/Delete methods
  on the Writer) AND a SQLite trigger that raises on any UPDATE/DELETE of an output row. The DB
  itself rejects mutation even if a future caller bypasses the Go API. This trigger is the headline
  new mechanic — it must be proven by a planted failure (attempt an UPDATE, assert it errors).

== WHAT B6 BUILDS ==
- New package `internal/output` (the store) cloning the B3c/B4 file shape:
  - A table (e.g. `season_scores`) keyed by (season, mfl_id, scoring_config_id) holding the
    persisted `engine.Result` fields. Decide with Christopher whether to flatten Result's
    sub-structs (Layer4Output, Tiebreaker) into columns or store a typed projection — recommend
    flattening the score-relevant scalars into columns (queryable), NOT a JSON blob (M1 will query
    rankings by AdjustedScore). Confidence values stay OFF the record (Hard Constraint, already
    true of Result).
  - **Writer (append-only):** `Write(season, scoringConfigID, []Result)` or per-player append;
    NO Update/Delete surface exists. DI to the score/transaction layer ONLY (mirror B3c's
    `StateWriter`). A re-write of the SAME (season, mfl_id, scoring_config_id) is a documented
    decision — recommend REJECT-as-drift (append-only means a duplicate key is an error, not an
    upsert), gate-check with Christopher.
  - **Reader:** `Reader()` returns a non-type-assertable view (clone B3c's `readerView` wrapper —
    unexported `*Store` field, NOT embedded, so it can't be cast back to the Writer; prove with a
    `TestReaderCannotMutate`). Reads for M1 rankings: by (season, scoring_config_id), sorted.
  - **The SQLite immutability trigger** (AD-04): a `CREATE TRIGGER ... BEFORE UPDATE/DELETE ...
    SELECT RAISE(ABORT, ...)` installed at schema init. Planted-failure proof: a raw UPDATE through
    the write pool must error.
- `scoring_config_id` SOURCE: it identifies the active scoring config (B3b rulebook's active
  version pointer is the natural source). Verify how rulebook exposes its active config id; the
  Writer must STAMP it, never invent it. Gate-check the exact provenance.

== GATE CHECK (confirm with Christopher BEFORE writing code) ==
1. Sequence: B6 next (recommended — completes the Tier-2 store floor; M1 row 25 is the first
   visible rankings view and depends on B6 + B3c). Confirm.
2. **Record shape** — flatten Result scalars into queryable columns (recommended) vs a typed
   blob. His call; M1 will query by AdjustedScore so flattening is the lean choice.
3. **Duplicate-key policy** — a second Write of the same (season, mfl_id, scoring_config_id):
   reject-as-drift (recommended, consistent with append-only + requireOneRow) vs ignore. His call.
4. **`scoring_config_id` provenance** — confirm it comes from the rulebook active-version pointer
   (B3b), stamped by the Writer; B6 never mints it.
5. Confirm AD-04 double immutability (Go append-only API + SQLite UPDATE/DELETE trigger) and that
   the trigger is proven by a planted UPDATE failure.

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch `session/b6-output-store`. Never `git --no-verify`.
- CT105: `export PATH=$PATH:/usr/local/go/bin`; `go build ./...`;
  `GOMEMLIMIT=1500MiB GOGC=20 make lint` (repo ROOT); `go test -race ./...`. Beelink GUI:
  `wails dev -tags webkit2_41`.
- Files < 400 lines (AD-17). Stores never import each other (depguard store-isolation rules) and
  the engine stays PURE — `internal/output` is a Layer-2 store; the engine must NOT import it
  (engine-is-pure already denies `internal/output`). Layer 1 cannot import it either (denied).
- Single-writer law: the write pool is MaxOpenConns=1; mutations serialize under `wmu`+`mu`
  (clone the B3c/B4 two-lock idiom). Reads deep-copy nested slices.
- Every custom mechanic proven by a planted failure (M3 — the trigger ABORT, the duplicate-key
  reject, the reader-cannot-mutate cast). Shared logic REUSED not copied (M17).
- Review gate: GLM 5.2 (OpenCode on bird), BLIND, output is LEADS — triage every finding vs
  source. INLINE a self-contained prompt (GLM/opencode HANGS on repo exploration) + the B3c store
  file as the template reference + the schema/trigger SQL + the Writer/Reader surface. Run:
  scp -i /root/.ssh/bird /tmp/prompt.md x@192.168.1.195:/tmp/ ; ssh -i /root/.ssh/bird
  x@192.168.1.195 'opencode run -m zai/glm-5.2 "$(cat /tmp/prompt.md)"'. scp/ssh have a standing
  settings.json allow rule. GLM track record: K 0 / S 0 / CB 0 / LB 0 / DE 0 / TE 0 / WR 1 / RB 1
  / DT 3 / QB 2.

== CARRIED FORWARD — leads, not blockers ==
- SL-018 receding RAS schedules / L3 RAS-buffers / coverage+IDP+offense film weights — all UNSET,
  pend the film-weight calibration pass (separate from any store work).
- K film cap/steepness/inflection (±3% / 10.0 / 0.50) are a v1.0 starting pin — revisit at the
  film-weight calibration pass. SL-OQ-041 (FG 70+ scoring) / SL-OQ-042 (Madden K pipeline) /
  CAL-032 carried.
- 3G (DT PFFAlpha assertion-wiring) still gatedPending; 3H (confidence floor) needs component
  confidence inputs on Layer4Input — not built.
- OQ-014 (Money-type / cap-math precision — `Salary float64` for now) still deferred to cap-math.
- AFTER B6: M1 — Asset Rankings (row 25), the first VISIBLE engine validation (32-team rankings +
  per-team roster drill-down, AD-20).

== CLOSE GATE FOR THIS SESSION ==
- go build + make lint 0 + go test -race green; depguard store-isolation green over the new
  `internal/output` pkg; engine-is-pure still green.
- AD-04 double immutability PROVEN: append-only Go API (no Update/Delete) + a planted raw-UPDATE
  that the SQLite trigger ABORTs. Reader-cannot-mutate proven by a failed cast. Duplicate-key
  policy planted-tested. DECISION-010: a record carries its scoring_config_id and an old record is
  never touched by a new-config write.
- GLM 5.2 BLIND review (sliced, inlined incl. the B3c template + schema/trigger SQL); triage.
- Functional gate: Christopher operates the harness — scores persist, a re-score under a new
  config creates NEW records without altering the old. Squash-merge after he confirms. Then write
  handoff 26 (M1 — Asset Rankings).
