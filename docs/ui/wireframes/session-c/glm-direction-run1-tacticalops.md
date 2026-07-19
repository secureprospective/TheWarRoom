# SESSION C — Tactical Ops Color & Atmosphere 

**Direction:** Near-black warm anthracite base. Instrumentation glows with backlit phosphor-warmth. The resting state is sharp, quiet, and mechanical. Cool semantic hues (green, blue) slice through the warmth as high-contrast signals. Amber is the native escalating pulse.

## 1. Base Atmosphere (`--surface-*`, `--edge-*`)
The ground is a warm anthracite. Elevation is strictly optical per Session A.

```css
/* The Canvas */
--surface-canvas: hsl(25, 8%, 6%);     /* Absolute base. Warm, near-black */
--surface-sunken: hsl(25, 6%, 4%);     /* Deep inset (e.g., layer bar tracks) */
--surface-tile: hsl(25, 10%, 9%);      /* Resting raised actionable (M1 rows) */
--surface-tile-hover: hsl(25, 12%, 12%); /* Mouse-over full row bleed */
--surface-tile-active: hsl(25, 14%, 15%); /* Depressed micro-switch state */
--surface-overlay: hsl(25, 12%, 14%);  /* Inspector panel, ConfirmModal */

/* The Bevel Mechanics (1px optical elevation per Session A) */
--edge-raise-highlight: hsl(35, 20%, 22%); /* Top/Left inset resting */
--edge-raise-shadow: hsl(25, 8%, 3%);     /* Bottom/Right inset resting */
--edge-press-highlight: hsl(25, 8%, 3%);  /* Top/Left inset :active */
--edge-press-shadow: hsl(35, 20%, 22%);   /* Bottom/Right inset :active */

/* Structural Hairlines */
--edge-hairline: hsl(25, 8%, 14%);      /* Static dividers */
--edge-hairline-strong: hsl(25, 10%, 20%); /* Emphasized dividers */
```

## 2. Semantic Ramps (`--signal-*`)
Locked convention: Green=elite, Blue=good, Amber=watch, Red=danger.

```css
/* Green: Elite / Positive Delta Reinforcement */
--signal-green-muted: hsl(160, 35%, 35%);  /* Cap-efficient salary */
--signal-green-base: hsl(158, 50%, 45%);   /* Score 80-89, +Delta reinforcement */
--signal-green-loud: hsl(155, 65%, 55%);   /* Score 90+, Top rank */

/* Blue: Good / Info */
--signal-blue-muted: hsl(210, 30%, 40%);   
--signal-blue-base: hsl(205, 50%, 55%);    /* Score 70-79, standard live info */
--signal-blue-loud: hsl(200, 65%, 65%);    

/* Amber: Watch / Escalating Time */
--signal-amber-muted: hsl(40, 30%, 30%);   
--signal-amber-base: hsl(38, 60%, 45%);    /* Score 55-69, resting watch */
--signal-amber-loud: hsl(35, 85%, 55%);    /* Bid clock < 1hr, high alert */

/* Red: Danger / Destructive Ops */
--signal-red-muted: hsl(0, 20%, 30%);      
--signal-red-base: hsl(355, 50%, 50%);     /* Score < 55, rank 32 */
--signal-red-loud: hsl(0, 75%, 60%);       /* ConfirmModal destructive op */
```

**Data Mapping:** 
Adjusted Score dictates hue: `>=90` (green-loud), `80-89` (green-base), `70-79` (blue-base), `55-69` (amber-base), `<55` (red-base).
Positive Delta: `font-weight: 600` + `--signal-green-base`. Negative Delta: `font-weight: 400` + `--signal-red-base`. 

## 3. Text Ramp (`--text-*`)
Faint warm whites over anthracite. Never pure `#FFF` (prevents eye-strain, breaks SaaS illusion).

```css
--text-hero: hsl(45, 15%, 96%);   /* 24px/700 Hero numerics (Adjusted Score) */
--text-primary: hsl(30, 10%, 88%);/* Standard data (Base Score, Player Name) */
--text-secondary: hsl(30, 8%, 70%);/* Franchise, sub-headers */
--text-muted: hsl(30, 6%, 45%);   /* Metadata, timestamps */
--text-disabled: hsl(30, 5%, 30%);/* Unavailable controls */
```

