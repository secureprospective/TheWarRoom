HANDOFF — Session 37: M4 Transaction UI — design gate PASSED, slice-1 build IN PROGRESS
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-12 · Branch: session/m4-transaction-ui (off main)

== WHERE WE ARE ==
- M4 is "the real transaction UI beyond the dev panel" (Build_Tracker row 30, completes Phase 1 critical path).
- DESIGN GATE PASSED via a 4-expert blind panel (GLM 5.2 · Gemini · DeepSeek · hy3), triaged vs source.
  Full design + 9 locked decisions: docs/transactions/M4_Transaction_UI_Design.md (THE RULING SPEC — read it first).
  Brief: M4_Transaction_UI_Panel_Brief.md · Raw responses: M4_Panel_Responses.md.
- Visual direction APPROVED by Christopher (mockup: https://claude.ai/code/artifact/1267c650-892c-4903-b28e-a22c05ad9f63).
  "Really good start, we'll refine over time." Matches the app's dark-slate world; blue = sole interactive
  accent, amber = phase+money, red = destructive/commissioner, green = confirm.

== THE 9 LOCKED DECISIONS (summary — details in the design doc) ==
- D1 subject-centric IA: franchise → roster (real names) → click player → panel shows ONLY that player's
  phase-legal ops. 14-flat-button wall gone by construction.
- D2 names from the server; NO hand-typed mflID in the new UI. Triage-confirmed both name sources exist
  (player names via players directory as in rankings; franchise names via internal/ingestion/league Name field).
- D3 two-phase quote → confirm → commit for priced ops.
- D4 commit resends INTENT (player+params), NEVER the quoted price — else the number "comes from the client"
  (invariant-1 violation). Quote advisory; engine recomputes on commit. Triage-confirmed NO undo exists in the
  engine (state/types.go:103, request_season.go:51) → preview-before-commit is MANDATORY.
- D5 preview mechanism = dryRun flag on the EXISTING ExecuteTransaction via SENTINEL ROLLBACK: WriteTx already
  rolls back on any closure error (coordinator.go:60-71). Run the real handler, capture figures into a closure
  var, return sentinel errDryRun → write rolls back → surface the quote. ONE write method, ONE compute path.
  (Rejected: a separate QuoteTransaction — risks compute drift.)
- D6 commissioner ops off the common path: pure calendar ops (ADVANCE_PHASE/ROLLOVER_SEASON/SET_SIGNING_WINDOW)
  → a "League Controls" tab; subject-bearing (RETIREMENT/DEATH→player, CAP_RELIEF→franchise) → segregated
  under a red "Commissioner" divider in-context.
- D7 FIRST SLICE = ROSTER_STATUS + WAIVER + SIGN (Christopher's call). WAIVER exercises the server-priced
  dead-cap preview; SIGN is client-priced; ROSTER_STATUS is the costless on-ramp (skips preview).
- D8 phased shell cutover: new "Transactions" workspace tab + "League Controls" tab; keep the OLD dev panel
  (TransactionsPanel.tsx) behind a flag until all 14 ops port over, then delete. App-title rename deferred.
- D9 defensive rendering is a HARD RULE: guard every list null/empty (the empty-pool→null→root-unmount bug
  class from handoff 36); per-view check that no bare mflID reaches render.

== SLICE-1 BUILD PLAN (design doc §"Slice 1 build plan") ==
1. Backend reads: GetFranchises() [id->name dir], enrich GetFranchiseState (+name/position/franchiseName),
   enrich GetFreeAgents (+name/position), GetPlayerDirectory() [global search index]. Then `wails generate module`
   (handoff-36 lesson: createFrom drops new DTO fields silently).
2. Backend preview: dryRun flag on ExecuteTransaction + errDryRun sentinel + figure capture in the WriteTx
   closure; wire breakdown for WAIVER (dead cap) + SIGN (cap impact). PLANT a test: a dryRun leaves state
   byte-identical (proves the rollback — a gate isn't real until a planted violation fails it).
3. Frontend shell: new "Transactions" workspace tab (franchise rail + roster table + slide-in panel) +
   "League Controls" tab stub; dev panel behind a flag.
4. Frontend ops: ROSTER_STATUS (plain confirm), WAIVER (quote→confirm→commit shows dead cap),
   SIGN (salary/term form → quote→confirm→commit). Shared confirm-modal component.
5. Phase-aware filtering: buttons gated by GetCurrentPhase(); illegal ops ABSENT not greyed.
6. Guards (D9): rows={players ?? []}, defined empty-states, no return-null from a list, empty-state test matrix.

== GATE (before merge) ==
make lint 0 · go test -race green · tsc+vite clean · the dryRun-leaves-state-unchanged test ·
live functional gate on the Beelink (sign a real FA, cut a real player with visible dead cap, toggle taxi —
all name-driven, no mflID typed) · GLM 5.2 blind code review.
Build env: PATH=/usr/local/go/bin:$PATH GOMEMLIMIT=1500MiB GOGC=20 (lint needs GOMEMLIMIT=3000MiB GOGC=40).

== DEFERRED to later slices ==
TAG/EXTENSION/BUYOUT (server-priced, reuse the quote path) · TRADE builder · RESTRUCTURE · the League Controls
ops · CAP_RELIEF/RETIREMENT/DEATH surfaces · batch waiver processing · app-title rename.

== BUILD STATE AT THIS SAVE (updated — BACKEND HALF DONE) ==
Branch session/m4-transaction-ui. lint 0 / race green / build OK at each commit.
- 98185a7 design gate + docs + handoff.
- 5f1fa48 BACKEND reads (m4_app.go): GetRoster / GetFreeAgentPool / GetFranchises — name-enriched
  (player name/position via players-DB Lookup, degrade-safe ids+warning), dev panel untouched (D8).
  FRANCHISE NAMES DEFERRED: league fetcher (internal/ingestion/league/types.go) does NOT parse them
  (leagueEnvelope has no franchises block) — rail shows ids until a follow-up extends that fetcher.
- 3242bde BACKEND preview (D5): Coordinator.Preview / PreviewSign (sentinel errDryRun → WriteTx
  rolls back; runs the REAL handler, no compute drift) + PreviewTransaction IPC (m4_app.go). Planted
  test preview_integration_test.go: PreviewLeavesStateUnchanged (+ Execute mutates) / SurfacesRejection.
  KNOWN GAP flagged to Christopher: the dollar breakdown (dead-cap / cap-after) is NOT surfaced
  pre-commit. Cap is derived from the POST-commit in-memory snapshot, so a rolled-back preview can't
  read "cap after" cheaply; exposing it needs the apply() signature (14 sealed reqs) to return a
  breakdown. Preview today reports will-succeed / will-reject-with-the-real-reason + playersAffected.
  DECISION RESOLVED (Christopher, 2026-07-12): SHIP SLICE-1 AS-IS — preview shows will-succeed /
  rejection-reason before commit; the dead-cap dollar figure appears in the post-commit roster+cap
  refresh. NO apply-breakdown surgery now. Revisit pre-commit dollar figures when TAG/EXTENSION/
  BUYOUT land (they need the number more; SIGN salary is client-supplied, ROSTER_STATUS costless,
  WAIVER dead cap is the one hidden number and it shows on the immediate post-commit refresh).

== FRONTEND HALF — BUILT (commit ae85ffb, 2026-07-12) ==
DONE: `wails generate module` (verified GetRoster/GetFreeAgentPool/GetFranchises/PreviewTransaction
+ M4Player/RosterResult/FreeAgentPoolResult/FranchisesResult/M4Franchise crossed into models.ts/
App.d.ts). New components/transactions/{TransactionWorkspace,ConfirmModal,LeagueControls}.tsx:
franchise rail (GetFranchises, ids) + FA entry → roster table (GetRoster) / FA pool (GetFreeAgentPool)
→ per-player ActionPanel. ROSTER_STATUS=plain confirm; WAIVER=PreviewTransaction→ConfirmModal→Execute;
SIGN=inline form (franchise/salary/term)→preview→confirm, gated to OFFSEASON+REGULAR_SEASON via
GetCurrentPhase (signable). ConfirmModal renders plain/quote/rejected (D5 authoritative reason on
reject, no confirm). App.tsx: Transactions(default)+League Controls tabs, dev panel behind
SHOW_DEV_PANEL (D8). Every list `?? []` (D9). Matches the mockup palette. Gate: tsc+vite CLEAN,
Go untouched.
KNOWN SLICE LIMITS (by design): franchise rail shows IDs (names deferred); WAIVER dead-cap dollar
shows post-commit not in the quote (resolved decision above); League Controls is a stub (D6 ops
still on the dev panel until a later slice); global player search box from the mockup omitted
(no GetPlayerDirectory backend yet).

== GATE STATUS (2026-07-12) — BOTH PASSED ==
- LIVE FUNCTIONAL GATE: PASSED on the Beelink (Christopher, 2026-07-12). Gates 0 + A–F all green,
  every action name-driven (no mflID typed), each cut/sign moved the cap on the post-commit refresh.
- GLM 5.2 BLIND REVIEW: ran (build · glm-5.2, over SSH). 3 MAJOR flow bugs + 1 MINOR guard APPLIED
  (commit a577185): L1 confirm() ignored Execute result (false success on a rejected commit);
  L2 modal dismissable mid-commit (false cancel of a destructive op); L3 preview reopened modal
  after cancel; L6 GetFreeAgentPool missing state.Err() guard. Triaged OUT vs source: L5 (SIGN
  dispatch byte-identical value assertion), L7 (zero-value Lookup.Facts nil-map-safe), L9 (WriteTx
  errDryRun propagation proven by the planted test), L10 (Execute/Preview identical order).
  L8/L4-gate stale-phase = NOTE deferred. The fixes harden cancel/reject EDGE paths only; the
  gated happy paths are unchanged → no re-gate required (§10 drift-guard precedent).
  Gate: tsc+vite clean · go build · make lint 0 · go test -race green. Branch pushed.
- MERGE-READY. Standing auth allows merge post-gate; the fixes landed after the visual pass so
  confirm the merge call. On merge: squash to main, then session-close (project CLAUDE.md + context).

== (superseded) REMAINING GATE (before merge) ==
- Live Beelink functional gate: pick a franchise → move a player to taxi / activate → cut a player
  (see the §8 dead cap land on the post-commit cap) → select a free agent → sign to a franchise —
  ALL name-driven, no mflID typed. Verify SIGN is hidden/blocked in PLAYOFFS.
- GLM 5.2 blind review over SSH (both the m4_app.go backend from the prior commits AND the new tsx).
- Reset the Beelink clone to clean main before AND after the gate ([[reference_beelink_functional_gate]]).

== (superseded) FRONTEND HALF — original plan ==
1. `wails generate module` FIRST (handoff-36 lesson: createFrom silently drops new DTO fields;
   new methods GetRoster/GetFreeAgentPool/GetFranchises/PreviewTransaction + M4Player/RosterResult/
   FreeAgentPoolResult/FranchisesResult/M4Franchise must cross IPC).
2. New TransactionWorkspace.tsx: franchise rail (GetFranchises, ids for now) + roster table (GetRoster,
   real names) + slide-in player action panel (only phase-legal ops). Match the approved mockup
   (scratchpad m4_mockup.html / artifact 1267c650): dark slate, blue = sole accent, amber = phase+money,
   red = destructive/commissioner, green = confirm; 0px radius / 2px interactive; guard every list `?? []`.
3. Wire 3 ops: ROSTER_STATUS (plain confirm, NO preview) / WAIVER (PreviewTransaction → confirm → Execute)
   / SIGN (salary+term form → PreviewTransaction → confirm → Execute). Shared confirm modal.
4. New App.tsx tabs: "Transactions" (workspace) + "League Controls" (stub); keep the dev panel behind a
   flag (D8). 5. Phase filter via GetCurrentPhase. 6. Guards (D9) + empty-state matrix.
Gate: tsc+vite clean + the live Beelink functional gate (name-driven sign/cut/taxi, no mflID typed) + GLM review.

Session-start build was GREEN (make lint 0, go test -race all green) on main before branching.
