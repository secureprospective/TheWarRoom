HANDOFF — Session 32: Season-Rollover Machinery (PLAYOFFS → OFFSEASON increment)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-11 (B7c §13 Special Situations session close)

== WHERE WE ARE ==
- B7c §13 Special Situations MERGED to main (squash `b64798f`, 2026-07-11; state-doc
  `a3f3d2c`). Retirement / Death (Gaines Adams) / Cap Relief all live, functionally gated on
  the Beelink, GLM-5.2 blind review SHIP (0 MAJOR; 1 MINOR applied — cap-relief snapped to the
  $10k grid, plant-tested `TestIntegration_CapReliefSnapsToGrid`).
- With §13 done, the entire §8–§13 transaction rulebook is built EXCEPT the season lifecycle:
  every op charges/counts against the ONE loaded `season` int, and there is no way to advance
  the league from one season to the next.
- The season-phase MACHINERY already exists (built with §12, D3): an append-only
  `season_phases` transition log, `domain.Phase` enum (OFFSEASON / REGULAR_SEASON / PLAYOFFS),
  the `ADVANCE_PHASE` op, and the default-deny op→phase gate. What does NOT exist is the
  ROLLOVER: incrementing the season integer when the calendar wraps PLAYOFFS → OFFSEASON.

== WHY THIS IS ITS OWN PANEL-GATED SESSION (do not skip) ==
- The §12 expert panel flagged the **season-int lifecycle as the critical must-fix** — it is a
  durable, standard-setting decision (it changes what "current season" means for cap, dead cap,
  remaining-year math, and per-season op counts). Per the Quality Gates, a durable decision goes
  through an **independent expert-AI panel FIRST** (parallel self-contained brief, triage vs
  source), BEFORE any code. Do NOT open this by writing Go.
- LOCKED INVARIANT that constrains the design (panel-unanimous, from §12): **offseason = START
  of season N** — the loaded `season` int is the season OFFSEASON belongs to. A rollover must
  preserve this: after PLAYOFFS(N) → OFFSEASON, the season int becomes N+1 and OFFSEASON is the
  start of N+1.

== OPEN DESIGN QUESTIONS FOR THE PANEL (triage vs `docs/league-rules/` + source) ==
1. **Where does the season int live and how does it move?** Today `season` is passed to
   `statepkg.New(pools, leagueID, season)` at startup. A rollover must persist the new season
   (a durable store value, not a constructor arg) and reload derived state against it. Is the
   season a column on a league-meta row? Derived from the latest `season_phases` row? Design it.
2. **What does a rollover DO to the ledgers?** Contract cells are absolute-year keyed
   (2026/2027/…), so they don't move — but CapUsed is computed for `season`, so incrementing the
   season naturally rolls the cap to the next year's cells. Confirm: dead_cap and cap_relief are
   `league_year`-keyed and this-season-scoped, so a new season zeroes them (rollover is off, §1).
   Verify this is the intended behavior and that nothing needs an explicit "carry" step.
3. **Per-season op counts** (`transaction_counts`, keyed by season) reset automatically when the
   season int advances — confirm the two-buyout / one-tag / one-extension limits reset correctly.
4. **The §11 in-season restructure unlock** and the §10 extension's is_restructured reset are the
   SHARED carry-forward that this session finally realizes — an extension's in-season unlock was
   "illusory" until season-rollover exists. Design how the rollover interacts with these flags.
5. **UFA promotion / free agency** — does a rollover promote the trailing UFA slot to a live
   free agent, or is that a separate free-agency build? (The "no re-bid on a bought-out player
   until next offseason" carry-forward is blocked on the FA pool, not on rollover — keep them
   separate.)

== LOCKED DECISIONS THAT CARRY (reuse, don't re-litigate) ==
- Money = int64 cents, FLAT math, universal `RoundToNearest10k` ($10k grid) on every figure.
  CapUsed = Σ(cells) + Σ(dead_cap) − Σ(cap_relief), floored at 0.
- Ledger is KING: cap / dead cap / remaining-years / cap-relief are DERIVED from the cells +
  the sibling ledgers, never stored as competing truth.
- Single-writer law (AD-02): all mutations through one spanning `WriteTx`. `TxWriter` is at its
  10-member interfacebloat cap — new season ops group into an embedded interface (the
  `SeasonScope` / `LedgerWriter` pattern) rather than adding bare methods.
- Ledgers are append-only + DOUBLE-immutable (write-only Go API + BEFORE UPDATE/DELETE
  RAISE(ABORT) triggers). `season_phases`, `dead_cap_ledger`, `cap_relief_ledger` all follow it.
- Phase machinery: `ADVANCE_PHASE` is commissioner-confirmed, no clock automation, any target
  allowed (rollback). The rollover is likely a NEW op (or an extension of ADVANCE_PHASE from
  PLAYOFFS) — the panel decides.
- Files: `internal/store/state/season_phase.go` (phase log), `.../cap_relief.go`,
  `.../ledger*.go`; `internal/transactions/deadcap/`, `.../contracts/`. Design docs:
  `docs/transactions/Season_Phase_Design.md`, `docs/transactions/Salary_Ledger_Design.md`,
  `docs/league-rules/Math_Rules_Reference.md`.

== BUILD GATES (every commit) ==
- `GOMEMLIMIT=1500MiB GOGC=20 make lint` → 0 (ifaceguard + filelen 400 + golangci-lint);
  `go test -race ./...` green; frontend `tsc` + `vite build` clean; depguard proven.
- Functional gate live on the Beelink (`/home/chris/opencode/TheWarRoom`, `wails dev -tags
  webkit2_41`); reset the clone to clean main BEFORE and AFTER the gate
  (`[[reference_beelink_functional_gate]]`).
- GLM-5.2 blind CODE review over SSH to the Beelink (leads-not-findings, triage vs source) —
  see `[[reference_glm_review_over_ssh]]`. Extract the assistant text from opencode's sqlite at
  `~/.local/share/opencode/opencode.db` (part table, session id from the log) — the `run`
  redirect only captures a stub; the analysis lands in the `reasoning` part.
- Expert PANEL design gate BEFORE code (GLM 5.2 + Gemini + DeepSeek, blind, parallel brief).

== OPEN THE SESSION BY ==
Branching fresh off main: `git checkout main && git pull && git checkout -b session/season-rollover`.
Then run the panel gate on the OPEN DESIGN QUESTIONS above before writing any Go.
