HANDOFF — Session 26: M1 — 32-Team Asset Rankings (the FIRST visible engine validation)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- B6 (row 24, squash `025e2a8` on main 2026-06-28) is MERGED. **The Layer-2 store floor is
  COMPLETE** — all four stores live: rulebook (B3b), state (B3c), params (B4), output (B6).
- The pure engine (B5a) + ALL 10/10 real Layer-4 rubrics (B5b QB…K) are done. `Pipeline.Score`
  turns a `PlayerInput`+`Calibration`+`ScoutingInput` into an `engine.Result`.
- `internal/output` persists `engine.Result` rows tagged by `(season, scoring_config_id,
  mfl_id)`, append-only + DB-immutable, and reads them back in FINAL RANKING ORDER (the L6
  tiebreaker is encoded in the ORDER BY). `Write(ctx, season, scoringConfigID, []ScoreRecord)`
  / `Reader().Scores(...)` / `Reader().Score(...)`.
- `internal/composition` already maps a `PlayerSpec` → engine inputs through narrow PORT
  interfaces (fake-testable, no live DB), and `internal/harness` runs the pipeline over a
  SANDBOX fixture set and exposes it via Wails IPC + a React board. M1 is where that sandbox
  becomes a REAL league-wide run.

== WHY M1 IS NEXT / WHAT MAKES IT DIFFERENT ==
Build_Tracker row 25, Module 1. Every build so far has been engine/store plumbing proven by
tests. M1 is the FIRST build a human SEES: it runs all 32 real rosters through the engine and
renders a ranked board. That changes the shape of the work — it is an ORCHESTRATION + UI build,
not a new pure layer:
- It is the first code that READS three stores at once (B3c rosters/contracts, B4 calibration,
  B3b active config) AND the players DB, assembles engine inputs per rostered player, scores
  them, PERSISTS to B6, and reads them back ranked. This "score the league" coordinator is new.
- It is the first build whose close gate is a real CLICK-THROUGH on the Beelink, not just a test.
- AD-20: it ships BOTH the league-wide ranked list AND a per-team roster drill-down. Module spec
  also names position-filtered and cap-efficiency (Adjusted Score per $ of salary) views.

