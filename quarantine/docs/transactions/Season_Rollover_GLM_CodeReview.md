I'll treat this as a task (no boot). Let me ground the review against actual source before issuing findings — the diff is applied to the working tree, so I'll read the load-bearing files.
Now the coordinator, schema, cap-relief read path, and test helpers — these determine whether the gates, atomicity, and test teeth actually hold.
Let me check the schema constraints, the dead-cap handler's league_year, and verify there are no out-of-lock reads of `s.season`.
Let me read the schema (rosters/contracts constraints) and the buyout handler to confirm the dead-cap league_year path.
Let me verify the phase-gate wiring and confirm whether any test exercises the real op followed by a store reboot (the end-to-end seam).
I've grounded every load-bearing claim against source. Here is the blind review.

---

# Season-Rollover §14 — Review LEADS

**Bottom line up front:** the design is well-reasoned and the happy-path implementation is largely correct — the `s.season` refresh seam (the make-or-break risk) is genuinely fixed in both `load()` and `Initialize`, all `s.season` reads sit under `wmu` (no race), and the centerpiece drift test has real teeth. I did **not** find a live CRITICAL bug in the normal path. What I found are a silent-corruption latent footgun, a missing invariant assertion the design explicitly mandates, and — most actionable — a hole in the test coverage of the exact seam the panel called highest-risk. Ranked below.

---

## L1 — MEDIUM · The make-or-break seam has NO end-to-end test (real op → reboot)
`internal/store/state/season_phase_test.go:41-81` vs `internal/transactions/rollover_integration_test.go`

The two test halves never join:
- `TestSeasonDerivedFromPhaseLogOnReboot` proves the **boot derivation** (D1/D7) — but via `simulateRollover` (`season_phase_test.go:18-34`), which is **raw SQL that jumps the season +2** and does **not** exercise `txWriter.RolloverSeason` at all.
- `TestIntegration_RolloverAdvances*` exercises the **real op** — but never closes/reopens the store.

So the specific failure class the panel flagged as biggest-risk — *a committed real rollover whose advanced year is lost on reboot because the boot path re-seeds or keeps the config season* — is proven only against a hand-rolled fake, not the actual write primitive. A regression in `RolloverSeason`'s roster `UPDATE … WHERE season=cur` (`season_phase.go:182`) that advanced the snapshot wrong would be invisible to the reboot test (the fake doesn't share that code). **Fix:** add one test that runs the real `RolloverSeason` op, then `pools.Close()` + `db.Open` + `New(configSeason)` + `Initialize`, and asserts the derived season and cap track N+1. This is the single highest-value addition.

## L2 — MEDIUM · `ROLLOVER_SEASON` sole-occupancy is convention, not enforced (silent-corruption footgun)
`internal/store/state/writes.go:33-76` (`WriteTx`), `internal/store/state/season_phase.go:163-190`

D6 mandates *"Rollover must be the SOLE occupant of its WriteTx."* Nothing enforces it. Today it's safe by accident: `Execute` (`coordinator.go:60`) runs exactly one `Request` per `WriteTx`, and `RolloverSeason` is a leaf `Request` (`request_season.go:55`). But the corruption is **silent** if a future composite op co-resides a contract op with the rollover: the contract op reads `w.s.season` (still **N** during the tx — `load()` hasn't run), writes its dead-cap `LeagueYear = w.Season() = N` (`deadcap/special.go:74`, `deadcap/buyout.go:114`), while the rollover moves the season to N+1 in the same tx. Dead cap lands in the **wrong year**, invisible until a cap audit. The phase-gate + PLAYOFFS guard do not help here. **Fix:** either a `WriteTx`-level assertion that a rollover is the only dispatched op, or document the constraint as a code-level invariant on `RolloverSeason` itself (a panic/guard if `fn` touched any non-rollover primitive).

## L3 — MEDIUM · The monotonic invariant D5 calls for ("assert `newSeason == currentSeason + 1`") is not implemented
`internal/store/state/season_phase.go:171-172`

