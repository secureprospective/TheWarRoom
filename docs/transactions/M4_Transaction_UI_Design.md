# M4 — Transaction UI: Design (LOCKED)

**Written:** 2026-07-12 · **Status:** design gate PASSED — approved to build slice 1 · **Supersedes:** the dev-panel era of `TransactionsPanel.tsx`
**Provenance:** 4-expert blind panel (GLM 5.2 · Gemini · DeepSeek · hy3), triaged vs source. Brief: `M4_Transaction_UI_Panel_Brief.md`. Raw responses: `M4_Panel_Responses.md`.

---

## North star (outranks all other goals)

The transaction interface must be **front and center, logical, organized, intuitive, clean, and attractive.** If the operator is overwhelmed, they will not use it. A wall of options is a failure even if every option works. The commissioner assumes a lot of work — **extra work is acceptable when the UI is clean and easy to understand.** The first slice is where the polish bar is SET, not where effort is saved.

---

## Locked decisions

- **D1 — IA is subject-centric, not op-centric.** Pick a franchise → see its roster with real names → click a player → a contextual panel shows ONLY the ops legal for THAT player in the CURRENT phase. The 14-flat-button wall becomes impossible by construction (~4-6 contextual ops per player). *(Panel unanimous.)*
- **D2 — Names come from the server; no hand-typed mflID ever appears in the new UI.** New/enriched read IPC resolves player names, positions, and franchise names via server-side joins over data that already exists (triage-confirmed: player names via the players directory as used in rankings; franchise names via the league fetcher `internal/ingestion/league`). *(Panel unanimous.)*
- **D3 — Two-phase quote → confirm → commit for priced ops.** No mutation fires blind. The operator sees the server-computed number before confirming. *(Panel unanimous.)*
- **D4 — The commit resends INTENT (player + parameters), never the quoted price.** The quote is advisory; the engine RECOMPUTES authoritatively on commit. Echoing the quoted number back as input would make the price "come from the client" — a silent violation of the numbers-never-from-client invariant. If a committed receipt ≠ the shown quote, flag it loudly. *(GLM's catch; triage-confirmed there is NO undo in the engine — `state/types.go:103`, `request_season.go:51` — so preview-before-commit is mandatory, not optional.)*
- **D5 — Preview mechanism = `dryRun` on the existing single write path, via sentinel rollback.** `WriteTx` already rolls back on any error the closure returns (`coordinator.go:60-71`). A dry run runs the REAL handler, captures the computed figures into a closure variable, then returns a sentinel `errDryRun` so the write rolls back and the captured quote is surfaced. This reuses the EXACT commit compute — one write method, one compute path, zero duplication/drift. (A separate `QuoteTransaction` method was considered and rejected: it risks the compute drifting from the commit path.)
- **D6 — Commissioner ops live off the common path.** An op's home is its subject; its prominence is its frequency. Pure calendar ops (ADVANCE_PHASE, ROLLOVER_SEASON, SET_SIGNING_WINDOW — no player subject) → a dedicated **League Controls** tab. Subject-bearing commissioner ops (RETIREMENT, DEATH → player; CAP_RELIEF → franchise) → co-located with their subject but visually segregated under a red "Commissioner" divider. All trivially hideable per-role at the later GM-per-team split. *(Panel unanimous.)*
- **D7 — First slice scope = ROSTER_STATUS + WAIVER + SIGN** (DeepSeek/Gemini), carrying D3/D4 (WAIVER exercises the server-priced dead-cap preview; SIGN is client-priced; ROSTER_STATUS is the costless on-ramp that skips preview). *(Christopher's call, 2026-07-12.)*
- **D8 — Phased shell cutover.** Add the real "Transactions" workspace tab + a "League Controls" tab; keep the old dev panel reachable behind a flag until all 14 ops are ported to the real UI, then delete it. Renaming the "Testing Harness" app title is deferred (cosmetic).
- **D9 — Defensive rendering is a hard rule, not a nicety.** Every list guards null/empty (the empty-pool → `null` → whole-root-unmount bug class). Every read path the UI touches must be name-enriched — a per-view check that no player object reaches render as a bare mflID. *(GLM Risk 1 + the known past bug.)*

---

## Information architecture (the shape)

```
┌───────────────────────────────────────────────────────────────────┐
│  The War Room       [● OFFSEASON]        [🔍 search player…]        │  header + phase badge + global search
├──────────┬────────────────────────────────────────────────────────┤
│ Franchises│  Dallas Cowboys            cap used $198.4M / …         │
│ ──────────│  ┌──────────────────────────────────────────────────┐  │
│ [filter…] │  │ Name          Pos  Status   Salary   Cap          │  │
│ ARI       │  │ CeeDee Lamb    WR  ROSTER   $6.8M    $6.8M    ▸    │  │   click a row →
│ ATL       │  │ Dak Prescott   QB  ROSTER   $40.0M   $40.0M   ▸    │  │   contextual panel
│ DAL  ◀    │  │ …                                                 │  │
│ … (32)    │  └──────────────────────────────────────────────────┘  │
│ ── ─────  │                                                         │
│ Free Agents│  ┌── CeeDee Lamb · WR · DAL ─────────────────────────┐ │   right contextual panel:
│           │  │ Salary $6.8M · Cap $6.8M · 2 yrs left              │ │   ONLY phase-legal ops
│           │  │  [ Move to Taxi ] [ Cut ] [ Trade → ]             │ │
│           │  │  [ Tag ] [ Extend ] [ Restructure ]               │ │
│           │  │  ─ Commissioner ───────────────────  (red)        │ │
│           │  │  [ Retire ] [ Death ]                             │ │
│           │  └───────────────────────────────────────────────────┘ │
├──────────┴────────────────────────────────────────────────────────┤
│  Tabs:  [ Transactions ]  [ League Controls ]  [ Rankings ] …       │
└───────────────────────────────────────────────────────────────────┘
```

- **Left rail:** the 32 franchises (filterable) + a "Free Agents" pseudo-franchise entry. Selecting one loads its roster in the center.
- **Center:** the selected franchise's roster — a flat, sortable, 0px-radius table with real names/positions/salaries/cap/status. The primary workspace. (Virtualize/paginate for safety — see risks.)
- **Right contextual panel:** slides in when a player row is clicked; player detail + only the phase-legal action buttons; commissioner actions segregated below a red divider.
- **Header:** app title + a prominent color-coded phase badge + a global player search (name-known, team-unknown power path).
- **Global search:** finds any player across all rosters + free agents → lands on that player's contextual panel. Essential for commissioner-does-everything.
- **TRADE** gets its own builder surface (full-panel/modal) — the only multi-franchise/multi-player op — deferred past slice 1.
- **League Controls tab:** the 3 pure calendar ops, clearly labeled, with their own confirm flow; deliberately not one click from daily roster work.

**Entry point:** open into the Transactions workspace with a default franchise selected (last-viewed, else first). Common action (SIGN) is ~3 clicks: Free Agents (or search) → player → enter salary/term → Confirm.

---

## Read IPC surface (new / enriched — all read-only, joins done server-side)

```
GetFranchises() -> [{ franchiseID, name }]                              // NEW: id->name directory for the rail + labels
GetFranchiseState(id) -> { franchiseID, franchiseName, capUsed,
    players:[{ mflID, name, position, rosterStatus, salary, capSalary }] }   // ENRICH in place (+name, position, franchiseName)
GetFreeAgents() -> { players:[{ mflID, name, position }] }             // ENRICH in place (+name, position)
GetPlayerDirectory() -> [{ mflID, name, position, franchiseID|null }]  // NEW: single normalized index backing global search
GetCurrentPhase() -> { phase }                                         // exists; drives legality filtering
```

- Prefer a dedicated `GetPlayerDirectory` for search over caching `GetRankings` + client-side stitching — keeps join logic out of React (doctrine) and sidesteps the stale-cache identity risk (hy3/GLM: a mid-session franchise change could make a cached index point at the wrong player). Re-fetch on every committed receipt + on phase change regardless.
- Adding read methods is not new business logic — it is explicitly allowed by the routing constraint.

## Write / preview IPC

```
ExecuteTransaction(req)  -> Receipt   // commits atomically (existing)
ExecuteTransaction(req{ dryRun:true }) -> Quote   // NEW flag: computes via the real handler, rolls back, returns the figures
```

`Quote` shape (advisory; superset the UI renders in the confirm modal):
```
{ ok, kind, playersAffected:[{ mflID, name }],
  breakdown:{ price?, deadCap?, capBefore, capAfter },   // fields present per op; ROSTER_STATUS returns none
  detail }
```
- **ROSTER_STATUS** is costless → the UI skips the quote step and shows a plain "Move X to Taxi Squad?" confirm.
- **WAIVER / SIGN** run the quote → confirm-modal-with-numbers → commit (which resends intent, per D4).

---

## Slice 1 build plan (one session, non-throwaway foundation)

1. **Backend reads:** `GetFranchises`, enriched `GetFranchiseState`, enriched `GetFreeAgents`, `GetPlayerDirectory`. Wails-regenerate bindings (remember: `wails generate module` or `createFrom` drops new DTO fields — the handoff-36 lesson).
2. **Backend preview:** `dryRun` flag on `ExecuteTransaction` + `errDryRun` sentinel + figure capture in the `WriteTx` closure; wire the breakdown for WAIVER (dead cap) and SIGN (cap impact). Plant a test that a `dryRun` leaves state byte-identical (prove the rollback).
3. **Frontend shell:** new "Transactions" workspace tab (franchise rail + roster table + slide-in contextual panel) + a "League Controls" tab stub; keep the dev panel behind a flag (D8).
4. **Frontend ops:** ROSTER_STATUS (plain confirm), WAIVER (quote→confirm→commit, shows dead cap), SIGN (salary/term form → quote→confirm→commit). Shared confirm-modal component.
5. **Phase-aware filtering:** each action button gated by `GetCurrentPhase()`; illegal ops absent (not greyed).
6. **Guards (D9):** `rows={players ?? []}`, defined empty-states, no `return null` from a list, an empty-state test matrix.

**Gate:** `make lint` 0 · `go test -race` green · tsc+vite clean · the dryRun-leaves-state-unchanged test · a live functional gate on the Beelink (sign a real FA, cut a real player with visible dead cap, toggle taxi — all name-driven, no mflID typed) · GLM 5.2 blind code review before merge.

## Deferred to later slices (each is a new form on the proven flow — incremental, not architectural)

TAG / EXTENSION / BUYOUT (server-priced, reuse the quote path) · TRADE builder (multi-leg modal) · RESTRUCTURE · the League Controls ops (advance-phase, rollover, signing-window) · CAP_RELIEF / RETIREMENT / DEATH commissioner surfaces · batch waiver processing (DeepSeek's flagged fatigue risk — only if real volume warrants) · app-title rename.

## Top risks carried (with mitigations)

1. **Name resolution incomplete in one view → silent mflID regression there** (GLM Risk 1) → per-view name-coverage check; no getter ships a bare mflID into a human surface.
2. **Blind commit / client-echoed price** (GLM Risk 2) → D4 is a hard rule; gate any priced op on the dry-run existing.
3. **WebKitGTK heavy-list / empty-list unmount on the APU** → virtualize the roster + FA lists, defensive empty-states (D9).
4. **Stale phase → confusing post-confirm rejection** (Gemini) → central phase source; buttons gated client-side with an inline "Offseason only" reason.
