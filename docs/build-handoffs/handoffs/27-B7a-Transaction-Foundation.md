HANDOFF — Session 27: B7a — Transaction Foundation + Coordinator (the sole runtime mutator)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-02T21:30-04:00 (M1 session close)

== WHERE WE ARE ==
- M1 (row 25, squash `2105f6a` on main 2026-07-02) is MERGED and functionally verified on the
  Beelink: the 32-team board renders from B6, drill-down/position/cap-efficiency views work,
  skip-if-present holds. The engine path is validated END TO END on real league data.
- The full store floor is live IN THE APP: app.go constructs rulebook + state + params + output
  over shared pools; stores seed from live MFL only on a fresh DB. `internal/rankings` is the
  composition-class orchestrator template (ports, fake-tested, planted gates).
- BasePoints is still the LABELED MFL-YTD proxy (L2 pending). Scouting sub-signals are
  Data-Parity absent in the M1 path (no scouting fetcher wired into the orchestrator yet).
- B3c shipped the Writer surface (MovePlayer / SetRosterStatus / ApplyContract) at B3c but
  NOTHING holds it yet — `state.Writer` is DI'd to NOBODY. That is B7a's whole point.

== WHAT B7a IS ==
Build_Tracker row 26. The transaction layer: the ONE component that mutates league state at
runtime (AD-02 single-writer law). Everything before it reads; B7a writes.
- `internal/transactions` root package: a `Coordinator` holding the injected `state.Writer`
  (the ONLY holder in the process) and exposing `Execute(ctx, Transaction) (Receipt, error)`.
- Handler subpackages (acquisitions / contracts / deadcap) behind the Coordinator — the
  depguard rule `transactions-only-through-coordinator` ALREADY EXISTS and denies importing
  them from outside; prove it with a planted import (M3).
- Transactions are atomic per B3c's write path (single-writer pool, per-player tx,
  requireOneRow drift guard). The Coordinator sequences multi-step transactions (e.g. a trade
  = N MovePlayer + contract updates) — decide: one SQLite tx spanning steps (B3c writes take
  the pool; may need a tx-scoped variant) vs sequential per-player txs + compensation. GATE-CHECK.
- Admin params stay OUT (AD-05: B4 has its own admin path, never through B7).

== READ FIRST ==
- Backend_Architecture §Transactions + AD-02/AD-05 (the single-writer law and its boundaries)
- `internal/store/state/writes.go` (the Writer surface + inTx/execPlayer/requireOneRow)
- `internal/rankings/rankings.go` (the current composition-template style to clone)
- Build_Tracker rows 10 (B3c) and 25 (M1)

== GATE CHECK (confirm with Christopher BEFORE code) ==
1. B7a v1 transaction set — which transaction types ship first (trade? add/drop? tag?
   contract rollover?). His call; the league's real workflow drives it.
2. Atomicity shape — one spanning tx vs per-step + compensation (above).
3. Dead-cap semantics — salaryadjustments ledger interaction (OQ-005 resolved the FETCH;
   the write-side derivation is B7 territory; OQ-014 Money-type may resurface here).
4. UI surface this session or B7b — a transaction has real consequences; the functional
   gate needs SOME operable surface (even a dev-only IPC form).

== CARRIED FORWARD — leads, not blockers ==
- L2 base scoring still pending (M1 proxy explicitly labeled); film/RAS-buffer calibration
  passes pending; 3G/3H harness cases still gatedPending.
- OQ-013 (created→official id reconciliation) and OQ-014 (Money precision) still open —
  OQ-014 is LIKELY to bite in B7a's contract math; flag it at gate-check.
- M1 NOTEs accepted as tradeoffs (documented in the review commit): Scores/Write TOCTOU,
  shared 120 s ScoreLeague budget, process-lifetime directory cache.
- GLM dispatch lesson (2026-07-02): bird was down → paste.md manual relay worked; the
  FULL-prompt opencode run HANGS — per-file chunks via Z.AI direct was the reliable path.
  GLM track record: M1 3 / B6 1 / K 0 / S 0 / CB 0 / LB 0 / DE 0 / TE 0 / WR 1 / RB 1 / DT 3 / QB 2.

== CONSTRAINTS ACTIVE ==
- Branch `session/b7a-transactions`. No work on main. Never --no-verify.
- CT105 gate: `export PATH=$PATH:/usr/local/go/bin`; `go build ./...`;
  `GOMEMLIMIT=1500MiB GOGC=20 make lint`; `go test -race ./...`. Beelink GUI:
  `wails dev -tags webkit2_41`.
- Files < 400 lines (AD-17). Engine stays pure. Stores never import each other. The
  Coordinator is the ONLY `state.Writer` holder — prove the boundary (M3), don't assert it.
- Review gate: GLM 5.2 BLIND, leads-not-findings, triage vs source. Chunk per-file.

== CLOSE GATE ==
- Lint 0 / race green / all depguard rules proven where new surface appears.
- Christopher EXECUTES a real transaction in the app (or dev IPC) on the Beelink and sees
  state + cap usage change correctly, then a second read confirms persistence. Squash-merge
  after he confirms. Then handoff 28.
