# TheWarRoom — UI Wireframe Session Plan
**Version:** 1.0 — 2026-07-19T00:00-05:00
**Status:** DRAFT — awaiting Christopher's vision confirmation before Session A fires.
**Branch:** `session/ui-wireframe-plan`
**Pair documents:** `UI_Direction_Document.md` (locked architecture — authority on IA/shell/roles/comms), `docs/modules/*` (module specs), the Phase-0 research digest (path in §2).

---

## 1. Mission & Authority Contract

Produce staged wireframe rough drafts of TheWarRoom UI through a ClaudeBox + GLM collaboration loop. GLM drives conceptual design; ClaudeBox translates to wireframes; Christopher confirms vision only — no pixel-pushing.

**End state of this roadmap (Christopher's ruling, 2026-07-19): a functional, Alpha-ready product for real-world testing.** The wireframe sessions (A–E) are the design half; a build track (§9.1) follows them and lands the Alpha. Two scope rulings inside that:
- **The calendar ships fully functional by roadmap end.** It is Google-Calendar-grade in feel (entry creation, fluid drag-and-drop) but it is *not* a huge job — the append-only backend already exists on `session/league-calendar`.
- **The chat box is NOT built in this roadmap** (it is a huge job by itself — the command backbone of §1.1, future build). Its visual home (Session D) and the Command Ledger ride along so nothing forecloses it.
- **Quick-dash rule:** the calendar and the chat panel must be summonable with one quick action and collapsible off-screen **from any screen in the app**. This is a shell-level requirement, designed in Session A geometry and Session D behavior.

**Tone target:** Anduril, not SaaS. A command console that costs more than a house — dark, precise, data-dense without a pixel of waste. Controls that look like they actuate real hardware. Capability communicated before the first click.

**The authority contract (resolves "creative latitude" vs the locked direction doc):**

- `UI_Direction_Document.md` (June 2026) **locks** the information architecture: 6 nav modules, 4-column shell, density tiers (Narrative/Tactical/Matrix via CSS custom properties), one-meaning-per-color semantics, comms-as-shell-element, trade-from-chat, role model, virtualization, performance budgets. **GLM does not re-litigate these.**
- The direction doc **never specifies the visual language** — no palette values, no type scale, no grid feel, no component anatomy, no motion language. It also explicitly defers (§20) *player table column design* and *inspector panel feature specification* to "a separate UI design session." **This effort is that session.** That is GLM's creative territory, and it is vast.
- If GLM wants to challenge a locked decision, it flags it as a **RIPCORD ITEM** — surfaced to Christopher verbatim, never silently adopted, never silently dropped. Only Christopher pulls the ripcord.

**Hard technical constraints every wireframe inherits (from the locked stack):** Wails v2 WebKitGTK webview (the Beelink APU compositor has a history of lockups — motion must be cheap: transforms/opacity only, no heavy filters/blurs on large surfaces), React + Tailwind + Zustand, CSS custom properties as sole theme truth, virtualized lists, dark-only Phase 1, desktop-first.

### 1.1 The Chat Command Backbone (standing constraint — Christopher's ruling, 2026-07-19)

Go/React was chosen over HTMX **not** for UI math in buttons but for the flexibility to let a future **Chat engine have complete control over the application — as if it were a terminal on a Linux system**, using a Guix-style simple, declarative command language. A lightweight LLM on the backend will sort the user's natural language into those commands and execute them. The chat box is NOT built in this session sequence, **but the sequence must not forget it is coming.** Binding consequences for every wireframe session:

- **The UI is a human-loving projection of a command layer.** Every mechanical action a wireframe renders (sign, cut, tag, extend, trade leg, advance phase, set weight, filter board…) is conceptually a command verb with arguments that the GUI happens to actuate with a click. Nothing may be designed that could ONLY ever be a GUI gesture with no command equivalent.
- **The Command Ledger** (`docs/ui/wireframes/Command_Ledger.md`, accumulating across sessions A–E): every actuating control in every wireframe gets a row — control → its future chat-command mapping (verb + args, Guix-simple), or an explicit `RETURN-SESSION` flag when the mapping needs real design work later. The ledger is the "explicitly document the builds for a return trip" artifact; no control ships in a wireframe without a row.
- Session D designs the *visual grammar* the command channel will live in (message-as-control anatomy); the chat engine, its language, and the LLM router are separate future builds that will consume the ledger.

### 1.2 The Spectrum Commitment (standing design law — Christopher's ruling, 2026-07-19)

One product must be attractive to the **1–2-league casual** and the **15–25+ league hardcore portfolio manager** — without splitting into two products. Three commitments bind every session and every build, on par with the chat backbone:

**1. Light, snappy, speedy controls.** The direction doc's §18 performance budgets are *design law, not QA criteria* — every control acknowledges input in under ~100ms, motion is feedback rather than decoration (short, non-blocking, and per Christopher's standing design voice: fast and snappy, never slow), and anything that would make the console feel heavy is cut before it is polished. A control that feels mechanical is one that responds like a mechanism. Research grounding (spectrum digest): pressed-state feedback within 50–100ms; optimistic UI before any spinner; skeletons over spinners where loading is unavoidable (they read ~2× faster at identical latency); the Doherty threshold (400ms) is the cliff where flow breaks; animate only GPU-cheap properties (transform/opacity).

