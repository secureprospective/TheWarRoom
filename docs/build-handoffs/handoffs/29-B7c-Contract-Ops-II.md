HANDOFF — Session 29: B7c — Contract Ops II (restructure / tag / extension / buyout / §13)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-04 (B7b session close)

== WHERE WE ARE ==
- B7b MERGED to main (squash `ffec0c3`, 2026-07-04) and functionally verified live on the
  Beelink: a §8 waiver cut removes the player, cap moves (his salary leaves, the §8 dead
  cap lands), the franchise stays visible via its dead cap, and it persists on a reread.
- Money is now `domain.Money int64` CENTS everywhere in Go (OQ-014). The additive
  REAL→cents migration is done (fresh DDL has the cents columns + CHECK>=0; existing DBs
  get an idempotent ALTER/backfill/0-mismatch-verify). The legacy REAL columns
  (`annual_salary`, `adjusted_salary`) are FROZEN + unread, kept one release for rollback.
- `dead_cap_ledger` is live: append-only, keyed to an ABSOLUTE `league_year`, double
  immutable (no update/delete API + BEFORE UPDATE/DELETE RAISE(ABORT) triggers). CapUsed
  is derived = live contracts + this-season ledger charges.
- The transaction seam is ready for money ops: `state.TxWriter` now has MovePlayer,
  SetRosterStatus, ApplyContract, AddDeadCap, ReleasePlayer, Player (pre-tx read), Season.
  Handlers live behind the Coordinator (`transactions-only-through-coordinator` depguard,
  PROVEN). The Waiver op is the template to clone for the new ops.

== WHAT B7c IS ==
Build_Tracker row 27 successor — the DEFERRED contract ops, each rulebook-grounded. These
are single-op transactions through `ApplyContract` (+ a dead-cap charge where a later cut
reads it). Read `docs/league-rules/Official_Rulebook.md` §9–§13 and triage every number.
Suggested order (simplest → heaviest dependency):

1. **Restructure (§11)** — the natural next op (is_restructured already flips a later cut
   from 35%→50% dead cap, wired in B7b). NOTE (locked with Christopher): the restructure
   "move" AMOUNT is the OWNER'S strategic decision, BOUNDED by the tier max (Contract-Year
   Salary ≥$3M→$1M, ≥$6M→$2M, ≥$12M→$3M) — NOT a mechanically uniform formula. §6 (flat
   salaries) and §11 both apply; do not blend them. Enforce: one restructure per team per
   year + one per contract (each extension unlocks one more). Sets is_restructured=1.
2. **Franchise Tag (§9)** — needs CROSS-STORE position aggregation: tag price = average of
   the top-5 salaries at that position league-wide (a read across all rosters). Floor rule:
   if tag price < player's prior-year salary → 120% of prior year. Max two consecutive
   years (2nd tag = 120% of first). Sets is_tagged.
3. **Extension (§10)** — position FLOOR table (§10) + "≥1 year remaining, UFAs ineligible"
   + adds ≤3 yrs, ≤6 total + extension yrs priced at 150% of highest-paid remaining year +
   one per GM per season + no second extension off a prior extension. Multi-op history.
4. **Buyout (§12)** — needs an OFFSEASON/season-phase concept (two per team, offseason
   only). Rate table by years remaining (2→60% / 3→75% / 4→90% of average remaining
   salary). Likely writes a dead-cap charge too.
5. **§13 Special Situations** — mostly commish/admin: Cap Relief Appeal (commissioner
   reduces a cap hit), Gaines Adams Rule (player death → remove, NO cap penalty — a cut
   with zero dead cap), Retirement (30% of remaining contract per year left → a dead-cap
   charge). Retirement + death are close cousins of the §8 waiver path already built.

Scope each at its gate-check — some (tag/extension/buyout) drag a real new dependency
(cross-store aggregation, a season-phase concept). Consider shipping restructure first as
its own commit since it has no new dependency and completes the B7b dead-cap story.

== READ FIRST ==
- `docs/league-rules/Official_Rulebook.md` §9–§14 (the source of truth for every number)
- `internal/transactions/deadcap/deadcap.go` (the pure-formula + handler template to clone)
- `internal/transactions/request.go` (the sealed Waiver Request to mirror)
- `internal/store/state/writes.go` (ApplyContract, ReleasePlayer, AddDeadCap, Player, Season)
- `internal/store/state/types.go` (ContractChange, DeadCapEntry, TxWriter)
- Build_Tracker row 27 (B7b) + `docs/roadmap/Roadmap_and_Open_Questions.md` OQ-014

== CARRIED FORWARD — leads, not blockers (from B7b) ==
- [GEMINI B7b, triaged] `dead_cap_ledger` UNIQUE (league,franchise,year,mfl) assumes ONE
  cut per player-per-franchise-per-year. Sound today (§8 bars re-signing a cut player; no
  FA returns a released player to a roster). WHEN FA / re-acquisition lands, a valid second
  cut becomes possible → re-key the ledger on the cut EVENT (seq/nano in the PK, drop the
  UNIQUE) or the second AddDeadCap aborts. Documented in writes.go:AddDeadCap. Do it WITH
  the FA build. $0 rows share the slot — resolves with the same change.
- [GEMINI B7b, triaged] cap-CEILING enforcement is UNBUILT league-wide (nothing rejects a
  transaction that busts the $125M cap). TxWriter is intentionally cap-blind. If a rule
  enforces the ceiling mid-tx, TxWriter needs a CapUsed read; if the Coordinator validates
  pre-tx, no change. Its own build.
- Legacy REAL money columns (`annual_salary`, `adjusted_salary`) are frozen + unread — DROP
  them one release on (a B7c-era migration once the cents path has proven out live).
- Poison policy guards WriteTx + GetFranchiseState; the M1/other read IPCs do NOT yet check
  `state.Err()` — extend the read-side poison guard when convenient (low-risk).
- A transaction_log surface (who/when/what) is deferred — restructure/tag/extension all
  want per-season op-count limits ("one per year"), which need a durable counter. Decide
  the shape (a counts table vs deriving from the ledger) at the first op that needs it.

== CONSTRAINTS ACTIVE ==
- Branch `session/b7c-contract-ops-ii` (or per-op branches). No work on main. Never --no-verify.
- CT105 gate: `export PATH=$PATH:/usr/local/go/bin`; `go build ./...`;
  `GOMEMLIMIT=1500MiB GOGC=20 make lint`; `go test -race ./...`. Beelink GUI:
  `wails dev -tags webkit2_41` (repo at /home/chris/opencode/TheWarRoom — verified 2026-07-04).
- Files < 400 lines (AD-17). Engine stays pure. Stores never import each other. New handler
  subpackages stay behind Coordinator.Execute — depguard already denies contracts/deadcap.
- Money is int64 cents everywhere in Go; never float64, never a JS number for money. Any
  percentage/average is exact integer-cents math (multiply then divide, round half-up),
  never a float intermediate.
- Review gate: GLM 5.2 BLIND (leads-not-findings, triage vs source). If GLM is down,
  hand-deliver to Gemini (proven this session). Bundle the diff + rulebook section into ONE
  self-contained prompt.

== CLOSE GATE ==
- Lint 0 / race green / depguard proven / each op's formula pinned to its rulebook value.
- Christopher EXECUTES the shipped op(s) live on the Beelink and sees cap change correctly
  (and dead cap where relevant), with a second read confirming persistence. Squash-merge
  after he confirms. Then handoff 30.
