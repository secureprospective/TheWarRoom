# SESSION D: THE OPS ROOM — COMMAND IS THE SPINE

**Design Direction Document v1.0**

The command line is not a feature; it is the nervous system. Chat, Feed, and Calendar are not peers—they are specialized readouts and actuators docked around the operator's primary instrument. An event is not a notification; it is an **outstanding verb** awaiting execution or resolution. We unify these surfaces under a strict grammatical ruleset, maintaining absolute visual restraint to serve both the 2-league casual fan and the 25-league portfolio operator.

---

## 1. THE UNIFIED EVENT GRAMMAR

Feed events, chat system alerts, and calendar deadlines are structurally identical. They vary in context, not anatomy. 

### Core Anatomy (`event-row`)
All surfaces use this base structure, rendered via CSS custom properties for instant density tier shifts (Tactical = 32px default):
*   **Severity Spine:** 2px left axis. Gray (`--edge-selection`) for standard info, semantic hues strictly mapped to state.
*   **Payload:** 
    *   *Subject:* Inter, 13px, `--text-primary`.
    *   *Data:* JetBrains Mono, 11px, `--text-muted`, `font-variant-numeric: tabular-nums`.
*   **Verb Affordance (The Actuator):** A recessed micro-switch (1px inset bevel). Appears as a precise text label (e.g., `[ ACCEPT ]`). `:hover` raises the bevel; `:active` depresses it. 

### Spectrum Strategy: Calm vs. Triaged
The grammar serves both users without a mode toggle by controlling **Verb Affordance Visibility (VAV)**:
*   **Casual (1-2 Leagues):** VAV is always visible. The feed reads as a simple checklist. Zero ambient anxiety because the queue is inherently short.
*   **Operator (25 Leagues):** VAV is hidden by default to prevent visual chaos. Verbs appear only on `:hover`, `:focus-within`, or if the event meets **Escalation** criteria. The feed becomes a scannable data matrix; action is deliberate, not demanded.

### Escalation & Recession (The Lifecycle)
*   **Escalation:** Time-compression or high-severity triggers a structural promotion. 
    *   *Trigger:* Bid clock < 1hr (`--signal-amber-loud`), Snipe/Danger (`--signal-red-loud`).
    *   *Visual:* Severity spine widens to 4px (translateX transform). Timestamp turns semantic. Verb Affordance becomes persistent and pulsates via a 150ms opacity transform. **No modals.** It does not interrupt; it demands attention via optical weight.
*   **Recession:** The verb is resolved (e.g., `txn.commit` fires) or expires.
    *   *Visual:* A 150ms cross-fade (opacity only). The row drops its semantic hue, spine reverts to 2px gray, row opacity reduces to 50% (`--text-muted`). It remains in the log but removes itself from the active triage queue. Zero accumulation of "unread" badges.

### Freshness & The Multi-League Seam
*   **Freshness:** If an event is cached/offline, the Severity Spine shifts from `--edge-freshness-live` (blue) to `-stale` (amber-muted). Timestamp gets `(cache)` suffix. 
*   **Cross-League Aggregation:** 
    *   *RETURN-SESSION FLAG (v2):* Full cross-league aggregation. For v1, the 25-league operator views a unified Feed, but the Chat/Calendar surfaces remain scoped to the active league in the Inspector. The seam is exposed via a `[L: ALL]` / `[L: ACTIVE]` toggle at the Feed header. 

---

## 2. CHAT AS THE TERMINAL'S VISUAL HOME

The right-edge 48px idle strip summons the 320px Comms overlay (`ui.summon target=comms`). This is not a chat box; it is the visual home of the Command Ledger.

### Thread Grammar & Topography
*   **Human Chatter:** Right-aligned. Inter 13px. Bubble background `--surface-tile`.
*   **System Answers:** Left-aligned. JetBrains Mono 11px. Prefixed with `>`. Color `--text-muted`. 
*   **Command Input (User typed `/verb args`):** Left-aligned. Inter 13px, weight 600. Background `--surface-raised` with a 1px top/left inset highlight. 

### Control Surfaces inside the Conversation
When a command like `/offer trade` executes, it spawns a **Control Card** natively in the thread. It does not route to a separate view.
*   **Anatomy:** `--surface-overlay` background, 1px `--edge-selection` border. 
*   **The Verbs:** Accept / Counter / Decline render as tactile actuation buttons inside the card.
*   **The Gate:** Selecting Accept or Counter engages the Session B `ConfirmModal` (480px centered, hold-to-fire). The card visually locks (opacity 0.5) while the modal is active. Engine-reject triggers the 150ms red flash.

