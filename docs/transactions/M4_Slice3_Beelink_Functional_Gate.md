# M4 Slice-3 — Beelink Live Functional Gate

**Project:** TheWarRoom · **Stack:** Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
**Branch under test:** `session/m4-slice3-trade` · **HEAD:** `ed3b2de` (pushed to origin)
**Written:** 2026-07-18 · **Status:** DEFERRED — this is the ONE remaining gate that blocks the squash-merge to main.

---

## Why this doc exists

Everything runnable on CT105/RC is done and green (lint 0 · race green · tsc+vite clean · GLM 5.2 blind review applied on all 4 builds). What is NOT done — and cannot be done off the Beelink — is *using the feature in the real GUI*. Per the Functional-Verification doctrine, a passing build ≠ a working feature; a feature is not done until Christopher has used it in the actual environment. This doc is the precise script for that session so nothing is verified by guessing.

Four builds landed on this one branch. The gate covers **all four in a single GUI session**.

---

## Environment & hygiene (do this first, every time)

**Beelink Wails clone:** `/home/chris/opencode/TheWarRoom`

**Reset the clone to clean main BEFORE and AFTER the gate** ([[reference_beelink_functional_gate]] — force checkout + `reset --hard` + `clean -fdx` + prune session branches). Record which commit the gate actually ran on. Then check out the branch under test:

```
git fetch origin
git checkout session/m4-slice3-trade
git rev-parse HEAD          # must print ed3b2de…
```

**PATH / build env (go, golangci-lint, wails are NOT on the default PATH):**
```
export PATH=/usr/local/go/bin:/root/go/bin:$PATH
```

**Launch the GUI:**
```
GOMEMLIMIT=3000MiB GOGC=40 wails dev -tags webkit2_41
```

**Load a database.** Use a **fresh** import for the franchise-name check (a stale db predating the franchise-directory capture will show id fallback — that is correct behavior, not a failure; see Build 2). If you want to exercise names, do a Reload+Promote or import fresh.

---

## What "passing" means overall

For every op below: **run it in the GUI → Preview (D5) opens the ConfirmModal → the quote is correct → Confirm (D4) → the post-commit refresh reflects the change in BOTH the roster(s) and the cap(s).** A rejected op must surface the **engine's own reason** in the modal, not a false success and not a generic error. Nothing commits from the quoted price — commit re-sends the intent (D4).

---

## GATE 1 — TRADE builder (multi-franchise / multi-player)

**Tab:** "Trade" (between "Transactions" and "League Controls").

### 1a. Two-team swap
1. In the rail, browse **Franchise A**'s roster → "Add →" one player to the cart.
2. Browse **Franchise B**'s roster → "Add →" one player.
3. Each cart leg gets a **destination `<select>`** — set A's player → B, B's player → A.
4. "Review trade…" → ConfirmModal opens.
5. Confirm.

**Looking for:**
- Both players appear on their **new** rosters after the post-commit refresh; gone from the old ones. One **atomic** N-leg swap (not two separate moves).
- **BOTH franchises' caps move** on the refresh — the incoming/outgoing salary cells transfer.
- The "from" label on each leg is the player's **actual** origin franchise (this was GLM M1 — a mid-fetch race that stamped the wrong origin; verify the label never lies).

### 1b. Three-team swap
Repeat with 3 legs across 3 franchises (A→B, B→C, C→A). Same pass criteria — all three rosters and caps move atomically.

### 1c. Guard rails (must be BLOCKED)
- A leg with **no destination** selected → "Review trade…" is disabled (stageable = ≥1 leg AND every leg has a destination).
- A **same-franchise** destination (player sent to his own team) → excluded from the dropdown; if forced, the engine rejects and the modal shows the reason.
- **Double-add** the same player → shows "Staged", can't be re-added; even a same-tick double-click can't double-add (GLM n2 — dedup inside the functional updater).
- If a commit fails, the **cart survives** (atomic-cart-after-fail) and the modal surfaces `ok:false` as a real failure, not a false success (L1).

---

## GATE 2 — Franchise names (real team names, not ids)

**Precondition:** a **fresh** db (or a Reload+Promote) so the rulebook payload carries the franchise directory.

**Looking for real names in all three places:**
1. The **rail** franchise list.
2. The **trade destination dropdowns**.
3. The **browsed-roster header** (RosterPicker `label`).

**Stale-db check (EXPECTED, not a bug):** on a db whose config version was stored *before* the franchise block was captured, names stay **empty → id fallback**. This is the immutable-versioned-snapshot model working correctly. Confirm the id fallback renders cleanly (no blank/undefined). A Reload+Promote re-fetches and names appear.

---

## GATE 3 — D6 Commissioner surfaces (League Controls tab)

