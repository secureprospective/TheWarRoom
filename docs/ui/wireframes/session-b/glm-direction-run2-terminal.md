# THE WAR ROOM · SESSION B · COMPONENT HIERARCHY & TYPOGRAPHY

## 1. Type Scale & Numeric Treatment
Terminal-forward hierarchy. Type does the heavy lifting; containers vanish.

*   **Display (Scores):** 24px / Inter / Weight 800 / `font-variant-numeric: tabular-nums`.
*   **Headers:** 16px / Inter / Weight 600.
*   **Body (Names/Labels):** 12px / Inter / Weight 400 (Normal) & 500 (Focus).
*   **Micro (Column/Section Labels):** 11px / Inter / Weight 500 / `letter-spacing: 0.04em` / Uppercase.
*   **Data/Mono (Ranks/Stats/Salary):** 11px - 12px / JetBrains Mono / Weight 400 / `tabular-nums`.

**Numeric Rule:** All numbers right-align. The Adjusted Score uses Display (24px), dominating the row. Base Score and Salary use Mono 12px in gray-500. A positive delta is Weight 600; negative is Weight 400.

## 2. Table & Row Anatomy (The Terminal Grid)
No gridlines. Rows are simply lines of type separated by optical spacing and baseline alignment.

*   **Structure:** Pure CSS grid per row. `grid-template-columns` mapped per density tier.
*   **Glance (Casual):** Massive 24px Adjusted Score + 12px Player Name. Everything else fades to gray-500. The eye instantly finds the highest number.
*   **Operate (Tactical):** The grid materializes. Rank, Pos, Base Score, Salary become readable columns.
*   **Interrogate (Expert):** Hover highlights the row (bg-gray-900). Click actuates `select.target` -> Inspector populates.
*   **Actuation Language:** 
    *   *Hover:* Full row bg transitions to gray-900 (<50ms).
    *   *Active/Selected:* 2px LEFT-axis inset border (gray-50), committing to the locked Session A spec. 

## 3. Deferred Column Specs (M1, M2, Roster)
**M1 Asset Rankings (Grid Template):**
*   *Matrix (22px):* `[Rank: 32px] [Player: 1fr] [Pos: 40px] [Adj Score: 56px] [Base Score: 56px] [Salary: 56px]`
*   *Tactical (32px):* + Franchise (96px) inserted after Pos.
*   *Narrative (48px):* Player Name (16px) + Pos (16px), drops Base Score & Salary.
*   *Sortable:* Rank, Adj Score, Base Score, Salary. Sort indicator is purely typographic: `▼` (gray-50) next to active header, `¦` (gray-800) on inactive.

**M2 Power Rankings:**
*   Cols: `[Franchise: 1fr] [Power Score: 80px] [All-Play %: 80px] [PA: 64px] [Weight Slider: 120px]`

**Transaction Roster Table:**
*   Cols: `[Player: 1fr] [Pos: 40px] [Salary: 64px] [Action: 80px]` (Action is a text-verb: e.g., `CUT`, `TRADE`).

## 4. Contextual Inspector (320px)
Populates on `select.target`. No boxes, just typographic blocks divided by 12px space.

*   **Header:** 11px Micro Label (POS · AGE · TEAM) -> 16px Player Name -> 24px Adjusted Score (gray-50) + 12px Mono Base Score (gray-500) underneath.
*   **Layer Breakdown:** 6 horizontal bars. Height: 4px. Background: gray-800. Fill: gray-100. No rounded corners. Label (11px L) + Value (11px Mono R) sit directly above the bar.
*   **Contract Block:** Monospace terminal output style. 
    `SALARY: $42.10`
    `YEARS: 3 (Through 2026)`
*   **Notes:** 12px Inter, raw text input borderless. Blinks cursor when active.

## 5. Card Anatomy (Home & Seasonal)
Differentiates from tables via spatial grouping, not containers. Zero borders, zero shadows.

*   **Home 2x2 Grid:** 24px gap. 
    *   Title: 11px Micro (gray-500).
    *   Metric: 24px Display (gray-50).
    *   Context: 12px Mono (gray-400).
    *   *Difference:* Cards stack vertically (Title -> Metric -> Context). Rows lay horizontally.
*   **Seasonal Cards (Calendar Summons):** Expands the grid. 11px Micro Dates -> 16px Matchups. Empty weeks are simply absent of type (true zero).

## 6. Buttons & Form Controls (Text that Actuates)
Chrome is stripped. Controls are verbs. Pressed acknowledgment is a HARD rule (<100ms, immediate DOM paint).

