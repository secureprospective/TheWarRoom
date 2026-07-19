# THEWARROOM: UNIFIED COMPONENT LANGUAGE SPEC

**1. TYPE SCALE & NUMERICS**
*Inter is text. JetBrains Mono is data. Greyscale palette: gray-0 (black) to gray-50 (white).*
* **Hero Numeric:** 24px JetBrains Mono, weight 700, `tabular-nums`, right-aligned, gray-50. (Matrix override: 16px).
* **Sub-Header (Sticky):** 12px Inter, weight 700, tracking `0.05em`, uppercase, gray-400. (Height: 36px).
* **Row Data (Standard):** 13px Inter (text) / 13px JetBrains Mono (numerics), `tabular-nums`, right-aligned, gray-100. (Matrix override: 11px, line-height 1.1).
* **Delta Numeric:** 13px JetBrains Mono. Positive delta: weight 600. Negative delta: weight 400. (Greyscale-honest signal; Session C will evaluate hue reinforcement).
* **Matrix Density (22px row height):** Overrides standard scale to 11px Inter/Mono, line-height tight.

**2. TABLE & ROW ANATOMY**
* **Rule (Ruling 2):** No vertical gridlines. Separation is strictly optical spacing (24px padding) and row-hover background bleed.
* **Reads:** 
  * *Glance:* Positional memory (Rank → Player → Pos → Age → Value).
  * *Operate:* Row hover exposes right-aligned action icons (e.g., Trade Block).
  * *Interrogate:* Click row targets Inspector.
* **States (Inset-Bevel Language):**
  * *Rest:* Transparent background, gray-100 text. 
  * *Hover:* Background swaps to gray-900. 1px top/left inset (raised) applied to the entire row container to signal focus. Labels appear if hidden.
  * *Active/Pressed:* Background gray-850. 1px bottom/right inset (pressed). 

**3. M1 / M2 / ROSTER COLUMN SPECS**
*Alignment defaults: Left for text, Right for all numerics.*
* **M1 Asset Rankings:**
  * *Narrative:* `grid-template-columns: 40px 1fr 60px 50px 120px 40px;` (Rank, Player, Pos, Age, Val, Action).
  * *Matrix:* `grid-template-columns: 24px 1fr 40px 40px 80px;` (Drops Action, tightens gaps).
* **M2 Depth (Roster):** `grid-template-columns: 40px 1fr 60px 80px 1fr;` (Depth, Player, Pos, Status, Notes).
* **Sort Indicator (Ruling 5):** Active: `▼` or `▲` in gray-50 next to header. Inactive: `¦` in gray-800. No icon chrome.

**4. CONTEXTUAL INSPECTOR (320px)**
* **Container:** Optical containment via a 1px left hairline (gray-800). No box-per-field.
* **Header:** Player Name (14px Inter, gray-50), Pos/Team. Dominated by 24px Hero Numeric (Engine Score).
* **Layer Breakdown:** 6 horizontal bars, 4px height. Track: gray-800. Fill: gray-100. 
* **Contract Block (Ruling 7):** Terminal-output style.
  * `SALARY: $42.10`
  * `YEARS: 3 (through 2026)`
* **Altitudes:** Floats over tables on widescreen; pushes grid on standard; full-screen modal on narrow.

**5. CARD ANATOMY**
* **Home (2×2):** Heavy machined container. 1px top/left inset (raised).
* **Seasonal Card:** Same bevel language, wider aspect ratio.
* **The Rule:** A *Card* represents an aggregate entity summary (boxed containment). A *Row* is a flat, comparative state in a continuous list.

**6. BUTTONS & FORM CONTROLS**
*Hard Rule: All controls must provide <100ms pressed acknowledgment via transform/inset swap.*
* **Buttons:** Rest = 1px top/left inset (raised). Hover = gray-800 bg. Pressed = 1px bottom/right inset (pressed). Disabled = opacity 0.4, cursor not-allowed.
* **Chips/Inputs/Selects:** Rest = 1px inset border (gray-700). Focus = border gray-400, no glow.
* **M2 Weight Slider:** 2px track (gray-800), 8px handle (gray-100). Dragging = handle scales to 10px + gray-50.

**7. THE CONFIRMMODAL (Ledger Verb: `txn.commit`)**
* **Geometry:** 480px CENTERED overlay. Workspace dims to opacity 0.5. NO backdrop blur.
* **Anatomy:** Terminal-preview block (mono text, breakdown of $ or assets) sits above the action button.
* **State Path (Ruling 6):**
  * *Rest:* Button text "HOLD TO FIRE". 1px top/left inset (armed/raised).
  * *Mousedown:* Instantly swaps to 1px bottom/right inset + text "FIRING" + progress fill (≤600ms, transform scale-x only).
  * *Release:* Before complete = Cancels. At complete = Commits.
  * *Engine Reject:* Modal HOLDS (non-dismissable). Text changes to weight 800 "REJECTED: <reason>".

**8. EMPTY / LOADING / ERROR STATES**
* **Fresh Install:** Engraved-wireframe-of-the-UI. 11px placard labels: "M1 ASSET RANKINGS — AWAITING LEAGUE IMPORT".
* **Loading:** Pulsing gray-800 blocks shaped exactly like the data (opacity 1.0→0.8, 800ms interval). NO spinners.
* **MFL Unreachable:** Sticky sub-header state line: "STATE: CACHE ONLY (MFL UNREACHABLE) · as of <ts>" rendered in weight 600 text with a 2px top edge. Data remains fully interactive.
* **Offseason:** Clean gray-500 text: "OFFSEASON — NO MATCHUP DATA". Explicitly no edge, no skeleton, not an error.

**9. GLOBAL KEYBOARD MAP**
| Key | Action | Command Ledger Verb | Density Scope |
| :--- | :--- | :--- | :--- |
| `J` / `K` | Move selection down/up | `nav.down` / `nav.up` | Matrix, Tactical |
| `Enter` | Open Inspector / Interrogate | `view.focus` | Matrix, Tactical |
| `T` | Initiate Trade / Send to Block | `txn.create` | Matrix, Tactical |
| `1` / `2` / `3` | Toggle Density (Matrix/Tac/Nar) | `ui.density` | Global |
| `/` | Focus Search | `search.focus` | Global |
| `Esc` | Close modal / Clear input | `ui.escape` | Global |

**10. EDGE-RESIZE HANDLE**
*Commit-on-RELEASE, no live reflow.*
1. **Idle:** 4px invisible edge boundary. Cursor changes to `col-resize` on hover.
2. **Hover:** 1px gray-600 hairline appears on the edge boundary.
3. **Dragging:** A 1px ghost line (gray-400) renders at the target pixel width.
4. **Released-Commit:** The grid snaps to the ghost line (`ui.resize`). The ghost line fades.

***

**RESIDUAL CHOICES**
*   **Focus Rings:** Clarifying that active keyboard focus uses a 1px gray-400 inset ring rather than an outline, preserving the machined aesthetic without breaking dense rows.
*   **Hover Scrolling:** Matrix density hover-bleed will pause during scroll velocity to prevent visual flashing of the raised inset state.
*   **Inspector Scrolling:** Inspector layer-breakdown bars will adopt a faint `gray-900` glow on hover to indicate deeper drill-downs (M2 features) without adding extra chrome.
