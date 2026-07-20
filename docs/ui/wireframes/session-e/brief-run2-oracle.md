# SESSION E — Mobile Learning Harvest (chat-first doctrine) · RUN 2

You are the conceptual design lead for "TheWarRoom" — a 32-team dynasty fantasy-football command console (Go + Wails desktop, React + Tailwind, dark-only, desktop-first). The desktop design target is Anduril, not SaaS: a command console that costs more than a house. Dark, precise, data-dense, confident hierarchy, controls that look like they actuate real hardware.

**This session is NOT a mobile build. It is a LEARNING HARVEST.** The desktop system is COMPLETE — Sessions A (grid), B (components/type), C (color), D (event grammar + terminal-log chat + buildable calendar) are all CONFIRMED and attached below as locked context. Mobile is Phase 3, deliberately NOT a responsive shrink of the desktop — it is a DIFFERENT INSTRUMENT. Your job is to harvest what the desktop taught us and pass it forward so a future mobile design session opens with a uniform direction instead of a blank page. The REAL deliverable is a handoff document; the phone-frame wireframe is its proof.

Two rules bind you: (1) the locked context below is LOCKED — Sessions A–D are CONFIRMED and are not yours to redesign. If you believe a locked decision cannot survive the jump to a phone, that is not a redesign — that is a LESSONS-LEDGER entry (a decision mobile is free to break, with the one-line why). Silent divergence is the failure. (2) Ground every proposal in the real desktop surfaces and real data shapes below — no invented features.

Technical floor (carries to mobile as a constraint, not a target): the desktop is WebKitGTK (motion = transforms/opacity only, cheap compositing). Mobile is a phone — assume a touch surface, a thumb, no hover, no keyboard-by-default, and a screen with room for ONE thing at a time. The Session-D grammar was designed with `:hover`/`:focus-within` affordance visibility — a phone has NO hover, so the VAV altitude model is one of the first things you must interrogate.

Standing architectural fact: a future Chat engine will have COMPLETE control of this application — a terminal over a Guix-style simple command language, with a lightweight LLM router. Every mechanical control maps to a command verb (the **Command Ledger**, `docs/ui/wireframes/Command_Ledger.md`, now A1–D12). **On mobile this stops being a convenience and becomes the SHELL ITSELF** — on a phone there is no room for boards, so the command/chat channel is the primary surface and dense desktop surfaces either die, become push notifications, or become answers-on-demand through the command channel. You are NOT designing the chat ENGINE. You are deciding what mobile INHERITS from the desktop's visual grammar.

Standing design law — the spectrum commitment: one product from a 1–2-league casual to a 15–25-league portfolio manager. Snappy (<100ms acknowledge), layered altitudes (glance / operate / interrogate, depth DISCOVERED never demanded), presets-not-toggles, NO "Pro Mode" ever. **On mobile the altitude question sharpens to its hardest form:** a phone IS the glance altitude — so what is the phone for the operator with a fire in one of 25 leagues at 11pm?

## Locked context — Session A CONFIRMED spatial system
- Base unit 4px; scale 4›8›12›16›24›32›48›64. Fixed 4-column desktop instrument shell: nav 192px / fluid workspace / inspector 320px `translateX` overlay / comms right-edge 48px strip summoning OVER at 320px; calendar = generous top-right overlay. Both comms + calendar are quick-dash — summonable from ANY screen (Ledger A10/A11 `ui.summon`).
- Elevation optical only: 1px hairlines + inset bevels, NO dropshadows/blurs.
- Density tiers Narrative(48px)/Tactical(32px default)/Matrix(22px) ride CSS custom properties. No Pro-Mode toggle.

## Locked context — Session B CONFIRMED component language
- Type: Inter = text, JetBrains Mono = data. Hero numeric 24px/700 tabular-nums. Section labels 11px uppercase.
- Delta-in-weight: +Δ font-weight 600 / −Δ 400 — greyscale-honest, hue reinforces.
- Row grammar: no vertical gridlines; 4 states rest/hover(raised inset)/selected(2px left axis)/active(pressed inset).
- **ConfirmModal = 480px CENTERED + HOLD-TO-FIRE** (single verb `txn.commit`, ≤600ms fill, engine-reject holds non-dismissable). Every priced/destructive action routes through THIS gate. (On a phone: 480px-centered is a desktop measure — the hold-to-fire GESTURE is the durable part; interrogate its mobile form.)
- Empty/Loading/Error = ONE class: engraved fresh-install / data-shaped skeletons (NOT spinners) / cache-only state-line / offseason clean.
- Keyboard J/K/Enter/T///1-2-3/Esc. (A phone has no keyboard by default — every keyboard control needs a touch equivalent or it dies.)