## 4. Categorical vs Semantic Tension (Position Badges)
**Resolution:** Categorical hues are strictly DESATURATED to ~15% saturation and mid-lightness. Semantic hues are highly saturated. Badges use an outline style rather than solid fill.

```css
--cat-qb: hsl(220, 15%, 55%);  /* Slate */
--cat-rb: hsl(285, 15%, 60%);  /* Mauve */
--cat-wr: hsl(160, 15%, 50%);  /* Sage */
--cat-te: hsl(35, 15%, 50%);   /* Bronze */
--cat-k: hsl(0, 0%, 60%);      /* Dust (Kickers are achromatic) */
--cat-def: hsl(15, 20%, 55%);  /* Rust */
```

## 5. Focus & Selection
```css
/* Machined Keyboard Focus Ring (Session B requirement) */
--edge-focus-kb: 0 0 0 1px var(--signal-amber-base) inset;

/* Mouse Selection (2px left-axis border on M1 rows) */
--edge-select-axis: var(--signal-blue-muted);
```

## 6. Restraint Rules (The Console Doctrine)
Color is scarce; scarcity creates signal.
1. **NO CHROMATIC BACKGROUNDS:** M1 table rows, Inspector tiles, and base canvases are strictly achromatic `--surface-*`. Do not tint row backgrounds green for a positive delta.
2. **METRICS ONLY:** Color belongs exclusively to Data (Scores, Deltas, Clocks) and State (Focus, Active, Hold-to-Fire fill). 
3. **NO ICON CHROME:** Sort arrows `▼/▲` use `--text-muted`. Only active sort uses `--text-hero`.
4. **MATRIX IS ACHROMATIC:** In Matrix density tier, all semantic colors yield to `--text-primary` except for Red Danger (`--signal-red-base`). Hierarchy relies entirely on Delta-in-Weight.

## 7. Motion Doctrine (WebKitGTK Budget)
Transforms/opacity only. ≤150ms. Matrix density terminates all transitions (`transition: none !important`).
*   **Feed Event Arrival:** `opacity: 0` + `translateY(8px)` to `0/0` (100ms ease-out). 
*   **Hold-to-Fire (ConfirmModal):** `scaleX(0)` to `scaleX(1)` transform origin left (500ms linear) colored `--signal-red-loud`. Reject state pulses opacity 1 to 0.8 (200ms).
*   **Clock < 1hr:** Pure opacity flicker on the timestamp text from `--text-secondary` to `--signal-amber-loud` (1s interval). No background pulsing.

## 8. Data-Freshness Honesty (`--freshness-*`)
Freshness is indicated by a 2px top-edge state line on the 36px sticky sub-header.

```css
--freshness-live: var(--signal-green-loud);  /* Live MFL fetch */
--freshness-cache: var(--signal-amber-base); /* Serving stale cache, fetch in flight/failed */
--freshness-offseason: transparent;          /* No edge, clean state */
```
**Grammar:** If `--freshness-cache` is active, the board’s timestamp text shifts to `--text-muted` and the 2px top edge illuminates. The data itself remains fully colored—do not wash out the numbers, as this is a money-decision surface requiring legibility.

---

## RIPCORD ITEMS
**1. Amber Warmth vs. Amber Signal Collision**
*   **The Decision:** The prompt requires a warm base (warm anthracite) and Amber as the locked semantic warning/watch color.
*   **The Objection:** A warm base naturally skews toward amber/yellow. If the base `--surface-canvas` and `--surface-tile` hues (currently H:25) are too saturated, the `--signal-amber-base` (H:38) will blend in visually rather than popping as a warning state, weakening the semantic contract.
*   **The Alternative/Fix:** I have severely desaturated the base surfaces (S: 6-14%) and pushed the `--signal-amber-loud` to a high luminance/saturation (H:35, 85% S, 55% L). If this still lacks contrast during prototyping, we must shift the base canvas hue to a neutral-warm gray (H:220, S:5%, L:6%) to ensure the amber truly glows.
