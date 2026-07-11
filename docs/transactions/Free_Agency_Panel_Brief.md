# Free Agency v1 — Expert Panel Brief (BLIND)

**You are one of three independent reviewers (you do NOT see the others' answers).** Answer the
design questions below from the brief ALONE. Where you are uncertain, say so — flag it, do not
fill the gap. Your output is LEADS for the maintainer to triage against source, not final rulings.

---

## Context

TheWarRoom is a **single-user** desktop app (Go + Wails + React + SQLite WAL) — a dynasty fantasy
football salary-cap engine for one commissioner/owner. It is NOT multi-user and has no real-time
concurrency: one writer, one spanning SQLite transaction per operation (the "single-writer law").

The salary system is a per-year **ledger**: `contract_years` has one row per player-year, status
PAID (counts against the cap) or UFA (the post-expiration marker, the offseason after the last
paid year). **The ledger is KING** — cap, dead cap, remaining-years are DERIVED from cells, never
stored as competing truth. Money is int64 cents, flat math, every figure snapped to a $10k grid.

Existing transaction ops (all built, all through one Coordinator): trade, waiver-cut (§8 dead
cap), franchise tag (§9), extension (§10), restructure (§11), buyout (§12), retirement/death/cap-
relief (§13), advance-phase + season-rollover (§14). A season-phase log (append-only, OFFSEASON /
REGULAR_SEASON / PLAYOFFS) gates op legality. Rollover (PLAYOFFS N → OFFSEASON N+1) advances the
roster/contract snapshot and re-derives the "current season" from the phase log.

`ReleasePlayer(mflID)` deletes the player's `rosters` + `contracts` rows — used by waiver-cut,
buyout, retirement, AND death. Its own doc comment says: *"the player re-enters via free agency,
a later build."* This IS that build.

## What we are building (v1, scope locked)

The free-agent **pool** + a **SIGN** op that RECORDS a signing outcome (NOT a live auction — the
rulebook's bid-point auction, sniping, 30-bid cap, RFA tenders, comp picks are all deferred as
multi-user Phase-2/3). Three pieces:
1. **UFA promotion:** on rollover to N+1, a rostered player with no PAID cell ≥ N+1 is released
   into the pool at **$0 dead cap** (expiry ≠ cut).
2. **The pool:** an explicit `player_status` marker (FREE_AGENT / RETIRED / DECEASED) — because
   retirement and death also call `ReleasePlayer` and must NOT appear signable. Pool = FREE_AGENT.
3. **SIGN:** assign a FA to a franchise with a new flat 1–4yr contract, lay ledger cells (source
   "signing"), enforce the §6 minimum-salary floor IF experience data is present else skip+flag,
   reject a bought-out player until the following offseason, NO cap-ceiling hard block.

Three maintainer rulings already LOCKED (do not relitigate): min-salary = enforce-if-present-else-
skip; cap = no hard block (consistent with every other op); pool = explicit status marker.

---

## DESIGN QUESTIONS — answer each with a recommendation + the tradeoff

**Q1 — `player_status` shape & mutability.** Every other durable log in this store (dead_cap,
cap_relief, season_phases, contract_year_changes) is APPEND-ONLY + double-immutable (write-only Go
API + BEFORE UPDATE/DELETE RAISE(ABORT) triggers). But a player cycles FREE_AGENT → rostered →
FREE_AGENT → … across seasons, so `player_status` is inherently MUTABLE. Options: (a) a mutable
single-row-per-player table (UPSERT the status); (b) an append-only STATUS-EVENT log where current
status = the latest row (immutable, matches house style, but heavier); (c) no table — derive
availability from existing state (off-roster AND no retirement/death audit row). Which, and why?
What is the right seam for it on the `TxWriter` interface (which is at its 10-member cap and groups
new surface into embedded interfaces like `LedgerWriter`/`SeasonScope`)?

**Q2 — Where does UFA promotion run?** On rollover, releasing every expired-contract player is a
MULTI-player mutation. The rollover primitive `RolloverSeason` is currently sole-occupancy in its
WriteTx (a strict invariant: nothing else may co-reside, because it moves the season int and a co-
resident op would read the wrong year). Options: (a) fold promotion INTO `RolloverSeason` (one
primitive, one tx — but grows a tight primitive); (b) a Coordinator-sequenced handler step in the
SAME tx after the season bump; (c) a separate op run after rollover. Which preserves the sole-
occupancy invariant and the ledger-is-king derivation best? Any ordering hazard (promotion reads
`season` — must it see N+1 or N)?

**Q3 — Buyout-lockout derivation soundness.** We plan to DERIVE the "can't sign a bought-out player
until next offseason" rule with ZERO new writes: a buyout writes a `dead_cap_ledger` row (reason
`buyout §12`, `league_year = N`); SIGN rejects if such a row exists with `league_year ≥ current
season`. Is this derivation sound across edge cases — a player bought out, lockout expires, re-
signed, bought out AGAIN (multiple rows); a player bought out then his old franchise folds; the
exact boundary (is he available IN season N+1 offseason or N+2)? Would you instead store an
explicit `lockout_until` on `player_status`? Tradeoff?

**Q4 — SIGN phase legality.** Which season phases may a SIGN happen in? The rulebook allows FA
signings in-season and in playoffs (with caveats we are NOT modeling in v1). The phase gate
(`phasePolicy` pure switch, default-deny) currently makes most ops all-phase and restricts buyout
to offseason-only. Recommend SIGN's phase set for v1 (all-phase? offseason+regular? ) and flag what
we lose by picking the simple option.

**Q5 — Cell-laying reuse & the trailing UFA slot.** SIGN must lay N..N+years-1 PAID cells + one
trailing UFA slot — the same fencepost the initial seed and `AppendExtensionYears` already
implement. Should SIGN get its OWN cell-writing primitive, or generalize the existing extension
cell-layer? What invariant must hold so the ledger stays "contiguous PAID cells + exactly one UFA
slot" (which `readContractTail` asserts)? Any risk a signing collides with stale VOID/UFA cells
from the player's PRIOR contract (he was on a roster, released, now re-signed — old cells linger)?

**Q6 — Anything we are missing.** Name the highest-risk omission or wrong assumption in this v1
design that would bite at build or functional-gate time. One concrete thing, most-severe first.

---

Answer Q1–Q6 concisely. Recommendation first, then the one-line why, then the tradeoff/risk.
Flag any question you cannot answer confidently from this brief rather than guessing.
