# SESSION A — Grid & Spatial System · RUN 1

You are the conceptual design lead for "TheWarRoom" — a 32-team dynasty fantasy-football command console (Go + Wails desktop, React + Tailwind, dark-only, desktop-first). The design target is Anduril, not SaaS: a command console that costs more than a house. Dark, precise, data-dense without a pixel of waste. Confident hierarchy. Controls that look like they actuate real hardware. The UI must communicate capability before the user clicks anything.

You have creative latitude and you are expected to use it — push past tired dashboard patterns. But two rules bind you: (1) the locked-context below is LOCKED information architecture — do not redesign navigation, density tiers, color semantics, roles, or the comms model; if you believe a locked decision is wrong, write a section titled RIPCORD ITEMS stating the decision, your objection, and your alternative — do not design around it silently. (2) Ground every proposal in the real surfaces and real data shapes in the facet map below — no invented features.

Technical floor: WebKitGTK webview (motion = transforms/opacity only, cheap compositing — no heavy filters/blurs on large surfaces; the Beelink APU compositor has locked up on expensive paint), CSS custom properties carry ALL tokens, virtualized lists, Tailwind-expressible.

Standing architectural fact: a future Chat engine will have COMPLETE control of this application — a terminal over a Guix-style simple command language, with a lightweight LLM translating user language into commands. You are not designing that chat box, but every mechanical control you propose must be conceivable as a command verb + arguments that the GUI merely projects. If you propose an interaction with no plausible command equivalent, flag it yourself.

Standing design law — the spectrum commitment: this one product must delight a 1–2-league casual AND a 15–25-league hardcore portfolio manager. Three rules bind everything you propose: (1) light, snappy, speedy — every control acknowledges in <100ms, motion is feedback not decoration, nothing may feel heavy; (2) layered engagement — every surface must read at three altitudes (glance / operate / interrogate) with depth DISCOVERED, never demanded: the casual is never shown configuration to get started, the expert is never capped; (3) configurability lands as one-click presets to adopt, never toggles to study — defaults must be excellent because casuals never configure. If a proposal serves one end of the spectrum by taxing the other, redesign it.

## Locked context (do not redesign — flag as RIPCORD if you must challenge)
- **4-column shell:** nav rail (~160–200px, 6 modules) / fluid workspace / contextual inspector (~200–320px, populates on any entity click, no navigation) / comms panel.
- **Comms panel — RULING (Christopher, 2026-07-19): comms is COLLAPSED BY DEFAULT and SUMMONS OVER as an overlay; its idle state is the 48px strip. It is NOT a permanently-parked fourth column.** This is the confirmed baseline — design the shell geometry around a workspace that owns the comms column's real estate until comms is summoned.
- **Density tiers** Narrative / Tactical / Matrix ride CSS custom properties ONLY; Tactical is the default. There is NO "Pro Mode" toggle ever — the density tiers + a curiosity trigger ARE the progression system.
- **Color:** one meaning per color (green=elite/positive, blue=good/info, amber=warning/watch, red=danger, gray=muted). You are NOT setting values this session (that is Session C) — work in greyscale.
- **Quick-dash rule (binding, shell-level):** BOTH the calendar and the comms/chat panel must be summonable with ONE quick action and collapsible fully off-screen FROM ANY SCREEN in the app.

## Facet map — real surfaces this session must honor
- **M1 Asset Rankings board** (reference dense module): a 32-team ranked player table. Real columns: Rank, Player, Pos, Franchise, Base, Adjusted Score, Salary. Has a position filter and a view toggle (Global / By franchise / Cap efficiency). This is the "scan-dense" archetype.
- **Home / League Landing:** a 2×2 card grid + a seasonal card system (calendar-triggered cards) — designed, unbuilt.
- **Contextual Inspector:** populates on any entity click, no navigation — engine score dominant, layer-breakdown, contract block, notes.
- Other dense modules that inherit the shell: M2 Power Rankings (table + live weight slider), Transaction Workspace (franchise rail → roster → per-player action panel), Trade Builder, League Controls (commissioner console).

## Your charge
Invent the spatial architecture for the 4-column shell and everything inside it. The direction doc fixes the columns' existence, not their feel. Answer with COMMITTED RULES (named values, not mood language):
1. **Base unit + spacing scale** — one number, its derived scale, one-line reason.
2. **Workspace grid thesis** — is it symmetric or does information density drive an asymmetric grid? What makes it NOT a generic dashboard grid? Where does negative space live in a data-dense console (it must exist somewhere or nothing reads)?
3. **Panel anatomy** — header / body / chrome parts, their heights/weights, and the RULE for how a panel earns a border vs elevation vs nothing.
4. **Density-tier behavior** — how the grid + panels shift across Narrative/Tactical/Matrix using ONLY CSS-variable swaps (name the variables).
5. **The M1 board treatment** — header band, filter placement, row height at Tactical, what dominates visually and what recedes.
6. **Home 2×2 grid relationship** — same grid language as the dense modules, or a deliberate break? Why?
7. **Quick-dash geometry** — where the calendar and comms live when collapsed, what they overlay or displace when summoned, how both coexist if summoned together. The comms 48px strip is the locked idle state; the calendar summon geometry is YOURS to invent.
8. **League-switcher seam** — the nav rail's league switcher must scale honestly from 1 league to 25+ (a dropdown that works at 2 embarrasses itself at 25). Do NOT design Phase-3 cross-league dashboards — just leave the geometric seam where a portfolio surface could later live, and say exactly where it is.

## Provocation seed for THIS run
**"INSTRUMENT CLUSTER"** — zones with fixed roles, asymmetric, real estate is EARNED. The workspace is not a flexible grid of equal cells; it is an instrument panel where each zone has a job and its size reflects that job's importance. The dense data surface is the primary instrument; everything else is bezel, gauge, or readout arranged around it.

## Output
An opinionated design-direction document in markdown, **target ≤250 lines**. Commit to specifics — named values, named rules, named anatomy. Flag uncertainty explicitly rather than guessing. End with a **RIPCORD ITEMS** section (or "none").
