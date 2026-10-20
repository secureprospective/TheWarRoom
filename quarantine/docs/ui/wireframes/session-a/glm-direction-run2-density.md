# THEWARROOM — SESSION A: SPATIAL ARCHITECTURE (RUN 2)

**Design Ethos:** Anduril command console. Dark, absolute, hardware-actuated. Zero wasted pixels. 

We are not building a SaaS dashboard; we are building an optical instrument. Structure is discovered by reading the density gradient, not by counting grid cells. 

---

### 1. BASE UNIT + SPACING SCALE
*   **Base Unit:** `4px` (0.25rem). 
*   **Scale:** `4 > 8 > 12 > 16 > 24 > 32 > 48 > 64`.
*   **Rule:** All padding, margins, and gaps must be multiples of 4. No arbitrary values. This guarantees crisp rendering on the Beelink APU by eliminating sub-pixel interpolation. 

### 2. WORKSPACE GRID THESIS (DENSITY GRADIENT)
The workspace is a **12-column fluid topography**, not a Bento box. Information density dictates footprint, driving an asymmetric grid. 
*   **The Mechanism:** The fluid workspace, Inspector, and Nav Rail share 100% of the viewport width. The Nav Rail is fixed (192px), but the Workspace and Inspector fluidly trade real estate based on the active entity. 
*   **Negative Space (Null Zones):** In a data-dense console, negative space is not empty margins; it is active *buffering* used to anchor high-density zones. Negative space lives exclusively at the **Top (Header)** and **Gutters (Column gaps)**. The data bleeds to the edges of the screens; the structural lines provide the breathing room.
*   **Command-Translatable:** The grid itself is a projection of `SET-WORKSPACE <module> --focus <entity_id>`. 

### 3. PANEL ANATOMY (EARNED CHROME)
Zero drop-shadows or heavy blurs. Hierarchy is achieved through structural recesses, contrast, and hairlines. A panel earns its borders via strict rules:
*   **Header Standard:** `40px` fixed height. Contains Title, Global Density Toggle, and Quick-Dash icons.
*   **The Border/Elevation Rule:**
    *   **Nothing (Base Layer):** Unselected, passive data (e.g., background workspace, unselected rows). Flat greyscale (`--gray-950`).
    *   **Hairline (Content Boundary):** `1px` solid `--gray-800`. Applied to all standard module boundaries and table rows.
    *   **Elevation (Focus State):** No shadows. Focus is achieved via a `2px` inset border on the left axis (`--gray-600`), combined with a subtle background lift (`--gray-900`). The UI mimics a physical recessed switch.

### 4. DENSITY-TIER BEHAVIOR (VARIABLE SWAPS)
Driven purely by CSS Custom Properties bound to the `<body>` class. No DOM re-rendering required.
*   **Variables:** `--row-height`, `--cell-padding-x`, `--cell-padding-y`, `--font-size-data`, `--global-gap`.
*   **Narrative (Tier 1):** `--row-height: 48px`, `--global-gap: 16px`, `--font-size-data: 15px`. Breathing room. Casual altitude.
*   **Tactical (Tier 2 - DEFAULT):** `--row-height: 32px`, `--global-gap: 8px`, `--font-size-data: 13px`. The combat baseline. Dense but readable.
*   **Matrix (Tier 3):** `--row-height: 22px`, `--global-gap: 4px`, `--font-size-data: 12px`. Pure signal. Data bleeds together; row hover highlights the entire width to track data across columns.

### 5. M1 ASSET RANKINGS TREATMENT
*   **Header Band:** 40px height. Inline title "ASSET RANKINGS". View Toggles (Global / Franchise / Cap) are pill-segments flush-right.
*   **Filter Placement:** A `36px` sticky sub-header directly below the header band. Contains the Pos Filter chips. Remains pinned during scroll.
*   **Visual Dominance:** "Adjusted Score" and "Base" dominate. They are right-aligned, tabular-nums, `--gray-50` (brightest). 
*   **Recession:** "Franchise" and "Pos" are text-muted `--gray-500`. "Salary" is `--gray-400`. 

### 6. HOME 2×2 GRID RELATIONSHIP
A **deliberate structural break**. 
*   The dense modules use hairlines and fluid gradients. The Home 2x2 Grid uses **hard physical boundaries** (8px gap, full 16:9 ratio cards). 
*   *Why:* Cognitive shifting. Home is "Basecamp" (orientation, low cognitive load). Modules are "The Field" (targeting, high cognitive load). Home uses larger geometry (Card padding: `24px`) to visually slow the user down before entering the dense Tactical environments.

### 7. QUICK-DASH GEOMETRY (CALENDAR + COMMS)
Both must be summonable from anywhere, instantly.
*   **Comms Panel (Locked 48px idle):** Anchored to the **Right Edge** of the viewport, floating *over* the Inspector. When idle, it is a 48px vertical strip. When summoned (Quick-Dash), it expands to `320px` width as a true overlay, casting a `1px` left hairline (`--gray-700`). It pushes no data; it overlays.
*   **Calendar (The Invention):** Anchored to the **Top Edge**. 
    *   *Idle:* A `32px` top header strip (contains Month/Week indicator + global density toggle). 
    *   *Summoned:* Expands downward as a `280px` horizontal blade spanning the width of the fluid workspace. It overlays the top 280px of the workspace. 
    *   *Coexistence:* Comms is Right-Edge vertical. Calendar is Top-Edge horizontal. They meet at the top-right corner without colliding. Both are pure CSS `transform: translate` overlays (zero heavy paint, zero layout shift).

### 8. LEAGUE-SWITCHER SEAM
A top-down dropdown fails at 25 leagues. We design a **Spatial Portfolio Seam**.
*   **Location:** The bottom `64px` of the Nav Rail (Locked). 
*   **Behavior:** 1 League = Full block logo. 3 Leagues = Stack of 3 interactive mini-logos. 25 Leagues = `--gray-900` contained list with native momentum scroll. 
*   **The Geometric Seam:** Above this 64px block, the Nav Rail modules (M1, M2, Trade, etc.) live in a `1fr` fluid space. The Portfolio Seam is structurally isolated at `height: 64px; flex-shrink: 0`. In Phase 3, a "Portfolio Summary" module can be injected *above* this seam without breaking the active league navigation below it.

---

### RIPCORD ITEMS
**None.** 
The locked context (Comms as a 48px overlay, the 4-column existence, and the 3-tier density model via CSS vars) perfectly aligns with the hardware-first, performance-bound constraints of the Wails/WebKitGTK floor. The density gradient approach maximizes the "Anduril" aesthetic by making the data itself act as the structural load-bearer.
