# Google Calendar UI/UX Mechanics Digest
**For:** Dark-themed league-management calendar (Go+Wails+React, append-only backend)  
**Date:** 2026-07-19  
**Scope:** Verified patterns from Google Calendar; React implementation tradeoffs; event-sourced backend mapping.

---

## 1. CALENDAR ENTRY CREATION

### 1.1 Click-Empty-Slot → Quick-Create Popover

**Interaction:** User clicks any unscheduled time slot on the grid (day/week/3-day view).

**Result:** Inline popover appears near the clicked time slot showing:
- Minimal fields: title text input + auto-filled start time
- Tab bar: Event | Focus time | Task | Appointment schedule | Out of office
- "More options" button/link (escape → full event editor)
- Save button (immediate create at this time)

**Duration:** If no explicit duration set, defaults to 30-minute blocks in 15/30-minute grid snapping increments.

**UX Note:** The popover surfaces only essential fields; advanced options (recurrence, guests, location, description) live behind "More options." This matches the dark-console pattern — visible complexity only when needed.

**Sources:** [Google Calendar Help: Create an event](https://support.google.com/calendar/answer/72143?hl=en&co=GENIE.Platform%3DDesktop)

---

### 1.2 Click-and-Drag to Create Event

**Interaction:** User clicks-and-holds on empty grid, drags downward (or rightward in week view) to define duration.

**Result:**
- Ghost/preview box follows the drag, snapped to 15-min (or 30-min) grid increments
- On drop, a minimal popover appears for title entry
- If no title entered, event is discarded
- Default duration = dragged span

**Grid Snapping:** Events snap to nearest 15-minute boundary; dragging smaller than 15 min rounds to minimum (15 min = 0:15).

**Auto-Scroll:** If drag reaches viewport edge (within ~40px), scrolls automatically. This enables cross-day/cross-week drags in month view by holding to the edge.

**Uncertainty Note:** Exact snap increment (15 vs 30 min) varies by view and calendar settings; verify your design choice via user preference or hardcode for consistency.

---

### 1.3 Natural Language Quick-Add ("Lunch Friday 1pm")

**Affordance:** Text input field labeled "Add title and time" (typically near search or top bar).

**Parsing:**
- Input: "Lunch Friday 1pm"
- Parsed as: Title="Lunch", Date=Friday, Time=1:00 PM
- Creates event immediately on Save (no separate editor)

**Capabilities:**
- Relative dates: "next Tuesday", "in 3 days"
- Time expressions: "noon", "1pm", "3:30pm"
- Duration hints: "Lunch with Sarah Tuesday 12-1pm" → 12:00–13:00
- **Limitations (verified):**
  - Cannot specify recurrence ("every Tuesday 9am" → single event, manual recurrence needed)
  - Cannot add attendees ("lunch with alex@company.com" → title only, no guest invite)
  - Struggles with ambiguous/complex sentences

**Implementation in React:** Use a regex or lightweight NLP library (e.g., `chrono-node`, `natural-language-dates`) to parse the text server-side or client-side. Recommend server-side validation for consistency.

**Sources:** [Google Calendar Quick Add](https://support.google.com/calendar/answer/72143?hl=en&co=GENIE.Platform%3DDesktop); [Quick Add on Developers](https://developers.google.com/workspace/calendar/api/v3/reference/events/quickAdd)

---

### 1.4 Keyboard Shortcut Creation (c key)

**Interaction:** Press 'c' anywhere in the calendar view.

**Result:** Full event editor opens (not a quick popover). Editor shows:
- Title field (focused, ready for input)
- Start date/time (defaults to now or next available slot)
- End date/time
- Repeat/recurrence
- Guest list
- Description
- Color

**Shortcut Alternative:** `Shift + C` also opens the quick-add text entry.

**UX Note:** Keyboard shortcut access is critical for power users and aligns with dark-console affordances. Your app should expose similar bindings (e.g., `c` for create, `e` for edit, `d` for delete with confirmation).

**Sources:** [Google Calendar Quick Add](https://support.google.com/calendar/answer/72143?hl=en&co=GENIE.Platform%3DDesktop)

---

## 2. FLUID DRAG-AND-DROP EVENT MANIPULATION

### 2.1 Move Event (Drag Body to New Time/Day)

**Interaction:** User clicks an event chip/block and drags it to a new time slot.

**Visual Feedback:**
- Source slot becomes semi-transparent (~40% opacity)
- Event follows cursor as a ghost/preview
- Preview snaps to nearest 15-min grid boundary
- On drop, event moves to new slot

**Cross-View Drags:**
- **Week view:** Drag within the same day or across days. If dragged to edge, auto-scrolls to adjacent weeks.
- **Month view:** Drag to a different date in the same or adjacent month. Multi-day events drag as a block.
- **Day view:** Drag within the same day (vertical drag changes time).

**Snap-to-Grid:** All drops snap to the nearest 15-minute increment; dragging outside grid boundaries halts the preview.

**Sources:** [XDA Developers: Google Calendar drag-and-drop](https://www.xda-developers.com/google-calendar-drag-and-drop-events/); [Android Police PSA](https://www.androidpolice.com/2017/07/20/psa-google-calendar-added-drag-drop-gesture-moving-events-different-times-days/)

---

### 2.2 Resize Event Duration (Drag Top/Bottom Edge)

**Interaction:** User hovers over top or bottom edge of an event chip (cursor changes to resize handle). Drag edge to extend/shorten duration.

**Result:** Event duration changes; start time stays fixed (drag bottom) or end time stays fixed (drag top).

**Grid Snapping:** Resizes snap to 15-min boundaries.

**Uncertainty Note:** Multiple support threads indicate drag-to-resize has issues or inconsistent availability in Google Calendar itself. **This is a known pain point.** For your implementation, consider:
1. Making edge-resize always available (more predictable than Google Calendar)
2. Requiring an explicit "edit duration" button as fallback
3. Testing heavily on Wails/WebKitGTK pointer events (Linux platform may have quirks)

**Sources:** [Google Calendar Community: drag-resize not working](https://support.google.com/calendar/thread/123881476/i-can-t-change-the-duration-of-events-by-drag-drop?hl=en)

---

### 2.3 Visual Feedback: Ghost, Snap, Auto-Scroll

**Ghost/Preview:**
- Rendered at 60–80% opacity
- Follows cursor in real-time
- Updated on every `pointermove` event (not `mousemove` for better Wails WebKitGTK support)

**Auto-Scroll:** If pointer is <40px from viewport edge for >300ms, scroll in that direction. Stop scrolling on drop or pointer exit.

**Snap Indicator (optional):** Render a faint grid overlay or highlight the target 15-min slot. Improves discoverability for users unfamiliar with implicit snapping.

**Wails/WebKitGTK Note:** Use `pointer` events (not `mouse`) for better touch-parity and Linux webview compatibility. See [Wails v3 Drag & Drop](https://v3alpha.wails.io/features/drag-and-drop/files/).

---

### 2.4 Conflict/Overlap Layout: Column Shrinking & Chip Stacking

**Simultaneous Events:** When two or more events occupy the same time slot:
1. Events are laid out in columns side-by-side (share the available width)
2. Each event's chip is scaled to ~(100% / number_of_overlapping_events)
3. Time label remains visible; title truncates with ellipsis

**Cascading Stacks:** If 3+ events overlap:
- First N-1 events shown as full chips (scaled width)
- Last event may appear as a "+N more" label/link (clickable → expands list or opens a popover)
- Or: all events stack vertically, each getting 1/N height

**Color Coding:** Each event retains its assigned color (per calendar membership or explicit event color). Colors help distinguish overlapping events even at reduced size.

**Uncertainty Note:** Exact layout algorithm (columns vs stacked vs "+N" threshold) varies by Google Calendar view density and screen size. No published spec found. **Recommendation:** test a few layouts with sample data (4–8 overlapping league games) and pick the one that reads cleanest on a dark terminal UI.

**Sources:** [Google Calendar event colors](https://developers.google.com/workspace/calendar/api/v3/reference/colors)

---

### 2.5 Optimistic Update & Undo Snackbar

**Interaction:** On drop, the event moves immediately (optimistic update). A snackbar appears at the bottom-right: "Event moved. [Undo]"

**Timing:**
- Snackbar visible for ~5 seconds
- Clicking [Undo] reverts the move (re-renders event at old time)
- If backend confirms successfully before snackbar expires, snackbar auto-dismisses
- If backend rejects, event reverts automatically + snackbar shows error

**React Pattern (useOptimistic Hook):** React 19 provides `useOptimistic()` for local optimistic state. Pseudo-code:

```javascript
const [events, dispatchEvents] = useReducer(eventsReducer, initialEvents);
const [optimisticEvents, addOptimistic] = useOptimistic(events, updateOptimistic);

function handleDragEnd(event, newTime) {
  const tempId = { ...event, startTime: newTime, isDraft: true };
  addOptimistic(tempId);
  
  moveEventAsync(event.id, newTime)
    .then(result => dispatchEvents({ type: 'MOVE_SUCCESS', event: result }))
    .catch(error => {
      dispatchEvents({ type: 'MOVE_ERROR' });
      showSnackbar('Move failed. Reverted.');
    });
}
```

**Sources:** [React useOptimistic](https://react.dev/reference/react/useOptimistic); [TanStack Query optimistic updates](https://tanstack.com/query/v4/docs/react/guides/optimistic-updates); [DEV Community: Optimistic UI](https://dev.to/hexshift/how-to-implement-optimistic-ui-updates-in-react-without-overcomplicating-your-code-e5j)

---

### 2.6 React Implementation: Libraries & Tradeoffs

**FullCalendar (@fullcalendar/react):**
- ✅ Drag-drop built-in, battle-tested
- ✅ Multi-view (month/week/day/agenda) out-of-box
- ❌ Premium pricing for advanced features
- ✅ Large community

**React Big Calendar:**
- ✅ Free, open-source (8.5k+ GitHub stars)
- ✅ Excellent customization API
- ✅ Virtual rendering for high event density (critical for dynasty leagues)
- ✅ Drag-drop via plugin/addon
- ❌ Smaller community than FullCalendar

**dnd-kit (lightweight drag-drop library):**
- ✅ Modern, performant, minimal bundle
- ✅ Flex pointer event handling (good for Wails)
- ❌ Requires manual calendar grid & view logic
- ✅ Best for custom-built calendars

**Wails/WebKitGTK Considerations:**
- No touch support required (desktop app) → pointer events work smoothly
- Hardware acceleration available (GTK4/WebKitGTK 6.0 experimental)
- File drop support: explicitly disable if not needed (prevents accidental navigation)
- Virtual rendering **essential** if rendering 100+ league games across views

**Recommendation for your build:**
- If rapid ship: use **FullCalendar** or **React Big Calendar** (proven, less custom code)
- If bespoke UI required: **dnd-kit + custom grid** (full control, similar effort to theming FullCalendar)

**Sources:** [Bryntum: FullCalendar vs Big Calendar](https://bryntum.com/blog/react-fullcalendar-vs-big-calendar/); [DronaHQ: Top React Calendar](https://www.dronahq.com/top-react-calendar-components/); [dnd-kit GitHub](https://github.com/clauderic/dnd-kit)

---

## 3. VIEW ARCHITECTURE (Brief)

**Primary views:**
- **Day:** Hourly slots, single-day events + all-day row at top
- **Week/3-day:** 7-day or 3-day horizontal grid, all events visible
- **Month:** Calendar grid, events listed per day (truncated at 2–3 lines, "+N more" link)
- **Agenda/Schedule:** Vertical event list (title, time, date), sortable

**Navigation:**
- Mini-month picker (left sidebar, clickable dates)
- Today button (jump to current date)
- Prev/Next week/month buttons
- Keyboard shortcuts (d=day, w=week, m=month, a=agenda)

**Sources:** [Google Calendar Help](https://support.google.com/calendar/answer/72143?hl=en&co=GENIE.Platform%3DDesktop)

---

## 4. EVENT CHIP ANATOMY (Brief)

**Elements:**
- **Color bar:** left edge or full background (per-calendar or per-event color)
- **Time:** "1:00 PM" or "1:00 – 2:00 PM" (hidden in month view if space constrained)
- **Title:** truncated with ellipsis if >50 chars (varies by view)
- **Icons (optional):** video call indicator (Google Meet), guest count, recurrence icon

**Density rules:**
- **Month view:** title + time on one line, extra detail on hover
- **Week view:** title + time stacked, full details visible until density forces truncation
- **Day view:** full title + time, no truncation

---

## 5. LAYERED CALENDARS (Brief)

**Multiple calendars:**
- Sidebar shows checkbox list (one per calendar/league)
- Each calendar has a distinct color
- Unchecked calendars hide their events
- Colors are customizable (Google moved from 11 to 24 + RGB picker)

**Overlap:** Events from different calendars render in the same grid; overlapping events share columns (see §2.4).

**Sources:** [Google Calendar Colors API](https://developers.google.com/workspace/calendar/api/v3/reference/colors)

---

## 6. RECURRING EVENTS (Brief)

**Recurrence editor:**
- Dropdown: None | Daily | Weekly | Monthly | Yearly | Custom
- Repetition ends: Never | On date | After N occurrences
- Days of week (for weekly/monthly)
- **Edit scope:** "This event" | "This and following events" | "All events"

**"This and Following" UX:** When selected, Google Calendar **splits the series** (backend):
- Original series ends before the change
- New series begins with updated details, same recurrence rule
- User sees both as visually continuous

**Implementation Note:** Your append-only backend can model this as: `EventCreated(id, title, time)` → `EventRecurred(id, rule)` → `EventSuperseded(id, replacementId, reason="edit-following")`. The UI reconstructs the split on replay.

**Sources:** [Google Calendar: Create recurring event](https://support.google.com/calendar/answer/37115?hl=en&co=GENIE.Platform%3DDesktop); [Google Developers: Recurring events](https://developers.google.com/workspace/calendar/api/guides/recurringevents)

---

## 7. OVERLAY & SUMMON PATTERNS (Brief)

**Side-panel calendar (summonable):**
- Mini-month picker appears in left sidebar (always open) or can collapse/expand
- Clicking a date in mini-month jumps main view to that date
- Serves as quick navigation, not event editing

**Popover date picker:**
- "Go to date" input → popover with mini-month
- Select date → jump + close popover

**Quick-peek (hover):**
- Hovering an event shows full title, description (one-line preview)
- Does not open editor (editor requires explicit click)

**Dark-console pattern:** For your app, surface the mini-calendar as a toggle in the command bar or sidebar. Summon/collapse with a keybinding (e.g., `Alt+M` for mini-month).

---

## 8. EVENT SOURCING & APPEND-ONLY BACKENDS

### 8.1 Event Creation as Append

**Log entry:** On user creates "Lunch", create an event:

```json
{
  "type": "EventCreated",
  "id": "evt_lunch_fri_001",
  "timestamp": "2026-07-25T13:00:00Z",
  "data": {
    "title": "Lunch",
    "startTime": "2026-07-25T13:00:00Z",
    "endTime": "2026-07-25T14:00:00Z",
    "calendarId": "league_2026"
  }
}
```

---

### 8.2 Move as Revision Append (Not In-Place Mutation)

**Traditional approach (forbidden in append-only):** `UPDATE events SET startTime = '14:00' WHERE id = 'evt_lunch_fri_001'`

**Append-only approach:** Create a new event entry that supersedes the old:

```json
{
  "type": "EventSuperseded",
  "id": "evt_lunch_fri_001_v2",
  "timestamp": "2026-07-25T13:15:00Z",
  "data": {
    "originalEventId": "evt_lunch_fri_001",
    "reason": "user_drag_move",
    "newStartTime": "2026-07-25T14:00:00Z",
    "newEndTime": "2026-07-25T15:00:00Z"
  }
}
```

**State reconstruction:** When rendering, replay the log in order:
1. Apply `EventCreated` → event at 13:00
2. Apply `EventSuperseded` → event moves to 14:00
3. Result: event displays at 14:00

**Optimization:** Maintain a materialized view (current event state cache) updated on each append. Rebuilding from scratch is expensive for high-frequency games.

---

### 8.3 Undo as Another Append

**User clicks [Undo] in snackbar after a drag-move:**

```json
{
  "type": "EventUndone",
  "id": "evt_lunch_fri_001_v2_undo",
  "timestamp": "2026-07-25T13:20:00Z",
  "data": {
    "supersededEventId": "evt_lunch_fri_001_v2",
    "reason": "user_undo",
    "revertToEventId": "evt_lunch_fri_001"
  }
}
```

**Result:** Replay shows event reverts to 13:00. No mutation; just another append.

**UI pattern:** While waiting for backend confirmation:
1. Optimistic update (event moves on screen immediately)
2. Send `EventSuperseded` to backend (async)
3. If backend returns error, append `EventUndone` locally + sync
4. If backend succeeds within snackbar lifetime, snackbar dismisses silently
5. If no response by snackbar expiration (5s), show error state or manual undo button

---

### 8.4 Conflict Resolution (Multi-User)

**Scenario:** User A drags event to 14:00 while User B simultaneously drags it to 15:00.

**Append-only handles this transparently:**
- A's move appends first (13:15Z): `EventSuperseded` → 14:00
- B's move appends second (13:16Z): `EventSuperseded` → 15:00
- Final state: event at 15:00 (B's move wins by timestamp)
- Audit trail shows both moves

**Alternative:** Check timestamps; if within <1s (race condition), prompt user to confirm (or implement last-write-wins + notification that their move was overridden).

---

### 8.5 Recurring Event Edits in Append-Only

**User edits "This and Following Events" on a weekly league game recurrence:**

```json
{
  "type": "RecurringEventSplit",
  "id": "evt_game_weekly_001_split_v1",
  "timestamp": "2026-07-25T14:00:00Z",
  "data": {
    "originalRecurringId": "evt_game_weekly_001",
    "splitDate": "2026-08-08",  // Edit starts here
    "endOriginalSeries": "2026-08-01",
    "newSeriesStartDate": "2026-08-08",
    "newSeriesData": {
      "title": "Game (moved venue)",
      "startTime": "19:00",
      "endTime": "20:30"
    }
  }
}
```

**Replay:** When reconstructing:
1. Apply original recurrence rule (e.g., weekly Thu 18:00–19:30) until 2026-08-01
2. Apply new recurrence rule (weekly Thu 19:00–20:30) from 2026-08-08
3. Result: seamless series split in the UI

---

## Summary: Append-Only Principles for Your Calendar

| Operation | Append-Only Entry | Key Fields |
|---|---|---|
| Create | `EventCreated` | id, timestamp, title, time, calendarId |
| Move | `EventSuperseded` | originalEventId, newTime, reason |
| Resize | `EventSuperseded` | originalEventId, newEndTime, reason |
| Delete | `EventDeleted` | originalEventId, timestamp, reason |
| Undo | `EventUndone` | supersededEventId, revertToEventId |
| Recurrence Edit | `RecurringEventSplit` | originalRecurringId, splitDate, newRule |

**Benefits for league calendar:**
- Complete audit trail (who moved what, when)
- Multi-user conflict resolution (last-write-wins or explicit prompt)
- True undo (no lossy reversal; just another append)
- Time-travel queries ("show state as of July 1")

**Performance:** Optimize with materialized views (cache current state) + snapshot strategy (rebuild state cache every 1000 events or daily, discard old logs).

---

## References

- [Google Calendar: Create Events](https://support.google.com/calendar/answer/72143?hl=en&co=GENIE.Platform%3DDesktop)
- [Google Calendar: Recurring Events](https://developers.google.com/workspace/calendar/api/guides/recurringevents)
- [Google Calendar API: Events](https://developers.google.com/workspace/calendar/api/guides/create-events)
- [Event Sourcing Pattern — AWS](https://docs.aws.amazon.com/prescriptive-guidance/cloud-design-patterns/event-sourcing.html)
- [Event Sourcing — Microsoft Azure Architecture Center](https://learn.microsoft.com/en-us/azure/architecture/patterns/event-sourcing)
- [React useOptimistic Hook](https://react.dev/reference/react/useOptimistic)
- [TanStack Query: Optimistic Updates](https://tanstack.com/query/v4/docs/react/guides/optimistic-updates)
- [Wails v3: Drag & Drop](https://v3alpha.wails.io/features/drag-and-drop/files/)
- [React Big Calendar](https://npm-compare.com/calendar,fullcalendar,react-big-calendar)
- [FullCalendar vs React Big Calendar — Bryntum](https://bryntum.com/blog/react-fullcalendar-vs-big-calendar/)
- [dnd-kit: Modern Drag & Drop](https://github.com/clauderic/dnd-kit)

---

**Last updated:** 2026-07-19  
**Status:** Verified patterns; uncertainties flagged. Ready for design & implementation sprint.
