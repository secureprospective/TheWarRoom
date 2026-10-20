# Free Agency v1 — Expert Panel Outcome + Locked Rulings

**Panel:** GLM 5.2 (SSH/Beelink) · Gemini · DeepSeek — BLIND, parallel, self-contained brief.
**Date:** 2026-07-11. Brief: `Free_Agency_Panel_Brief.md`. Leads, triaged vs source below.

The panel earned its keep: **Q6 (the `ReleasePlayer` chokepoint) was named by all three
independently and was not one of the six questions asked** — it reshapes the build.

---

## Locked rulings (design gate PASSED — build against these)

**Q1 — `player_status` = append-only STATUS-EVENT log, double-immutable.** (GLM + Gemini; DeepSeek
favored a mutable UPSERT row but conceded append-only is the house pattern.) Current status = the
latest row for a player. Same armor as `dead_cap` / `cap_relief` / `season_phases` (write-only Go
API + `BEFORE UPDATE/DELETE RAISE(ABORT)` triggers). New embedded `StatusWriter` (append event) +
a status read seam — keeps `TxWriter` under its 10-member interfacebloat cap. Rationale: matches
the codebase idiom exactly, carries the transition reason, strictly subsumes the mutable option.

**Q2 — UFA promotion folds INTO `RolloverSeason`.** (GLM; Gemini/DeepSeek proposed a sequenced
step but conceded it IS rollover's own logical work.) A private helper invoked AFTER the
snapshot-advance, inside the one sole-occupancy tx. Ordering (unanimous): promotion evaluates
"expired" against **season N+1** and reads the just-advanced N+1 roster. Rule: any rostered player
with no PAID cell for year ≥ N+1 → released into the pool (status FREE_AGENT) at **$0 dead cap**.
Preserves sole-occupancy literally (one occupant that internally bumps then promotes).

**Q3 — buyout lockout = DERIVED, zero new writes.** (Unanimous.) SIGN rejects if a
`dead_cap_ledger` row exists for the player with reason == the `buyoutReason` CONSTANT and
`league_year ≥ current season`. Both preconditions already hold in source: `buyoutReason` is a Go
constant (not free-text — no typo risk), and the ledger is append-only/immutable (never purged).
Boundary: bought out in season N → blocked through N → available in offseason N+1. Lockout is
GLOBAL (any franchise) — intended. If a retention policy ever purges dead_cap, promote to an
explicit `lockout_until` (documented carry-forward).

**Q4 — SIGN phase set = OFFSEASON + REGULAR_SEASON (block PLAYOFFS).** (Christopher's ruling;
Gemini's option.) **FORWARD SEAM (Christopher, required):** a future first-class,
COMMISSIONER-DISCRETION UFA calendar is coming — the UFA window CLOSES when the Super Bowl goes
live, STAYS closed until the commissioner reopens it, and STAYS open until it closes again (finer
than the 3-phase model — this is what the `season_phases.meta` nullable slot was built for). v1
routes SIGN's legality through a SINGLE `signingWindowOpen()` predicate (v1 impl = the phase set
above) so the commissioner-toggled window replaces the phase check later WITHOUT touching the SIGN
handler. Add the arbitrary calendar hook now; wire the real calendar later.

**Q5 — SIGN gets its OWN cell primitive + CLEARS prior cells.** (Unanimous on generalize + guard.)
SOURCE-CONFIRMED: `ReleasePlayer` deletes only `contracts` + `rosters` rows, **NOT**
`contract_years` — so a re-signed player's OLD cells linger, and the boundary-year collision is
REAL (old UFA slot at year N+1 vs a new PAID cell at N+1). SIGN MUST clear the player's prior
`contract_years` (logged to `contract_year_changes`) before laying fresh PAID(N+1..N+years) + one
trailing UFA slot. Share the low-level PAID+trailing-UFA fencepost helper with the extension
layer, but SIGN owns its primitive because it also creates the roster+contract parent rows and
requires a clean slate (extension appends to an existing tail). `readContractTail` must fail LOUD
on non-contiguous / duplicate-UFA (confirm at build).

**Q6 — `ReleasePlayer` becomes the ENFORCED status chokepoint.** (Unanimous, unprompted, ranked
most-severe.) Change the signature to `ReleasePlayer(ctx, mflID, status, reason)` so no release
path can silently forget to set player_status. Retrofit all four callers:

| Caller | Status to set |
|--------|---------------|
| waiver-cut (§8) | FREE_AGENT |
| buyout (§12) | FREE_AGENT (lockout derived from the dead_cap row, Q3) |
| retirement (§13) | RETIRED |
| death (§13, Gaines-Adams) | DECEASED |
| rollover UFA-expiry (§14, new) | FREE_AGENT |

Miss one → a player who is neither rostered nor findable (silent data loss). Second-tier
(pre-existing, verified OK): the buyout-lockout derivation depends on `dead_cap_ledger` surviving
`ReleasePlayer` — CONFIRMED, dead_cap is a separate table and is untouched by the release.

---

## Build order (blocked-by chain)
1. `player_status` schema + `StatusWriter`/status-read seam (double-immutable event log).
2. `ReleasePlayer(ctx, mflID, status, reason)` signature change + retrofit the 4 callers.
3. Fold UFA-expiry promotion into `RolloverSeason` (after snapshot-advance).
4. SIGN handler + own cell primitive (clear-then-lay) + `signingWindowOpen()` seam + derived lockout.
5. Sealed `Sign` Request + IPC + React dev control.
6. Gates: lint 0 / go test -race / tsc+vite / depguard proven / GLM blind code review / functional (Beelink).
