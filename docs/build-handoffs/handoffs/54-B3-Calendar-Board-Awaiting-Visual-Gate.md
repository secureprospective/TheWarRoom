# Handoff 54 — B-3 Commissioner Calendar Board (frontend)

**Date:** 2026-07-25
**Branch:** `session/ui-b3-calendar-board` (HEAD `cc31442`, pushed to origin)
**Status:** Build ✅ · GLM 5.2 review gate ✅ · **Visual gate PENDING** (merge held on it)

## What shipped
The `calendar` summon target is now a live **agenda/list board** (`frontend/src/components/calendar/CalendarBoard.tsx`) over the append-only `calendar_events` backend (merged `c22be1e`). It:
- reads the head view via `GetCalendarEvents`;
- drives **SCHEDULE / RESCHEDULE / CANCEL** through the shared D5 preview → `ConfirmModal` → D4 re-send-intent path;
- groups blobs by day, filled/hollow status dots + time, PLANNED rows expose ⇄ reschedule (inline datetime picker) and ✕ cancel; FIRED/CANCELLED heads render inert (history);
- schedule form is scoped to the trivial-payload **season-clock trio** (advance phase / roll season / signing window); §13 acts schedule from LeagueControls but render/reschedule/cancel here (payload re-sent verbatim).

`App.tsx`: routes `summoned==='calendar'` to `<CalendarBoard>`; `SummonPlaceholder` narrowed to comms-only.

## Decisions this session
- **Render model = agenda/list** (not month time-grid + drag) for v1 — Christopher, 2026-07-25. Fits the summon drawer, reuses the whole `.twr-board` token contract, no new dep. Month grid can layer on later.
- **Auto-fire = out of scope** — resolved by source: backend `DuePlannedEvents` is the §12-gated Phase-6 scheduler; no fire-now IPC exists yet.

## Contract fidelity
- reschedule/cancel append a new row sharing `event_id`, re-sending head `kind`+`payload` verbatim (never an edit);
- inset-bevel elevation only (no drop shadows — Session-A/C restraint);
- zero new hex/rgba/legacy-palette (grep-swept); CSS flat at 16.95 KB, JS 224.9→234.5 KB.

## GLM 5.2 review (triaged vs source)
Fixed: stale-preview race (`stageGen` token), unhandled `ExecuteTransaction` throw (catch → rejected state), empty-datetime reschedule no-op (button disabled). Non-issues: CAP_RELIEF amber dot is an intentional weight axis; head view is `ORDER BY scheduled_at`; `target` prop used in aria-label.

## Next
Run the visual gate (commands in `/root/paste.md`, Beelink `/home/chris/opencode/TheWarRoom`). On **PASS** → squash-merge to main (linear history required — `git reset --soft origin/main` + single commit, no merge commit), delete branch, update CLAUDE.md build-state header. Then B-4 (Home + Inspector).
