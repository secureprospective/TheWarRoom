# Pending Live/Functional Gate Catalog

**Why this file exists:** Christopher batched the per-session Beelink live/functional gate into one end-of-chain pass (2026-07-27) instead of stopping after every GLM Build Workflow session. This is the running list of what to actually click/test, in one sitting, on the Beelink binary — updated as each session merges.

---

## Sessions completed so far

- **Session 1 — Activity/Transaction Feed** (merged `ae584c9`): read-only chronological feed, OQ-013 reconciliation.
- **Session 2 — Transaction Correction + Roster/Position/Taxi/IR Enforcement** (backend only — merge pending as of this entry): append-only correction ledger + reconciled projection; roster/position/taxi/IR limit enforcement on Sign/Trade/RosterStatusChange. **Frontend correction UI (the "Correct this entry" action in FeedBoard.tsx) is NOT implemented** — GLM's 5-hour quota ran out mid-session before frontend work started. Only the backend/IPC half is live-gateable right now.

---

## Test 1 — Session 1: Activity/Transaction Feed

**Setup:** Load the app against your real league data (the M2/M1 boards should already populate normally).

**What to check:**
1. Open the Feed/Pulse facet (third summon-strip tab alongside Comms/Calendar).
2. Confirm it shows a chronological river of past transactions — trades, waivers, signings, releases, dead-cap/cap-relief entries — most-recent-first.
3. Spot-check a few rows against known real transactions from your league history: correct kind, correct franchise(s), correct timestamp.
4. Find at least one trade row with a picks-note/rationale attached — confirm it renders.
5. If any commissioner-created player (a manually-entered player not yet synced to an official MFL id) appears in history, confirm it renders with an "unknown player" treatment rather than crashing or blanking (OQ-013 reconciliation).

**Pass:** feed renders, rows are accurate, no console errors, unknown-player rows degrade gracefully.

---

## Test 2 — Session 2 (backend only): Roster/Position/Taxi/IR Enforcement

**No frontend affordance exists for the correction feature itself yet** — skip trying to "correct" anything from the UI; there's nothing wired to click. What you CAN test is whether the new limit gate actually rejects an over-limit transaction.

**Setup:** Know your league's real `RosterLimits`/`Starters.Positions` config and current Taxi/IR slot counts (check `LeagueControls.tsx` for the Session-0 IR/Taxi slot overrides) before starting.

**What to try (pick whichever is easiest to safely test against your real 32-team league without messing up real state — a scratch/test league clone is safer than testing live if you have one):**
1. **Roster size:** attempt a Sign or Trade that would push a franchise's total roster count past its configured roster-size cap. Expect: the transaction is REJECTED with a clear "roster limit" error, not silently accepted.
2. **Position limit:** attempt a Sign/Trade that would push one franchise's count at a single position (e.g. QB) past that position's configured max. Expect: rejected with a per-position limit error.
3. **Taxi squad:** attempt a RosterStatusChange moving a player INTO taxi when that franchise's taxi squad is already at its configured cap. Expect: rejected.
4. **IR:** same as taxi, for the IR slot cap.
5. **Sanity check the "no false rejection" side too:** confirm a normal, well-within-limits Sign/Trade/RosterStatusChange still succeeds exactly as before — the enforcement must not have broken the legal path.
6. **Override respected:** if you have the Session-0 IR/Taxi override set to `0` (mechanic disabled) on a franchise, confirm that franchise is NOT gated on that axis (0 = unlimited/disabled, not "cap of zero players").

**Pass:** every over-limit attempt is rejected with a legible error; every within-limits attempt still succeeds; the override behavior matches Session 0's "0 = off" rule.

**What is explicitly NOT testable yet:** initiating a correction from a Feed row, seeing a corrected/reversed entry render with both the original and the fix visible. That's the frontend half GLM didn't reach — it needs its own session/fast-follow before it has a live-gate test.

---

## Format for future entries

Each new session should append here in the same shape: setup, numbered steps, pass criteria, and an explicit "not testable yet" note for anything shipped backend-only.