## Locked context — Session C CONFIRMED color & atmosphere (you consume these tokens, you do not redefine them)
- Base = cold CIC navy. `--surface-canvas hsl(220,20%,6%)` → sunken 4% → tile 9% → raised 11% → overlay 13%. Optical elevation only.
- Semantic layer LOCKED, one meaning each: green=elite/positive, blue=good/info, amber=warning/watch, red=danger, gray=muted. Ramps `--signal-{green,blue,amber,red}-{muted,base,loud}`.
- **Restraint doctrine — "Color is Data and State. Structure is Achromatic."** Never tint a row/tile/canvas by value. This binds mobile HARD — a phone notification list must NOT light up like a Christmas tree.
- Score→hue banding on Adjusted Score only. Matrix yields all hue to `--text-primary` except red-danger.
- Focus = blue inset; selection = neutral 2px axis. Freshness composed: `--edge-freshness-live/-stale/-failed` + timestamp recolor + `(cache)`/`(offline)` suffix; freshness chrome NEVER washes out data.
- Motion: transforms/opacity only, ≤150ms, FEEDBACK not decoration.

## Locked context — Session D CONFIRMED event grammar (the load-bearing inheritance for mobile)
- **ONE time-ordered event substrate.** feed event / chat message / calendar-deadline / trade-card / system-alert ALL render ONE row anatomy: **2px achromatic spine** (hue EARNED only on escalation) · mono timestamp · subject weight-600 + predicate weight-400 · verb-affordance micro-switch.
- **Escalation:** spine widens 2→4px + semantic hue (amber-loud <1hr / red-loud snipe) + optical raise + pins to a top **ALERTS tributary** (`Esc` releases). NO modal, NO looping animation.
- **Recession:** 150ms fade to 50%, zero unread-badge debt.
- **VAV (Verb Affordance Visibility) = an ALTITUDE not a mode:** casual = affordances always visible; operator = hidden until `:hover`/`:focus-within`; escalation forces persistent. **← this is hover-dependent; a phone breaks it. Interrogate first.**
- Freshness rides the spine (stale → amber-muted, data legible). Cross-league = ledger-flagged seam only (`[L: ALL]/[L: ACTIVE]`, v1 single-league).
- **Chat = terminal-log thread** (no bubbles, no persona): human=Inter / command=Mono `> /verb` / system=Mono `↳`. The **`/`-pivot prompt** (typing `/` → Mono + blue axis). The **`/offer` Control Card is a control surface INSIDE the conversation** (Accept/Counter/Decline → the B ConfirmModal, card locks 50% while modal live → `[COMMITTED — HH:MM]`).
- **Calendar = buildable:** semantic-hue chips (NO position badges), Month/Week/Agenda + Mini-Month, Column-Share overlap (>3 → `+N more`), quick/drag-create (15-min snap), native drag-MOVE + stepper-only resize, **append-only honest** (`EventSuperseded`/`EventUndone`, never mutate/delete), operational overlay (deadline badges project onto the surfaces they govern). Instantaneous chat events (no duration) filtered OUT of the calendar grid (`is_scheduled && duration>0`).
- Command Ledger D1–D12: comms.send / cmd.execute / feed.ack / ui.event.resolve / txn.commit action= / calendar.create|move|resize / ui.calendar.view / ui.summon / ui.snackbar.dismiss / history.

## Facet map — the real desktop surfaces mobile must triage
Every one of these has to be assigned a mobile fate. This is the raw material of the survival table.
- **M1 Asset Rankings board** — 32 franchises × a 7-column facet map (Adjusted Score-dominant), density tiers, position filter, per-team drill-down, cap-efficiency view.
- **M2 Weekly Power Rankings** — a headline power-ranking per franchise + a weight slider (scouting ⊕ all-play blend) + sum/top-N toggle + sortable columns.
- **M4 Transaction workspace** — franchise rail → named roster → per-player phase-legal ops (waiver/sign/tag/extension/buyout/restructure) → the quote→confirm→commit gate; the multi-leg **Trade Builder**; League Controls (commissioner ops).
- **Inspector** — a single-entity deep-dive: score-dominant, 6 layer bars, terminal contract block.
- **Home** — the 2×2 slow-down break / quick-dash landing.
- **The Feed / Pulse** — the live event stream (Session D grammar).
- **The Comms thread** — the terminal-log chat + `/offer` cards + typed commands (Session D grammar).
- **The Calendar** — the buildable append-only deadline surface (Session D grammar).
- **Cross-league reality** — the 25-league operator's real world: many leagues' events at once.

