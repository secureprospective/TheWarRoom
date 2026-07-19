# THE WAR ROOM — UNIFIED SPATIAL SPEC (SESSION A)

**Tone:** Anduril. A command console that costs more than a house. Dark, precise, data-dense. Hardware actuation, zero waste.

---

### 1. BASE UNIT + SPACING SCALE
*Confirmed. The grid is rigid.*
*   **Base Unit:** `4px`. No `2px`, `6px`, or `10px` values permitted anywhere in the system.
*   **Scale:** `4 > 8 > 12 > 16 > 24 > 32 > 48 > 64`.
*   **Elevation:** NO dropshadows. NO blurs. Depth is strictly optical via 1px hairlines and inset bevels (WebKitGTK cheap-paint floor respected).

### 2. THE FUSED GRID THESIS
**The shell is an unyielding hardware chassis; data is the coolant flowing through it.** 
Zones possess fixed spatial roles, rejecting fluid 12-col drag. The **Nav Rail (Left, 192px)**, **Workspace (Center, fluid)**, and **Inspector (Right, 384px)** are distinct hardware bays. The Inspector does not negotiate width; it slides in/out via `transform: translateX(384px)` as a pure overlay. Inside these rigid bays, Run 2's density gradient applies: data bleeds to the absolute edges of the zone. Negative space is not external margin, but structural isolation—the 1px `gray-800` hairlines and 12px internal gutters are the only breathing room. Data bears its own load.

### 3. PANEL ANATOMY + EARNED BORDER RULE
One reconciled hardware language. Containment, focus, and click-states are purely optical.
*   **Containment:** 1px `gray-800` hairline border on `gray-950` base.
*   **Headers:** 32px rigid module headers (`gray-900` fill, `gray-800` bottom border).
*   **Active/Selected Entity:** The "Recessed Switch". A `2px` LEFT-axis inset border (`gray-400` to `white`) + `gray-900` background lift. 
*   **Actuation (The Depressed Button):** Default state for actionable tiles is a 1px inset top/left highlight (raised). On `:active` (click), it instantly swaps to a 1px bottom/right inset shadow (depressed). Cheap to paint, feels like physical micro-switches.

### 4. DENSITY-TIER VARIABLE TABLE
Tiers deployed via CSS custom properties on the Workspace container. 

| Tier | Variable | Row Height | Primary Use Case |
| :--- | :--- | :--- | :--- |
| **Narrative** | `--density-narrative` | `48px` | Home Screen, Deep Dossiers |
| **Tactical** | `--density-tactical` | `32px` | Standard Data Tables (Default) |
| **Matrix** | `--density-matrix` | `22px` | 32-Team Season View, Macro-scan |

### 5. M1 BOARD TREATMENT
*   **Topography:** 32px Module Header (Title) + 36px Sticky Filter Sub-Header (Pins on scroll). Total fixed overhead: 68px.
*   **Column Dominance:** **Adjusted Score** and **Base Score** dominate the visual hierarchy (slightly wider, brighter text `gray-50`). Supporting context (Vol, XP) recedes (`gray-500`).
*   **Row Anatomy (Tactical):** 32px row height. Hover highlights the *entire* row width across the workspace bleed. Active row gets the LEFT-axis 2px inset border.

### 6. HOME 2×2 (STANDBY MODE)
A deliberate slow-down break. Basecamp vs. The Field.
*   **Geometry:** A symmetric 2×2 quadrant grid filling the Workspace. 
*   **Density:** Forces `--density-narrative` (48px+ scale). 
*   **Fill:** Rich text, external orbital data, and deep narrative metrics replace dense tables. The grid lines vanish; negative space takes over to reset the operator's eye.

### 7. QUICK-DASH GEOMETRY & COEXISTENCE
Motion is strictly `transform` overlays. Pushes nothing, costs zero table rows.

*   **Comms (Right-Edge Seam):** 
    *   *Idle:* 48px vertical strip pinned to the absolute right edge, over the Inspector.
    *   *Summoned:* `transform: translateX(-272px)` → expands to a 320px pure overlay panel on the right.
*   **Calendar (Generous Overlay):**
    *   *Idle Seam:* A 32px TOP strip integrated into the M1 Sub-Header (occupying the right-most 192px).
    *   *Summoned:* Drops down via `transform: translateY(-100%)` to `translateY(0)`. Footprint: `640px` wide, `480px` high, anchored top-right. Generous enough for a future week/month grid.
*   **Coexistence:** Perpendicular, non-colliding. Comms drops vertically down the right edge; Calendar drops from the top-right header. If both are summoned, they form a flush right-angle corner, layering cleanly via `z-index`.

### 8. LEAGUE SWITCHER SEAM
*   **Location:** Top of the Nav Rail (192px width).
*   **Hierarchy:** `[PORTFOLIO SECTOR]` sits at the absolute top (160px max height). `[MODULE SECTOR]` begins immediately below.
*   **Scaling (1→25+ Leagues):** The Portfolio crest acts as a dynamic block. Scales gracefully from a single full-size crest (1 league) to a dense, hardware-actuated grid of micro-crests (25+ leagues) within the 160px bay.
*   **Phase-3 Injection Seam:** The exact `160px` Y-coordinate of the Nav Rail. A 1px `gray-800` horizontal divider. The future cross-league portfolio module injects *directly below* this line, expanding the `[MODULE SECTOR]` downward.

*(No residual uncertainties. All physical constraints and ruling conditions satisfied. Build ready.)*
