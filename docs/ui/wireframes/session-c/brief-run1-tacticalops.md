# SESSION C — Color, Dark Mode & Atmosphere · RUN 1

You are the conceptual design lead for "TheWarRoom" — a 32-team dynasty fantasy-football command console (Go + Wails desktop, React + Tailwind, dark-only, desktop-first). The design target is Anduril, not SaaS: a command console that costs more than a house. Dark, precise, data-dense without a pixel of waste. Confident hierarchy. Controls that look like they actuate real hardware. The UI must communicate capability before the user clicks anything.

You have creative latitude and you are expected to use it — push past tired dashboard patterns. But two rules bind you: (1) the locked-context below is LOCKED — Sessions A and B are CONFIRMED and are not yours to redesign; if you believe a locked decision is wrong, write a section titled RIPCORD ITEMS stating the decision, your objection, and your alternative — do not design around it silently. (2) Ground every proposal in the real surfaces and real data shapes in the facet map below — no invented features.

Technical floor: WebKitGTK webview (motion = transforms/opacity only, cheap compositing — no heavy filters/blurs on large surfaces; the Beelink APU compositor has locked up on expensive paint), CSS custom properties carry ALL tokens, virtualized lists, Tailwind-expressible.

Standing architectural fact: a future Chat engine will have COMPLETE control of this application — a terminal over a Guix-style simple command language. Every mechanical control maps to a command verb; you are not designing that this session, but nothing you propose may be impossible to express as a verb.

Standing design law — the spectrum commitment: this one product must delight a 1–2-league casual AND a 15–25-league hardcore portfolio manager. (1) light, snappy, speedy — every control acknowledges in <100ms, motion is feedback not decoration; (2) layered engagement — every surface reads at three altitudes (glance / operate / interrogate), depth DISCOVERED never demanded; (3) configurability lands as one-click presets, never toggles to study. If a proposal serves one end of the spectrum by taxing the other, redesign it.

## Locked context — Session A CONFIRMED spatial system (build ON this)
- **Base unit 4px.** Scale 4 › 8 › 12 › 16 › 24 › 32 › 48 › 64. No off-scale values.
- **Elevation is optical only:** 1px hairline borders + inset bevels. NO dropshadows, NO blurs (WebKitGTK cheap-paint floor). Raised actionable tile = 1px inset top/left highlight; `:active` swaps to 1px bottom/right inset (depressed micro-switch). This actuation language is CONFIRMED — your color system must serve it (a bevel highlight/shadow needs a token, not a hardcoded rgba).
- **Fixed 4-column instrument shell:** nav rail (192px) / fluid workspace / contextual inspector (fixed 320px `transform: translateX` overlay) / comms (right-edge 48px idle strip, summons over at 320px). Calendar summons as a generous top-right overlay.
- **Density tiers Narrative(48px) / Tactical(32px, default) / Matrix(22px) ride CSS custom properties ONLY.** No "Pro Mode" toggle ever.
- **M1 board:** 32px module header + 36px sticky filter sub-header; Adjusted Score + Base Score dominate; active row = 2px LEFT-axis inset border; hover highlights full row across the workspace bleed.

## Locked context — Session B CONFIRMED component language (build ON this)
- **Type:** Inter = text, JetBrains Mono = data. Hero numeric 24px/700 `tabular-nums` right-aligned (currently gray-50 — you assign its color token). Section labels 11px uppercase 0.05em tracking.
- **DELTA-IN-WEIGHT (critical for you):** positive Δ renders at font-weight 600, negative Δ at 400 — a GREYSCALE-HONEST signal that already carries +/− without any color. Session C may add HUE to REINFORCE it, but the weight encoding SHIPS and is the baseline. Do NOT design a color system that assumes hue is the only delta signal; hue is redundant reinforcement layered on a signal that already works in pure greyscale (and must, for the colorblind and for the Matrix tier).
- **Sort indicator is typographic:** ▼/▲ active, ¦ inactive. No icon chrome. (These get a text-color token, not a new color.)
- **Table rows:** no vertical gridlines — optical spacing + row-hover background bleed inside the inset-bevel shell. 4 row states: rest / hover(raised inset) / selected(2px left axis) / active(pressed inset). Each needs surface + edge tokens.
- **M1 columns = locked 7-col facet map** Rank·Player·Pos·Franchise·Base·Adjusted·Salary (Matrix drops Franchise+Base). Position badges (QB/RB/WR/TE/…) need a palette WITHIN the locked semantic convention.
- **ConfirmModal = 480px CENTERED + HOLD-TO-FIRE** (single verb `txn.commit`, ≤600ms fill, engine-reject holds non-dismissable). The fill/arm/fire/reject states need motion + color tokens.
- **Inspector:** score-dominant header + 6 layer bars (4px, currently gray-800 track / gray-100 fill) + terminal-style contract block. The layer bars are a prime candidate for the semantic ramp.
- **Empty/Loading/Error = ONE class:** engraved fresh-install / data-shaped skeletons (NOT spinners) / **cache-only state-line + 2px top edge** / offseason clean-no-edge. Session B rendered the cache-only edge in greyscale with red RESERVED for you — the `--freshness-*` family is the through-line here.