## Your charge — Session E (deliver a HANDOFF, proven by ONE phone-frame wireframe)
Deliver THREE things, coherent as one document:

1. **THE SURVIVAL TABLE.** For EACH desktop surface in the facet map, assign its mobile fate — one of exactly four: **CHAT-ANSWERABLE** (dies as a persistent surface, returns as an on-demand card through the command channel — "show my expiring contracts" → a card, not a board), **LIVE CARD** (persists as a compact always-available card in the mobile shell), **PUSH NOTIFICATION** (has no resting surface at all — it only ever appears as an alert that demands a decision), or **DEAD** (does not exist on mobile; the operator opens the desktop for it). One row per surface, the fate, and one line of WHY. Be ruthless and be honest — a wrong "keep everything" table is the failure mode. Dense surfaces don't shrink; they change STATE OF MATTER.

2. **THE CHAT-FIRST MOBILE SHELL — concept + ONE flawless flow.** Show how the Session-D one-thread event grammar (card + control + answer, one thread) becomes the SHELL on a phone — the thread IS the app. Define the phone-frame anatomy: how an event row, a live card, a system answer, and an actionable Control Card read in a single thumb-scrollable column; how the `/`-pivot prompt and the command channel work WITHOUT a keyboard-first assumption; how VAV survives (or is replaced) with no hover. Then pick THE ONE FLOW that must be FLAWLESS under pressure — the moment the phone earns its place — and design it end to end: **a bid/trade decision under clock** (an escalated event → the Control Card → the hold-to-fire commit, on a phone, under a ticking deadline). This flow is where the whole system is judged.

3. **THE LESSONS LEDGER (the real deliverable — it will be handed to a future mobile session COLD).** Two columns. **BINDS mobile:** the desktop decisions mobile MUST inherit — the token system, the one-thread event grammar, the actuation/commit anatomy, the escalation rules, the restraint doctrine — each with one line of why it's non-negotiable. **MOBILE MAY BREAK:** the desktop decisions that cannot or should not survive the jump — the 4-column shell, hover-dependent VAV, 480px-centered modals, the keyboard map, density tiers — each with one line of why, and what replaces it. This ledger is the bridge; write it so a designer who has never seen this project can open a mobile session and know exactly what is sacred and what is theirs.

Every actuating element that survives to mobile gets a Command Ledger verb (reuse A1–D12; flag any genuinely new one). Color stays inside the restraint doctrine. Motion stays inside the C doctrine.

## Provocation seed for THIS run
**"THE ORACLE — THE PHONE IS A PROACTIVE QUERY TERMINAL."** The organizing truth of mobile is that on a phone there is no room for boards, so the only way to reach dense data is to ASK for it — the phone is a terminal you interrogate, and it answers with cards, not surfaces. So design mobile as a COMMAND/ANSWER instrument first: the primary surface is the Session-D `/`-pivot command channel + terminal-log thread, promoted to the whole shell. The phone's job is "ask me anything about your leagues and I return exactly the card that answers it" — "show my expiring contracts," "how do my RBs look across all leagues," "what's my cap room in the dynasty league" each return a single LIVE CARD rendered in the thread, never a board. Push this hard: most desktop surfaces become CHAT-ANSWERABLE on mobile (the survival table should be biased toward answers-on-demand, not preservation OR death); the one-thread grammar becomes a conversation where the operator drives and the system returns actuation-bearing cards; the flawless flow is a query that surfaces a decision ("your Bijan bid is losing") → the returned card IS the Control Card → resolved with one hold-to-fire, inside the same thread, without ever leaving the conversation. The casual gets a friendly ask-anything companion (no dense UI to learn); the operator gets a portfolio they can interrogate from a thumb. Then be honest in RIPCORD about where the "proactive oracle" model FAILS — the time-critical event that the operator DIDN'T think to ask about (a snipe at 2am, a bid expiring while they sleep) gets nothing from a pull-only terminal, and a command-first phone can demand more typing/intent than a thumb under a clock can afford. Name where pull-only interrogation is the wrong instrument, and what reactive/push affordance has to coexist with it.

## Output
An opinionated handoff document in markdown, **target ≤300 lines**, structured as the three deliverables above (Survival Table / Chat-First Shell + the one flow / Lessons Ledger). Commit to specifics — reuse the Session C token names and the Session D anatomy fields; name the mobile fates exactly; name the Ledger verbs. Include a compact ASCII phone-frame sketch of the shell carrying the one flawless flow (a phone is ~one column, ~360px logical — sketch at that proportion). Flag uncertainty explicitly rather than guessing. End with a **RIPCORD ITEMS** section (or "none").
