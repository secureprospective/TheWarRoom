# SESSION C — Color, Dark Mode & Atmosphere

**Direction:** NAVAL CIC. The interface is a cold, deep plot table in a darkened bridge. The ground recedes into desaturated anthracite-navy depth. The data floats above it as illuminated phosphor tracks. Green and Blue are the cool, steady state of the machine. Amber and Red are heat—they command attention by temperature contrast before the brain even processes the hue. 

Darkness here is not a void; it is a controlled environment built for high-altitude visual acuity.

---

## 1. Base Atmosphere (`--surface-*` & `--edge-*`)
Dark ≠ gray. The base is a deep, cool midnight blue (`hsl(220, 20%, 6%)`). Elevation is strictly optical via Session A's inset bevels—no dropshadows. 

```css
/* The Deep Plot Table */
--surface-canvas:   #0A0E14; /* hsl(220, 20%, 6%) - The lowest depth */
--surface-sunken:   #070A0E; /* hsl(220, 20%, 4%) - Input wells, depressed tiles */
--surface-tile:     #11161E; /* hsl(220, 20%, 9%) - Raised actionable rows/tiles */
--surface-overlay:  #161C26; /* hsl(220, 22%, 12%) - Modals, Inspector, Calendar */

/* The Machined Bevels (Session A Actuation Language) */
--edge-hairline:        #1C2330; /* Resting 1px borders */
--edge-bevel-highlight: rgba(255, 255, 255, 0.04); /* Inset top/left for raised tiles */
--edge-bevel-shadow:    rgba(0, 0, 0, 0.5); /* Inset bottom/right for :active state */
```

## 2. The Semantic Ramps (`--signal-*`)
One meaning per color. Amber and Red carry warm luminosity to pop against the cold base. Green is phosphor radar.

```css
/* GREEN: Elite / Positive / Cap-Efficient */
--signal-green-loud:  #00FF9D; /* Score 90+. The bright radar return. */
--signal-green-base:  #2EA043; /* Score 75-89, standard positive Δ, cap-efficient */
--signal-green-muted: #176F2C; /* Background fills, layer bars */

/* BLUE: Good / Info / Friendly Track */
--signal-blue-loud:   #79C0FF; /* Live feed arrivals, active info */
--signal-blue-base:   #1F6FEB; /* Standard info, focused inputs */
--signal-blue-muted:  #0D419D; /* Background fills */

/* AMBER: Warning / Watch (The Heat) */
--signal-amber-loud:  #FFB224; /* Bid clock < 60 mins */
--signal-amber-base:  #D29922; /* Standard warning */
--signal-amber-muted: #9E6A03; /* Background fills */

/* RED: Danger / Destructive / Reject */
--signal-red-loud:    #FF5252; /* Snipe alert, engine-reject hold */
--signal-red-base:    #F85149; /* Failed refresh, destructive op */
--signal-red-muted:   #A53737; /* Background fills */
```

## 3. Text Ramp (`--text-*`)
Designed for WCAG AAA contrast on the midnight canvas. Hero numerics are cool, pure white.

```css
--text-hero:     #F0F6FC; /* 24px Adjusted Scores. Pure, dominant. */
--text-primary:  #C9D1D9; /* Standard data and labels */
--text-secondary:#8B949E; /* Section labels (11px), supportive text */
--text-muted:    #484F58; /* Timestamps, disabled states */
--text-disabled: #2D333B;
```

## 4. Position-Badge Palette (Categorical)
**Tension resolved:** Categorical badges will NOT use solid fills. They use a neutral `--surface-sunken` background with a 1px colored border and text. This ensures 500 rows don't look like a rainbow, while preserving the semantic signal hierarchy (Green/Red). 

