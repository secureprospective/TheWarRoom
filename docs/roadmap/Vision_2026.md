# TheWarRoom — Vision 2026
**Version:** 1.0 — 2026-07-06
**Status:** Approved strategic plan. Sibling to `North_Star.md` (purpose/scope) — this document is the forward vision + the locked engineering decisions that shape it. This is a plan, not a build order; the build order stays in `docs/build-handoffs/Build_Tracker.md`.

**One rule governs everything here: the MFL harness is temporary.** Design for a world where MFL doesn't exist. Cut a dependency only after running it in parity.

---

## The Thesis

TheWarRoom is not a fantasy app with a chat feature coming. It is a **deterministic contract-and-valuation engine** whose first interface is buttons, whose second will be language, and whose third will be time — the same engine, asked increasingly interesting questions.

The paper-league simplicity of the current UI is not a compromise. Thin surfaces over a thick deterministic engine is the *only* architecture in which a small local model is safe to put in front of money math. The discipline is the prerequisite for the ambition.

**Two invariants, effective now, doctrine-level:**
1. **Numbers never originate in a model.** The UI renders engine/tool outputs verbatim; any model narrates around them.
2. **Every write is staged and human-confirmed.** A model can propose an intent; only an owner's confirmation executes it.

---

## The Three Horizons

### Horizon 1 — The Capologist (the chat interface)

A first-class conversational interface, targeting a small local model (~4B class; hot-swappable — model floors expire). Not a search box. Four registers:

- **Interrogation** — every state question answered from the ledger, with receipts. The append-only `contract_year_changes` audit log means every number carries provenance to the transaction that created it.
- **Explanation** — the engine's `Result` exposes every intermediate (age pull, L4 combined, cap multiplier, tiebreak). The chat answers *"why is X ranked above Y?"* by diffing two scoring pipelines. No fantasy platform has ever had this register.
- **Staging** — "what would tagging him cost?" calls the pure pricing functions speculatively. "Do it" produces a **staged intent** that lands in the normal UI as a pending card the owner confirms. Chat proposes; the ledger disposes; the paper page confirms.
- **Initiative** — the calendar speaks first: deadline briefings, priced tag candidates, window warnings. A real front office doesn't wait to be asked.

The model is safe because it is just another button-presser: it sends sealed intent Requests to the Coordinator — the same surface the React buttons use. "All math out of the buttons" was always the doctrine that makes a small model viable.

**Entry point (Horizon 0.5):** read-only "ask the ledger" — local model + read-only tool surface + receipts rendering. No writes, no risk, cheapest possible test of the whole thesis.

### Horizon 2 — The Portfolio Desk

Owners in 32-team dynasty leagues hold multiple teams across multiple leagues. That's not five games; it's a fund with correlated positions. The product frame is **asset management**:

- **Exposure** — "you hold this player in 3 of 5 leagues; one injury tears 60% of your RB value."
- **Arbitrage** — the rulebook store holds league config as versioned snapshots and the engine takes config as a parameter, so the *existing engine* can score one player under N leagues' rules simultaneously. A player is worth five different things in five leagues; the deltas are trade routes.
- **One calendar** — every deadline in every league, one briefing.

Technically: the existing engine × N league configs + a cross-league ingestion pass (Layer-1 work — the most proven muscle in the codebase).

### Horizon 3 — The January War Room (the franchise planner)

The name has been announcing this since day one. The owner forks their franchise — an **exact copy of the ledger**, not a spreadsheet estimate — and branches futures like a developer branches code. Each branch runs the **real Coordinator** against the forked cells: same validation, same pricing, same rounding. Cap consequences are the truth, dated forward.

The finishing move: the chosen plan **exports as the season's script**. When the tag window opens, the app says "Plan A calls for tagging him — the price is now $14.2M, execute?" Drift from the plan is tracked and re-priced.

Honesty is a product feature here: **ledger math is exact; market behavior is a forecast.** The planner visually segregates the two — this cap number is truth; this trade suggestion is weather.

The "learning the league" layer is conventional statistics over transaction history (bid distributions, positional overpay tendencies, deadline behavior) computed deterministically in Go and *narrated* by the model — not fine-tuning. Small data, honest math.

---

## The Two-Product Frame

There are two products wearing one codebase, and naming them now resolves a real contradiction (per-league self-hosting vs. inherently cross-league portfolio):

| | **League Instance** | **Owner's Cockpit** |
|---|---|---|
| Scope | One league, system of record | One owner, all their leagues |
| Users | Whole league, roles/auth | Single user, own credentials |
| Needs league buy-in? | Yes | **No** — MFL's API is owner-readable |
| Horizons served | Chat (league ops), calendar, content | Portfolio, planner, chat (personal) |

The cockpit is the wedge: one GM installs it, gets value across every league they're in, zero collective action required. Sell GMs, not leagues. Every cockpit user is the advocate who later brings their league onto the instance.

---

## The Keystone Builds (bridges between today and the horizons)

1. **The shadow ledger** — a dry-run mode: execute any transaction against a forked copy of state, return the receipt, commit nothing. One capability serves three masters: chat staging, the M7 trade analyzer (already on the tracker), and the planner. Build it once, deliberately.
2. **The tool contract** — the sealed Requests + `Coordinator.Execute` are already an intent API. The chat is the third client of the existing surface, not a new architecture.
3. **The season calendar** — currently a blocker note on Buyout §12; actually the planner's clock and the chat's initiative. Build it as a first-class citizen when §12 forces it into existence.

---

## The MFL Cutover Plan (D1)

