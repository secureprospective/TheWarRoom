# SESSION A — Grid & Spatial System · SYNTHESIS

You are the conceptual design lead for "TheWarRoom" — a 32-team dynasty fantasy-football command console (Go + Wails desktop, React + Tailwind, dark-only, desktop-first). Tone: Anduril, not SaaS — a command console that costs more than a house. Dark, precise, data-dense, not a pixel wasted, controls that actuate like hardware.

Two divergent design directions were produced for the 4-column shell spatial architecture, then triaged by the head brain. Your job: **FUSE them into ONE coherent, buildable spatial spec and resolve the named tensions exactly as the head brain ruled.** Do not re-litigate the rulings — implement them into a single consistent system, filling in the anatomy that makes them cohere. Where a ruling leaves a detail open, commit to a specific value and say why in one line.

## The two directions (committed specifics)

**RUN 1 — "Instrument Cluster":** app as physical hardware; workspace is the primary gauge, controls are bezels, negative space = structural isolation, real estate EARNED by fixed function. Rigid asymmetric static matrix (rejects 12-col drag grid). Panels: 32px header / 12px body inset / optional 24px status footer; no dropshadows, 1px border=containment, 1px inset highlight=raised → swaps to bottom/right shadow on click ("depressed button"). Comms 48px strip pinned BOTTOM edge → summons up as 320px footer. Calendar 48×48 date-stamp button top-right → square popover. League switcher TOP of nav rail: [PORTFOLIO SECTOR] top 160px above [MODULE SECTOR]. Home 2×2 = "Standby Mode", symmetric quadrant, Narrative density. RIPCORD: 48px bottom strip eats ~4.4% vertical (premium for 32-team tables).

**RUN 2 — "Density Gradient":** app as optical instrument; structure discovered by reading the density gradient, data is the structural load-bearer. Fluid 12-col topography — workspace & inspector fluidly TRADE width by active entity (nav rail fixed 192px). Data bleeds to screen edges; hairlines + gutters are the only breathing room. Panels: 40px header; no shadows; nothing (flat gray-950) / 1px hairline gray-800 boundary / focus = 2px LEFT-axis inset border + background lift (recessed switch). Comms 48px strip RIGHT edge over inspector → summons left to 320px. Calendar 32px TOP strip → summons down as 280px horizontal blade. League switcher BOTTOM 64px seam. M1: 40px header + 36px sticky filter sub-header (pins on scroll); Adjusted Score AND Base dominate. Matrix row 22px, hover highlights full row width. RIPCORD: none.

## Head-brain rulings (IMPLEMENT — do not relitigate)

1. **Grid soul = FIX THE ZONES, FILL THEM BY GRADIENT.** Adopt Run 1's fixed instrument-zone shell (zones have fixed roles; the inspector appears/collapses via `transform`, it does NOT fluidly negotiate width on every entity click — a width reflow per click violates the <100ms snappy law and the WebKitGTK cheap-paint floor). Adopt Run 2's WITHIN-ZONE fill philosophy: inside each fixed zone, data bleeds to the zone's edges, hairlines are the breathing room, zero wasted margin. Reconcile these into one rule set.
2. **Comms = Run 2 right-edge vertical.** 48px idle strip on the right edge over the inspector; summons leftward to 320px as a pure transform overlay, pushes nothing. (This costs zero table rows and resolves Run 1's own RIPCORD.)
3. **Calendar = a generous summon-over OVERLAY, not a corner popover.** It must one day hold a real week/month view (future build), so the summon geometry must accommodate a large panel. Design only the SEAM + summon geometry this session (where it lives idle, how it summons, what it overlays). Pick a specific idle affordance and a specific summoned footprint.
3b. **Coexistence:** define how calendar + comms coexist when both summoned (comms right-edge vertical; calendar your generous overlay) without collision.
4. **League switcher = Run 1 TOP sector.** Top of nav rail: a [PORTFOLIO] region above the [MODULE] region; a pinned portfolio/overview crest scales honestly 1→25+ leagues. Leave the exact geometric seam where a Phase-3 cross-league portfolio module later injects — say precisely where.
5. **Header = 32px module header (Run 1) + Run 2's 36px sticky filter sub-header for M1. Matrix row = 22px.**

## Also settled (both runs agreed — keep)
4px base unit; scale 4>8>12>16>24>32>48>64; NO dropshadows/blurs (inset-bevel elevation only, WebKitGTK floor); density tiers via CSS custom properties ONLY (Tactical=32px rows default); Home 2×2 as a DELIBERATE slow-down break (Basecamp vs The Field); Adjusted Score as the dominant M1 read; all quick-dash motion is pure `transform`/`translate` overlay.

## Deliver — ONE unified spatial spec, target ≤200 lines, in this order
1. **Base unit + spacing scale** (confirm).
2. **The fused grid thesis** — one paragraph resolving fixed-zones + gradient-fill into a single coherent rule; name the fixed zones and their widths (nav rail / workspace / inspector), state exactly how the inspector appears/collapses (transform, not width-trade), and where negative space lives.
3. **Panel anatomy + the earned border/elevation rule** — one reconciled rule set (fold Run 1's actuation/depressed-button interaction into Run 2's hairline/inset-focus system so they're one language, not two).
4. **Density-tier variable table** — name the CSS custom properties and give Narrative / Tactical / Matrix values (rows 48 / 32 / 22).
5. **M1 board treatment** — 32px header + 36px sticky filter sub-header; column dominance/recession; row anatomy at Tactical.
6. **Home 2×2** — the slow-down break, geometry.
7. **Quick-dash geometry** — comms (right-edge) + calendar (generous overlay) idle & summoned states + coexistence, all as transforms.
8. **League-switcher seam** — top sector, the 1→25 scaling behavior, and the exact Phase-3 injection seam.

Commit to named values throughout. Flag any residual uncertainty explicitly. No RIPCORD needed (rulings are final) unless a ruling is physically un-buildable — then say so precisely.
