### THE UNIFIED TOKEN SHEET

```css
:root {
  /* 1. SURFACE RAMP (Cold CIC Base) */
  --surface-canvas: hsl(220, 20%, 6%);       /* App background */
  --surface-sunken: hsl(220, 22%, 4%);       /* Inset data wells, table body */
  --surface-tile: hsl(220, 18%, 9%);         /* Base row / standard tile */
  --surface-raised: hsl(220, 18%, 11%);      /* Hover state */
  --surface-overlay: hsl(220, 22%, 13%);     /* Modals, ConfirmModal bg */

  /* 2. OPTICAL ELEVATION & STRUCTURAL EDGES */
  --bevel-highlight: 1px solid hsl(220, 15%, 16%);
  --bevel-shadow: inset 0 1px 0 0 hsl(220, 25%, 2%);
  --edge-hairline: 1px solid hsl(220, 15%, 14%);

  /* 3. TEXT & CHROME */
  --text-primary: hsl(220, 20%, 92%);        /* High contrast data & headers */
  --text-secondary: hsl(220, 12%, 65%);      /* Labels, secondary data */
  --text-tertiary: hsl(220, 10%, 45%);       /* Sort arrows, UI chrome icons */
  --text-disabled: hsl(220, 10%, 30%);

  /* 4. SEMANTIC RAMPS (Mid-Saturation, tuned for Cold Base) */
  --signal-green-muted: hsl(145, 35%, 45%);
  --signal-green-base: hsl(145, 45%, 55%);   /* Tuned: slight L bump to cut cold */
  --signal-green-loud: hsl(145, 55%, 68%);   /* Tuned: L raised to pop on navy */

  --signal-blue-muted: hsl(210, 45%, 50%);
  --signal-blue-base: hsl(210, 60%, 62%);    /* Tuned: S raised to read clearly */
  --signal-blue-loud: hsl(210, 75%, 75%);

  --signal-amber-muted: hsl(35, 40%, 45%);
  --signal-amber-base: hsl(35, 60%, 58%);    /* Tuned: L pushed for max clarity */
  --signal-amber-loud: hsl(35, 80%, 70%);

  --signal-red-muted: hsl(0, 40%, 50%);
  --signal-red-base: hsl(0, 55%, 62%);       /* Tuned: S dropped, L raised */
  --signal-red-loud: hsl(0, 70%, 72%);

  /* 5. STATE EDGES */
  --edge-focus: inset 0 0 0 1px var(--signal-blue-base); /* Keyboard focus */
  --edge-selection: 2px solid var(--text-primary);        /* Neutral active axis */

  /* 6. FRESHNESS CHROME (Composed) */
  --edge-freshness-live: 1px solid var(--signal-blue-base);
  --edge-freshness-stale: 1px solid var(--signal-amber-muted);
  --edge-freshness-failed: 2px solid var(--signal-red-loud);
  
  --text-timestamp-live: var(--text-secondary);
  --text-timestamp-stale: var(--signal-amber-base);
  --text-timestamp-failed: var(--signal-red-loud);
  --text-suffix-live: var(--text-tertiary);   /* e.g., "(cache)" */
  --text-suffix-offline: var(--signal-red-base);

  /* 7. POSITION BADGES (Categorical - Sunken style) */
  /* Hues explicitly muted to read as "tags", isolating them from semantic signals */
  --badge-qb-hue: 280; /* Violet */
  --badge-rb-hue: 190; /* Cyan */
  --badge-wr-hue: 45;  /* Gold */
  --badge-te-hue: 120; /* Forest */
  --badge-bg: hsla(var(--badge-hue), 10%, 20%, 0.5);
  --badge-border: 1px solid hsla(var(--badge-hue), 25%, 50%);
  --badge-text: hsla(var(--badge-hue), 35%, 75%);

  /* 8. MOTION DOCTRINE */
  --motion-multiplier: 1; /* Matrix tier sets to 0 */
  --motion-feed-arrival: 100ms ease-out;   /* TranslateY+Fade */
  --motion-hold-to-fire: 150ms linear;     /* scaleX fill */
  --motion-engine-reject: 150ms ease-in-out; /* Opacity red flash */
  --motion-bid-clock: 50ms step-end;       /* Amber timestamp tick */
}
```

### SCORE → HUE BANDING MAP
Used strictly for the Adjusted Score value and its horizontal layer bar in the Inspector.
*   **≥ 90:** `var(--signal-green-loud)`
*   **80 – 89:** `var(--signal-green-base)`
*   **70 – 79:** `var(--signal-blue-base)`
*   **55 – 69:** `var(--signal-amber-base)`
*   **< 55:** `var(--signal-red-base)`

### RESTRAINT DOCTRINE (THE 500-ROW RULE)
**"Color is Data and State. Structure is Achromatic."**
1.  **NO ROW TINTING:** Never apply semantic hue to a row, tile, or canvas background. Hover, select, and active states must be achieved purely via the `--surface-*` ramp and `--bevel-*` optics.
2.  **CHROME IS GRAY:** Sort arrows (▼/▲/¦), icons, gridlines, and dividers use `--text-*` or `--edge-hairline`. Never use semantic colors for chrome.
3.  **DELTA HONESTY:** `+Δ` (weight 600) and `−Δ` (weight 400) operate perfectly in greyscale. The semantic hue (green/red) is purely reinforcement. 
4.  **MATRIX YIELD:** In Matrix density (22px), semantic hue yields entirely to `--text-primary` to maximize scanability—**EXCEPT** `--signal-red-loud`, which is retained to flag critical danger/benchings.

### MOTION DOCTRINE
*   **Constraints:** Transform/Opacity only. No layout shifts, no blurs. `--motion-multiplier: 0` (Matrix) universally disables transitions.
*   **The 4 Named Cases:**
    1.  *Feed Arrival:* `transform: translateY(8px)` -> `0` + `opacity: 0` -> `1` (`var(--motion-feed-arrival)`).
    2.  *Hold-to-Fire:* `transform: scaleX(0)` -> `scaleX(1)` fill on ConfirmModal trigger (`var(--motion-hold-to-fire)`).
    3.  *Engine Reject:* `opacity: 1` -> `0.2` -> `1` rapid red flash on validation failure (`var(--motion-engine-reject)`).
    4.  *Clock < 1hr:* Timestamp text shifts to `var(--signal-amber-base)`, stepping on minute boundaries (`var(--motion-bid-clock)`).

### RESIDUAL CHOICES
*   **Badge Contrast Verification:** The categorical badge hues (`Violet, Cyan, Gold, Forest`) were chosen specifically because they do not overlap with the semantic hues (`Green, Blue, Amber, Red`). L/S values were kept artificially low (`10% L` bg / `35% S` text) to guarantee they read as inert categorical tags rather than actionable alerts, but this will require visual confirmation on actual displays to ensure they don't muddy into the background.
*   **Sunken Data Well Scope:** By committing to `--surface-sunken` for table bodies, we create a micro-optical inset against the `--surface-canvas`. We need to ensure the `--bevel-shadow` is applied to the wrapper, not individual rows, to prevent double-beveling on 500-row scrolls.