MFL provides six things: hosting/system-of-record, scoring, lineups, waivers/auction, draft, comms. Cut order — **each cut gated on running it in parity first:**

1. **Contract/cap bookkeeping — already cut.** The per-year ledger is the sole cap truth; MFL's salary fields are a downstream mirror we reconcile against.
2. **Scoring — next.** The scoring matrix is already stored as versioned config; nflverse stats are already ingested. Real L2 base scoring = stored rules × nflverse stats. This build also retires the labeled BasePoints proxy under M1 — one build closes an engine gap *and* cuts a dependency.
3. **Waivers/auction + 24h clock** — needs the free-agent-pool concept + the calendar. The first *operational* cut.
4. **Rookie draft room** — same pool + calendar machinery.
5. **Lineups** — needs multi-user; with 2+4 done, MFL is display-only.
6. **Hosting + comms** — last, and only if the succession option is taken.

**The trust-earning feature is the parity report, run as a product:** a full season of automated weekly scoring parity plus one offseason transaction window (tag → draft → UFA) in dual-record, with a discrepancy ledger. Trust is earned the day the report catches an MFL or bookkeeping error — the app audits MFL, not the other way around. No operational cut before one clean parity season.

---

## Locked Engineering Decisions (D2–D6)

**D2 · Fork primitive.** Fork at the database file layer: `VACUUM INTO` a temp/`:memory:` DB, instantiate a second Store + Coordinator over the copy — the full real stack (WriteTx, triggers, pricing, limits) runs unmodified against the fork. Rejected: `branch_id` columns (poisons the double-immutable audit design) and long-held SAVEPOINTs (the write pool is single-connection; a planning session would block every real write). **A saved plan is not a saved database — it's an ordered list of sealed intent Requests, replayed against a fresh fork.** Plans stay tiny, diffable, and re-price automatically when reality changes — drift detection for free. *(Build-time check: confirm `VACUUM INTO` on the modernc driver.)*

**D3 · Calendar entity.** One `SeasonPhase` enum per league-year (exact phase list comes from a rulebook pass — not yet pinned). No CRUD: an **append-only transitions table**; current phase = latest row. Transitions are a first-class audited op (`ADVANCE_PHASE`) through the Coordinator — commissioner-confirmed, no clock automation in v1 (the app suggests, the human confirms). **Phase eligibility is enforced declaratively in `Coordinator.Execute`** via a static op_kind → allowed-phases map; handlers keep only op-specific temporal logic. Build with Buyout §12, sized for the planner.

**D4 · Content engine.** Doctrine: the model never generates a number — template slots render struct fields verbatim, the model writes connective prose only, and a post-generation validator rejects any numeral not present in the input payload. **First shippable: the Transaction Recap** — assembled entirely from the Receipt, ledger before/after, pricing path, and op-count state, all of which already exist; **event-triggered at commit** (peak league attention), no cron. Power-ranking diff writeup second, triggered by a new output-store batch. This is also the cheapest live test of a small model doing connective-tissue-only work.

**D5 · Confidence.** No `(score, confidence)` pairs — there is no calibration data to make a scalar honest, and a fake error bar is worse than none. The right mechanism is **incremental fact revelation**: a derived `CoverageReport` beside `Result` — per player, which sub-signals were Present/Absent/Unknown and from which source tier, built from the three-state flags and exclusion reporting the boundary already computes. Deterministic, honest, chat-narratable ("ranked on 7 of 9 signals; film missing"). *Note: surfacing this in UI requires explicitly unlocking the "confidence never in UI" Hard Constraint — a docketed decision, not a route-around. Score-side certainty stays in the calibration fidelity-discount pass, per locked doctrine.*

**D6 · GM persona.** Zero new instrumentation — the ledger is already a behavioral record: risk appetite (dead-cap absorbed), time horizon (roster age × contract years), positional ideology (cap allocation), op fingerprint (transaction counts by kind), timing behavior (once the calendar exists). **v1 schema: none** — a pure feature-extractor over the existing Reader emitting a derived `GMProfile` (compute-on-read; derived truth, never stored as competing truth). The off-season engagement hook is **the Franchise Record Book** — identity accrual (longest-held player, dead cap survived, tag hit-rate, era summaries), all computable from the append-only history. Explicitly *not* a public rating of league mates in v1.

---

## Sequencing

Near-term build order is **unchanged**: §10 Extension → §12 Buyout (+ the calendar, D3) → remaining ops. Every §-op built is a future chat tool; the current grind is Horizon-1 work wearing overalls.

Horizon sequence after the transaction suite:

```
0.5  Ask-the-ledger (read-only chat)      — cheapest test of the whole thesis
 1   Shadow ledger + staging chat          — serves M7 on the existing tracker
 2   Portfolio-lite (read-only, N leagues) — Layer-1 work; seed of the cockpit
 3   The January planner                   — fork + calendar + pricing + plan-as-script
 4   Behavioral layer + content engine     — stats in Go, narration by the model
 →   Two-product fork decision             — league instance vs owner cockpit
```

## Open Decisions Docket

1. **D5 unlock** — surfacing `CoverageReport` in UI vs. the "confidence never in UI" Hard Constraint (expert-panel gate).
2. **D3 enum** — rulebook pass to pin the real phase list and window boundaries.
3. **Two-product fork** — when portfolio-lite proves out, decide the cockpit/instance split formally.

---

*Companion documents: `North_Star.md` (purpose, pillars, principles) · `docs/build-handoffs/Build_Tracker.md` (build sequence) · `docs/roadmap/Roadmap_and_Open_Questions.md` (OQs + locked decisions).*
