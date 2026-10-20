# M4 — Transaction UI: Expert Design Panel Brief

**Written:** 2026-07-12 · **Status:** design gate (no code yet) · **Panel:** GLM 5.2 · Gemini · DeepSeek (blind, parallel)

You are one of three independent experts. Answer from this brief alone; do not assume anything not stated. Your output is **leads for the maintainer to triage against source**, not final decisions. Where you are uncertain, say so — do not fill gaps with invention.

---

## 1. What the product is

**The War Room** is a Go + Wails v2 (WebKit) desktop app with a React + Tailwind + Zustand frontend and a SQLite (WAL) store. It runs a 32-team dynasty fantasy-football league for a single operator today (the commissioner), heading toward league-wide multi-user later (Phase 2/3). It is **not** a web app — it is a local desktop tool.

The transaction engine is DONE and battle-tested. Every league rule (§6 free agency, §8 waiver/dead-cap, §9 franchise tag, §10 extension, §11 restructure, §12 buyout, §13 special situations, §14 season rollover, phase machinery) is implemented behind a single atomic backend coordinator. **This panel is about the UI that sits on top of it — nothing about the rules is open.**

## 2. The one design goal that outranks all others

> **The transaction interface must be front and center, logical, organized, and intuitive. If the operator is overwhelmed, they will not use it.**

Approachability beats completeness. The common action must be obvious; rare/commissioner-only actions must be reachable but out of the way. A wall of options is a failure even if every option works.

## 3. What exists today (the anti-pattern to replace)

A single **dev panel** (`TransactionsPanel.tsx`) exposes all 14 operation kinds as a flat row of 14 buttons. Selecting one reveals raw text inputs that require the operator to **type a player's internal `mflID` by hand** (e.g. "13598"), plus raw franchise IDs and dollar amounts in millions. It was built for backend debugging ("debuggability over polish"), and it is explicitly the thing M4 replaces. The whole app still self-titles "Testing Harness."

The 14 operation kinds:

| Kind | Who initiates | Notes |
|---|---|---|
| TRADE | GM | multi-leg: player → franchise, any number of legs |
| ROSTER_STATUS | GM | ROSTER ↔ TAXI_SQUAD |
| WAIVER (cut) | GM | releases + charges §8 dead cap |
| RESTRUCTURE | GM | lowers cap salary by an owner-chosen move ($M) |
| TAG (§9) | GM | price resolved **server-side**; nothing to enter but the player |
| EXTENSION (§10) | GM | +1–3 years; price resolved **server-side** |
| BUYOUT (§12) | GM | OFFSEASON only; charge resolved **server-side** |
| SIGN (§6) | GM | sign a free agent: salary/yr ($M) + 1–4 yr term |
| ADVANCE_PHASE | Commissioner | OFFSEASON / REGULAR_SEASON / PLAYOFFS |
| ROLLOVER_SEASON (§14) | Commissioner | PLAYOFFS only; closes the season |
| RETIREMENT (§13) | Commissioner | 30% dead cap |
| DEATH (§13) | Commissioner | zero dead cap |
| CAP_RELIEF (§13) | Commissioner | commissioner credit ($M) to a franchise |
| SET_SIGNING_WINDOW (§6) | Commissioner | open/close the free-agency window |

Note the split: ~8 ops a team owner/GM would initiate, ~6 that only the commissioner performs (phase/calendar/discretionary). Today they are undifferentiated.

## 4. The backend surface M4 must build on (hard constraint: route through these only)

All writes go through ONE method; reads are three getters. **No new business logic may live in the frontend** — React sends *intent only*, Go computes/validates/writes.

**Write:**
- `ExecuteTransaction(req) → {ok, kind, playersAffected, at, detail}` — one atomic transaction; a failure changes nothing. `req` carries `kind` + the fields that kind needs (player mflID, franchise id, move/salary/amount in $M strings, added years, target phase, window open bool, note).

