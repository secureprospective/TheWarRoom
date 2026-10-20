# THE WAR ROOM — SESSION B: COMPONENT HIERARCHY & TYPOGRAPHY

## 1. TYPE SCALE & NUMERICS (The Placard System)
Hierarchy is carried by weight, tracking, and optical alignment—not size. Size remains compressed to honor the 4px base unit.
*   **Hero Numeric (Adjusted Score, Cap Space):** 24px, JetBrains Mono, Weight 700. Tabular figures (`font-variant-numeric: tabular-nums`). Greyscale `gray-50`.
*   **Primary Data (Player Name, Franchise):** 14px, Inter, Weight 500. `gray-200`.
*   **Secondary Data (Pos, Base Score):** 13px, Inter/Mono, Weight 400. `gray-400`.
*   **Placard Labels (Headers, Inspector Labels):** 11px, Inter, Weight 600. Uppercase. Tracking `0.08em`. `gray-500`.
*   **Matrix Density Override:** 13px Primary, 11px Secondary, 9px Placard Labels. 
*   **Numbers Rule:** All numeric columns use right-alignment and tabular-figures. Adjusted Score dominates via 14px/700 weight; Salary recedes via 12px/400 weight.

## 2. TABLE & ROW ANATOMY (M1 Asset Rankings)
The aerospace placard concept is executed via positional memory. Headers are fixed legend plates; the data rows are the readouts.
*   **Glance (Casual):** Scans the left-heavy columns. Sees Rank, Player Name, and Adjusted Score (bright `gray-50`). The rest is muted.
*   **Operate (Tactical):** Scans all columns. Reads Base vs Adjusted to spot deltas. Clicks a row to inspect.
*   **Interrogate (Matrix):** Hides non-essential text, relies on spatial alignment. Hover reveals a 1px tooltip placard.
*   **Row Structure:** 32px height (Tactical). Left padding 16px, right padding 16px. 12px gutters between clustered data.
*   **Actuation States (Extending Session A):**
    *   *Rest:* Transparent bg, 1px transparent bottom border.
    *   *Hover:* `bg-gray-900/50` (full row bleed), 1px top/bottom `gray-700` inset bevel to feel "raised".
    *   *Active:* 2px left inset border `gray-50`. Background `gray-900`. 
    *   *Pressed:* Inverted bevel (1px bottom/right inset). Acknowledges in <100ms.

## 3. THE M1 & M2 COLUMN SPEC
**M1 Asset Board (7 Cols):**
1.  **Rank:** 32px width. Mono. Center aligned. Header: "RNK".
2.  **Player:** Fluid (min 120px). Inter. Left aligned. Header: "ASSET".
3.  **Pos:** 40px width. Inter. Center. Header: "POS".
4.  **Franchise:** 100px. Inter. Left. Header: "FRANCHISE".
5.  **Base Score:** 80px. Mono. Right. Header: "BASE".
6.  **Adjusted Score:** 90px. Mono. Right. Header: "ADJ". *Sortable (Default)*.
7.  **Salary:** 80px. Mono. Right. Header: "SAL". *Sortable*.

*Density Shifts:* Matrix (22px) drops Base Score & Franchise. Narrative (48px) expands Player column to include a secondary 11px line ("Bye: 7 | Tag: Franchise").

**M2 Power Rankings Board (32 rows):**
*   Cols: Rank | Franchise | Power Score (Mono, 14px) | All-Play% (Mono) | PA (Mono) | Weight Slider (64px interactive track).
*   Sortable headers on Rank, Power Score, All-Play%.

## 4. CONTEXTUAL INSPECTOR ANATOMY (320px)
Slides in via `translateX`. No blur, no drop-shadow. 1px left border `gray-800`.
*   **Header (Placard Plate):** 11px Uppercase Label ("ASSET DOSSIER"). 24px Player Name. 24px Adjusted Score.
*   **Layer Breakdown (The 6 Layers):** Vertical list. 
    *   Label (11px, left) | Value (11px Mono, right).
    *   Track: 4px height, `gray-800` bg, 4px inner padding.
    *   Fill: `gray-400` bg. 
*   **Contract Block:** Grid layout. 2x2. 
    *   Labels: "YEARS", "GUARANTEE". Values in 16px Mono.
*   **Notes Block:** 13px Inter, `gray-400`. Top 1px hairline border separating it from mechanical data above.

## 5. CARD ANATOMY (Home & Seasonal)
Differs from tables by introducing structural grouping. Base unit 4px. 
*   **Home Card (2x2):** 1px `gray-800` border. Inset top/left 1px `gray-700` highlight (raised tile). 
    *   Header: 11px Label ("LEAGUE 01: MOTOR CITY").
    *   Metric: 24px Mono Score. 
    *   Footer: 11px Trend data.
