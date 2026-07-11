# Free Agency — Pool + Record-a-Signing (v1 Design)

**Status:** DESIGN — panel-gated, pre-code
**Written:** 2026-07-11 (Free Agency session open)
**Supersedes carry-forwards:** the two parked FA-pool items — (1) "no re-bid on a
bought-out player until next offseason" (§12), (2) trailing-UFA-slot promotion (§14 rollover).

---

## 1. Scope (LOCKED with Christopher)

**v1 = the free-agent POOL + a SIGN op that RECORDS a signing outcome.** NOT a live auction.

The rulebook's §6/§7 describe a real-time, multi-GM auction: bid points, year multipliers,
30-active-bid cap, the 20-hour "snipe" rule, RFA tenders + pick compensation, comp picks. That
assumes several owners bidding concurrently — a Phase-2/3 multi-user model this single-user
desktop tool does not have. **All of §6's auction mechanics and all of §7 (RFA) are DEFERRED.**

What v1 builds:
1. **The pool** — players who leave a roster with an expired/terminated contract become available
   free agents, distinct from retired/deceased players.
2. **UFA promotion** — on season rollover, a contract whose PAID years are exhausted releases its
   player into the pool at **$0 dead cap** (an expired contract is not a cut).
3. **SIGN** — a commissioner/GM records signing a free agent to a franchise: a new flat contract
   (1–4 yrs), ledger cells laid, minimum-salary floor applied when experience data is present.
4. **Buyout lockout** — a bought-out player cannot be signed until the following offseason.

---

## 2. Christopher's locked rulings (not the panel's to relitigate)

| # | Ruling | Decision |
|---|--------|----------|
| R1 | Minimum-salary floor (§6) keyed to years of experience | **Enforce if experience data present; else SKIP the floor and record the signing with a VISIBLE flag.** Never fabricate an experience number. Mirrors M1's exclusion-with-reasons posture. |
| R2 | Cap-ceiling check on SIGN | **No hard block** — CapUsed reflects the signing; over-cap is visible, not blocked. Consistent with buyout/tag/extension (none block on the ceiling today). League-wide ceiling enforcement stays its own future build. |
| R3 | Pool vs retired/deceased (all use `ReleasePlayer`) | **Explicit `player_status` marker** (FREE_AGENT / RETIRED / DECEASED). Pool = status FREE_AGENT. Retirement/death set their own terminal status so they never appear signable. Not derived from audit-row reason strings (fragile). |

---

## 3. The three seams (grounded in current code)

### 3a. UFA promotion — extend `RolloverSeason`
`internal/store/state/season_phase.go::RolloverSeason` today appends the PLAYOFFS→OFFSEASON row
and advances the roster/contract snapshot N→N+1, but **leaves expired contracts on rosters**. A
player with PAID cells 2026/2027 + UFA 2028 still sits rostered at 2028 with no cap-bearing cell.

**Rule:** on rollover to N+1, any rostered player with **no PAID cell for year ≥ N+1** is released
into the pool (status FREE_AGENT), **$0 dead cap** (expiry, not a cut). Stays inside the existing
sole-occupancy rollover tx (D6). Multi-player mutation in one op — still one leaf Request.

Open: does promotion belong in the store primitive (`RolloverSeason`) or a handler step the
Coordinator sequences? (Sole-occupancy says the primitive; see panel Q2.)

### 3b. The pool — `player_status`
`ReleasePlayer` deletes the `rosters` + `contracts` rows, so status **cannot** be a column on
`rosters` (the row is gone). It needs its own small table keyed by `mfl_id`:

- `player_status(league_id, mfl_id, status, since_season, ...)` — **MUTABLE**, unlike the
  append-only ledgers: a player cycles FREE_AGENT → rostered → FREE_AGENT across seasons. This is a
  deliberate divergence from the double-immutable ledger pattern (see panel Q1).
- Set FREE_AGENT by the waiver-cut (§8), buyout (§12), and rollover-expiry (§14) release paths.
- Set RETIRED / DECEASED by the §13 retirement / Gaines-Adams paths.
- Cleared (row removed or status → rostered) by SIGN and any acquisition that rosters the player.

Pool query = `WHERE status = 'FREE_AGENT'`.

### 3c. SIGN + buyout lockout
New handler (likely `internal/transactions/freeagency/`): `Sign(ctx, w, mflID, franchiseID,
salary, years)`:
1. player must be a FREE_AGENT (status check) — reject rostered/retired/deceased/unknown.
2. **buyout lockout** — DERIVED, zero new writes: a buyout already writes `dead_cap_ledger` row
   (reason `buyout §12`, `league_year = N`). SIGN rejects if such a row exists with
   `league_year ≥ current season`. Available the following offseason (season N+1).
3. min-salary floor (R1): experience known → enforce; unknown → skip + flag.
4. years ∈ [1,4], flat salary; lay PAID cells for N..N+years-1 (source `"signing"`) + a trailing
   UFA slot at N+years, mirroring the seed fencepost + `AppendExtensionYears` cell discipline.
5. create `rosters` + `contracts` rows; set `player_status` → rostered / remove FA row.
6. NO cap hard-block (R2). CapUsed reflects it.

Sealed `Sign{MFLID, FranchiseID, Salary, Years}` Request in the root `transactions` pkg; handler
behind the Coordinator depguard; IPC branch + React dev control.

---

## 4. Locked decisions that carry (reuse, don't relitigate)
- Money = int64 cents, FLAT math, universal `RoundToNearest10k` on every figure.
- CapUsed = Σ(cells) + Σ(dead_cap) − Σ(cap_relief), floored at 0. Ledger is KING.
- Single-writer law (AD-02): all mutations through one spanning `WriteTx`.
- `TxWriter` is at its 10-member interfacebloat cap — new surface groups into an embedded
  interface (`LedgerWriter` / `SeasonScope` pattern), never bare methods. A pool surface is a
  candidate new embedded interface (`PoolWriter`?) — panel Q1.
- Ledgers append-only + DOUBLE-immutable. `player_status` is the FIRST deliberately-mutable
  state-store table (rosters/contracts are also mutable but are the live snapshot) — justify it.
- Phase gate (`internal/transactions/phase_gate.go`): SIGN's phase legality — panel Q4.

See `Season_Phase_Design.md`, `Season_Rollover_Design.md`, `Salary_Ledger_Design.md`,
`docs/league-rules/Math_Rules_Reference.md`, `Official_Rulebook.md` §6/§7/§8/§12.