== THE BLOCKER TO RESOLVE FIRST — BasePoints (L2 is deferred) ==
`engine.PlayerInput.BasePoints` is a SUPPLIED input — B5a deliberately did NOT build L2 base
scoring (no L2 file; the harness fed fixed sandbox values). A real ranked board needs a real
BasePoints per player, or the rankings are meaningless. This is the FIRST gate-check question,
before any code. Options (gate-check with Christopher):
  (a) Build a minimal L2 now from MFL weekly scoring history (the league's own fantasy points)
      as BasePoints — most faithful, but it is a new fetch + aggregation block (scope grows).
  (b) v1.0 visible-validation placeholder: feed MFL season fantasy points (or a documented proxy)
      as BasePoints, clearly labeled "L2 pending", so the board renders and the engine path is
      validated end to end now; swap in real L2 later. (Recommended to keep M1 visible-first.)
  (c) Defer M1 until a real L2 block ships.
Do NOT pick silently — the choice changes M1's scope materially.

== WHAT M1 BUILDS ==
- A "score-the-league" orchestrator (likely extend `internal/composition`, or a new sibling that
  composition feeds): for each rostered player in B3c, assemble `PlayerInput` (incl. BasePoints
  per the resolved blocker) + `Calibration` (B4 globals + per-position B5b values) +
  `ScoutingInput` (from the B2b scouting data, or Data-Parity neutral where a signal is absent —
  the rubrics already neutralize absent sub-signals), run `Pipeline.Score`, collect
  `[]output.ScoreRecord` (MFLID + Result), and `output.Writer().Write(...)` tagged by the
  rulebook ACTIVE config id.
- `scoring_config_id` provenance: rulebook exposes only an UNEXPORTED `activeVersion(ctx)`. Add a
  PUBLIC accessor (e.g. `rulebook.ActiveVersion(ctx) (int, error)`) and have the orchestrator
  stamp B6 with it. B6 never mints it. (Confirm this is the right source — it is the natural one.)
- Wails IPC + React views (extend the harness sandbox shell or add a real module view): the
  global ranked list (by AdjustedScore), the per-team drill-down (AD-20), position filter, and
  cap-efficiency (AdjustedScore ÷ effective salary). Read straight from B6 via the Reader for the
  ranked order; the drill-down filters by franchise (join B3c roster → B6 score on mfl_id).
- Idempotency: re-running a score for an already-scored (season, config) hits B6's append-only
  drift guard (ErrDuplicate). Decide the UX: skip-if-present, or require a config bump. Gate-check.

== GATE CHECK (confirm with Christopher BEFORE writing code) ==
1. **BasePoints / L2** — option (a) build minimal L2, (b) labeled placeholder proxy (recommended,
   keeps M1 visible-first), or (c) defer. His call — it sets M1's scope.
2. **Scope / M1b split** — ship global + per-team + position + cap-efficiency in one session, or
   split the per-team drill-down to M1b (Build_Tracker note) if scope tightens. His call.
3. **`scoring_config_id` source** — confirm rulebook active version via a new public accessor,
   stamped by the orchestrator.
4. **Re-score UX** — on an already-scored (season, config): skip, or require a new config id.
5. **Data realism** — run on the REAL MFL league already loaded into B3b/B3c (not the sandbox cap
   "1000"), with scouting signals Data-Parity-neutral where a fetcher hasn't been wired into the
   orchestrator yet (the rubrics already handle absent sub-signals — no fake zeros).

== CONSTRAINTS ACTIVE THIS SESSION ==
- No work on main; branch `session/m1-asset-rankings`. Never `git --no-verify`.
- CT105: `export PATH=$PATH:/usr/local/go/bin`; `go build ./...`;
  `GOMEMLIMIT=1500MiB GOGC=20 make lint`; `go test -race ./...`. Beelink GUI:
  `wails dev -tags webkit2_41` (needs `libwebkit2gtk-4.1-dev`).
- Files < 400 lines (AD-17). The engine stays PURE (engine-is-pure). The orchestrator READS
  stores and writes ONLY through `output.Writer` — it is NOT a store and must not duplicate one.
  Stores never import each other; the orchestrator may import several (it is composition, the one
  place that is allowed to).
- Every custom mechanic proven by a planted failure (M3). Shared logic REUSED not copied (M17).
- Review gate: GLM 5.2 (OpenCode on bird), BLIND, output is LEADS — triage every finding vs
  source. INLINE a self-contained prompt (GLM/opencode HANGS on repo exploration): the new
  orchestrator + the composition PORT interfaces + the B6 Reader/Writer surface + engine.Result.
  Dispatch: `scp -i /root/.ssh/bird /tmp/prompt.md x@192.168.1.195:/tmp/ ; ssh -i /root/.ssh/bird
  x@192.168.1.195 'opencode run -m zai/glm-5.2 "$(cat /tmp/prompt.md)"'` (standing settings.json
  allow rule). GLM track record: B6 1 / K 0 / S 0 / CB 0 / LB 0 / DE 0 / TE 0 / WR 1 / RB 1 /
  DT 3 / QB 2.

== CARRIED FORWARD — leads, not blockers ==
- SL-018 receding RAS schedules / L3 RAS-buffers / coverage+IDP+offense film weights — all UNSET,
  pend the film-weight calibration pass. The rubrics neutralize absent signals, so M1 renders
  correctly without them (it just won't yet reflect film/breakout movement for positions whose
  scouting isn't wired into the orchestrator).
- K film cap/steepness/inflection (±3% / 10.0 / 0.50) are a v1.0 starting pin — revisit at the
  film-weight calibration pass. SL-OQ-041 / SL-OQ-042 / CAL-032 carried.
- 3G (DT PFFAlpha assertion-wiring) still gatedPending; 3H (confidence floor) needs component
  confidence inputs on Layer4Input — not built.
- OQ-013 (created→official player-id reconciliation) and OQ-014 (Money-type / cap-math precision,
  `Salary float64`) still deferred.
- AFTER M1: B7a — Transaction Foundation + Coordinator (row 26), the sole-writer transaction layer
  that wires B3c's StateWriter.

== CLOSE GATE FOR THIS SESSION ==
- go build + make lint 0 + go test -race green; engine-is-pure still green; store-isolation green.
- The orchestrator scores all 32 real rosters, persists to B6 tagged by the active config id, and
  the views read back from B6. Any new mechanic (config-id stamping, re-score guard, cap-efficiency
  derivation) proven by a test, with a planted failure where it is a custom gate.
- GLM 5.2 BLIND review (sliced, inlined); triage vs source.
- FUNCTIONAL GATE (the real one this time): Christopher operates it live on the Beelink — the
  32-team ranked board renders, a team drill-down shows that team's players ranked, the position
  filter and cap-efficiency view work, and a re-score under a new config produces a fresh board
  without altering the old (DECISION-010 visible end to end). Squash-merge after he confirms.
  Then write handoff 27 (B7a — Transaction Foundation + Coordinator).
