# SESSION D: THE WATCH FLOOR — ONE RIVER OF TIME

**DESIGN DIRECTION · v1.0**

The command console does not have "apps." It has a single chronological substance—the **Timeline**—and three lenses for viewing it. The Feed is the river flowing toward NOW. The Calendar is the topographic map of the river ahead. The Comms thread is a conversation pinned to a specific bend in the water. We design the bedrock grammar of this timeline, ensuring the 25-league operator sees a whitewater torrent, while the 1-league casual sees a calm brook, using the exact same data structures.

---

## 1. THE UNIFIED EVENT GRAMMAR (THE RIVERBED)

Every occurrence in TheWarRoom—a chat message, a calendar deadline, a live trade, a snipe—is a **Timeline Event Object (TEO)**. It renders via one unified grammar. 

### Anatomy of an Event Row (Height: 48px Narrative / 32px Tactical / 22px Matrix)
Every TEO shares this left-to-right anatomy:
*   **Severity Spine (2px left edge):** Achromatic `--edge-selection` by default. Hue is EARNED only via escalation.
*   **Timestamp (Mono 11px):** `HH:MM` or relative (`-12m`). Past events are `--text-muted`.
*   **Subject (Inter 13px/600):** The primary noun (Player, Franchise, League).
*   **Predicate (Inter 13px/400):** The verb (was dropped, offered trade, cleared waivers).
*   **Action Affordance (Right-aligned):** Optional. A hairline-bordered micro-switch (e.g., `[ ADD ]`).

### Escalation & Recession (The Current)
We do not use modals to interrupt. The river's current dictates visibility.
*   **Standard (The Calm Broook):** Spine is `--text-muted`. Predicate reads `--text-secondary`. 
*   **Escalation (Imminent/Danger):** A bid clock crossing 1hr, or a snipe.
    *   *Color:* Timestamp escalates to `--signal-amber-loud` (<1hr) or `--signal-red-loud` (snipe).
    *   *Structure:* The Severity Spine ignites with the matched semantic hue. Row background optically raises via `--surface-raised` (11% L) to break the plane.
*   **Recession (Resolution):** Upon expiry or action (`txn.commit`), the TEO does not disappear; it sinks.
    *   *Motion:* 150ms fade. Background returns to `--surface-canvas`. Spine reverts to `--edge-selection`. 
    *   *Debt-free:* Resolved events sink to the bottom of the Feed view but remain in the Calendar's historical log. No "unread" badge anxiety.

### Cross-League Aggregation Seam (Phase-3 Flag)
The operator's 25-league torrent requires a filter. 
*   **Mechanism:** A `ui.tributary` toggle in the Feed header. 
*   **v1:** Renders single-league timeline perfectly. 
*   **v3 Seam:** When engaged, TEAs group by League (Subject) rather than strict chronological order, rendering as expandable "Tributary" rows (e.g., `[+4] UFL - West`).

---

## 2. CHAT AS THE TERMINAL'S VISUAL HOME

The 320px right-edge Comms overlay (`ui.summon target=comms`) is the visual birthplace of the future LLM terminal. Messages and Action Cards share one scannable, non-bubble thread.

### Thread Grammar (Full width, 4px hairline separators)
*   **Human/Chatter:** `12:04 [User]: ` Inter 13px `--text-primary`. 
*   **Command Input:** `12:05 > /offer` Mono 13px `--text-secondary`.
*   **System Answer:** `12:05 ↳ Queued for review.` Mono 12px `--text-muted`. *No assistant persona. Just raw data response.*
*   **Action Card (The `/offer` Trade):** A full-width inset tile (`--surface-raised`). Contains Player A for Player B. 
    *   *Controls:* `[ ACCEPT ]` `[ COUNTER ]` `[ DECLINE ]`. 
    *   *Actuation:* These are not bubble buttons. They are `--edge-hairline` inset bevels. `:active` presses the micro-switch (1px bottom/right inset). 
    *   *Routing:* All three route through the B-locked `txn.commit` ConfirmModal (480px, Hold-to-Fire ≤600ms). 
    *   *Recession:* On Hold-to-Fire completion, the card sinks (opacity 50%, controls replaced by `[ COMMITTED - 12:06 ]`).

### The Prompt (Bottom Dock)
*   **Anatomy:** 48px high dock, spanning the 320px width. `--surface-sunken`. Inset top border.
*   **Affordance:** Prefix `>` in `--text-muted` mono. Placeholder: `Enter command or message...`.
*   **Actuation:** `Enter` executes via `comms.send`. Focus state rings the dock with `--edge-focus` (blue inset). 

---

## 3. THE CALENDAR (THE MAP)

