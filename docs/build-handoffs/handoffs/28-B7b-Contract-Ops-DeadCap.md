HANDOFF — Session 28: B7b — Money→int64-cents migration + Contract Ops + (simple) Dead-Cap
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-03 (B7a session close)

== PROGRESS (updated 2026-07-04, session in flight on branch session/b7b-contract-ops) ==
- v1 rescoped (rulebook audit): B7b = 3 commits — (1) money→int64 cents, (2) dead_cap_ledger
  store, (3) waiver-cut op. Restructure/tag/extension/buyout/§13 → B7c. NFL-model leakage
  (signing bonus, Hamilton distribution, per-game/÷5 rationale) STRUCK from OQ-014 + this
  handoff. See the corrected GATE CHECK / dead-cap section below.
- [x] COMMIT 1/3 DONE — money→int64 cents. `domain.Money` (cents) + exact string→cents
  parser (no float64), additive cents-column migration (fresh DDL CHECK>=0 + idempotent
  ALTER/backfill/0-mismatch-verify for existing DBs), state store fully cents-typed, float
  only at 3 documented edges (engine L5 cap-ratio, M1 + txn IPC display DTOs). No behavior
  change. Gate: build ok / lint 0 / race green (+ money & migration tests). Commits 64faa3d
  (docs) + 0daec43 (code), on branch, NOT yet merged to main.
- [x] COMMIT 2/3 DONE — dead_cap_ledger store surface. Append-only table keyed
  (league_id, franchise_id, league_year, mfl_id), dead_cap_cents INTEGER CHECK(>=0), BEFORE
  UPDATE/DELETE RAISE(ABORT) triggers (B6 double-immutability). TxWriter.AddDeadCap append
  primitive (WriteTx-only); store records verbatim, no §8 formula (B3c divergence). CapUsed
  = live contracts + this-season ledger charges (league_year==season); dead-cap-only
  franchise still surfaces. Split schema/migration into schema.go (AD-17 cap). Gate: build
  ok / lint 0 / race green (+5 dead-cap tests). Commit c0c4aa6, on branch, NOT merged.
- [x] COMMIT 3/3 DONE — §8 waiver-cut op. Waiver{MFLID} sealed Request +
  internal/transactions/deadcap (pure Charge() §8 formula + Waive() read→compute→release→
  charge in one spanning tx). Depguard PROVEN (planted used import from rankings fired the
  deny, reverted). Remaining years = expiration_year − season (exclusive; final-year/UFA =
  0 → $0), CONFIRMED w/ Christopher. state.TxWriter gained ReleasePlayer/Player/Season.
  Whole §8 charge lands in the CUT year (LeagueYear = season). IPC + React "Waiver (cut)"
  wired = live gate surface. Gate: build ok / lint 0 / race green / tsc+vite clean.
  Commit 2fbc08b, on branch, NOT merged.