**2. Layered engagement — every surface reads at three altitudes.** *Glance* (color + headline: the casual knows the state of their league in seconds), *operate* (the working tier: tables, actions, the Tactical default), *interrogate* (Matrix: every intermediate, keyboard-driven). These map onto the locked density tiers, but the commitment is stronger than density: **depth is discovered, not demanded.** The casual is never shown a config screen to get started; the expert is never capped. The curiosity trigger (§7.5 of the direction doc) is the canonical mechanism — expansion is earned by interest, one card at a time. Research grounding: the industry's best (Blender 2.8, VS Code, Excel) ship NO explicit beginner/expert mode toggle — progression is implicit via excellent defaults, workspace templates, and features elevated by observed use. TheWarRoom follows suit: there is never a "Pro Mode" switch; the density tiers + curiosity trigger ARE the progression system.

**3. Maximum configurable states, zero configuration homework.** The configurability surface (density tiers, per-module overrides, Edit Mode, named presets, panel settings) is already locked — the law here is how it lands: **defaults must be excellent because the casual will never configure anything**, and configuration is packaged as *presets to adopt* (one click: Draft Mode, Gameday, Trade Season) rather than toggles to study. The 25-league operator builds their perfect cockpit; the 2-league casual never learns the cockpit is configurable and loses nothing.

**The portfolio dimension (design-forward, build-later):** the direction doc locks a league switcher; a 15–25-league operator also needs *cross-league scanning* — what needs me NOW across the portfolio, where a player they own in many leagues just got hurt, which deadlines land today. Deep multi-league aggregation is Phase-3 scope and is NOT built in this roadmap, but Session A's shell geometry and Session D's event/notification grammar must not foreclose it — the switcher, the unified event model, and the quick-dash panels are the seams it will grow from. Ledger discipline applies: portfolio-scale stubs get `RETURN-SESSION` rows, not silent omission. Research grounding (spectrum digest): Sleeper's proven pattern is a recency-sorted switcher overlay (most-recent leagues first, "see all" overflow) that never takes the user out of their current league's context; portfolio attention at scale uses three-tier alert triage (P0 rare push / P1 in-app bell / P2 digest bundle) — Session D's priority rules should land in that shape, and the future cross-league inbox is P0/P1 aggregated across leagues.

---

## 2. Inputs & Artifacts

| Artifact | Path | Role |
|---|---|---|
| Locked UI architecture | `docs/ui/UI_Direction_Document.md` | IA authority — excerpted into every GLM brief |
| Phase-0 research digest | `docs/ui/wireframes/phase0-research-digest.md` (copied from scratchpad once the Haiku agent lands it) | Grounding for GLM's creative freedom |
| Google-Calendar mechanics digest | `docs/ui/wireframes/gcal-mechanics-digest.md` | Entry-creation + drag-and-drop mechanics; feeds Session D and the calendar build |
| Spectrum-UX digest | `docs/ui/wireframes/spectrum-ux-digest.md` (Haiku, in flight 2026-07-19) | Layered novice→expert UX, perceived-speed doctrine, multi-league portfolio patterns; grounds §1.2 and feeds every brief |
| Calendar backend WIP | branch `session/league-calendar` (append-only store + IPC, HEAD `a11cfb5`) | The backend the functional calendar build consumes |
| Facet map | §3 of this document | The raw material GLM reshapes |
| GLM session briefs | §6 (A–E) | Exact prompts, scripted |
| Wireframe drafts | `docs/ui/wireframes/session-{a..e}/` | The working artifacts Christopher judges |
| Design direction outputs | `docs/ui/wireframes/session-{a..e}/glm-direction*.md` | GLM's raw output, preserved for the trail |

---

## 3. Facet Map (grounded in source, 2026-07-19, main @ `686a574`)

