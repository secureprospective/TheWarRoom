# TheWarRoom: Spatial Architecture & Grid System (Run 1)

**Design Thesis:** The "Instrument Cluster". TheWarRoom is not a modular dashboard of equal cards. It is a physical piece of hardware. The workspace is a primary gauge; controls are bezels; data is permanent and heavy; negative space is structural isolation, not wasted potential. 

---

## 1. Base Unit + Spacing Scale
- **Base Unit:** `4px` (`0.25rem`). 
- **Scale:** `4 > 8 > 12 > 16 > 24 > 32 > 48 > 64`.
- **Reason:** A 4px base allows for hyper-precise data alignment (hairlines, micro-padding) required for dense matrices, scaling cleanly to standard 4px/8px Tailwind defaults. 
- **Command Mapping:** All spatial layouts are static; no explicit command verbs needed for layout unless resizing panels (`ui.resize target=inspector width=320`).

## 2. Workspace Grid Thesis: The Instrument Cluster
**Thesis:** The workspace is an asymmetric, statically mapped instrument cluster. Real estate is earned by functional necessity, not modular flexibility. 

Unlike SaaS dashboards that use 12-column CSS grids to freely drag-and-drop widgets, TheWarRoom uses a rigid structural matrix. 
- **The Primary Gauge (Data Surface):** The central workspace (M1 Board, Trade Builder) occupies the massive central footprint. It is always the visual anchor.
- **Negative Space (Bezel Space):** In a data-dense console, negative space is not empty; it is the *bezel*. We use 1px to 2px structural hairlines (`--w-border-bezel`) and 16px structural gutters to completely isolate instruments from one another. The background (pure void) acts as the chassis. Data breathes *between* blocks, never *inside* them. 

## 3. Panel Anatomy & Elevation Rules
**Anatomy:**
- **Header Band:** `32px` fixed height at Tactical density (`--w-header-h`). Contains the module name (uppercase, tracked), contextual verbs (filter, view toggle), and read-only state (e.g., "Live"). 
- **Body:** Flexible. Respects a `12px` structural inset (`--w-pad`) at the cardinal edges to isolate the data from the chassis.
- **Footer/Status Bar:** `24px` optional height for aggregation metrics (e.g., "Total Cap: $X / Y Players").

**Elevation vs. Border Rule:**
1. **Borders define containment:** Every distinct module uses a `1px` solid border (`rgba(255,255,255,0.08)`). 
2. **NO DROPSHADOWS.** The WebKitGTK floor forbids heavy paint operations. We do not simulate physical elevation via blur. 
3. **Simulated Bevels (The Actuation Rule):** "Elevation" is simulated via a 1px top/left inset highlight (`rgba(255,255,255,0.04)`). When a control is "actuated" (clicked), the highlight swaps to a 1px bottom/right shadow (`rgba(0,0,0,0.5)`), simulating a depressed mechanical button without filter costs.

## 4. Density-Tier Behavior (CSS Variables Only)
Tiers swap exclusive CSS custom properties on the `:root` or `.density-context` wrapper. No DOM changes.

| Variable | Narrative (Casual) | Tactical (Default) | Matrix (Hardcore) |
| :--- | :--- | :--- | :--- |
| `--w-pad` | 16px | 12px | 8px |
| `--w-row-h` | 48px | 32px | 24px |
| `--w-font-meta` | 14px | 12px | 11px |
| `--w-gap` | 16px | 8px | 4px |

**Progression:** Casuals are dropped into Narrative. They do not toggle settings; they encounter a "Matrix" icon on a specific module, click it, and the grid tightens *visually* to show them deeper data. `view.set_density matrix`

## 5. M1 Asset Rankings Board Treatment
The M1 is the apex primary gauge.
- **Header Band:** Sticky 32px band. Contains: Left = "ASSET RANKINGS" title. Center = View Toggle (Global / By Franchise / Cap Efficiency) as heavy mechanical segmented buttons. Right = Position Filter (RB, WR, QB, etc.) as toggle pills.
- **Row Treatment (Tactical - 32px):** 
  - **Rank:** 24px wide column. Monospaced, muted gray.
  - **Player/Pos:** Left-aligned, `12px` bold white. Dominant visual.
  - **Franchise:** `12px` muted gray.
  - **Adjusted Score:** Right-aligned. **Dominant Data Point.** `16px` bold, tabular-nums. This is the "speedometer" reading.
  - **Base/Salary:** `12px` right-aligned, muted gray. Recedes until hovered.
- **Command Mapping:** `m1.sort target=adjusted_score dir=desc`, `m1.filter pos=rb`

## 6. Home 2×2 Grid Relationship
**A deliberate, temporary break.**
The Home 2×2 grid is the "Standby Mode". The instrument cluster is powered down. 
- It uses the same chassis (4px base, structural bezels, heavy typography) but the layout shifts from an asymmetric cluster to a symmetric quadrant to establish stability. 
- The cards are significantly larger, utilizing the Narrative density tier. They are *launchpads* and *seasonal alerts*, not continuous data feeds.
- **Why?** A casual user entering the app needs clear, isolated options. Once a module (M1) is engaged, the shell snaps from Standby (2x2) to Instrument Cluster (dense table + rails). 

## 7. Quick-Dash Geometry (Calendar & Comms)
**Locked Context Honored:** The app defaults to a 3-column physical layout: Nav Rail (200px) | Fluid Workspace | Contextual Inspector (280px). 

- **Comms Panel (48px Strip):** Pinned to the absolute bottom of the screen, spanning the width of the Workspace + Inspector. It is a 48px high status bar ("Command Console: Standby"). 
  - *Summon:* Transforms upward (`translateY`) as a `320px` high footer overlay. It overlays the bottom of the Workspace and Inspector, preserving their layout above the 320px threshold. `ui.summon target=comms`
- **Calendar (Invented Geometry):** Idle state is a `48x48px` Date Stamp button pinned to the top-right of the Workspace header. 
  - *Summon:* Expands downward (`scaleY`) and left as a `320px` square Pop-Over overlay anchored to that top-right corner. It overlays the data below it without shifting the grid. `ui.summon target=calendar`
- **Coexistence:** If both are summoned, the Calendar occupies the top-right corner, Comms occupies the bottom edge. They do not intersect.

## 8. League-Switcher Seam
The Nav Rail must scale gracefully without UI reconfiguration.
- **1–4 Leagues:** A vertical list of 48x48px league crests at the top of the Nav Rail. Active league has a 2px left bezel indicator.
- **5–25+ Leagues:** The list becomes a searchable, scrollable command-rail. A "PORTFOLIO OVERVIEW" crest is permanently pinned at the very top, distinct from individual leagues.
- **The Geometric Seam:** The top `160px` of the Nav Rail is explicitly partitioned as `[PORTFOLIO SECTOR]`. Below it is `[MODULE SECTOR]`. Currently, selecting a league dynamically scopes the modules below it. 
- **Future Phase 3:** The seam allows for a Portfolio module to be injected into the `[MODULE SECTOR]`, allowing cross-league aggregation dashboards without redesigning the Nav Rail.

---

## RIPCORD ITEMS

1. **Comms Panel Idle State (48px Strip)**
   - **Objection:** A 48px strip permanently docked at the bottom of the screen consumes premium vertical real estate (which is critical for 32-team data tables). On a 1080p monitor, this strips ~4.4% of vertical height.
   - **Alternative:** Allow the 48px strip to collapse entirely into a 32x32px icon in the Nav Rail or Global Header when totally idle, returning that real estate to the primary gauge. 
   - **Status:** Flagged. Design currently adheres to the locked 48px strip baseline.