## Facet map — real surfaces & data shapes this session must honor
- **M1 Asset Rankings board:** Rank, Player, Pos, Franchise, Base, Adjusted Score, Salary. Plausible row: `4 · Jahmyr Gibbs · RB · Motor City Maulers · 71.20 · 88.40 · $42.10`. Scores span ~30–95; a score-88 is elite (green), a score-52 is muted. Salary $0.5M–$60M+; cap efficiency has its own green.
- **M2 Power Rankings board:** franchise table, blended power score, MFL all-play%, PA. Ranks 1–32 (rank-1 elite, rank-32 danger-adjacent).
- **Contextual Inspector (320px):** engine score dominant, 6 layer-breakdown bars, contract block, notes.
- **Transaction / Trade / League Controls:** ConfirmModal is the commit gate for WAIVER/SIGN/TAG/EXTENSION/BUYOUT/TRADE.
- **Comms + Feed + Calendar:** feed events arrive live; a bid clock crosses under 1 hour (amber→? escalation); a snipe alert (danger). These drive your MOTION doctrine.
- **Real data states:** live MFL fetch in flight, MFL-unreachable failed refresh showing stale cache, fresh install, offseason emptiness. Freshness honesty is a MONEY-DECISION trust surface.

## Your charge — Session C (answer with COMMITTED tokens, named values)
Build the functional color system as CSS custom-property tokens. **The semantic layer is LOCKED: green=elite/positive, blue=good/info, amber=warning/watch, red=danger, gray=muted — one meaning per color, everywhere.** Everything else is yours:
1. **Base atmosphere** — dark ≠ gray. Pick a temperature and commit to exact values: the `--surface-*` elevation ramp (base canvas → sunken → raised tile → overlay → the inset-bevel highlight & shadow tokens the confirmed actuation language needs). Name every step with a hex/hsl value.
2. **The semantic ramps** — exact values for each locked color AND its ramp: a score-88 green and a cap-efficient green need family resemblance without ambiguity; amber-watch vs amber on a clock crossing 1hr; red-danger vs red-destructive-op. Define `--signal-{green,blue,amber,red}-{muted,base,loud}` (or your naming) with values. State how each maps to real data (score band → ramp step).
3. **Text ramp** — `--text-*` over the atmosphere: hero / primary / secondary / muted / disabled, each a value, each meeting legibility on the base canvas.
4. **Position-badge palette** — QB/RB/WR/TE/K/DEF within the locked convention (these are CATEGORICAL, not semantic — resolve the tension: how do categorical badge hues coexist with semantic signal hues without either polluting the other?).
5. **Focus / selection treatment** — the machined focus ring (Session B floated 1px gray-400 inset) gets a color token; keyboard-focus vs mouse-selection distinction.
6. **Restraint rules — THE MOST IMPORTANT PART:** in a console, color is signal. Define what is FORBIDDEN to be colored so signals stay loud. Which surfaces are permanently achromatic. The rule that keeps a 500-row board from lighting up like a Christmas tree.
7. **Motion doctrine** — the micro-animation language for async ops (preview pending, commit in flight, feed event arrival, clock under 1hr), under the WebKitGTK budget: transforms/opacity only, ≤200ms, never blocking, every animation earns its place as FEEDBACK ("did it register? is it working? did it land?"). Decoration is cut. **Matrix density kills all of it** — state that explicitly.
8. **DATA-FRESHNESS HONESTY** — `--freshness-*` tokens + the visual grammar for "as of when": a live-fetched surface vs a cached board vs a failed refresh showing stale data, each unmistakable at a glance without shouting (timestamp treatment / edge state / muted badge — your call), consistent across every surface, reinforcing Session B's cache-only state-line.

Deliver the full token sheet: `--surface-*`, `--signal-*`, `--text-*`, `--edge-*`, `--freshness-*`, named and valued.

## Provocation seed for THIS run
**"TACTICAL OPS — NEAR-BLACK, AMBER-WARM INSTRUMENTATION"** — the atmosphere of a forward operations console at night: near-black base with a faint WARM cast (not neutral gray, not blue — a hint of warm anthracite), and instrumentation that glows like backlit amber-and-white gauges. Warmth is the resting state; the cool semantic colors (blue, green) read as COOLER against a warm ground, which makes them pop as signal. Amber is at home here — the watch/warning state feels native, not alarming, until it escalates. Push the warm-instrument feel hard: think illuminated milspec panel, phosphor-warm readouts, the confident glow of a system that runs all night. Then be honest in RIPCORD about where warmth muddies the green/red legibility or fights the "dark ≠ gray, but also ≠ sepia" line.

## Output
An opinionated design-direction document in markdown, **target ≤250 lines**. Commit to specifics — named hex/hsl values, named token names, named ramp steps, named motion durations — not mood language. Every token must have a value. Flag uncertainty explicitly rather than guessing. End with a **RIPCORD ITEMS** section (or "none").