**Reads (this is where the pain is):**
- `GetFranchiseState(franchiseID) → {capUsed, players:[{mflID, rosterStatus, salary, capSalary}]}`
- `GetFreeAgents() → {mflIDs:[…]}`
- `GetCurrentPhase() → {phase}`

**The foundational gap:** `GetFranchiseState` and `GetFreeAgents` return **bare `mflID`s — no player names, no positions, no franchise names.** Human-readable names exist in the app in exactly one place: the rankings read path (`GetRankings() → rows:[{mflID, name, position, franchiseID, salary, …}]`, all 32 teams' rostered players). A "logical, intuitive" UI cannot ask a human to recognize "13598." So M4 almost certainly requires **new/enriched read IPC** (e.g. names+positions on franchise state and free agents, and a franchise-id→name directory) — proposing that surface is part of your task. Adding a read method is cheap and allowed; it is not new *business logic*.

## 5. Two doctrine invariants (non-negotiable, apply to every UI decision)

1. **Numbers never come from the client.** Every dollar figure the engine enforces (tag price, extension price, buyout charge, dead-cap hit, resulting cap) is computed **server-side, in-transaction**. The client submits an intent (a player, a year count) and gets back a receipt. A consequence to reason about: for TAG/EXTENSION/BUYOUT the operator commits **without having pre-seen the price** unless we add a preview path.
2. **Every write is staged and confirmed.** No mutation fires on a stray click. The operator sees what they are about to do, then confirms.

## 6. Environment / style facts (so you don't propose the impossible)

- Desktop WebView (WebKitGTK on an AMD APU). Single-window. Keep the DOM light — a past bug: an empty list rendered a `null` and unmounted the whole React root. Defensive, simple rendering matters.
- Existing app chrome is dark slate Tailwind, tabbed shell (Rankings / Sandbox / Validation / Transactions-dev) with a persistent right sidebar (Admin). You may keep, extend, or rethink the shell.
- Christopher's layout defaults elsewhere: 0px border-radius (2px on interactive elements), no glass/blur. Snappy, not slow. (These are defaults, not laws — justify a divergence.)
- Single operator today; design for the commissioner-doing-everything case, but don't paint us out of a later GM-per-team split.

## 7. Questions to answer (this is your deliverable)

Answer each explicitly. Rank your own confidence per answer.

- **Q1 — Information architecture.** How do you tame 14 operations into something a non-overwhelmed operator navigates intuitively? (e.g. player-centric: pick a player → see only the ops legal for that player, vs op-centric menus, vs a commissioner/GM split, vs phase-driven surfacing.) Give the concrete top-level shape.
- **Q2 — The entry point.** What is "front and center" on open? What is the single most common action, and how few clicks to it?
- **Q3 — Name/identity resolution.** What read IPC should exist so the UI is name-driven (player search, roster with names/positions, free-agent pool with names, franchise names)? Specify the getter shapes you'd add. How does the operator find a player — global search, or always via a franchise roster?
- **Q4 — Staged-and-confirmed + server-priced ops.** Design the confirm flow given that TAG/EXTENSION/BUYOUT prices are only known after the engine computes them. Do we need a two-phase "quote → confirm → commit" IPC (a dry-run that returns the price without writing), or is a post-commit receipt with undo/uncommitted-preview enough? Recommend one, with the trade-off.
- **Q5 — Commissioner vs GM ops.** Where do the 6 commissioner-only ops (phase, rollover, cap relief, retirement, death, signing window) live so they don't clutter the common path but stay reachable?
- **Q6 — Session scope.** Given all the above, what is the smallest coherent first slice to build in ONE session that is genuinely useful and not a throwaway? (Name it as the foundation to build the rest on.)
- **Q7 — Risks.** Name the top 2 ways this UI design could go wrong (UX or technical), and how to avoid them.

Keep answers concrete and buildable. Prefer one clear recommendation over a menu of options.