- [ ] CLOSE GATE PENDING — (1) push branch, live Beelink cut → dead cap hits ledger, cap
  moves (salary out / §8 in), conservation holds, 2nd read persists; (2) BLIND review —
  GLM 5.2 DOWN this session, so Christopher hand-delivers to Gemini (leads-not-findings,
  triage vs source); (3) squash-merge after Christopher confirms live → then handoff 29.

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
  `docs/roadmap/Roadmap_and_Open_Questions.md`. Companion league-rules locks (rulebook-
  grounded, CORRECTED 2026-07-04): dead-cap = the §8 waiver formula **35% × annual salary ×
  remaining years** (50% if restructured, §11), ZERO if claimed off waivers. There is NO
  signing-bonus concept in this league — the earlier "signing-bonus proration" was NFL-model
  leakage and is STRUCK. NO cap rollover (`cap_used ≥ 0` / `dead_cap ≥ 0` is a hard
  invariant — §1 fixes a $125M/yr cap with no carry-forward).

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
3. **Dead-Cap** (`internal/transactions/deadcap`): on a WAIVER/cut, the releasing franchise
   owes the §8 formula **35% × annual salary × remaining years** (50% if the contract was
   restructured, §11); ZERO if the player is claimed. This is FLAT MATH, exact in integer
   cents — there is NO fractional distribution, so no Hamilton/largest-remainder step (that
   was NFL-model leakage; see the OQ-014 correction). Key dead-cap rows to an ABSOLUTE
   league year, never relative slots. Dead-cap is a NEW store surface — decide table shape
   (a `dead_cap` ledger keyed (franchise, league_year)) at gate-check.
   NOTE: trades do NOT create dead cap in this league (§14: salary-cap trading abolished
   2017; a traded player's contract goes with him). Dead cap arises on waivers/cuts (§8),
   restructure-then-waive (§11), buyout (§12), and retirement (§13, 30%).

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
3. B7b SCOPE — RESOLVED 2026-07-04 with Christopher. **B7b v1 = cents migration +
   dead_cap_ledger + WAIVER-cut (§8) ONLY.** All three fully rulebook-grounded. Deferred to
   **B7c ("Contract Ops II")**: restructure §11, tag §9, extension §10, buyout §12,
   retirement/death/cap-relief §13 — each drags a heavier dependency (cross-store position
   aggregation for tag, a season-phase/offseason concept for buyout, position-floor tables +
   multi-op history for extension) or is admin/commish (§13). Waiver has NO per-season limit
   and its re-sign gate is a no-op until FA exists, so v1 needs NO eligibility-counter
   surface (transaction_log defers to B7c with restructure).
   NOTE for B7c restructure (§11): the "move" AMOUNT is the OWNER's strategic decision,
   BOUNDED by the tier max ($1M/$2M/$3M by contract-year salary) — NOT a mechanically uniform
   formula. §6 (flat salaries) and §11 both apply; do not blend them. is_restructured → a
   later waive reads 50% dead cap instead of 35%.
   [RESOLVED — items 3 (acceleration fraction) and 4 (signing-bonus proration) STRUCK: both
   were NFL-model leakage. Dead cap is the flat §8 formula; no signing bonus exists.]

== CARRIED FORWARD — leads, not blockers ==
- Add/drop (waiver/FA) still deferred — needs a free-agent-pool concept in state (not yet
  modeled). Its own build after B7b.
- [GEMINI B7b REVIEW, triaged] dead_cap_ledger UNIQUE (league,franchise,year,mfl) assumes
  ONE cut per player-per-franchise-per-year. Sound today (§8 bars re-signing a cut player;
  no FA path returns a released player to a roster). WHEN FA/re-acquisition lands, a valid
  second cut becomes possible → the ledger must key on the cut EVENT (seq/nano in the PK,
  drop the UNIQUE) or the second AddDeadCap aborts. Documented in writes.go:AddDeadCap. Do
  the fix WITH the FA build, not before (the guard also blocks accidental double-charge).
  Corollary: $0 dead-cap rows (expiring-player cuts) occupy the same slot — resolves with
  the same change.
- [GEMINI B7b REVIEW, triaged] cap-CEILING enforcement is unbuilt everywhere (nothing
  rejects a transaction that busts the $125M cap). TxWriter is intentionally cap-blind. If
  a future rule enforces the ceiling mid-transaction, TxWriter needs a CapUsed read; if
  the Coordinator validates pre-tx, no change. Out of B7b scope — its own build.
- Poison policy currently guards WriteTx + GetFranchiseState; the M1/other read IPCs do NOT
  yet check `state.Err()` — extend the read-side poison guard when convenient (low-risk).
- WAL read-after-commit visibility across the split pools is relied upon (sound under wmu);
  a one-shot driver-pinning test is a nice-to-have (GLM could-not-verify #2).
- Money test floor (CORRECTED): string→cents parse exactness, migration 0-mismatch
  (REAL↔cents backfill), round-trip idempotency, non-negativity, and each contract-op
  formula pinned to its rulebook value — write these WITH the cents migration, not after.
  (The randomized-ordering byte-identical property test is RETIRED: no fractional
  distribution exists in this league to drift under reordering.)
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