A top-right generous overlay (`ui.summon target=calendar`). Google Calendar, viewed through a targeting pod. 

### Event-Chip Anatomy
*   **Structure:** Optical elevation only. `--surface-tile` with a 1px inset top/left bevel.
*   **Color/Bar:** A 2px left Severity Spine dictates the categorical hue (Waiver = `--signal-blue-base`, Trade = `--signal-amber-base`, Regular Season = `--text-muted`).
*   **Density:** 
    *   *Month:* Time + Subject only (`14:00 Waivers`). 
    *   *Week:* Time + Subject + Predicate (`14:00 Cut Deadline - Roster lock`).

### Overlap Layout (Commit)
WebKitGTK hates heavy paint. We commit to **Column-Share**. 
If two events overlap at 14:00, the grid column splits 50/50. If >3 events overlap, they render side-by-side at unreadable widths, and the 3rd is replaced by a `+N more` aggregated chip. Clicking it opens a tactical dropdown list.

### Creation & Manipulation (Cheap-Paint Floor)
*   **Creation:** Click-empty → Quick-Create popover (Subject, Time, League). Click-drag-to-create renders a 1px `--edge-hairline` ghost that snaps to a 15-min grid (height = 22px/15min).
*   **Move vs. Resize (The Risk):** Per digest §2.2, drag-move is safe (transform: translateY ghost). Edge-resize triggers layout thrashing on WebKitGTK.
    *   *Solution:* **No native edge-drag resizing.** 
    *   *Fallback:* To change duration, click the event chip → Inspector panel → Edit Duration tactical stepper buttons (`ui.event.resize +/- 15m`). Commit fires `ui.event.commit`.

### Append-Only Honesty
A dragged move commits an `EventSuperseded` appended revision to the backend. 
*   **The Truth Mechanism:** The UI does not lie. When an event is moved, a 32px Snackbar appears at the bottom of the overlay: `Event moved. Revision 2 appended. [ UNDO ]`.
*   **Undo:** Clicking `[ UNDO ]` does not delete the revision; it appends an `EventUndone` event. The snackbar clears via `ui.snackbar.dismiss`.

### Operational Overlay (Deadlines in the Wild)
The Calendar is not the only place deadlines live. Deadlines project onto the surfaces they govern.
*   **Trade Builder:** The ConfirmModal header reads `Trade Deadline: 4d 12h` in `--signal-amber-base`.
*   **Transaction Panel:** Roster lock countdown projects as a persistent top-bar in the workspace, ticking `--signal-amber-loud` under 1 hour.

---

## 4. COMMAND LEDGER (Verbs for this Session)

Every actuation is a Guix-simple verb.

*   `ui.summon target=comms` / `ui.summon target=calendar` (Summons overlays)
*   `ui.event.escalate` (Internal flag tripping structure to `--surface-raised`)
*   `ui.event.resolve` (Sinks the TEO, clears action affordance)
*   `comms.send payload=<string>` (Routes to LLM router or chat backend)
*   `txn.commit type=trade` / `type=waiver` (Triggers B-locked ConfirmModal)
*   `ui.calendar.view mode=<week|month|agenda>` (Changes lens)
*   `ui.calendar.create` / `ui.calendar.move id=<uuid>` (Appends revision)
*   `ui.snackbar.dismiss` (Closes append-only confirmation)

---

## 5. RIPCORD ITEMS

*   **The "Jump vs Flow" Conflict:** The "One River of Time" metaphor assumes the user wants to chronologically scroll. A 25-league operator managing fires doesn't want to flow downstream; they want to *teleport* to the fire. 
    *   *Objection:* Grouping by strict chronological order will bury a critical UFA signing under 20 meaningless waiver clears from another league simply because of timestamp order.
    *   *Alternative:* **The "Triage Tributary" Override.** I propose that the Feed defaults to strict chronological time, BUT when event severity escalates to `amber-loud` or `red-loud`, those TEAs are pulled out of the chronological flow and grouped into a pinned "ALERTS" tributary at the top of the Feed. The user can hit `Esc` to let them flow back into the river.
*   **Unscheduled Chat vs Scheduled Calendar:** A chat message and a calendar event both occupy the timeline, but a chat has no "duration" or "time-to-live" in the structural sense. 
    *   *Objection:* Forcing chat messages to carry empty time/duration fields in the Timeline Event Object (TEO) just to fit the unified grammar is database bloat.
    *   *Alternative:* The Calendar view simply filters TEAs by `duration > 0` or `is_scheduled == true`. Chat remains an instantaneous point on the timeline; it renders in the Feed and Comms, but physically cannot exist in the Month/Week grid views.