`cur := w.s.season; next := cur + 1` — computed, never asserted against the phase log. Monotonicity *happens* to hold today **by construction** (only `RolloverSeason` writes a bumped season row; `AppendPhaseTransition` always writes `w.s.season`; `refreshSeason` reads `ORDER BY seq DESC LIMIT 1`), so this is defense-in-depth, not a live rewind. But D5 explicitly names the assert, and the moment any second season-moving primitive lands (or a manual DB edit inserts a phase row), the absence of a guard means a backward/skipped crossing is accepted silently rather than failing loud — the exact posture the rest of this codebase rejects. **Fix:** inside `RolloverSeason`, re-derive the latest phase row's season from the log and assert `cur == logLatest && next == logLatest+1` before the writes.

## L4 — LOW · `simulateRollover` is not atomic (test-fixture integrity)
`internal/store/state/season_phase_test.go:22-33`

The helper issues the phase `INSERT` + two `UPDATE`s as **three independent `pools.Write()` calls**, not one tx. A failure mid-sequence leaves the fixture DB with the phase log at N+1 but rosters still at N — precisely the split-brain state the production code is carefully built to avoid. Test-only, but it undercuts the credibility of the very drift test it feeds. Wrap the three statements in one `BeginTx`/`Commit`.

## L5 — LOW · Misleading failure-mode comment on the drift test
`internal/transactions/rollover_integration_test.go:85`

The comment claims *"A stale season would leave it at $10M."* That's not the actual stale-season failure mode here: if `s.season` stayed at N while the rollover advanced rosters to N+1, `load()`'s join (`WHERE r.season = ?` = N, `state.go:281`) finds **zero** roster rows → empty franchises map → `capOf` returns `ok=false` and fatals on *"franchise not found"*, not $10M. The test still has teeth (it fails loud either way — verified by tracing the load path), but the comment mischaracterizes the mechanism and would mislead a future debugger. Correct the comment to describe the real empty-load failure.

## L6 — LOW/INFO · Frontend cap displays go stale after a league-wide rollover
`frontend/src/components/TransactionsPanel.tsx:89`

After a rollover the handler calls `refreshPhase()` and, `if (franchise)`, refreshes one franchise's state. A rollover selects **no** franchise, so the per-franchise cap refresh is skipped — yet every franchise's `CapUsed` just rolled to next year's cells. The UI will show prior-season caps until the user manually re-fetches. Engine is correct; this is a UX staleness gap. Refresh all franchise states (or force a full reload) on `ROLLOVER_SEASON`.

---

### Things I verified as CORRECT (so they don't get re-litigated)
- **No `s.season` race / no stale read path:** every read of `s.season` (`Season()`, `OpCount`, `IncOpCount`, `AppendPhaseTransition`, `RolloverSeason`, `load`, `rosterCount`, `loadDeadCap`, `loadCapRelief`, `loadCellCap`, `seed`) is under `wmu`; `refreshSeason` writes it under `wmu`. Readers serve from the in-memory `franchises` map (under `mu`) which stores *derived CapUsed*, not the season — so no reader caches a stale year across reload. `-race` would pass.
- **Boot-order / re-seed guard:** `Initialize` derives the season from the phase log *before* `hasState` (`state.go:119-122`), so a rolled-over DB rebooted with unchanged config will not re-seed. Verified against the normal lifecycle.
- **PLAYOFFS-only gate:** enforced twice (phase policy `phase_gate.go:38-41` + store re-check `season_phase.go:168-169`); proven by `TestIntegration_RolloverRejectedOutsidePlayoffs`.
- **Atomicity of phase-append + snapshot UPDATE:** all three statements run on the one shared `w.tx` (`season_phase.go:174-188`); commit/rollback is unitary via `WriteTx`.
- **Monotonicity across phase rollback:** `AppendPhaseTransition` writes `season = w.s.season` (never a decrement), so an in-season rollback to PLAYOFFS keeps the year at N+1 — proven by `TestIntegration_RolloverIsMonotonic`.
- **Ledger correctness is downstream of D1 and holds:** dead-cap/relief are `league_year`-scoped and untouched; new season sums to 0 by absence. The buyout op-count reset is proven by `TestIntegration_RolloverResetsOpCountsDropsDeadCap`.

The implementation matches a sound design. The leads above are about **making the guarantees provable and the latent footguns loud** — L1 first.
