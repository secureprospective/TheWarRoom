HANDOFF — Session 28: B7b — Money→int64-cents migration + Contract Ops + (simple) Dead-Cap
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-03 (B7a session close)

== WHERE WE ARE ==
- B7a (squash on main 2026-07-03) is MERGED and functionally verified on the Beelink: a
  real trade persists (state + cap change on both franchises), and a bad-leg trade rolls
  back WHOLE (atomic). The `internal/transactions.Coordinator` is the sole `state.Writer`
  holder; every mutation runs through `state.WriteTx` (one SQLite tx + one reload) via the
  new `Writer = Reader + WriteTx` interface. Handlers live in `internal/transactions/
  acquisitions` behind the proven `transactions-only-through-coordinator` depguard.
- v1 transaction set shipped: TRADE (multi-leg, atomic) + ROSTER_STATUS. Dev IPC
  (`ExecuteTransaction`/`GetFranchiseState`, typed) + a React "B7a: Transactions (dev)" tab.
- GLM 5.2 blind review applied: committed-but-stale-reload → store POISON policy
  (`state.Err()`, retry-once-then-poison, reads/writes fail loud); no-op self-move rejected;
  trade legs capped (maxTradeLegs 256). All planted-tested.
- **OQ-014 is RESOLVED (2026-07-03, expert panel): money = `int64` CENTS.** Full record in
  `docs/roadmap/Roadmap_and_Open_Questions.md`. Companion league-rules locks: dead-cap v1
  = SIMPLE (salary + signing-bonus proration only); NO cap rollover (`cap_used ≥ 0` /
  `dead_cap ≥ 0` is a hard invariant).

== WHAT B7b IS ==
Build_Tracker row 27. The money-bearing transactions, now unblocked by the OQ-014 lock.
Three parts, in order:
1. **Money → int64 cents.** Additive migration: `ALTER TABLE contracts ADD COLUMN
   annual_salary_cents INTEGER`, `adjusted_salary_cents INTEGER`; backfill
   `CAST(ROUND(x*1e6) AS INTEGER)`; verify 0 mismatches; `CHECK(*_cents >= 0)`; keep the
   legacy REAL columns one release, then drop. **Kill the float64 intermediate**: parse
   the MFL decimal STRING straight to int64 cents (`normalize.parseSalary`,
   `internal/normalize/roster.go:103`). Money never round-trips through float64 or a JS
   number — string in, format at the render edge. Introduce a `domain.Money int64` (cents).
2. **Contract Ops** (`internal/transactions/contracts` handler behind the Coordinator):
   tag / restructure / extension / year-rollover via the existing `ApplyContract` path,
   now cents-typed. These are single-op transactions; validate cents ≥ 0.
3. **(simple) Dead-Cap** (`internal/transactions/deadcap`): on a cut/trade, accelerate a
   fraction of remaining guaranteed salary onto the franchise's cap. **Round the total
   ONCE at cents, then distribute with the largest-remainder (Hamilton) method** (ties →
   earliest year first) so `Σ splits == total` exactly. Key dead-cap rows to an ABSOLUTE
   league year, never relative slots. Dead-cap is a NEW store surface — decide table shape
   (a `dead_cap` ledger keyed (franchise, league_year)) at gate-check.

== READ FIRST ==
- `docs/roadmap/Roadmap_and_Open_Questions.md` → OQ-014 (the locked money decision + all mechanics)
- `internal/store/state/writes.go` + `types.go` (the WriteTx/TxWriter pattern to extend; ContractChange)
- `internal/transactions/` (coordinator.go, request.go, acquisitions/ — the handler + sealed-Request pattern to clone)
- `internal/normalize/roster.go` (parseSalary — the float64 intermediate to replace)
- Build_Tracker rows 24 (B6), 25 (M1), 26 (B7a)

== GATE CHECK (confirm with Christopher BEFORE code) ==
1. Migration timing — do the REAL→cents migration as its own first commit (recommended) or
   folded into B7b? An existing Beelink DB has live REAL rows to convert.
2. Dead-cap store shape — a new `dead_cap` ledger table (append-only, keyed by absolute
   league_year) vs derived columns on contracts. The rollover=OFF lock simplifies this.
3. Acceleration rule — what fraction accelerates, and over how many years, for a cut vs a
   trade? (League-rules input; drives the largest-remainder split.)
4. Signing-bonus proration model for v1 (the ONE fraction in scope) — straight-line over
   contract years? Confirm the divisor.

== CARRIED FORWARD — leads, not blockers ==
- Add/drop (waiver/FA) still deferred — needs a free-agent-pool concept in state (not yet
  modeled). Its own build after B7b.
- Poison policy currently guards WriteTx + GetFranchiseState; the M1/other read IPCs do NOT
  yet check `state.Err()` — extend the read-side poison guard when convenient (low-risk).
- WAL read-after-commit visibility across the split pools is relied upon (sound under wmu);
  a one-shot driver-pinning test is a nice-to-have (GLM could-not-verify #2).
- Money property test (randomized-ordering sum == byte-identical) is the load-bearing
  drift-catcher — write it WITH the cents migration, not after.
- GLM track record: B7a 3 (1 MAJOR+2 MINOR) / M1 3 / B6 1 / K 0 / S 0 / CB 0 / LB 0 / DE 0
  / TE 0 / WR 1 / RB 1 / DT 3 / QB 2.

== CONSTRAINTS ACTIVE ==
- Branch `session/b7b-contract-ops`. No work on main. Never --no-verify.
- CT105 gate: `export PATH=$PATH:/usr/local/go/bin`; `go build ./...`;
  `GOMEMLIMIT=1500MiB GOGC=20 make lint`; `go test -race ./...`. Beelink GUI:
  `wails dev -tags webkit2_41` (repo at /home/chris/opencode/TheWarRoom).
- Files < 400 lines (AD-17). Engine stays pure. Stores never import each other. New
  handler subpackages (contracts/deadcap) stay behind Coordinator.Execute — the depguard
  rule already denies them; PROVE it with a planted import.
- Money is int64 cents everywhere in Go; never float64, never a JS number for money.
- Review gate: GLM 5.2 BLIND, leads-not-findings, triage vs source. Bundle the diff +
  unchanged context into ONE self-contained prompt (opencode exploration hangs; z.ai direct).

== CLOSE GATE ==
- Lint 0 / race green / depguard proven / the money property test passes.
- Christopher EXECUTES a tag/restructure AND a cut-with-dead-cap live on the Beelink and
  sees cap + dead-cap change correctly, with conservation holding; a second read confirms
  persistence. Squash-merge after he confirms. Then handoff 29.