*   **Seasonal Card:** Visually identical to Home Card, but features a 4px left-axis border indicating calendar state (Amber for active, Gray-600 for completed).

## 6. CONTROLS & FORM LANGUAGE
Micro-switch actuation. Hard rule: <100ms pressed feedback. 
*   **Text Inputs:** `bg-gray-900`, 1px `gray-700` border. `:focus` -> 1px `gray-400` border, no glow.
*   **Primary Button (Confirm/Execute):**
    *   Rest: `gray-800` bg, 1px top/left inset highlight.
    *   Hover: `gray-700` bg.
    *   Pressed: 1px bottom/right inset highlight, `translate-y-px` (0.5px shift). Fires instantly.
    *   Disabled: `gray-900` bg, `gray-700` text.
*   **Chips (Filters):** Same bevel logic. Active state holds inverted bevel (depressed).
*   **M2 Weight Slider:** 64px track, 8px wide thumb. Thumb rests on track; on drag, thumb shifts 1px down (inverted bevel) to simulate physical push.

## 7. CONFIRMMODAL (Arm -> Fire)
No theatrical spinners. The commit gate feels like arming a warhead.
*   **Anatomy:** Fixed center overlay, 480px wide. `gray-950` bg. 1px `gray-700` border. 
*   **Preview Area:** Static table of dollar breakdowns or rejection reasons.
*   **Actuation Path (The Command Ledger projection):**
    1.  *Arm:* User clicks "ARM". Button state locks to pressed/inverted bevel. UI registers intent.
    2.  *Fire:* User clicks "FIRE" (a secondary button that appears at 16px right margin). 
*   *No Command Equivalent Flag:* The double-click requirement breaks the LLM chat ledger. Re-mapped: Clicking "ARM" executes `transaction.arm`, clicking "FIRE" executes `transaction.commit`.

## 8. DATA STATES (Honest & Fast)
No spinners. Emptiness is designed, not accidental.
*   **Fresh Install:** Static engraved wireframe of the UI. 11px Placard Labels describing what data goes where ("M1 ASSET RANKINGS - AWAITING LEAGUE IMPORT").
*   **Fetch In-Flight:** Skeleton blocks (`bg-gray-800`, pulsing opacity 1.0 to 0.8 over 800ms via CSS transform/opacity only).
*   **MFL Unreachable:** Solid red 2px top border on the workspace. Header reads: "MFL UNREACHABLE. SERVING LAST CACHE [TIMESTAMP]." Data remains fully interactive.
*   **Seasonal Emptiness:** No red. Gray-500 text in primary workspace: "OFFSEASON. NO MATCHUP DATA." Calendar card left-border is `gray-700`.

## 9. KEYBOARD MAP (Command Ledger Floor)
Every GUI action is a chat verb.

| Key | Action | Ledger Verb | Density Scope |
| :--- | :--- | :--- | :--- |
| `J` / `K` | Navigate down/up | `nav.next` / `nav.prev` | Matrix / Tactical |
| `Enter` | Open Inspector | `inspect.entity` | Matrix / Tactical |
| `T` | Add to Trade Builder | `trade.add` | Matrix / Tactical |
| `/` | Focus Search | `search.focus` | All |
| `Esc` | Close Inspector/Modal | `nav.close` | All |

## 10. EDGE-RESIZE HANDLE
Strict optical mechanics. 4px wide invisible hit area.
*   **Idle:** 1px `gray-800` line.
*   **Hover:** 1px `gray-600` line. Cursor: `col-resize`.
*   **Dragging:** 1px `gray-400` dashed ghost line. The panel width does *not* update live; the ghost simply maps the new boundary (preserves APU paint budget).
*   **Released:** Panel width snaps to ghost position via CSS transition (`transition-width 150ms ease-out`).

***

## RIPCORD ITEMS
*   **Objection to "AEROSPACE PLACARD" at Matrix Density:** The provocation demands everything wear its name like a mil-spec legend plate. However, at Matrix density (22px rows), inline labels ("SAL", "ADJ") destroy the scan-ability of the 500+ rostered players and shatter the 4px spatial rhythm. 
*   **The Alternative (Confirmed in Design):** I have relegated permanent text labels strictly to the 36px Sticky Sub-Headers and Inspector Panels. The rows themselves act as the readouts, relying on strict positional memory and right-alignment for instant optical parsing. Labels are only shown on `hover` or in `Narrative` density.
