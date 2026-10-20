HANDOFF — Session 44: M4 Read-Model Store — route M4 reads through a store (SLIM_MAP §6.2)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

Origin: full-codebase slim review (docs/reviews/warroom-fullmap-2026-07/SLIM_MAP.md §6.2).
Christopher's ruling 2026-07-20: MOVE the read-model to a store; keep ephemeral form/cart/
modal state local. Follow-up cleanup, NOT an original Build_Tracker row.

== WHERE WE ARE ==
- Just completed: SLIM_MAP tiers 1–3 on session/slim-cleanup (pushed). The M4 stage()
  try/catch fix + the DUP4 format.tsx extraction already landed there — build on them.
- Working tree: assume clean off main after slim-cleanup merges (confirm at start).
- This session's branch: session/m4-readmodel-store (fresh off main).

== READ FIRST ==
- docs/reviews/warroom-fullmap-2026-07/SLIM_MAP.md §6.2 (this ruling), §3 DUP5 (the hook)
- frontend/src/store/harness.ts (the store-owns-IPC pattern M1–M3 follow — the model)
- frontend/src/components/transactions/{TransactionWorkspace,TradeBuilder,LeagueControls}.tsx
- frontend/src/components/transactions/{ConfirmModal,format}.tsx

== RECON (Explore fan-out — run before design/build) ==
- Ask for: every Wails IPC call made directly from the four M4 components, split into
  READ-MODEL (GetFranchises, GetRoster, GetFreeAgentPool, GetCurrentPhase, GetLegalOps)
  vs ACTION (PreviewTransaction, ExecuteTransaction) vs ephemeral local state (cart, form,
  modal/stageGen). Only the read-model moves to the store.
- Verify against source which reads are duplicated across components (the dedup win).

== GATE CHECK (confirm before writing code) ==
- Upstream complete: M4 slices 1–3 merged; slim-cleanup merged. Verified: <y/n>
- Open questions that block: none. Scope boundary is the ruling: read-model → store,
  ephemeral state stays local. Do NOT do the "full store migration" (Christopher rejected it).

== WHAT THIS SESSION BUILDS ==
- New frontend/src/store/transactions.ts (Zustand slice) owning the M4 READ-MODEL:
  franchises, phase, legalOps, rosters/FA pool — with the IPC calls moved off the components.
- Rewire the four M4 components to read from the store; keep form/cart/modal/stageGen LOCAL.
- Consider folding DUP5 (the stage→preview→confirm→cancel lifecycle + stageGen token,
  triplicated across the 3 components) into a useTransactionStaging(refreshFn) hook in the
  same pass — SLIM_MAP §5 #3 ties it here (and the try/catch already added belongs in it).
- Public surface: the store slice + optional hook. No Go/backend change.

== CONSTRAINTS ACTIVE THIS SESSION ==
- Standards: store slices never import each other (harness.ts rule); the IPC call lives in
  the store, never the component (the exact pattern this session enforces); TransactionWorkspace.tsx
  is 735 lines — the hook + store extraction should bring it toward the 400 target.
- Architectural: D2 (no mflID typed — names from server), D4 (commit re-sends INTENT, never
  the quoted price), D9 (every list `?? []`) all still hold — do not regress them.
- Anti-spaghetti: ephemeral per-op UI state does NOT belong in the shared store; only the
  read-model that multiple components/screens consume.

== CARRIED FROM LAST SESSION ==
- Decisions: the store pattern DOES apply to M4's read-model (Christopher, 2026-07-20).
- Mistakes/learnings: M4 stage() previously hung on a thrown preview (fixed in slim-cleanup
  5eae57f) — the hook must preserve that catch → rejected-state behavior.
- Open items carried: none blocking.

== CLOSE GATE FOR THIS SESSION ==
- Build green: frontend tsc + vite build clean; pre-commit hooks green.
- GLM 5.2 blind review of the diff (leads-not-findings, triage vs source).
- Functional check (Beelink live gate): every M4 surface still works end-to-end — rail →
  named roster → priced op → preview → confirm → commit (cap moves); TRADE multi-leg atomic
  swap; League Controls commissioner ops + phase-legality; a thrown preview shows the
  rejected state (not a hang). Reset Wails clone before/after.
- Handoff: write the next session's handoff before clearing.