Two layers of ground truth: **what is built today** (the testing-harness shell — 8 flat tabs + a fixed sidebar, deliberately "debuggability over polish") and **what is designed** (the direction doc's 6-module shell that the built surfaces migrate into). The wireframes target the designed shell, populated with the built surfaces' real data shapes.

### 3.1 Built surfaces (frontend/src, all shipped + merged)

| Surface | File (lines) | Density | Hierarchy | What it really is |
|---|---|---|---|---|
| Transaction Workspace | `components/transactions/TransactionWorkspace.tsx` (735) | Dense, drill-down | Franchise rail → named roster → per-player action panel | Subject-centric operator console. **Ops appear only if `GetLegalOps` says the phase allows them** — engine-driven control visibility. Quote → confirm → commit via shared modal. |
| Trade Builder | `transactions/TradeBuilder.tsx` (418) | Dense, transactional | Browse any roster → cart of legs → destination per leg → atomic N-leg commit | The one multi-franchise op. Cart metaphor, preview-gated. |
| League Controls | `transactions/LeagueControls.tsx` (351) | Sparse, high-stakes | Two groups: season calendar ops; red-divider destructive commissioner powers | Commissioner console. Every op preview→confirm gated. Maps to Control Room. |
| ConfirmModal | `transactions/ConfirmModal.tsx` (187) | — | Shared commit gate | The "actuation" moment: non-dismissable mid-commit, rejected-state hold, stale-preview discard. The single most hardware-like interaction in the app. |
| M1 Asset Rankings | `RankingsBoard.tsx` (213) | Dense table | Global list + position filter / per-team drill-down / cap-efficiency view | 32-team ranked board, 3 client views over persisted rows. Maps to War Room Asset Board. |
| M2 Power Rankings | `PowerRankingsBoard.tsx` (304) | Dense table + live controls | Headline blend + sortable MFL context columns | Live weight slider (0–100% scouting↔performance), sum/top-N toggle. Maps to League Pulse. |
| Rookie Sandbox | `RookieTable.tsx` (100) | Max debug density | Flat scored table, every engine intermediate visible | Matrix-density archetype. |
| Validation Board | `ValidationBoard.tsx` (47) | Sparse status board | 12 cases, PASS/FAIL/PENDING tri-state | Systems-status archetype (amber ≠ red honesty). |
| Admin Panel | `AdminPanel.tsx` (51) | Sparse, persistent sidebar | Live calibration params → board re-scores | Maps to Control Room Admin Calibration. |
| Dev panel | `TransactionsPanel.tsx` (619) | — | Raw-mflID dev surface behind `SHOW_DEV_PANEL` | **Excluded** — dies at D8 cutover completion. |

### 3.2 Designed-but-unbuilt surfaces (direction doc + backend WIP)

| Surface | Source | Status |
|---|---|---|
| Home / League Landing | Direction doc §10–11: 2×2 card grid + seasonal card system (9 calendar-triggered cards) | Designed, unbuilt |
| Contextual Inspector | §5: populates on any entity click, no navigation. §20: feature spec deferred to this session | Designed shell only |
| Comms layer | §13: 6 channels, auto league feed, 48px collapsed strip | Designed, unbuilt |
| Trade-from-chat | §15: /offer, multi-asset ledger, card states, DOT routing | Designed, unbuilt |
| League Calendar | Branch `session/league-calendar`: append-only calendar store (`internal/store/state/calendar.go`, 210 lines) + `transactions_calendar_app.go` IPC — **backend WIP exists, zero UI** | Backend WIP |
| AI chat/command interface | Memory: React chosen specifically so owners + commissioner can issue league queries through a chat channel later | Vision only |
| Quick Access Panel / Edit Mode / presets | §8–9 | Designed, unbuilt |

### 3.3 Density & hierarchy read (what the wireframes must honor)

- **Glanceable:** Home cards, seasonal card, validation states, phase banner, cap headlines, unread badges.
- **Scan-dense:** M1/M2 boards, rosters, free-agent pool (F-pattern table territory; column design is a deliverable of this effort).
- **Drill-down:** Inspector, layer breakdowns, contract detail, draft-board expansion.
- **Actuation:** ConfirmModal path, trade cart commit, commissioner destructive group — where the "mechanical" feel concentrates.
- **Event-driven visibility (the cockpit pattern, already in the domain):** phase-legal ops appear/vanish with the season phase; seasonal cards rotate on calendar triggers; feed events escalate (snipe alerts, sub-1hr clocks). The engine already drives this — the UI's job is to *express* it.

---

## 4. Resource Map & Pipeline

```
Haiku (research, done once) ──► ClaudeBox (head brain) ──► GLM 5.2 (creative engine, cheap tokens)
                                      ▲                          │
                                      │  triage + translate      │ design directions (2 divergent + 1 synthesis per session)
                                      │                          ▼
                              Christopher ◄── wireframe rough draft (vision confirmation ONLY)
```

**Roles:**
- **Haiku** — Phase-0 research digest. Fires once, up front. Output is a standing input to every brief.
- **ClaudeBox (Claude, head brain)** — owns the loop: assembles briefs, launches GLM runs, triages GLM output (leads-not-findings — same doctrine as code review), writes the wireframes, presents to Christopher, incorporates corrections. Claude never outsources judgment; GLM output is raw creative material.
- **GLM 5.2 on z.ai** — conceptual design engine. Cheap tokens: run it hot (see §5). Every run scripted + detached + sentinel-notified; never hand-polled.
- **Christopher** — vision confirmation gates only (see §8 triggers). "That's the direction" or "here's what's off." Nothing else is asked of him.

**Christopher pull-in triggers (exhaustive — he is not pulled in otherwise):**
1. **Now:** approve this plan (facet map framing + brief thrust) before Session A fires.
2. **Per session (A–E):** one vision confirmation on the wireframe draft, against §8's criteria.
3. **Ripcord items:** any GLM challenge to a locked decision, presented verbatim with Claude's recommendation.
4. **Correction round:** if he flags "what's off," the corrected draft returns once for re-confirmation, then the next session fires.

---

## 5. GLM Operations Doctrine (cheap tokens — lean into it)

GLM tokens are cheap; Christopher's attention is not; Claude's context is the scarce middle. Therefore:

1. **Script every prompt.** Brief = one self-contained markdown file (no conversation memory assumed) written to a scratch dir; the runner passes it as an argument, never stdin. GLM answers cold, every time.
2. **Run detached, never poll by hand.** Runner: **`scripts/glm-design-session.sh <brief.md> <out.md> [model]` — BUILT + PROVEN (Session 0, 2026-07-19: 40s round trip).** It ships the brief to the Beelink, runs the z.ai **coding**-endpoint call (`/api/coding/paas/v4`) detached there via setsid (the API key lives ONLY in the Beelink's `~/.config/opencode/zai.env` and never crosses to CT105), and blocks on the `EXIT=` sentinel. Invariants learned in Session 0: SSH must use `-i /root/.ssh/beelink` (default key fails); the nested remote-quoting block is fragile — re-test with a small run after any edit. Fallback: pass `glm-4.7` on a 5.2 hang (known headless issue). Claude launches it via `Bash run_in_background` — the harness re-invokes Claude when the task completes. **Zero polling turns.**
3. **Divergence before convergence — keep GLM busy.** Each design session = **three GLM runs**: two *divergent* directions launched in parallel (same brief, different provocation seeds — e.g. "asymmetric density-driven grid" vs "instrument-cluster grid"), then Claude triages both against the facet map + constraints, then one *synthesis* run: "here are the two directions and the head brain's triage — fuse and resolve the named tensions." The synthesis output is what becomes the wireframe.
4. **Leads, not findings.** GLM's design directions are triaged against source (direction doc, real data shapes, WebKitGTK constraints) before a single wireframe line is written. What survives triage is recorded at the top of the wireframe as "adopted / adapted / rejected (why)."
5. **Generous budgets, incremental writes.** Long timeout, output flushed as it streams, so a timeout never loses a run.
6. **Wait for good work.** If a GLM run comes back thin (generic SaaS-speak, no committed opinions), don't hand-fix it — re-fire with a sharpened provocation. Tokens are cheap; Claude rewriting GLM's job is not.

---

## 6. GLM Session Briefs

Every brief file is assembled from this **common preamble** + the per-session charge:

> **Common preamble (verbatim in every brief):**
> You are the conceptual design lead for "TheWarRoom" — a 32-team dynasty fantasy-football command console (Go + Wails desktop, React + Tailwind, dark-only, desktop-first). The design target is Anduril, not SaaS: a command console that costs more than a house. Dark, precise, data-dense without a pixel of waste. Confident hierarchy. Controls that look like they actuate real hardware. The UI must communicate capability before the user clicks anything.
> You have creative latitude and you are expected to use it — push past tired dashboard patterns. But two rules bind you: (1) the attached UI_Direction_Document excerpt is LOCKED information architecture — do not redesign navigation, density tiers, color semantics, roles, or the comms model; if you believe a locked decision is wrong, write a section titled RIPCORD ITEMS stating the decision, your objection, and your alternative — do not design around it silently. (2) Ground every proposal in the attached facet map's real surfaces and real data shapes — no invented features.
> Technical floor: WebKitGTK webview (motion = transforms/opacity only, cheap compositing), CSS custom properties carry all tokens, virtualized lists, Tailwind-expressible.
> Standing architectural fact: a future Chat engine will have COMPLETE control of this application — a terminal over a Guix-style simple command language, with a lightweight LLM translating user language into commands. You are not designing that chat box, but every mechanical control you propose must be conceivable as a command verb + arguments that the GUI merely projects. If you propose an interaction with no plausible command equivalent, flag it yourself.
> Standing design law — the spectrum commitment: this one product must delight a 1–2-league casual AND a 15–25-league hardcore portfolio manager. Three rules bind everything you propose: (1) light, snappy, speedy — every control acknowledges in <100ms, motion is feedback not decoration, nothing may feel heavy; (2) layered engagement — every surface must read at three altitudes (glance / operate / interrogate) with depth DISCOVERED, never demanded: the casual is never shown configuration to get started, the expert is never capped; (3) configurability lands as one-click presets to adopt, never toggles to study — defaults must be excellent because casuals never configure. If a proposal serves one end of the spectrum by taxing the other, redesign it.
> Output: an opinionated design-direction document in markdown, with an explicit length target stated per brief (Session 0 lesson: a hard cap sharpens commitment — full sessions target ≤250 lines per run). Commit to specifics — named values, named rules, named anatomy — not mood language. Flag uncertainty explicitly rather than guessing. Attached: [facet map §3] [research-digest EXCERPTS — only the sections this session needs, never whole digest files (Session 0: lean briefs beat fat briefs)] [direction-doc excerpt] [prior session outputs, if any].

### Session A — Grid & Spatial System
**Charge to GLM:**
> Invent the spatial architecture for the 4-column shell (nav rail / fluid workspace / contextual inspector / comms panel) and everything inside it. The direction doc fixes the columns' existence, not their feel. Questions you must answer with committed rules: What is the base unit and why? Is the workspace grid symmetric or does density drive an asymmetric grid (a cockpit is not a spreadsheet)? What are the container rules — how do panels earn borders, elevation, or separation in a UI with zero wasted pixels? How does the grid behave across the three density tiers using only CSS variable swaps? Where does negative space live in a data-dense console (it must exist somewhere or nothing reads)? How does the 2×2 Home card grid relate spatially to the dense module views — same grid language or a deliberate break? Define: spacing scale, panel anatomy (header/body/chrome), workspace grid spec per density tier, collapse behavior geometry. One shell-level requirement is binding: the CALENDAR and the CHAT panel must be summonable with one quick action and collapsible fully off-screen FROM ANY SCREEN — design the geometry of that quick-dash (where they live when collapsed, what they overlay or displace when summoned, how both coexist if summoned together); the comms panel's 48px collapsed strip is the locked starting point for chat, the calendar's summon geometry is yours to invent. Spectrum note for the shell: the nav rail's league switcher must scale honestly from 1 league to 25+ (a dropdown that works at 2 embarrasses itself at 25 — but do NOT design Phase-3 cross-league dashboards; leave the geometric seam where a portfolio surface could later live, and say where it is).
**Divergent seeds:** run 1 = "instrument cluster — zones with fixed roles, asymmetric, earned real estate"; run 2 = "density gradient — one continuous grid where information pressure, not layout slots, creates structure."
**ClaudeBox output:** wireframe skeleton — the shell + one dense module (M1 board) + Home grid, greyscale boxes, real column counts.

### Session B — Component Hierarchy & Typography
**Charge to GLM:**
> Define the component language: type scale, table/row anatomy, card anatomy, button language, form controls, the ConfirmModal (the single most important component — it is the commit gate for destructive league operations; it should feel like arming and firing a system, honestly, without theater). Compressed scale, max ~24px, single sans (Inter or equivalent), monospace where numbers align — but the specifics are yours: how many steps, what weights carry hierarchy when size is compressed, what a section label looks like at 11px, how a 32-row ranked table stays scannable at Matrix density. This session also OWNS two deliverables the architecture session explicitly deferred: (1) the player-table column design — which columns at which density, alignment, sortability affordance — for the M1 board, M2 board, and roster tables (real columns are in the facet map); (2) the Contextual Inspector anatomy — engine score dominant, layer-breakdown bars, contract block, notes block. Controls should feel mechanical: define the 4 interaction states for every control class with sub-100ms acknowledgment as a hard rule (pressed state is instant; async work never blocks the pressed feedback), and what "actuation" looks like for the preview→confirm→commit path. Component anatomy must be LAYERED, not merely dense: define each component's glance / operate / interrogate reads — what a player row gives a casual in half a second (color + name + one number), what it gives the working operator (the Tactical columns), and what it opens into for the interrogator (Matrix / inspector) — and how the same anatomy expands across those altitudes without becoming a different component. Two more deliverables are YOURS, first-class: (1) EMPTY / LOADING / ERROR ANATOMY — every surface needs a designed answer for four honest states: no data yet (a fresh install — a casual's first five minutes are almost entirely empty-state experience, so emptiness must feel intentional, oriented, and inviting, never broken), fetch in flight (skeleton language, not spinners), MFL unreachable (failed refresh — what the surface says and what stays usable from cache), and seasonal emptiness (an offseason matchup view is EMPTY BY NATURE, not by failure — design the difference). Define the anatomy once as a component class, applied everywhere. (2) THE GLOBAL KEYBOARD MAP — one page: what every key does, per density tier (Matrix locks J/K/Enter/T as the floor; Tactical gets a working subset; Narrative near-zero). Every shortcut is a projection of a command verb — each keymap row cross-references its Command Ledger verb, so the keyboard, the GUI, and the future chat terminal are three faces of one command surface.
**Divergent seeds:** run 1 = "aerospace placard — engraved, static, everything labeled"; run 2 = "terminal-forward — type does all the work, chrome approaches zero."
**ClaudeBox output:** component-sheet wireframe (one page, every component class rendered with real data — including the empty/loading/error states firing on real cases) + the M1 board re-rendered with the committed column spec + the keyboard-map page.

### Session C — Color, Dark Mode & Atmosphere
**Charge to GLM:**
> Build the functional color system as CSS custom-property tokens. The semantic layer is locked (green=elite/positive, blue=good/info, amber=warning/watch, red=danger, gray=muted — one meaning per color, everywhere). Everything else is yours: the base atmosphere (dark ≠ gray — anthracite? midnight blue? near-black with a temperature?), the surface elevation ramp, the exact semantic values and their ramps (a score-88 green and a cap-efficient green need family resemblance without ambiguity), position-badge palette within the locked convention, focus/selection treatment, and the restraint rules — in a console, color is signal; define what is FORBIDDEN to be colored so signals stay loud. Motion doctrine rides with atmosphere: define the micro-animation language for async operations (preview pending, commit in flight, feed event arrival, clock under 1 hour) under the WebKitGTK budget — transforms/opacity, ≤200ms, never blocking, and every animation must earn its place as FEEDBACK (answering "did it register? is it working? did it land?"); anything that is decoration, or that makes the console feel slower than instant, is cut. Matrix density kills all of it. One more token family is yours: DATA FRESHNESS HONESTY — this console blends live MFL fetches with cached SQLite state, and it makes money decisions, so trust in the data's age IS the UX. Design the visual grammar for "as of when": a live-fetched surface vs a cached board vs a failed refresh showing stale data, each unmistakable at a glance without shouting (a timestamp treatment, an edge state, a muted badge — your call), consistent across every surface. Deliver the full token sheet: `--surface-*`, `--signal-*`, `--text-*`, `--edge-*`, `--freshness-*`, named and valued.
**Divergent seeds:** run 1 = "tactical ops — near-black, amber-warm instrumentation"; run 2 = "naval CIC — cold midnight base, phosphor-cool signals."
**ClaudeBox output:** Sessions A+B wireframes re-skinned with the token sheet; a one-page atmosphere board (same components, tokens applied, all four signal colors firing on real cases).

### Session D — Chat + Calendar Command Layer
**Charge to GLM:**
> Design the layer that makes this a command console rather than a viewer: communication and time, fused with operation. The locked model: comms is a persistent shell element (6 channels incl. auto-generated League Feed; 48px collapsed strip), trade-from-chat exists (/offer, multi-asset ledger, Accept/Counter/Decline cards with 5 states), seasonal cards rotate on the league calendar, and an append-only league-calendar backend already exists with no UI. Your charge: (1) the unified event grammar — a feed event, a chat message, a trade card, a calendar deadline, and a system alert are all events; design the single visual grammar that lets one eye scan all of them in one hierarchy, and define how an event ESCALATES (bid clock crossing 1hr, snipe alert) and RECEDES (expired, resolved) — event-driven visibility, not static layout. Attention management must serve the whole spectrum: the 2-league casual gets a calm surface where only what genuinely needs them ever escalates (no notification anxiety), while the 25-league operator gets a triaged "what needs me NOW" queue built from the same event grammar — define the priority rules that make one grammar serve both, and leave the seam (ledger-flagged) where cross-league aggregation plugs in later. Freshness is behavioral here too: Session C gives you the `--freshness-*` tokens — define WHEN a surface transitions (how stale is stale, what a failed refresh does to the surfaces that depend on it, how recovery clears the state) and how a stale-data event rides the same event grammar; (2) chat as the future terminal — the standing architectural fact is that a chat engine with an LLM language-router will eventually drive the WHOLE app through a Guix-style command language; you are designing the thread grammar that terminal will inhabit: message-anatomy where a card, a control, a typed command, and a system answer all live in one scannable thread (a /offer card is a control surface inside a conversation; "show me every expiring contract under $2M" returns an answer-card in the same grammar). Design the visual form of a command being issued, echoed, and confirmed — terminal honesty, console beauty. The chat ENGINE itself is not yours to design — its visual home is; (3) the CALENDAR — and unlike the chat engine, this one WILL be built to full function by roadmap end, so your design must be buildable, not aspirational. Target feel: Google Calendar operated from a command console. The attached Google-Calendar mechanics digest gives you the interaction floor: click-empty-slot quick-create, drag-to-create spanning a duration, drag-to-move and edge-drag-to-resize with live ghost + snap increments, event-chip anatomy, overlap layout, view architecture (week/month/agenda + mini-month navigator). Design its full anatomy in the console language, PLUS its two operational lives: the quick-dash summon (one action from any screen, collapse fully off-screen — same shell rule as chat) and the operational overlay — deadlines and windows appearing ON the surfaces they govern (a signing window on the SIGN control, a trade deadline on the trade builder), not only in the calendar view. One honest constraint: the backend is an APPEND-ONLY event log — a dragged "move" commits as an appended superseding revision, undo as another append; design the optimistic drag UX so it maps to that truthfully (the digest's final section covers the pattern).
**Divergent seeds:** run 1 = "the feed is the spine — everything is a timeline"; run 2 = "the console owns the surface — comms/time annotate the controls they govern."
**ClaudeBox output:** interaction-model wireframe: comms panel expanded + collapsed states, one trade-from-chat negotiation sequence (all 5 card states), the FULL calendar anatomy (week view with real league events, quick-create popover, a mid-drag ghost state, the quick-dash summoned + collapsed states), the calendar overlay on two operational surfaces, one AI-command exchange mock.

### Session E — Mobile Learning Harvest (chat-first doctrine)
**Purpose (Christopher's ruling, 2026-07-19):** E is not a standalone mobile design — it is the bridge that harvests what the desktop system taught us and passes it forward, so the future mobile sessions open with a uniform direction instead of a blank page. **Hard dependency: E fires only after Session D is confirmed** — mobile leans into the chat interface because of the lack of screen real estate, so the chat-with-mechanical-controls layer must exist in working form on desktop before mobile has a real chance.
**Charge to GLM:**
> The desktop system is complete (Sessions A–D attached). Mobile is Phase 3 and deliberately NOT responsive — it is a different instrument, and its primary surface is the CHAT/COMMAND CHANNEL: on a phone there is no room for boards, so the Session-D event grammar and actuating cards become the shell itself. Dense surfaces don't shrink — they either die, become push notifications, or become answers-on-demand through the command channel ("show my expiring contracts" returns a card, not a board). Deliver: (1) the survival table — each desktop surface → its mobile form (chat-answerable / live card / push notification / dead); (2) the chat-first mobile shell concept — how the Session-D message anatomy (card, control, answer in one thread grammar) carries actuation on a phone, including the one flow that must be flawless under pressure (recommend and justify — likely bid response under clock or trade accept/counter); (3) the LESSONS LEDGER — the desktop decisions that BIND mobile (token system, event grammar, actuation anatomy, escalation rules) vs the ones mobile is free to break, each with one line of why. The ledger is the real deliverable: it will be handed to a future mobile design session cold.
**Seeds:** single run + synthesis (two divergent runs optional if run 1 comes back thin).
**ClaudeBox output:** the mobile handoff document (`docs/ui/wireframes/session-e/Mobile_Handoff.md`: survival table + lessons ledger) + a phone-frame concept wireframe of the chat-first shell carrying one actuation flow.

---

## 7. Wireframe Output Template (what a "rough draft" physically is)

One **self-contained HTML file** per session (inline CSS, zero external deps — same discipline as an Artifact page), committed to `docs/ui/wireframes/session-X/`, presented to Christopher rendered (side panel / browser), not as code.

Standing anatomy of every draft:
1. **Header strip:** session letter + name, date, GLM runs consumed, direction doc version.
2. **Triage block:** GLM proposals — adopted / adapted / rejected, one line each. (The trail that makes correction rounds cheap.)
3. **The wireframe proper:** real surfaces from the facet map, real data shapes (32 franchises, actual column names, plausible values — no lorem ipsum, no "Item 1"), rendered at desktop width (~1440px logical; Session E uses a phone frame). Greyscale-only until Session C lands the tokens; Sessions C–E carry full atmosphere.
4. **Numbered callouts:** small annotation markers on the layout, keyed to a footnote list — each callout names the design decision it embodies, so Christopher can point at a number and say "that one's wrong."
5. **RIPCORD ITEMS** (if any): boxed, verbatim from GLM, with Claude's recommendation.
6. **Command Ledger delta (§1.1):** every actuating control introduced by this draft appended to `Command_Ledger.md` — verb + args mapping, or `RETURN-SESSION` flag. The draft links its ledger rows so the future chat build is never forgotten.

Fidelity bar: structured enough to judge hierarchy, density, and feel; loose enough that nothing looks finished. Boxes with real labels beat polished fakes. No JS interactivity except where the interaction IS the deliverable (Session D card states may use trivial CSS-only state toggles).

---

## 8. Vision Confirmation Criteria (what Christopher judges)

Per session, Christopher answers **only** these — each a yes / "here's what's off".

**Standing spectrum check (every session, alongside the per-session questions):** would the 2-league casual feel at home in five minutes — and would the 25-league portfolio manager still be discovering depth in month three? Does every control feel light and instant?

| Session | Questions |
|---|---|
| A — Grid | 1. Does the shell read as one console, not four glued panels? 2. Is real estate obviously *earned* — the dense surface dominant, chrome recessive? 3. Would you know where to look first on every screen? |
| B — Components | 1. Does a 32-row board scan without effort at Tactical density? 2. Does the ConfirmModal feel like arming a system? 3. Do the deferred deliverables (table columns, inspector anatomy) show the right information in the right order? 4. Do the empty/loading/error states make a fresh install's first five minutes feel intentional, not broken? |
| C — Atmosphere | 1. Is this the room you want to run a league from at 11pm? 2. Do the four signal colors read instantly as meaning, not decoration? 3. Anduril or SaaS? |
| D — Command layer | 1. Can one eye scan chat, feed, deadlines, and alerts as a single hierarchy? 2. Does a trade card feel like a control surface, not a chat bubble? 3. Does the calendar show up where decisions happen, not just in a calendar tab? 4. Does the calendar feel like Google Calendar's fluidity in the console's skin — and is the quick-dash summon/collapse right? |
| E — Mobile harvest | 1. Is the survival table right — did the correct modes die? 2. Does the chat-first shell match your intent for mobile? 3. Is the lessons ledger sufficient to open a future mobile session cold? |

Rules of the gate: vision only, no pixel-pushing; a "here's what's off" triggers exactly one correction round (Claude may spend GLM freely on it); two consecutive misses on the same session = stop and re-frame the brief with Christopher before burning more rounds.

---

## 9. Sequencing & Session Flow

```
[Phase 0] Haiku research digest ─── done once, feeds all briefs
[Phase 1] Facet map ─────────────── this document §3 (done)
   │
   ▼  Christopher approves THIS PLAN  ◄── you are here
[A] grid ──► draft ──► ✅ ──► [B] components ──► draft ──► ✅ ──► [C] atmosphere ──► draft ──► ✅
                                                                      │
                                            [E] mobile ◄── ✅ ◄── [D] command layer ◄──┘
```

Each session's confirmed output is a standing attachment to every later brief — the system compounds. Session artifacts + GLM raw outputs are committed on this branch as they land; the branch merges only after Session E's confirmation (or earlier at Christopher's call).

**One session per sitting is the expected pace** — a draft is presented, Christopher confirms when he's ready, the next session fires. GLM runs for session N+1's divergent round MAY be pre-fired while awaiting confirmation on N *only* for Session B after A (grid → components is weakly coupled); C–E depend too tightly on the prior confirmation to pre-fire.

### 9.1 Build Track → Alpha (the roadmap's second half)

The wireframe loop is design; the roadmap ends at a **functional, Alpha-ready product in real-world testing**. After (or interleaved with, at Christopher's call) the confirmed wireframes, build sessions land in this order.

**Code standards — non-negotiable (Christopher's ruling, 2026-07-19).** Every build session runs under the full gate stack: session branch, the Christopher-Coding-Standards 11-layer overlay + Agent Codex loaded before any code, `make lint` 0 / `go test -race` green / tsc+vite clean, pre-commit hooks (golangci-lint + gitleaks + ifaceguard), GLM 5.2 blind review triaged vs source, live Beelink functional gate before merge. No build in this track is exempt because "it's just UI" — the frontend is where the D1–D9 discipline (engine-authoritative legality, no client-side money, `?? []` defensive rendering, preview→confirm→commit) lives or dies.

| Build | What lands | Consumes |
|---|---|---|
| B-1 Shell & tokens | The real 4-column shell + nav rail + CSS custom-property token system + density plumbing, replacing the flat testing-harness tab bar | Sessions A + C |
| B-2 Module migration | The shipped surfaces (M1, M2, Transaction Workspace, Trade Builder, League Controls, Admin) re-homed into the shell with Session-B component language + column specs | Session B |
| B-3 **Calendar — fully functional** | The complete calendar: views + quick-create + drag-to-move/resize (append-a-revision semantics) + quick-dash summon/collapse from any screen + operational overlays on governed surfaces. Consumes the `session/league-calendar` backend WIP (its own pending full-plan session folds in here) | Session D + gcal digest |
| B-4 Home & Inspector | League landing 2×2 card grid + seasonal card system (**Alpha may slim to the 3–4 highest-value seasonal cards** — the trigger map is locked, the full 9-card set is not an Alpha blocker) + the Contextual Inspector per Session-B anatomy | Sessions A/B/D |
| B-5 Alpha hardening | Command Ledger completeness check, empty/loading/error states verified on every surface (per the Session-B anatomy — verification here, DESIGN happens in B), data-freshness grammar live, performance budgets (§18 of the direction doc) verified live, **in-app Alpha feedback capture** (a dead-simple "flag this screen" — screen id + optional note + app version → append-only local log Christopher collects from testers; beats "text the commissioner"), packaging | All |
| **ALPHA GATE** | Christopher runs the league on it in the real environment. Functional-verification doctrine applies: not done until used. **HARD DEPENDENCY: the Versioning & Releases phase** (`docs/roadmap/Versioning_and_Releases_Planning.md`, D-V1…D-V7 — already scheduled, Christopher-led) **must complete before any Alpha binary leaves this machine**: stamped/reproducible builds and the schema-migration story (D-V6) are prerequisites of real-world testing, not parallel work. Named here so it is not discovered late. | — |

Not in this roadmap: the chat engine + LLM router (§1.1 — the ledger and Session-D grammar are its inheritance), Phase-2 multi-user features, mobile (Session E's handoff is its inheritance). Build-track ordering may be re-cut when the wireframe loop finishes; the fixed points are **B-3 calendar full function** and the **Alpha gate**.

**Scope doctrine (Christopher's ruling, 2026-07-19): the plan's biggest risk is not missing ideas — it is scope.** No more design sessions, no more research passes, no more configurability get added to this roadmap; the spectrum commitment (§1.2) is the guardrail that keeps GLM's creativity from bloating it. The posture is *ready to cut*: Session E is already marked cuttable, B-4's seasonal cards slim to 3–4 for Alpha, and any GLM proposal that grows the pipeline gets triaged against this doctrine first. Additions from here require Christopher naming them.

---

## 10. Open Items

- **SESSION A (Grid & Spatial System) CONFIRMED 2026-07-19 — "passed with flying colors" (Christopher, vision gate).** 3 GLM runs (instrument-cluster / density-gradient divergent + synthesis), triaged; fused draft at `docs/ui/wireframes/session-a/session-a-wireframe.html` (branch `session/ui-session-a-grid`). Confirmed decisions now bind every later brief: fixed 4-column instrument zones + density-gradient fill; comms right-edge summon-over; calendar generous summon-over overlay; league switcher top sector + Phase-3 injection seam at nav y=160; inspector fixed 320px transform overlay (NOT per-click width-trade); M1 at Tactical (32px header + 36px sticky filter, Adjusted-Score-dominant); Home 2×2 slow-down break; 4px base, no-dropshadow inset-bevel elevation. One Christopher-named addition captured (panel edge-resize → Session B / B-1, Ledger A12). Command Ledger seeded (A1–A12). **Session A is a standing attachment to Sessions B–E.**
- **SESSION 0 (test run) COMPLETE 2026-07-19** — the pipeline is proven end-to-end; the worked example every session follows is `docs/ui/wireframes/session-0-test/Session-0-Example.md` (brief → runner → 40s GLM direction → triage → greyscale draft + callouts → visual gate). **One ripcord awaits Christopher:** comms column → summoned overlay (Claude recommends adopting; it substantially matches the quick-dash rule).
- Phase-0 research digest: landed at the §2 path.
- Google-Calendar mechanics digest: landed at the §2 path (feeds Session D + B-3).
- Calendar full-plan session: memory records the league-calendar backend as WIP with "full plan session pending" — that planning folds into Session D (design) + B-3 (build) rather than running separately, unless Christopher wants it standalone.
- Append-only vs drag-drop: a dragged move/resize commits as an appended superseding revision (undo = another append). Session D must design this honestly; B-3 implements it. If the current backend WIP lacks a supersede/revision read-path, that surfaces as a B-3 backend work item, not a UI compromise.
- `scripts/glm-design-session.sh`: to be written (from the proven GLM-review runner pattern) as Session A's first step.
- Session E vs direction-doc §20 (mobile deferred to Phase 3): resolved 2026-07-19 — E is a **learning harvest, not a mobile build**. Its handoff document (survival table + lessons ledger + chat-first shell concept) is the input to future mobile design sessions at a later date; nothing in Phase 1 changes. E is hard-gated behind Session D's confirmation.
- **Panel edge-resize — Christopher-named addition (2026-07-19, at the Session A gate).** User-initiated edge-grab resize of the zone dividers (nav rail / workspace / inspector), like a normal window, for in-the-moment cockpit customization. This is NOT the rejected Run-2 auto-width-trade (that was an involuntary per-click reflow); this is a deliberate operator action, aligned with the spectrum law (expert personalizes, casual never touches) and the "depth discovered, not demanded" model (VS Code / Blender). Command verb already reserved: `ui.resize` (Ledger A12). **Scope: Session A stays fixed-zones (correct resting state); the resize-handle anatomy + its 4 interaction states are a Session-B deliverable; the behavior builds in B-1 (shell & tokens).** Hard constraint (same one that drove the fixed-zone ruling): the drag must be cheap — a 1px ghost divider that commits width on RELEASE, no live per-frame reflow, to respect the WebKitGTK paint floor. Presets (Draft Mode, Gameday) can ship canned widths.