### The Prompt Affordance
*   **Visual:** A flush input field at the bottom, separated by a 1px `--edge-hairline`.
*   **Behavior:** Default text reads `Message or /command`. 
*   **The Pivot:** Upon keystroke `/`, the input's left axis gains a 2px `--edge-focus` (blue), signaling command mode. Font-family switches to JetBrains Mono. This cues the operator without alienating the casual user who simply types text.

---

## 3. THE CALENDAR — FULLY BUILDABLE

Top-right generous overlay. Event-sourced append-only mechanics. Google Calendar from a command console.

### Event-Chip Anatomy (`cal-event`)
*   **Structure:** Left 2px color bar + Title (Inter 11px/13px depending on view) + Time (Mono).
*   **Hue Restraint:** Categorical positions (QB violet, RB magenta) do NOT apply here. Calendar hues map strictly to semantic deadlines:
    *   Waiver Run: `--signal-blue-base`
    *   Trade Deadline: `--signal-amber-base`
    *   Draft/Live Events: `--signal-green-base`
    *   Hard Cutoffs/Roster Locks: `--signal-red-base`

### Views & Overlap
*   **Views:** Month (grid), Week (time-col), Agenda (list). Plus a permanent Mini-Month navigator in the overlay's upper quadrant.
*   **Overlap Commitment (Column-Share):** Simultaneous events split the column width with a 1px hairline divider. No stacking, no "+N more" modals. If an event is < 15min, title truncates, time remains. 

### Manipulation & Append-Only Honesty
WebKitGTK cheap-paint floor respected (transforms only).
*   **Creation:** Click-empty → quick-create popover (Title + Verb mapping). Click-drag-to-create leaves a `--surface-raised` ghost block snapping to a 15-min grid.
*   **Move/Resize:** Drag-move and edge-resize utilize `transform: translate3d` / `scaleY`. The original chip remains at 50% opacity (proving history), the ghost snaps to the new grid. 
*   **Commit & Honesty:** On mouse release, the drag does *not* mutate the DB. It appends an `EventSuperseded` record. 
    *   *UI Honesty:* A Snackbar triggers at the bottom: `Event moved. [UNDO]`. Clicking UNDO executes another append (`EventSuperseded` of the move). 
*   **Fallback:** Edge-resize is risky on some compositors. If pointer-up fails to register drag delta > 4px, release triggers an inline "Edit Duration" mono-input text field rather than committing a bad transform.

### Operational Overlay (Projections)
Deadlines do not stay trapped in the overlay. They project onto the surfaces they govern.
*   **Trade Builder:** A persistent badge in the header: `[T- 02D 14H] Trade Deadline` (`--signal-amber-base`).
*   **Transaction Panel:** Live countdown clock `T- 00:45:12` (`--signal-amber-loud` clock<1hr rule). 

---

## 4. COMMAND LEDGER ADDITIONS

*   `ui.summon target=comms|calendar` *(Confirmed A10/A11)*
*   `feed.ack <event_id>` *(Recedes an escalated event)*
*   `txn.commit <trade_id> action=accept|counter|decline` *(Routes through ConfirmModal)*
*   `chat.send <string>` / `cmd.execute <string>` *(Auto-routes based on `/` prefix)*
*   `calendar.create <title> <datetime> <duration> hue=semantic` *(Appends EventCreated)*
*   `calendar.move <event_id> <new_datetime>` *(Appends EventSuperseded, triggers Undo Snackbar)*

---

## RIPCORD ITEMS

1.  **OBJECTION TO PROVOCATION: "The Command Line is the Spine."**
    *   *The Conflict:* The prompt demands the chat terminal be the PRIMARY instrument that the whole app is driven from. However, the Spectrum Commitment mandates we delight a 1-2 league casual user who "wants a calm league they enjoy, not a terminal with a blinking cursor." If the chat terminal is the spine, the casual user is forced into an operational paradigm they didn't ask for.
    *   *The Resolution:* I have designed the Chat/Command surface to be *structurally* capable of driving the app (the spine), but *visually* recessive for users who don't engage it. The 48px idle strip at the right edge keeps the command line out of the primary focal zone. The casual user can click rows and use traditional UI actuations their entire season without ever typing a `/` command. The command line is the spine for the *engine*, but the UI is the face for the *user*.
2.  **OBJECTION TO REcession RULE: "Zero Unread Anxiety Debt"**
    *   *The Conflict:* The prompt mandates that resolved events calm and sink, accumulating no unread debt. However, for high-stakes actions (e.g., a trade was accepted), users often need a historical record of *what* happened that doesn't just fade to 50% opacity.
    *   *The Alternative:* Expired/Resolved events fade to 50% opacity in the active Feed view, but they do not disappear. A dedicated `/history` verb in the Chat terminal dumps a clean, 100% opacity mono-spaced log of all resolved events. The active surface remains calm; the log remains exact.
