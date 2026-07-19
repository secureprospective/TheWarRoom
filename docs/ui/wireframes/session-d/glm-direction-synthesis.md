# SESSION D — SYNTHESIS: Command, Calendar & The Spine

## 1. Unified Event Anatomy & Feed Grammar
The substrate is a single time-ordered event stream. 

**Anatomy:** `id` | `timestamp` | `category` (feed/chat/cal/system) | `is_scheduled` (bool) | `duration` (mins) | `payload` | `status` (active/superseded/resolved) | `severity` (base/escalated).
**Severity Spine:** A 2px left-axis border. Default state is `--edge-focus` (blue) or achromatic. 
**Freshness (R-E):** If cached/offline, spine shifts to `--edge-freshness-stale` (amber-muted), timestamp gets `(cache)` suffix. Data value remains unchanged.
**Escalation (R-F):** On `--signal-amber-loud` or `--signal-red-loud`: spine widens 2px→4px (transform scale), adopts semantic hue, row raises to `--surface-raised`. No looping animations.
**Recession:** `ui.event.resolve` cross-fades row to 50% opacity in 150ms. Unread badges are forbidden.

**The Feed (R-A, R-G):**
- Default state is strict chronological (Watch Floor river).
- **Triage Tributary:** Escalated events pin to an `ALERTS` group at the top of the Feed. `Esc` collapses the Alerts group back into the river.
- **Cross-League Seam:** Feed header features a `[L: ALL] / [L: ACTIVE]` toggle (Phase-3 injection point). v1 defaults to ACTIVE.
- **VAV (Verb Affordance Visibility) (R-A):** An altitude setting, not a mode. 
  - *Casual (1-2 leagues):* Verb affordances (Ack, Resolve) are persistently visible.
  - *Operator (25 leagues):* Verb affordances are `opacity: 0`, snapping to `opacity: 1` on row `:hover` / `:focus-within`. Escalated events force persistent affordances regardless of altitude.

## 2. Terminal-Log Comms (R-B)
Chat is a full-width terminal log, not a SaaS bubble UI. 1px `--border-subtle` hairline separators.
- **Human Chatter:** Inter 13px, `--text-primary`.
- **User Command:** JetBrains Mono, `--text-secondary`, prefix `> /verb`.
- **System Answer:** JetBrains Mono, `--text-muted`, prefix `↳`. No assistant persona.

**The `/`-Pivot Prompt:**
Standard input is Inter. On keystroke `/`, input border switches to `--edge-focus` (inset blue) and font switches to JetBrains Mono. `Esc` reverts to text mode.

**In-Thread Control Card (`/offer`):**
Executing `/offer` injects a full-width inset tile (`--surface-raised` bevel) directly into the chat log.
- Contains Accept / Counter / Decline micro-switches.
- Clicking a switch spawns the 480px centered ConfirmModal (`txn.commit`).
- While modal is active, Control Card locks to `opacity: 0.5`.
- On commit, card content replaces with `[ COMMITTED — HH:MM ]`.

## 3. The Buildable Calendar (R-C, R-D, R-H)
**Grid & Views:** Month, Week, Agenda + persistent Mini-Month navigator (top-right).
**Substrate Filter:** Renders ONLY events where `is_scheduled == true` AND `duration > 0`. Chat/Feed noise is excluded.

**Event Chip Anatomy:**
- `--surface-base` background, 2px left-axis color bar indicating deadline semantic hue (Waiver=`--signal-blue-base`, Trade=`--signal-amber-base`, Draft=`--signal-green-base`, Roster-lock=`--signal-red-base`). 
- Position badges (QB/RB/etc) DO NOT apply on the calendar.

**Column-Share Overlap:**
Simultaneous events divide column width evenly, separated by a 1px `--border-subtle` hairline. If >3 events overlap, they collapse into a `+N more` aggregate chip, which opens a tactical dropdown list on click.

**Interactions:**
- **Quick-Create:** Click empty grid space → popover form.
- **Drag-Create:** Click-drag empty grid space → generates a 15-min snapping ghost block. Release commits.
- **Native Drag-MOVE:** Compositor-safe. Click-drag an existing event uses `transform: translate(x, y)` ghost with `opacity: 0.8`. Snaps to 15-min increments. Commits on release.
- **Stepper-Only RESIZE:** NO native edge-drag resize. Click chip → Inspector 320px panel → tactical `+15m` / `-15m` stepper duration switches.
- **Append-Only Move:** A drag-move does not mutate. It appends `EventSuperseded` and creates a new event. Triggers a 32px bottom snackbar: `Event moved. [UNDO]`. UNDO appends `EventUndone`. History is strictly linear.

**Operational Overlay (R-H):**
- **Trade Builder/ConfirmModal:** Header projects `Trade Deadline: Nd Nh` badge in `--signal-amber-base`.
- **Workspace Top-Bar:** Projects roster-lock countdown. Ticks `--signal-amber-loud` under 1 hour.

## 4. Command Ledger (Verbs)
Consolidated Guix-simple verbs for the Command Layer.

| Verb | Args | Action / Notes |
| :--- | :--- | :--- |
| `comms.send` | `text`, `league_id` | Standard human chat input. |
| `cmd.execute` | `verb`, `payload` | Triggered by `/`-pivot prompt. Routes `/offer`, `/history`. |
| `feed.ack` | `event_id` | Silences ambient alerts, updates last-viewed timestamp. |
| `ui.event.resolve` | `event_id` | Cross-fades event to 50% opacity (active surface). |
| `txn.commit` | `action`, `target_id` | `action` = "accept" / "decline" / "counter". Fires ConfirmModal hold-to-fire. |
| `calendar.create` | `start`, `duration`, `type` | Triggered by quick-create or drag-create ghost. |
| `calendar.move` | `event_id`, `new_start` | Appends `EventSuperseded`. Triggers UNDO snackbar. |
| `calendar.resize` | `event_id`, `duration_delta` | Triggered by tactical stepper in Inspector. |
| `ui.snackbar.dismiss`| `action_id` | Dismisses append-only UNDO snackbar. |
| `history` | `range`, `filter` | Dumps mono log of resolved/expired events at 100% opacity in terminal. |