**Tab:** "League Controls." Two groups: a season **CALENDAR** group and, under a **RED divider**, the destructive **COMMISSIONER** group.

### 3a. Season calendar group
- **ADVANCE_PHASE** → confirm → after refresh, the **workspace's legal ops change** (the set of allowed transactions shifts with the new phase). This is the observable proof the phase advanced.
- **ROLLOVER_SEASON (§14)** → run it when **in PLAYOFFS** → succeeds. Run it when **NOT** in PLAYOFFS → the modal shows the **engine's rejection reason** (do NOT accept a generic error — it must be the real reason).
- **SET_SIGNING_WINDOW (§6)** → confirm → the signing window updates (latest-row-wins meta).

### 3b. Commissioner group (red divider)
- **RETIREMENT (§13)** → confirm → the player leaves the roster; dead-cap/relief as the engine computes.
- **DEATH (§13)** → confirm → same, per §13. (Death with $0 dead cap shows **no** cap line — see Gate 4.)
- **CAP_RELIEF (§13)** → uses the **franchise-name picker** from Build 2 → confirm → **that franchise's cap drops** on the refresh (a cap-relief credit is a negative delta = lower cap used).

**Cross-check:** after a retirement/death commits in-tab, the cap-relief picker's counts refresh (GLM L2 — no stale post-retirement state shown in-tab; frontend refreshes the directory after a successful commit).

---

## GATE 4 — Pre-commit dollar breakdown (the newest, heaviest build)

This is the reason the whole quote path matters: the sealed `apply()` now **RETURNS** its cap deltas (an `applyResult{Deltas []CapDelta}`), so the deltas survive the `errDryRun` rollback and show in the ConfirmModal **before** commit — dodging the in-tx read wall (a `TxWriter` read reflects committed state, not the tx's own uncommitted writes).

**Scope (deadcap-first):** only the ops that already compute their charge wire a real pre-commit delta — **WAIVE / BUYOUT / RETIRE / DEATH / CAP_RELIEF**. Other ops return **empty** deltas (no "Cap impact" section, which is correct — not a bug).

**In the ConfirmModal, look for a "Cap impact (pre-commit)" section:**
- A **charge** (raises cap used, `cents >= 0`) renders **red** (`#f87171`).
- A **credit** (lowers cap used, `cents < 0`, e.g. cap relief) renders **green** (`#34d399`).
- The **franchise name** resolves (from `FranchiseNames()`), id fallback otherwise.
- The dollar amount is **signed and $10k-snapped**. Critical: the quoted number must **equal what actually lands in the ledger** after commit. Cap relief in particular snaps to the $10k grid inside `Relieve` — the quote uses the **snapped** `entry.Amount`, so an off-grid request (e.g. $3.006M) must quote **$3.01M** and the post-commit ledger must show the same $3.01M. No drift between quote and ledger.
- The **reason** shown matches the ledger's stored reason (freeform commissioner reason for cap relief, not a hardcoded string — GLM L2 fix).

**Explicit no-line cases (EXPECTED):**
- **Death with $0 dead cap** → no cap line (`deadCapDeltas` returns nil when `DeadCap <= 0`).
- A phase advance / signing-window / trade-only op → **no** "Cap impact" section (empty deltas by design in the deadcap-first scope).

**The money-safety invariant to eyeball:** quote a WAIVE/BUYOUT/RETIRE/DEATH/CAP_RELIEF → note the number in the modal → confirm → open the cap/ledger view → the committed dead-cap/relief entry is the **same signed, snapped dollar amount**. Quote == ledger, every time.

---

## After the gate passes

1. Record the commit the gate ran on (should be `ed3b2de`).
2. **Squash-merge `session/m4-slice3-trade` → main.**
3. Run the session-close merge steps (update project CLAUDE.md build state + handoff, append to context.md, commit backbone).
4. **Reset the Beelink clone back to clean main** (post-gate hygiene).

If ANY sub-gate fails: do **not** merge. Capture the exact behavior (screenshot the modal / roster / cap view), note which op and which criterion, and bring it back — the fix is code, not a re-quote.

---

## Related

- [[project_thewarroom]] — read the CLAUDE.md header + handoff 38 first for full state and all gotchas.
- [[reference_beelink_functional_gate]] — the reset-before-and-after hygiene.
- [[project_thewarroom_league_calendar]] — SEPARATE branch, NOT part of this gate (backend WIP only, no UI, its own future gate).
- `docs/build-handoffs/handoffs/38-M4-Slice3-Trade-Builder.md` — the build handoff this gate closes.
- `docs/transactions/M4_Slice3_Breakdown_GLM_CodeReview.md` — the GLM review of the dollar-breakdown build.