*   **Text Button / Verb (e.g., EXECUTE, FILTER):**
    *   *Rest:* 11px Micro Uppercase, gray-400.
    *   *Hover:* gray-50. 1px top/left inset border appears (gray-700).
    *   *Pressed:* 1px bottom/right inset border (gray-900) swaps instantly. Text shifts +1px down/right. 
    *   *Disabled:* gray-700, no pointer events.
*   **Position Chips (QB/RB/WR/TE):**
    *   *Rest:* 11px, gray-500, 4px padding.
    *   *Active:* Weight 600, gray-50. 1px bottom border (gray-100) acts as an underline.
*   **Weight Slider (M2):**
    *   *Track:* 1px gray-800 line.
    *   *Thumb:* 8px wide, 2px tall gray-400 block. Hover: gray-100. Dragging: expands to 12px wide, 4px tall.
*   **Text Inputs:** Borderless. Bottom 1px border (gray-700). Active focus drops bottom border to gray-100.

## 7. ConfirmModal (The Commit Gate)
The exception to zero-chrome. This MUST feel physical. It is a mechanical relay.

*   **Container:** 320px wide, right-aligned overlay. Solid gray-950 background. 1px solid gray-50 hairline border (pure optical elevation). No backdrop blur (WebKitGTK floor). Workspace dims to opacity 0.5.
*   **Anatomy:** 
    1.  **Verb:** 16px Weight 600 (e.g., "CONFIRM TRADE").
    2.  **Preview Breakdown:** Pure Mono text. 
        `SEND: $42.10, 2025 1st`
        `RECV: RB J. Gibbs`
*   **Arm -> Fire Path:**
    *   *State 1 (Armed):* "HOLD TO FIRE" button. 1px top/left inset (gray-700).
    *   *State 2 (Firing - mousedown):* Swaps instantly to 1px bottom/right inset (gray-900). Text changes to "FIRING".
    *   *State 3 (Commit):* DOM releases modal, executes command. If rejected by engine, modal holds bottom/right inset, text turns to "REJECTED: CAP EXCEEDED" (Weight 800).

## 8. Data States (Honest Surfaces)
*   **Fresh Install:** 24px Display "AWAITING LEAGUE IMPORT". 12px Mono `cursor.select file.league`.
*   **Fetch in Flight:** Skeletons are NOT spinners. Skeletons are pulsing 4px gray-800 blocks replacing data type. (e.g., a gray block where the 24px score goes).
*   **MFL Unreachable:** Sticky 32px top sub-header turns to gray-50 Weight 600: `STATES: CACHE ONLY (MFL UNREACHABLE)`. Data remains fully interactive from local cache.
*   **Offseason Emptiness:** 16px Inter Weight 400 gray-500: `OFFSEASON STATE`. No skeletons, no errors. Pure clean type.

## 9. Global Keyboard Map (Command Ledger)
Keys map directly to Guix-style terminal verbs. 
*   **Matrix Floor (Hardcore):**
    *   `J` / `K`: `cursor.move` (down/up)
    *   `Enter`: `select.target`
    *   `T`: `trade.init`
    *   `F`: `filter.toggle`
    *   `1-6`: `view.density` (1=Narrative, 2=Tactical, 3=Matrix)
*   **Tactical Floor:** J/K, Enter active. T/F require `Shift+T`.
*   **Narrative Floor:** J/K disabled. Up/Down arrows, Mouse Click only.

## 10. Edge-Resize Handle
Strict adherence to 4px base grid. Invisible until needed.
*   **Idle:** 1px column at x:192 / x:fluid. Pure workspace color (gray-950).
*   **Hover:** Expands to 4px column. Background shifts to gray-900. Cursor: `col-resize`.
*   **Dragging:** 1px ghost guide (gray-600) renders at the target X coordinate. The actual panels DO NOT REFLOW until mouseup (Commit).
*   **Commit (Release):** Guide line snaps away. CSS custom properties (--nav-width, --workspace-width) update simultaneously. <100ms transition.

---

## RIPCORD ITEMS
*   **Objection 1: The "Terminal-Forward" Provocation vs. The Casual's First 5 Minutes.**
    *   *The Decision:* Zero-chrome, type-only hierarchy.
    *   *The Objection:* A 1-2 league casual opening this app for the first time will see a blank screen with massive numbers and micro-labels. Without containers, spatial grouping must do the heavy lifting. If the 12px/24px grid spacing is slightly off, the UI doesn't just look messy—it becomes completely unreadable, like a broken terminal log. 
    *   *The Alternative:* Allow a single subtle background fill (`bg-gray-900/50`) for major data clusters (Inspector blocks, Modal) without introducing 1px borders. This provides optical anchoring (a "surface") without violating the cheap-paint/blur constraint or adding heavy chrome. (I held to pure zero-chrome in this doc as instructed, but this is the primary failure point).