```css
--badge-qb-text: #79C0FF; --badge-qb-border: rgba(121, 192, 255, 0.3);
--badge-rb-text: #D2A8FF; --badge-rb-border: rgba(210, 168, 255, 0.3);
--badge-wr-text: #FFA657; --badge-wr-border: rgba(255, 166, 87, 0.3);
--badge-te-text: #7EE787; --badge-te-border: rgba(126, 231, 135, 0.3);
--badge-def-text:#FF7B72; --badge-def-border:rgba(255, 123, 114, 0.3);
--badge-k-text:  #C9D1D9; --badge-k-border: rgba(201, 209, 217, 0.2);
```

## 5. Focus & Selection Treatment
*   **Keyboard Focus:** `outline: 1px solid var(--signal-blue-base); outline-offset: -1px;` (Validates the Session B proposal).
*   **Mouse Selection:** `--edge-axis-active: 1px solid var(--signal-green-loud);` applied to the left edge of the row (Session A).

## 6. Restraint Rules (The Christmas Tree Axiom)
*Color is signal. If everything is colored, nothing is.*
1.  **Rows are Achromatic:** At rest, a 500-row board is pure grayscale text on `--surface-canvas`. 
2.  **Data-Cell Coloring:** Only Adjusted Score and Δ values receive semantic hues. Player names, ranks, and salaries remain `--text-primary`.
3.  **Hover/Select States:** Do NOT introduce color on hover. Hover is purely optical (background shifts to `--surface-tile` + bevel highlight). Selection shifts the left border to Green.
4.  **Icons:** UI icons (arrows, settings, filters) are strictly `--text-secondary`. Only system-state icons (Live, Error, Clock) get semantic hues.

## 7. Motion Doctrine (WebKitGTK Budget)
Transforms and opacity only. `transition: all 120ms cubic-bezier(0.4, 0, 0.2, 1);`
*   **Feed Arrival:** `transform: translateX(8px); opacity: 0;` -> `0`. (Shifts into view smoothly).
*   **ConfirmModal (Hold-to-Fire):** `transform: scaleX(0)` -> `1` (origin-left) over ≤600ms. Fill color transitions from `--signal-blue-base` to `--signal-green-loud` at 90% fill. Engine reject flashes `--signal-red-loud` via `opacity` for 200ms.
*   **Matrix Density Kills Motion:** In the 22px Matrix tier, `--motion-multiplier: 0`. All transitions bypass to instant state changes to preserve APU cycles and data-scanning focus.

## 8. Data-Freshness Honesty (`--freshness-*`)
Money-decisions require absolute trust. Freshness is communicated via a 1px top edge (extending Session B's state-line) and timestamp typography.

```css
--freshness-live-edge:   1px solid var(--signal-blue-base);  /* MFL fetch in flight */
--freshness-stale-edge:  1px solid var(--signal-amber-base); /* Cache-only fallback */
--freshness-failed-edge: 2px solid var(--signal-red-base);   /* Network failure */
--freshness-offseason:   0; /* Clean, no edge */
```
*   **Live:** Timestamp renders in `--signal-blue-loud` (e.g., `11:02:45 AM`).
*   **Stale:** Timestamp renders in `--text-muted`. Appended with `(cache)`.
*   **Failed:** Timestamp renders in `--signal-red-base`. Appended with `(offline)`.

---

## RIPCORD ITEMS
1.  **Objection:** The deep cool midnight (`#0A0E14`) base severely muddies standard Blue info signals (`#1F6FEB`).
    *   *Alternative/Action:* I assigned `--signal-blue-loud` (`#79C0FF`) for all Blue *text* and borders. The base blue is strictly reserved for solid fills (like the hold-to-fire modal track). If Blue text legibility fails testing, we must warm the surface canvas slightly (e.g., `hsl(220, 15%, 8%)`).
2.  **Objection:** The CIC mandate makes the resting console highly clinical. A casual 1-league user might find the all-achromatic grid intimidating rather than commanding.
    *   *Alternative/Action:* I mandated that the Franchise names and Position Badges (the "fantasy" elements) retain *muted* color borders to provide inherent visual interest without violating the restraint rules. If this feels too strict, we introduce a one-click "Vibrant" preset that saturates the categorical badges, keeping signals intact.
